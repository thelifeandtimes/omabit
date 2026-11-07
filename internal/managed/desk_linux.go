//go:build linux && amd64

package managed

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type deskRuntime interface {
	ListDesks(context.Context) ([]string, error)
	Observe(context.Context, string) (deskwork.Observation, error)
	Create(context.Context, string) error
}
type deskBackend interface {
	OpenManagedDesk(context.Context, runtimequalify.ManagedState, string) (deskRuntime, error)
	OpenProjectDesk(context.Context, runtimequalify.ProjectCurrentReadinessSelection, string, string) (deskRuntime, error)
	OpenExpandedProjectDesk(context.Context, runtimequalify.ProjectExpansion, string, string) (deskRuntime, error)
}

func (realBackend) OpenManagedDesk(ctx context.Context, s runtimequalify.ManagedState, id string) (deskRuntime, error) {
	return runtimequalify.OpenManagedDesk(ctx, s, id)
}
func (realBackend) OpenProjectDesk(ctx context.Context, s runtimequalify.ProjectCurrentReadinessSelection, id, cid string) (deskRuntime, error) {
	return runtimequalify.OpenProjectDesk(ctx, s, id, cid)
}
func (realBackend) OpenExpandedProjectDesk(ctx context.Context, s runtimequalify.ProjectExpansion, id, cid string) (deskRuntime, error) {
	return runtimequalify.OpenExpandedProjectDesk(ctx, s, id, cid)
}

// The caller holds the same mutex as configuration, lifecycle admission and
// Setup policy edits. The backend independently pins the current live process.
func (r *Server) openDesk(ctx context.Context, selection protocol.DeskSelection) (deskRuntime, error) {
	if err := selection.Validate(); err != nil {
		return nil, bad(err.Error())
	}
	if err := appPolicyTarget(r.s.db, selection.Target); err != nil {
		return nil, err
	}
	b, ok := r.b.(deskBackend)
	if !ok {
		return nil, bad("desk runtime adapter unavailable")
	}
	target := selection.Target
	if target.ProjectID != "" {
		var active int
		if err := r.s.db.QueryRow("SELECT count(*) FROM operations WHERE json_extract(body,'$.project_id')=? AND json_extract(body,'$.status') IN ('queued','running')", target.ProjectID).Scan(&active); err != nil {
			return nil, err
		}
		if active != 0 {
			return nil, conflict("runtime_changed", "wait for the project operation to finish")
		}
		if expanded, err := loadProjectExpansion(r.s.db, target.ProjectID); err == nil {
			if expanded.Project.Revision != selection.ConfigRevision {
				return nil, conflict("runtime_changed", "project revision changed")
			}
			return b.OpenExpandedProjectDesk(ctx, expanded, target.InstanceID, selection.RuntimeID)
		} else if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		s, err := r.s.projectFleetSelection(target.ProjectID, target.InstanceID)
		if err != nil {
			return nil, err
		}
		if s.Project.Revision != selection.ConfigRevision {
			return nil, conflict("runtime_changed", "project revision changed")
		}
		return b.OpenProjectDesk(ctx, s, target.InstanceID, selection.RuntimeID)
	}
	v, err := loadInstance(r.s.db, target.InstanceID)
	if err != nil {
		return nil, err
	}
	p := v.Public
	if p.ConfigRevision != selection.ConfigRevision || p.ObservedState != "running" || p.ActiveOperationID != "" || p.RecoveryRequired || p.Resources == nil || p.Resources.ContainerID != selection.RuntimeID {
		return nil, conflict("runtime_changed", "select the current running instance and container")
	}
	return b.OpenManagedDesk(ctx, v.Runtime, selection.RuntimeID)
}
func deskSelection(i deskwork.Intent) protocol.DeskSelection {
	return protocol.DeskSelection{Target: i.Target, RuntimeID: i.RuntimeID, ConfigRevision: i.ConfigRevision}
}
func (r *Server) deskReply(receipt deskwork.Receipt) protocol.DeskOperation {
	return protocol.DeskOperation{Version: 1, RunnerID: r.s.runnerID, Mode: r.s.mode, Receipt: receipt}
}
func (r *Server) serveDeskObservation(w http.ResponseWriter, req *http.Request, role string) {
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "live desk reads require operator access"}})
		return
	}
	var selection protocol.DeskSelection
	if err := decode(w, req, &selection); err != nil {
		r.respondError(w, err)
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(protocol.DeskReadTimeout + time.Second))
	ctx, cancel := context.WithTimeout(req.Context(), protocol.DeskReadTimeout)
	defer cancel()
	runtime, err := r.openDesk(ctx, selection)
	if err != nil {
		r.respondError(w, err)
		return
	}
	observation, err := runtime.Observe(ctx, selection.Target.Desk)
	if err != nil {
		r.respondError(w, &fault{502, protocol.APIError{Code: "desk_unavailable", Message: "desk observation unavailable for this runtime"}})
		return
	}
	policy, err := loadAppPolicy(r.s.db)
	if err != nil {
		r.respondError(w, err)
		return
	}
	respond(w, 200, protocol.DeskObservation{Version: 1, RunnerID: r.s.runnerID, Mode: r.s.mode, Selection: selection, PolicyRevision: policy.Revision, Observation: observation, ReadAt: time.Now().UTC()})
}

// Receipt reads do not wait for the effect mutex. They contain no source or
// credentials and remain available while a bounded asynchronous effect runs.
func (r *Server) serveDeskReceipt(w http.ResponseWriter, req *http.Request, role string) {
	id := strings.TrimPrefix(req.URL.Path, protocol.DeskOperationsPath+"/")
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "desk receipts require operator access"}})
		return
	}
	if !protocol.ValidDeskOperationID(id) {
		r.respondError(w, bad("invalid desk operation ID"))
		return
	}
	// Read current body and its history in one transaction: a worker may append.
	tx, err := r.s.db.Begin()
	if err != nil {
		r.respondError(w, err)
		return
	}
	defer tx.Rollback()
	receipt, err := loadDeskOperation(tx, id)
	if errors.Is(err, sql.ErrNoRows) {
		r.respondError(w, &fault{404, protocol.APIError{Code: "desk_operation_not_found", Message: "desk operation not found"}})
		return
	}
	if err != nil {
		r.respondError(w, err)
		return
	}
	if err = tx.Commit(); err != nil {
		r.respondError(w, err)
		return
	}
	respond(w, 200, r.deskReply(receipt))
}
func (r *Server) serveDeskOperation(w http.ResponseWriter, req *http.Request, actor, role string) {
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "desk effects require operator access"}})
		return
	}
	var intent deskwork.Intent
	if err := decode(w, req, &intent); err != nil {
		r.respondError(w, err)
		return
	}
	if intent.Action != "create-desk" {
		r.respondError(w, bad("this adapter currently supports create-desk only"))
		return
	}
	if err := intent.Validate(); err != nil {
		r.respondError(w, bad(err.Error()))
		return
	}
	if _, ok := r.b.(deskBackend); !ok {
		r.respondError(w, bad("desk runtime adapter unavailable"))
		return
	}
	receipt, fresh, err := r.s.createDeskOperation(actor, intent)
	if err != nil {
		r.respondError(w, err)
		return
	}
	if fresh {
		r.queueDeskOperation(receipt)
	}
	respond(w, http.StatusAccepted, r.deskReply(receipt))
}

// Every effect is dispatched once by this process, after its durable intent.
// Restart only marks unfinished receipts; it never calls queueDeskOperation.
func (r *Server) queueDeskOperation(receipt deskwork.Receipt) {
	r.updateWorkers++
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.mu.Lock()
		defer r.mu.Unlock()
		defer func() { r.updateWorkers-- }()
		appendStep := func(state string, observation *deskwork.Observation) bool {
			next, err := r.s.appendDeskStep(receipt, deskwork.Step{Kind: receipt.Intent.Action, State: state, At: time.Now().UTC(), Observation: observation})
			if err != nil {
				r.failed.Store(true)
				return false
			}
			receipt = next
			return true
		}
		if r.closing || r.ctx.Err() != nil {
			appendStep("interrupted", nil)
			return
		}
		if r.failed.Load() || r.updateFence.Held || !r.routineAuthorized(receipt.Actor) {
			appendStep("blocked", nil)
			return
		}
		ctx, cancel := context.WithTimeout(r.ctx, 4*time.Minute)
		defer cancel()
		runtime, err := r.openDesk(ctx, deskSelection(receipt.Intent))
		if err != nil {
			appendStep("blocked", nil)
			return
		}
		current, err := runtime.Observe(ctx, receipt.Intent.Target.Desk)
		if err != nil {
			appendStep("blocked", nil)
			return
		}
		policy, err := loadAppPolicy(r.s.db)
		if err != nil {
			r.failed.Store(true)
			return
		}
		if err = receipt.Intent.Check(policy, current); err != nil {
			appendStep("blocked", nil)
			return
		}
		if !appendStep("dispatched", nil) {
			return
		}
		if err = runtime.Create(ctx, receipt.Intent.Target.Desk); err != nil {
			appendStep("uncertain", nil)
			return
		}
		if !appendStep("acknowledged", nil) {
			return
		}
		after, err := runtime.Observe(ctx, receipt.Intent.Target.Desk)
		if err != nil || !after.Exists || after.Validate() != nil {
			appendStep("uncertain", nil)
			return
		}
		appendStep("observed", &after)
	}()
}

// Reconciliation observes the selected runtime. It never redispatches creation
// and cannot treat a missing desk as proof that a lost request will not arrive.
func (r *Server) serveDeskReconciliation(w http.ResponseWriter, req *http.Request, actor, role string) {
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "desk reconciliation requires operator access"}})
		return
	}
	id := strings.TrimSuffix(strings.TrimPrefix(req.URL.Path, protocol.DeskOperationsPath+"/"), "/reconcile")
	if !protocol.ValidDeskOperationID(id) {
		r.respondError(w, bad("invalid desk operation ID"))
		return
	}
	var empty struct{}
	if err := decode(w, req, &empty); err != nil {
		r.respondError(w, err)
		return
	}
	receipt, err := loadDeskOperation(r.s.db, id)
	if err != nil {
		r.respondError(w, err)
		return
	}
	state := receipt.Steps[len(receipt.Steps)-1].State
	if state == "observed" {
		respond(w, 200, r.deskReply(receipt))
		return
	}
	if state == "uncertain" && deskwork.IsReleaseAction(receipt.Intent.Action) {
		r.queueDeskReleaseReconciliation(receipt, actor)
		respond(w, 202, r.deskReply(receipt))
		return
	}
	if state == "uncertain" && receipt.Intent.Action == "commit" {
		r.queueDeskCommitReconciliation(receipt, actor)
		respond(w, 202, r.deskReply(receipt))
		return
	}
	if state != "uncertain" || receipt.Intent.Action != "create-desk" {
		r.respondError(w, conflict("desk_reconciliation_unavailable", "this receipt has no supported uncertain effect to reconcile"))
		return
	}
	r.updateWorkers++
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.mu.Lock()
		defer r.mu.Unlock()
		defer func() { r.updateWorkers-- }()
		if r.closing || r.failed.Load() || r.updateFence.Held || !r.routineAuthorized(actor) {
			return
		}
		current, err := loadDeskOperation(r.s.db, id)
		if err != nil {
			r.failed.Store(true)
			return
		}
		if current.Steps[len(current.Steps)-1].State != "uncertain" {
			return
		}
		ctx, cancel := context.WithTimeout(r.ctx, protocol.DeskReadTimeout)
		defer cancel()
		runtime, err := r.openDesk(ctx, deskSelection(current.Intent))
		if err != nil {
			return
		}
		observed, err := runtime.Observe(ctx, current.Intent.Target.Desk)
		if err != nil || !observed.Exists || observed.Validate() != nil {
			return
		}
		_, err = r.s.appendDeskStep(current, deskwork.Step{Kind: current.Intent.Action, State: "observed", Actor: actor, At: time.Now().UTC(), Observation: &observed})
		if err != nil {
			r.failed.Store(true)
		}
	}()
	respond(w, 202, r.deskReply(receipt))
}

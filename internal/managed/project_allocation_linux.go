//go:build linux && amd64

package managed

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectusage"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type projectAllocationBackend interface {
	ExpandedProjectAllocation(context.Context, runtimequalify.ProjectExpansion, projectusage.Binding) (projectusage.Observation, error)
}

func (realBackend) ExpandedProjectAllocation(ctx context.Context, state runtimequalify.ProjectExpansion, binding projectusage.Binding) (projectusage.Observation, error) {
	return runtimequalify.ReadExpandedProjectAllocation(ctx, state, binding)
}
func (r *Server) projectAllocationFeatures() []string {
	if _, ok := r.b.(projectAllocationBackend); ok {
		return []string{protocol.ProjectAllocationFeature}
	}
	return nil
}

func projectAllocationMatchesState(state runtimequalify.ProjectExpansion, out projectusage.Observation) bool {
	if len(state.Members) == 0 || len(state.Members) > 32 || len(out.DeclaredRoots) != 2*len(state.Members)+1 {
		return false
	}
	base := ""
	for index, member := range state.Members {
		mount, cache := out.DeclaredRoots[index*2], out.DeclaredRoots[index*2+1]
		selected := member.Creator.Storage
		if member.Initialized.Identity.Inode != 0 {
			selected = member.Initialized
		}
		if mount.ID != fmt.Sprintf("mount-%d", index) || mount.Kind != "member-mount" || mount.InstanceID != member.Member.InstanceID || mount.RuntimeID != member.Creator.RuntimeID || mount.Expected != selected.MountIdentity || !strings.HasSuffix(mount.Path, "/"+pierstore.MountPath(pierstore.Relative(selected.Path))) {
			return false
		}
		if cache.ID != fmt.Sprintf("runtime-%d", index) || cache.Kind != "runtime-cache" || cache.InstanceID != member.Member.InstanceID || cache.RuntimeID != member.Creator.RuntimeID || !strings.HasSuffix(cache.Path, "/.local/share/urbit-runner/managed/"+member.Creator.RuntimeID) {
			return false
		}
		if base == "" {
			base = filepath.Dir(cache.Path)
		} else if base != filepath.Dir(cache.Path) {
			return false
		}
	}
	last := out.DeclaredRoots[len(out.DeclaredRoots)-1]
	return last.ID == "project-cache" && last.Kind == "project-cache" && last.Path == filepath.Join(base, "project-resources-"+state.Project.ProjectID)
}

func (s *store) projectAllocationSelection(ctx context.Context, in projectusage.Request) (runtimequalify.ProjectExpansion, projectusage.Binding, error) {
	var zero runtimequalify.ProjectExpansion
	var binding projectusage.Binding
	if in.Validate() != nil || in.RunnerID != s.runnerID {
		return zero, binding, bad("exact private project allocation selection required")
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return zero, binding, err
	}
	defer tx.Rollback()
	if _, err = projectReservations(tx); err != nil {
		return zero, binding, err
	}
	p, err := loadProject(tx, in.ProjectID)
	if err != nil {
		return zero, binding, err
	}
	digest, err := p.Plan.Digest()
	if err != nil || p.Revision != in.ExpectedRevision || digest != in.ReviewDigest {
		return zero, binding, conflict("project_allocation_selection_changed", "project revision or plan changed; select saved status again")
	}
	state, err := loadProjectExpansion(tx, in.ProjectID)
	kind := projectusage.ExpandedLedger
	var body []byte
	if errors.Is(err, sql.ErrNoRows) {
		state, body, err = projectOriginalAllocationSelection(ctx, tx, p, s.runnerID)
		kind = projectusage.OriginalHistory
	} else if err == nil {
		err = tx.QueryRowContext(ctx, "SELECT body FROM project_expansions WHERE project_id=?", in.ProjectID).Scan(&body)
	}
	if err != nil {
		return zero, binding, err
	}
	if state.Project.RunnerID != s.runnerID || state.Project.Revision != p.Revision || state.Project.Generation == 0 {
		return zero, binding, conflict("project_allocation_selection_changed", "saved project allocation binding changed")
	}
	plan, err := state.Project.Plan.Digest()
	if err != nil || plan != digest {
		return zero, binding, conflict("project_allocation_selection_changed", "saved member plan changed")
	}
	binding = projectusage.Binding{RunnerID: s.runnerID, Mode: s.mode, ProjectID: in.ProjectID, Revision: p.Revision, Generation: state.Project.Generation, PlanDigest: digest, LedgerKind: kind, LedgerSHA256: stopEvidenceDigest(body)}
	if binding.Validate() != nil {
		return zero, binding, bad("invalid saved project allocation binding")
	}
	return state, binding, nil
}

func (r *Server) serveProjectAllocation(w http.ResponseWriter, req *http.Request, role string) {
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "exact project allocation is operator-private"}})
		return
	}
	b, ok := r.b.(projectAllocationBackend)
	if !ok {
		r.respondError(w, &fault{501, protocol.APIError{Code: "unsupported", Message: "project allocation adapter unavailable"}})
		return
	}
	var in projectusage.Request
	if err := decode(w, req, &in); err != nil {
		r.respondError(w, err)
		return
	}
	if in.Validate() != nil {
		r.respondError(w, bad("select exact runner/project/revision/plan and private-data consent"))
		return
	}
	ctx, cancel := context.WithTimeout(req.Context(), protocol.ProjectAllocationTimeout)
	defer cancel()
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(protocol.ProjectAllocationTimeout + time.Second))
	r.mu.Lock()
	if r.projectAllocationActive {
		r.mu.Unlock()
		r.respondError(w, conflict("project_allocation_busy", "one bounded storage census is already running; retry this read later with its saved selection"))
		return
	}
	state, binding, err := r.s.projectAllocationSelection(ctx, in)
	if err == nil {
		r.projectAllocationActive = true
	}
	r.mu.Unlock()
	if err != nil {
		r.respondError(w, err)
		return
	}
	defer func() { r.mu.Lock(); r.projectAllocationActive = false; r.mu.Unlock() }()
	out, err := b.ExpandedProjectAllocation(ctx, state, binding)
	if err != nil || out.Validate() != nil || out.Binding != binding || !projectAllocationMatchesState(state, out) || out.Growth != nil || out.StartedAt.After(time.Now().Add(time.Second)) || out.ReadAt.After(time.Now().Add(time.Second)) || time.Since(out.StartedAt) > protocol.ProjectAllocationTimeout+time.Second {
		r.mu.Lock()
		if r.projectAllocations != nil {
			delete(r.projectAllocations, in.ProjectID)
		}
		r.mu.Unlock()
		r.respondError(w, conflict("project_allocation_unavailable", "bound storage metadata is unavailable or changed; preserve enrolled paths and original receipts"))
		return
	}
	r.mu.Lock()
	after, next, err := r.s.projectAllocationSelection(ctx, in)
	if err != nil || next != binding || !reflect.DeepEqual(state, after) {
		delete(r.projectAllocations, in.ProjectID)
		r.mu.Unlock()
		r.respondError(w, conflict("project_allocation_selection_changed", "project member ledger changed during storage observation"))
		return
	}
	growth := r.projectAllocationGrowth(out, time.Now())
	out.Growth = &growth
	r.mu.Unlock()
	respond(w, 200, out)
}

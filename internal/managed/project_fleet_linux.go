//go:build linux && amd64

package managed

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"sync"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type projectFleetBackend interface {
	ProjectFleet(context.Context, runtimequalify.ProjectCurrentReadinessSelection) ([]protocol.Instance, error)
	ProjectLoginCode(context.Context, runtimequalify.ProjectCurrentReadinessSelection, string, string) (string, error)
}

func (realBackend) ProjectFleet(ctx context.Context, s runtimequalify.ProjectCurrentReadinessSelection) ([]protocol.Instance, error) {
	return runtimequalify.ReadProjectFleet(ctx, s)
}
func (realBackend) ProjectLoginCode(ctx context.Context, s runtimequalify.ProjectCurrentReadinessSelection, id, container string) (string, error) {
	return runtimequalify.ReadProjectLoginCode(ctx, s, id, container)
}

func (s *store) projectFleetSelection(project, instance string) (runtimequalify.ProjectCurrentReadinessSelection, error) {
	var zero runtimequalify.ProjectCurrentReadinessSelection
	var active int
	if err := s.db.QueryRow("SELECT count(*) FROM operations WHERE json_extract(body,'$.project_id')=? AND json_extract(body,'$.status') IN ('queued','running')", project).Scan(&active); err != nil {
		return zero, err
	}
	if active != 0 {
		return zero, errors.New("project operation is in progress")
	}
	jobs, err := projectMemberReadinessJobs(s.db)
	if err != nil {
		return zero, err
	}
	for _, j := range jobs {
		if j.ProjectID != project || j.Phase != "observed" || j.Evidence == nil || instance != "" && j.InstanceID != instance {
			continue
		}
		src, err := projectContextReadinessSource(s.db, project, j.InstanceID, j.OperationID, nil)
		if err != nil {
			return zero, err
		}
		a := src.Launches[0].Anchor.Prepared
		return runtimequalify.ProjectCurrentReadinessSelection{Project: runtimequalify.ProjectPreparationSelection{RunnerID: s.runnerID, ProjectID: project, OperationID: a.OperationID, Revision: src.Project.Revision, Generation: a.Generation, Plan: src.Project.Plan}, Launches: src.Launches, Resumes: src.Resumes, Original: src.Original, RecoveryHistory: src.RecoveryHistory}, nil
	}
	return zero, errors.New("project has not completed its initial boot")
}

func (r *Server) attachProjectFleet(ctx context.Context, snapshot *protocol.Snapshot) {
	b, ok := r.b.(projectFleetBackend)
	if !ok {
		return
	}
	inventory, err := r.s.projectInventory(ctx)
	if err != nil {
		return
	}
	type result struct {
		rows []protocol.Instance
		err  error
	}
	results := make([]result, len(inventory.Projects))
	slots := make(chan struct{}, 4)
	var wg sync.WaitGroup
	for index, p := range inventory.Projects {
		// Capture immutable intent under the mutation lock. Slow network and
		// stats reads must not block Stop or serialize unrelated projects.
		r.mu.Lock()
		expanded, expansionErr := loadProjectExpansion(r.s.db, p.ID)
		var selection runtimequalify.ProjectCurrentReadinessSelection
		selectedErr := expansionErr
		if errors.Is(expansionErr, sql.ErrNoRows) {
			selection, selectedErr = r.s.projectFleetSelection(p.ID, "")
		}
		r.mu.Unlock()
		wg.Add(1)
		go func(index int, expanded runtimequalify.ProjectExpansion, selection runtimequalify.ProjectCurrentReadinessSelection, expandedOK bool, selectedErr error) {
			defer wg.Done()
			if selectedErr != nil {
				results[index].err = selectedErr
				return
			}
			select {
			case slots <- struct{}{}:
				defer func() { <-slots }()
			case <-ctx.Done():
				results[index].err = ctx.Err()
				return
			}
			if expandedOK {
				if eb, ok := r.b.(projectExpansionBackend); ok {
					results[index].rows, results[index].err = eb.ExpandedProjectFleet(ctx, expanded)
				} else {
					results[index].err = errors.New("expanded fleet adapter unavailable")
				}
			} else {
				results[index].rows, results[index].err = b.ProjectFleet(ctx, selection)
			}
		}(index, expanded, selection, expansionErr == nil, selectedErr)
	}
	wg.Wait()
	for index, p := range inventory.Projects {
		rows := results[index].rows
		if results[index].err != nil {
			for _, m := range append(append([]protocol.ProjectMemberReservation{}, p.Members...), p.AdditionalMembers...) {
				rows = append(rows, protocol.Instance{ID: m.InstanceID, Label: m.InstanceID, Ship: m.Ship, Mode: "fake", ProjectID: p.ID, ProjectLabel: p.Label, ConfigRevision: p.Revision, Config: protocol.Configuration{HTTPPort: m.HTTPPort, Loom: 31}, ObservedState: "unknown", Health: "unknown"})
			}
		}
		for i, row := range rows {
			for _, op := range snapshot.Operations {
				if op.ProjectID == p.ID && op.InstanceID == row.ID && (op.Status == "queued" || op.Status == "running") {
					rows[i].ActiveOperationID = op.ID
				}
			}
		}
		snapshot.Instances = append(snapshot.Instances, rows...)
	}
}

// Called under the server mutation lock, exactly like ordinary code retrieval.
func (r *Server) projectLoginCode(w http.ResponseWriter, req *http.Request, actor, id string, body protocol.LoginCodeRequest) bool {
	var project string
	if err := r.s.db.QueryRow("SELECT project_id FROM project_members WHERE instance_id=? UNION SELECT project_id FROM project_expansion_members WHERE instance_id=?", id, id).Scan(&project); err != nil {
		return false
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(protocol.LoginCodeReplyTimeout))
	ctx, cancel := context.WithTimeout(req.Context(), protocol.LoginCodeReadTimeout)
	defer cancel()
	var code string
	var err error
	var active int
	if err = r.s.db.QueryRow("SELECT count(*) FROM operations WHERE json_extract(body,'$.project_id')=? AND json_extract(body,'$.status') IN ('queued','running')", project).Scan(&active); err != nil || active > 0 {
		r.respondError(w, conflict("runtime_changed", "wait for the project operation to finish"))
		return true
	}
	if expanded, e := loadProjectExpansion(r.s.db, project); e == nil {
		eb, ok := r.b.(projectExpansionBackend)
		if !ok || expanded.Project.Revision != body.ExpectedRevision {
			r.respondError(w, bad("project selection changed"))
			return true
		}
		code, err = eb.ExpandedProjectCode(ctx, expanded, id, body.ContainerID)
	} else if !errors.Is(e, sql.ErrNoRows) {
		r.respondError(w, e)
		return true
	} else {
		s, e := r.s.projectFleetSelection(project, id)
		b, ok := r.b.(projectFleetBackend)
		if e != nil || !ok || s.Project.Revision != body.ExpectedRevision {
			r.respondError(w, conflict("runtime_changed", "select a ready project member"))
			return true
		}
		code, err = b.ProjectLoginCode(ctx, s, id, body.ContainerID)
	}
	success := err == nil && ctx.Err() == nil && protocol.ValidLoginCode(code)
	if err := r.s.auditCode(actor, id, body.ExpectedRevision, success, err, ctx.Err() == context.DeadlineExceeded); err != nil {
		r.failed.Store(true)
		r.respondError(w, err)
		return true
	}
	if !success {
		r.respondError(w, &fault{502, protocol.APIError{Code: "code_unavailable", Message: "login code could not be read from the selected project runtime"}})
		return true
	}
	respond(w, 200, protocol.LoginCode{RunnerID: r.s.runnerID, InstanceID: id, ConfigRevision: body.ExpectedRevision, ContainerID: body.ContainerID, Code: code})
	return true
}

type projectLogsBackend interface {
	ProjectLogs(context.Context, runtimequalify.ProjectCurrentReadinessSelection, string) (protocol.RuntimeLogs, error)
	ExpandedProjectLogs(context.Context, runtimequalify.ProjectExpansion, string) (protocol.RuntimeLogs, error)
}

func (realBackend) ProjectLogs(ctx context.Context, s runtimequalify.ProjectCurrentReadinessSelection, id string) (protocol.RuntimeLogs, error) {
	return runtimequalify.ReadProjectLogs(ctx, s, id)
}
func (realBackend) ExpandedProjectLogs(ctx context.Context, s runtimequalify.ProjectExpansion, id string) (protocol.RuntimeLogs, error) {
	return runtimequalify.ReadExpandedProjectLogs(ctx, s, id)
}
func (r *Server) projectLogs(w http.ResponseWriter, req *http.Request, id string, body protocol.LogsRequest) bool {
	var project string
	if err := r.s.db.QueryRow("SELECT project_id FROM project_members WHERE instance_id=? UNION SELECT project_id FROM project_expansion_members WHERE instance_id=?", id, id).Scan(&project); errors.Is(err, sql.ErrNoRows) {
		return false
	} else if err != nil {
		r.respondError(w, err)
		return true
	}
	b, ok := r.b.(projectLogsBackend)
	if !ok {
		r.respondError(w, bad("project logs unavailable"))
		return true
	}
	ctx, cancel := context.WithTimeout(req.Context(), 4*time.Second)
	defer cancel()
	var logs protocol.RuntimeLogs
	var err error
	var revision uint64
	if s, e := loadProjectExpansion(r.s.db, project); e == nil {
		revision = s.Project.Revision
		logs, err = b.ExpandedProjectLogs(ctx, s, id)
	} else if !errors.Is(e, sql.ErrNoRows) {
		err = e
	} else {
		s, e := r.s.projectFleetSelection(project, id)
		err = e
		if e == nil {
			revision = s.Project.Revision
			logs, err = b.ProjectLogs(ctx, s, id)
		}
	}
	if err != nil || revision != body.ExpectedRevision {
		r.respondError(w, conflict("runtime_changed", "project runtime changed; refresh before reading logs"))
		return true
	}
	logs.RunnerID = r.s.runnerID
	logs.InstanceID = id
	logs.ConfigRevision = revision
	respond(w, 200, logs)
	return true
}

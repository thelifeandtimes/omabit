//go:build linux && amd64

package managed

import (
	"context"
	"errors"
	"log"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type projectAnchorStartBackend interface {
	OpenProjectAnchorStart(context.Context, runtimequalify.ProjectPreparationSelection, projectnetwork.Preparation) (projectnetwork.AnchorStartEngine, error)
}

func (realBackend) OpenProjectAnchorStart(ctx context.Context, selection runtimequalify.ProjectPreparationSelection, prepared projectnetwork.Preparation) (projectnetwork.AnchorStartEngine, error) {
	engine, err := runtimequalify.OpenProjectAnchorStart(ctx, selection, prepared)
	if err != nil {
		return nil, err
	}
	return &anchorStartDiagnostic{AnchorStartEngine: engine, project: selection.ProjectID}, nil
}

// This wrapper reports only fixed adapter diagnostics: Docker stdout/stderr are
// discarded by runtimequalify before an error reaches this boundary. It never
// logs credentials, request bodies or ship application data.
type anchorStartDiagnostic struct {
	projectnetwork.AnchorStartEngine
	project string
}

func (d *anchorStartDiagnostic) InspectAnchor(ctx context.Context, prepared projectnetwork.Preparation) (projectnetwork.Observation, error) {
	observed, err := d.AnchorStartEngine.InspectAnchor(ctx, prepared)
	if err != nil {
		message := err.Error()
		if len(message) > 512 {
			message = message[:512]
		}
		log.Printf("project-anchor-start project=%s inspection_failed=%q", d.project, message)
	}
	return observed, err
}

// Explicit initial anchor Start; the public route preserves this queue and fences.
func (r *Server) queueProjectAnchorStart(actor, id string, req projectAnchorStartRequest) (protocol.Operation, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closing || r.failed.Load() || r.ctx.Err() != nil {
		return protocol.Operation{}, false, conflict("runner_unavailable", "runner is stopping or state storage needs review")
	}
	if r.updateFence.Held {
		return protocol.Operation{}, false, conflict("management_quiesced", "management update holds admission; resume the activation journal")
	}
	b, ok := r.b.(projectAnchorStartBackend)
	if !ok {
		return protocol.Operation{}, false, bad("project anchor Start is unavailable on this runner")
	}
	op, replayed, err := r.s.acceptProjectAnchorStart(actor, id, req)
	if err != nil || replayed {
		return op, replayed, err
	}
	r.updateWorkers++
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer func() { r.mu.Lock(); r.updateWorkers--; r.mu.Unlock() }()
		var runErr error
		select {
		case r.workGate <- struct{}{}:
			defer func() { <-r.workGate }()
			runErr = r.executeProjectAnchorStart(b, op)
		case <-r.ctx.Done():
			runErr = r.ctx.Err()
		}
		if runErr != nil {
			log.Printf("project-anchor-start project=%s operation=%s stopped=%q", op.ProjectID, op.ID, runErr.Error())
			if err := r.s.interruptAnchorStart(op.ProjectID, op.ID); err != nil {
				r.failed.Store(true)
			}
		}
	}()
	return op, false, nil
}
func (r *Server) executeProjectAnchorStart(b projectAnchorStartBackend, op protocol.Operation) error {
	if err := r.ctx.Err(); err != nil {
		return err
	}
	if err := validateProjectJobs(r.s.db); err != nil {
		return err
	}
	if err := validateProjectAnchorStarts(r.s.db); err != nil {
		return err
	}
	actual, err := loadOperation(r.s.db, op.ID)
	if err != nil {
		return err
	}
	state, err := loadProjectAnchorStart(r.s.db, op.ProjectID)
	if err != nil {
		return err
	}
	p, err := loadProject(r.s.db, op.ProjectID)
	if err != nil {
		return err
	}
	if actual.Status != "queued" || actual.ProjectID != p.ID || actual.ConfigRevision != p.Revision || state.OperationID != op.ID || state.Phase != "planned" {
		return errors.New("project anchor Start is no longer freshly queued")
	}
	if err = projectRetirementClear(r.s.db, op.ProjectID, op.ID); err != nil {
		return err
	}
	if err = projectMaintenanceClear(r.s); err != nil {
		return err
	}
	selection := runtimequalify.ProjectPreparationSelection{RunnerID: r.s.runnerID, ProjectID: p.ID, Revision: p.Revision, OperationID: state.Prepared.OperationID, Generation: state.Prepared.Generation, Plan: p.Plan}
	engine, err := b.OpenProjectAnchorStart(r.ctx, selection, state.Prepared)
	if err != nil {
		return err
	}
	return r.s.runProjectAnchorStart(r.ctx, p.ID, op.ID, engine)
}
func projectMaintenanceClear(s *store) error {
	all, err := instances(s.db)
	if err != nil {
		return err
	}
	for _, v := range all {
		for _, c := range v.Runtime.Runtime.Containers {
			if (c.Kind == "pack" || c.Kind == "meld" || c.Kind == "roll" || c.Kind == "comet-pack" || c.Kind == "comet-meld" || c.Kind == "comet-roll") && (!c.Removed || c.CreateUncertain) {
				return conflict("host_recovery_required", "resolve retained maintenance resources before project work")
			}
		}
	}
	return nil
}

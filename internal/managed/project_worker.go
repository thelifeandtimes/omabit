package managed

import (
	"context"
	"errors"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

// runProjectPreparation requires the runner-wide work/management fence and a
// verified engine adapter. The explicit server queue supplies these; no automatic
// startup path schedules project preparation.
func (s *store) runProjectPreparation(ctx context.Context, id, opID string, prepared projectnetwork.Preparation, engine projectnetwork.PreparationEngine) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if engine == nil {
		return errors.New("project preparation engine is required")
	}
	if err := s.beginProjectPreparation(id, opID, prepared); err != nil {
		return err
	}
	p, err := loadProject(s.db, id)
	if err == nil {
		_, err = projectnetwork.ExecutePreparation(ctx, p.Plan, prepared, engine, func(sequence uint64, kind, phase, resourceID string) error {
			return s.advanceProjectPreparation(id, opID, sequence, kind, phase, resourceID)
		})
	}
	if err == nil {
		return nil
	}
	return errors.Join(err, s.interruptProjectPreparation(id, opID))
}

// Publish uncertainty even if setup failed before begin or the caller canceled.
// Completed receipts stay unchanged. The store lives until all server workers exit.
func (s *store) interruptProjectPreparation(id, opID string) error {
	// The caller's cancellation cannot cancel publication of known uncertainty.
	// No inverse mutation or retry is sent to the engine on this failure path.
	tx, saveErr := s.db.Begin()
	if saveErr != nil {
		return saveErr
	}
	defer tx.Rollback()
	j, saveErr := loadProjectJob(tx, id)
	if saveErr != nil {
		return saveErr
	}
	op, saveErr := loadOperation(tx, opID)
	if saveErr != nil {
		return saveErr
	}
	if j.OperationID != opID || op.ProjectID != id {
		return errors.New("project preparation lost its operation ownership")
	}
	if op.Status == "running" || op.Status == "queued" {
		if saveErr = interruptProjectOperation(tx, op); saveErr != nil {
			return saveErr
		}
		op, saveErr = loadOperation(tx, opID)
		if saveErr != nil {
			return saveErr
		}
		op.Message = "project resource preparation did not finish; retain its original journal and inspect before recovery"
		if saveErr = saveOperation(tx, op); saveErr != nil {
			return saveErr
		}
	}
	return tx.Commit()
}

package managed

import (
	"database/sql"
	"errors"
)

// Only fresh user selection, after global validators and actor-scoped replay.
// Historical readers still treat missing stored references as corruption.
func projectRecoveryReceiptAdmission(q querier, project, id string) error {
	if _, err := loadProject(q, project); err != nil {
		return err
	}
	op, err := loadOperation(q, id)
	if errors.Is(err, sql.ErrNoRows) {
		return conflict("project_recovery_not_found", "selected recovery receipt does not exist; review saved project status")
	}
	if err != nil {
		return err
	}
	if op.ProjectID != project || op.Kind != "recover-project-anchor" || op.InstanceID != "" {
		return conflict("project_recovery_mismatch", "select this project's coordinated recovery parent")
	}
	return nil
}

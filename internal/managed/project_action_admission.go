package managed

import (
	"database/sql"
	"errors"
	"slices"
)

// Admission checks distinguish a caller's missing or mismatched selection from a
// broken stored causal chain. Call only for a fresh request, after journal
// validation. Historical readers and actor-scoped replay keep their own rules.
func projectMemberAdmission(q querier, project, instance string) error {
	p, err := loadProject(q, project)
	if err != nil {
		return err
	}
	for _, m := range p.Members {
		if m.InstanceID == instance {
			return nil
		}
	}
	return conflict("project_member_not_found", "select a member reserved by this project")
}
func projectReceiptAdmission(q querier, project, instance, id string, kinds ...string) error {
	if err := projectMemberAdmission(q, project, instance); err != nil {
		return err
	}
	op, err := loadOperation(q, id)
	if errors.Is(err, sql.ErrNoRows) {
		return conflict("project_prerequisite_missing", "selected prerequisite receipt does not exist; review saved project status")
	}
	if err != nil {
		return err
	}
	if op.ProjectID != project || op.InstanceID != instance || !slices.Contains(kinds, op.Kind) {
		return conflict("project_prerequisite_mismatch", "select the required receipt for this exact project member")
	}
	return nil
}

package managed

import (
	"database/sql"
	"errors"
	"reflect"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

// Rebuild the parent and its sole referencing journal in the same migration
// transaction, with foreign keys enabled throughout. Copy bodies verbatim: old
// readiness, interrupted work and actor-scoped receipts remain historical facts.
func migrateRepeatedMemberResumes(tx *sql.Tx, version int) error {
	if version < 32 || version >= 35 {
		return nil
	}
	var enabled int
	if err := tx.QueryRow("PRAGMA foreign_keys").Scan(&enabled); err != nil {
		return err
	}
	if enabled != 1 {
		return errors.New("member resume migration requires foreign keys")
	}
	var unexpected int
	if err := tx.QueryRow(`SELECT count(*) FROM sqlite_master m JOIN pragma_foreign_key_list(m.name) f WHERE m.type='table' AND f.[table]='project_member_resumes' AND NOT (m.name='project_member_resumed_readiness' AND f.[from]='resume_operation_id' AND f.[to]='operation_id')`).Scan(&unexpected); err != nil {
		return err
	}
	if unexpected != 0 {
		return errors.New("unknown member resume journal reference")
	}
	var child int
	if err := tx.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='project_member_resumed_readiness'").Scan(&child); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE TABLE project_member_resumes_v35 (operation_id TEXT PRIMARY KEY REFERENCES operations(id),project_id TEXT NOT NULL REFERENCES projects(id),instance_id TEXT NOT NULL REFERENCES project_members(instance_id),retirement_operation_id TEXT NOT NULL UNIQUE REFERENCES project_member_retirements(operation_id),body BLOB NOT NULL);
 INSERT INTO project_member_resumes_v35 SELECT * FROM project_member_resumes;`); err != nil {
		return err
	}
	if child != 0 {
		if _, err := tx.Exec(`CREATE TEMP TABLE retained_resumed_readiness_v35 AS SELECT * FROM project_member_resumed_readiness;
 DROP TABLE project_member_resumed_readiness;`); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`DROP TABLE project_member_resumes; ALTER TABLE project_member_resumes_v35 RENAME TO project_member_resumes;`); err != nil {
		return err
	}
	if child != 0 {
		if err := initializeProjectMemberResumedReadiness(tx); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO project_member_resumed_readiness SELECT * FROM retained_resumed_readiness_v35; DROP TABLE retained_resumed_readiness_v35;`); err != nil {
			return err
		}
	}
	rows, err := tx.Query("PRAGMA foreign_key_check")
	if err != nil {
		return err
	}
	damaged := rows.Next()
	err = errors.Join(rows.Err(), rows.Close())
	if err != nil {
		return err
	}
	if damaged {
		return errors.New("member resume migration lost foreign key integrity")
	}
	return nil
}

// Select by exact causal binding, never by a timestamp or the largest counter.
// Historical validation may resolve an earlier prefix; fresh mutation must use
// the latest complete history. Unfinished replacement dispatch stays fenced.
func memberResumeGeneration(q querier, retired projectnetwork.MemberRetirement, current bool) (uint64, error) {
	stop := retired.Exit.Stop
	p, launches, _, _, err := projectReadinessPrerequisites(q, stop.Binding.ProjectID, stop.InstanceID)
	if err != nil {
		return 0, err
	}
	resumes, err := completedProjectMemberResumes(q, p.ID)
	if err != nil {
		return 0, err
	}
	selected, err := recoveryGenerationSelection(q, p, launches, resumes, stop.Binding, current)
	if err != nil {
		return 0, err
	}
	if !reflect.DeepEqual(selected.Binding, stop.Binding) {
		return 0, errors.New("retirement is not the current project generation")
	}
	for _, m := range selected.Members {
		if m.InstanceID != stop.InstanceID {
			continue
		}
		if m.OperationID != stop.LaunchOperationID || m.Storage.Path != stop.Storage.Path || m.Storage.MountIdentity != stop.Storage.MountIdentity || ((m.Mode == "resume" || m.Mode == "recovery") && m.Storage != stop.Storage) {
			return 0, errors.New("retirement lost its original member creation or storage")
		}
		return m.Generation, nil
	}
	return 0, errors.New("retirement lost its member generation")
}

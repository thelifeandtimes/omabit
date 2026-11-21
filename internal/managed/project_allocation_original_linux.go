//go:build linux && amd64

package managed

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

const maxAllocationJournalBytes = 4 << 20

// Derive current storage from completed original journals, never by persisting an
// expansion, adopting a path or manufacturing readiness. This initial adapter
// refuses any retirement/resume/recovery history until its own coverage exists.
func projectOriginalAllocationSelection(ctx context.Context, tx *sql.Tx, p storedProject, runner string) (runtimequalify.ProjectExpansion, []byte, error) {
	var zero runtimequalify.ProjectExpansion
	unsupported := func() (runtimequalify.ProjectExpansion, []byte, error) {
		return zero, nil, conflict("project_allocation_unsupported_phase", "allocation requires current expanded storage or complete unchanged original boot/readiness journals; reservations, replacement and recovery histories need their own adapter")
	}
	var changed int
	if err := tx.QueryRowContext(ctx, `SELECT
 (SELECT count(*) FROM project_member_retirements WHERE project_id=?) +
 (SELECT count(*) FROM project_member_resumes WHERE project_id=?) +
 (SELECT count(*) FROM project_anchor_recoveries WHERE project_id=?)`, p.ID, p.ID, p.ID).Scan(&changed); err != nil {
		return zero, nil, err
	}
	if changed != 0 {
		return unsupported()
	}
	// Bound selected journal allocation before loading bodies or decoding them.
	for _, query := range []string{
		"SELECT count(*),COALESCE(sum(length(CAST(body AS BLOB))+length(COALESCE(runtime,x''))+length(CAST(fingerprint AS BLOB))+length(CAST(id AS BLOB))),0) FROM operations WHERE json_extract(body,'$.project_id')=?",
		"SELECT count(*),COALESCE(sum(length(CAST(body AS BLOB))),0) FROM project_member_readiness WHERE project_id=?",
	} {
		var count, bytes int64
		if err := tx.QueryRowContext(ctx, query, p.ID).Scan(&count, &bytes); err != nil {
			return zero, nil, err
		}
		if count > 1024 || bytes > maxAllocationJournalBytes {
			return zero, nil, conflict("project_allocation_history_limit", "selected original journals exceed this bounded allocation adapter; no partial storage is reported")
		}
	}
	if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM operations WHERE json_extract(body,'$.project_id')=? AND json_extract(body,'$.status') IN ('queued','running')", p.ID).Scan(&changed); err != nil {
		return zero, nil, err
	}
	if changed != 0 {
		return unsupported()
	}
	jobRows, err := tx.QueryContext(ctx, "SELECT operation_id FROM project_member_readiness WHERE project_id=? ORDER BY operation_id LIMIT 1025", p.ID)
	if err != nil {
		return zero, nil, err
	}
	ids := []string{}
	for jobRows.Next() {
		var id string
		if err = jobRows.Scan(&id); err != nil {
			break
		}
		ids = append(ids, id)
	}
	err = errors.Join(err, jobRows.Err(), jobRows.Close())
	if err != nil {
		return zero, nil, err
	}
	if len(ids) > 1024 {
		return unsupported()
	}
	jobs := []projectMemberReadinessJob{}
	for _, id := range ids {
		job, e := loadProjectMemberReadiness(tx, id)
		if e != nil {
			return zero, nil, e
		}
		jobs = append(jobs, job)
	}
	var state runtimequalify.ProjectExpansion
	proofs := [][]byte{}
	for index, member := range p.Plan.Members {
		var initialized pierstore.Location
		found := false
		for _, job := range jobs {
			if job.ProjectID != p.ID || job.InstanceID != member.InstanceID || job.Phase != "observed" || job.Evidence == nil {
				continue
			}
			src, err := projectContextReadinessSource(tx, p.ID, member.InstanceID, job.OperationID, nil)
			if err != nil {
				return zero, nil, err
			}
			if src.RecoveryHistory != nil || len(src.Resumes) != 0 || src.Context.Member.Mode != "boot" || src.Context.Member.Generation != 1 {
				return unsupported()
			}
			if found && initialized != src.Context.Storage {
				return zero, nil, errors.New("original allocation storage proofs conflict")
			}
			initialized, found = src.Context.Storage, true
			var raw []byte
			if err = tx.QueryRowContext(ctx, "SELECT body FROM project_member_readiness WHERE operation_id=?", job.OperationID).Scan(&raw); err != nil {
				return zero, nil, err
			}
			proofs = append(proofs, append([]byte(nil), raw...))
			if index == 0 && state.Version == 0 {
				a := src.Launches[0].Anchor.Prepared
				state, err = runtimequalify.SeedProjectExpansion(runtimequalify.ProjectCurrentReadinessSelection{Project: runtimequalify.ProjectPreparationSelection{RunnerID: runner, ProjectID: p.ID, OperationID: a.OperationID, Revision: p.Revision, Generation: a.Generation, Plan: p.Plan}, Launches: src.Launches, Original: src.Original})
				if err != nil {
					return zero, nil, err
				}
			}
		}
		if !found || initialized.Identity.Inode == 0 || len(state.Members) != len(p.Plan.Members) {
			return unsupported()
		}
		m := &state.Members[index]
		if m.Member.InstanceID != member.InstanceID || m.Creator.Storage.Path != initialized.Path || m.Creator.Storage.MountIdentity != initialized.MountIdentity {
			return zero, nil, errors.New("original allocation member storage changed")
		}
		m.Initialized = initialized
	}
	if state.Version != 1 || !reflect.DeepEqual(state.Project.Plan, p.Plan) {
		return unsupported()
	}
	// Bind every raw operation/mirror/fingerprint of this project, bounded to
	// 1024 receipts. This catches a Stop/readiness admission/completion during the
	// census even when the immutable creation/storage projection stays identical.
	type rawOperation struct {
		ID            string
		Body, Runtime []byte
		Fingerprint   string
	}
	rows, err := tx.QueryContext(ctx, "SELECT id,body,runtime,fingerprint FROM operations WHERE json_extract(body,'$.project_id')=? ORDER BY id LIMIT 1025", p.ID)
	if err != nil {
		return zero, nil, err
	}
	operations := []rawOperation{}
	for rows.Next() {
		var row rawOperation
		if err = rows.Scan(&row.ID, &row.Body, &row.Runtime, &row.Fingerprint); err != nil {
			break
		}
		operations = append(operations, row)
	}
	err = errors.Join(err, rows.Err(), rows.Close())
	if err != nil {
		return zero, nil, err
	}
	if len(operations) > 1024 {
		return unsupported()
	}
	body, err := json.Marshal(struct {
		Kind       string
		State      runtimequalify.ProjectExpansion
		Readiness  [][]byte
		Operations []rawOperation
	}{"original-ready-history-v1", state, proofs, operations})
	return state, body, err
}

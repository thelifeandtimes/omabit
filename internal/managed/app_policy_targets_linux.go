//go:build linux && amd64

package managed

import (
	"database/sql"
	"errors"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// Reads saved identities only: no runtime, filesystem pier or HTTP inspection.
func appPolicyTargets(q querier) ([]protocol.AppPolicyTarget, error) {
	out := []protocol.AppPolicyTarget{}
	all, err := instances(q)
	if err != nil {
		return nil, err
	}
	for _, v := range all {
		out = append(out, protocol.AppPolicyTarget{Target: apppolicy.Target{InstanceID: v.Public.ID}, Label: v.Public.Label, Ship: v.Public.Ship, Ready: !v.Public.DeletePending && v.Public.Handoff == nil})
	}
	if _, err = projectReservations(q); err != nil {
		return nil, err
	}
	rows, err := q.Query("SELECT id FROM projects ORDER BY id")
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			break
		}
		ids = append(ids, id)
	}
	if err = errors.Join(err, rows.Err(), rows.Close()); err != nil {
		return nil, err
	}
	jobs, err := projectMemberReadinessJobs(q)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		p, err := loadProject(q, id)
		if err != nil {
			return nil, err
		}
		expanded, expErr := loadProjectExpansion(q, id)
		if expErr != nil && !errors.Is(expErr, sql.ErrNoRows) {
			return nil, expErr
		}
		if expErr == nil {
			for _, m := range expanded.Members {
				out = append(out, protocol.AppPolicyTarget{Target: apppolicy.Target{InstanceID: m.Member.InstanceID, ProjectID: id, Generation: m.Creator.Generation, Creator: m.Creator.OperationID}, Label: p.Plan.Label + " / " + m.Member.InstanceID, Ship: m.Member.Ship, Ready: m.Phase == "ready"})
			}
			continue
		}
		for _, m := range p.Members {
			v := protocol.AppPolicyTarget{Target: apppolicy.Target{InstanceID: m.InstanceID, ProjectID: id}, Label: p.Plan.Label + " / " + m.InstanceID, Ship: m.Ship}
			for _, j := range jobs {
				if j.ProjectID != id || j.InstanceID != m.InstanceID || j.Phase != "observed" || j.Evidence == nil {
					continue
				}
				src, e := projectContextReadinessSource(q, id, m.InstanceID, j.OperationID, nil)
				if e != nil {
					continue
				} // unresolved history remains blocked
				v.Target.Generation = src.Context.Member.Generation
				v.Target.Creator = src.Context.Member.OperationID
				v.Ready = true
				break
			}
			out = append(out, v)
		}
	}
	return out, nil
}

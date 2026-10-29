package managed

import (
	"encoding/json"
	"errors"
	"reflect"
	"sort"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

// Recovery history is assembled from actual retained receipts. These raw readers
// deliberately avoid lifecycle validation recursion: global validation checks all
// Stop/exit/retirement chains separately, including interrupted historical work.
func projectRecoveryMirror(q querier, id string, value any) error {
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var mirror []byte
	if err = q.QueryRow("SELECT runtime FROM operations WHERE id=?", id).Scan(&mirror); err != nil {
		return err
	}
	if string(body) != string(mirror) {
		return errors.New("recovery prerequisite journal and operation mirror disagree")
	}
	return nil
}

func recoveryOriginal(q querier, p storedProject, launches []projectnetwork.MemberLaunch, id string) (projectnetwork.InitialMemberReadiness, error) {
	j, err := loadProjectMemberReadiness(q, id)
	if err != nil {
		return projectnetwork.InitialMemberReadiness{}, err
	}
	op, err := loadOperation(q, id)
	if err != nil {
		return projectnetwork.InitialMemberReadiness{}, err
	}
	if j.ProjectID != p.ID || j.Phase != "observed" || j.Evidence == nil || j.InstanceID != j.Evidence.InstanceID || j.LaunchOperationID != j.Evidence.LaunchOperationID || !reflect.DeepEqual(j.Binding, j.Evidence.Binding) || j.Evidence.Validate(p.Plan, launches) != nil || op.Kind != "observe-project-member" || op.ProjectID != p.ID || op.InstanceID != j.InstanceID || op.Actor == "" || op.Status != "succeeded" || op.ConfigRevision != p.Revision {
		return projectnetwork.InitialMemberReadiness{}, errors.New("recovery lost its original initialized-pier receipt")
	}
	if err = projectRecoveryMirror(q, id, j); err != nil {
		return projectnetwork.InitialMemberReadiness{}, err
	}
	return *j.Evidence, nil
}

func recoveryCurrent(q querier, h projectnetwork.AnchorRecoveryHistory, id string) (projectnetwork.CurrentMemberReadiness, error) {
	var empty projectnetwork.CurrentMemberReadiness
	j, err := loadProjectMemberContextReadiness(q, id)
	if err != nil {
		return empty, err
	}
	op, err := loadOperation(q, id)
	if err != nil {
		return empty, err
	}
	p, err := loadProject(q, h.Launches[0].Anchor.Prepared.ProjectID)
	if err != nil {
		return empty, err
	}
	if j.Evidence == nil || j.Phase != "observed" || j.ProjectID != p.ID || j.InstanceID != j.Evidence.InstanceID || op.Kind != "observe-current-project-member" || op.ProjectID != p.ID || op.InstanceID != j.InstanceID || op.Actor == "" || op.Status != "succeeded" || op.ConfigRevision != p.Revision {
		return empty, errors.New("recovery lost current-readiness ownership")
	}
	original, err := recoveryOriginal(q, p, h.Launches, j.OriginalReadinessOperationID)
	if err != nil {
		return empty, err
	}
	prefix, err := projectnetwork.AnchorRecoveryHistoryAtBinding(h, j.Evidence.Binding)
	if err != nil {
		return empty, err
	}
	var c projectnetwork.MemberContext
	if len(prefix.Epochs) == 0 {
		c, err = projectnetwork.SelectMemberContext(prefix.Plan, prefix.Launches, prefix.Resumes, original)
		if err == nil {
			err = j.Evidence.Validate(prefix.Plan, prefix.Launches, prefix.Resumes, original)
		}
	} else {
		c, err = projectnetwork.SelectRecoveryMemberContext(prefix, j.InstanceID)
		if err == nil {
			err = j.Evidence.ValidateRecovered(prefix)
		}
	}
	if err != nil {
		return empty, err
	}
	if !reflect.DeepEqual(c, j.Context) {
		return empty, errors.New("recovery current-readiness context changed")
	}
	if err = projectRecoveryMirror(q, id, j); err != nil {
		return empty, err
	}
	return *j.Evidence, nil
}

func recoverySourceReferences(q querier, j projectAnchorRecoveryJob) error {
	if len(j.Source.Launches) == 0 {
		return errors.New("recovery source has no initial launches")
	}
	p, launches, _, _, err := projectReadinessPrerequisites(q, j.ProjectID, j.Source.Launches[0].Storage.InstanceID)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(p.Plan, j.Source.Plan) || !reflect.DeepEqual(launches, j.Source.Launches) || len(j.OriginalReadiness) != len(j.Source.Originals) || len(j.CurrentReadiness) != len(j.Source.Current) {
		return errors.New("recovery source differs from retained project")
	}
	used := map[string]bool{}
	for i, id := range j.OriginalReadiness {
		r, err := recoveryOriginal(q, p, launches, id)
		if err != nil {
			return err
		}
		if used[id] || !reflect.DeepEqual(r, j.Source.Originals[i]) {
			return errors.New("recovery original proof differs from database")
		}
		used[id] = true
	}
	for i, id := range j.CurrentReadiness {
		r, err := recoveryCurrent(q, j.Source, id)
		if err != nil {
			return err
		}
		if used[id] || !reflect.DeepEqual(r, j.Source.Current[i]) {
			return errors.New("recovery current proof differs from database")
		}
		used[id] = true
	}
	return nil
}

func readProjectRecoveryHistory(q querier, project string, includeCurrent bool) (projectnetwork.AnchorRecoveryHistory, []string, []string, error) {
	fail := func(err error) (projectnetwork.AnchorRecoveryHistory, []string, []string, error) {
		return projectnetwork.AnchorRecoveryHistory{}, nil, nil, err
	}
	p, err := loadProject(q, project)
	if err != nil {
		return fail(err)
	}
	if len(p.Members) == 0 {
		return fail(errors.New("recovery requires project members"))
	}
	_, launches, _, _, err := projectReadinessPrerequisites(q, project, p.Members[0].InstanceID)
	if err != nil {
		return fail(err)
	}
	jobs, err := projectAnchorRecoveryJobs(q)
	if err != nil {
		return fail(err)
	}
	var records []projectnetwork.CompletedRecoveryRecord
	var originalIDs []string
	for _, j := range jobs {
		if j.ProjectID != project {
			continue
		}
		if err = validateRecoveryReceipt(q, j); err != nil {
			return fail(err)
		}
		if err = recoverySourceReferences(q, j); err != nil {
			return fail(err)
		}
		if j.Journal.Phase != "complete" {
			continue
		}
		if len(originalIDs) == 0 {
			originalIDs = append([]string(nil), j.OriginalReadiness...)
		} else if !reflect.DeepEqual(originalIDs, j.OriginalReadiness) {
			return fail(errors.New("recovery boundaries changed original proof references"))
		}
		records = append(records, projectnetwork.CompletedRecoveryRecord{Source: j.Source, Readiness: *j.Journal.Readiness})
	}
	root := projectnetwork.AnchorRecoveryHistory{Plan: p.Plan, Launches: launches}
	if len(originalIDs) == 0 {
		ready, err := projectMemberReadinessJobs(q)
		if err != nil {
			return fail(err)
		}
		// Earliest successful proof is a stable floor; later observations remain
		// separate receipts and never rewrite an existing recovery's originals.
		for _, member := range p.Members {
			var candidates []projectMemberReadinessJob
			for _, j := range ready {
				if j.ProjectID == project && j.InstanceID == member.InstanceID && j.Phase == "observed" && j.Evidence != nil {
					candidates = append(candidates, j)
				}
			}
			sort.Slice(candidates, func(i, k int) bool {
				a, b := candidates[i], candidates[k]
				if a.Evidence.CapturedAt.Equal(b.Evidence.CapturedAt) {
					return a.OperationID < b.OperationID
				}
				return a.Evidence.CapturedAt.Before(b.Evidence.CapturedAt)
			})
			if len(candidates) == 0 {
				return fail(conflict("project_not_observed", "recovery requires original readiness for every member"))
			}
			originalIDs = append(originalIDs, candidates[0].OperationID)
		}
	}
	for _, id := range originalIDs {
		r, err := recoveryOriginal(q, p, launches, id)
		if err != nil {
			return fail(err)
		}
		root.Originals = append(root.Originals, r)
	}
	resumes, err := completedProjectMemberResumes(q, project)
	if err != nil {
		return fail(err)
	}
	h, err := projectnetwork.AssembleRecoveryHistory(root, resumes, records)
	if err != nil {
		return fail(err)
	}
	// Canonicalize the ordinary suffix by its causal binding. SQL/operation-ID
	// order is not generation order, while retained source bytes stay immutable.
	resolved, err := projectnetwork.ResolveAnchorRecoveryHistory(h)
	if err != nil {
		return fail(err)
	}
	canonical, err := projectnetwork.AnchorRecoveryHistoryAtBinding(h, resolved.Binding)
	if err != nil {
		return fail(err)
	}
	h.Resumes = canonical.Resumes
	if len(h.Resumes) == 0 {
		h.Resumes = nil
	}
	// Every retained source must be an exact prefix of the complete real history,
	// including interrupted attempts. Current observations are its explicit refs.
	for _, j := range jobs {
		if j.ProjectID != project {
			continue
		}
		c, err := projectnetwork.SelectAnchorRecoveryHistoryContext(j.Source)
		if err != nil {
			return fail(err)
		}
		prefix, err := projectnetwork.AnchorRecoveryHistoryAtBinding(h, c.Binding)
		if err != nil {
			return fail(err)
		}
		if len(prefix.Epochs) == 0 {
			prefix.Epochs = nil
		}
		if len(prefix.Resumes) == 0 {
			prefix.Resumes = nil
		}
		prefix.Current = j.Source.Current
		if !reflect.DeepEqual(prefix, j.Source) {
			return fail(errors.New("recovery source is not an exact retained history prefix"))
		}
	}
	var currentIDs []string
	if includeCurrent {
		current, err := projectnetwork.ResolveAnchorRecoveryHistory(h)
		if err != nil {
			return fail(err)
		}
		ready, err := projectMemberContextReadinessJobs(q)
		if err != nil {
			return fail(err)
		}
		for _, member := range current.Members {
			var chosen *projectMemberContextReadinessJob
			for i := range ready {
				j := &ready[i]
				if j.ProjectID != project || j.InstanceID != member.InstanceID || j.Phase != "observed" || j.Evidence == nil || j.Evidence.CreatorOperationID != member.OperationID {
					continue
				}
				if _, err = recoveryCurrent(q, h, j.OperationID); err != nil {
					return fail(err)
				}
				if chosen == nil || j.Evidence.CapturedAt.After(chosen.Evidence.CapturedAt) || j.Evidence.CapturedAt.Equal(chosen.Evidence.CapturedAt) && j.OperationID < chosen.OperationID {
					chosen = j
				}
			}
			if chosen != nil {
				h.Current = append(h.Current, *chosen.Evidence)
				currentIDs = append(currentIDs, chosen.OperationID)
			}
		}
	}
	if _, err = projectnetwork.SelectAnchorRecoveryHistoryContext(h); err != nil {
		return fail(err)
	}
	return h.Clone(), originalIDs, currentIDs, nil
}

package managed

import (
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

// Loading complete creation history must not recursively invoke the lifecycle
// validators: resume -> retirement -> exit -> Stop is the existing dependency
// order. Global validation separately checks every journal and prerequisite.
func completedProjectMemberResumes(q querier, project string) ([]projectnetwork.MemberResume, error) {
	var exists int
	if err := q.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='project_member_resumes'").Scan(&exists); err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, nil
	} // Earlier migrations have not created this table yet.
	jobs, err := projectMemberResumeJobs(q)
	if err != nil {
		return nil, err
	}
	var out []projectnetwork.MemberResume
	for _, j := range jobs {
		if j.ProjectID != project {
			continue
		}
		op, err := loadOperation(q, j.OperationID)
		if err != nil {
			return nil, err
		}
		if op.Status != "succeeded" {
			continue
		}
		p, err := loadProject(q, project)
		if err != nil {
			return nil, err
		}
		if j.Resume == nil || j.Resume.Phase != "running" || j.Resume.Validate(p.Plan) != nil || j.Resume.OperationID != j.OperationID || j.Resume.Retirement.Exit.Stop.InstanceID != j.InstanceID || j.Resume.Retirement.OperationID != j.RetirementOperationID || !reflect.DeepEqual(j.Resume.Retirement, j.Retirement) || op.Kind != "resume-project-member" || op.ProjectID != project || op.InstanceID != j.InstanceID || op.ConfigRevision != p.Revision || op.Actor == "" {
			return nil, errors.New("completed replacement history lost journal ownership")
		}
		retired, err := loadProjectMemberRetirement(q, j.RetirementOperationID)
		if err != nil {
			return nil, err
		}
		retiredOp, err := loadOperation(q, j.RetirementOperationID)
		if err != nil {
			return nil, err
		}
		if retiredOp.Status != "succeeded" || retiredOp.Kind != "retire-project-member" || retiredOp.ProjectID != project || retiredOp.InstanceID != j.InstanceID || retiredOp.ConfigRevision != p.Revision || retiredOp.Actor == "" || retired.ProjectID != project || retired.InstanceID != j.InstanceID || !reflect.DeepEqual(retired.Retirement, j.Retirement) {
			return nil, errors.New("completed replacement lost original retirement")
		}
		var retiredMirror []byte
		if err = q.QueryRow("SELECT runtime FROM operations WHERE id=?", retired.OperationID).Scan(&retiredMirror); err != nil {
			return nil, err
		}
		retiredBody, _ := json.Marshal(retired)
		if string(retiredBody) != string(retiredMirror) {
			return nil, errors.New("replacement's original retirement mirror changed")
		}
		body, _ := json.Marshal(j)
		var mirror []byte
		if err = q.QueryRow("SELECT runtime FROM operations WHERE id=?", j.OperationID).Scan(&mirror); err != nil {
			return nil, err
		}
		if string(body) != string(mirror) {
			return nil, errors.New("completed replacement journal and receipt disagree")
		}
		out = append(out, *j.Resume)
	}
	return out, nil
}
func stopWasReplaced(resumes []projectnetwork.MemberResume, stop projectnetwork.MemberStop) bool {
	for _, r := range resumes {
		if reflect.DeepEqual(r.Retirement.Exit.Stop, stop) {
			return true
		}
	}
	return false
}

// Historical selection preserves old receipts. Current selection additionally
// requires the latest complete project binding, so stale readiness cannot stop a
// previous container after a replacement.
func memberStopSelection(q querier, project, instance, readiness, operation string, current bool) (storedProject, projectnetwork.MemberStop, string, error) {
	p, launches, _, _, err := projectReadinessPrerequisites(q, project, instance)
	if err != nil {
		return p, projectnetwork.MemberStop{}, "", err
	}
	resumes, err := completedProjectMemberResumes(q, project)
	if err != nil {
		return p, projectnetwork.MemberStop{}, "", err
	}
	_, initialErr := loadProjectMemberReadiness(q, readiness)
	var state projectnetwork.MemberStop
	if initialErr == nil {
		p, state, _, err = initialMemberStopSelection(q, project, instance, readiness, operation)
	} else if errors.Is(initialErr, sql.ErrNoRows) {
		r, e := loadProjectMemberResumedReadiness(q, readiness)
		if errors.Is(e, sql.ErrNoRows) {
			state, err = contextMemberStopSelection(q, p, instance, readiness, operation, current, launches, resumes)
		} else {
			if e != nil {
				return p, state, "", e
			}
			op, e := loadOperation(q, readiness)
			if e != nil {
				return p, state, "", e
			}
			if r.ProjectID != project || r.InstanceID != instance || r.Phase != "observed" || r.Evidence == nil || op.Status != "succeeded" || op.Kind != "observe-resumed-project-member" || op.ProjectID != project || op.InstanceID != instance || op.ConfigRevision != p.Revision || op.Actor == "" || r.ResumeOperationID != r.Evidence.ResumeOperationID || !reflect.DeepEqual(r.Binding, r.Evidence.Binding) {
				return p, state, "", conflict("project_not_observed", "complete a matching replacement readiness observation before this Stop")
			}
			body, _ := json.Marshal(r)
			var mirror []byte
			if e = q.QueryRow("SELECT runtime FROM operations WHERE id=?", readiness).Scan(&mirror); e != nil {
				return p, state, "", e
			}
			if string(body) != string(mirror) {
				return p, state, "", errors.New("replacement readiness journal and receipt disagree")
			}
			if h, e := completedProjectRecoveryHistory(q, project); e != nil {
				return p, state, "", e
			} else if h != nil {
				prefix, e := projectnetwork.AnchorRecoveryHistoryAtBinding(*h, r.Binding)
				if e != nil {
					return p, state, "", e
				}
				if len(prefix.Epochs) != 0 {
					return p, state, "", conflict("project_not_observed", "use current-context readiness for a recovered member")
				}
				resumes = prefix.Resumes
			}
			if current {
				state, err = projectnetwork.NewResumedMemberStop(p.Plan, launches, resumes, *r.Evidence, readiness, operation)
			} else {
				state = projectnetwork.MemberStop{OperationID: operation, LaunchOperationID: r.ResumeOperationID, ReadinessOperationID: readiness, InstanceID: instance, Binding: r.Binding, Storage: r.Evidence.Storage, Phase: "planned"}
				err = projectnetwork.ValidateResumedMemberStop(p.Plan, launches, resumes, *r.Evidence, state)
			}
		}
	} else {
		return p, state, "", initialErr
	}
	if err != nil {
		return p, state, "", err
	}
	selected, err := recoveryGenerationSelection(q, p, launches, resumes, state.Binding, current)
	if err != nil {
		return p, state, "", err
	}
	if !reflect.DeepEqual(selected.Binding, state.Binding) {
		return p, state, "", conflict("project_recovery_required", "readiness belongs to an older project membership; observe the current generation")
	}
	for _, m := range state.Binding.Members {
		if m.InstanceID == instance {
			return p, state, m.Container.ID, nil
		}
	}
	return p, state, "", errors.New("Stop lost current container binding")
}

package managed

import (
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

// Raw historical selection avoids a validator cycle through replacement history.
// Admission still validates all journals and requires the latest complete binding.
func contextMemberStopSelection(q querier, p storedProject, instance, readiness, operation string, current bool, launches []projectnetwork.MemberLaunch, resumes []projectnetwork.MemberResume) (projectnetwork.MemberStop, error) {
	j, err := loadProjectMemberContextReadiness(q, readiness)
	if err != nil {
		return projectnetwork.MemberStop{}, err
	}
	src, op, err := contextReadinessJobSource(q, j)
	if err != nil {
		return projectnetwork.MemberStop{}, err
	}
	if j.ProjectID != p.ID || j.InstanceID != instance || j.Phase != "observed" || j.Evidence == nil || op.Status != "succeeded" {
		return projectnetwork.MemberStop{}, conflict("project_not_observed", "complete matching current member readiness before Stop")
	}
	if src.RecoveryHistory != nil {
		h := src.RecoveryHistory
		if current {
			h, err = completedProjectRecoveryHistory(q, p.ID)
			if err != nil {
				return projectnetwork.MemberStop{}, err
			}
			if h == nil {
				return projectnetwork.MemberStop{}, errors.New("completed recovery disappeared")
			}
		}
		return projectnetwork.NewRecoveredCurrentMemberStop(*h, *j.Evidence, readiness, operation)
	}
	history := src.Resumes
	if current {
		history = resumes
	}
	return projectnetwork.NewCurrentMemberStop(p.Plan, launches, history, src.Original, *j.Evidence, readiness, operation)
}

func validateContextMemberStopProcess(q querier, stop projectnetwork.MemberStop) error {
	op, err := loadOperation(q, stop.ReadinessOperationID)
	if err != nil {
		return err
	}
	if op.Kind != "observe-current-project-member" {
		return nil
	}
	j, err := loadProjectMemberContextReadiness(q, stop.ReadinessOperationID)
	if err != nil {
		return err
	}
	src, readyOp, err := contextReadinessJobSource(q, j)
	if err != nil {
		return err
	}
	if j.Phase != "observed" || j.Evidence == nil || readyOp.Status != "succeeded" || j.InstanceID != stop.InstanceID {
		return errors.New("current Stop lost completed readiness")
	}
	if src.RecoveryHistory != nil {
		return projectnetwork.ValidateRecoveredCurrentMemberStop(*src.RecoveryHistory, *j.Evidence, stop)
	}
	return projectnetwork.ValidateCurrentMemberStop(src.Project.Plan, src.Launches, src.Resumes, src.Original, *j.Evidence, stop)
}

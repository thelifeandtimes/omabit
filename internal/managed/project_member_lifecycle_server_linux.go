//go:build linux && amd64

package managed

import (
	"database/sql"
	"errors"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

// Reconstruct the creation source at the retained Stop's historical binding.
// A readiness factory reopening generation3 must receive the history before its
// creation, while every retained journal is still validated against the full tail.
func projectMemberLifecycleSource(s *store, p storedProject, stop projectnetwork.MemberStop) (runtimequalify.ProjectMemberResumeSelection, error) {
	var source runtimequalify.ProjectMemberResumeSelection
	if err := validateProjectMemberContextReadiness(s.db); err != nil {
		return source, err
	}
	_, launches, _, _, err := projectReadinessPrerequisites(s.db, p.ID, stop.InstanceID)
	if err != nil {
		return source, err
	}
	all, err := completedProjectMemberResumes(s.db, p.ID)
	if err != nil {
		return source, err
	}
	var prefix []projectnetwork.MemberResume
	var recovered *projectnetwork.AnchorRecoveryHistory
	if h, e := completedProjectRecoveryHistory(s.db, p.ID); e != nil {
		return source, e
	} else if h != nil {
		selected, e := projectnetwork.AnchorRecoveryHistoryAtBinding(*h, stop.Binding)
		if e != nil {
			return source, e
		}
		prefix = selected.Resumes
		if len(selected.Epochs) > 0 {
			recovered = &selected
		}
	} else {
		prefix, err = projectnetwork.MemberResumesAtBinding(p.Plan, launches, all, stop.Binding)
	}
	if err != nil {
		return source, err
	}
	jobs, err := projectMemberStopJobs(s.db)
	if err != nil {
		return source, err
	}
	var peers []projectnetwork.MemberStop
	for _, j := range jobs {
		if j.ProjectID != p.ID || j.OperationID == stop.OperationID {
			continue
		}
		matched := false
		for _, m := range stop.Binding.Members {
			if m.InstanceID == j.InstanceID && m.Container.ID == j.ContainerID {
				matched = true
			}
		}
		if !matched {
			oldOrdinary := prefix
			if recovered != nil {
				oldOrdinary = recoveryOrdinaryHistory(*recovered)
			}
			if stopWasReplaced(oldOrdinary, j.Stop) {

				continue
			}
			return source, errors.New("lifecycle source contains an unaccounted member Stop")
		}
		if j.InstanceID == stop.InstanceID || j.Stop.Phase != "stopped" {
			return source, errors.New("another member Stop remains unresolved")
		}
		peers = append(peers, j.Stop)
	}
	a := launches[0].Anchor.Prepared
	project := runtimequalify.ProjectPreparationSelection{RunnerID: s.runnerID, ProjectID: p.ID, OperationID: a.OperationID, Revision: p.Revision, Generation: a.Generation, Plan: p.Plan}
	if ready, err := loadProjectMemberContextReadiness(s.db, stop.ReadinessOperationID); err == nil {
		src, op, err := contextReadinessJobSource(s.db, ready)
		if err != nil {
			return source, err
		}
		if ready.Evidence == nil || ready.Phase != "observed" || op.Status != "succeeded" || ready.InstanceID != stop.InstanceID {
			return source, errors.New("current readiness missing")
		}
		source.CurrentStop = &runtimequalify.ProjectCurrentMemberStopSelection{ReadinessSource: runtimequalify.ProjectCurrentReadinessSelection{Project: project, Launches: launches, Resumes: prefix, Original: src.Original, RecoveryHistory: recovered}, Readiness: *ready.Evidence, Stop: stop, StoppedPeers: peers}
		return source, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return source, err
	}
	ready, err := loadProjectMemberReadiness(s.db, stop.ReadinessOperationID)
	if err == nil {
		if ready.Evidence == nil {
			return source, errors.New("initial readiness evidence missing")
		}
		source.Stop = runtimequalify.ProjectMemberStopSelection{Project: project, Launches: launches, Readiness: *ready.Evidence, Stop: stop, StoppedPeers: peers}
	} else if errors.Is(err, sql.ErrNoRows) {
		ready, err := loadProjectMemberResumedReadiness(s.db, stop.ReadinessOperationID)
		if err != nil {
			return source, err
		}
		if ready.Evidence == nil {
			return source, errors.New("replacement readiness evidence missing")
		}
		source.ResumedStop = &runtimequalify.ProjectResumedMemberStopSelection{Project: project, Launches: launches, Resumes: prefix, Readiness: *ready.Evidence, Stop: stop, StoppedPeers: peers}
	} else {
		return source, err
	}
	return source, nil
}

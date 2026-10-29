//go:build linux && amd64

package managed

import (
	"errors"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// All setup and member lifecycle dispatch uses actual collector/local RPC.
// Application reads/writes remain direct to the private Eyre endpoints.
func TestProjectPublicMembersDisposableVM(t *testing.T) {
	exerciseProjectDisposableVM(t, true, true, true, true, true, true, true, false, false, false, false, false, true)
}

func vmQueueMemberAction(r *Server, b *vmMemberLaunchBackend, project, member string, input any, replay bool) (protocol.Operation, bool, error) {
	req := protocol.ProjectActionRequest{InstanceID: member, ConfirmAction: true}
	switch q := input.(type) {
	case projectMemberLaunchRequest:
		if b.memberCLI == nil {
			return r.queueProjectMemberLaunch("qualification", project, member, q)
		}
		req.Kind, req.RequestID, req.ExpectedRevision, req.ReviewDigest = "launch", q.RequestID, q.ExpectedRevision, q.ReviewDigest
	case projectMemberReadinessRequest:
		if b.memberCLI == nil {
			return r.queueProjectMemberReadiness("qualification", project, member, q)
		}
		req.Kind, req.RequestID, req.ExpectedRevision, req.ReviewDigest = "observe", q.RequestID, q.ExpectedRevision, q.ReviewDigest
	case projectMemberStopRequest:
		if b.memberCLI == nil {
			return r.queueProjectMemberStop("qualification", project, member, q)
		}
		req.Kind, req.RequestID, req.ExpectedRevision, req.ReviewDigest = "stop", q.RequestID, q.ExpectedRevision, q.ReviewDigest
		req.AfterOperationID = q.ReadinessOperationID
	case projectMemberStopExitRequest:
		if b.memberCLI == nil {
			return r.queueProjectMemberStopExit("qualification", project, member, q)
		}
		req.Kind, req.RequestID, req.ExpectedRevision, req.ReviewDigest = "observe-exit", q.RequestID, q.ExpectedRevision, q.ReviewDigest
		req.AfterOperationID = q.StopOperationID
	case projectMemberRetirementRequest:
		if b.memberCLI == nil {
			return r.queueProjectMemberRetirement("qualification", project, member, q)
		}
		req.Kind, req.RequestID, req.ExpectedRevision, req.ReviewDigest = "retire", q.RequestID, q.ExpectedRevision, q.ReviewDigest
		req.AfterOperationID = q.ExitOperationID
	case projectMemberResumeRequest:
		if b.memberCLI == nil {
			return r.queueProjectMemberResume("qualification", project, member, q)
		}
		req.Kind, req.RequestID, req.ExpectedRevision, req.ReviewDigest = "resume", q.RequestID, q.ExpectedRevision, q.ReviewDigest
		req.AfterOperationID = q.RetirementOperationID
	case projectMemberResumedReadinessRequest:
		if b.memberCLI == nil {
			return r.queueProjectMemberResumedReadiness("qualification", project, member, q)
		}
		req.Kind, req.RequestID, req.ExpectedRevision, req.ReviewDigest = "observe-resumed", q.RequestID, q.ExpectedRevision, q.ReviewDigest
		req.AfterOperationID = q.ResumeOperationID
	case projectMemberContextReadinessRequest:
		if b.memberCLI == nil {
			return r.queueProjectMemberContextReadiness("qualification", project, member, q)
		}
		req.Kind, req.RequestID, req.ExpectedRevision, req.ReviewDigest = "observe-current", q.RequestID, q.ExpectedRevision, q.ReviewDigest
		req.AfterOperationID = q.CreatorOperationID
		req.OriginalReadinessOperationID = q.OriginalReadinessOperationID
	default:
		return protocol.Operation{}, false, errors.New("unknown member fixture action")
	}
	if b.memberCLI.project != project {
		return protocol.Operation{}, false, errors.New("member CLI project changed")
	}
	op, err := b.memberCLI.operateRequest(req, replay)
	return op, replay, err
}

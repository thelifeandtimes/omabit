package privatechain

import (
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"time"
)

var ErrMemberBirth = errors.New("initial private member pier capture is unresolved; retain its launch and partial pier for review")

// Process observation excludes ship payload and HTTP readiness. The concrete
// adapter must inspect the complete selected container contract and actual
// kernel namespace, not infer them from labels, names or a previous observation.
type MemberBirthProcess struct {
	ObservedAt       time.Time                        `json:"observed_at"`
	StartedAt        time.Time                        `json:"started_at"`
	ContainerID      string                           `json:"container_id"`
	ContractDigest   string                           `json:"contract_digest"`
	Namespace        projectnetwork.NamespaceIdentity `json:"namespace"`
	CompleteContract bool                             `json:"complete_contract"`
	Running          bool                             `json:"running"`
}

type MemberBirthEvidence struct {
	InstanceID      string             `json:"instance_id"`
	OperationID     string             `json:"operation_id"`
	RuntimeID       string             `json:"runtime_id"`
	ContainerID     string             `json:"container_id"`
	ContractDigest  string             `json:"contract_digest"`
	EmptyObservedAt time.Time          `json:"empty_observed_at"`
	CapturedAt      time.Time          `json:"captured_at"`
	Pier            pierstore.Identity `json:"pier"`
	Before          MemberBirthProcess `json:"before"`
	After           MemberBirthProcess `json:"after"`
}

func validMemberBirthProcess(s MemberLaunch, o MemberBirthProcess, empty, now time.Time) bool {
	return o.ContainerID == s.ContainerID && o.ContractDigest == s.ContractDigest && o.Namespace == s.Gateway.Observation.Namespace && o.CompleteContract && o.Running && !o.StartedAt.IsZero() && !o.StartedAt.Before(empty) && !o.ObservedAt.Before(o.StartedAt) && memberFresh(o.ObservedAt, now)
}

// Historical evidence is evaluated at CapturedAt, not expired on later reads.
// Persistence must bind this to the original launch before readiness/resume may
// consume the identities. This is not a stopped-state or recovery authorization.
func (e MemberBirthEvidence) Validate(p Plan, w Workspace, s MemberLaunch) error {
	if s.validateCore(p, w) != nil || (s.Phase != "started" && s.Phase != "captured" && s.Phase != "running") || e.InstanceID != s.InstanceID || e.OperationID != s.OperationID || e.RuntimeID != s.RuntimeID || e.ContainerID != s.ContainerID || e.ContractDigest != s.ContractDigest || e.EmptyObservedAt.IsZero() || e.EmptyObservedAt.Before(s.Key.CheckedAt) || !validMemberBirthProcess(s, e.Before, e.EmptyObservedAt, e.CapturedAt) || !validMemberBirthProcess(s, e.After, e.EmptyObservedAt, e.CapturedAt) || e.After.ObservedAt.Before(e.Before.ObservedAt) || e.CapturedAt.Before(e.After.ObservedAt) {
		return ErrMemberBirth
	}
	before, after := e.Before, e.After
	before.ObservedAt = time.Time{}
	after.ObservedAt = time.Time{}
	if before != after {
		return ErrMemberBirth
	}
	if s.Running != nil && (s.Running.StartedAt != e.After.StartedAt || s.Running.Namespace != e.After.Namespace) {
		return ErrMemberBirth
	}
	reserved := []pierstore.DirectoryIdentity{w.Parent, w.Root, w.Observation.PiersRoot, s.Gateway.Observation.Chain, s.Gateway.Observation.Secrets, s.Artifact}
	reserved = append(reserved, w.Observation.Mounts...)
	for _, peer := range s.Peers {
		reserved = append(reserved, memberPierObjects(peer.Pier)...)
	}
	for _, id := range memberPierObjects(e.Pier) {
		if id.Device != w.Root.Device {
			return ErrMemberBirth
		}
	}
	if !memberClaimObjects(&reserved, memberPierObjects(e.Pier)...) {
		return ErrMemberBirth
	}
	return nil
}

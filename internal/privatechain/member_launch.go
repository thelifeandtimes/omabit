package privatechain

import (
	"errors"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

// Initial member execution, not ship readiness or resume authority. The common
// runner fence and durable ownership lookup are required before execution. Peer
// references must be resolved against retained launch/readiness records, never
// accepted from a caller or discovered by container name.
type MemberLaunch struct {
	Version        int                         `json:"version"`
	Gateway        Gateway                     `json:"gateway"`
	InstanceID     string                      `json:"instance_id"`
	RuntimeID      string                      `json:"runtime_id"`
	OperationID    string                      `json:"operation_id"`
	ContractDigest string                      `json:"contract_digest"`
	Pill           protocol.BootPill           `json:"pill"`
	Peers          []MemberPeer                `json:"peers"`
	Phase          string                      `json:"phase"`
	Artifact       pierstore.DirectoryIdentity `json:"artifact"`
	Key            *MemberKeyAcceptance        `json:"key,omitempty"`
	ContainerID    string                      `json:"container_id"`
	Running        *MemberRunningObservation   `json:"running,omitempty"`
	Capture        *MemberBirthEvidence        `json:"capture,omitempty"`
}

// Initial serial boot permits already-ready running peers only. A stopped or
// uncertain peer requires its own lifecycle/recovery contract, not relabelling.
type MemberPeer struct {
	InstanceID      string             `json:"instance_id"`
	RuntimeID       string             `json:"runtime_id"`
	OperationID     string             `json:"operation_id"`
	ContainerID     string             `json:"container_id"`
	ContractDigest  string             `json:"contract_digest"`
	ReadinessDigest string             `json:"readiness_digest"`
	Generation      uint64             `json:"generation"`
	StartedAt       time.Time          `json:"started_at"`
	Pier            pierstore.Identity `json:"pier"`
}

// The whole-file key digest is private recovery metadata. No key bytes, seed,
// command expression or credential belong in this record or public status.
// Verified denotes the independently checked key/chain proof and held host file;
// a saved receipt alone does not authorize a later effect.
type MemberKeyAcceptance struct {
	CheckedAt      time.Time          `json:"checked_at"`
	ContractDigest string             `json:"contract_digest"`
	KeySHA256      string             `json:"key_sha256"`
	Gateway        GatewayObservation `json:"gateway"`
	Verified       bool               `json:"verified"`
}

// Running is exact process/container evidence. HTTP readiness remains separate.
// Version3 commits first-boot capture separately; version2 retains it only in
// completed running history, and version1 history has none.
// Neither grants a second boot, Stop
// acknowledgement, deletion or reopening of any pier.
type MemberRunningObservation struct {
	ObservedAt       time.Time                        `json:"observed_at"`
	ContainerID      string                           `json:"container_id"`
	ContractDigest   string                           `json:"contract_digest"`
	StartedAt        time.Time                        `json:"started_at"`
	Namespace        projectnetwork.NamespaceIdentity `json:"namespace"`
	Gateway          GatewayObservation               `json:"gateway"`
	Peers            []MemberPeer                     `json:"peers"`
	CompleteContract bool                             `json:"complete_contract"`
	Running          bool                             `json:"running"`
	Birth            *MemberBirthEvidence             `json:"birth,omitempty"`
}

var ErrMemberLaunch = errors.New("private initial member launch differs from its retained contract")
var ErrMemberLaunchUnresolved = errors.New("private member launch outcome unresolved; retain the original intent, artifact, container and partial pier; do not repeat boot")
var memberRuntimeID = regexp.MustCompile(`^[0-9a-f]{32}$`)

func (s MemberLaunch) Clone() MemberLaunch {
	if s.Capture != nil {
		b := *s.Capture
		s.Capture = &b
	}
	s.Gateway = s.Gateway.Clone()
	s.Peers = slices.Clone(s.Peers)
	if s.Key != nil {
		v := *s.Key
		v.Gateway = v.Gateway.Clone()
		s.Key = &v
	}
	if s.Running != nil {
		v := *s.Running
		v.Gateway = v.Gateway.Clone()
		if v.Birth != nil {
			b := *v.Birth
			v.Birth = &b
		}
		v.Peers = slices.Clone(v.Peers)
		s.Running = &v
	}
	return s
}

func NewMemberLaunch(p Plan, w Workspace, g Gateway, instance, runtime, operation, contract string, peers []MemberPeer) (MemberLaunch, error) {
	s := MemberLaunch{Version: 2, Gateway: g, InstanceID: instance, RuntimeID: runtime, OperationID: operation, ContractDigest: contract, Pill: protocol.BootPill{SHA256: runtimecandidate.PillSHA256, Bytes: runtimecandidate.PillBytes}, Peers: peers, Phase: "planned"}.Clone()
	return s, s.Validate(p, w)
}

// Version3 saves first-boot capture before later running observation. Persistence
// uses schema46; original version1/2 histories keep their original evidence.
func NewCapturedMemberLaunch(p Plan, w Workspace, g Gateway, instance, runtime, operation, contract string, peers []MemberPeer) (MemberLaunch, error) {
	s, err := NewMemberLaunch(p, w, g, instance, runtime, operation, contract, peers)
	if err != nil {
		return MemberLaunch{}, err
	}
	s.Version = 3
	return s, s.Validate(p, w)
}

func memberRuntimeValid(id string) bool {
	return memberRuntimeID.MatchString(id) && id != strings.Repeat("0", 32)
}
func memberPierObjects(id pierstore.Identity) []pierstore.DirectoryIdentity {
	return []pierstore.DirectoryIdentity{{Device: id.Device, Inode: id.Inode, Birth: id.Birth}, {Device: id.UrbDevice, Inode: id.UrbInode, Birth: id.UrbBirth}}
}
func memberClaimObjects(objects *[]pierstore.DirectoryIdentity, more ...pierstore.DirectoryIdentity) bool {
	for _, id := range more {
		if !workspaceIdentity(id) {
			return false
		}
		for _, prior := range *objects {
			if id.SameObject(prior) {
				return false
			}
		}
		*objects = append(*objects, id)
	}
	return true
}

func memberGatewayMatches(p Plan, w Workspace, g Gateway, o GatewayObservation, at time.Time) bool {
	if g.checkObservation(p, w, o, at) != nil {
		return false
	}
	a, b := g.Observation.Clone(), o.Clone()
	a.ObservedAt = time.Time{}
	b.ObservedAt = time.Time{}
	return reflect.DeepEqual(a, b)
}
func memberFresh(observed, now time.Time) bool {
	return !observed.IsZero() && !now.IsZero() && now.Sub(observed) <= 30*time.Second && observed.Sub(now) <= 5*time.Second
}

func (s MemberLaunch) Validate(p Plan, w Workspace) error {
	if err := s.validateCore(p, w); err != nil {
		return err
	}
	if s.Capture != nil && s.Capture.Validate(p, w, s) != nil {
		return ErrMemberLaunch
	}
	if s.Version == 3 && s.Running != nil && !reflect.DeepEqual(s.Capture, s.Running.Birth) {
		return ErrMemberLaunch
	}
	if s.Running == nil {
		return nil
	}
	if s.Version == 1 {
		if s.Running.Birth != nil {
			return ErrMemberLaunch
		}
		return nil // Original44 history remains valid, without birth authority.
	}
	b := s.Running.Birth
	if b == nil || b.Validate(p, w, s) != nil || b.CapturedAt.After(s.Running.Gateway.ObservedAt) || b.CapturedAt.After(s.Running.ObservedAt) {
		return ErrMemberLaunch
	}
	return nil
}

// The birth validator uses this non-recursive structural check; it then checks
// its exact process/pier binding. All persisted reads call full Validate above.
func (s MemberLaunch) validateCore(p Plan, w Workspace) error {
	if (s.Version != 1 && s.Version != 2 && s.Version != 3) || s.Gateway.Validate(p, w) != nil || s.Gateway.Phase != "ready" || !memberRuntimeValid(s.RuntimeID) || !protocol.ValidID(s.OperationID) || s.OperationID == w.OperationID || s.OperationID == s.Gateway.Selection.OperationID || !hash(s.ContractDigest, false) || s.Pill != (protocol.BootPill{SHA256: runtimecandidate.PillSHA256, Bytes: runtimecandidate.PillBytes}) || len(s.Peers) >= len(p.Members) {
		return ErrMemberLaunch
	}
	planned := map[string]bool{}
	for _, m := range p.Members {
		planned[m.InstanceID] = true
	}
	if !planned[s.InstanceID] {
		return ErrMemberLaunch
	}
	ids := map[string]bool{s.Gateway.NetworkID: true, s.Gateway.ContainerID: true}
	instances := map[string]bool{s.InstanceID: true}
	runtimes := map[string]bool{s.RuntimeID: true}
	operations := map[string]bool{s.OperationID: true, w.OperationID: true, s.Gateway.Selection.OperationID: true}
	o := s.Gateway.Observation
	objects := []pierstore.DirectoryIdentity{w.Parent, w.Root, w.Observation.PiersRoot, o.Chain, o.Secrets}
	objects = append(objects, o.Mounts...)
	for n, m := range s.Peers {
		if !planned[m.InstanceID] || instances[m.InstanceID] || !memberRuntimeValid(m.RuntimeID) || runtimes[m.RuntimeID] || !protocol.ValidID(m.OperationID) || operations[m.OperationID] || !hash(m.ContainerID, false) || ids[m.ContainerID] || !hash(m.ContractDigest, false) || !hash(m.ReadinessDigest, false) || m.Generation == 0 || m.StartedAt.IsZero() || (n > 0 && s.Peers[n-1].InstanceID >= m.InstanceID) {
			return ErrMemberLaunch
		}
		for _, id := range memberPierObjects(m.Pier) {
			if id.Device != w.Root.Device {
				return ErrMemberLaunch
			}
		}
		if !memberClaimObjects(&objects, memberPierObjects(m.Pier)...) {
			return ErrMemberLaunch
		}
		ids[m.ContainerID], instances[m.InstanceID], runtimes[m.RuntimeID], operations[m.OperationID] = true, true, true, true
	}
	stages := map[string]int{"planned": 0, "artifact-dispatched": 1, "artifact-created": 2, "key-verified": 3, "create-dispatched": 4, "created": 5, "start-dispatched": 6, "started": 7, "running": 8}
	if s.Version == 3 {
		stages["captured"] = 8
		stages["running"] = 9
	}
	stage, ok := stages[s.Phase]
	if !ok {
		return ErrMemberLaunch
	}
	if s.Version != 3 || stage < 8 {
		if s.Capture != nil {
			return ErrMemberLaunch
		}
	} else if s.Capture == nil {
		return ErrMemberLaunch
	}
	if stage < 2 {
		if s.Artifact != (pierstore.DirectoryIdentity{}) {
			return ErrMemberLaunch
		}
	} else if !memberClaimObjects(&objects, s.Artifact) {
		return ErrMemberLaunch
	}
	if stage < 3 {
		if s.Key != nil {
			return ErrMemberLaunch
		}
	} else if s.Key == nil || !s.Key.Verified || s.Key.ContractDigest != s.ContractDigest || !hash(s.Key.KeySHA256, false) || s.Key.CheckedAt.Before(o.ObservedAt) || !memberGatewayMatches(p, w, s.Gateway, s.Key.Gateway, s.Key.CheckedAt) {
		return ErrMemberLaunch
	}
	if s.Key != nil {
		for _, peer := range s.Peers {
			if peer.StartedAt.After(s.Key.CheckedAt) {
				return ErrMemberLaunch
			}
		}
	}
	if stage < 5 {
		if s.ContainerID != "" {
			return ErrMemberLaunch
		}
	} else if !hash(s.ContainerID, false) || ids[s.ContainerID] {
		return ErrMemberLaunch
	}
	if s.Phase != "running" {
		if s.Running != nil {
			return ErrMemberLaunch
		}
	} else if s.Running == nil || s.checkRunning(p, w, *s.Running, s.Running.ObservedAt) != nil {
		return ErrMemberLaunch
	}
	return nil
}

func (s MemberLaunch) checkKey(p Plan, w Workspace, k MemberKeyAcceptance, now time.Time) error {
	if !k.Verified || k.ContractDigest != s.ContractDigest || !hash(k.KeySHA256, false) || !memberFresh(k.CheckedAt, now) || k.CheckedAt.Before(s.Gateway.Observation.ObservedAt) || !memberGatewayMatches(p, w, s.Gateway, k.Gateway, k.CheckedAt) {
		return ErrMemberLaunch
	}
	return nil
}
func (s MemberLaunch) checkRunning(p Plan, w Workspace, o MemberRunningObservation, now time.Time) error {
	if s.Key == nil || o.ContainerID != s.ContainerID || o.ContractDigest != s.ContractDigest || !o.CompleteContract || !o.Running || o.StartedAt.IsZero() || o.StartedAt.Before(s.Key.CheckedAt) || o.ObservedAt.Before(o.StartedAt) || !memberFresh(o.ObservedAt, now) || o.Namespace != s.Gateway.Observation.Namespace || !reflect.DeepEqual(o.Peers, s.Peers) || !memberGatewayMatches(p, w, s.Gateway, o.Gateway, o.ObservedAt) {
		return ErrMemberLaunch
	}
	return nil
}

type MemberLaunchResult struct {
	Artifact    pierstore.DirectoryIdentity
	Key         *MemberKeyAcceptance
	ContainerID string
	Running     *MemberRunningObservation
	Capture     *MemberBirthEvidence
}

func (s MemberLaunch) Advance(p Plan, w Workspace, expected, phase string, result MemberLaunchResult) (MemberLaunch, error) {
	next := map[string]string{"planned": "artifact-dispatched", "artifact-dispatched": "artifact-created", "artifact-created": "key-verified", "key-verified": "create-dispatched", "create-dispatched": "created", "created": "start-dispatched", "start-dispatched": "started", "started": "running"}
	if s.Version == 3 {
		next["started"] = "captured"
		next["captured"] = "running"
	}
	if s.Validate(p, w) != nil || s.Phase != expected || next[expected] != phase {
		return MemberLaunch{}, ErrMemberLaunch
	}
	remaining := result
	switch phase {
	case "artifact-created":
		s.Artifact = result.Artifact
		remaining.Artifact = pierstore.DirectoryIdentity{}
	case "key-verified":
		s.Key = result.Key
		remaining.Key = nil
	case "created":
		s.ContainerID = result.ContainerID
		remaining.ContainerID = ""
	case "captured":
		s.Capture = result.Capture
		remaining.Capture = nil
	case "running":
		s.Running = result.Running
		remaining.Running = nil
	}
	if !reflect.DeepEqual(remaining, MemberLaunchResult{}) {
		return MemberLaunch{}, ErrMemberLaunch
	}
	s.Phase = phase
	s = s.Clone()
	return s, s.Validate(p, w)
}

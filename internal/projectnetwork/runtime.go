package projectnetwork

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// ResourceIdentity combines the exact engine resource ID with a digest of its
// immutable creation contract. The Docker adapter must independently validate
// the complete contract (labels, options, image, mounts and containment), then
// derive this digest from inspection. Echoing the saved digest is not validation.
// These types are not yet persisted or exposed as an RPC capability.
type ResourceIdentity struct {
	ID             string `json:"id"`
	ContractDigest string `json:"contract_digest"`
}

// NamespaceIdentity is valid only on this host boot. An inode alone is not a
// persistent identity; an anchor can keep its container ID while changing netns.
type NamespaceIdentity struct {
	BootID string `json:"boot_id"`
	Device uint64 `json:"device"`
	Inode  uint64 `json:"inode"`
}

type RuntimeMember struct {
	InstanceID string           `json:"instance_id"`
	RuntimeID  string           `json:"runtime_id"`
	Container  ResourceIdentity `json:"container"`
}

// RuntimeBinding describes an already recorded materialized generation, not
// permission to adopt resources found by name/label. Initial creation, uncertainty
// journals and admitting a replacement generation require separate store actions.
type RuntimeBinding struct {
	ProjectID  string            `json:"project_id"`
	Revision   uint64            `json:"revision"`
	Generation uint64            `json:"generation"`
	PlanDigest string            `json:"plan_digest"`
	EngineID   string            `json:"engine_id"`
	Bridge     ResourceIdentity  `json:"bridge"`
	Anchor     ResourceIdentity  `json:"anchor"`
	Namespace  NamespaceIdentity `json:"namespace"`
	Members    []RuntimeMember   `json:"members"`
}

type ResourceObservation struct {
	Resource ResourceIdentity `json:"resource"`
	Present  bool             `json:"present"`
}

type Publication struct {
	InternalPort uint16 `json:"internal_port"`
	HostAddress  string `json:"host_address"`
	HostPort     uint16 `json:"host_port"`
	Protocol     string `json:"protocol"`
}

type AnchorObservation struct {
	ResourceObservation
	Running   bool              `json:"running"`
	Namespace NamespaceIdentity `json:"namespace"`
	BridgeID  string            `json:"bridge_id"`
	Requested []Publication     `json:"requested"`
	Active    []Publication     `json:"active"`
}

type MemberObservation struct {
	InstanceID string `json:"instance_id"`
	ResourceObservation
	Running   bool              `json:"running"`
	AnchorID  string            `json:"anchor_id"`
	Namespace NamespaceIdentity `json:"namespace"`
}

// Observation must be collected under the runner's project mutation fence. The
// adapter verifies complete inventories and brackets collection with unchanged
// engine/anchor identity. Missing results are not evidence that a resource stopped.
// AssessRuntime does not open an engine or authorize mutation from a cached result.
type Observation struct {
	ProjectID         string              `json:"project_id"`
	Revision          uint64              `json:"revision"`
	Generation        uint64              `json:"generation"`
	EngineID          string              `json:"engine_id"`
	ObservedAt        time.Time           `json:"observed_at"`
	Complete          bool                `json:"complete"`
	Bridge            ResourceObservation `json:"bridge"`
	BridgeAttachments []string            `json:"bridge_attachments"`
	Anchor            AnchorObservation   `json:"anchor"`
	Members           []MemberObservation `json:"members"`
}

type Assessment struct {
	State             string            `json:"state"`
	RecoveryRequired  bool              `json:"recovery_required"`
	Running           []string          `json:"running"`
	Stopped           []string          `json:"stopped"`
	RecordedNamespace NamespaceIdentity `json:"recorded_namespace"`
	ObservedNamespace NamespaceIdentity `json:"observed_namespace"`
}

var digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
var runtimePattern = regexp.MustCompile(`^[0-9a-f]{32}$`)
var bootPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var ErrRuntimeObservation = errors.New("project runtime observation cannot establish ownership and readiness")

func (r ResourceIdentity) valid() bool {
	return digestPattern.MatchString(r.ID) && digestPattern.MatchString(r.ContractDigest)
}
func (n NamespaceIdentity) valid() bool {
	return bootPattern.MatchString(n.BootID) && n.Device != 0 && n.Inode != 0
}
func validEngine(id string) bool {
	return id != "" && len(id) <= 256 && strings.IndexFunc(id, unicode.IsSpace) < 0 && strings.IndexFunc(id, unicode.IsControl) < 0
}

func (b RuntimeBinding) Validate(plan Plan) error {
	digest, err := plan.Digest()
	if err != nil || !protocol.ValidID(b.ProjectID) || b.Revision == 0 || b.Generation == 0 || digest != b.PlanDigest || !validEngine(b.EngineID) || !b.Bridge.valid() || !b.Anchor.valid() || !b.Namespace.valid() || len(b.Members) != len(plan.Members) {
		return fmt.Errorf("%w: invalid recorded binding", ErrRuntimeObservation)
	}
	ids := map[string]bool{b.Bridge.ID: true, b.Anchor.ID: true}
	if b.Bridge.ID == b.Anchor.ID {
		return fmt.Errorf("%w: duplicate resource", ErrRuntimeObservation)
	}
	runtimes := map[string]bool{}
	for i, m := range b.Members {
		if m.InstanceID != plan.Members[i].InstanceID || !runtimePattern.MatchString(m.RuntimeID) || runtimes[m.RuntimeID] || !m.Container.valid() || ids[m.Container.ID] {
			return fmt.Errorf("%w: invalid recorded membership", ErrRuntimeObservation)
		}
		ids[m.Container.ID], runtimes[m.RuntimeID] = true, true
	}
	return nil
}

func samePublications(plan Plan, actual []Publication) bool {
	if len(actual) != len(plan.Members) {
		return false
	}
	expected := make(map[Publication]bool, len(plan.Members))
	for _, m := range plan.Members {
		expected[Publication{m.InternalHTTPPort, plan.BindAddress, m.HTTPPort, "tcp"}] = true
	}
	for _, p := range actual {
		if !expected[p] {
			return false
		}
		delete(expected, p)
	}
	return len(expected) == 0
}

func exactResource(want ResourceIdentity, got ResourceObservation) bool {
	// Absence must still identify the exact queried ID. A missing contract has no
	// authority to declare cleanup complete; the assessment keeps recovery required.
	if !got.Present {
		return got.Resource.ID == want.ID && got.Resource.ContractDigest == ""
	}
	return got.Resource == want
}

func AssessRuntime(plan Plan, binding RuntimeBinding, observed Observation, now time.Time) (Assessment, error) {
	refuse := func(reason string) (Assessment, error) {
		return Assessment{}, fmt.Errorf("%w: %s", ErrRuntimeObservation, reason)
	}
	if err := binding.Validate(plan); err != nil {
		return Assessment{}, err
	}
	if now.IsZero() || !observed.Complete || observed.ObservedAt.IsZero() || now.Sub(observed.ObservedAt) > 30*time.Second || observed.ObservedAt.Sub(now) > 5*time.Second || observed.ProjectID != binding.ProjectID || observed.Revision != binding.Revision || observed.Generation != binding.Generation || observed.EngineID != binding.EngineID {
		return refuse("incomplete, stale or different generation")
	}
	if !exactResource(binding.Bridge, observed.Bridge) || !exactResource(binding.Anchor, observed.Anchor.ResourceObservation) {
		return refuse("resource identity or contract changed")
	}
	if len(observed.Members) != len(binding.Members) {
		return refuse("incomplete member inventory")
	}
	result := Assessment{State: "ready", RecordedNamespace: binding.Namespace, ObservedNamespace: observed.Anchor.Namespace}
	missingMember, otherNamespace := false, false
	expected := map[string]RuntimeMember{}
	for _, m := range binding.Members {
		expected[m.InstanceID] = m
	}
	for _, m := range observed.Members {
		want, ok := expected[m.InstanceID]
		if !ok || !exactResource(want.Container, m.ResourceObservation) {
			return refuse("unexpected member or container")
		}
		delete(expected, m.InstanceID)
		if !m.Present {
			if m.Running || m.Namespace != (NamespaceIdentity{}) || m.AnchorID != "" {
				return refuse("inconsistent missing member")
			}
			missingMember = true
			continue
		}
		if m.AnchorID != binding.Anchor.ID {
			return refuse("member attached to another anchor")
		}
		if m.Running {
			if !m.Namespace.valid() {
				return refuse("running member lacks namespace identity")
			}
			result.Running = append(result.Running, m.InstanceID)
			otherNamespace = otherNamespace || m.Namespace != binding.Namespace
		} else {
			if m.Namespace != (NamespaceIdentity{}) {
				return refuse("stopped member has live namespace")
			}
			result.Stopped = append(result.Stopped, m.InstanceID)
		}
	}
	if len(expected) != 0 {
		return refuse("missing member inventory")
	}
	slices.Sort(result.Running)
	slices.Sort(result.Stopped)
	// Only the anchor is a direct bridge endpoint. Members use container:<exact ID>.
	if len(observed.BridgeAttachments) > 1 || len(observed.BridgeAttachments) == 1 && observed.BridgeAttachments[0] != binding.Anchor.ID {
		return refuse("unknown bridge attachment")
	}
	anchor := observed.Anchor
	if !observed.Bridge.Present {
		if len(observed.BridgeAttachments) != 0 || anchor.Running {
			return refuse("inconsistent missing bridge")
		}
		result.State = "bridge-unavailable"
	} else if !anchor.Present || !anchor.Running {
		result.State = "anchor-unavailable"
	} else {
		if anchor.BridgeID != binding.Bridge.ID || len(observed.BridgeAttachments) != 1 || !anchor.Namespace.valid() || !samePublications(plan, anchor.Requested) {
			return refuse("anchor topology or requested publications changed")
		}
		if anchor.Namespace != binding.Namespace {
			result.State = "namespace-changed"
		} else if otherNamespace {
			result.State = "member-namespace-changed"
		} else if !samePublications(plan, anchor.Active) {
			result.State = "publication-unavailable"
		} else if missingMember {
			result.State = "member-unavailable"
		} else if len(result.Stopped) > 0 {
			result.State = "member-stopped"
		}
	}
	if !anchor.Present {
		if anchor.Running || anchor.Namespace != (NamespaceIdentity{}) || anchor.BridgeID != "" || len(anchor.Requested) != 0 || len(anchor.Active) != 0 || len(observed.BridgeAttachments) != 0 {
			return refuse("inconsistent missing anchor")
		}
	} else if !anchor.Running {
		if anchor.Namespace != (NamespaceIdentity{}) || len(anchor.Active) != 0 || len(observed.BridgeAttachments) != 0 || anchor.BridgeID != binding.Bridge.ID || !samePublications(plan, anchor.Requested) {
			return refuse("inconsistent stopped anchor")
		}
	}
	result.RecoveryRequired = result.State != "ready" && result.State != "member-stopped"
	return result, nil
}

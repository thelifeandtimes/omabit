package privatechain

import (
	"errors"
	"regexp"
	"slices"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

// Gateway is the first-initialization journal, never a restart recipe. Its
// selection comes from retained runner records and a separately reviewed input
// generation. Schema43 binds it to one actor/request and commits checkpoints
// atomically. Internal scheduling holds the shared runner mutation/update fence;
// startup only interrupts unfinished intent. No public API accepts this descriptor.
type Gateway struct {
	Selection   GatewaySelection    `json:"selection"`
	Phase       string              `json:"phase"`
	NetworkID   string              `json:"network_id"`
	ContainerID string              `json:"container_id"`
	Observation *GatewayObservation `json:"observation,omitempty"`
}

type GatewaySelection struct {
	Version              int                    `json:"version"`
	Binding              Binding                `json:"binding"`
	Revision             uint64                 `json:"revision"`
	OperationID          string                 `json:"operation_id"`
	PlanDigest           string                 `json:"plan_digest"`
	WorkspaceOperationID string                 `json:"workspace_operation_id"`
	WorkspaceDigest      string                 `json:"workspace_digest"`
	Inputs               RuntimeInputInspection `json:"inputs"`
	EngineID             string                 `json:"engine_id"`
	Image                string                 `json:"image"`
}

// All observations are public metadata. No key seed, keyfile, cookie, RPC token,
// database contents or process log may be included. CompleteContract means the
// adapter independently checked labels, image, command, mounts, resource limits,
// network ownership/options/attachments and exact publication, not echoed a
// saved selection. Observe must also freshly verify genesis, all contract code,
// head hash and the public network keys against the selected owned chain.
type GatewayObservation struct {
	Selection        GatewaySelection                 `json:"selection"`
	ObservedAt       time.Time                        `json:"observed_at"`
	NetworkID        string                           `json:"network_id"`
	ContainerID      string                           `json:"container_id"`
	ImageID          string                           `json:"image_id"`
	Namespace        projectnetwork.NamespaceIdentity `json:"namespace"`
	HostNamespace    projectnetwork.NamespaceIdentity `json:"host_namespace"`
	Root             pierstore.DirectoryIdentity      `json:"root"`
	Chain            pierstore.DirectoryIdentity      `json:"chain"`
	Secrets          pierstore.DirectoryIdentity      `json:"secrets"`
	Mounts           []pierstore.DirectoryIdentity    `json:"mounts"`
	Foundation       Foundation                       `json:"foundation"`
	Head             uint64                           `json:"head"`
	HeadHash         string                           `json:"head_hash"`
	PublicKeys       []GatewayPublicKey               `json:"public_keys"`
	Publications     []projectnetwork.Publication     `json:"publications"`
	CompleteContract bool                             `json:"complete_contract"`
	Running          bool                             `json:"running"`
	Private          bool                             `json:"private"`
	Synced           bool                             `json:"synced"`
	RPCReadOnly      bool                             `json:"rpc_read_only"`
}

type GatewayPublicKey struct {
	Ship  string `json:"ship"`
	Auth  string `json:"auth"`
	Crypt string `json:"crypt"`
	Life  uint64 `json:"life"`
	Suite uint64 `json:"suite"`
}

func (o GatewayObservation) Clone() GatewayObservation {
	o.Mounts = slices.Clone(o.Mounts)
	o.Foundation.Contracts = slices.Clone(o.Foundation.Contracts)
	o.PublicKeys = slices.Clone(o.PublicKeys)
	o.Publications = slices.Clone(o.Publications)
	return o
}

func (g Gateway) Clone() Gateway {
	if g.Observation != nil {
		o := g.Observation.Clone()
		g.Observation = &o
	}
	return g
}

func NewGateway(plan Plan, w Workspace, operation, engine string, inputs RuntimeInputInspection) (Gateway, error) {
	g := Gateway{Selection: GatewaySelection{
		Version: 1, Binding: w.Binding, Revision: w.Revision, OperationID: operation,
		PlanDigest: w.PlanDigest, WorkspaceOperationID: w.OperationID,
		WorkspaceDigest: digest("omabit-private-workspace-v1", w), Inputs: inputs,
		EngineID: engine, Image: runtimecandidate.Image,
	}, Phase: "planned"}
	if err := g.Validate(plan, w); err != nil {
		return Gateway{}, err
	}
	return g, nil
}

var ErrGatewayJournal = errors.New("private gateway differs from its retained first-initialization contract")
var ErrGatewayObservation = errors.New("private gateway observation is incomplete, stale or changed")
var gatewayBootID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func (g Gateway) Validate(plan Plan, w Workspace) error {
	s := g.Selection
	i := s.Inputs
	if w.Validate(plan) != nil || w.Phase != "observed" || s.Version != 1 || s.Binding != w.Binding ||
		s.Revision != w.Revision || !protocol.ValidID(s.OperationID) || s.OperationID == w.OperationID ||
		s.PlanDigest != w.PlanDigest || s.WorkspaceOperationID != w.OperationID ||
		s.WorkspaceDigest != digest("omabit-private-workspace-v1", w) || !hostcompat.ValidEngineIdentity(s.EngineID) ||
		s.Image != runtimecandidate.Image || i.Format != RuntimeInputFormat || !hash(i.ManifestSHA256, false) ||
		i.FoundationDigest != w.Binding.FoundationDigest || i.Files <= 0 || i.Files > MaxRuntimeInputFiles ||
		i.Bytes <= 0 || i.Bytes > MaxRuntimeInputBytes || i.RuntimeAuthorized {
		return ErrGatewayJournal
	}
	switch g.Phase {
	case "planned", "network-dispatched":
		if g.NetworkID != "" || g.ContainerID != "" || g.Observation != nil {
			return ErrGatewayJournal
		}
	case "network-created", "gateway-dispatched":
		if !hash(g.NetworkID, false) || g.ContainerID != "" || g.Observation != nil {
			return ErrGatewayJournal
		}
	case "gateway-created", "start-dispatched", "started", "ready":
		if !hash(g.NetworkID, false) || !hash(g.ContainerID, false) || g.NetworkID == g.ContainerID {
			return ErrGatewayJournal
		}
		if g.Phase == "ready" {
			if g.Observation == nil || g.checkObservation(plan, w, *g.Observation, g.Observation.ObservedAt) != nil {
				return ErrGatewayJournal
			}
		} else if g.Observation != nil {
			return ErrGatewayJournal
		}
	default:
		return ErrGatewayJournal
	}
	return nil
}

func (g Gateway) checkObservation(plan Plan, w Workspace, o GatewayObservation, now time.Time) error {
	if now.IsZero() || o.ObservedAt.IsZero() || now.Sub(o.ObservedAt) > 30*time.Second || o.ObservedAt.Sub(now) > 5*time.Second ||
		o.Selection != g.Selection || o.NetworkID != g.NetworkID || o.ContainerID != g.ContainerID ||
		(o.ImageID != runtimecandidate.ConfigDigest && o.ImageID != runtimecandidate.ManifestDigest) ||
		!o.CompleteContract || !o.Running || !o.Private || !o.Synced || !o.RPCReadOnly ||
		!gatewayBootID.MatchString(o.Namespace.BootID) || o.Namespace.BootID != o.HostNamespace.BootID ||
		o.Namespace.Device == 0 || o.Namespace.Inode == 0 || o.HostNamespace.Device == 0 || o.HostNamespace.Inode == 0 ||
		(o.Namespace.Device == o.HostNamespace.Device && o.Namespace.Inode == o.HostNamespace.Inode) ||
		!w.Root.SameObject(o.Root) || len(o.Mounts) != len(w.Observation.Mounts) || o.Head != 0x50 || !hash(o.HeadHash, true) ||
		len(o.PublicKeys) != len(plan.Members) || len(o.Publications) != len(plan.Members) {
		return ErrGatewayObservation
	}
	foundation, err := o.Foundation.Digest()
	if err != nil || foundation != g.Selection.Binding.FoundationDigest {
		return ErrGatewayObservation
	}
	objects := []pierstore.DirectoryIdentity{w.Parent, o.Root, w.Observation.PiersRoot, o.Chain, o.Secrets}
	objects = append(objects, o.Mounts...)
	for n, object := range objects {
		if !workspaceIdentity(object) || object.Device != w.Root.Device {
			return ErrGatewayObservation
		}
		for _, prior := range objects[:n] {
			if object.SameObject(prior) {
				return ErrGatewayObservation
			}
		}
	}
	keys := make(map[string]bool)
	for n, member := range plan.Members {
		key := o.PublicKeys[n]
		if !w.Observation.Mounts[n].SameObject(o.Mounts[n]) || key.Ship != member.Ship || key.Life != 1 || key.Suite != 1 ||
			o.Publications[n] != (projectnetwork.Publication{InternalPort: member.InternalHTTPPort, HostAddress: plan.BindAddress, HostPort: member.HTTPPort, Protocol: "tcp"}) {
			return ErrGatewayObservation
		}
		for _, public := range []string{key.Auth, key.Crypt} {
			if !hash(public, true) || keys[public] {
				return ErrGatewayObservation
			}
			keys[public] = true
		}
	}
	return nil
}

func (g Gateway) advance(plan Plan, w Workspace, phase, network, container string, observation *GatewayObservation) (Gateway, error) {
	if g.Validate(plan, w) != nil {
		return Gateway{}, ErrGatewayJournal
	}
	next := map[string]string{"planned": "network-dispatched", "network-dispatched": "network-created", "network-created": "gateway-dispatched", "gateway-dispatched": "gateway-created", "gateway-created": "start-dispatched", "start-dispatched": "started", "started": "ready"}
	if next[g.Phase] != phase || (g.Phase != "network-dispatched" && network != g.NetworkID) || (g.Phase != "gateway-dispatched" && container != g.ContainerID) {
		return Gateway{}, ErrGatewayJournal
	}
	g.Phase, g.NetworkID, g.ContainerID = phase, network, container
	g.Observation = observation
	g = g.Clone()
	return g, g.Validate(plan, w)
}

// Advance validates a store checkpoint against its exact prior phase. It cannot
// change selection, substitute resources, skip effects or repeat initialization.
func (g Gateway) Advance(plan Plan, w Workspace, expected, phase, network, container string, observation *GatewayObservation) (Gateway, error) {
	if g.Phase != expected {
		return Gateway{}, ErrGatewayJournal
	}
	return g.advance(plan, w, phase, network, container, observation)
}

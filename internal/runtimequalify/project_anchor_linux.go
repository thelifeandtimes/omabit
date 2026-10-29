//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

const projectInventoryFormat = `{"id":{{json .Id}},"name":{{json .Name}},"network_mode":{{json .HostConfig.NetworkMode}}}`
const projectInventoryLimit = 1024

var projectBootIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var projectNamespacePattern = regexp.MustCompile(`^[1-9][0-9]*:[1-9][0-9]*$`)

type projectAnchorEngine struct {
	base       *projectDockerEngine
	prepared   projectnetwork.Preparation
	readBootID func() (string, error)
	readyAt    time.Time
	dispatched bool
}

// OpenProjectAnchorStart only reopens metadata for the exact completed creation
// journal. It never creates/replaces a workspace, resource or pier. The caller
// supplies an accepted Start receipt and holds the runner resource/update fence.
func OpenProjectAnchorStart(ctx context.Context, s ProjectPreparationSelection, prepared projectnetwork.Preparation) (projectnetwork.AnchorStartEngine, error) {
	return openProjectAnchorStart(ctx, s, prepared, defaults())
}
func openProjectAnchorStart(ctx context.Context, s projectSelection, prepared projectnetwork.Preparation, d dependencies) (projectnetwork.AnchorStartEngine, error) {
	if prepared.Validate(s.Plan) != nil || prepared.Sequence != 6 || s.Generation != 1 {
		return nil, errors.New("anchor Start requires completed initial preparation")
	}
	// Validate paths and identities before access; image identity is read below.
	if _, err := newProjectResourceContract(s, prepared.EngineID, candidate.ManifestDigest); err != nil {
		return nil, err
	}
	q := &qualifier{d: d, phase: "project-anchor-start"}
	q.base = filepath.Join(d.home, ".local/share/urbit-runner/managed")
	if err := q.verifyEngine(ctx); err != nil {
		return nil, err
	}
	if err := hostcompat.CheckEngineIdentity(q.engineEvidence, prepared.EngineID); err != nil {
		return nil, err
	}
	q.r.Directory = filepath.Join(q.base, "project-resources-"+s.ProjectID)
	q.config = filepath.Join(q.r.Directory, "docker-config")
	for _, p := range append(q.storagePaths(), q.r.Directory, q.config) {
		if err := q.owned(p, os.ModeDir, p == q.base || p == q.r.Directory || p == q.config); err != nil {
			return nil, err
		}
	}
	if err := q.inspectImage(ctx); err != nil {
		return nil, err
	}
	contract, err := newProjectResourceContract(s, prepared.EngineID, q.r.ImageID)
	if err != nil {
		return nil, err
	}
	expected := contract.Preparation
	expected.Sequence = 6
	expected.Bridge.Phase, expected.Bridge.ID = "observed", prepared.Bridge.ID
	expected.Anchor.Phase, expected.Anchor.ID = "observed", prepared.Anchor.ID
	if !reflect.DeepEqual(prepared, expected) {
		return nil, errors.New("anchor Start differs from its fixed creation contract")
	}
	base := &projectDockerEngine{q: q, contract: contract}
	if err = base.VerifyEngine(ctx, prepared.EngineID); err != nil {
		return nil, err
	}
	return &projectAnchorEngine{base: base, prepared: prepared, readBootID: func() (string, error) {
		data, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
		return strings.TrimSpace(string(data)), err
	}}, nil
}

func (e *projectAnchorEngine) VerifyEngine(ctx context.Context, id string) error {
	return e.base.VerifyEngine(ctx, id)
}
func (e *projectAnchorEngine) selected(p projectnetwork.Preparation) error {
	if !reflect.DeepEqual(p, e.prepared) {
		return errors.New("anchor Start selection changed")
	}
	return nil
}
func (e *projectAnchorEngine) listIDs(ctx context.Context) ([]string, error) {
	q := e.base.q
	out, err := q.docker(ctx, q.d.times.command, "container", "ls", "--all", "--quiet", "--no-trunc")
	if err != nil {
		return nil, err
	}
	ids := strings.Fields(out)
	if len(ids) > projectInventoryLimit {
		return nil, errors.New("project inventory exceeds bounded observation limit")
	}
	slices.Sort(ids)
	for i, id := range ids {
		if !containerIDPattern.MatchString(id) || i > 0 && ids[i-1] == id {
			return nil, errUntrustedContainer
		}
	}
	return ids, nil
}
func (e *projectAnchorEngine) inventory(ctx context.Context) ([]projectnetwork.NetworkContainer, error) {
	ids, err := e.listIDs(ctx)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, errUntrustedContainer
	}
	args := append([]string{"container", "inspect", "--format", projectInventoryFormat}, ids...)
	q := e.base.q
	out, err := q.docker(ctx, q.d.times.command, args...)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(strings.NewReader(out))
	decoder.DisallowUnknownFields()
	values := make([]projectnetwork.NetworkContainer, 0, len(ids))
	for range ids {
		var v projectnetwork.NetworkContainer
		if decoder.Decode(&v) != nil {
			return nil, errUntrustedContainer
		}
		values = append(values, v)
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return nil, errUntrustedContainer
	}
	slices.SortFunc(values, func(a, b projectnetwork.NetworkContainer) int { return strings.Compare(a.ID, b.ID) })
	for i, id := range ids {
		if values[i].ID != id {
			return nil, errUntrustedContainer
		}
	}
	after, err := e.listIDs(ctx)
	if err != nil {
		return nil, err
	}
	if !slices.Equal(ids, after) {
		return nil, errors.New("container inventory changed during observation")
	}
	return values, nil
}
func (e *projectAnchorEngine) anchor(ctx context.Context) (container, error) {
	q := e.base.q
	out, err := q.docker(ctx, q.d.times.command, "container", "inspect", e.prepared.Anchor.ID)
	if err != nil {
		return container{}, err
	}
	var values []container
	if json.Unmarshal([]byte(out), &values) != nil || len(values) != 1 || values[0].State == nil || values[0].State.Running == nil {
		return container{}, errUntrustedContainer
	}
	v := values[0]
	if err = e.base.contract.validateAnchor(v, e.prepared.Anchor.ID, e.prepared.Bridge.ID, *v.State.Running); err != nil {
		return container{}, err
	}
	return v, nil
}
func (e *projectAnchorEngine) bridge(ctx context.Context) (bridge, error) {
	q := e.base.q
	out, err := q.docker(ctx, q.d.times.command, "network", "inspect", e.prepared.Bridge.ID)
	if err != nil {
		return bridge{}, err
	}
	var values []bridge
	if json.Unmarshal([]byte(out), &values) != nil || len(values) != 1 {
		return bridge{}, errUntrustedNetwork
	}
	n := values[0]
	return n, e.base.contract.validateBridge(n, e.prepared.Bridge.ID, e.prepared.Anchor.ID)
}
func (e *projectAnchorEngine) namespace(ctx context.Context) (projectnetwork.NamespaceIdentity, error) {
	return e.namespaceFor(ctx, e.prepared.Anchor.ID)
}
func (e *projectAnchorEngine) namespaceFor(ctx context.Context, id string) (projectnetwork.NamespaceIdentity, error) {
	if !containerIDPattern.MatchString(id) {
		return projectnetwork.NamespaceIdentity{}, errUntrustedContainer
	}
	if e.readBootID == nil {
		return projectnetwork.NamespaceIdentity{}, errors.New("host boot identity reader unavailable")
	}
	before, err := e.readBootID()
	if err != nil || !projectBootIDPattern.MatchString(before) {
		return projectnetwork.NamespaceIdentity{}, errors.New("host boot identity unavailable")
	}
	q := e.base.q
	out, err := q.docker(ctx, q.d.times.command, "container", "exec", id, "/bin/stat", "-L", "-c", "%d:%i", "/proc/self/ns/net")
	if err != nil {
		return projectnetwork.NamespaceIdentity{}, err
	}
	text := strings.TrimSpace(out)
	if !projectNamespacePattern.MatchString(text) {
		return projectnetwork.NamespaceIdentity{}, errors.New("container namespace identity unavailable")
	}
	parts := strings.Split(text, ":")
	device, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return projectnetwork.NamespaceIdentity{}, errUntrustedContainer
	}
	inode, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return projectnetwork.NamespaceIdentity{}, errUntrustedContainer
	}
	after, err := e.readBootID()
	if err != nil || after != before {
		return projectnetwork.NamespaceIdentity{}, errors.New("host boot changed during namespace observation")
	}
	return projectnetwork.NamespaceIdentity{BootID: before, Device: device, Inode: inode}, nil
}
func (e *projectAnchorEngine) InspectAnchor(ctx context.Context, p projectnetwork.Preparation) (projectnetwork.Observation, error) {
	e.readyAt = time.Time{}
	if err := e.selected(p); err != nil {
		return projectnetwork.Observation{}, err
	}
	if err := e.VerifyEngine(ctx, p.EngineID); err != nil {
		return projectnetwork.Observation{}, err
	}
	observedAt := time.Now().UTC()
	inventory, err := e.inventory(ctx)
	if err != nil {
		return projectnetwork.Observation{}, err
	}
	members, err := projectnetwork.NamespaceMembers(p.Anchor.ID, inventory)
	if err != nil || len(members) != 0 {
		return projectnetwork.Observation{}, errors.New("initial anchor Start requires a complete inventory with no namespace members")
	}
	n, err := e.bridge(ctx)
	if err != nil {
		return projectnetwork.Observation{}, err
	}
	v, err := e.anchor(ctx)
	if err != nil {
		return projectnetwork.Observation{}, err
	}
	running := *v.State.Running
	if running && len(n.Containers) != 1 {
		return projectnetwork.Observation{}, errUntrustedNetwork
	}
	namespace := projectnetwork.NamespaceIdentity{}
	if running {
		namespace, err = e.namespace(ctx)
		if err != nil {
			return projectnetwork.Observation{}, err
		}
	}
	afterInventory, err := e.inventory(ctx)
	if err != nil {
		return projectnetwork.Observation{}, err
	}
	afterBridge, err := e.bridge(ctx)
	if err != nil {
		return projectnetwork.Observation{}, err
	}
	afterAnchor, err := e.anchor(ctx)
	if err != nil {
		return projectnetwork.Observation{}, err
	}
	if !reflect.DeepEqual(inventory, afterInventory) || !reflect.DeepEqual(n, afterBridge) || !reflect.DeepEqual(v, afterAnchor) {
		return projectnetwork.Observation{}, errors.New("anchor or inventory changed during observation")
	}
	if running {
		again, err := e.namespace(ctx)
		if err != nil {
			return projectnetwork.Observation{}, err
		}
		if namespace != again {
			return projectnetwork.Observation{}, errors.New("anchor namespace changed during observation")
		}
	}
	if err = e.VerifyEngine(ctx, p.EngineID); err != nil {
		return projectnetwork.Observation{}, err
	}
	if time.Since(observedAt) > 30*time.Second {
		return projectnetwork.Observation{}, errors.New("anchor observation expired")
	}
	o := projectnetwork.Observation{ProjectID: p.ProjectID, Revision: p.ProjectRevision, Generation: p.Generation, EngineID: p.EngineID, ObservedAt: observedAt, Complete: true, Bridge: projectnetwork.ResourceObservation{Resource: projectnetwork.ResourceIdentity{ID: p.Bridge.ID, ContractDigest: e.base.contract.digest("bridge")}, Present: true}, Anchor: projectnetwork.AnchorObservation{ResourceObservation: projectnetwork.ResourceObservation{Resource: projectnetwork.ResourceIdentity{ID: p.Anchor.ID, ContractDigest: e.base.contract.digest("anchor")}, Present: true}, Running: running, BridgeID: p.Bridge.ID, Namespace: namespace}}
	for id := range n.Containers {
		o.BridgeAttachments = append(o.BridgeAttachments, id)
	}
	slices.Sort(o.BridgeAttachments)
	for _, m := range e.base.contract.Selection.Plan.Members {
		o.Anchor.Requested = append(o.Anchor.Requested, projectnetwork.Publication{InternalPort: m.InternalHTTPPort, HostAddress: e.base.contract.Selection.Plan.BindAddress, HostPort: m.HTTPPort, Protocol: "tcp"})
	}
	if running {
		o.Anchor.Active = slices.Clone(o.Anchor.Requested)
	} else {
		e.readyAt = observedAt
	}
	return o, nil
}
func (e *projectAnchorEngine) StartAnchor(ctx context.Context, p projectnetwork.Preparation) error {
	if err := e.selected(p); err != nil {
		return err
	}
	if e.dispatched || e.readyAt.IsZero() || time.Since(e.readyAt) > 30*time.Second {
		return errors.New("anchor Start needs a fresh complete inspection and cannot replay")
	}
	if err := e.VerifyEngine(ctx, p.EngineID); err != nil {
		return err
	}
	e.dispatched = true
	e.readyAt = time.Time{}
	q := e.base.q
	out, err := q.docker(ctx, q.d.times.command, "container", "start", p.Anchor.ID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) != p.Anchor.ID {
		return errors.New("anchor Start acknowledgement did not identify the selected container")
	}
	return nil
}

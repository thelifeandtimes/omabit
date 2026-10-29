package privatechain

import (
	"context"
	"errors"
	"regexp"
	"slices"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// Workspace records a path relative to the private runner data root, separate
// from the general pier catalogue. A future pier is one ship-named child of its
// exclusive member mount parent. Workspace creates only an owned project
// directory and empty member mount parents. It neither copies a chain nor generates keys, creates piers or
// contacts an engine. Store integration must retain the original reservation,
// hold the common runner mutation/update fence and commit before each effect.
type Workspace struct {
	Binding     Binding                     `json:"binding"`
	Revision    uint64                      `json:"revision"`
	OperationID string                      `json:"operation_id"`
	PlanDigest  string                      `json:"plan_digest"`
	Path        string                      `json:"path"`
	Parent      pierstore.DirectoryIdentity `json:"parent"`
	Mounts      []WorkspaceMount            `json:"mounts"`
	Phase       string                      `json:"phase"`
	Root        pierstore.DirectoryIdentity `json:"root"`
	Observation *WorkspaceObservation       `json:"observation,omitempty"`
}

type WorkspaceMount struct {
	InstanceID string `json:"instance_id"`
	Ship       string `json:"ship"`
	Path       string `json:"path"`
}

// A driver must observe the complete expected directory set under the held root
// and verify owner-only directories, absence of links/mounts and empty leaf
// parents. Complete is not inferred from a pathname or a prior receipt.
type WorkspaceObservation struct {
	ObservedAt time.Time                     `json:"observed_at"`
	Parent     pierstore.DirectoryIdentity   `json:"parent"`
	Root       pierstore.DirectoryIdentity   `json:"root"`
	PiersRoot  pierstore.DirectoryIdentity   `json:"piers_root"`
	Mounts     []pierstore.DirectoryIdentity `json:"mounts"`
	Complete   bool                          `json:"complete"`
	Private    bool                          `json:"private"`
	Empty      bool                          `json:"empty"`
	Synced     bool                          `json:"synced"`
}

func (w Workspace) Clone() Workspace {
	w.Mounts = slices.Clone(w.Mounts)
	if w.Observation != nil {
		o := *w.Observation
		o.Mounts = slices.Clone(o.Mounts)
		w.Observation = &o
	}
	return w
}

var workspaceBirth = regexp.MustCompile(`^[0-9]{1,20}:(0|[1-9][0-9]{0,8})$`)

func workspaceIdentity(d pierstore.DirectoryIdentity) bool {
	return d.Device != 0 && d.Inode != 0 && workspaceBirth.MatchString(d.Birth)
}

func NewWorkspace(plan Plan, binding Binding, revision uint64, operation string, parent pierstore.DirectoryIdentity) (Workspace, error) {
	digest, err := plan.Digest()
	if err != nil {
		return Workspace{}, err
	}
	w := Workspace{Binding: binding, Revision: revision, OperationID: operation, Parent: parent, PlanDigest: digest, Path: "private-projects/" + binding.ProjectID, Phase: "planned"}
	for _, m := range plan.Members {
		w.Mounts = append(w.Mounts, WorkspaceMount{m.InstanceID, m.Ship, w.Path + "/piers/" + m.InstanceID})
	}
	return w, w.Validate(plan)
}

func (w Workspace) Validate(plan Plan) error {
	fail := func() error { return errors.New("private project workspace differs from its owned reservation") }
	digest, err := plan.Digest()
	if err != nil || digest != w.PlanDigest || !workspaceIdentity(w.Parent) || w.Binding.Validate() != nil || w.Binding.FoundationDigest != plan.FoundationDigest || w.Binding.Generation != 1 || w.Revision != 1 || !strings.HasPrefix(w.Binding.ProjectID, "private-project-") || !protocol.ValidID(w.OperationID) || w.Path != "private-projects/"+w.Binding.ProjectID || !protocol.ValidPierName(w.Binding.ProjectID) || len(w.Mounts) != len(plan.Members) {
		return fail()
	}
	for n, m := range plan.Members {
		if !protocol.ValidPierName(m.InstanceID) || w.Mounts[n] != (WorkspaceMount{m.InstanceID, m.Ship, w.Path + "/piers/" + m.InstanceID}) {
			return fail()
		}
	}
	switch w.Phase {
	case "planned", "root-dispatched":
		if w.Root != (pierstore.DirectoryIdentity{}) || w.Observation != nil {
			return fail()
		}
	case "root-created", "layout-dispatched", "layout-created":
		if !workspaceIdentity(w.Root) || w.Root.SameObject(w.Parent) || w.Observation != nil {
			return fail()
		}
	case "observed":
		if !workspaceIdentity(w.Root) || w.Root.SameObject(w.Parent) || w.Observation == nil || w.checkObservation(*w.Observation, w.Observation.ObservedAt) != nil {
			return fail()
		}
	default:
		return fail()
	}
	return nil
}

func (w Workspace) checkObservation(o WorkspaceObservation, now time.Time) error {
	fail := func() error { return errors.New("private workspace observation is incomplete, stale or changed") }
	if now.IsZero() || o.ObservedAt.IsZero() || now.Sub(o.ObservedAt) > 30*time.Second || o.ObservedAt.Sub(now) > 5*time.Second || !o.Complete || !o.Private || !o.Empty || !o.Synced || !workspaceIdentity(o.Parent) || !w.Parent.SameObject(o.Parent) || !workspaceIdentity(o.Root) || !w.Root.SameObject(o.Root) || !workspaceIdentity(o.PiersRoot) || len(o.Mounts) != len(w.Mounts) {
		return fail()
	}
	all := []pierstore.DirectoryIdentity{o.Parent, o.Root, o.PiersRoot}
	all = append(all, o.Mounts...)
	for n, d := range all {
		if !workspaceIdentity(d) || d.Device != o.Root.Device {
			return fail()
		}
		for _, prior := range all[:n] {
			if d.SameObject(prior) {
				return fail()
			}
		}
	}
	return nil
}

func (w Workspace) Advance(plan Plan, expected, phase string, root pierstore.DirectoryIdentity, observed *WorkspaceObservation) (Workspace, error) {
	if w.Validate(plan) != nil || w.Phase != expected {
		return Workspace{}, errors.New("stale private workspace checkpoint")
	}
	next := map[string]string{"planned": "root-dispatched", "root-dispatched": "root-created", "root-created": "layout-dispatched", "layout-dispatched": "layout-created", "layout-created": "observed"}
	if next[w.Phase] != phase {
		return Workspace{}, errors.New("private workspace cannot skip or repeat a mutation checkpoint")
	}
	if w.Phase != "root-dispatched" && root != w.Root {
		return Workspace{}, errors.New("private workspace root binding changed")
	}
	w = w.Clone()
	w.Phase = phase
	w.Root = root
	w.Observation = nil
	if observed != nil {
		o := *observed
		o.Mounts = slices.Clone(o.Mounts)
		w.Observation = &o
	}
	return w, w.Validate(plan)
}

type WorkspaceDriver interface {
	// Each mutation checks its exact owned parent/root again under the same fence.
	// CreateRoot must use exclusive creation, never adopt an existing directory.
	// Return a created root identity even when sync fails after its creation.
	CreateRoot(context.Context, Workspace) (pierstore.DirectoryIdentity, error)
	InspectRoot(context.Context, Workspace) error
	CreateLayout(context.Context, Workspace) error
	WorkspaceObserver
}

// Recovery requires inspection/sync only; a caller need not expose mkdir methods.
type WorkspaceObserver interface {
	ObserveLayout(context.Context, Workspace) (WorkspaceObservation, error)
}
type WorkspaceCommit func(expected string, next Workspace) error

// ExecuteWorkspace accepts a fresh journal only. A lost reply preserves the
// dispatch phase and never retries a directory mutation. All caller-owned data is
// cloned across callbacks so a driver cannot rewrite the retained selection.
func ExecuteWorkspace(ctx context.Context, plan Plan, w Workspace, d WorkspaceDriver, commit WorkspaceCommit, now func() time.Time) (Workspace, error) {
	if w.Validate(plan) != nil || w.Phase != "planned" || d == nil || commit == nil || now == nil {
		return w, errors.New("private workspace requires fresh intent, driver and durable commit")
	}
	step := func(phase string, root pierstore.DirectoryIdentity, o *WorkspaceObservation) error {
		next, e := w.Advance(plan, w.Phase, phase, root, o)
		if e != nil {
			return e
		}
		if e = commit(w.Phase, next.Clone()); e != nil {
			return e
		}
		w = next
		return nil
	}
	uncertain := func() error {
		return errors.Join(errors.New("private workspace outcome unresolved; preserve its original operation and partial tree"), ctx.Err())
	}
	if e := ctx.Err(); e != nil {
		return w, e
	}
	if e := step("root-dispatched", w.Root, nil); e != nil {
		return w, e
	}
	if e := ctx.Err(); e != nil {
		return w, e
	}
	root, e := d.CreateRoot(ctx, w.Clone())
	if root != (pierstore.DirectoryIdentity{}) {
		if ce := step("root-created", root, nil); ce != nil {
			return w, ce
		}
	}
	if e != nil || w.Phase != "root-created" {
		return w, uncertain()
	}
	if e = ctx.Err(); e != nil {
		return w, e
	}
	if e = d.InspectRoot(ctx, w.Clone()); e != nil {
		return w, uncertain()
	}
	if e = ctx.Err(); e != nil {
		return w, e
	}
	if e = step("layout-dispatched", w.Root, nil); e != nil {
		return w, e
	}
	if e = ctx.Err(); e != nil {
		return w, e
	}
	if e = d.CreateLayout(ctx, w.Clone()); e != nil {
		return w, uncertain()
	}
	if e = step("layout-created", w.Root, nil); e != nil {
		return w, e
	}
	if e = ctx.Err(); e != nil {
		return w, e
	}
	observation, e := d.ObserveLayout(ctx, w.Clone())
	if e != nil {
		return w, uncertain()
	}
	if e = w.checkObservation(observation, now()); e != nil {
		return w, e
	}
	if e = ctx.Err(); e != nil {
		return w, e
	}
	if e = step("observed", w.Root, &observation); e != nil {
		return w, e
	}
	return w, nil
}

// ObserveWorkspace recovers a complete retained layout without directory changes.
// The observer syncs verified objects before returning evidence. It can
// reconcile a lost layout acknowledgment but cannot fill missing directories,
// recreate a root or turn a root-creation uncertainty into new authority.
func ObserveWorkspace(ctx context.Context, plan Plan, w Workspace, d WorkspaceObserver, commit WorkspaceCommit, now func() time.Time) (Workspace, error) {
	if w.Validate(plan) != nil || (w.Phase != "layout-dispatched" && w.Phase != "layout-created") || d == nil || commit == nil || now == nil {
		return w, errors.New("workspace observation requires an owned root and dispatched layout")
	}
	if e := ctx.Err(); e != nil {
		return w, e
	}
	o, e := d.ObserveLayout(ctx, w.Clone())
	if e != nil {
		return w, errors.New("retained workspace observation failed")
	}
	if e = w.checkObservation(o, now()); e != nil {
		return w, e
	}
	if e = ctx.Err(); e != nil {
		return w, e
	}
	next := w.Clone()
	next.Phase = "observed"
	next.Observation = &o
	next = next.Clone()
	if e = next.Validate(plan); e != nil {
		return w, e
	}
	if e = commit(w.Phase, next.Clone()); e != nil {
		return w, e
	}
	return next, nil
}

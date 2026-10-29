package projectnetwork

import (
	"context"
	"errors"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// AnchorStart is the next lifecycle contract after completed preparation. It is
// persisted by the private runner store; it is not exposed to an RPC caller.
// Its creation receipt/IDs remain immutable across this distinct start operation.
// Completion establishes a namespace/publication binding, never member readiness.
type AnchorStart struct {
	Prepared    Preparation        `json:"prepared"`
	OperationID string             `json:"operation_id"`
	Phase       string             `json:"phase"`
	Namespace   *NamespaceIdentity `json:"namespace,omitempty"`
}

func NewAnchorStart(plan Plan, p Preparation, operation string) (AnchorStart, error) {
	s := AnchorStart{Prepared: p, OperationID: operation, Phase: "planned"}
	return s, s.Validate(plan)
}
func (s AnchorStart) Validate(plan Plan) error {
	if s.Prepared.Validate(plan) != nil || s.Prepared.Sequence != 6 || s.Prepared.Anchor.Phase != "observed" || !protocol.ValidID(s.OperationID) || s.OperationID == s.Prepared.OperationID {
		return errors.New("anchor start requires a distinct operation and completed resource preparation")
	}
	switch s.Phase {
	case "planned", "dispatched", "started":
		if s.Namespace != nil {
			return errors.New("unobserved anchor start cannot claim a namespace")
		}
	case "observed":
		if s.Namespace == nil || !s.Namespace.valid() {
			return errors.New("observed anchor lacks a boot-bound namespace")
		}
	default:
		return errors.New("unsupported anchor start phase")
	}
	return nil
}
func (s AnchorStart) Advance(plan Plan, expected, phase string, namespace *NamespaceIdentity) (AnchorStart, error) {
	if s.Validate(plan) != nil || s.Phase != expected {
		return AnchorStart{}, errors.New("stale or invalid anchor start")
	}
	next := map[string]string{"planned": "dispatched", "dispatched": "started", "started": "observed"}
	if next[s.Phase] != phase {
		return AnchorStart{}, errors.New("anchor start cannot skip or replay a phase")
	}
	s.Phase = phase
	s.Namespace = nil
	if namespace != nil {
		value := *namespace
		s.Namespace = &value
	}
	return s, s.Validate(plan)
}

// AnchorStartEngine must inspect the complete fixed creation contracts by exact
// ID, and obtain a fresh host-boot-bound namespace for a running anchor. No member
// can have been launched for this initial start. Unknown attachments are refused.
// Complete must include the private engine's full container network-mode inventory,
// including indirect container: joins; bridge endpoints alone cannot establish it.
type AnchorStartEngine interface {
	VerifyEngine(context.Context, string) error
	InspectAnchor(context.Context, Preparation) (Observation, error)
	StartAnchor(context.Context, Preparation) error
}
type AnchorStartCommit func(expected, phase string, namespace *NamespaceIdentity) error

func (s AnchorStart) inspect(plan Plan, o Observation, now time.Time, running bool) error {
	p := s.Prepared
	bridge := ResourceIdentity{p.Bridge.ID, p.Bridge.ContractDigest}
	anchor := ResourceIdentity{p.Anchor.ID, p.Anchor.ContractDigest}
	if now.IsZero() || !o.Complete || o.ObservedAt.IsZero() || now.Sub(o.ObservedAt) > 30*time.Second || o.ObservedAt.Sub(now) > 5*time.Second || o.ProjectID != p.ProjectID || o.Revision != p.ProjectRevision || o.Generation != p.Generation || o.EngineID != p.EngineID || len(o.Members) != 0 || !o.Bridge.Present || !o.Anchor.Present || !exactResource(bridge, o.Bridge) || !exactResource(anchor, o.Anchor.ResourceObservation) || o.Anchor.BridgeID != bridge.ID || o.Anchor.Running != running || !samePublications(plan, o.Anchor.Requested) {
		return ErrRuntimeObservation
	}
	if len(o.BridgeAttachments) > 1 || len(o.BridgeAttachments) == 1 && o.BridgeAttachments[0] != anchor.ID {
		return ErrRuntimeObservation
	}
	if running {
		if len(o.BridgeAttachments) != 1 || !o.Anchor.Namespace.valid() || !samePublications(plan, o.Anchor.Active) {
			return ErrRuntimeObservation
		}
	} else if o.Anchor.Namespace != (NamespaceIdentity{}) || len(o.Anchor.Active) != 0 {
		return ErrRuntimeObservation
	}
	return nil
}

// ExecuteAnchorStart requires a durable compare-and-commit callback and the
// runner resource/update fence. This controller never retries an uncertain Start,
// adopts/replaces resources, starts members or stops anything on failure.
func ExecuteAnchorStart(ctx context.Context, plan Plan, s AnchorStart, engine AnchorStartEngine, commit AnchorStartCommit, now func() time.Time) (AnchorStart, error) {
	if err := s.Validate(plan); err != nil {
		return s, err
	}
	if s.Phase != "planned" || engine == nil || commit == nil || now == nil {
		return s, errors.New("anchor start requires a fresh journal, driver and durable callback")
	}
	fail := func(message string) error { return errors.Join(errors.New(message), ctx.Err()) }
	verify := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if engine.VerifyEngine(ctx, s.Prepared.EngineID) != nil {
			return fail("anchor engine identity could not be verified")
		}
		return ctx.Err()
	}
	inspect := func(running bool) (Observation, error) {
		if err := verify(); err != nil {
			return Observation{}, err
		}
		o, err := engine.InspectAnchor(ctx, s.Prepared)
		if err != nil {
			return o, fail("anchor inspection did not establish its recorded ownership")
		}
		if err = verify(); err != nil {
			return o, err
		}
		if err = s.inspect(plan, o, now(), running); err != nil {
			return o, err
		}
		return o, nil
	}
	advance := func(phase string, namespace *NamespaceIdentity) error {
		next, err := s.Advance(plan, s.Phase, phase, namespace)
		if err != nil {
			return err
		}
		// Pass an independent copy so a callback cannot rewrite this state's proof.
		var copyNamespace *NamespaceIdentity
		if next.Namespace != nil {
			value := *next.Namespace
			copyNamespace = &value
		}
		if err = commit(s.Phase, phase, copyNamespace); err != nil {
			return err
		}
		s = next
		return nil
	}
	if _, err := inspect(false); err != nil {
		return s, err
	}
	if err := advance("dispatched", nil); err != nil {
		return s, err
	}
	if err := ctx.Err(); err != nil {
		return s, err
	}
	if engine.StartAnchor(ctx, s.Prepared) != nil {
		return s, fail("anchor Start outcome is unresolved; retain its original receipt")
	}
	// Preserve a successful client acknowledgement even if cancellation follows it.
	if err := advance("started", nil); err != nil {
		return s, err
	}
	observed, err := inspect(true)
	if err != nil {
		return s, err
	}
	if err = advance("observed", &observed.Anchor.Namespace); err != nil {
		return s, err
	}
	return s, nil
}

package projectnetwork

import (
	"context"
	"errors"
)

// PreparationEngine is a private driver boundary, not a public command API.
// Create must use the fixed reviewed contract; Inspect must independently check
// its complete immutable settings and exact ID. Neither method may adopt by name.
type PreparationEngine interface {
	VerifyEngine(context.Context, string) error
	Create(context.Context, string, Creation, string) (string, error)
	Inspect(context.Context, string, Creation, string) (ResourceIdentity, error)
}

type PreparationCommit func(sequence uint64, kind, phase, id string) error

// ExecutePreparation creates only the bridge and stopped anchor. The callback
// durably compares the operation/sequence and commits each step before a later
// engine call. Existing nonzero sequences require explicit recovery; this entry
// point never replays a dispatched create, auto-starts members or cleans up.
func ExecutePreparation(ctx context.Context, plan Plan, p Preparation, engine PreparationEngine, commit PreparationCommit) (Preparation, error) {
	if err := p.Validate(plan); err != nil {
		return p, err
	}
	if p.Sequence != 0 || engine == nil || commit == nil {
		return p, errors.New("project preparation requires a fresh owned journal and driver")
	}
	failure := func(message string) error { return errors.Join(errors.New(message), ctx.Err()) }
	advance := func(kind, phase, id string) error {
		next, err := p.Advance(plan, p.Sequence, kind, phase, id)
		if err != nil {
			return err
		}
		if err = commit(p.Sequence, kind, phase, id); err != nil {
			return err
		}
		p = next
		return nil
	}
	verify := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if engine.VerifyEngine(ctx, p.EngineID) != nil {
			return failure("project engine binding could not be verified")
		}
		return ctx.Err()
	}
	for _, kind := range []string{"bridge", "anchor"} {
		if err := verify(); err != nil {
			return p, err
		}
		if err := advance(kind, "dispatched", ""); err != nil {
			return p, err
		}
		// A canceled caller cannot dispatch after the write-ahead commit.
		if err := ctx.Err(); err != nil {
			return p, err
		}
		resource := p.Bridge
		bridgeID := ""
		if kind == "anchor" {
			resource = p.Anchor
			bridgeID = p.Bridge.ID
		}
		id, createErr := engine.Create(ctx, kind, resource, bridgeID)
		if digestPattern.MatchString(id) {
			// Even a failed client call can return a valid creation ID. Preserve it
			// before interpreting the error; inspection cannot run until this commits.
			if err := advance(kind, "created", id); err != nil {
				return p, err
			}
		} else {
			return p, failure("project creation outcome is unresolved; retain the original journal")
		}
		if createErr != nil {
			return p, failure("project creation returned an uncertain outcome; retain its resource ID")
		}
		if err := verify(); err != nil {
			return p, err
		}
		resource = p.Bridge
		if kind == "anchor" {
			resource = p.Anchor
		}
		actual, err := engine.Inspect(ctx, kind, resource, bridgeID)
		if err != nil || actual.ID != resource.ID || actual.ContractDigest != resource.ContractDigest {
			return p, failure("created project resource did not match its recorded contract")
		}
		if err := verify(); err != nil {
			return p, err
		}
		if err := advance(kind, "observed", id); err != nil {
			return p, err
		}
	}
	return p, nil
}

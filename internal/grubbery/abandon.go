package grubbery

import "context"

// AbandonProof is a trusted read-only statement about the original pending
// record and its still-unchanged live source. It is never a caller assertion.
// It does not authorize deleting staging, replacing markers, or changing apps.
type AbandonProof struct {
	Version              int    `json:"version"`
	PlanDigest           string `json:"plan_digest"`
	BindingDigest        string `json:"binding_digest"`
	Revision             uint64 `json:"revision"`
	PreviousDigest       string `json:"previous_digest"`
	Epoch                uint64 `json:"epoch"`
	ReleaseRoot          string `json:"release_root"`
	NativeEvidenceDigest string `json:"native_evidence_digest"`
}

// AbandonBackend must freshly check role, policy, exact framework/process/pier,
// original observed ownership markers and that the live source still equals
// Plan.Before. It performs no HTTP write or native dispatch.
type AbandonBackend interface {
	ObserveAbandon(context.Context, Record) (*AbandonProof, error)
}

func (r Record) tailDigest() string {
	if len(r.Events) == 0 {
		return r.HeaderDigest
	}
	return r.Events[len(r.Events)-1].Digest
}
func (p AbandonProof) matches(r Record) bool {
	return p.Version == 1 && p.PlanDigest == r.Plan.Digest && p.BindingDigest == bindingDigest(r.Plan) && p.Revision == r.Revision && p.PreviousDigest == r.tailDigest() && p.Epoch == r.Plan.Before.Epoch && p.ReleaseRoot == r.Plan.Before.ReleaseRoot && digest.MatchString(p.NativeEvidenceDigest)
}
func (r Record) Abandoned() bool {
	return len(r.Events) > 0 && r.Events[len(r.Events)-1].State == "abandoned"
}

// abandonable is evaluated on the prefix before its terminal abandonment event.
// Every submitted write must already have its original exact after-proof. Even
// an observed source poke forbids abandonment: it could have activated code.
// Reads may remain uncertain, including a failed compilation observation.
func (r Record) abandonable() (bool, bool) {
	states := map[string]string{}
	for _, event := range r.Events {
		states[event.StepID] = event.State
	}
	effects := false
	n, _ := r.Plan.Binding.Namespace()
	created := map[string]bool{}
	markers := map[string]bool{}
	for _, step := range r.Plan.Steps {
		state := states[step.ID]
		if state == "" {
			continue
		}
		if step.Kind == "poke-source" {
			return false, false
		}
		if step.Kind == "observe-code" || step.Kind == "observe-followed-source" {
			continue
		}
		if state != "observed" {
			return false, false
		}
		effects = true
		if step.Kind == "make-directory" {
			created[step.Path] = true
		}
		if step.Kind == "make-file" {
			markers[step.Path] = true
		}
	}
	if created[n.SourceRoot] && !markers[n.SourceRoot+"/omabit-owner.json"] || created[n.DeskRoot] && !markers[n.DataRoot+"/omabit-owner.json"] {
		return false, false
	}
	return true, effects
}

// Terminal distinguishes abandonment from source-followed completion. In both
// cases the store releases only this exact operation's namespace lease.
func (r Record) Terminal() (bool, error) {
	if r.Validate() != nil {
		return false, ErrHistory
	}
	if r.Abandoned() {
		return true, nil
	}
	_, complete, err := r.Current()
	return complete, err
}

// Abandon appends an audit event and releases the lease; all source, markers,
// receipts and previous live app data remain. It cannot resolve unknown writes
// or a dispatched activation. A reply loss replays without cache or probing.
func (e Execution) Abandon(ctx context.Context, c Call) (Record, error) {
	r, err := e.load(ctx, c)
	if err != nil {
		return Record{}, err
	}
	if _, err = r.State(c.StepID); err != nil {
		return r, err
	}
	if r.Abandoned() {
		return r, nil
	}
	step, done, err := r.Current()
	if err != nil {
		return r, err
	}
	if done || step.ID != c.StepID {
		return r, ErrPhase
	}
	allowed, effects := r.abandonable()
	if !allowed {
		return r, ErrAdmission
	}
	var proof *AbandonProof
	if effects {
		backend, ok := e.Backend.(AbandonBackend)
		if !ok || ctx.Err() != nil {
			return r, ErrAdmission
		}
		proof, err = backend.ObserveAbandon(ctx, r)
		if err != nil || proof == nil || !proof.matches(r) {
			return r, ErrAdmission
		}
	}
	return e.append(ctx, c, r, Event{StepID: step.ID, State: "abandoned", At: e.now(), Reason: "preactivation-abandoned", AbandonProof: proof})
}

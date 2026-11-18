package grubbery

import (
	"errors"
	"time"
)

var (
	ErrMissing       = errors.New("app operation not found")
	ErrConflict      = errors.New("app operation binding changed")
	ErrPhase         = errors.New("select the exact original app phase in saved progress")
	ErrHistory       = errors.New("app operation history changed")
	ErrCAS           = errors.New("app operation revision changed")
	ErrNamespaceBusy = errors.New("app namespace has another unfinished operation")
	ErrAdmission     = errors.New("app phase was not admitted")
	ErrSource        = errors.New("exact staged app source is unavailable")
	ErrJournal       = errors.New("app phase journal outcome is unconfirmed; read original saved progress")
)

// Review is produced only by a trusted admission adapter after current role,
// framework/process/pier/namespace/ownership/resource/policy checks. It is never
// accepted as authorization from a tool argument or HTTP request body.
type Review struct {
	Version              int    `json:"version"`
	PlanDigest           string `json:"plan_digest"`
	StepID               string `json:"step_id"`
	BindingDigest        string `json:"binding_digest"`
	PolicyRevision       uint64 `json:"policy_revision"`
	NativeEvidenceDigest string `json:"native_evidence_digest"`
}

// Proof is a trusted adapter's bounded read-after statement, not an app's health
// assertion. A followed-source proof does not establish route, permit or app tests.
type Proof struct {
	Version              int    `json:"version"`
	PlanDigest           string `json:"plan_digest"`
	StepID               string `json:"step_id"`
	BindingDigest        string `json:"binding_digest"`
	NativeEvidenceDigest string `json:"native_evidence_digest"`
}

func bindingDigest(p Plan) string { return hash([]byte(textJSON(p.Binding))) }
func (r Review) matches(p Plan, id string) bool {
	return r.Version == 1 && r.PlanDigest == p.Digest && r.StepID == id && r.BindingDigest == bindingDigest(p) && r.PolicyRevision <= 1<<53-1 && digest.MatchString(r.NativeEvidenceDigest)
}
func (r Proof) matches(p Plan, id string) bool {
	return r.Version == 1 && r.PlanDigest == p.Digest && r.StepID == id && r.BindingDigest == bindingDigest(p) && digest.MatchString(r.NativeEvidenceDigest)
}

type Event struct {
	Sequence       uint64    `json:"sequence"`
	StepID         string    `json:"step_id"`
	State          string    `json:"state"`
	At             time.Time `json:"at"`
	Review         *Review   `json:"review,omitempty"`
	Proof          *Proof    `json:"proof,omitempty"`
	ResponseDigest string    `json:"response_digest,omitempty"`
	Reason         string    `json:"reason,omitempty"`
	PreviousDigest string    `json:"previous_digest"`
	Digest         string    `json:"digest"`
}

type Record struct {
	Version      int       `json:"version"`
	Plan         Plan      `json:"plan"`
	CreatedAt    time.Time `json:"created_at"`
	HeaderDigest string    `json:"header_digest"`
	Revision     uint64    `json:"revision"`
	Events       []Event   `json:"events"`
}

func recordHeader(r Record) string {
	return hash([]byte(textJSON(struct {
		Version    int
		PlanDigest string
		CreatedAt  time.Time
	}{r.Version, r.Plan.Digest, r.CreatedAt})))
}

func utcTime(at time.Time) bool {
	_, offset := at.Zone()
	return !at.IsZero() && at.Year() >= 1 && at.Year() <= 9999 && offset == 0
}
func eventDigest(e Event) string { e.Digest = ""; return hash([]byte(textJSON(e))) }

func NewRecord(p Plan, at time.Time) (Record, error) {
	if p.ValidateMetadata() != nil || !utcTime(at) {
		return Record{}, ErrHistory
	}
	r := Record{Version: 1, Plan: p, CreatedAt: at.UTC(), Events: []Event{}}
	r.HeaderDigest = recordHeader(r)
	return r, nil
}

// Validate keeps the immutable plan and each previous event hash bound. Clock
// corrections do not reorder history: sequence/CAS and the digest chain do.
func (r Record) Validate() error {
	if r.Version != 1 || r.Plan.ValidateMetadata() != nil || !utcTime(r.CreatedAt) || r.HeaderDigest != recordHeader(r) || r.Events == nil || len(r.Events) > 1024 || r.Revision != uint64(len(r.Events)) {
		return ErrHistory
	}
	states := make([]string, len(r.Plan.Steps))
	previous := r.HeaderDigest
	for i, event := range r.Events {
		if event.Sequence != uint64(i+1) || !utcTime(event.At) || event.PreviousDigest != previous || event.Digest != eventDigest(event) {
			return ErrHistory
		}
		index := -1
		for j, step := range r.Plan.Steps {
			if step.ID == event.StepID {
				index = j
				break
			}
		}
		if index < 0 {
			return ErrHistory
		}
		for j := 0; j < index; j++ {
			if states[j] != "observed" {
				return ErrHistory
			}
		}
		before := states[index]
		switch event.State {
		case "dispatched":
			if before != "" || event.Review == nil || !event.Review.matches(r.Plan, event.StepID) || event.Proof != nil || event.ResponseDigest != "" || event.Reason != "" {
				return ErrHistory
			}
		case "acknowledged":
			if before != "dispatched" || event.Review != nil || event.Proof != nil || !digest.MatchString(event.ResponseDigest) || event.Reason != "" {
				return ErrHistory
			}
		case "uncertain":
			if before != "dispatched" && before != "acknowledged" || event.Review != nil || event.Proof != nil || event.ResponseDigest != "" || event.Reason != "transport-outcome-unknown" && event.Reason != "observation-unavailable" && event.Reason != "worker-interrupted" {
				return ErrHistory
			}
		case "observed":
			if before != "dispatched" && before != "acknowledged" && before != "uncertain" || event.Review != nil || event.Proof == nil || !event.Proof.matches(r.Plan, event.StepID) || event.ResponseDigest != "" || event.Reason != "" {
				return ErrHistory
			}
		default:
			return ErrHistory
		}
		states[index] = event.State
		previous = event.Digest
	}
	return nil
}

func (r Record) WithEvent(event Event) (Record, error) {
	if r.Validate() != nil {
		return Record{}, ErrHistory
	}
	event.Sequence = r.Revision + 1
	event.PreviousDigest = r.HeaderDigest
	if len(r.Events) > 0 {
		event.PreviousDigest = r.Events[len(r.Events)-1].Digest
	}
	event.At = event.At.UTC()
	event.Digest = eventDigest(event)
	out := r
	out.Events = append(append([]Event{}, r.Events...), event)
	out.Revision++
	if out.Validate() != nil {
		return Record{}, ErrHistory
	}
	return out, nil
}

func (r Record) State(stepID string) (string, error) {
	if r.Validate() != nil {
		return "", ErrHistory
	}
	known := false
	for _, s := range r.Plan.Steps {
		if s.ID == stepID {
			known = true
			break
		}
	}
	if !known {
		return "", ErrPhase
	}
	state := "pending"
	for _, e := range r.Events {
		if e.StepID == stepID {
			state = e.State
		}
	}
	return state, nil
}

// Current is saved progress only. It does not assert current runtime/app health
// and selecting it does not advance a phase or authorize an effect.
func (r Record) Current() (Step, bool, error) {
	if r.Validate() != nil {
		return Step{}, false, ErrHistory
	}
	states := map[string]string{}
	for _, event := range r.Events {
		states[event.StepID] = event.State
	}
	for _, s := range r.Plan.Steps {
		if states[s.ID] != "observed" {
			return s, false, nil
		}
	}
	return Step{}, true, nil
}

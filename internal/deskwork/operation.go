package deskwork

import (
	"errors"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
)

// Observation is evidence for one selected runtime and desk at a point in time,
// not an authorization. Effects must obtain another observation under the
// runner's mutation lock and check the current policy immediately before use.
type Observation struct {
	Exists      bool   `json:"exists"`
	Revision    uint64 `json:"revision,omitempty"`
	ClayHash    string `json:"clay_hash,omitempty"`
	Publication string `json:"publication,omitempty"`
}

func (o Observation) Validate() error {
	if !o.Exists {
		if o.Revision != 0 || o.ClayHash != "" || o.Publication != "" {
			return errors.New("invalid absent desk observation")
		}
		return nil
	}
	if o.Revision == 0 || o.Revision > apppolicy.MaxRevision || len(o.ClayHash) < 3 || len(o.ClayHash) > 128 || o.ClayHash[:2] != "0v" || o.Publication != "private" && o.Publication != "published" {
		return errors.New("invalid desk observation")
	}
	for _, c := range o.ClayHash[2:] {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'v' || c == '.') {
			return errors.New("invalid Clay hash")
		}
	}
	return nil
}

// Intent records what the agent reviewed. It never carries a grant. RuntimeID
// is the exact full container identity; project targets also bind generation.
type Intent struct {
	Version        int              `json:"version"`
	ID             string           `json:"id"`
	Target         apppolicy.Target `json:"target"`
	RuntimeID      string           `json:"runtime_id"`
	ConfigRevision uint64           `json:"config_revision"`
	PolicyRevision uint64           `json:"policy_revision"`
	Action         string           `json:"action"`
	Baseline       Observation      `json:"baseline"`
	SourceDigest   string           `json:"source_digest,omitempty"`
}

func (i Intent) Validate() error {
	if i.Version != 1 || !digestPattern.MatchString(i.ID) || !digestPattern.MatchString(i.RuntimeID) || i.ConfigRevision == 0 || i.ConfigRevision > apppolicy.MaxRevision || i.PolicyRevision > apppolicy.MaxRevision || i.Target.Validate() != nil || i.Target.InstanceID == "" || i.Target.Desk == "" || i.Target.ProjectID != "" && i.Target.Generation == 0 || i.Baseline.Validate() != nil {
		return errors.New("invalid desk operation intent")
	}
	switch i.Action {
	case "install-remote":
		if i.Baseline.Exists || !digestPattern.MatchString(i.SourceDigest) {
			return errors.New("receiver installation requires absence and a frozen publisher selection")
		}
	case "create-desk":
		if i.Baseline.Exists || i.SourceDigest != "" {
			return errors.New("desk creation requires observed absence")
		}
	case "commit", "adopt-source", "upload-frontend":
		if !i.Baseline.Exists || !digestPattern.MatchString(i.SourceDigest) {
			return errors.New("commit requires an observed desk and frozen source")
		}
	case "publish-source", "publish-treaty", "activate-local":
		if !i.Baseline.Exists || i.SourceDigest != "" {
			return errors.New("publication requires an observed desk")
		}
	default:
		return errors.New("unsupported desk operation")
	}
	return nil
}
func (i Intent) Check(p apppolicy.Policy, current Observation) error {
	if err := i.Validate(); err != nil {
		return err
	}
	if err := current.Validate(); err != nil {
		return err
	}
	if current != i.Baseline {
		return errors.New("desk changed after review; inspect again")
	}
	publication := current.Publication
	if !current.Exists {
		publication = "absent"
	}
	if i.Action == "upload-frontend" {
		// Serving an artifact can immediately publish it. Both gates apply even
		// while Clay is private; native HTTP has no atomic publication fence.
		if err := p.Check(i.Target, i.PolicyRevision, "commit", publication); err != nil {
			return err
		}
		return p.Check(i.Target, i.PolicyRevision, "publish", publication)
	}
	effect := i.Action
	if effect == "activate-local" || effect == "adopt-source" {
		effect = "commit"
	}
	if effect == "publish-source" || effect == "publish-treaty" {
		effect = "publish"
	}
	return p.Check(i.Target, i.PolicyRevision, effect, publication)
}

type Step struct {
	Actor                string            `json:"actor,omitempty"`
	VerifiedSourceDigest string            `json:"verified_source_digest,omitempty"`
	Kind                 string            `json:"kind"`
	State                string            `json:"state"`
	At                   time.Time         `json:"at"`
	Observation          *Observation      `json:"observation,omitempty"`
	FrontendSourceDigest string            `json:"frontend_source_digest,omitempty"`
	Frontend             *FrontendState    `json:"frontend,omitempty"`
	FrontendTransfer     *FrontendTransfer `json:"frontend_transfer,omitempty"`
	Application          *AppState         `json:"application,omitempty"`
}
type Receipt struct {
	Version int    `json:"version"`
	Intent  Intent `json:"intent"`
	Actor   string `json:"actor"`
	Steps   []Step `json:"steps"`
}

func (r Receipt) Validate() error {
	if r.Version != 1 || r.Intent.Validate() != nil || r.Actor == "" || len(r.Actor) > 128 || len(r.Steps) == 0 || len(r.Steps) > 16 {
		return errors.New("invalid desk operation receipt")
	}
	state := ""
	for _, s := range r.Steps {
		if s.At.IsZero() || s.Kind != r.Intent.Action || len(s.Actor) > 128 {
			return errors.New("invalid desk operation step")
		}
		valid := false
		switch state {
		case "":
			valid = s.State == "planned"
		case "planned":
			valid = s.State == "dispatched" || s.State == "blocked" || s.State == "interrupted"
			if r.Intent.Action == "adopt-source" {
				valid = s.State == "observed" || s.State == "blocked"
			}
		case "dispatched":
			valid = s.State == "acknowledged" || s.State == "rejected" || s.State == "uncertain"
		case "acknowledged", "uncertain":
			valid = s.State == "observed" || s.State == "uncertain" && state != "uncertain"
		}
		if !valid {
			return errors.New("invalid desk operation transition")
		}
		if s.State == "observed" {
			if s.Observation == nil || s.Observation.Validate() != nil || !s.Observation.Exists {
				return errors.New("completed desk effect requires observation")
			}
			switch r.Intent.Action {
			case "upload-frontend":
				if s.Observation.Revision <= r.Intent.Baseline.Revision || s.VerifiedSourceDigest != r.Intent.SourceDigest || s.Frontend == nil || s.Application == nil || !digestPattern.MatchString(s.FrontendSourceDigest) || s.Frontend.Validate() != nil || s.Application.Validate() != nil || s.Application.SourceRevision != s.Observation.Revision || s.Frontend.SourceRevision != s.Observation.Revision {
					return errors.New("frontend result was not observed")
				}
			case "install-remote":
				if s.Application == nil {
					return errors.New("receiver subscription observation absent")
				}
				publisher := Publisher{Ship: s.Application.SourceShip, Desk: s.Application.SourceDesk, ExpectedHash: s.Observation.ClayHash}
				if VerifyReceiver(publisher, *s.Observation, *s.Application) != nil || publisher.Digest() != r.Intent.SourceDigest {
					return errors.New("reviewed publisher source not observed")
				}
			case "adopt-source":
				if *s.Observation != r.Intent.Baseline || s.VerifiedSourceDigest != r.Intent.SourceDigest {
					return errors.New("adoption changed the reviewed source observation")
				}
			case "commit":
				if s.Observation.Revision <= r.Intent.Baseline.Revision || s.VerifiedSourceDigest != r.Intent.SourceDigest {
					return errors.New("commit revision was not observed")
				}
			case "publish-source":
				fallthrough
			case "publish-treaty", "activate-local":
				if s.Application == nil || VerifyRelease(r.Intent.Action, r.Intent.Baseline, *s.Observation, *s.Application) != nil {
					return errors.New("selected release stage was not observed")
				}
			}
		} else if s.Observation != nil || s.VerifiedSourceDigest != "" || s.Application != nil || s.Frontend != nil || s.FrontendSourceDigest != "" {
			return errors.New("unexpected desk step observation")
		}
		if r.Intent.Action != "commit" && r.Intent.Action != "adopt-source" && r.Intent.Action != "upload-frontend" && s.VerifiedSourceDigest != "" {
			return errors.New("unexpected source verification")
		}
		if !IsReleaseAction(r.Intent.Action) && r.Intent.Action != "install-remote" && r.Intent.Action != "upload-frontend" && s.Application != nil {
			return errors.New("unexpected application stage evidence")
		}
		if (s.Frontend != nil || s.FrontendSourceDigest != "") && r.Intent.Action != "upload-frontend" {
			return errors.New("unexpected frontend evidence")
		}
		if r.Intent.Action == "upload-frontend" && (s.State == "acknowledged" || s.State == "rejected") && s.FrontendTransfer == nil {
			return errors.New("frontend transport evidence absent")
		}
		if s.FrontendTransfer != nil {
			if r.Intent.Action != "upload-frontend" || s.FrontendTransfer.Validate() != nil || s.State != "acknowledged" && s.State != "uncertain" && s.State != "rejected" || (s.State == "acknowledged") != s.FrontendTransfer.Acknowledged || s.State == "rejected" && s.FrontendTransfer.Attempted {
				return errors.New("invalid frontend transport evidence")
			}
		}
		state = s.State
	}
	return nil
}

// Recover marks unfinished work without replay. A separately requested,
// read-only reconciliation can append exact effect evidence to uncertain work.
func (r Receipt) Recover(at time.Time) (Receipt, error) {
	if err := r.Validate(); err != nil {
		return r, err
	}
	next := ""
	switch r.Steps[len(r.Steps)-1].State {
	case "planned":
		next = "interrupted"
	case "dispatched", "acknowledged":
		next = "uncertain"
	}
	if next != "" {
		r.Steps = append(append([]Step{}, r.Steps...), Step{Kind: r.Intent.Action, State: next, At: at})
	}
	return r, r.Validate()
}

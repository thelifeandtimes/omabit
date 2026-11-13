package deskwork

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"
)

// Adoption records explicit ownership of selected staged files that already
// equal Clay. It never writes source, commits Clay, or acquires unspecified paths.
// Existing ownership outside the selection is retained, including its original
// baseline: later external changes still conflict in the commit planner.
type AdoptionPlan struct {
	Version      int       `json:"version"`
	Previous     Ownership `json:"previous"`
	Current      Manifest  `json:"current"`
	Source       Manifest  `json:"source"`
	Known        Manifest  `json:"known"`
	Next         Ownership `json:"next"`
	Conflicts    []string  `json:"conflicts"`
	NewCount     int       `json:"new_count"`
	RefreshCount int       `json:"refresh_count"`
	Digest       string    `json:"digest"`
}

func PlanAdoption(previous Ownership, current, source, known Manifest) (AdoptionPlan, error) {
	var out AdoptionPlan
	if previous.Validate() != nil || !canonicalManifest(current) || source.Validate() != nil || !canonicalManifest(known) || len(source.Files) == 0 {
		return out, errors.New("adoption requires nonempty, valid source and Clay inventories")
	}
	now, normalized, selected := current.Index(), known.Index(), source.Index()
	for name := range normalized {
		if _, ok := selected[name]; !ok {
			return out, errors.New("normalization outside adoption selection")
		}
	}
	raw, clay := previous.Raw.Index(), previous.Clay.Index()
	out = AdoptionPlan{Version: 1, Previous: previous, Current: current, Source: source, Known: known, Conflicts: []string{}}
	for _, f := range source.Files {
		c, present := now[f.Path]
		normalizedFile, converted := normalized[f.Path]
		if !present || !converted || normalizedFile != c {
			out.Conflicts = append(out.Conflicts, f.Path)
			continue
		}
		if old, owned := raw[f.Path]; !owned {
			out.NewCount++
		} else if old != f || clay[f.Path] != c {
			out.RefreshCount++
		}
		raw[f.Path], clay[f.Path] = f, c
	}
	list := func(m map[string]File) []File {
		a := []File{}
		for _, f := range m {
			a = append(a, f)
		}
		return a
	}
	a, err := NewManifest(list(raw))
	if err != nil {
		return AdoptionPlan{}, err
	}
	b, err := NewManifest(list(clay))
	if err != nil {
		return AdoptionPlan{}, err
	}
	out.Next = Ownership{1, a, b}
	data, _ := json.Marshal(out)
	sum := sha256.Sum256(data)
	out.Digest = hex.EncodeToString(sum[:])
	return out, nil
}
func (p AdoptionPlan) Validate() error {
	want, err := PlanAdoption(p.Previous, p.Current, p.Source, p.Known)
	if err != nil {
		return err
	}
	a, _ := json.Marshal(p)
	b, _ := json.Marshal(want)
	if !bytes.Equal(a, b) {
		return errors.New("adoption plan changed")
	}
	return nil
}

// The fields match a staged commit's binding, but the action and receipt cannot
// be confused with a Clay commit. Adoption changes only runner metadata.
type AdoptionIntent CommitIntent

func (i AdoptionIntent) Validate() error {
	if i.Intent.Validate() != nil || i.Intent.Action != "adopt-source" {
		return errors.New("invalid source adoption intent")
	}
	c := CommitIntent(i)
	c.Intent.Action = "commit"
	return c.Validate()
}

type AdoptionRecord struct {
	Version   int            `json:"version"`
	Intent    AdoptionIntent `json:"intent"`
	Actor     string         `json:"actor"`
	Plan      AdoptionPlan   `json:"plan"`
	CreatedAt time.Time      `json:"created_at"`
}

func (r AdoptionRecord) Validate() error {
	if r.Version != 1 || r.Intent.Validate() != nil || !settingText(r.Actor, 128, false) || r.Actor == "" || r.Plan.Validate() != nil || len(r.Plan.Conflicts) != 0 || r.Plan.NewCount+r.Plan.RefreshCount == 0 || r.Intent.PlanDigest != r.Plan.Digest || r.Intent.Intent.SourceDigest != r.Plan.Source.Digest || r.CreatedAt.IsZero() {
		return errors.New("invalid source adoption record")
	}
	return nil
}
func (o OwnedSource) CheckAdoption(r AdoptionRecord, receipt Receipt) error {
	if o.Validate() != nil || r.Validate() != nil || receipt.Validate() != nil {
		return errors.New("invalid source adoption evidence")
	}
	s := receipt.Steps[len(receipt.Steps)-1]
	if receipt.Intent != r.Intent.Intent || receipt.Actor != r.Actor || s.State != "observed" || s.Observation == nil || *s.Observation != o.Observation || s.VerifiedSourceDigest != r.Plan.Source.Digest || !s.At.Equal(o.ObservedAt) || o.Target != receipt.Intent.Target || o.OperationID != receipt.Intent.ID || o.SourceName != r.Intent.SourceName || o.Continuity != r.Intent.Continuity || o.Ownership.Raw.Digest != r.Plan.Next.Raw.Digest || o.Ownership.Clay.Digest != r.Plan.Next.Clay.Digest {
		return errors.New("ownership does not match adoption receipt")
	}
	return nil
}

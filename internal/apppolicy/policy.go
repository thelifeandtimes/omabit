// Package apppolicy defines permissions for Omabit-managed Clay and publication
// effects. A policy decision is not a reusable authorization token.
package apppolicy

import (
	"errors"
	"fmt"
	"regexp"
)

const Version = 2
const MaxEntries = 2048
const MaxRevision = 1<<53 - 1 // Also exactly representable by the desktop.

type Permission string

const (
	Inherit Permission = "inherit"
	Allow   Permission = "allow"
	Block   Permission = "block"
)

type Gates struct {
	Commit     Permission `json:"commit"`
	Publish    Permission `json:"publish"`
	CreateDesk Permission `json:"create_desk"`
}

func Blocked() Gates   { return Gates{Block, Block, Block} }
func Inherited() Gates { return Gates{Inherit, Inherit, Inherit} }
func (g Gates) Validate(global bool) error {
	for _, p := range []Permission{g.Commit, g.Publish, g.CreateDesk} {
		if p != Allow && p != Block && (global || p != Inherit) {
			return errors.New("permissions must be allow, block, or scoped inherit")
		}
	}
	return nil
}

// A project target includes its current creator and generation. Replacing a
// member cannot carry a previous generation's grant into the replacement.
type Target struct {
	InstanceID string `json:"instance_id,omitempty"`
	ProjectID  string `json:"project_id,omitempty"`
	Generation uint64 `json:"generation,omitempty"`
	Creator    string `json:"creator,omitempty"`
	Desk       string `json:"desk,omitempty"`
}

var id = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
var desk = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

func (t Target) Validate() error {
	if t.InstanceID == "" {
		if t != (Target{}) {
			return errors.New("ship identity required")
		}
		return nil
	}
	if !id.MatchString(t.InstanceID) || t.Desk != "" && !desk.MatchString(t.Desk) {
		return errors.New("invalid ship or desk identity")
	}
	if t.ProjectID == "" && (t.Generation != 0 || t.Creator != "") || t.ProjectID != "" && (!id.MatchString(t.ProjectID) || t.Generation > MaxRevision || t.Generation == 0 && t.Creator != "" || t.Generation != 0 && !id.MatchString(t.Creator)) {
		return errors.New("invalid project member generation")
	}
	return nil
}
func (t Target) Ship() Target { t.Desk = ""; return t }

type Entry struct {
	Target Target `json:"target"`
	Gates  Gates  `json:"gates"`
}
type Policy struct {
	Version  int     `json:"version"`
	Revision uint64  `json:"revision"`
	Global   Gates   `json:"global"`
	Runner   Gates   `json:"runner"`
	Entries  []Entry `json:"entries"`
}

func Default() Policy {
	return Policy{Version: Version, Global: Blocked(), Runner: Inherited(), Entries: []Entry{}}
}
func (p Policy) Validate() error {
	if p.Version != Version || p.Revision > MaxRevision || p.Entries == nil || len(p.Entries) > MaxEntries {
		return errors.New("unsupported or invalid application policy")
	}
	if err := p.Global.Validate(true); err != nil {
		return err
	}
	if err := p.Runner.Validate(false); err != nil {
		return err
	}
	seen := map[Target]bool{}
	for _, e := range p.Entries {
		if e.Target.InstanceID == "" || e.Target.Validate() != nil || e.Gates.Validate(false) != nil || e.Target.Desk != "" && e.Gates.CreateDesk != Inherit || seen[e.Target] {
			return errors.New("invalid or duplicate policy scope")
		}
		seen[e.Target] = true
	}
	return nil
}

type Decision struct {
	Value  Permission `json:"value"`
	Source string     `json:"source"`
}
type Effective struct {
	Commit     Decision `json:"commit"`
	Publish    Decision `json:"publish"`
	CreateDesk Decision `json:"create_desk"`
}

func (p Policy) Effective(t Target) Effective {
	out := Effective{Decision{Block, "unavailable"}, Decision{Block, "unavailable"}, Decision{Block, "unavailable"}}
	if p.Validate() != nil || t.Validate() != nil {
		return out
	}
	apply := func(g Gates, scope string) {
		if g.Commit != Inherit {
			out.Commit = Decision{g.Commit, scope}
		}
		if g.Publish != Inherit {
			out.Publish = Decision{g.Publish, scope}
		}
		// Desks cannot authorize the creation of sibling desks.
		if scope != "desk" && g.CreateDesk != Inherit {
			out.CreateDesk = Decision{g.CreateDesk, scope}
		}
	}
	apply(p.Global, "global")
	apply(p.Runner, "runner")
	for _, e := range p.Entries {
		if t.InstanceID != "" && e.Target == t.Ship() {
			apply(e.Gates, "ship")
		}
	}
	for _, e := range p.Entries {
		if t.Desk != "" && e.Target == t {
			apply(e.Gates, "desk")
		}
	}
	return out
}

// Check describes effects, not command names: Urgit pushes, automatic mount
// commits and composite deployments must request the same gates. Call with fresh
// authoritative publication evidence immediately before each effect.
func (p Policy) Check(t Target, revision uint64, effect, publication string) error {
	if p.Validate() != nil || t.Validate() != nil || t.InstanceID == "" || t.Desk == "" || t.ProjectID != "" && t.Generation == 0 {
		return errors.New("application policy target is unresolved")
	}
	if p.Revision != revision {
		return errors.New("application policy changed; review the current revision")
	}
	e := p.Effective(t)
	switch effect {
	case "install-remote":
		if publication != "absent" {
			return errors.New("receiver desk absence is not confirmed")
		}
		if e.CreateDesk.Value != Allow || e.Commit.Value != Allow {
			return errors.New("receiver installation requires Create Desk and Commit permission")
		}
		return nil
	case "create-desk":
		if publication != "absent" {
			return errors.New("new desk absence is not confirmed")
		}
		if e.CreateDesk.Value != Allow {
			return fmt.Errorf("desk creation blocked by %s policy", e.CreateDesk.Source)
		}
		return nil
	case "commit":
		if e.Commit.Value != Allow {
			return fmt.Errorf("commit blocked by %s policy", e.Commit.Source)
		}
		if publication != "private" && publication != "published" {
			return errors.New("desk publication state is unresolved")
		}
		if publication == "private" {
			return nil
		}
	case "publish":
	default:
		return errors.New("unsupported application effect")
	}
	if e.Publish.Value != Allow {
		return fmt.Errorf("publication blocked by %s policy", e.Publish.Source)
	}
	return nil
}

type Update struct {
	ExpectedRevision uint64 `json:"expected_revision"`
	Scope            string `json:"scope"`
	Target           Target `json:"target"`
	Gates            Gates  `json:"gates"`
}

func (u Update) Validate() error {
	if u.ExpectedRevision >= MaxRevision || u.Target.Validate() != nil || u.Gates.Validate(u.Scope == "global") != nil {
		return errors.New("invalid application policy update")
	}
	switch u.Scope {
	case "global", "runner":
		if u.Target != (Target{}) {
			return errors.New("global and runner updates cannot name a ship")
		}
	case "ship":
		if u.Target.InstanceID == "" || u.Target.Desk != "" {
			return errors.New("ship scope requires a ship")
		}
	case "desk":
		if u.Target.InstanceID == "" || u.Target.Desk == "" {
			return errors.New("desk scope requires a ship and desk")
		}
		if u.Gates.CreateDesk != Inherit {
			return errors.New("desk creation permission must be inherited from the ship")
		}
	default:
		return errors.New("invalid application policy scope")
	}
	return nil
}

// UpgradeLegacy preserves all existing commit/publication choices and closes
// the new gate. Callers must preserve the old audit receipt and fence old writers.
func UpgradeLegacy(p Policy) (Policy, error) {
	if p.Version != 1 || p.Entries == nil || p.Global.CreateDesk != "" || p.Runner.CreateDesk != "" {
		return Policy{}, errors.New("invalid legacy application policy")
	}
	p.Version = Version
	p.Global.CreateDesk = Block
	p.Runner.CreateDesk = Inherit
	p.Entries = append([]Entry{}, p.Entries...)
	for i := range p.Entries {
		if p.Entries[i].Gates.CreateDesk != "" {
			return Policy{}, errors.New("unexpected legacy desk-creation grant")
		}
		p.Entries[i].Gates.CreateDesk = Inherit
	}
	return p, p.Validate()
}
func (p *Policy) Apply(u Update) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if err := u.Validate(); err != nil {
		return err
	}
	if p.Revision != u.ExpectedRevision {
		return errors.New("application policy revision conflict; refresh before saving")
	}
	switch u.Scope {
	case "global":
		p.Global = u.Gates
	case "runner":
		p.Runner = u.Gates
	default:
		entries := make([]Entry, 0, len(p.Entries)+1)
		for _, e := range p.Entries {
			if e.Target != u.Target {
				entries = append(entries, e)
			}
		}
		if u.Gates != Inherited() {
			entries = append(entries, Entry{u.Target, u.Gates})
		}
		if len(entries) > MaxEntries {
			return errors.New("application policy entry limit reached")
		}
		p.Entries = entries
	}
	p.Revision++
	return nil
}

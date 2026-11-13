package deskwork

import (
	"errors"
	"reflect"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
)

// SourceBinding prevents a baseline from silently following a new runtime,
// configuration, project generation or user-selected source workflow.
type SourceBinding struct {
	Target           apppolicy.Target `json:"target"`
	RuntimeID        string           `json:"runtime_id"`
	ConfigRevision   uint64           `json:"config_revision"`
	SettingsRevision uint64           `json:"settings_revision"`
	SourceName       string           `json:"source_name"`
}

func (b SourceBinding) Validate() error {
	if b.Target.Validate() != nil || b.Target.InstanceID == "" || b.Target.Desk == "" || b.Target.ProjectID != "" && b.Target.Generation == 0 || !digestPattern.MatchString(b.RuntimeID) || b.ConfigRevision == 0 || b.ConfigRevision > apppolicy.MaxRevision || b.SettingsRevision > apppolicy.MaxRevision || !ValidSourceName(b.SourceName) {
		return errors.New("invalid source binding")
	}
	return nil
}

type SourceBaseline struct {
	Version    int           `json:"version"`
	Binding    SourceBinding `json:"binding"`
	Manifest   Manifest      `json:"manifest"`
	ObservedAt time.Time     `json:"observed_at"`
}

func canonicalManifest(m Manifest) bool {
	if m.Validate() != nil {
		return false
	}
	for _, f := range m.Files {
		if f.Bytes != 0 {
			return false
		}
	}
	return true
}
func (b SourceBaseline) Validate() error {
	if b.Version != 1 || b.Binding.Validate() != nil || !canonicalManifest(b.Manifest) || b.ObservedAt.IsZero() {
		return errors.New("invalid synchronized source baseline")
	}
	return nil
}

// Validate recomputes classifications from the actual normalized manifests.
// A successful status is evidence of source acceptance/equality, not compilation.
func (s SourceStatus) Validate(baseline *Manifest) error {
	if s.Version != 1 || s.Observation.Validate() != nil || !s.Observation.Exists || s.ReadAt.IsZero() || s.Unix.Validate() != nil || !canonicalManifest(s.CanonicalUnix) || !canonicalManifest(s.Clay) || s.Autocommit != "active" && s.Autocommit != "inactive" && s.Autocommit != "unknown" {
		return errors.New("invalid source status")
	}
	if len(s.Unix.Files) != len(s.CanonicalUnix.Files) {
		return errors.New("canonical source path mismatch")
	}
	for i, f := range s.Unix.Files {
		if f.Path != s.CanonicalUnix.Files[i].Path {
			return errors.New("canonical source path mismatch")
		}
	}
	base := s.Clay
	if baseline != nil {
		if !canonicalManifest(*baseline) {
			return errors.New("invalid canonical baseline")
		}
		base = *baseline
	}
	comparison, err := Compare(base, s.CanonicalUnix, s.Clay)
	if err != nil {
		return err
	}
	if baseline == nil && comparison.State != "synchronized" {
		comparison.State = "different"
	}
	delta, err := Compare(s.Clay, s.CanonicalUnix, s.Clay)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(s.Comparison, comparison) || !reflect.DeepEqual(s.Differences, delta.UnixChanges) {
		return errors.New("source comparison does not match manifests")
	}
	return nil
}

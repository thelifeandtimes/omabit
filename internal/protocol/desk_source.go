package protocol

import (
	"errors"
	"sort"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

const DeskSourceFeature = "desk-source-status-v1"
const DeskSourcePath = "/v1/desk-source-status"
const MaxSourceDetails = 256

type DeskSourceRequest struct {
	Selection        DeskSelection `json:"selection"`
	SettingsRevision uint64        `json:"settings_revision"`
}

func (q DeskSourceRequest) Validate() error {
	if q.Selection.Validate() != nil || q.SettingsRevision > apppolicy.MaxRevision {
		return errors.New("invalid source status selection")
	}
	return nil
}
func (q DeskSourceRequest) Binding(sourceName string) deskwork.SourceBinding {
	return deskwork.SourceBinding{Target: q.Selection.Target, RuntimeID: q.Selection.RuntimeID, ConfigRevision: q.Selection.ConfigRevision, SettingsRevision: q.SettingsRevision, SourceName: sourceName}
}

type SourceDifference struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
}

// Compact view deliberately omits manifests and source bytes. File detail is
// bounded, with the exact total reported even when the view is truncated.
type DeskSourceStatus struct {
	Version             int                  `json:"version"`
	RunnerID            string               `json:"runner_id"`
	Mode                string               `json:"mode"`
	Request             DeskSourceRequest    `json:"request"`
	SourceName          string               `json:"source_name"`
	ReadAt              time.Time            `json:"read_at"`
	BaselineAt          *time.Time           `json:"baseline_at,omitempty"`
	Observation         deskwork.Observation `json:"observation"`
	State               string               `json:"state"`
	Autocommit          string               `json:"autocommit"`
	UnixDigest          string               `json:"unix_digest"`
	CanonicalUnixDigest string               `json:"canonical_unix_digest"`
	ClayDigest          string               `json:"clay_digest"`
	DifferenceCount     int                  `json:"difference_count"`
	UnixChangeCount     int                  `json:"unix_change_count"`
	ClayChangeCount     int                  `json:"clay_change_count"`
	ConflictCount       int                  `json:"conflict_count"`
	Differences         []SourceDifference   `json:"differences"`
}

func NewDeskSourceStatus(runner, mode string, q DeskSourceRequest, source string, s deskwork.SourceStatus, baseline *deskwork.SourceBaseline) DeskSourceStatus {
	out := DeskSourceStatus{Version: 1, RunnerID: runner, Mode: mode, Request: q, SourceName: source, ReadAt: s.ReadAt, Observation: s.Observation, State: s.Comparison.State, Autocommit: s.Autocommit, UnixDigest: s.Unix.Digest, CanonicalUnixDigest: s.CanonicalUnix.Digest, ClayDigest: s.Clay.Digest, DifferenceCount: len(s.Differences), UnixChangeCount: len(s.Comparison.UnixChanges), ClayChangeCount: len(s.Comparison.ClayChanges), ConflictCount: len(s.Comparison.Conflicts), Differences: []SourceDifference{}}
	if baseline != nil {
		at := baseline.ObservedAt
		out.BaselineAt = &at
	}
	unix, clay := s.CanonicalUnix.Index(), s.Clay.Index()
	for _, path := range s.Differences {
		if len(out.Differences) == MaxSourceDetails {
			break
		}
		_, u := unix[path]
		_, c := clay[path]
		kind := "modified"
		if !u {
			kind = "only-clay"
		} else if !c {
			kind = "only-unix"
		}
		out.Differences = append(out.Differences, SourceDifference{Path: path, Kind: kind})
	}
	return out
}
func (s DeskSourceStatus) Validate() error {
	if s.Version != 1 || !ValidID(s.RunnerID) || !ValidID(s.Mode) || s.Request.Validate() != nil || !deskwork.ValidSourceName(s.SourceName) || s.ReadAt.IsZero() || s.Observation.Validate() != nil || !s.Observation.Exists || !ValidDeskOperationID(s.UnixDigest) || !ValidDeskOperationID(s.CanonicalUnixDigest) || !ValidDeskOperationID(s.ClayDigest) {
		return errors.New("invalid source response")
	}
	if s.BaselineAt != nil && (s.BaselineAt.IsZero() || s.BaselineAt.After(s.ReadAt)) {
		return errors.New("invalid source baseline time")
	}
	if s.Autocommit != "active" && s.Autocommit != "inactive" && s.Autocommit != "unknown" {
		return errors.New("invalid autocommit evidence")
	}
	for _, count := range []int{s.DifferenceCount, s.UnixChangeCount, s.ClayChangeCount, s.ConflictCount} {
		if count < 0 || count > deskwork.MaxFiles*2 {
			return errors.New("invalid change count")
		}
	}
	if s.ConflictCount > s.UnixChangeCount || s.ConflictCount > s.ClayChangeCount {
		return errors.New("invalid conflict count")
	}
	switch s.State {
	case "synchronized":
		if s.DifferenceCount != 0 || s.CanonicalUnixDigest != s.ClayDigest || s.UnixChangeCount != s.ClayChangeCount || s.ConflictCount != 0 {
			return errors.New("false synchronized result")
		}
	case "different":
		if s.BaselineAt != nil || s.UnixChangeCount != s.DifferenceCount || s.ClayChangeCount != 0 || s.ConflictCount != 0 {
			return errors.New("ambiguous baseline classification")
		}
	case "unix-changes":
		if s.BaselineAt == nil || s.UnixChangeCount == 0 || s.UnixChangeCount != s.DifferenceCount || s.ClayChangeCount != 0 {
			return errors.New("invalid Unix change classification")
		}
	case "clay-changes":
		if s.BaselineAt == nil || s.ClayChangeCount == 0 || s.ClayChangeCount != s.DifferenceCount || s.UnixChangeCount != 0 {
			return errors.New("invalid Clay change classification")
		}
	case "diverged":
		if s.BaselineAt == nil || s.UnixChangeCount == 0 || s.ClayChangeCount == 0 {
			return errors.New("invalid divergent classification")
		}
	default:
		return errors.New("invalid source state")
	}
	if s.State != "synchronized" && (s.DifferenceCount == 0 || s.CanonicalUnixDigest == s.ClayDigest) {
		return errors.New("false difference result")
	}
	want := s.DifferenceCount
	if want > MaxSourceDetails {
		want = MaxSourceDetails
	}
	if s.Differences == nil || len(s.Differences) != want {
		return errors.New("invalid difference detail count")
	}
	paths := []string{}
	for _, d := range s.Differences {
		if !deskwork.ValidPath(d.Path) || d.Kind != "modified" && d.Kind != "only-clay" && d.Kind != "only-unix" {
			return errors.New("invalid source difference")
		}
		paths = append(paths, d.Path)
	}
	if !sort.StringsAreSorted(paths) {
		return errors.New("unordered source differences")
	}
	for i := 1; i < len(paths); i++ {
		if paths[i] == paths[i-1] {
			return errors.New("duplicate source difference")
		}
	}
	return nil
}

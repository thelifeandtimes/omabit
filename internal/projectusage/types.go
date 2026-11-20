// Package projectusage describes bounded filesystem observations. It grants no
// runtime, retention, policy or resumption authority.
package projectusage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path"
	"reflect"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

const MaxEntries = 65536
const MaxRoots = 65
const MaxDepth = 64
const Scope = "current-member-mounts-and-private-caches-v1"
const ExpandedLedger = "expanded-ledger-v1"
const OriginalHistory = "original-ready-history-v1"

var ErrInvalid = errors.New("invalid project allocation observation or exact selection")

var exclusions = []string{"historical-or-retired-member-storage", "shared-engine-images-and-content", "global-management-journal", "application-source-staging", "unrelated-storage"}

func Exclusions() []string { return append([]string(nil), exclusions...) }

func hash(value string) bool {
	if len(value) != 64 || value != strings.ToLower(value) || value == strings.Repeat("0", 64) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func runtimeID(value string) bool {
	if len(value) != 32 || value != strings.ToLower(value) || value == strings.Repeat("0", 32) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

type Request struct {
	RunnerID           string `json:"runner_id"`
	ProjectID          string `json:"project_id"`
	ExpectedRevision   uint64 `json:"expected_revision"`
	ReviewDigest       string `json:"review_digest"`
	IncludePrivateData bool   `json:"include_private_data"`
}

func (r Request) Validate() error {
	if !protocol.ValidID(r.RunnerID) || !protocol.ValidID(r.ProjectID) || r.ExpectedRevision == 0 || !hash(r.ReviewDigest) || !r.IncludePrivateData {
		return ErrInvalid
	}
	return nil
}

type Binding struct {
	RunnerID     string `json:"runner_id"`
	Mode         string `json:"mode"`
	ProjectID    string `json:"project_id"`
	Revision     uint64 `json:"revision"`
	Generation   uint64 `json:"generation"`
	PlanDigest   string `json:"plan_digest"`
	LedgerKind   string `json:"ledger_kind"`
	LedgerSHA256 string `json:"ledger_sha256"`
}

func (b Binding) Validate() error {
	if !protocol.ValidID(b.RunnerID) || !protocol.ValidID(b.Mode) || !protocol.ValidID(b.ProjectID) || b.Revision == 0 || b.Generation == 0 || !hash(b.PlanDigest) || !hash(b.LedgerSHA256) || b.LedgerKind != ExpandedLedger && b.LedgerKind != OriginalHistory {
		return ErrInvalid
	}
	return nil
}

// Logical bytes and inode blocks are charged once per non-directory inode.
// Directory/special inode blocks count too. Shared extents/compression do not
// provide exclusive physical ownership or a free-space reservation.
type Totals struct {
	RegularFiles     uint64 `json:"regular_files"`
	Directories      uint64 `json:"directories"`
	SpecialEntries   uint64 `json:"special_entries"`
	UniqueFileInodes uint64 `json:"unique_file_inodes"`
	HardlinkEntries  uint64 `json:"hardlink_entries"`
	LogicalBytes     uint64 `json:"logical_bytes"`
	AllocatedBytes   uint64 `json:"allocated_bytes"`
}

func (t Totals) Validate() error {
	if t.RegularFiles > MaxEntries || t.Directories > MaxEntries || t.SpecialEntries > MaxEntries || t.RegularFiles+t.Directories+t.SpecialEntries > MaxEntries || t.UniqueFileInodes > t.RegularFiles+t.SpecialEntries || t.HardlinkEntries > t.RegularFiles+t.SpecialEntries || t.RegularFiles+t.SpecialEntries-t.UniqueFileInodes > t.HardlinkEntries {
		return ErrInvalid
	}
	return nil
}

func (t Totals) Add(other Totals) (Totals, error) {
	if t.Validate() != nil || other.Validate() != nil || t.LogicalBytes > ^uint64(0)-other.LogicalBytes || t.AllocatedBytes > ^uint64(0)-other.AllocatedBytes {
		return Totals{}, ErrInvalid
	}
	out := Totals{t.RegularFiles + other.RegularFiles, t.Directories + other.Directories, t.SpecialEntries + other.SpecialEntries, t.UniqueFileInodes + other.UniqueFileInodes, t.HardlinkEntries + other.HardlinkEntries, t.LogicalBytes + other.LogicalBytes, t.AllocatedBytes + other.AllocatedBytes}
	return out, out.Validate()
}

type Root struct {
	ID         string                      `json:"id"`
	Kind       string                      `json:"kind"`
	InstanceID string                      `json:"instance_id,omitempty"`
	RuntimeID  string                      `json:"runtime_id,omitempty"`
	Path       string                      `json:"path"`
	Expected   pierstore.DirectoryIdentity `json:"expected_identity"`
}

func (r Root) Validate() error {
	if !protocol.ValidID(r.ID) || !strings.HasPrefix(r.Path, "/") || path.Clean(r.Path) != r.Path || r.Path == "/" || len(r.Path) > 2048 || strings.ContainsAny(r.Path, "\x00\r\n") {
		return ErrInvalid
	}
	switch r.Kind {
	case "member-mount":
		if !protocol.ValidID(r.InstanceID) || !runtimeID(r.RuntimeID) || r.Expected.Device == 0 || r.Expected.Inode == 0 || r.Expected.Birth == "" {
			return ErrInvalid
		}
	case "runtime-cache":
		if !protocol.ValidID(r.InstanceID) || !runtimeID(r.RuntimeID) || r.Expected != (pierstore.DirectoryIdentity{}) {
			return ErrInvalid
		}
	case "project-cache":
		if r.InstanceID != "" || r.RuntimeID != "" || r.Expected != (pierstore.DirectoryIdentity{}) {
			return ErrInvalid
		}
	default:
		return ErrInvalid
	}
	return nil
}

type RootUsage struct {
	Root     Root                        `json:"root"`
	Observed pierstore.DirectoryIdentity `json:"observed_identity"`
	Totals   Totals                      `json:"totals"`
}

type ByteDelta struct {
	Direction string `json:"direction"`
	Bytes     uint64 `json:"bytes"`
}

func (d ByteDelta) valid() bool {
	return d.Direction == "unchanged" && d.Bytes == 0 || (d.Direction == "increased" || d.Direction == "decreased") && d.Bytes > 0
}

type Growth struct {
	Status       string     `json:"status"`
	Reason       string     `json:"reason,omitempty"`
	ElapsedNanos int64      `json:"elapsed_nanos"`
	Allocated    *ByteDelta `json:"allocated,omitempty"`
	Logical      *ByteDelta `json:"logical,omitempty"`
	Basis        string     `json:"basis"`
}

func (g Growth) Validate() error {
	if g.Basis != "server-held-censuses-and-monotonic-completion-times; net inode/file change only, not event payload growth or action authority" {
		return ErrInvalid
	}
	if g.Status == "unknown" {
		switch g.Reason {
		case "need-two-samples", "binding-changed", "current-unavailable", "window-too-short", "sample-expired":
		default:
			return ErrInvalid
		}
		if g.ElapsedNanos != 0 || g.Allocated != nil || g.Logical != nil {
			return ErrInvalid
		}
		return nil
	}
	if g.Status != "measured" || g.Reason != "" || g.ElapsedNanos < int64(time.Second) || g.ElapsedNanos > int64(5*time.Minute) || g.Allocated == nil || g.Logical == nil || !g.Allocated.valid() || !g.Logical.valid() {
		return ErrInvalid
	}
	return nil
}

type Observation struct {
	Version                int         `json:"version"`
	Binding                Binding     `json:"binding"`
	Scope                  string      `json:"scope"`
	Status                 string      `json:"status"`
	Reason                 string      `json:"reason,omitempty"`
	UnknownRootID          string      `json:"unknown_root_id,omitempty"`
	DeclaredRoots          []Root      `json:"declared_roots"`
	Roots                  []RootUsage `json:"roots,omitempty"`
	Totals                 *Totals     `json:"totals,omitempty"`
	ExcludedComponents     []string    `json:"excluded_components"`
	SharedAllowanceStatus  string      `json:"shared_allowance_status"`
	FullProjectBudgetKnown bool        `json:"full_project_budget_known"`
	StartedAt              time.Time   `json:"started_at"`
	ReadAt                 time.Time   `json:"read_at"`
	Basis                  string      `json:"basis"`
	Growth                 *Growth     `json:"growth,omitempty"`
}

func (o Observation) Validate() error {
	if o.Version != 1 || o.Binding.Validate() != nil || o.Scope != Scope || o.StartedAt.IsZero() || o.ReadAt.Before(o.StartedAt) || o.Basis == "" || o.SharedAllowanceStatus != "not-configured" || o.FullProjectBudgetKnown || !reflect.DeepEqual(o.ExcludedComponents, exclusions) || len(o.DeclaredRoots) < 1 || len(o.DeclaredRoots) > MaxRoots {
		return ErrInvalid
	}
	seen := map[string]bool{}
	for i, r := range o.DeclaredRoots {
		if r.Validate() != nil || seen[r.ID] {
			return ErrInvalid
		}
		seen[r.ID] = true
		for _, old := range o.DeclaredRoots[:i] {
			if pierstore.Overlap(r.Path, old.Path) {
				return ErrInvalid
			}
		}
	}
	if o.Growth != nil && o.Growth.Validate() != nil {
		return ErrInvalid
	}
	if o.Status == "unknown" {
		if o.Growth != nil && (o.Growth.Status != "unknown" || o.Growth.Reason != "current-unavailable") {
			return ErrInvalid
		}
		switch o.Reason {
		case "changed", "unsafe-storage", "entry-limit", "depth-limit", "unavailable", "cancelled", "overflow", "cross-mount", "external-hardlink":
		default:
			return ErrInvalid
		}
		if o.UnknownRootID != "" && !seen[o.UnknownRootID] || o.Totals != nil || len(o.Roots) != 0 {
			return ErrInvalid
		}
		return nil
	}
	if o.Status != "measured" || o.Reason != "" || o.UnknownRootID != "" || o.Totals == nil || len(o.Roots) != len(o.DeclaredRoots) {
		return ErrInvalid
	}
	var total Totals
	for i, r := range o.Roots {
		if r.Root != o.DeclaredRoots[i] || r.Observed.Device == 0 || r.Observed.Inode == 0 || r.Observed.Birth == "" || r.Totals.Directories == 0 || r.Root.Kind == "member-mount" && !r.Root.Expected.SameObject(r.Observed) {
			return ErrInvalid
		}
		var err error
		total, err = total.Add(r.Totals)
		if err != nil {
			return ErrInvalid
		}
	}
	if total != *o.Totals {
		return ErrInvalid
	}
	return nil
}

func (o Observation) Clone() Observation {
	o.DeclaredRoots = append([]Root(nil), o.DeclaredRoots...)
	o.Roots = append([]RootUsage(nil), o.Roots...)
	o.ExcludedComponents = append([]string(nil), o.ExcludedComponents...)
	if o.Totals != nil {
		t := *o.Totals
		o.Totals = &t
	}
	if o.Growth != nil {
		g := *o.Growth
		if g.Allocated != nil {
			v := *g.Allocated
			g.Allocated = &v
		}
		if g.Logical != nil {
			v := *g.Logical
			g.Logical = &v
		}
		o.Growth = &g
	}
	return o
}

func (o Observation) IdentityDigest() (string, error) {
	if o.Validate() != nil || o.Status != "measured" {
		return "", ErrInvalid
	}
	type rootIdentity struct {
		Root     Root
		Observed pierstore.DirectoryIdentity
	}
	roots := make([]rootIdentity, len(o.Roots))
	for i, r := range o.Roots {
		roots[i] = rootIdentity{r.Root, r.Observed}
	}
	b, _ := json.Marshal(struct {
		Binding Binding
		Scope   string
		Roots   []rootIdentity
	}{o.Binding, o.Scope, roots})
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

func Compare(previous, current Observation, elapsed time.Duration) Growth {
	out := Growth{Status: "unknown", Reason: "current-unavailable", Basis: "server-held-censuses-and-monotonic-completion-times; net inode/file change only, not event payload growth or action authority"}
	if current.Validate() != nil || current.Status != "measured" {
		return out
	}
	if previous.Validate() != nil || previous.Status != "measured" {
		out.Reason = "need-two-samples"
		return out
	}
	a, _ := previous.IdentityDigest()
	b, _ := current.IdentityDigest()
	if a != b {
		out.Reason = "binding-changed"
		return out
	}
	if elapsed < time.Second {
		out.Reason = "window-too-short"
		return out
	}
	if elapsed > 5*time.Minute {
		out.Reason = "sample-expired"
		return out
	}
	delta := func(a, b uint64) *ByteDelta {
		if b > a {
			return &ByteDelta{"increased", b - a}
		}
		if b < a {
			return &ByteDelta{"decreased", a - b}
		}
		return &ByteDelta{"unchanged", 0}
	}
	out.Status, out.Reason, out.ElapsedNanos = "measured", "", int64(elapsed)
	out.Allocated = delta(previous.Totals.AllocatedBytes, current.Totals.AllocatedBytes)
	out.Logical = delta(previous.Totals.LogicalBytes, current.Totals.LogicalBytes)
	return out
}

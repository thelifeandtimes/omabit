// Package epochs describes observed epoch storage and non-executable retention
// previews. It has no pruning or filesystem mutation API.
package epochs

import (
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	Source           = "vere-4.6-epoch-v2"
	ImportSource     = "vere-import-epochs-v1-v2"
	MaxEpochs        = 128
	MaxFilesPerEpoch = 32
	ProtectedSuffix  = 2
	MaxAge           = 5 * time.Minute
)

var digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
var pierPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

type Entry struct {
	LayoutVersion  uint8  `json:"layout_version,omitempty"`
	VereVersion    string `json:"vere_version,omitempty"`
	Name           string `json:"name"`
	Digest         string `json:"digest,omitempty"`
	LogicalBytes   int64  `json:"logical_bytes"`
	AllocatedBytes int64  `json:"allocated_bytes"`
	FileCount      int    `json:"file_count"`
	SnapshotBytes  int64  `json:"snapshot_bytes,omitempty"`
	// SharedFiles reports hardlinks, not filesystem extent/reflink sharing.
	SharedFiles bool `json:"shared_files"`
	Protected   bool `json:"protected"`
}

type Inventory struct {
	Source     string    `json:"source"`
	PierID     string    `json:"pier_id"`
	ObservedAt time.Time `json:"observed_at"`
	Digest     string    `json:"digest"`
	Entries    []Entry   `json:"entries"`
}

type Preview struct {
	RunnerID                    string    `json:"runner_id"`
	InstanceID                  string    `json:"instance_id"`
	ConfigRevision              uint64    `json:"config_revision"`
	InventoryDigest             string    `json:"inventory_digest"`
	ObservedAt                  time.Time `json:"observed_at"`
	Cutoff                      string    `json:"cutoff"`
	Selected                    []Entry   `json:"selected"`
	Retained                    []Entry   `json:"retained"`
	SelectedLogicalBytes        int64     `json:"selected_logical_bytes"`
	SelectedAllocatedBytes      int64     `json:"selected_allocated_bytes"`
	SharedFiles                 bool      `json:"shared_files"`
	DeletionSupported           bool      `json:"deletion_supported"`
	RequiresStoppedRevalidation bool      `json:"requires_stopped_revalidation"`
}

// Epoch names remain strings on the wire to preserve uint64 values in JS clients.
func Number(name string) (uint64, error) {
	if !strings.HasPrefix(name, "0i") || len(name) < 3 || len(name) > 22 {
		return 0, errors.New("require a canonical epoch name such as 0i0 or 0i123")
	}
	n, err := strconv.ParseUint(name[2:], 10, 64)
	if err != nil || strconv.FormatUint(n, 10) != name[2:] {
		return 0, errors.New("require a canonical decimal epoch number")
	}
	return n, nil
}

func ValidDigest(s string) bool { return digestPattern.MatchString(s) }

func Plan(inv *Inventory, cutoff, expectedDigest string, now time.Time) (Preview, error) {
	if inv == nil || inv.Source != Source || !pierPattern.MatchString(inv.PierID) || !ValidDigest(expectedDigest) || inv.Digest != expectedDigest {
		return Preview{}, errors.New("epoch inventory is missing or its digest changed; inspect again")
	}
	if inv.ObservedAt.IsZero() || now.Sub(inv.ObservedAt) > MaxAge || inv.ObservedAt.After(now.Add(5*time.Second)) {
		return Preview{}, errors.New("epoch inventory is stale; request a new inspection")
	}
	if _, err := Number(cutoff); err != nil {
		return Preview{}, err
	}
	if len(inv.Entries) == 0 || len(inv.Entries) > MaxEpochs {
		return Preview{}, errors.New("invalid epoch inventory size")
	}
	index := -1
	var previous uint64
	for i, e := range inv.Entries {
		n, err := Number(e.Name)
		if err != nil || (i > 0 && n <= previous) || e.LogicalBytes < 0 || e.AllocatedBytes < 0 || e.SnapshotBytes < 0 || e.FileCount < 1 || e.FileCount > MaxFilesPerEpoch || e.Protected != (i >= len(inv.Entries)-ProtectedSuffix) {
			return Preview{}, errors.New("invalid epoch ordering, measurement or protected suffix")
		}
		previous = n
		if e.Name == cutoff {
			index = i
		}
	}
	if index < 0 {
		return Preview{}, errors.New("cutoff epoch is absent from the inventory")
	}
	if index >= len(inv.Entries)-ProtectedSuffix {
		return Preview{}, errors.New("cutoff includes a protected newest epoch")
	}
	p := Preview{InventoryDigest: inv.Digest, ObservedAt: inv.ObservedAt, Cutoff: cutoff,
		Selected: append([]Entry(nil), inv.Entries[:index+1]...), Retained: append([]Entry(nil), inv.Entries[index+1:]...), RequiresStoppedRevalidation: true}
	for _, e := range p.Selected {
		if e.LogicalBytes > math.MaxInt64-p.SelectedLogicalBytes || e.AllocatedBytes > math.MaxInt64-p.SelectedAllocatedBytes {
			return Preview{}, errors.New("epoch size sum overflow")
		}
		p.SelectedLogicalBytes += e.LogicalBytes
		p.SelectedAllocatedBytes += e.AllocatedBytes
		p.SharedFiles = p.SharedFiles || e.SharedFiles
	}
	return p, nil
}

// GenesisPlan is the exact deletion scope qualified in checkpoint 6G. It does
// not authorize other prefixes, even if the general preview permits selection.
func GenesisPlan(inv *Inventory, cutoff, digest string, now time.Time) (Preview, error) {
	p, err := Plan(inv, cutoff, digest, now)
	if err != nil {
		return Preview{}, err
	}
	if cutoff != "0i0" || len(inv.Entries) != 4 || len(p.Selected) != 1 || len(p.Retained) != 3 || p.Selected[0].Name != "0i0" || p.Selected[0].FileCount != 4 || p.Selected[0].SnapshotBytes != 0 {
		return Preview{}, errors.New("only genesis epoch 0i0 in a four-epoch inventory is qualified for deletion")
	}
	for i, e := range inv.Entries {
		if e.SharedFiles || !ValidDigest(e.Digest) || (i > 0 && e.SnapshotBytes <= 0) {
			return Preview{}, errors.New("genesis deletion requires exclusive files, complete fingerprints and retained snapshots")
		}
	}
	return p, nil
}

// PrunePlan applies the pinned layout and exclusive-file requirements proven by
// prefix qualification. Selection still includes the cutoff and every older
// epoch, with at least the newest two retained.
func PrunePlan(inv *Inventory, cutoff, digest string, now time.Time) (Preview, error) {
	p, err := Plan(inv, cutoff, digest, now)
	if err != nil {
		return Preview{}, err
	}
	for _, e := range inv.Entries {
		files := 5
		if e.Name == "0i0" {
			files = 4
		}
		if e.SharedFiles || !ValidDigest(e.Digest) || e.FileCount != files || (e.Name != "0i0" && e.SnapshotBytes <= 0) || (e.Name == "0i0" && e.SnapshotBytes != 0) {
			return Preview{}, errors.New("prefix deletion requires complete pinned-layout fingerprints, snapshots and exclusive files")
		}
	}
	return p, nil
}

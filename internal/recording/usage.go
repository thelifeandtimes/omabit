package recording

import "time"

const MaxUsageEntries = 65536
const MaxUsageInstances = 1024 // Keep optional usage inside the existing RPC reply bound.

// StorageTotals counts inode-reported blocks and regular-file lengths. Shared
// extents and filesystem compression mean blocks are not exclusive physical
// ownership. Directories contribute blocks/counts but no logical file bytes.
type StorageTotals struct {
	Files          uint64 `json:"files"`
	Directories    uint64 `json:"directories"`
	LogicalBytes   uint64 `json:"logical_bytes"`
	AllocatedBytes uint64 `json:"allocated_bytes"`
}

func (a StorageTotals) Add(b StorageTotals) (StorageTotals, error) {
	if a.Files > MaxUsageEntries || b.Files > MaxUsageEntries || a.Directories > MaxUsageEntries || b.Directories > MaxUsageEntries || a.Files+a.Directories+b.Files+b.Directories > MaxUsageEntries || a.LogicalBytes > ^uint64(0)-b.LogicalBytes || a.AllocatedBytes > ^uint64(0)-b.AllocatedBytes {
		return StorageTotals{}, ErrSelection
	}
	return StorageTotals{a.Files + b.Files, a.Directories + b.Directories, a.LogicalBytes + b.LogicalBytes, a.AllocatedBytes + b.AllocatedBytes}, nil
}

type InstanceStorageUsage struct {
	InstanceID string        `json:"instance_id"`
	Totals     StorageTotals `json:"totals"`
}

// StorageUsage is a bounded non-atomic stat census, separate from admission
// commitments. Unknown results carry no partial totals. Older readers may omit
// this optional evidence entirely; omission never means zero usage.
type StorageUsage struct {
	Version   int                    `json:"version"`
	RunnerID  string                 `json:"runner_id"`
	Status    string                 `json:"status"`
	Reason    string                 `json:"reason,omitempty"`
	Totals    *StorageTotals         `json:"totals,omitempty"`
	Shared    *StorageTotals         `json:"shared,omitempty"`
	Instances []InstanceStorageUsage `json:"instances,omitempty"`
	StartedAt time.Time              `json:"started_at"`
	ReadAt    time.Time              `json:"read_at"`
	Basis     string                 `json:"basis"`
}

func (u StorageUsage) Validate(runner string, instances []InstanceCommitment) error {
	if u.Version != 1 || u.RunnerID != runner || u.StartedAt.IsZero() || u.ReadAt.Before(u.StartedAt) || u.Basis == "" {
		return ErrSelection
	}
	if u.Status == "unknown" {
		switch u.Reason {
		case "changed", "unsafe-storage", "entry-limit", "unavailable", "cancelled", "overflow":
		default:
			return ErrSelection
		}
		if u.Totals != nil || u.Shared != nil || len(u.Instances) != 0 {
			return ErrSelection
		}
		return nil
	}
	if u.Status != "measured" || u.Reason != "" || u.Totals == nil || u.Shared == nil || len(u.Instances) != len(instances) || len(u.Instances) > MaxUsageInstances {
		return ErrSelection
	}
	sum, err := (StorageTotals{}).Add(*u.Shared)
	if err != nil || u.Shared.Directories != 2 {
		return ErrSelection
	}
	for i, item := range u.Instances {
		if item.InstanceID != instances[i].InstanceID {
			return ErrSelection
		}
		sum, err = sum.Add(item.Totals)
		if err != nil {
			return ErrSelection
		}
	}
	if sum != *u.Totals {
		return ErrSelection
	}
	return nil
}

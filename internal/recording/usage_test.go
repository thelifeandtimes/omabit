package recording

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"
)

func TestStorageUsageKeepsMeasurementSeparateAndRejectsPartialOrForeignTotals(t *testing.T) {
	at := time.Now().UTC()
	capacity := Capacity{Version: 1, RunnerID: "runner", Jobs: 1, CommittedBytes: 64 << 20, Instances: []InstanceCommitment{{InstanceID: "one", Jobs: 1, CommittedBytes: 64 << 20}}, ReadAt: at}
	shared := StorageTotals{Directories: 2, Files: 1}
	ship := StorageTotals{Directories: 2, Files: 3, LogicalBytes: 4 << 20, AllocatedBytes: 16384}
	total, err := shared.Add(ship)
	if err != nil {
		t.Fatal(err)
	}
	base := StorageUsage{Version: 1, RunnerID: "runner", Status: "measured", Shared: &shared, Totals: &total, Instances: []InstanceStorageUsage{{InstanceID: "one", Totals: ship}}, StartedAt: at, ReadAt: at, Basis: "private recording roots only; non-atomic inode blocks"}
	if capacity.Validate() != nil {
		t.Fatal("legacy omitted measurement refused")
	}
	for _, name := range []string{"measured", "unknown", "foreign-runner", "foreign-instance", "bad-total", "partial-unknown", "unknown-reason", "reversed-clock"} {
		t.Run(name, func(t *testing.T) {
			u := base
			u.Instances = append([]InstanceStorageUsage(nil), base.Instances...)
			switch name {
			case "unknown", "partial-unknown", "unknown-reason":
				u.Status, u.Reason = "unknown", "unsafe-storage"
				u.Totals, u.Shared, u.Instances = nil, nil, nil
				if name == "partial-unknown" {
					u.Totals = &total
				}
				if name == "unknown-reason" {
					u.Reason = "private path or arbitrary diagnostic"
				}
			case "foreign-runner":
				u.RunnerID = "other"
			case "foreign-instance":
				u.Instances[0].InstanceID = "other"
			case "bad-total":
				changed := total
				changed.LogicalBytes++
				u.Totals = &changed
			case "reversed-clock":
				u.ReadAt = at.Add(-time.Second)
			}
			c := capacity
			c.Usage = &u
			valid := name == "measured" || name == "unknown"
			if (c.Validate() == nil) != valid {
				t.Fatal("invalid usage accepted or valid evidence lost", name)
			}
			if c.CommittedBytes != 64<<20 {
				t.Fatal("usage changed admission commitment")
			}
		})
	}
	if _, err := (StorageTotals{LogicalBytes: math.MaxUint64}).Add(StorageTotals{LogicalBytes: 1}); err == nil {
		t.Fatal("logical byte overflow accepted")
	}
	if _, err := (StorageTotals{AllocatedBytes: math.MaxUint64}).Add(StorageTotals{AllocatedBytes: 1}); err == nil {
		t.Fatal("allocated byte overflow accepted")
	}
	if _, err := (StorageTotals{Files: MaxUsageEntries}).Add(StorageTotals{Directories: 1}); err == nil {
		t.Fatal("unbounded census accepted")
	}
}

func TestStorageUsageLargestDetailedReplyStaysWithinExistingReadLimit(t *testing.T) {
	at := time.Now().UTC()
	shared := StorageTotals{Directories: 2}
	u := StorageUsage{Version: 1, RunnerID: "runner", Status: "measured", Shared: &shared, StartedAt: at, ReadAt: at, Basis: "bounded private recording roots"}
	c := Capacity{Version: 1, RunnerID: "runner", Instances: []InstanceCommitment{}, ReadAt: at, Usage: &u}
	total := shared
	for i := 0; i < MaxUsageInstances; i++ {
		id := strings.Repeat("a", 59) + fmt.Sprintf("%05d", i)
		c.Instances = append(c.Instances, InstanceCommitment{InstanceID: id, Jobs: 1, CommittedBytes: 1 << 30})
		c.Jobs++
		c.CommittedBytes += 1 << 30
		ship := StorageTotals{Directories: 2, Files: 6, LogicalBytes: 1 << 30, AllocatedBytes: 1 << 30}
		var err error
		total, err = total.Add(ship)
		if err != nil {
			t.Fatal(err)
		}
		u.Instances = append(u.Instances, InstanceStorageUsage{InstanceID: id, Totals: ship})
	}
	u.Totals = &total
	if c.Validate() != nil {
		t.Fatal("valid detailed boundary refused")
	}
	raw, err := json.Marshal(c)
	if err != nil || len(raw) > 768<<10 {
		t.Fatal("usage exceeds existing1MiB RPC headroom", len(raw), err)
	}
}

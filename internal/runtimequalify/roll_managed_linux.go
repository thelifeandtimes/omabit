//go:build linux && amd64

package runtimequalify

import "git.thechristophgroup.org/thelifeandtimes/omabit/internal/epochs"

func epochNames(inv *epochs.Inventory) []string {
	names := make([]string, 0, len(inv.Entries))
	for _, e := range inv.Entries {
		names = append(names, e.Name)
	}
	return names
}

// This is the qualified creation contract, not inference from a zero exit code.
// A no-growth/ambiguous result remains quarantined rather than silently retried.
func managedRollGrowth(before, after *epochs.Inventory) bool {
	if before == nil || after == nil || before.Source != epochs.Source || after.Source != before.Source || after.PierID != before.PierID || len(before.Entries) == 0 || len(after.Entries) != len(before.Entries)+1 || len(after.Entries) > epochs.MaxEpochs || !epochPrefix(before, after) {
		return false
	}
	previous, err1 := epochs.Number(before.Entries[len(before.Entries)-1].Name)
	latest := after.Entries[len(after.Entries)-1]
	next, err2 := epochs.Number(latest.Name)
	return err1 == nil && err2 == nil && next > previous && latest.SnapshotBytes > 0
}

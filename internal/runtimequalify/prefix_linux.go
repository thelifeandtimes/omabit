//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/epochs"
)

func isPrefixRecovery(kind string) bool {
	return kind == "prefix-recovery-1" || kind == "prefix-recovery-2"
}

func prefixProfile(profile string) bool {
	return profile == ProfilePrefix || profile == ProfileCometPrefix
}
func isPrefixContainer(kind string) bool {
	return isPrefixRecovery(kind) || kind == "prefix-restart-1" || kind == "prefix-restart-2" || kind == "prefix-recovered-1" || kind == "prefix-recovered-2"
}

func prefixShape(round int) (before, selected, after int) {
	if round == 1 {
		return 5, 2, 3
	}
	if round == 2 {
		return 3, 1, 2
	}
	return 0, 0, 0
}

func sameRetainedPrefix(before, after *epochs.Inventory, selected int) bool {
	if before == nil || after == nil || selected < 1 || len(before.Entries)-selected != len(after.Entries) || len(after.Entries) < epochs.ProtectedSuffix {
		return false
	}
	for i, e := range after.Entries {
		if !epochs.ValidDigest(e.Digest) || !reflect.DeepEqual(before.Entries[i+selected], e) {
			return false
		}
	}
	return true
}

func selectedEpochFile(epoch, name string) bool {
	return genesisFile(name) || (epoch != "0i0" && name == "image.bin")
}

func (q *qualifier) prepareEpochDeletion(log *os.Root, entry epochs.Entry) (EpochDeletion, error) {
	d := EpochDeletion{Name: entry.Name, Files: []PruneFile{}, RemovedFiles: []string{}}
	var err error
	d.Directory, err = pruneFile(log, entry.Name, true)
	if err != nil {
		return d, err
	}
	target, err := log.OpenRoot(entry.Name)
	if err != nil {
		return d, errors.New("selected epoch unavailable")
	}
	defer target.Close()
	actual, err := pruneFile(target, ".", true)
	if err != nil || actual.Device != d.Directory.Device || actual.Inode != d.Directory.Inode {
		return d, errors.New("selected epoch identity changed")
	}
	entries, err := boundedEntries(target, epochs.MaxFilesPerEpoch)
	if err != nil {
		return d, err
	}
	for _, e := range entries {
		if !selectedEpochFile(entry.Name, e.Name()) {
			return d, errors.New("unknown file in selected epoch")
		}
		f, err := pruneFile(target, e.Name(), false)
		if err != nil {
			return d, err
		}
		if f.Device != d.Directory.Device {
			return d, errors.New("cross-device epoch file")
		}
		d.Files = append(d.Files, f)
	}
	expected := 5
	if entry.Name == "0i0" {
		expected = 4
	}
	if len(d.Files) != expected || len(d.Files) != entry.FileCount {
		return d, errors.New("selected epoch file set is incomplete")
	}
	return d, nil
}

func (q *qualifier) unlinkEpochTarget(ctx context.Context, log *os.Root, d *EpochDeletion) error {
	if err := q.stoppedPruneBoundary(ctx); err != nil {
		return err
	}
	current, err := pruneFile(log, d.Name, true)
	if err != nil || current != d.Directory {
		return errors.New("selected epoch changed before unlink")
	}
	target, err := log.OpenRoot(d.Name)
	if err != nil {
		return errors.New("selected epoch unavailable")
	}
	defer target.Close()
	actual, err := pruneFile(target, ".", true)
	if err != nil || actual.Device != d.Directory.Device || actual.Inode != d.Directory.Inode {
		return errors.New("selected epoch identity changed")
	}
	for _, file := range d.Files {
		if err := q.stoppedPruneBoundary(ctx); err != nil {
			return err
		}
		current, err := pruneFile(target, file.Name, false)
		if err != nil || current != file {
			return errors.New("selected file changed before unlink")
		}
		if err := q.d.remove(target, file.Name); err != nil {
			return errors.New("prefix file unlink failed; retain for review")
		}
		if err := syncRoot(target); err != nil {
			return err
		}
		d.RemovedFiles = append(d.RemovedFiles, file.Name)
		if err := q.save(); err != nil {
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := boundedEntries(target, 0); err != nil {
		return errors.New("unexpected file appeared in selected epoch")
	}
	current, err = pruneFile(log, d.Name, true)
	if err != nil || current.Device != d.Directory.Device || current.Inode != d.Directory.Inode {
		return errors.New("epoch changed before directory removal")
	}
	if err := q.d.remove(log, d.Name); err != nil {
		return errors.New("prefix directory removal failed")
	}
	if err := syncRoot(log); err != nil {
		return err
	}
	d.Removed = true
	return q.save()
}

func (q *qualifier) pruneWiderPrefix(ctx context.Context, round int) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if !prefixProfile(q.d.profile) || round < 1 || round > 2 || len(q.r.PrefixPrunes) != round-1 {
		return errors.New("invalid disposable prefix sequence")
	}
	if err := q.pruneBoundary(ctx); err != nil {
		return err
	}
	e := q.r.EpochLifecycle
	if e == nil || len(e.Rolls) != 4 || e.Final == nil || e.Preview == nil {
		return errors.New("prefix deletion requires completed five-epoch preparation")
	}
	expected := e.Final
	if round == 2 {
		previous := q.r.PrefixPrunes[0]
		if previous == nil || previous.Uncertain || previous.Recovery == nil || previous.Recovery.Uncertain || previous.Recovery.PostRestart == nil {
			return errors.New("prior prefix and recovery must be resolved")
		}
		expected = previous.Recovery.PostRestart
	}
	count, selected, remaining := prefixShape(round)
	before, err := q.inspectEpochs(ctx)
	if err != nil {
		return err
	}
	if validEpochSample(before, q.r.RunID, count) != nil {
		return errors.New("unexpected prefix inventory shape")
	}
	p, err := epochs.Plan(before, before.Entries[selected-1].Name, expected.Digest, time.Now().UTC())
	if err != nil {
		return err
	}
	if len(p.Selected) != selected || len(p.Retained) != remaining {
		return errors.New("prefix selection changed")
	}
	return q.deletePrefix(ctx, p.Cutoff, expected.Digest, func(r *PrefixPruneRecord) { q.r.PrefixPrunes = append(q.r.PrefixPrunes, r) })
}

// Shared qualified unlink engine. The caller binds/persists this complete record
// before the first mutation; qualification and managed callers use separate roots.
func (q *qualifier) deletePrefix(ctx context.Context, cutoff, digest string, install func(*PrefixPruneRecord)) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if install == nil {
		return errors.New("prefix deletion requires a durable record owner")
	}
	if err := q.stoppedPruneBoundary(ctx); err != nil {
		return err
	}
	before, err := q.inspectEpochs(ctx)
	if err != nil {
		return err
	}
	p, err := epochs.PrunePlan(before, cutoff, digest, time.Now().UTC())
	if err != nil {
		return err
	}
	log, err := q.openPierDirectory("log")
	if err != nil {
		return err
	}
	defer log.Close()
	targets := []EpochDeletion{}
	for _, entry := range p.Selected {
		target, err := q.prepareEpochDeletion(log, entry)
		if err != nil {
			return err
		}
		targets = append(targets, target)
	}
	revalidated, err := q.inspectEpochs(ctx)
	if err != nil {
		return err
	}
	if revalidated.Digest != before.Digest {
		return errors.New("prefix inventory changed; no deletion dispatched")
	}
	r := &PrefixPruneRecord{StartedAt: time.Now().UTC(), Before: before, Cutoff: p.Cutoff, Targets: targets, Uncertain: true}
	install(r)
	if err := q.save(); err != nil {
		return err
	}
	for i := range r.Targets {
		if err := q.unlinkEpochTarget(ctx, log, &r.Targets[i]); err != nil {
			return err
		}
	}
	after, err := q.inspectEpochs(ctx)
	if err != nil {
		return err
	}
	r.After = after
	if !sameRetainedPrefix(before, after, len(p.Selected)) {
		return errors.New("retained prefix metadata changed")
	}
	now := time.Now().UTC()
	r.FinishedAt, r.Uncertain = &now, false
	if err := q.save(); err != nil {
		r.FinishedAt, r.Uncertain = nil, true
		return err
	}
	return nil
}

func (q *qualifier) prefixRecord(round int) (*PrefixPruneRecord, error) {
	if !prefixProfile(q.d.profile) || round < 1 || round > 2 || len(q.r.PrefixPrunes) != round {
		return nil, errors.New("invalid prefix qualification step")
	}
	r := q.r.PrefixPrunes[round-1]
	if r == nil || r.Uncertain || r.FinishedAt == nil || r.After == nil {
		return nil, errors.New("prefix deletion remains unresolved")
	}
	return r, nil
}

func (q *qualifier) restartWithEpochs(ctx context.Context, expected *epochs.Inventory, kind string) (*epochs.Inventory, error) {
	if err := q.pruneBoundary(ctx); err != nil {
		return nil, err
	}
	if expected == nil {
		return nil, errors.New("restart requires an epoch inventory")
	}
	if _, _, err := q.epochAdmission(); err != nil {
		return nil, err
	}
	boot := q.boot
	if q.d.profile == ProfileCometPrefix {
		boot = q.cometRuntime
	}
	if err := boot(ctx, epochContainerKind(q.d.profile, kind)); err != nil {
		return nil, err
	}
	inv, err := q.inspectEpochs(ctx)
	if err != nil {
		return nil, err
	}
	if validEpochSample(inv, q.r.RunID, len(expected.Entries)) != nil || !sameEpochNames(expected, inv) {
		return nil, errors.New("retained epoch set changed across restart")
	}
	return inv, nil
}

func (q *qualifier) restartPrefix(ctx context.Context, round int) error {
	r, err := q.prefixRecord(round)
	if err != nil {
		return err
	}
	if r.PostRestart != nil {
		return errors.New("prefix restart cannot repeat")
	}
	inv, err := q.restartWithEpochs(ctx, r.After, fmt.Sprintf("prefix-restart-%d", round))
	if err != nil {
		return err
	}
	r.PostRestart = inv
	if err := q.save(); err != nil {
		r.PostRestart = nil
		return err
	}
	return nil
}

func (q *qualifier) recoverPrefix(ctx context.Context, round int) error {
	r, err := q.prefixRecord(round)
	if err != nil {
		return err
	}
	if r.PostRestart == nil {
		return errors.New("prefix recovery requires ordinary restart")
	}
	return q.recoverLatest(ctx, r.PostRestart, epochContainerKind(q.d.profile, fmt.Sprintf("prefix-recovery-%d", round)), &r.Recovery)
}

func (q *qualifier) restartRecoveredPrefix(ctx context.Context, round int) error {
	r, err := q.prefixRecord(round)
	if err != nil {
		return err
	}
	if r.Recovery == nil || r.Recovery.Uncertain || r.Recovery.FinishedAt == nil || r.Recovery.PostRestart != nil {
		return errors.New("latest-epoch recovery is unresolved or already restarted")
	}
	inv, err := q.restartWithEpochs(ctx, r.Recovery.After, fmt.Sprintf("prefix-recovered-%d", round))
	if err != nil {
		return err
	}
	r.Recovery.PostRestart = inv
	if err := q.save(); err != nil {
		r.Recovery.PostRestart = nil
		return err
	}
	return nil
}

func validPrefixRecord(p *PrefixPruneRecord, previous *epochs.Inventory, id string, round int) error {
	count, selected, remaining := prefixShape(round)
	if p == nil || count == 0 || p.StartedAt.IsZero() || p.FinishedAt == nil || p.FinishedAt.Before(p.StartedAt) || validEpochSample(p.Before, id, count) != nil || previous == nil || p.Before.Digest != previous.Digest || p.Cutoff != p.Before.Entries[selected-1].Name || len(p.Targets) != selected || validEpochSample(p.After, id, remaining) != nil || !sameRetainedPrefix(p.Before, p.After, selected) {
		return errors.New("invalid completed prefix evidence")
	}
	for i, target := range p.Targets {
		files := 5
		if target.Name == "0i0" {
			files = 4
		}
		if target.Name != p.Before.Entries[i].Name || target.Directory.Name != target.Name || target.Directory.Inode == 0 || !target.Removed || len(target.Files) != files || len(target.RemovedFiles) != files {
			return errors.New("invalid prefix target manifest")
		}
		seen := map[string]bool{}
		for j, f := range target.Files {
			if !selectedEpochFile(target.Name, f.Name) || seen[f.Name] || f.Inode == 0 || f.Device != target.Directory.Device || f.Size < 0 || target.RemovedFiles[j] != f.Name {
				return errors.New("invalid prefix file progress")
			}
			seen[f.Name] = true
		}
	}
	if p.PostRestart != nil && (validEpochSample(p.PostRestart, id, remaining) != nil || !sameEpochNames(p.After, p.PostRestart)) {
		return errors.New("invalid prefix restart evidence")
	}
	return nil
}

//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/epochs"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

// This destructive path is deliberately confined to a newly generated
// qualification run. There is no managed action or caller-supplied pier path.
func (q *qualifier) pruneBoundary(ctx context.Context) error {
	base := filepath.Join(q.d.home, ".local/share/urbit-runner/qualification")
	if q.d.managed || (q.d.profile != ProfilePrune && !prefixProfile(q.d.profile)) || q.r.Profile != q.d.profile || !runIDPattern.MatchString(q.r.RunID) || q.base != base || q.r.Directory != filepath.Join(base, q.r.RunID) || q.r.ReportPath != filepath.Join(q.r.Directory, "report.json") || !q.r.LimitsObserved {
		return errors.New("destructive qualification requires its own fresh disposable run")
	}
	if q.d.profile == ProfileCometPrefix && (q.r.Comet == nil || !q.r.Comet.BootCompleted || !q.r.Comet.Uncertain || q.r.Comet.ResumeCompleted || !cometName.MatchString(q.r.Comet.Ship)) {
		return errors.New("comet prefix work requires its owned fresh identity and unfinished qualification")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, c := range q.r.Containers {
		if !c.Removed || c.CreateUncertain {
			return errors.New("destructive qualification requires all owned containers removed")
		}
	}
	return q.boundary()
}

func pruneFile(root *os.Root, name string, directory bool) (PruneFile, error) {
	info, err := root.Lstat(name)
	if err != nil {
		return PruneFile{}, errors.New("deletion target metadata unavailable")
	}
	kind := os.FileMode(0)
	if directory {
		kind = os.ModeDir
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || info.Mode().Type() != kind || info.Size() < 0 || (!directory && st.Nlink != 1) {
		return PruneFile{}, errors.New("deletion target must have the expected type and exclusive file links")
	}
	return PruneFile{Name: name, Device: st.Dev, Inode: st.Ino, Size: info.Size(), Modified: info.ModTime().UnixNano(), Changed: st.Ctim.Nano()}, nil
}

func syncRoot(root *os.Root) error {
	f, err := root.Open(".")
	if err != nil {
		return errors.New("cannot open directory for durable deletion")
	}
	err = errors.Join(f.Sync(), f.Close())
	if err != nil {
		return errors.New("cannot persist directory deletion")
	}
	return nil
}

func sameRetainedEntries(before, after *epochs.Inventory) bool {
	if before == nil || after == nil || len(before.Entries) != 4 || len(after.Entries) != 3 {
		return false
	}
	for i, e := range after.Entries {
		if !epochs.ValidDigest(e.Digest) || !reflect.DeepEqual(before.Entries[i+1], e) {
			return false
		}
	}
	return true
}

func (q *qualifier) prunePrefix(ctx context.Context) error {
	if err := q.pruneBoundary(ctx); err != nil {
		return err
	}
	e := q.r.EpochLifecycle
	if e == nil || len(e.Rolls) != 3 || e.Preview == nil || e.Final == nil || e.Rolls[2].PostRestart == nil || e.Rolls[2].Uncertain {
		return errors.New("pruning requires the completed multi-epoch preview")
	}
	if e.Preview.InventoryDigest != e.Final.Digest {
		return errors.New("qualification preview changed")
	}
	return q.deleteGenesis(ctx, e.Final.Digest)
}

// The registered-pier branch is reachable only through managed admission and a
// durable, explicitly confirmed request. Qualification still uses its own guard.
func (q *qualifier) stoppedPruneBoundary(ctx context.Context) error {
	if !q.d.managed {
		if err := q.pruneBoundary(ctx); err != nil {
			return err
		}
		return q.pruneLockBoundary()
	}
	base := filepath.Join(q.d.home, ".local/share/urbit-runner/managed")
	if (q.d.profile != ProfileHTTP && q.d.profile != ProfileComet) || q.r.Profile != q.d.profile || !runIDPattern.MatchString(q.r.RunID) || q.base != base || q.r.Directory != filepath.Join(base, q.r.RunID) {
		return errors.New("prefix deletion requires its registered managed pier")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, c := range q.r.Containers {
		if !c.Removed || c.CreateUncertain {
			return errors.New("prefix deletion requires every owned container removed")
		}
	}
	if err := q.boundary(); err != nil {
		return err
	}
	return q.pruneLockBoundary()
}

func (q *qualifier) pruneLockBoundary() error {
	if q.d.managed && q.d.storage == nil {
		return errors.New("prefix deletion requires recorded storage")
	}
	if pierstore.Busy(q.pierPath()) {
		return errors.New("prefix deletion requires a pier without recognized Vere locks")
	}
	return nil
}

// Shared exact unlink engine. No arbitrary path, recursion or broader cutoff.
func (q *qualifier) deleteGenesis(ctx context.Context, expectedDigest string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := q.stoppedPruneBoundary(ctx); err != nil {
		return err
	}
	if q.r.Prune != nil || q.r.Recovery != nil {
		return errors.New("genesis deletion cannot be retried or applied twice")
	}
	before, err := q.inspectEpochs(ctx)
	if err != nil {
		return err
	}
	_, err = epochs.GenesisPlan(before, "0i0", expectedDigest, time.Now().UTC())
	if err != nil {
		return err
	}
	if validEpochSample(before, q.r.RunID, 4) != nil {
		return errors.New("qualified oldest-only selection changed")
	}
	for _, entry := range before.Entries {
		if entry.SharedFiles || !epochs.ValidDigest(entry.Digest) {
			return errors.New("prune qualification requires exclusive files and complete epoch fingerprints")
		}
	}
	log, err := q.openPierDirectory("log")
	if err != nil {
		return err
	}
	defer log.Close()
	directory, err := pruneFile(log, "0i0", true)
	if err != nil {
		return err
	}
	target, err := log.OpenRoot("0i0")
	if err != nil {
		return errors.New("selected epoch unavailable")
	}
	defer target.Close()
	actual, err := pruneFile(target, ".", true)
	if err != nil || actual.Device != directory.Device || actual.Inode != directory.Inode {
		return errors.New("selected epoch identity changed")
	}
	entries, err := boundedEntries(target, epochs.MaxFilesPerEpoch)
	if err != nil {
		return err
	}
	files := []PruneFile{}
	for _, entry := range entries {
		if !genesisFile(entry.Name()) {
			return errors.New("unrecognized file in selected genesis epoch")
		}
		file, err := pruneFile(target, entry.Name(), false)
		if err != nil {
			return err
		}
		if file.Device != directory.Device {
			return errors.New("cross-device deletion target")
		}
		files = append(files, file)
	}
	if len(files) != 4 || len(files) != before.Entries[0].FileCount {
		return errors.New("selected epoch file set changed")
	}
	revalidated, err := q.inspectEpochs(ctx)
	if err != nil {
		return err
	}
	if revalidated.Digest != before.Digest {
		return errors.New("stale prune inventory; no deletion dispatched")
	}
	r := &PruneRecord{StartedAt: time.Now().UTC(), Before: before, Cutoff: "0i0", Directory: directory, Files: files, RemovedFiles: []string{}, Uncertain: true}
	q.r.Prune = r
	if err := q.save(); err != nil {
		return err
	}
	for _, file := range files {
		if err := q.stoppedPruneBoundary(ctx); err != nil {
			return err
		}
		current, err := pruneFile(target, file.Name, false)
		if err != nil || current != file {
			return errors.New("selected file changed before unlink; retain for review")
		}
		if err := q.d.remove(target, file.Name); err != nil {
			return errors.New("selected file deletion failed; retain for review")
		}
		if err := syncRoot(target); err != nil {
			return err
		}
		r.RemovedFiles = append(r.RemovedFiles, file.Name)
		if err := q.save(); err != nil {
			return err
		}
	}
	if err := q.stoppedPruneBoundary(ctx); err != nil {
		return err
	}
	if _, err := boundedEntries(target, 0); err != nil {
		return errors.New("unexpected file appeared in selected epoch")
	}
	current, err := pruneFile(log, "0i0", true)
	if err != nil || current.Device != directory.Device || current.Inode != directory.Inode {
		return errors.New("selected epoch changed before directory removal")
	}
	if err := q.d.remove(log, "0i0"); err != nil {
		return errors.New("selected epoch directory removal failed")
	}
	if err := syncRoot(log); err != nil {
		return err
	}
	r.EpochRemoved = true
	if err := q.save(); err != nil {
		return err
	}
	after, err := q.inspectEpochs(ctx)
	if err != nil {
		return err
	}
	r.After = after
	if validEpochSample(after, q.r.RunID, 3) != nil || !sameRetainedEntries(before, after) {
		return errors.New("post-prune retained epoch evidence changed")
	}
	now := time.Now().UTC()
	r.FinishedAt, r.Uncertain = &now, false
	if err := q.save(); err != nil {
		r.FinishedAt, r.Uncertain = nil, true
		return err
	}
	return nil
}

func (q *qualifier) postPrune(ctx context.Context) error {
	if err := q.pruneBoundary(ctx); err != nil {
		return err
	}
	p := q.r.Prune
	if p == nil || p.Uncertain || p.FinishedAt == nil || p.After == nil || p.PostRestart != nil {
		return errors.New("post-prune restart requires a resolved deletion")
	}
	if _, _, err := q.epochAdmission(); err != nil {
		return err
	}
	if err := q.boot(ctx, "post-prune"); err != nil {
		return err
	}
	inv, err := q.inspectEpochs(ctx)
	if err != nil {
		return err
	}
	if validEpochSample(inv, q.r.RunID, 3) != nil || !sameEpochNames(p.After, inv) {
		return errors.New("retained epoch set changed during post-prune restart")
	}
	p.PostRestart = inv
	if err := q.save(); err != nil {
		p.PostRestart = nil
		return err
	}
	return nil
}

func (q *qualifier) recoverEpoch(ctx context.Context) error {
	if err := q.pruneBoundary(ctx); err != nil {
		return err
	}
	p := q.r.Prune
	if p == nil || p.Uncertain || p.PostRestart == nil || q.r.Recovery != nil {
		return errors.New("checkpoint recovery requires successful post-prune restart")
	}
	return q.recoverLatest(ctx, p.PostRestart, "epoch-recovery", &q.r.Recovery)
}

func (q *qualifier) recoverLatest(ctx context.Context, expected *epochs.Inventory, kind string, slot **EpochRecoveryRecord) error {
	if err := q.pruneBoundary(ctx); err != nil {
		return err
	}
	allowed := q.d.profile == ProfilePrune && kind == "epoch-recovery" || q.d.profile == ProfilePrefix && isPrefixRecovery(kind) || q.d.profile == ProfileCometPrefix && isCometPrefixRecovery(kind)
	if !allowed || expected == nil || slot == nil || *slot != nil {
		return errors.New("invalid recovery intent")
	}
	before, err := q.inspectEpochs(ctx)
	if err != nil {
		return err
	}
	if validEpochSample(before, q.r.RunID, len(expected.Entries)) != nil || !sameEpochNames(expected, before) {
		return errors.New("recovery epoch set changed")
	}
	mem, disk, err := q.epochAdmission()
	if err != nil {
		return err
	}
	chk, err := q.openPierDirectory("chk")
	if err != nil {
		return err
	}
	defer chk.Close()
	if err := q.owned(filepath.Join(q.pierPath(), ".urb/chk/image.bin"), 0, false); err != nil {
		return err
	}
	file, err := pruneFile(chk, "image.bin", false)
	if err != nil || file.Size <= 0 {
		return errors.New("current checkpoint missing or unsafe before recovery test")
	}
	r := &EpochRecoveryRecord{StartedAt: time.Now().UTC(), Before: before, Checkpoint: file, MemAvailableBytes: mem, DiskAvailableBytes: disk, Uncertain: true}
	*slot = r
	if err := q.save(); err != nil {
		return err
	}
	if err := q.pruneBoundary(ctx); err != nil {
		return err
	}
	actual, err := pruneFile(chk, "image.bin", false)
	if err != nil || actual != file {
		return errors.New("checkpoint changed before controlled removal")
	}
	if err := q.d.remove(chk, "image.bin"); err != nil {
		return errors.New("controlled checkpoint removal failed")
	}
	if err := syncRoot(chk); err != nil {
		return err
	}
	if _, err := chk.Lstat("image.bin"); !errors.Is(err, os.ErrNotExist) {
		return errors.New("checkpoint removal not confirmed")
	}
	r.CheckpointRemoved = true
	if err := q.save(); err != nil {
		return err
	}
	if err := q.offlineUtility(ctx, kind); err != nil {
		return err
	}
	restored, err := pruneFile(chk, "image.bin", false)
	if err != nil || restored.Size <= 0 {
		return errors.New("latest-epoch replay did not restore the checkpoint")
	}
	after, err := q.inspectEpochs(ctx)
	if err != nil {
		return err
	}
	if validEpochSample(after, q.r.RunID, len(before.Entries)) != nil || !sameEpochNames(before, after) {
		return errors.New("epoch set changed during latest-epoch replay")
	}
	now := time.Now().UTC()
	r.After, r.RecoveredCheckpointBytes, r.FinishedAt, r.Uncertain = after, restored.Size, &now, false
	if err := q.save(); err != nil {
		r.FinishedAt, r.Uncertain = nil, true
		return err
	}
	return nil
}

func (q *qualifier) postRecovery(ctx context.Context) error {
	if err := q.pruneBoundary(ctx); err != nil {
		return err
	}
	r := q.r.Recovery
	if r == nil || r.Uncertain || r.FinishedAt == nil || r.After == nil || r.PostRestart != nil {
		return errors.New("post-recovery restart requires resolved replay")
	}
	if _, _, err := q.epochAdmission(); err != nil {
		return err
	}
	if err := q.boot(ctx, "post-recovery"); err != nil {
		return err
	}
	inv, err := q.inspectEpochs(ctx)
	if err != nil {
		return err
	}
	if validEpochSample(inv, q.r.RunID, 3) != nil || !sameEpochNames(r.After, inv) {
		return errors.New("epoch set changed after recovered restart")
	}
	r.PostRestart = inv
	if err := q.save(); err != nil {
		r.PostRestart = nil
		return err
	}
	return nil
}

func genesisFile(name string) bool {
	return name == "data.mdb" || name == "epoc.txt" || name == "lock.mdb" || name == "vere.txt"
}

func validPruneRecord(p *PruneRecord, e *EpochLifecycleRecord, id string) error {
	if p == nil || e == nil || e.Final == nil || p.StartedAt.IsZero() || p.FinishedAt == nil || p.FinishedAt.Before(p.StartedAt) || validEpochSample(p.Before, id, 4) != nil || p.Before.Digest != e.Final.Digest || p.Cutoff != "0i0" || p.Directory.Name != "0i0" || p.Directory.Inode == 0 || !p.EpochRemoved || len(p.Files) != 4 || len(p.RemovedFiles) != 4 || validEpochSample(p.After, id, 3) != nil || !sameRetainedEntries(p.Before, p.After) {
		return errors.New("invalid completed prefix-deletion evidence")
	}
	seen := map[string]bool{}
	for i, f := range p.Files {
		if !genesisFile(f.Name) || seen[f.Name] || f.Inode == 0 || f.Device != p.Directory.Device || f.Size < 0 || p.RemovedFiles[i] != f.Name {
			return errors.New("invalid exact deletion manifest or progress")
		}
		seen[f.Name] = true
	}
	if p.PostRestart != nil && (validEpochSample(p.PostRestart, id, 3) != nil || !sameEpochNames(p.After, p.PostRestart)) {
		return errors.New("invalid post-prune restart evidence")
	}
	return nil
}

func validRecoveryRecord(r *EpochRecoveryRecord, p *PruneRecord, id string) error {
	if p == nil {
		return errors.New("missing prune evidence")
	}
	return validLatestRecovery(r, p.PostRestart, id)
}

func validLatestRecovery(r *EpochRecoveryRecord, expected *epochs.Inventory, id string) error {
	if r == nil || expected == nil || r.StartedAt.IsZero() || r.FinishedAt == nil || r.FinishedAt.Before(r.StartedAt) || r.Checkpoint.Name != "image.bin" || r.Checkpoint.Inode == 0 || r.Checkpoint.Size <= 0 || !r.CheckpointRemoved || r.RecoveredCheckpointBytes <= 0 || r.MemAvailableBytes < 5<<30 || r.DiskAvailableBytes < 8<<30 || validEpochSample(r.Before, id, len(expected.Entries)) != nil || !sameEpochNames(expected, r.Before) || validEpochSample(r.After, id, len(expected.Entries)) != nil || !sameEpochNames(r.Before, r.After) {
		return errors.New("invalid latest-epoch recovery evidence")
	}
	if r.PostRestart != nil && (validEpochSample(r.PostRestart, id, len(expected.Entries)) != nil || !sameEpochNames(r.After, r.PostRestart)) {
		return errors.New("invalid recovered restart evidence")
	}
	return nil
}

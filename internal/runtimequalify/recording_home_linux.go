//go:build linux && amd64

package runtimequalify

import (
	"errors"
	"os"
	"path/filepath"
)

// PrepareManagedRecordingHome provisions only the verified owner's fixed
// recording roots, after its manager has opened/migrated the matching store.
// An arbitrary New(tempDir) cannot target installed state. Existing objects are
// never chmodded, replaced, cleaned or repaired. No runtime/unit is contacted.
// Normal server construction does not select this preparation yet.
func PrepareManagedRecordingHome(managedDirectory string) (RecordingWorkerOptions, error) {
	return prepareManagedRecordingHome(managedDirectory, defaults())
}

func prepareManagedRecordingHome(managedDirectory string, d dependencies) (RecordingWorkerOptions, error) {
	base := filepath.Join(d.home, ".local/share/urbit-runner")
	if !recordingJobPath(d.home) || d.home == "/" || managedDirectory != filepath.Join(base, "managed") {
		return RecordingWorkerOptions{}, ErrRecordingWorker
	}
	q := &qualifier{d: d}
	if q.accountBoundary(false) != nil || q.ancestors(base, true) != nil || q.owned(base, os.ModeDir, false) != nil || q.owned(managedDirectory, os.ModeDir, true) != nil {
		return RecordingWorkerOptions{}, ErrRecordingWorker
	}
	before, err := os.Lstat(base)
	if err != nil || !before.IsDir() {
		return RecordingWorkerOptions{}, ErrRecordingWorker
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		return RecordingWorkerOptions{}, ErrRecordingWorker
	}
	defer root.Close()
	file, err := root.Open(".")
	if err != nil {
		return RecordingWorkerOptions{}, ErrRecordingWorker
	}
	opened, err := file.Stat()
	file.Close()
	if err != nil || !os.SameFile(before, opened) {
		return RecordingWorkerOptions{}, ErrRecordingWorker
	}
	for _, leaf := range []string{"recording-jobs", "recordings", "recording-images"} {
		if err := root.Mkdir(leaf, 0700); err != nil && !errors.Is(err, os.ErrExist) {
			return RecordingWorkerOptions{}, ErrRecordingWorker
		}
		if q.owned(filepath.Join(base, leaf), os.ModeDir, true) != nil {
			return RecordingWorkerOptions{}, ErrRecordingWorker
		}
		// Sync both the child inode and its parent entry before returning any
		// setup acknowledgement. A failed partial setup remains untouched.
		child, err := root.OpenRoot(leaf)
		if err != nil {
			return RecordingWorkerOptions{}, ErrRecordingWorker
		}
		if !recordingJobChildBound(root, leaf, child) {
			child.Close()
			return RecordingWorkerOptions{}, ErrRecordingWorker
		}
		err = recordingJobSync(child)
		child.Close()
		if err != nil {
			return RecordingWorkerOptions{}, ErrRecordingWorker
		}
	}
	after, err := os.Lstat(base)
	if err != nil || !os.SameFile(before, after) || recordingJobSync(root) != nil {
		return RecordingWorkerOptions{}, ErrRecordingWorker
	}
	return recordingServiceOptions(d)
}

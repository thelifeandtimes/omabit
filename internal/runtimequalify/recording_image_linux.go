//go:build linux && amd64

package runtimequalify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const recordingImageMaxBytes = 128 << 20
const recordingImageMaxCount = 8

// RecordingExecutable is a private local handoff, never an RPC response. Keeping
// exact executable bytes outside a mutable installation symlink lets an already
// admitted worker retain its original version across later manager upgrades.
type RecordingExecutable struct {
	Path   string
	SHA256 string
}

// FreezeRecordingExecutable copies only the kernel-selected running executable
// into the selected owner's existing private recording-images directory. It
// accepts no source/destination path and installs or executes nothing. Normal
// managed launch does not call this preparation yet.
func FreezeRecordingExecutable(ctx context.Context) (RecordingExecutable, error) {
	d := defaults()
	q := &qualifier{d: d}
	root := filepath.Join(d.home, ".local/share/urbit-runner/recording-images")
	if ctx.Err() != nil || q.accountBoundary(false) != nil || q.ancestors(root, true) != nil || q.owned(root, os.ModeDir, true) != nil {
		return RecordingExecutable{}, ErrRecordingWorker
	}
	self, err := observeRecordingWorkerIdentity(q)
	if err != nil {
		return RecordingExecutable{}, ErrRecordingWorker
	}
	f, err := os.Open("/proc/self/exe")
	if err != nil {
		return RecordingExecutable{}, ErrRecordingWorker
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return RecordingExecutable{}, ErrRecordingWorker
	}
	return freezeRecordingExecutable(ctx, root, self.ExecutableSHA256, info.Size(), f, recordingJobSpace)
}

func recordingImageInfo(info os.FileInfo, size int64) bool {
	if info == nil {
		return false
	}
	s, ok := info.Sys().(*syscall.Stat_t)
	return ok && info.Mode() == 0500 && info.Size() == size && size > 0 && size <= recordingImageMaxBytes && s.Uid == uint32(os.Geteuid()) && s.Nlink == 1
}

func verifyRecordingImage(ctx context.Context, root *os.Root, name, digest string, size int64) error {
	f, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return ErrRecordingWorker
	}
	defer f.Close()
	before, err := f.Stat()
	if err != nil || !recordingImageInfo(before, size) {
		return ErrRecordingWorker
	}
	h := sha256.New()
	n, err := io.Copy(h, &recordingProcessReader{ctx: ctx, r: io.LimitReader(f, recordingImageMaxBytes+1)})
	path, pathErr := root.Lstat(name)
	after, statErr := f.Stat()
	if err != nil || n != size || hex.EncodeToString(h.Sum(nil)) != digest || pathErr != nil || statErr != nil || !recordingImageInfo(after, size) || !os.SameFile(before, after) || !os.SameFile(after, path) || f.Sync() != nil || recordingJobSync(root) != nil || ctx.Err() != nil {
		return ErrRecordingWorker
	}
	return nil
}

func freezeRecordingExecutable(ctx context.Context, path, digest string, size int64, source io.Reader, free func(*os.Root, uint64) error) (RecordingExecutable, error) {
	if ctx.Err() != nil || !recordingJobHex(digest, 64) || size <= 0 || size > recordingImageMaxBytes || source == nil || free == nil {
		return RecordingExecutable{}, ErrRecordingWorker
	}
	root, err := openRecordingPrivateRoot(path)
	if err != nil {
		return RecordingExecutable{}, err
	}
	defer root.Close()
	lock, err := recordingJobFile(root, "admission.lock", os.O_RDWR|os.O_CREATE, 0)
	if err != nil {
		return RecordingExecutable{}, ErrRecordingWorker
	}
	defer lock.Close()
	if syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) != nil || !recordingJobBound(root, "admission.lock", lock, 0) {
		return RecordingExecutable{}, ErrRecordingWorker
	}
	name := digest + ".runner"
	if _, err := root.Lstat(name); err == nil {
		if verifyRecordingImage(ctx, root, name, digest, size) != nil {
			return RecordingExecutable{}, ErrRecordingWorker
		}
		return RecordingExecutable{Path: filepath.Join(path, name), SHA256: digest}, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return RecordingExecutable{}, ErrRecordingWorker
	}
	dir, err := root.Open(".")
	if err != nil {
		return RecordingExecutable{}, ErrRecordingWorker
	}
	entries, err := dir.ReadDir(recordingImageMaxCount + 2)
	dir.Close()
	if err != nil && err != io.EOF || len(entries) > recordingImageMaxCount {
		return RecordingExecutable{}, ErrRecordingJobLimit
	}
	count := 0
	for _, entry := range entries {
		if entry.Name() == "admission.lock" {
			continue
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".runner") || !recordingJobHex(strings.TrimSuffix(entry.Name(), ".runner"), 64) || entry.Type() != 0 {
			return RecordingExecutable{}, ErrRecordingWorker
		}
		count++
	}
	if count >= recordingImageMaxCount {
		return RecordingExecutable{}, ErrRecordingJobLimit
	}
	if err := free(root, uint64(size)+(64<<20)); err != nil {
		if errors.Is(err, ErrRecordingJobLimit) {
			return RecordingExecutable{}, ErrRecordingJobLimit
		}
		return RecordingExecutable{}, ErrRecordingWorker
	}
	f, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0600)
	if err != nil {
		return RecordingExecutable{}, ErrRecordingWorker
	}
	// A partial/hash-refused original stays non-executable and is never repaired,
	// overwritten or automatically removed. Only a complete exact hash can
	// become executable. The selected executable is checked again before use.
	h := sha256.New()
	n, copyErr := io.Copy(io.MultiWriter(f, h), &recordingProcessReader{ctx: ctx, r: io.LimitReader(source, size+1)})
	if copyErr == nil && (n != size || hex.EncodeToString(h.Sum(nil)) != digest || !recordingJobBound(root, name, f, size)) {
		copyErr = ErrRecordingWorker
	}
	if copyErr == nil {
		copyErr = f.Sync()
	}
	if copyErr == nil {
		copyErr = f.Chmod(0500)
	}
	err = errors.Join(copyErr, f.Sync(), f.Close(), recordingJobSync(root))
	if err != nil || verifyRecordingImage(ctx, root, name, digest, size) != nil {
		return RecordingExecutable{}, ErrRecordingWorker
	}
	return RecordingExecutable{Path: filepath.Join(path, name), SHA256: digest}, nil
}

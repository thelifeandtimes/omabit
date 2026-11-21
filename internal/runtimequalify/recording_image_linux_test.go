//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func recordingImageFixture(t *testing.T) (string, []byte, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	payload := []byte("inert executable-byte fixture, never executed\x00\xff")
	return root, payload, recordingJobSHA(payload)
}

func recordingImageEnough(*os.Root, uint64) error { return nil }

type recordingImageNoRead struct{ t *testing.T }

func (r recordingImageNoRead) Read([]byte) (int, error) {
	r.t.Fatal("verified existing image was recopied")
	return 0, io.ErrUnexpectedEOF
}

func TestRecordingImageKeepsExactImmutableBytesAndReusesWithoutReplacement(t *testing.T) {
	root, payload, digest := recordingImageFixture(t)
	image, err := freezeRecordingExecutable(t.Context(), root, digest, int64(len(payload)), bytes.NewReader(payload), recordingImageEnough)
	if err != nil || image.SHA256 != digest || image.Path != filepath.Join(root, digest+".runner") {
		t.Fatal("image selection changed", err)
	}
	info, err := os.Lstat(image.Path)
	if err != nil || !recordingImageInfo(info, int64(len(payload))) {
		t.Fatal("frozen image is exposed, writable or incomplete", err)
	}
	got, err := os.ReadFile(image.Path)
	if err != nil || !bytes.Equal(got, payload) {
		t.Fatal("image bytes transformed", err)
	}
	for pass := 0; pass < 2; pass++ {
		saved, err := freezeRecordingExecutable(t.Context(), root, digest, int64(len(payload)), recordingImageNoRead{t}, recordingImageEnough)
		after, statErr := os.Lstat(image.Path)
		if err != nil || statErr != nil || saved != image || !os.SameFile(info, after) || info.ModTime() != after.ModTime() {
			t.Fatal("reuse changed original image", err)
		}
	}
}

func TestRecordingImageRejectsAlteredExposedLinkedAndPartialOriginalsWithoutRepair(t *testing.T) {
	for _, scenario := range []string{"partial", "wrong-hash", "writable", "exposed", "symlink", "hardlink"} {
		t.Run(scenario, func(t *testing.T) {
			root, payload, digest := recordingImageFixture(t)
			path := filepath.Join(root, digest+".runner")
			data, mode := payload, os.FileMode(0500)
			if scenario == "partial" {
				data, mode = payload[:2], 0600
			}
			if scenario == "wrong-hash" {
				data = bytes.Repeat([]byte("x"), len(payload))
			}
			if scenario == "writable" {
				mode = 0700
			}
			if scenario == "exposed" {
				mode = 0555
			}
			if scenario == "symlink" {
				if err := os.WriteFile(filepath.Join(root, "original-image"), data, 0500); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("original-image", path); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.WriteFile(path, data, mode); err != nil {
					t.Fatal(err)
				}
				if scenario == "hardlink" {
					if err := os.Link(path, filepath.Join(root, "original-link")); err != nil {
						t.Fatal(err)
					}
				}
			}
			before, err := os.Lstat(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := freezeRecordingExecutable(t.Context(), root, digest, int64(len(payload)), recordingImageNoRead{t}, recordingImageEnough); err == nil {
				t.Fatal("changed/linked original was adopted")
			}
			after, err := os.Lstat(path)
			got, readErr := os.ReadFile(path)
			if err != nil || readErr != nil || !os.SameFile(before, after) || before.Mode() != after.Mode() || !bytes.Equal(data, got) {
				t.Fatal("refusal repaired or removed original", err)
			}
		})
	}
}

func TestRecordingImageCopyFailureRetainsNonExecutableBytesAndNeverRetries(t *testing.T) {
	for _, scenario := range []string{"short", "long", "wrong-hash", "read-error"} {
		t.Run(scenario, func(t *testing.T) {
			root, payload, digest := recordingImageFixture(t)
			var source io.Reader = bytes.NewReader(payload)
			switch scenario {
			case "short":
				source = bytes.NewReader(payload[:3])
			case "long":
				source = bytes.NewReader(append(payload, 1))
			case "wrong-hash":
				source = bytes.NewReader(bytes.Repeat([]byte("x"), len(payload)))
			case "read-error":
				source = io.MultiReader(bytes.NewReader(payload[:3]), recordingImageReadError{})
			}
			if _, err := freezeRecordingExecutable(t.Context(), root, digest, int64(len(payload)), source, recordingImageEnough); err == nil {
				t.Fatal("incomplete image accepted")
			}
			path := filepath.Join(root, digest+".runner")
			info, err := os.Lstat(path)
			if err != nil || info.Mode() != 0600 {
				t.Fatal("failed copy was removed or executable", err)
			}
			before, _ := os.ReadFile(path)
			if _, err := freezeRecordingExecutable(t.Context(), root, digest, int64(len(payload)), bytes.NewReader(payload), recordingImageEnough); err == nil {
				t.Fatal("partial original was silently retried")
			}
			after, _ := os.ReadFile(path)
			if !bytes.Equal(before, after) {
				t.Fatal("partial bytes were replaced")
			}
		})
	}
}

type recordingImageReadError struct{}

func (recordingImageReadError) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestRecordingImageAdmissionBoundsAndExclusiveCopy(t *testing.T) {
	root, payload, digest := recordingImageFixture(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := freezeRecordingExecutable(ctx, root, digest, int64(len(payload)), bytes.NewReader(payload), recordingImageEnough); err == nil {
		t.Fatal("canceled copy accepted")
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != 0 {
		t.Fatal("canceled admission touched storage")
	}
	free := func(_ *os.Root, required uint64) error {
		if required != uint64(len(payload))+(64<<20) {
			t.Fatal("image bytes/reserve not admitted")
		}
		return ErrRecordingJobLimit
	}
	if _, err := freezeRecordingExecutable(t.Context(), root, digest, int64(len(payload)), recordingImageNoRead{t}, free); !errors.Is(err, ErrRecordingJobLimit) {
		t.Fatal("disk pressure accepted", err)
	}
	if _, err := freezeRecordingExecutable(t.Context(), root, digest, int64(len(payload)), recordingImageNoRead{t}, func(*os.Root, uint64) error { return ErrRecordingJobStorage }); !errors.Is(err, ErrRecordingWorker) || errors.Is(err, ErrRecordingJobLimit) {
		t.Fatal("unknown storage failure mislabeled as disk pressure", err)
	}
	lock, err := os.OpenFile(filepath.Join(root, "admission.lock"), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	if _, err := freezeRecordingExecutable(t.Context(), root, digest, int64(len(payload)), recordingImageNoRead{t}, recordingImageEnough); err == nil {
		t.Fatal("another copier bypassed active admission")
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_UN); err != nil {
		t.Fatal(err)
	}
	for n := 0; n < recordingImageMaxCount; n++ {
		data := append(bytes.Clone(payload), byte(n))
		if _, err := freezeRecordingExecutable(t.Context(), root, recordingJobSHA(data), int64(len(data)), bytes.NewReader(data), recordingImageEnough); err != nil {
			t.Fatal("bounded image copy failed", n, err)
		}
	}
	if _, err := freezeRecordingExecutable(t.Context(), root, digest, int64(len(payload)), recordingImageNoRead{t}, recordingImageEnough); !errors.Is(err, ErrRecordingJobLimit) {
		t.Fatal("unbounded image accumulation", err)
	}
}

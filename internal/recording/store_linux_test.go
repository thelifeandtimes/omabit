//go:build linux

package recording

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func openTestStore(t *testing.T, limit int64) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	s, err := OpenStore(dir, Limits{MaxSessionBytes: limit, ReserveFreeBytes: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s, dir
}

func TestStorePinsIndependentAttachmentIdentityAfterAdmission(t *testing.T) {
	s, _ := openTestStore(t, 1<<20)
	m := attachmentMetadata()
	original, err := m.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	w, err := s.Create(m)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	m.Identity.Target.InstanceID = "other"
	m.Identity.ProjectNamespace.Inode++
	if _, err = w.Append(Stdout, []byte("original"), m.AttachedAt.UnixNano(), 0); err != nil {
		t.Fatal(err)
	}
	page, err := s.ReadPage(t.Context(), m.SessionID, nil, 1, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	got, err := page.Metadata.Bytes()
	if err != nil || !bytes.Equal(got, original) {
		t.Fatal("caller rebound admitted writer metadata", err)
	}
	if w.metadata.Identity.Target.InstanceID != "member" || w.metadata.Identity.ProjectNamespace.Inode != 200 {
		t.Fatal("writer retained caller-owned nested pointer")
	}
}

func TestPrivateStorePersistsOriginalBytesAndExactPagingAcrossReopen(t *testing.T) {
	s, dir := openTestStore(t, 1<<20)
	m := attachmentMetadata()
	w, err := s.Create(m)
	if err != nil {
		t.Fatal(err)
	}
	data := append(bytes.Repeat([]byte("trace\n"), 350), bytes.Repeat([]byte("x"), 12288)...)
	data = append(data, []byte("0wdiagnostic.atom Authorization: fake-private\r\x1b[2K\xff\x00")...)
	gap, _ := (GapEvent{Version: 1, Reason: "transport-interrupted"}).Bytes()
	exit, _ := (ExitEvent{Version: 1, ContainerID: m.ContainerID, StartedUnixNano: m.AttachedAt.UnixNano(), FinishedUnixNano: m.AttachedAt.Add(time.Second).UnixNano(), Code: 0}).Bytes()
	want := [][]byte{data, []byte("stderr\n"), gap, exit}
	kinds := []Kind{Stdout, Stderr, Gap, Exit}
	for i, b := range want {
		if _, err := w.Append(kinds[i], b, m.AttachedAt.UnixNano()+int64(i), uint64(i)); err != nil {
			t.Fatal(err)
		}
	}
	live, err := s.ReadPage(context.Background(), m.SessionID, nil, 1, 1<<20)
	if err != nil || len(live.Frames) != 1 || !live.WriterPresent || !bytes.Equal(live.Frames[0].Data, data) {
		t.Fatal(live, err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = OpenStore(dir, Limits{MaxSessionBytes: 1 << 20, ReserveFreeBytes: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	page, err := s.ReadPage(context.Background(), m.SessionID, &live.Next, 256, 1<<20)
	if err != nil || page.Tail != "snapshot-eof" || page.WriterPresent || len(page.Frames) != 3 {
		t.Fatal(page, err)
	}
	for i, f := range page.Frames {
		if f.Sequence != uint64(i+2) || f.Kind != kinds[i+1] || !bytes.Equal(f.Data, want[i+1]) {
			t.Fatal("original/timing/stream coverage changed", f)
		}
	}
	repeated, err := s.ReadPage(context.Background(), m.SessionID, &live.Next, 256, 1<<20)
	if err != nil || len(repeated.Frames) != 3 || repeated.Next != page.Next {
		t.Fatal("cursor replay changed coverage", err)
	}
	end, err := s.ReadPage(context.Background(), m.SessionID, &page.Next, 256, 1<<20)
	if err != nil || len(end.Frames) != 0 || end.Next != page.Next || end.Tail != "snapshot-eof" {
		t.Fatal("cursor repeated retained output", end, err)
	}
	if _, err := s.Create(m); !errors.Is(err, os.ErrExist) {
		t.Fatal("existing original session reused", err)
	}
	bad := live.Next
	bad.MetadataSHA256 = strings.Repeat("f", 64)
	if _, err := s.ReadPage(context.Background(), m.SessionID, &bad, 1, 1<<20); !errors.Is(err, ErrCursor) {
		t.Fatal("foreign metadata cursor accepted", err)
	}
}

func TestStoreReportsPartialCorruptAndLostCursorWithoutRepair(t *testing.T) {
	for _, damage := range []string{"partial", "corrupt"} {
		t.Run(damage, func(t *testing.T) {
			s, dir := openTestStore(t, 1<<20)
			m := attachmentMetadata()
			w, err := s.Create(m)
			if err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 2; i++ {
				if _, err := w.Append(Stdout, []byte("original payload"), m.AttachedAt.UnixNano()+int64(i), uint64(i)); err != nil {
					t.Fatal(err)
				}
			}
			if err := w.Close(); err != nil {
				t.Fatal(err)
			}
			complete, err := s.ReadPage(context.Background(), m.SessionID, nil, 256, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			file := filepath.Join(dir, m.SessionID, "frames.obr")
			original, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			broken := append([]byte(nil), original...)
			wantTail := "corrupt"
			if damage == "partial" {
				broken = broken[:len(broken)-1]
				wantTail = "incomplete"
			} else {
				broken[len(broken)-1] ^= 1
			}
			if err := os.WriteFile(file, broken, 0600); err != nil {
				t.Fatal(err)
			}
			page, err := s.ReadPage(context.Background(), m.SessionID, nil, 256, 1<<20)
			if err != nil || len(page.Frames) != 1 || page.Next.Sequence != 1 || page.Tail != wantTail {
				t.Fatal(page, err)
			}
			if _, err := s.ReadPage(context.Background(), m.SessionID, &complete.Next, 1, 1<<20); !errors.Is(err, ErrCursor) {
				t.Fatal("lost cursor silently rewound", err)
			}
			after, _ := os.ReadFile(file)
			if !bytes.Equal(after, broken) {
				t.Fatal("reader repaired/truncated original")
			}
		})
	}
}

func TestStorePinsPrivateRootAndRefusesLinkedOrExposedFiles(t *testing.T) {
	s, dir := openTestStore(t, 1<<20)
	m := attachmentMetadata()
	moved := dir + "-moved"
	if err := os.Rename(dir, moved); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(moved)
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	w, err := s.Create(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, m.SessionID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("root pathname replacement rebound writer")
	}
	data := filepath.Join(moved, m.SessionID, "frames.obr")
	if err := os.Link(data, filepath.Join(t.TempDir(), "other")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReadPage(context.Background(), m.SessionID, nil, 1, 1<<20); !errors.Is(err, ErrStorage) {
		t.Fatal("hardlinked recording accepted", err)
	}
	public := t.TempDir()
	if err := os.Chmod(public, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenStore(public, s.limits); !errors.Is(err, ErrStorage) {
		t.Fatal("exposed root accepted", err)
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(moved, link); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenStore(link, s.limits); !errors.Is(err, ErrStorage) {
		t.Fatal("symbolic root accepted", err)
	}
}

func TestStorePressureAndLimitsRefuseBeforeWriting(t *testing.T) {
	s, dir := openTestStore(t, 4096)
	m := attachmentMetadata()
	realFree := s.free
	s.free = func(*os.Root, uint64) error { return ErrReserve }
	if _, err := s.Create(m); !errors.Is(err, ErrReserve) {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, m.SessionID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("reserve failure created recording")
	}
	s.free = realFree
	w, err := s.Create(m)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	file := filepath.Join(dir, m.SessionID, "frames.obr")
	before, _ := os.ReadFile(file)
	s.free = func(*os.Root, uint64) error { return ErrReserve }
	if _, err := w.Append(Stdout, []byte("unwritten"), m.AttachedAt.UnixNano(), 0); !errors.Is(err, ErrReserve) {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(file)
	if !bytes.Equal(before, after) {
		t.Fatal("reserve refusal wrote bytes")
	}
	s.free = realFree
	if _, err := w.Append(Stdout, bytes.Repeat([]byte("x"), 3000), m.AttachedAt.UnixNano(), 0); err != nil {
		t.Fatal(err)
	}
	before, _ = os.ReadFile(file)
	if _, err := w.Append(Stdout, bytes.Repeat([]byte("y"), 1000), m.AttachedAt.UnixNano()+1, 1); !errors.Is(err, ErrLimit) {
		t.Fatal(err)
	}
	after, _ = os.ReadFile(file)
	if !bytes.Equal(before, after) {
		t.Fatal("quota refusal wrote bytes")
	}
}

func TestWriterRefusesWrongStreamExitOrReplacedOriginal(t *testing.T) {
	s, dir := openTestStore(t, 1<<20)
	m := attachmentMetadata()
	w, err := s.Create(m)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, m.SessionID, "frames.obr")
	before, _ := os.ReadFile(file)
	bad, _ := (ExitEvent{Version: 1, ContainerID: strings.Repeat("f", 64), StartedUnixNano: 1, FinishedUnixNano: 2, Code: 0}).Bytes()
	for _, record := range []struct {
		kind Kind
		data []byte
	}{{Exit, bad}, {PTYCombined, []byte("not a combined stream")}} {
		if _, err := w.Append(record.kind, record.data, m.AttachedAt.UnixNano(), 0); !errors.Is(err, ErrInvalid) {
			t.Fatal("wrong recording binding accepted", err)
		}
	}
	after, _ := os.ReadFile(file)
	if !bytes.Equal(before, after) {
		t.Fatal("binding refusal wrote bytes")
	}
	retained := file + ".retained"
	if err := os.Rename(file, retained); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("replacement"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Append(Stdout, []byte("must not go to orphan or replacement"), m.AttachedAt.UnixNano(), 0); !errors.Is(err, ErrStorage) {
		t.Fatal("writer rebound replaced file", err)
	}
	if _, err := w.Append(Stdout, []byte("no retry"), m.AttachedAt.UnixNano(), 0); !errors.Is(err, ErrPoisoned) {
		t.Fatal(err)
	}
	if err := w.Close(); !errors.Is(err, ErrPoisoned) {
		t.Fatal("writer loss hidden at close", err)
	}
	after, _ = os.ReadFile(retained)
	if !bytes.Equal(before, after) {
		t.Fatal("replaced original changed")
	}
}

func TestLiveStorePagingDoesNotDuplicateOrLoseCompleteFrames(t *testing.T) {
	s, _ := openTestStore(t, 1<<20)
	m := attachmentMetadata()
	w, err := s.Create(m)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		for i := 0; i < 100; i++ {
			if _, err := w.Append(Stdout, bytes.Repeat([]byte{byte(i)}, 512), m.AttachedAt.UnixNano()+int64(i), uint64(i)); err != nil {
				done <- err
				return
			}
		}
		done <- w.Close()
	}()
	var cursor *Cursor
	received := 0
	for received < 100 {
		page, err := s.ReadPage(ctx, m.SessionID, cursor, 7, 1<<20)
		if err != nil {
			t.Fatal(err)
		}
		if page.Tail == "corrupt" {
			t.Fatal("live append became corruption")
		}
		for _, frame := range page.Frames {
			if frame.Sequence != uint64(received+1) || len(frame.Data) != 512 || frame.Data[0] != byte(received) {
				t.Fatal("live cursor lost/duplicated output", frame.Sequence, received)
			}
			received++
		}
		copy := page.Next
		cursor = &copy
		if len(page.Frames) == 0 {
			runtime.Gosched()
		}
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestPrivateStoreRefusesNewLegacyCaptureWithoutInventingIdentity(t *testing.T) {
	s, dir := openTestStore(t, 1<<20)
	m := testMetadata()
	if _, err := s.Create(m); !errors.Is(err, ErrLegacyMetadata) {
		t.Fatal("new legacy capture admitted", err)
	}
	if _, err := os.Lstat(filepath.Join(dir, m.SessionID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("legacy refusal changed storage", err)
	}
	m.Version = 4
	if _, err := s.Create(m); !errors.Is(err, ErrInvalid) {
		t.Fatal("unknown metadata version treated as legacy", err)
	}
	if _, err := os.Lstat(filepath.Join(dir, m.SessionID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("unknown version refusal changed storage", err)
	}
}

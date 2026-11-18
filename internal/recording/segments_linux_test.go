//go:build linux

package recording

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func segmentedMetadata() Metadata { m := attachmentMetadata(); m.Version = 3; return m }
func segmentStore(t *testing.T, total, segment int64) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	s, err := OpenStore(dir, Limits{MaxSessionBytes: total, MaxSegmentBytes: segment, ReserveFreeBytes: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s, dir
}
func fileSnapshot(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	entries, e := os.ReadDir(dir)
	if e != nil {
		t.Fatal(e)
	}
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			t.Fatal("nonordinary evidence file")
		}
		out[entry.Name()], e = os.ReadFile(filepath.Join(dir, entry.Name()))
		if e != nil {
			t.Fatal(e)
		}
	}
	return out
}
func TestSegmentsRotateAndResumeExactCursorsWithoutGaps(t *testing.T) {
	s, dir := segmentStore(t, 1<<20, 4096)
	m := segmentedMetadata()
	w, e := s.Create(m)
	if e != nil {
		t.Fatal(e)
	}
	var want []Frame
	for i := 0; i < 24; i++ {
		kind := Stdout
		if i%3 == 0 {
			kind = Stderr
		}
		f, e := w.Append(kind, bytes.Repeat([]byte{byte(i)}, 900), m.AttachedAt.UnixNano()+int64(i), uint64(i))
		if e != nil {
			t.Fatal(e)
		}
		want = append(want, f)
		if i == 7 {
			if e = w.Rotate(); e != nil {
				t.Fatal(e)
			}
		}
	}
	if w.segment < 6 {
		t.Fatal("fixture did not rotate")
	}
	if e = w.Close(); e != nil {
		t.Fatal(e)
	}
	originals := fileSnapshot(t, filepath.Join(dir, m.SessionID))
	var cursor *Cursor
	var got []Frame
	for len(got) < len(want) {
		page, e := s.ReadPage(t.Context(), m.SessionID, cursor, 3, 1<<20)
		if e != nil || len(page.Frames) == 0 {
			t.Fatal(page, e)
		}
		replay, e := s.ReadPage(t.Context(), m.SessionID, cursor, 3, 1<<20)
		if e != nil || !reflect.DeepEqual(page.Frames, replay.Frames) || page.Next != replay.Next {
			t.Fatal("page reconnect changed retained bytes", e)
		}
		got = append(got, page.Frames...)
		next := page.Next
		cursor = &next
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatal("rotation changed bytes, stream, timing, sequence or digest")
	}
	if !reflect.DeepEqual(originals, fileSnapshot(t, filepath.Join(dir, m.SessionID))) {
		t.Fatal("reader changed originals")
	}
}
func TestSegmentsContinueWithNewEpochAndExplicitUnknownGap(t *testing.T) {
	for _, partial := range []bool{false, true} {
		t.Run(map[bool]string{false: "closed", true: "torn-tail"}[partial], func(t *testing.T) {
			s, dir := segmentStore(t, 1<<20, 4096)
			m := segmentedMetadata()
			w, e := s.Create(m)
			if e != nil {
				t.Fatal(e)
			}
			first, e := w.Append(Stdout, []byte("original\x00\xff\x1b[2K0wdiagnostic\n"), m.AttachedAt.UnixNano(), 100)
			if e != nil {
				t.Fatal(e)
			}
			if _, e = w.Append(Stderr, []byte("possibly lost"), m.AttachedAt.UnixNano()+1, 101); e != nil {
				t.Fatal(e)
			}
			if e = w.Close(); e != nil {
				t.Fatal(e)
			}
			path := filepath.Join(dir, m.SessionID, "frames.obr")
			if partial {
				info, e := os.Stat(path)
				if e != nil {
					t.Fatal(e)
				}
				if e = os.Truncate(path, info.Size()-1); e != nil {
					t.Fatal(e)
				}
			}
			before := fileSnapshot(t, filepath.Join(dir, m.SessionID))
			page, e := s.ReadPage(t.Context(), m.SessionID, nil, 1, 1<<20)
			if e != nil || len(page.Frames) != 1 || !reflect.DeepEqual(page.Frames[0], first) {
				t.Fatal(page, e)
			}
			// Wall-clock reversal is valid; the new segment declares a new clock epoch.
			at := m.AttachedAt.Add(-time.Second)
			continued, e := s.Continue(t.Context(), m, at)
			if e != nil {
				t.Fatal(e)
			}
			if _, e = continued.Append(Stderr, []byte("resumed\n"), at.UnixNano()+1, 1); e != nil {
				t.Fatal(e)
			}
			if e = continued.Close(); e != nil {
				t.Fatal(e)
			}
			after := fileSnapshot(t, filepath.Join(dir, m.SessionID))
			for name, data := range before {
				if !bytes.Equal(after[name], data) {
					t.Fatal("continuation rewrote original", name)
				}
			}
			tail, e := s.ReadPage(t.Context(), m.SessionID, &page.Next, 256, 1<<20)
			if e != nil || tail.Tail != "snapshot-eof" {
				t.Fatal(tail, e)
			}
			offset := 1
			if partial {
				offset = 0
			}
			if len(tail.Frames) != offset+2 {
				t.Fatal("missing or duplicated prefix", len(tail.Frames))
			}
			g, e := ReadGap(tail.Frames[offset].Data)
			reason := "recorder-restarted"
			if partial {
				reason = "corrupt-tail"
			}
			if e != nil || tail.Frames[offset].Kind != Gap || g.Reason != reason || g.KnownMissingBytes != nil || tail.Frames[offset].ElapsedNano != 0 || tail.Frames[offset].UnixNano != at.UnixNano() || tail.Frames[offset].EpochUnixNano != at.UnixNano() || tail.Frames[offset].Segment != 2 {
				t.Fatal("restart implied continuous or zero-loss coverage", g, e)
			}
			if string(tail.Frames[offset+1].Data) != "resumed\n" {
				t.Fatal("lost resumed bytes")
			}
		})
	}
}
func TestSegmentsContinuationRefusesAuthorityChangesConcurrentWriterAndCorruption(t *testing.T) {
	for _, scenario := range []string{"writer-live", "binding", "corrupt", "missing-segment", "foreign-header", "unmarked-torn-tail", "terminal", "reserve", "legacy", "cancelled"} {
		t.Run(scenario, func(t *testing.T) {
			s, dir := segmentStore(t, 1<<20, 4096)
			m := segmentedMetadata()
			w, e := s.Create(m)
			if e != nil {
				t.Fatal(e)
			}
			for i := 0; i < 2; i++ {
				if _, e = w.Append(Stdout, []byte("original"), m.AttachedAt.UnixNano()+int64(i), uint64(i)); e != nil {
					t.Fatal(e)
				}
			}
			if scenario == "missing-segment" || scenario == "foreign-header" || scenario == "unmarked-torn-tail" {
				if e = w.Rotate(); e != nil {
					t.Fatal(e)
				}
				if _, e = w.Append(Stdout, []byte("next"), m.AttachedAt.UnixNano()+3, 3); e != nil {
					t.Fatal(e)
				}
			}
			if scenario == "terminal" {
				b, _ := (ExitEvent{Version: 1, ContainerID: m.ContainerID, StartedUnixNano: 1, FinishedUnixNano: 2, Code: 7}).Bytes()
				if _, e = w.Append(Exit, b, m.AttachedAt.UnixNano()+4, 4); e != nil {
					t.Fatal(e)
				}
				if _, e = w.Append(Stdout, []byte("after exit"), m.AttachedAt.UnixNano()+5, 5); !errors.Is(e, ErrPoisoned) {
					t.Fatal("terminal recording accepted another output frame", e)
				}
			}
			if scenario != "writer-live" {
				if e = w.Close(); e != nil {
					t.Fatal(e)
				}
			} else {
				defer w.Close()
			}
			expected := m
			root := filepath.Join(dir, m.SessionID)
			switch scenario {
			case "binding":
				expected.Identity.Target.Generation++ // writer's independent copy remains bound
			case "legacy":
				expected.Version = 2
			case "reserve":
				s.free = func(*os.Root, uint64) error { return ErrReserve }
			case "corrupt":
				p := filepath.Join(root, "frames.obr")
				b, _ := os.ReadFile(p)
				b[len(b)-1] ^= 1
				if e = os.WriteFile(p, b, 0600); e != nil {
					t.Fatal(e)
				}
			case "missing-segment":
				if e = os.Rename(filepath.Join(root, segmentName(2)), filepath.Join(root, segmentName(3))); e != nil {
					t.Fatal(e)
				}
			case "foreign-header":
				p := filepath.Join(root, segmentName(2))
				b, _ := os.ReadFile(p)
				b[headerBytes-1] ^= 1
				if e = os.WriteFile(p, b, 0600); e != nil {
					t.Fatal(e)
				}
			case "unmarked-torn-tail":
				p := filepath.Join(root, "frames.obr")
				b, _ := os.ReadFile(p)
				if e = os.Truncate(p, int64(len(b)-1)); e != nil {
					t.Fatal(e)
				}
			}
			before := fileSnapshot(t, root)
			ctx := t.Context()
			if scenario == "cancelled" {
				cancelled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = cancelled
			}
			continued, e := s.Continue(ctx, expected, m.AttachedAt.Add(time.Second))
			if e == nil {
				continued.Close()
				t.Fatal("continued without exact complete admission", scenario)
			}
			if !reflect.DeepEqual(before, fileSnapshot(t, root)) {
				t.Fatal("refusal changed original files", scenario)
			}
		})
	}
}
func TestSegmentsRotationAdmissionCountsNewHeadersAndPreservesOriginals(t *testing.T) {
	s, dir := segmentStore(t, 4096, 2000)
	m := segmentedMetadata()
	w, e := s.Create(m)
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	for i := 0; i < 2; i++ {
		if _, e = w.Append(Stdout, bytes.Repeat([]byte("x"), 900), m.AttachedAt.UnixNano()+int64(i), uint64(i)); e != nil {
			t.Fatal(e)
		}
	}
	before := fileSnapshot(t, filepath.Join(dir, m.SessionID))
	if _, e = w.Append(Stdout, bytes.Repeat([]byte("x"), 1760), m.AttachedAt.UnixNano()+4, 4); !errors.Is(e, ErrLimit) {
		t.Fatal("session limit ignored rotation header", e)
	}
	if !reflect.DeepEqual(before, fileSnapshot(t, filepath.Join(dir, m.SessionID))) {
		t.Fatal("quota refusal created a segment")
	}
}
func TestSegmentReaderRequiresTornPrefixGapAcrossEmptySuccessor(t *testing.T) {
	s, dir := segmentStore(t, 1<<20, 4096)
	m := segmentedMetadata()
	w, e := s.Create(m)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = w.Append(Stdout, []byte("prefix"), m.AttachedAt.UnixNano(), 0); e != nil {
		t.Fatal(e)
	}
	if _, e = w.Append(Stdout, []byte("torn"), m.AttachedAt.UnixNano()+1, 1); e != nil {
		t.Fatal(e)
	}
	w.Close()
	path := filepath.Join(dir, m.SessionID, "frames.obr")
	info, _ := os.Stat(path)
	if e = os.Truncate(path, info.Size()-1); e != nil {
		t.Fatal(e)
	}
	w, e = s.Continue(t.Context(), m, m.AttachedAt.Add(time.Second))
	if e != nil {
		t.Fatal(e)
	}
	w.Close()
	// Simulate process death between the successor's durable header and its gap.
	second := filepath.Join(dir, m.SessionID, segmentName(2))
	if e = os.Truncate(second, int64(headerBytes)); e != nil {
		t.Fatal(e)
	}
	w, e = s.Continue(t.Context(), m, m.AttachedAt.Add(2*time.Second))
	if e != nil {
		t.Fatal(e)
	}
	w.Close()
	page, e := s.ReadPage(t.Context(), m.SessionID, nil, 256, 1<<20)
	if e != nil || page.Tail != "snapshot-eof" || len(page.Frames) != 2 {
		t.Fatal(page, e)
	}
	gap, e := ReadGap(page.Frames[1].Data)
	if e != nil || gap.Reason != "corrupt-tail" {
		t.Fatal("empty successor hid torn prefix", gap, e)
	}
}
func TestSegmentReaderRefusesExtraFilesLinksAndForeignCursor(t *testing.T) {
	for _, scenario := range []string{"extra", "link", "cursor"} {
		t.Run(scenario, func(t *testing.T) {
			s, dir := segmentStore(t, 1<<20, 4096)
			m := segmentedMetadata()
			w, e := s.Create(m)
			if e != nil {
				t.Fatal(e)
			}
			w.Append(Stdout, []byte("data"), m.AttachedAt.UnixNano(), 0)
			w.Close()
			page, e := s.ReadPage(t.Context(), m.SessionID, nil, 256, 1<<20)
			if e != nil {
				t.Fatal(e)
			}
			switch scenario {
			case "extra":
				e = os.WriteFile(filepath.Join(dir, m.SessionID, "unclassified"), []byte("x"), 0600)
			case "link":
				e = os.Link(filepath.Join(dir, m.SessionID, "frames.obr"), filepath.Join(t.TempDir(), "linked"))
			case "cursor":
				page.Next.Digest = hex.EncodeToString(bytes.Repeat([]byte{1}, 32))
			}
			if e != nil {
				t.Fatal(e)
			}
			_, e = s.ReadPage(t.Context(), m.SessionID, &page.Next, 256, 1<<20)
			if e == nil || errors.Is(e, io.EOF) {
				t.Fatal("invalid layout or cursor silently accepted", scenario, e)
			}
		})
	}
}

func TestSegmentedRecordingOriginalWriterQualification(t *testing.T) {
	root := os.Getenv("OMABIT_RECORDING_SEGMENT_UPGRADE_ROOT")
	if root == "" {
		t.Skip("explicit original-writer qualification only")
	}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || !strings.HasPrefix(root, "/tmp/omabit-recording-segments-upgrade-") {
		t.Fatal("explicit disposable source fixture required")
	}
	original := filepath.Join(root, "original")
	before := fileSnapshot(t, filepath.Join(original, strings.Repeat("1", 32)))
	s, e := OpenStore(original, Limits{MaxSessionBytes: 1 << 20, ReserveFreeBytes: 1 << 20})
	if e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 3; i++ {
		first, e := s.ReadPage(t.Context(), strings.Repeat("1", 32), nil, 1, 1<<20)
		if e != nil || first.Metadata.Version != 2 || len(first.Frames) != 1 || !bytes.Equal(first.Frames[0].Data, []byte("original stdout\r\n\x00\xff\x1b[2K")) {
			t.Fatal(first, e)
		}
		rest, e := s.ReadPage(t.Context(), first.Metadata.SessionID, &first.Next, 256, 1<<20)
		if e != nil || len(rest.Frames) != 3 || !bytes.Equal(rest.Frames[0].Data, []byte("original stderr\n0wfixture.atom")) {
			t.Fatal(rest, e)
		}
		if _, e = s.Continue(t.Context(), first.Metadata, time.Now()); !errors.Is(e, ErrLegacyLayout) {
			t.Fatal("legacy session continued into unreadable layout", e)
		}
		s.Close()
		s, e = OpenStore(original, Limits{MaxSessionBytes: 1 << 20, ReserveFreeBytes: 1 << 20})
		if e != nil {
			t.Fatal(e)
		}
	}
	s.Close()
	if !reflect.DeepEqual(before, fileSnapshot(t, filepath.Join(original, strings.Repeat("1", 32)))) {
		t.Fatal("new reader changed actual old writer fixture")
	}
	successor := filepath.Join(root, "successor")
	if e = os.Mkdir(successor, 0700); e != nil {
		t.Fatal(e)
	}
	s, e = OpenStore(successor, Limits{MaxSessionBytes: 1 << 20, MaxSegmentBytes: 4096, ReserveFreeBytes: 1 << 20})
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	m := segmentedMetadata()
	w, e := s.Create(m)
	if e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 5; i++ {
		if _, e = w.Append(Stdout, bytes.Repeat([]byte("successor\x00\xff"), 90), m.AttachedAt.UnixNano()+int64(i), uint64(i)); e != nil {
			t.Fatal(e)
		}
	}
	w.Close()
	w, e = s.Continue(t.Context(), m, m.AttachedAt.Add(time.Second))
	if e != nil {
		t.Fatal(e)
	}
	w.Close()
	page, e := s.ReadPage(t.Context(), m.SessionID, nil, 256, 1<<20)
	if e != nil || len(page.Frames) != 6 || page.Metadata.Version != 3 {
		t.Fatal(page, e)
	}
}

func TestSegmentWriterDeathHelper(t *testing.T) {
	root := os.Getenv("OMABIT_SEGMENT_DEATH_FIXTURE")
	if root == "" {
		t.Skip("owned child helper only")
	}
	s, e := OpenStore(root, Limits{MaxSessionBytes: 1 << 20, MaxSegmentBytes: 4096, ReserveFreeBytes: 1 << 20})
	if e != nil {
		t.Fatal(e)
	}
	m := segmentedMetadata()
	w, e := s.Create(m)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = w.Append(Stdout, []byte("durable before recorder death\x00\xff"), m.AttachedAt.UnixNano(), 100); e != nil {
		t.Fatal(e)
	}
	fmt.Println("OMABIT_SEGMENT_DURABLE")
	time.Sleep(30 * time.Second)
	os.Exit(91)
}

func TestSegmentsProcessDeathReleasesOnlyWriterLockAndPreservesDurablePrefix(t *testing.T) {
	s, dir := segmentStore(t, 1<<20, 4096)
	command := exec.Command(os.Args[0], "-test.run=^TestSegmentWriterDeathHelper$", "-test.v")
	command.Env = append(os.Environ(), "OMABIT_SEGMENT_DEATH_FIXTURE="+dir)
	stdout, e := command.StdoutPipe()
	if e != nil {
		t.Fatal(e)
	}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if e = command.Start(); e != nil {
		t.Fatal(e)
	}
	defer func() { command.Process.Kill(); command.Wait() }()
	ready := make(chan bool, 1)
	go func() {
		scan := bufio.NewScanner(stdout)
		for scan.Scan() {
			if scan.Text() == "OMABIT_SEGMENT_DURABLE" {
				ready <- true
				return
			}
		}
		ready <- false
	}()
	select {
	case ok := <-ready:
		if !ok {
			t.Fatal("owned writer did not acknowledge durable append")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("owned writer deadline")
	}
	m := segmentedMetadata()
	before := fileSnapshot(t, filepath.Join(dir, m.SessionID))
	if _, e = s.Continue(t.Context(), m, m.AttachedAt.Add(time.Second)); !errors.Is(e, ErrStorage) {
		t.Fatal("live child lost its writer lock", e)
	}
	if e = command.Process.Kill(); e != nil {
		t.Fatal(e)
	}
	if e = command.Wait(); e == nil {
		t.Fatal("writer death was mistaken for a clean exit")
	}
	w, e := s.Continue(t.Context(), m, m.AttachedAt.Add(time.Second))
	if e != nil {
		t.Fatal(e)
	}
	if e = w.Close(); e != nil {
		t.Fatal(e)
	}
	for name, data := range before {
		got, e := os.ReadFile(filepath.Join(dir, m.SessionID, name))
		if e != nil || !bytes.Equal(got, data) {
			t.Fatal("process-death recovery rewrote durable prefix", name, e)
		}
	}
	page, e := s.ReadPage(t.Context(), m.SessionID, nil, 256, 1<<20)
	if e != nil || page.WriterPresent || len(page.Frames) != 2 || page.Frames[0].Kind != Stdout || page.Frames[1].Kind != Gap {
		t.Fatal(page, e)
	}
	gap, e := ReadGap(page.Frames[1].Data)
	if e != nil || gap.Reason != "recorder-restarted" || gap.KnownMissingBytes != nil {
		t.Fatal("recorder death fabricated process exit or zero loss", gap, e)
	}
}

func TestSegmentsLiveCursorCrossesConcurrentRotationWithoutDuplicateFrames(t *testing.T) {
	s, _ := segmentStore(t, 1<<20, 4096)
	m := segmentedMetadata()
	w, e := s.Create(m)
	if e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	go func() {
		for i := 0; i < 200; i++ {
			if _, e := w.Append(Stdout, bytes.Repeat([]byte{byte(i)}, 900), m.AttachedAt.UnixNano()+int64(i), uint64(i)); e != nil {
				done <- e
				return
			}
		}
		done <- w.Close()
	}()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	var cursor *Cursor
	count := 0
	for count < 200 {
		page, e := s.ReadPage(ctx, m.SessionID, cursor, 7, 1<<20)
		if e != nil || page.Tail == "corrupt" {
			t.Fatal("concurrent rotation lost verified prefix", e, page.Tail)
		}
		for _, f := range page.Frames {
			if f.Sequence != uint64(count+1) || len(f.Data) != 900 || f.Data[0] != byte(count) {
				t.Fatal("duplicate or missing live frame", f.Sequence, count)
			}
			count++
		}
		next := page.Next
		cursor = &next
		if len(page.Frames) == 0 {
			runtime.Gosched()
		}
	}
	if e := <-done; e != nil {
		t.Fatal(e)
	}
}

func TestSegmentsContinueRechecksOriginalFileAfterReserveAdmission(t *testing.T) {
	s, dir := segmentStore(t, 1<<20, 4096)
	m := segmentedMetadata()
	w, e := s.Create(m)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = w.Append(Stdout, []byte("original"), m.AttachedAt.UnixNano(), 0); e != nil {
		t.Fatal(e)
	}
	w.Close()
	path := filepath.Join(dir, m.SessionID, "frames.obr")
	original, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	realFree := s.free
	s.free = func(root *os.Root, n uint64) error {
		if e := os.Rename(path, path+".retained"); e != nil {
			return e
		}
		if e := os.WriteFile(path, original, 0600); e != nil {
			return e
		}
		return realFree(root, n)
	}
	if _, e = s.Continue(t.Context(), m, m.AttachedAt.Add(time.Second)); !errors.Is(e, ErrStorage) {
		t.Fatal("continuation rebound a replaced verified original", e)
	}
	if _, e = os.Lstat(filepath.Join(dir, m.SessionID, segmentName(2))); !errors.Is(e, os.ErrNotExist) {
		t.Fatal("changed admission created a successor", e)
	}
	got, e := os.ReadFile(path + ".retained")
	if e != nil || !bytes.Equal(got, original) {
		t.Fatal("replacement refusal changed retained original", e)
	}
}

func TestSegmentsRequireExplicitGapBeforeResettingReceiveClock(t *testing.T) {
	for _, scenario := range []string{"new-epoch-without-gap", "backwards-same-epoch", "restart-with-nonzero-elapsed"} {
		t.Run(scenario, func(t *testing.T) {
			s, dir := segmentStore(t, 1<<20, 4096)
			m := segmentedMetadata()
			w, e := s.Create(m)
			if e != nil {
				t.Fatal(e)
			}
			first, e := w.Append(Stdout, []byte("prefix"), m.AttachedAt.UnixNano(), 100)
			if e != nil {
				t.Fatal(e)
			}
			w.Close()
			digest, _ := m.Digest()
			id, _ := hex.DecodeString(m.SessionID)
			h := Header{MetadataSHA256: digest, Segment: 2, FirstSequence: 2, StartedUnixNano: m.AttachedAt.UnixNano(), PreviousDigest: first.Digest}
			copy(h.SessionID[:], id)
			if scenario == "new-epoch-without-gap" {
				h.StartedUnixNano++
			}
			path := filepath.Join(dir, m.SessionID, segmentName(2))
			f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
			if e != nil {
				t.Fatal(e)
			}
			encoder, e := NewEncoder(f, h)
			if e != nil {
				t.Fatal(e)
			}
			kind := Stdout
			data := []byte("must not imply clock continuity")
			if scenario == "restart-with-nonzero-elapsed" {
				kind = Gap
				data, _ = (GapEvent{Version: 1, Reason: "recorder-restarted"}).Bytes()
			}
			if _, e = encoder.Append(kind, data, m.AttachedAt.UnixNano()+1, 1); e != nil {
				t.Fatal(e)
			}
			f.Close()
			before := fileSnapshot(t, filepath.Join(dir, m.SessionID))
			page, e := s.ReadPage(t.Context(), m.SessionID, nil, 256, 1<<20)
			if e != nil || page.Tail != "corrupt" || len(page.Frames) != 1 {
				t.Fatal("valid hashes hid invalid clock epoch", scenario, page, e)
			}
			if _, e = s.Continue(t.Context(), m, m.AttachedAt.Add(time.Second)); !errors.Is(e, ErrCorrupt) {
				t.Fatal("invalid clock history continued", e)
			}
			if !reflect.DeepEqual(before, fileSnapshot(t, filepath.Join(dir, m.SessionID))) {
				t.Fatal("clock refusal changed original")
			}
		})
	}
}

//go:build linux

package recording

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func pendingBootstrapMetadata() Metadata {
	m := captureMetadata()
	m.ProjectID = ""
	m.Identity.Target.ProjectID = ""
	m.Identity.Target.Generation = 0
	m.Identity.Target.Creator = ""
	m.Identity.ProjectNamespace = nil
	m.Identity.NativeShip = ""
	m.Identity.NativeShipStatus = "pending-comet-mining"
	m.Identity.PierBinding = "prepared-storage"
	return m
}

func TestMetadata4PreservesExplicitUnknownsAndLegacyCanonicalBytes(t *testing.T) {
	legacy := segmentedMetadata()
	before, err := legacy.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	old, err := ReadMetadata(before)
	after, encodeErr := old.Bytes()
	if err != nil || encodeErr != nil || !bytes.Equal(before, after) || old.Identity.NativeShipStatus != "" || old.Identity.PierBinding != "" {
		t.Fatal("legacy metadata was relabeled or changed", err, encodeErr)
	}
	for _, m := range []Metadata{captureMetadata(), pendingBootstrapMetadata()} {
		raw, err := m.Bytes()
		if err != nil {
			t.Fatal(err)
		}
		got, err := ReadMetadata(raw)
		if err != nil || !reflect.DeepEqual(got, m) {
			t.Fatal("explicit attachment basis was lost", got, err)
		}
		if m.Identity.NativeShip == "" && (!bytes.Contains(raw, []byte(`"native_ship":""`)) || !bytes.Contains(raw, []byte(`"native_ship_status":"pending-comet-mining"`)) || !bytes.Contains(raw, []byte(`"pier_binding":"prepared-storage"`))) {
			t.Fatal("unknown bootstrap acquired a ship or future pier object", string(raw))
		}
	}
}

func TestMetadata4RefusesInventedOrMixedBootstrapIdentityBeforeStorage(t *testing.T) {
	for _, scenario := range []string{"missing-ship-status", "claimed-confirmation", "missing-pier-basis", "unknown-pier-basis", "pending-with-name", "registered-without-name", "pending-project", "old-with-new-fields", "unknown-storage-digest"} {
		t.Run(scenario, func(t *testing.T) {
			m := pendingBootstrapMetadata()
			switch scenario {
			case "missing-ship-status":
				m.Identity.NativeShipStatus = ""
			case "claimed-confirmation":
				m.Identity.NativeShipStatus = "confirmed"
			case "missing-pier-basis":
				m.Identity.PierBinding = ""
			case "unknown-pier-basis":
				m.Identity.PierBinding = "future-pier-inode"
			case "pending-with-name":
				m.Identity.NativeShip = "~zod"
			case "registered-without-name":
				m.Identity.NativeShipStatus = "registered"
			case "pending-project":
				m.ProjectID, m.Identity.Target.ProjectID = "project", "project"
				m.Identity.Target.Generation, m.Identity.Target.Creator = 9, "creator"
				m.Identity.ProjectNamespace = &ProjectNamespace{BootID: m.Identity.HostBootID, Device: 4, Inode: 200}
			case "old-with-new-fields":
				m = captureMetadata()
				m.Version = 3
			case "unknown-storage-digest":
				m.Identity.PierContinuitySHA256 = strings.Repeat("0", 64)
			}
			s, dir := openTestStore(t, 1<<20)
			if m.Validate() == nil {
				t.Fatal("invalid bootstrap basis was accepted")
			}
			if _, err := s.Create(m); !errors.Is(err, ErrInvalid) {
				t.Fatal("invalid bootstrap basis reached persistence", err)
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 0 {
				t.Fatal("refused bootstrap changed the storage root", err)
			}
		})
	}
}

func TestMetadata4UnknownBootstrapSurvivesSegmentsGapExitAndReopen(t *testing.T) {
	s, dir := segmentStore(t, 1<<20, 4096)
	m := pendingBootstrapMetadata()
	original, _ := m.Bytes()
	w, err := s.Create(m)
	if err != nil {
		t.Fatal(err)
	}
	var expected []byte
	for i := 0; i < 12; i++ {
		data := bytes.Repeat([]byte{byte(i)}, 900)
		expected = append(expected, data...)
		if _, err = w.Append(Stdout, data, m.AttachedAt.UnixNano()+int64(i), uint64(i)); err != nil {
			t.Fatal(err)
		}
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	first, err := s.ReadPage(t.Context(), m.SessionID, nil, 3, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	w, err = s.Continue(t.Context(), m, m.AttachedAt.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	exit, _ := (ExitEvent{Version: 1, ContainerID: m.ContainerID, StartedUnixNano: m.AttachedAt.UnixNano(), FinishedUnixNano: m.AttachedAt.Add(time.Second).UnixNano(), Code: 1}).Bytes()
	if _, err = w.Append(Exit, exit, m.AttachedAt.Add(time.Second).UnixNano(), 1); err != nil {
		t.Fatal(err)
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = OpenStore(dir, Limits{MaxSessionBytes: 1 << 20, MaxSegmentBytes: 4096, ReserveFreeBytes: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	rest, err := s.ReadPage(t.Context(), m.SessionID, &first.Next, 256, 1<<20)
	if err != nil || rest.WriterPresent || rest.Metadata.Version != 4 || rest.Metadata.Identity.NativeShip != "" || rest.Metadata.Identity.NativeShipStatus != "pending-comet-mining" || rest.Metadata.Identity.PierBinding != "prepared-storage" {
		t.Fatal("continuation/exit invented a bootstrap identity", rest.Metadata, err)
	}
	var raw []byte
	for _, frame := range append(first.Frames, rest.Frames...) {
		if frame.Kind == Stdout {
			raw = append(raw, frame.Data...)
		}
	}
	if !bytes.Equal(raw, expected) {
		t.Fatal("bootstrap raw bytes changed across cursor/reopen")
	}
	after, err := os.ReadFile(filepath.Join(dir, m.SessionID, "metadata.json"))
	if err != nil || !bytes.Equal(original, after) {
		t.Fatal("bootstrap metadata was rewritten after process exit", err)
	}
}

func TestCaptureCanPreserveBootstrapBytesBeforeShipOrPierExists(t *testing.T) {
	s, _ := openTestStore(t, 1<<20)
	m := pendingBootstrapMetadata()
	data := append(bytes.Repeat([]byte("mining and bootstrap trace\n"), 350), []byte("\x1b[2K\r0wdiagnostic.atom\x00\xff")...)
	ready := false
	out, err := s.CaptureDocker(t.Context(), m, DockerCaptureSource{
		Attach: func(context.Context) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(capturePacket(1, data))), nil
		},
		Ready: func(bound Metadata) error {
			ready = true
			if bound.Identity.NativeShip != "" || bound.Identity.NativeShipStatus != "pending-comet-mining" || bound.Identity.PierBinding != "prepared-storage" {
				t.Fatal("readiness fabricated future identity or storage")
			}
			return nil
		},
		ObserveExit: func(context.Context) (*ExitEvent, error) { return nil, nil },
	})
	if !errors.Is(err, ErrCaptureIncomplete) || !ready || !out.GapSaved || out.GapReason != "process-observation-unavailable" || out.ExitSaved || !out.ClosedDurably {
		t.Fatal("unknown bootstrap or EOF became completed process evidence", out, err)
	}
	page, err := s.ReadPage(t.Context(), m.SessionID, nil, 256, 1<<20)
	if err != nil || page.Metadata.Identity.NativeShip != "" || len(page.Frames) != 2 || !bytes.Equal(page.Frames[0].Data, data) || page.Frames[1].Kind != Gap {
		t.Fatal("early bootstrap originals or explicit uncertainty were lost", err)
	}
}

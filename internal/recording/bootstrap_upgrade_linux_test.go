//go:build linux

package recording

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// The separately compiled frozen3 writer supplies the original. Ordinary tests
// never synthesize original-writer provenance or open an installed recording.
func TestBootstrapRecordingOriginalWriterQualification(t *testing.T) {
	root := os.Getenv("OMABIT_RECORDING_BOOTSTRAP_UPGRADE_ROOT")
	if root == "" {
		t.Skip("explicit actual original3 writer qualification only")
	}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || !strings.HasPrefix(root, "/tmp/omabit-recording-bootstrap-upgrade-") {
		t.Fatal("explicit fresh source qualification path required")
	}
	snapshot := func(dir string) map[string]string {
		t.Helper()
		files := map[string]string{}
		err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			if !entry.Type().IsRegular() {
				return ErrStorage
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			d := sha256.Sum256(data)
			name, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			files[name] = hex.EncodeToString(d[:])
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		return files
	}
	original := filepath.Join(root, "original")
	before := snapshot(original)
	id := strings.Repeat("1", 32)
	limits := Limits{MaxSessionBytes: 1 << 20, MaxSegmentBytes: 4096, ReserveFreeBytes: 1 << 20}
	for i := 0; i < 3; i++ {
		s, err := OpenStore(original, limits)
		if err != nil {
			t.Fatal(err)
		}
		full, err := s.ReadPage(t.Context(), id, nil, 256, 1<<20)
		if err != nil || len(full.Frames) != 12 || full.WriterPresent || full.Metadata.Version != 3 || full.Metadata.Identity.NativeShipStatus != "" || full.Metadata.Identity.PierBinding != "" || full.Frames[len(full.Frames)-1].Segment < 3 {
			t.Fatal("actual original3 segmented bytes or omitted identity basis changed", err)
		}
		if !bytes.Equal(full.Frames[0].Data, []byte("original stdout\r\n\x00\xff\x1b[2K")) || !bytes.Equal(full.Frames[1].Data, []byte("original stderr\n0wfixture.atom")) {
			t.Fatal("actual original3 raw bytes changed")
		}
		first, err := s.ReadPage(t.Context(), id, nil, 2, 1<<20)
		if err != nil {
			t.Fatal(err)
		}
		rest, err := s.ReadPage(t.Context(), id, &first.Next, 256, 1<<20)
		if err != nil || !reflect.DeepEqual(rest.Frames, full.Frames[2:]) {
			t.Fatal("actual original3 cursor changed", err)
		}
		if _, err = s.Continue(t.Context(), full.Metadata, time.Now()); !errors.Is(err, ErrInvalid) {
			t.Fatal("completed original3 silently continued or changed identity", err)
		}
		if err = s.Close(); err != nil || !reflect.DeepEqual(snapshot(original), before) {
			t.Fatal("actual original3 files changed during reopen/read/refusal", err)
		}
	}
	successor := filepath.Join(root, "successor")
	if err := os.Mkdir(successor, 0700); err != nil {
		t.Fatal(err)
	}
	s, err := OpenStore(successor, limits)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	m := pendingBootstrapMetadata()
	w, err := s.Create(m)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = w.Append(Stdout, []byte("bootstrap before ship/pier identity\n\x00\xff"), m.AttachedAt.UnixNano(), 0); err != nil {
		t.Fatal(err)
	}
	if err = w.Rotate(); err != nil {
		t.Fatal(err)
	}
	if _, err = w.Append(Stderr, []byte("second source segment\n"), m.AttachedAt.UnixNano()+1, 1); err != nil {
		t.Fatal(err)
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	w, err = s.Continue(t.Context(), m, m.AttachedAt.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	exit, _ := (ExitEvent{Version: 1, ContainerID: m.ContainerID, StartedUnixNano: m.AttachedAt.UnixNano(), FinishedUnixNano: m.AttachedAt.Add(time.Second).UnixNano(), Code: 1}).Bytes()
	if _, err = w.Append(Exit, exit, m.AttachedAt.Add(time.Second).UnixNano()+1, 1); err != nil {
		t.Fatal(err)
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	page, err := s.ReadPage(t.Context(), id, nil, 256, 1<<20)
	if err != nil || page.Metadata.Version != 4 || page.Metadata.Identity.NativeShip != "" || page.Metadata.Identity.NativeShipStatus != "pending-comet-mining" || page.Metadata.Identity.PierBinding != "prepared-storage" || len(page.Frames) != 4 || page.Frames[3].Segment != 3 {
		t.Fatal("successor bootstrap uncertainty or segment ancestry changed", err)
	}
}

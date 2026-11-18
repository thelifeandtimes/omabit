//go:build linux

package recording

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Only an explicit source qualification invokes this gate. Ordinary unit tests
// do not invent an original writer or touch any installed recording root.
func TestRecordingMetadataOriginalWriterQualification(t *testing.T) {
	root := os.Getenv("OMABIT_RECORDING_UPGRADE_ROOT")
	if root == "" {
		t.Skip("explicit original-writer qualification only")
	}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || !strings.HasPrefix(root, "/tmp/omabit-recording-upgrade-") {
		t.Fatal("explicit disposable qualification root required")
	}
	snapshot := func(dir string) map[string]string {
		t.Helper()
		out := map[string]string{}
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
			sum := sha256.Sum256(data)
			relative, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			out[relative] = hex.EncodeToString(sum[:])
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	original := filepath.Join(root, "original")
	before := snapshot(original)
	if len(before) != 3 {
		t.Fatal("unexpected original store layout")
	}
	s, err := OpenStore(original, Limits{MaxSessionBytes: 1 << 20, ReserveFreeBytes: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		page, e := s.ReadPage(t.Context(), strings.Repeat("1", 32), nil, 256, 1<<20)
		if e != nil || page.Metadata.Version != 1 || page.Metadata.Identity != nil || len(page.Frames) != 4 || page.WriterPresent {
			t.Fatal("actual original store failed preserved read", e)
		}
		if !bytes.Equal(page.Frames[0].Data, []byte("original stdout\r\n\x00\xff\x1b[2K")) || !bytes.Equal(page.Frames[1].Data, []byte("original stderr\n0wfixture.atom")) {
			t.Fatal("original arbitrary bytes changed")
		}
		gap, e := ReadGap(page.Frames[2].Data)
		if e != nil || gap.Reason != "transport-interrupted" {
			t.Fatal("original gap changed")
		}
		exit, e := ReadExit(page.Frames[3].Data)
		if e != nil || exit.Code != 7 {
			t.Fatal("original terminal evidence changed")
		}
		if _, e = s.Create(page.Metadata); !errors.Is(e, ErrLegacyMetadata) {
			t.Fatal("legacy metadata admitted new capture", e)
		}
		first, e := s.ReadPage(t.Context(), page.Metadata.SessionID, nil, 1, 1<<20)
		if e != nil || len(first.Frames) != 1 {
			t.Fatal("original cursor prefix unavailable", e)
		}
		rest, e := s.ReadPage(t.Context(), page.Metadata.SessionID, &first.Next, 256, 1<<20)
		if e != nil || len(rest.Frames) != 3 || !reflect.DeepEqual(rest.Frames, page.Frames[1:]) {
			t.Fatal("original cursor resume changed bytes or digest chain", e)
		}
		if err = s.Close(); err != nil {
			t.Fatal(err)
		}
		s, err = OpenStore(original, Limits{MaxSessionBytes: 1 << 20, ReserveFreeBytes: 1 << 20})
		if err != nil {
			t.Fatal(err)
		}
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	after := snapshot(original)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("new reader rewrote original store")
	}
	successor := filepath.Join(root, "successor")
	if err = os.Mkdir(successor, 0700); err != nil {
		t.Fatal(err)
	}
	s, err = OpenStore(successor, Limits{MaxSessionBytes: 1 << 20, ReserveFreeBytes: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	m := attachmentMetadata()
	w, err := s.Create(m)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = w.Append(Stdout, []byte("successor original bytes\x00\xff"), m.AttachedAt.UnixNano(), 0); err != nil {
		t.Fatal(err)
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	report := struct {
		Version                   int               `json:"version"`
		OriginalReadOnlyPreserved bool              `json:"original_read_only_preserved"`
		Original                  map[string]string `json:"original_files"`
		Successor                 map[string]string `json:"successor_files"`
	}{1, true, before, snapshot(successor)}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "candidate-report.json"), append(raw, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}

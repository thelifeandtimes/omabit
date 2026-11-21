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

func TestActualOriginalMetadata1FixtureRemainsReadOnlyAndByteExact(t *testing.T) {
	fixture := filepath.Join("testdata", "original-metadata1")
	var manifest struct {
		Version              int               `json:"version"`
		OriginalSource       string            `json:"original_source"`
		OriginalWriterSHA256 string            `json:"original_writer_sha256"`
		SourceManifestSHA256 string            `json:"original_source_manifest_sha256"`
		Files                map[string]string `json:"files"`
	}
	raw, err := os.ReadFile(filepath.Join(fixture, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if json.Unmarshal(raw, &manifest) != nil || manifest.Version != 1 || manifest.OriginalSource != "3218e6fb87ef293cc87d9083c574b3232de09271" || manifest.OriginalWriterSHA256 != "0b27ba1be956f24e6ebcfb0de417ede59b84c4b48cd8534f96357f91454ef843" || len(manifest.Files) != 3 {
		t.Fatal("original fixture provenance changed")
	}
	source, err := os.ReadFile(filepath.Join(fixture, "original-source.json"))
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(source)
	if hex.EncodeToString(digest[:]) != manifest.SourceManifestSHA256 {
		t.Fatal("original source input manifest changed")
	}
	store, dir := openTestStore(t, 1<<20)
	id := strings.Repeat("1", 32)
	session := filepath.Join(dir, id)
	if err = os.Mkdir(session, 0700); err != nil {
		t.Fatal(err)
	}
	before := map[string][]byte{}
	for name, pin := range manifest.Files {
		if name != "metadata.json" && name != "frames.obr" && name != "writer.lock" {
			t.Fatal("unexpected fixture filename")
		}
		data, err := os.ReadFile(filepath.Join(fixture, name))
		if err != nil {
			t.Fatal(err)
		}
		d := sha256.Sum256(data)
		if hex.EncodeToString(d[:]) != pin {
			t.Fatal("actual original writer bytes changed", name)
		}
		before[name] = data
		if err = os.WriteFile(filepath.Join(session, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	page, err := store.ReadPage(t.Context(), id, nil, 256, 1<<20)
	if err != nil || page.Metadata.Version != 1 || page.Metadata.Identity != nil || len(page.Frames) != 4 || page.WriterPresent {
		t.Fatal("actual original fixture unreadable", err)
	}
	if !bytes.Equal(page.Frames[0].Data, []byte("original stdout\r\n\x00\xff\x1b[2K")) || !bytes.Equal(page.Frames[1].Data, []byte("original stderr\n0wfixture.atom")) {
		t.Fatal("actual original octets changed")
	}
	if _, err = page.Metadata.CaptureIdentity(); !errors.Is(err, ErrLegacyMetadata) {
		t.Fatal("actual legacy fixture invented a current identity", err)
	}
	if _, err = store.Create(page.Metadata); !errors.Is(err, ErrLegacyMetadata) {
		t.Fatal("actual legacy fixture admitted new capture", err)
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = OpenStore(dir, Limits{MaxSessionBytes: 1 << 20, ReserveFreeBytes: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	first, err := store.ReadPage(t.Context(), id, nil, 1, 1<<20)
	if err != nil || len(first.Frames) != 1 {
		t.Fatal("actual original first page unavailable", err)
	}
	rest, err := store.ReadPage(t.Context(), id, &first.Next, 256, 1<<20)
	if err != nil || !reflect.DeepEqual(rest.Frames, page.Frames[1:]) {
		t.Fatal("actual original cursor changed after reopen", err)
	}
	for name, want := range before {
		got, err := os.ReadFile(filepath.Join(session, name))
		if err != nil || !bytes.Equal(got, want) {
			t.Fatal("reader rewrote actual original file", name, err)
		}
	}
}

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

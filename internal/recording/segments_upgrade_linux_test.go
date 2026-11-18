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
	"time"
)

func TestActualOriginalMetadata2FixtureRetainsExactSingleSegmentReads(t *testing.T) {
	fixture := filepath.Join("testdata", "original-metadata2")
	var manifest struct {
		Version        int               `json:"version"`
		Source         string            `json:"original_source"`
		Writer         string            `json:"original_writer_sha256"`
		SourceManifest string            `json:"original_source_manifest_sha256"`
		Files          map[string]string `json:"files"`
	}
	raw, e := os.ReadFile(filepath.Join(fixture, "manifest.json"))
	if e != nil {
		t.Fatal(e)
	}
	if json.Unmarshal(raw, &manifest) != nil || manifest.Version != 1 || manifest.Source != "2eb1b67a7e42ac8eb8ae428ac9092cbd61a50507" || manifest.Writer != "f6c51e94537e62031c0e29b1bf867bcef65eab7b5a4d518604aa6e895884e335" || manifest.SourceManifest != "908608c7e5d76e33d36a6010e85a7569bb07b21b38d9911bf7ada2d923f23f2d" || len(manifest.Files) != 3 {
		t.Fatal("actual original2 provenance changed")
	}
	source, e := os.ReadFile(filepath.Join(fixture, "original-source.json"))
	if e != nil {
		t.Fatal(e)
	}
	sum := sha256.Sum256(source)
	if hex.EncodeToString(sum[:]) != manifest.SourceManifest {
		t.Fatal("original source manifest changed")
	}
	s, dir := openTestStore(t, 1<<20)
	id := strings.Repeat("1", 32)
	root := filepath.Join(dir, id)
	if e = os.Mkdir(root, 0700); e != nil {
		t.Fatal(e)
	}
	for name, pin := range manifest.Files {
		if name != "metadata.json" && name != "writer.lock" && name != "frames.obr" {
			t.Fatal("unexpected original file")
		}
		data, e := os.ReadFile(filepath.Join(fixture, name))
		if e != nil {
			t.Fatal(e)
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != pin {
			t.Fatal("actual original writer bytes changed", name)
		}
		if e = os.WriteFile(filepath.Join(root, name), data, 0600); e != nil {
			t.Fatal(e)
		}
	}
	before := fileSnapshot(t, root)
	full, e := s.ReadPage(t.Context(), id, nil, 256, 1<<20)
	if e != nil || full.Metadata.Version != 2 || len(full.Frames) != 4 || full.WriterPresent || full.Tail != "snapshot-eof" {
		t.Fatal(full, e)
	}
	if !bytes.Equal(full.Frames[0].Data, []byte("original stdout\r\n\x00\xff\x1b[2K")) || !bytes.Equal(full.Frames[1].Data, []byte("original stderr\n0wfixture.atom")) {
		t.Fatal("actual original octets changed")
	}
	identity, e := full.Metadata.CaptureIdentity()
	if e != nil || identity.NativeShip != "~zod" || identity.Target.Generation != 9 {
		t.Fatal("actual original identity changed", e)
	}
	gap, e := ReadGap(full.Frames[2].Data)
	if e != nil || gap.Reason != "transport-interrupted" {
		t.Fatal("actual original missing coverage changed", e)
	}
	exit, e := ReadExit(full.Frames[3].Data)
	if e != nil || exit.Code != 7 {
		t.Fatal("actual original terminal evidence changed", e)
	}
	for i := 0; i < 3; i++ {
		first, e := s.ReadPage(t.Context(), id, nil, 1, 1<<20)
		if e != nil || len(first.Frames) != 1 {
			t.Fatal(e)
		}
		rest, e := s.ReadPage(t.Context(), id, &first.Next, 256, 1<<20)
		if e != nil || !reflect.DeepEqual(rest.Frames, full.Frames[1:]) {
			t.Fatal("actual original cursor changed", e)
		}
		for _, f := range rest.Frames {
			if f.Segment != 1 || f.EpochUnixNano != full.Metadata.AttachedAt.UnixNano() {
				t.Fatal("original read invented a new segment/epoch")
			}
		}
		if _, e = s.Continue(t.Context(), full.Metadata, time.Now()); !errors.Is(e, ErrLegacyLayout) {
			t.Fatal("original2 silently continued into new layout", e)
		}
		s.Close()
		s, e = OpenStore(dir, Limits{MaxSessionBytes: 1 << 20, ReserveFreeBytes: 1 << 20})
		if e != nil {
			t.Fatal(e)
		}
	}
	defer s.Close()
	if !reflect.DeepEqual(before, fileSnapshot(t, root)) {
		t.Fatal("new reader/refused continuation changed original2 files")
	}
}

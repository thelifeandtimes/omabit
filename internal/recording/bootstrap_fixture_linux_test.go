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

func TestActualOriginalMetadata3FixtureRetainsSegmentsAndUnknownBasis(t *testing.T) {
	fixture := filepath.Join("testdata", "original-metadata3")
	var manifest struct {
		Version        int               `json:"version"`
		Source         string            `json:"original_source"`
		Writer         string            `json:"original_writer_sha256"`
		SourceManifest string            `json:"original_source_manifest_sha256"`
		Helper         string            `json:"declared_helper_sha256"`
		Files          map[string]string `json:"files"`
	}
	raw, err := os.ReadFile(filepath.Join(fixture, "manifest.json"))
	if err != nil || json.Unmarshal(raw, &manifest) != nil || manifest.Version != 1 || manifest.Source != "6e810bc937e57a9f5ff521c8957502fdceb71bb5" || manifest.Writer != "647ece5d9da0b1da08ac4e6b40860268c7e9592291a743f96db3f3d2f95e48ad" || manifest.SourceManifest != "764dc8f33483369db87c8b8716b7363419322ef8aaea77ccfc629391d38f3c13" || manifest.Helper != "6bd1c61b4a671b65bda250fb3d5e5bcc6a1a65d6444094387de0493ea82083d0" || len(manifest.Files) != 6 {
		t.Fatal("actual original3 provenance changed", err)
	}
	readPinned := func(name, pin string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(fixture, name))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != pin {
			t.Fatal("actual original source/bytes changed", name)
		}
		return data
	}
	readPinned("original-source.json", manifest.SourceManifest)
	readPinned("declared-original-writer.go.txt", manifest.Helper)
	s, dir := segmentStore(t, 1<<20, 4096)
	id := strings.Repeat("1", 32)
	root := filepath.Join(dir, id)
	if err = os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	for name, pin := range manifest.Files {
		switch name {
		case "metadata.json", "writer.lock", "frames.obr", "frames-00000002.obr", "frames-00000003.obr", "frames-00000004.obr":
		default:
			t.Fatal("unexpected original writer file")
		}
		if err = os.WriteFile(filepath.Join(root, name), readPinned(name, pin), 0600); err != nil {
			t.Fatal(err)
		}
	}
	before := fileSnapshot(t, root)
	full, err := s.ReadPage(t.Context(), id, nil, 256, 1<<20)
	if err != nil || full.Metadata.Version != 3 || full.Metadata.Identity.NativeShipStatus != "" || full.Metadata.Identity.PierBinding != "" || len(full.Frames) != 12 || full.WriterPresent || full.Frames[0].Segment != 1 || full.Frames[11].Segment != 4 {
		t.Fatal("actual original3 segments/unknown basis changed", err)
	}
	if !bytes.Equal(full.Frames[0].Data, []byte("original stdout\r\n\x00\xff\x1b[2K")) || !bytes.Equal(full.Frames[1].Data, []byte("original stderr\n0wfixture.atom")) {
		t.Fatal("original raw octets changed")
	}
	gap, err := ReadGap(full.Frames[10].Data)
	if err != nil || full.Frames[10].Kind != Gap || gap.Reason != "recorder-restarted" || full.Frames[10].EpochUnixNano == full.Frames[0].EpochUnixNano {
		t.Fatal("original continuation invented continuous coverage", err)
	}
	exit, err := ReadExit(full.Frames[11].Data)
	if err != nil || full.Frames[11].Kind != Exit || exit.Code != 7 || exit.ContainerID != full.Metadata.ContainerID {
		t.Fatal("original terminal evidence changed", err)
	}
	for range 3 {
		first, err := s.ReadPage(t.Context(), id, nil, 5, 1<<20)
		if err != nil {
			t.Fatal(err)
		}
		rest, err := s.ReadPage(t.Context(), id, &first.Next, 256, 1<<20)
		if err != nil || !reflect.DeepEqual(rest.Frames, full.Frames[5:]) {
			t.Fatal("original cross-segment cursor changed", err)
		}
		if _, err = s.Continue(t.Context(), full.Metadata, time.Now()); !errors.Is(err, ErrInvalid) {
			t.Fatal("completed original3 silently continued", err)
		}
		if err = s.Close(); err != nil {
			t.Fatal(err)
		}
		s, err = OpenStore(dir, Limits{MaxSessionBytes: 1 << 20, MaxSegmentBytes: 4096, ReserveFreeBytes: 1 << 20})
		if err != nil {
			t.Fatal(err)
		}
	}
	defer s.Close()
	if !reflect.DeepEqual(before, fileSnapshot(t, root)) {
		t.Fatal("new reader/reopen/refused continuation changed actual original3")
	}
}

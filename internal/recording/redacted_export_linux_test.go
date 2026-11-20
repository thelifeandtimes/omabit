//go:build linux

package recording

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRedactedManifestPartsAssembleExactDerivedZIPWithoutOriginalFallback(t *testing.T) {
	s, root, in, result := redactedBundleFixture(t)
	before := fileSnapshot(t, root)
	manifest, err := PlanRedactedExport(t.Context(), in, result, s.ExportPart)
	if err != nil || manifest.Validate() != nil {
		t.Fatal("derived manifest failed", err)
	}
	var direct bytes.Buffer
	_, err = WriteRedactedBundle(t.Context(), in, result, s.ExportPart, nil, &direct)
	if err != nil {
		t.Fatal(err)
	}
	var assembled bytes.Buffer
	fetch := func(ctx context.Context, req RedactedExportPartRequest) (RedactedExportPart, error) {
		return ReadRedactedExportPart(ctx, req, result, s.ExportPart)
	}
	if err = WriteRedactedExportParts(t.Context(), in, manifest, fetch, &assembled); err != nil || !bytes.Equal(assembled.Bytes(), direct.Bytes()) {
		t.Fatal("derived chunk assembly differs", err)
	}
	result.Manifest.ReadAt = result.Manifest.ReadAt.Add(time.Minute)
	result.ManifestSHA256, err = result.Manifest.Digest()
	if err != nil {
		t.Fatal(err)
	}
	again, err := PlanRedactedExport(t.Context(), in, result, s.ExportPart)
	if err != nil || !reflect.DeepEqual(again, manifest) {
		t.Fatal("volatile read time changed derived bytes", err)
	}
	base := RedactedExportPartRequest{RedactedExportRequest: in, SourceSHA256: manifest.SourceSHA256, ZIPBytes: manifest.ZIPBytes, ZIPSHA256: manifest.ZIPSHA256, ChunkSHA256: manifest.Chunks[0]}
	for _, variant := range []string{"source", "zip", "size", "chunk", "original"} {
		req := base
		switch variant {
		case "source":
			req.SourceSHA256 = strings.Repeat("f", 64)
		case "zip":
			req.ZIPSHA256 = strings.Repeat("f", 64)
		case "size":
			req.ZIPBytes++
		case "chunk":
			req.ChunkSHA256 = strings.Repeat("f", 64)
		case "original":
			req.Representation = "original"
		}
		got, err := ReadRedactedExportPart(t.Context(), req, result, s.ExportPart)
		if err == nil || len(got.Data) != 0 {
			t.Fatal("changed derived selection leaked chunk", variant)
		}
	}
	if !reflect.DeepEqual(before, fileSnapshot(t, root)) {
		t.Fatal("derived manifest/parts changed original source")
	}
}

func TestRedactedZIPHashKeepsOnlySelectedChunkAcrossWriterBoundaries(t *testing.T) {
	data := make([]byte, 3*ExportChunkBytes+17)
	for i := range data {
		data[i] = byte(i*31 + i/257)
	}
	w := newRedactedZIPHash(1)
	for offset := 0; offset < len(data); {
		n := min(2059, len(data)-offset)
		if written, err := w.Write(data[offset : offset+n]); err != nil || written != n {
			t.Fatal(err)
		}
		offset += n
	}
	m := w.finish(RedactedBundleSummary{})
	h := sha256.Sum256(data)
	if m.ZIPSHA256 != hex.EncodeToString(h[:]) || m.ZIPBytes != uint64(len(data)) || len(m.Chunks) != 4 || !bytes.Equal(w.part, data[ExportChunkBytes:2*ExportChunkBytes]) {
		t.Fatal("chunk hash boundaries wrong")
	}
	for i, digest := range m.Chunks {
		h := sha256.Sum256(data[i*ExportChunkBytes : min((i+1)*ExportChunkBytes, len(data))])
		if digest != hex.EncodeToString(h[:]) {
			t.Fatal("independent chunk SHA mismatch")
		}
	}
}

package recording

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
)

type RedactedExportManifest struct {
	Version        int                   `json:"version"`
	Representation string                `json:"representation"`
	SourceSHA256   string                `json:"source_sha256"`
	ZIPBytes       uint64                `json:"zip_bytes"`
	ZIPSHA256      string                `json:"zip_sha256"`
	ChunkBytes     int                   `json:"chunk_bytes"`
	Chunks         []string              `json:"chunk_sha256"`
	Summary        RedactedBundleSummary `json:"summary"`
}

func (m RedactedExportManifest) Validate() error {
	if m.Version != 1 || m.Representation != "redacted" || !APIHex(m.SourceSHA256, 64) || !APIHex(m.ZIPSHA256, 64) || m.ZIPBytes == 0 || m.ZIPBytes > MaxRedactedBundleBytes || m.ChunkBytes != ExportChunkBytes || uint64(len(m.Chunks)) != (m.ZIPBytes+ExportChunkBytes-1)/ExportChunkBytes || m.Summary.Representation != "redacted" || m.Summary.Version != 1 || m.Summary.SourceSHA256 != m.SourceSHA256 || !APIHex(m.Summary.FramesSHA256, 64) || !m.Summary.AllSourceFilesVerified || !m.Summary.VerifiedPrefixTransformed || !m.Summary.ExactBindingsOmitted || m.Summary.UnparsedRawBytesIncluded || m.Summary.RuntimeComplete {
		return ErrSelection
	}
	s, c := m.Summary, m.Summary.Coverage
	if s.Frames != c.VerifiedFrames || s.Frames > MaxTimingIndexFrames || s.OmittedPayloadBytes > s.OriginalPayloadBytes || s.OriginalPayloadBytes < c.OutputBytes || s.DerivedBytes == 0 || s.DerivedBytes > MaxRedactedBundleBytes || c.GapFrames > c.VerifiedFrames || s.ExitBindingsAliased > 1 || s.ExitBindingsAliased != 0 && !c.ExitSaved || c.ExitSaved && s.ExitBindingsAliased != 1 {
		return ErrSelection
	}
	if c.VerifiedFrames == 0 && (c.FirstUnixNano != 0 || c.LastUnixNano != 0 || c.OutputBytes != 0 || c.ExitSaved) || c.VerifiedFrames > 0 && (c.FirstUnixNano <= 0 || c.LastUnixNano <= 0) {
		return ErrSelection
	}
	if c.Tail != "snapshot-eof" && c.Tail != "incomplete" && c.Tail != "corrupt" {
		return ErrSelection
	}
	for _, h := range m.Chunks {
		if !APIHex(h, 64) {
			return ErrSelection
		}
	}
	return nil
}

type RedactedExportPartRequest struct {
	RedactedExportRequest
	SourceSHA256 string `json:"source_sha256"`
	ZIPBytes     uint64 `json:"zip_bytes"`
	ZIPSHA256    string `json:"zip_sha256"`
	ChunkIndex   int    `json:"chunk_index"`
	ChunkSHA256  string `json:"chunk_sha256"`
}

func (r RedactedExportPartRequest) Validate() error {
	if r.RedactedExportRequest.Validate() != nil || !APIHex(r.SourceSHA256, 64) || !APIHex(r.ZIPSHA256, 64) || r.ZIPBytes == 0 || r.ZIPBytes > MaxRedactedBundleBytes || r.ChunkIndex < 0 || uint64(r.ChunkIndex) >= (r.ZIPBytes+ExportChunkBytes-1)/ExportChunkBytes || !APIHex(r.ChunkSHA256, 64) {
		return ErrSelection
	}
	return nil
}

type RedactedExportPart struct {
	Version int                       `json:"version"`
	Request RedactedExportPartRequest `json:"request"`
	Data    []byte                    `json:"data"`
}

func (p RedactedExportPart) Validate() error {
	if p.Version != 1 || p.Request.Validate() != nil || uint64(len(p.Data)) != min(uint64(ExportChunkBytes), p.Request.ZIPBytes-uint64(p.Request.ChunkIndex)*ExportChunkBytes) {
		return ErrSelection
	}
	h := sha256.Sum256(p.Data)
	if hex.EncodeToString(h[:]) != p.Request.ChunkSHA256 {
		return ErrSelection
	}
	return nil
}

// Bounded streaming ZIP hashing, optionally retaining exactly one chosen chunk.
// No generated export or unredacted original is written to temporary storage.
type redactedZIPHash struct {
	whole      hash.Hash
	chunk      hash.Hash
	bytes      uint64
	chunkBytes int
	chunks     []string
	selected   int
	part       []byte
}

func newRedactedZIPHash(selected int) *redactedZIPHash {
	return &redactedZIPHash{whole: sha256.New(), chunk: sha256.New(), selected: selected}
}
func (w *redactedZIPHash) Write(p []byte) (int, error) {
	if uint64(len(p)) > MaxRedactedBundleBytes-w.bytes {
		return 0, ErrLimit
	}
	count := len(p)
	for len(p) > 0 {
		n := min(len(p), ExportChunkBytes-w.chunkBytes)
		data := p[:n]
		w.whole.Write(data)
		w.chunk.Write(data)
		if len(w.chunks) == w.selected {
			w.part = append(w.part, data...)
		}
		w.bytes += uint64(n)
		w.chunkBytes += n
		p = p[n:]
		if w.chunkBytes == ExportChunkBytes {
			w.chunks = append(w.chunks, hex.EncodeToString(w.chunk.Sum(nil)))
			w.chunk.Reset()
			w.chunkBytes = 0
		}
	}
	return count, nil
}
func (w *redactedZIPHash) finish(summary RedactedBundleSummary) RedactedExportManifest {
	if w.chunkBytes > 0 {
		w.chunks = append(w.chunks, hex.EncodeToString(w.chunk.Sum(nil)))
	}
	return RedactedExportManifest{Version: 1, Representation: "redacted", SourceSHA256: summary.SourceSHA256, ZIPBytes: w.bytes, ZIPSHA256: hex.EncodeToString(w.whole.Sum(nil)), ChunkBytes: ExportChunkBytes, Chunks: w.chunks, Summary: summary}
}

// PlanRedactedExport computes a complete deterministic default-policy derived
// ZIP manifest. It exposes no raw binding names, original files or local paths.
func PlanRedactedExport(ctx context.Context, in RedactedExportRequest, result ExportManifestResult, fetch func(context.Context, ExportPartRequest) (ExportPart, error)) (RedactedExportManifest, error) {
	w := newRedactedZIPHash(-1)
	summary, err := WriteRedactedBundle(ctx, in, result, fetch, nil, w)
	if err != nil {
		return RedactedExportManifest{}, err
	}
	out := w.finish(summary)
	if out.Validate() != nil {
		return RedactedExportManifest{}, ErrSelection
	}
	return out, nil
}

// ReadRedactedExportPart re-derives and verifies the complete selected source and
// ZIP before returning one bounded chunk. This first implementation trades
// repeated scans for no persistent derived cache or new retention authority.
func ReadRedactedExportPart(ctx context.Context, in RedactedExportPartRequest, result ExportManifestResult, fetch func(context.Context, ExportPartRequest) (ExportPart, error)) (RedactedExportPart, error) {
	var zero RedactedExportPart
	if in.Validate() != nil {
		return zero, ErrSelection
	}
	w := newRedactedZIPHash(in.ChunkIndex)
	summary, err := WriteRedactedBundle(ctx, in.RedactedExportRequest, result, fetch, nil, w)
	if err != nil {
		return zero, err
	}
	m := w.finish(summary)
	if m.Validate() != nil || m.SourceSHA256 != in.SourceSHA256 || m.ZIPBytes != in.ZIPBytes || m.ZIPSHA256 != in.ZIPSHA256 || in.ChunkIndex >= len(m.Chunks) || m.Chunks[in.ChunkIndex] != in.ChunkSHA256 {
		return zero, ErrSelection
	}
	out := RedactedExportPart{Version: 1, Request: in, Data: w.part}
	if out.Validate() != nil {
		return zero, ErrSelection
	}
	return out, nil
}

// WriteRedactedExportParts assembles only the explicit redacted representation.
// The caller owns an unpublished private output. Failure must retain it as
// incomplete rather than substitute an original export or another source.
func WriteRedactedExportParts(ctx context.Context, in RedactedExportRequest, manifest RedactedExportManifest, fetch func(context.Context, RedactedExportPartRequest) (RedactedExportPart, error), output io.Writer) error {
	if in.Validate() != nil || manifest.Validate() != nil || fetch == nil || output == nil {
		return ErrSelection
	}
	h := sha256.New()
	for index, digest := range manifest.Chunks {
		if err := ctx.Err(); err != nil {
			return err
		}
		request := RedactedExportPartRequest{RedactedExportRequest: in, SourceSHA256: manifest.SourceSHA256, ZIPBytes: manifest.ZIPBytes, ZIPSHA256: manifest.ZIPSHA256, ChunkIndex: index, ChunkSHA256: digest}
		part, err := fetch(ctx, request)
		if err != nil {
			return err
		}
		if part.Request != request || part.Validate() != nil {
			return ErrSelection
		}
		if err = write(output, part.Data); err != nil {
			return err
		}
		h.Write(part.Data)
	}
	if hex.EncodeToString(h.Sum(nil)) != manifest.ZIPSHA256 {
		return ErrCorrupt
	}
	return ctx.Err()
}

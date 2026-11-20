package recording

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
)

const MaxTimingIndexBytes = 256 << 20
const MaxTimingIndexFrames = 1 << 20
const MaxTimingIndexLineBytes = 256 << 10

type TimingFile struct {
	Name   string `json:"name"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

// This is a content binding, not owner/source authenticity or action authority.
// Volatile read timestamps are excluded so unchanged closed sources retain the
// same identity across independently requested exports and timing queries.
type TimingSource struct {
	Version        int               `json:"version"`
	Session        SessionDescriptor `json:"session"`
	SessionID      string            `json:"session_id"`
	MetadataSHA256 string            `json:"metadata_sha256"`
	Transport      string            `json:"transport"`
	Files          []TimingFile      `json:"files"`
	Coverage       ExportCoverage    `json:"coverage"`
}

type TimingEntry struct {
	File          string `json:"file"`
	Offset        int64  `json:"offset"`
	EncodedBytes  int64  `json:"encoded_bytes"`
	Segment       uint64 `json:"segment"`
	Sequence      uint64 `json:"sequence"`
	UnixNano      int64  `json:"unix_nano"`
	EpochUnixNano int64  `json:"epoch_unix_nano"`
	ElapsedNano   uint64 `json:"elapsed_nano"`
	Kind          Kind   `json:"kind"`
	PayloadBytes  uint32 `json:"payload_bytes"`
	Digest        string `json:"digest"`
}

type TimingIndexSummary struct {
	SourceSHA256           string         `json:"source_sha256"`
	IndexSHA256            string         `json:"index_sha256"`
	Bytes                  uint64         `json:"bytes"`
	Frames                 uint64         `json:"frames"`
	Coverage               ExportCoverage `json:"coverage"`
	AllSourceFilesVerified bool           `json:"all_source_files_verified"`
	VerifiedPrefixComplete bool           `json:"verified_prefix_complete"`
	RuntimeComplete        bool           `json:"runtime_complete"`
}

type TimingIndexRecord struct {
	Type         string          `json:"type"`
	Source       *TimingSource   `json:"source,omitempty"`
	SourceSHA256 string          `json:"source_sha256,omitempty"`
	Entry        *TimingEntry    `json:"entry,omitempty"`
	End          *TimingIndexEnd `json:"end,omitempty"`
}

type TimingIndexEnd struct {
	Coverage               ExportCoverage `json:"coverage"`
	AllSourceFilesVerified bool           `json:"all_source_files_verified"`
	VerifiedPrefixComplete bool           `json:"verified_prefix_complete"`
	RuntimeComplete        bool           `json:"runtime_complete"`
}

// WriteTimingIndex reads exact private original parts, verifies chunk/whole-file
// hashes and the codec chain, and emits metadata only. No paths, execution,
// content rendering, original writes, retries or retention effects occur. A
// failure emits no completion record/summary. The caller must retain any partial
// output as incomplete and own authorization/publication of the destination.
func WriteTimingIndex(ctx context.Context, in ExportRequest, result ExportManifestResult, fetch func(context.Context, ExportPartRequest) (ExportPart, error), output io.Writer) (TimingIndexSummary, error) {
	var zero TimingIndexSummary
	if output == nil {
		return zero, ErrSelection
	}
	var summary TimingIndexSummary
	indexHash := sha256.New()
	emit := func(record TimingIndexRecord) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		b, err := json.Marshal(record)
		if err != nil {
			return err
		}
		b = append(b, '\n')
		if len(b) > MaxTimingIndexLineBytes || uint64(len(b)) > MaxTimingIndexBytes-summary.Bytes {
			return ErrLimit
		}
		if err = write(output, b); err != nil {
			return err
		}
		indexHash.Write(b)
		summary.Bytes += uint64(len(b))
		return nil
	}
	verified, err := scanExportFrames(ctx, in, result, fetch, func(source TimingSource, sourceSHA string) error {
		return emit(TimingIndexRecord{Type: "header", Source: &source, SourceSHA256: sourceSHA})
	}, func(entry TimingEntry, _ Frame) error { return emit(TimingIndexRecord{Type: "frame", Entry: &entry}) })
	if err != nil {
		return zero, err
	}
	end := TimingIndexEnd{Coverage: verified.coverage, AllSourceFilesVerified: true, VerifiedPrefixComplete: true}
	if err = emit(TimingIndexRecord{Type: "end", End: &end}); err != nil {
		return zero, err
	}
	summary.SourceSHA256 = verified.sourceSHA
	summary.IndexSHA256 = hex.EncodeToString(indexHash.Sum(nil))
	summary.Frames = verified.coverage.VerifiedFrames
	summary.Coverage = verified.coverage
	summary.AllSourceFilesVerified, summary.VerifiedPrefixComplete = true, true
	return summary, nil
}

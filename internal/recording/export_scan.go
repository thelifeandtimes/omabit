package recording

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"hash"
	"io"
)

type verifiedExport struct {
	source    TimingSource
	sourceSHA string
	coverage  ExportCoverage
}

type exportPartReader struct {
	ctx      context.Context
	selected ExportRequest
	result   ExportManifestResult
	file     ExportFile
	fetch    func(context.Context, ExportPartRequest) (ExportPart, error)
	whole    hash.Hash
	buffer   []byte
	chunk    int
	read     int64
	fatal    error
}

func newExportPartReader(ctx context.Context, in ExportRequest, result ExportManifestResult, file ExportFile, fetch func(context.Context, ExportPartRequest) (ExportPart, error)) *exportPartReader {
	return &exportPartReader{ctx: ctx, selected: in, result: result, file: file, fetch: fetch, whole: sha256.New()}
}

func (r *exportPartReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if r.fatal != nil {
		return 0, r.fatal
	}
	if err := r.ctx.Err(); err != nil {
		r.fatal = err
		return 0, err
	}
	if r.read == r.file.Bytes {
		return 0, io.EOF
	}
	if len(r.buffer) == 0 {
		if r.chunk >= len(r.file.Chunks) {
			r.fatal = ErrCorrupt
			return 0, r.fatal
		}
		in := ExportPartRequest{ExportRequest: r.selected, SessionID: r.result.Manifest.Metadata.SessionID, MetadataSHA256: r.result.Manifest.MetadataSHA256, FileName: r.file.Name, FileBytes: r.file.Bytes, ChunkIndex: r.chunk, ChunkSHA256: r.file.Chunks[r.chunk]}
		part, err := r.fetch(r.ctx, in)
		if err != nil {
			r.fatal = err
			return 0, err
		}
		if part.Request != in || part.Validate() != nil {
			r.fatal = ErrSelection
			return 0, r.fatal
		}
		r.buffer = part.Data
		r.chunk++
	}
	n := copy(p, r.buffer)
	r.buffer = r.buffer[n:]
	r.read += int64(n)
	r.whole.Write(p[:n])
	if r.read == r.file.Bytes && hex.EncodeToString(r.whole.Sum(nil)) != r.file.SHA256 {
		r.fatal = ErrCorrupt
		return n, r.fatal
	}
	return n, nil
}

func timingTail(err error) string {
	if err == io.EOF {
		return "snapshot-eof"
	}
	if errors.Is(err, ErrIncomplete) {
		return "incomplete"
	}
	if errors.Is(err, ErrCorrupt) || errors.Is(err, ErrInvalid) {
		return "corrupt"
	}
	return ""
}

// scanExportFrames is the shared private original verifier for derived output.
// Callbacks can emit only provisional data. Completion requires success after
// every raw file hash and final codec/manifest coverage agree.
func scanExportFrames(ctx context.Context, in ExportRequest, result ExportManifestResult, fetch func(context.Context, ExportPartRequest) (ExportPart, error), onSource func(TimingSource, string) error, onFrame func(TimingEntry, Frame) error) (verifiedExport, error) {
	var zero verifiedExport
	if result.Validate(in) != nil || fetch == nil || onSource == nil || onFrame == nil || ctx.Err() != nil {
		return zero, ErrSelection
	}
	m := result.Manifest.Metadata
	source := TimingSource{Version: 1, Session: result.Session, SessionID: m.SessionID, MetadataSHA256: result.Manifest.MetadataSHA256, Transport: m.Transport, Coverage: result.Manifest.Coverage}
	for _, f := range result.Manifest.Files {
		source.Files = append(source.Files, TimingFile{f.Name, f.Bytes, f.SHA256})
	}
	sourceBytes, _ := json.Marshal(source)
	sourceHash := sha256.Sum256(sourceBytes)
	sourceSHA := hex.EncodeToString(sourceHash[:])
	if err := onSource(source, sourceSHA); err != nil {
		return zero, err
	}
	meta := newExportPartReader(ctx, in, result, result.Manifest.Files[0], fetch)
	raw, err := io.ReadAll(io.LimitReader(meta, MaxMetadataBytes+1))
	if err != nil {
		return zero, err
	}
	want, _ := m.Bytes()
	if meta.fatal != nil || !bytes.Equal(raw, want) {
		return zero, ErrCorrupt
	}
	var sequence uint64
	var digest [32]byte
	var epoch int64
	var elapsed uint64
	terminal, needGap, stopped := false, false, false
	coverage := ExportCoverage{}
	for index, file := range result.Manifest.Files[1:] {
		if err := ctx.Err(); err != nil {
			return zero, err
		}
		r := newExportPartReader(ctx, in, result, file, fetch)
		if !stopped {
			d, decodeErr := NewDecoder(r)
			if r.fatal != nil {
				return zero, r.fatal
			}
			if decodeErr == nil {
				h := d.Header
				if m.CheckHeader(h) != nil || h.Segment != uint64(index+1) || h.FirstSequence != sequence+1 || h.PreviousDigest != digest {
					decodeErr = ErrCorrupt
				}
			}
			if decodeErr != nil {
				// The first header must have been valid even for a damaged-prefix
				// manifest. Later malformed headers are part of its reported tail.
				if index == 0 {
					return zero, decodeErr
				}
				coverage.Tail = timingTail(decodeErr)
				if coverage.Tail == "" {
					return zero, decodeErr
				}
				stopped = true
			} else {
				digest = d.LastDigest()
				if index == 0 {
					epoch = d.Header.StartedUnixNano
				}
				first := true
				for {
					if err := ctx.Err(); err != nil {
						return zero, err
					}
					offset := d.CommittedOffset()
					frame, e := d.Next()
					if r.fatal != nil {
						return zero, r.fatal
					}
					if e != nil {
						hasNext := index+2 < len(result.Manifest.Files)
						if e == io.EOF || errors.Is(e, ErrIncomplete) {
							if errors.Is(e, ErrIncomplete) {
								needGap = true
							}
							if hasNext && !terminal {
								break
							}
							if hasNext && terminal {
								e = ErrCorrupt
							} else if needGap {
								e = ErrIncomplete
							}
						}
						coverage.Tail = timingTail(e)
						if coverage.Tail == "" {
							return zero, e
						}
						stopped = true
						break
					}
					valid := !terminal && transportKind(m, frame.Kind)
					g, gapErr := ReadGap(frame.Data)
					if first && index > 0 {
						restart := frame.Kind == Gap && gapErr == nil && (g.Reason == "recorder-restarted" || g.Reason == "corrupt-tail")
						if restart {
							valid = valid && frame.ElapsedNano == 0
						} else {
							valid = valid && frame.EpochUnixNano == epoch && frame.ElapsedNano >= elapsed
						}
					}
					if needGap {
						valid = valid && frame.Kind == Gap && gapErr == nil && g.Reason == "corrupt-tail"
					}
					if frame.Kind == Exit {
						exit, _ := ReadExit(frame.Data)
						valid = valid && exit.ContainerID == m.ContainerID
					}
					if !valid {
						coverage.Tail = "corrupt"
						stopped = true
						break
					}
					if coverage.VerifiedFrames == MaxTimingIndexFrames {
						return zero, ErrLimit
					}
					entry := TimingEntry{File: file.Name, Offset: offset, EncodedBytes: d.CommittedOffset() - offset, Segment: frame.Segment, Sequence: frame.Sequence, UnixNano: frame.UnixNano, EpochUnixNano: frame.EpochUnixNano, ElapsedNano: frame.ElapsedNano, Kind: frame.Kind, PayloadBytes: uint32(len(frame.Data)), Digest: hex.EncodeToString(frame.Digest[:])}
					if err := onFrame(entry, frame); err != nil {
						return zero, err
					}
					if coverage.VerifiedFrames == 0 {
						coverage.FirstUnixNano = frame.UnixNano
					}
					coverage.VerifiedFrames++
					coverage.LastUnixNano = frame.UnixNano
					switch frame.Kind {
					case Stdout, Stderr, PTYCombined:
						coverage.OutputBytes += uint64(len(frame.Data))
					case Gap:
						coverage.GapFrames++
					case Exit:
						coverage.ExitSaved = true
						terminal = true
					}
					first, needGap = false, false
					sequence, digest, epoch, elapsed = frame.Sequence, frame.Digest, frame.EpochUnixNano, frame.ElapsedNano
				}
			}
		}
		// Preserve/hash all original bytes, including damaged tails and later
		// files excluded from the verified prefix. Never repair or reinterpret them.
		if _, err := io.Copy(io.Discard, r); err != nil {
			return zero, err
		}
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	if coverage != result.Manifest.Coverage {
		return zero, ErrCorrupt
	}
	return verifiedExport{source: source, sourceSHA: sourceSHA, coverage: coverage}, nil
}

//go:build linux

package recording

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"syscall"
	"time"
)

const maxSegments = 1024

var ErrLegacyLayout = errors.New("recording layout is single-segment; preserve it and create a distinct current-format session")

func segmentName(number uint64) string {
	if number == 1 {
		return "frames.obr"
	}
	return fmt.Sprintf("frames-%08d.obr", number)
}

// Rotate seals the current segment and starts a linked segment in the same
// receive-clock epoch. It neither changes metadata nor claims a transport gap.
// Time rotation can call this on the capture coordinator's own schedule.
func (w *SessionWriter) Rotate() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed || w.poisoned || w.terminal {
		return ErrPoisoned
	}
	if w.metadata.Version < 3 {
		return ErrLegacyLayout
	}
	if !boundFile(w.root, segmentName(w.segment), w.file, w.encoder.CommittedOffset()) || !boundFile(w.root, "writer.lock", w.lock, 0) {
		w.poisoned = true
		return ErrStorage
	}
	return w.rotateLocked()
}

func (w *SessionWriter) rotateLocked() error {
	if w.segment >= maxSegments || w.closedBytes+w.encoder.CommittedOffset()+int64(headerBytes) > w.store.limits.MaxSessionBytes {
		return ErrLimit
	}
	if err := w.store.admit(w.root, uint64(headerBytes+4096)); err != nil {
		return err
	}
	previous, next, elapsed := w.encoder.LastDigest(), w.encoder.next, w.encoder.lastElapsed
	if err := errors.Join(w.file.Sync(), w.file.Close()); err != nil {
		w.poisoned = true
		return err
	}
	w.closedBytes += w.encoder.CommittedOffset()
	if err := w.newSegment(w.segment+1, next, previous); err != nil {
		w.poisoned = true
		return err
	}
	w.encoder.lastElapsed = elapsed
	return nil
}

func (w *SessionWriter) newSegment(number, next uint64, previous [32]byte) error {
	file, err := privateFile(w.root, segmentName(number), os.O_WRONLY|os.O_CREATE|os.O_EXCL, w.store.limits.MaxSessionBytes)
	if err != nil {
		return err
	}
	w.file = file
	w.segment = number
	digest, _ := w.metadata.Digest()
	id, _ := hex.DecodeString(w.metadata.SessionID)
	h := Header{MetadataSHA256: digest, Segment: number, FirstSequence: next, StartedUnixNano: w.epochUnixNano, PreviousDigest: previous}
	copy(h.SessionID[:], id)
	w.encoder, err = NewEncoder(file, h)
	if err != nil {
		return err
	}
	return errors.Join(file.Sync(), syncDirectory(w.root))
}

// Continue requires the exact immutable metadata and exclusive writer lock.
// It never appends to or repairs an old file. A new linked segment begins with
// an fsynced unknown-coverage gap in a new receive-clock epoch. The caller must
// independently revalidate the same attachment/process/host authority; matching
// metadata alone is not permission to reconnect to a runtime. A host/identity
// change requires a distinct session, not rewriting this attachment's metadata.
func (s *Store) Continue(ctx context.Context, expected Metadata, at time.Time) (_ *SessionWriter, err error) {
	if expected.Validate() != nil {
		return nil, ErrInvalid
	}
	if expected.Version < 3 {
		return nil, ErrLegacyLayout
	}
	want, err := expected.Bytes()
	if err != nil || at.Before(time.Unix(0, 1)) || at.After(time.Unix(0, math.MaxInt64)) {
		return nil, ErrInvalid
	}
	r, err := childDirectory(s.root, expected.SessionID)
	if err != nil {
		return nil, err
	}
	w := &SessionWriter{store: s, root: r, epochUnixNano: at.UnixNano()}
	defer func() {
		if err != nil {
			if w.file != nil {
				w.file.Close()
			}
			if w.lock != nil {
				w.lock.Close()
			}
			r.Close()
		}
	}()
	w.lock, err = privateFile(r, "writer.lock", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	if syscall.Flock(int(w.lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) != nil {
		return nil, ErrStorage
	}
	meta, err := privateFile(r, "metadata.json", os.O_RDONLY, MaxMetadataBytes)
	if err != nil {
		return nil, err
	}
	got, readErr := io.ReadAll(io.LimitReader(meta, MaxMetadataBytes+1))
	closeErr := meta.Close()
	if readErr != nil || closeErr != nil || !bytes.Equal(got, want) {
		return nil, ErrInvalid
	}
	w.metadata, err = ReadMetadata(got) // pin independent identity pointers
	if err != nil {
		return nil, err
	}
	scan, err := s.openSegments(r, w.metadata)
	if err != nil {
		return nil, err
	}
	defer scan.Close()
	for {
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		_, err = scan.Next()
		if err != nil {
			break
		}
	}
	partial := errors.Is(err, ErrIncomplete)
	if err != io.EOF && !partial {
		return nil, err
	}
	if scan.terminal {
		return nil, ErrInvalid
	}
	if scan.number >= maxSegments || scan.sequence == math.MaxUint64-1 {
		return nil, ErrLimit
	}
	reason := "recorder-restarted"
	if partial {
		reason = "corrupt-tail"
	}
	gap, _ := (GapEvent{Version: 1, Reason: reason}).Bytes()
	// Include all original partial bytes, the new header and its mandatory gap.
	extra := int64(headerBytes + frameHeaderBytes + 32 + len(gap))
	if scan.total+extra > s.limits.MaxSessionBytes || extra > s.limits.MaxSegmentBytes {
		return nil, ErrLimit
	}
	if err = s.admit(r, uint64(extra)+4096); err != nil {
		return nil, err
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	if !scan.unchanged() || !boundFile(r, "writer.lock", w.lock, 0) {
		return nil, ErrStorage
	}
	w.closedBytes = scan.total
	if err = w.newSegment(scan.number+1, scan.sequence+1, scan.digest); err != nil {
		return nil, err
	}
	if _, err = w.Append(Gap, gap, at.UnixNano(), 0); err != nil {
		return nil, err
	}
	return w, nil
}

// segmentReader verifies originals using a bounded file-list/size snapshot.
// Later segments must link the previous verified prefix, including a new header
// if that earlier segment has no complete frame. A torn payload remains intact;
// crossing it requires the first successor frame to state corrupt-tail.
type segmentReader struct {
	store               *Store
	root                *os.Root
	metadata            Metadata
	names               []string
	file                *os.File
	decoder             *Decoder
	number, sequence    uint64
	digest, firstDigest [32]byte
	total               int64
	terminal, needGap   bool
	firstInSegment      bool
	epoch               int64
	elapsed             uint64
	verified            map[string]os.FileInfo
}

func (s *Store) openSegments(r *os.Root, m Metadata) (*segmentReader, error) {
	f, err := r.Open(".")
	if err != nil {
		return nil, ErrStorage
	}
	entries, e := f.ReadDir(maxSegments + 3)
	f.Close()
	if e != nil && e != io.EOF || len(entries) > maxSegments+2 {
		return nil, ErrStorage
	}
	names := []string{}
	for _, entry := range entries {
		name := entry.Name()
		if name == "metadata.json" || name == "writer.lock" {
			continue
		}
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		if names[i] == "frames.obr" {
			return names[j] != "frames.obr"
		}
		if names[j] == "frames.obr" {
			return false
		}
		return names[i] < names[j]
	})
	if len(names) == 0 || len(names) > maxSegments {
		return nil, ErrStorage
	}
	for i, name := range names {
		if name != segmentName(uint64(i+1)) {
			return nil, ErrStorage
		}
	}
	reader := &segmentReader{store: s, root: r, metadata: m, names: names, verified: map[string]os.FileInfo{}}
	if err = reader.openNext(); err != nil {
		reader.Close()
		return nil, err
	}
	reader.firstDigest = reader.digest
	return reader, nil
}
func (r *segmentReader) Close() error {
	if r.file != nil {
		return r.file.Close()
	}
	return nil
}
func (r *segmentReader) openNext() error {
	if r.file != nil {
		if err := r.file.Close(); err != nil {
			return err
		}
		r.file = nil
	}
	file, err := privateFile(r.root, r.names[r.number], os.O_RDONLY, r.store.limits.MaxSessionBytes)
	if err != nil {
		return err
	}
	r.file = file
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.Size() > r.store.limits.MaxSessionBytes-r.total {
		return ErrLimit
	}
	r.total += info.Size()
	r.verified[r.names[r.number]] = info
	decoder, err := NewDecoder(io.NewSectionReader(file, 0, info.Size()))
	if err != nil {
		return err
	}
	h := decoder.Header
	if r.metadata.CheckHeader(h) != nil || h.Segment != r.number+1 || h.FirstSequence != r.sequence+1 || h.PreviousDigest != r.digest {
		return ErrCorrupt
	}
	r.number++
	r.decoder = decoder
	r.firstInSegment = true
	if r.number == 1 {
		r.epoch = h.StartedUnixNano
	}
	r.digest = decoder.LastDigest()
	return nil
}

// The writer lock excludes cooperating writers. Also refuse namespace/file
// replacement or resize between the verified scan and continuation admission.
func (r *segmentReader) unchanged() bool {
	for name, before := range r.verified {
		after, e := r.root.Lstat(name)
		if e != nil || !privateFileInfo(after, before.Size()) || after.Size() != before.Size() || !os.SameFile(before, after) {
			return false
		}
	}
	return true
}

func (r *segmentReader) Next() (Frame, error) {
	for {
		frame, err := r.decoder.Next()
		if err == nil {
			if r.firstInSegment && r.number > 1 {
				g, gapErr := ReadGap(frame.Data)
				restart := frame.Kind == Gap && gapErr == nil && (g.Reason == "recorder-restarted" || g.Reason == "corrupt-tail")
				if restart {
					if frame.ElapsedNano != 0 {
						return Frame{}, ErrCorrupt
					}
				} else if frame.EpochUnixNano != r.epoch || frame.ElapsedNano < r.elapsed {
					return Frame{}, ErrCorrupt
				}
			}
			if r.terminal || !transportKind(r.metadata, frame.Kind) {
				return Frame{}, ErrCorrupt
			}
			if r.needGap {
				g, e := ReadGap(frame.Data)
				if frame.Kind != Gap || e != nil || g.Reason != "corrupt-tail" {
					return Frame{}, ErrCorrupt
				}
				r.needGap = false
			}
			if frame.Kind == Exit {
				e, _ := ReadExit(frame.Data)
				if e.ContainerID != r.metadata.ContainerID {
					return Frame{}, ErrCorrupt
				}
				r.terminal = true
			}
			r.firstInSegment = false
			r.epoch, r.elapsed = frame.EpochUnixNano, frame.ElapsedNano
			r.sequence = frame.Sequence
			r.digest = frame.Digest
			return frame, nil
		}
		if err != io.EOF && !errors.Is(err, ErrIncomplete) {
			return Frame{}, err
		}
		if errors.Is(err, ErrIncomplete) {
			r.needGap = true
		}
		if int(r.number) == len(r.names) {
			if r.needGap {
				return Frame{}, ErrIncomplete
			}
			return Frame{}, io.EOF
		}
		if r.terminal {
			return Frame{}, ErrCorrupt
		}
		if err = r.openNext(); err != nil {
			return Frame{}, err
		}
	}
}

func (s *Store) readSegmentedPage(ctx context.Context, r *os.Root, m Metadata, cursor *Cursor, maxFrames, maxBytes int) (Page, error) {
	scan, err := s.openSegments(r, m)
	if err != nil {
		return Page{}, err
	}
	defer scan.Close()
	digest, _ := m.Digest()
	hash := hex.EncodeToString(digest[:])
	initial := hex.EncodeToString(scan.firstDigest[:])
	want := Cursor{Version: 1, SessionID: m.SessionID, MetadataSHA256: hash, Digest: initial}
	if cursor != nil {
		want = *cursor
		if want.Version != 1 || want.SessionID != m.SessionID || want.MetadataSHA256 != hash || !lowerHex(want.Digest, 64) {
			return Page{}, ErrCursor
		}
	}
	matched := want.Sequence == 0 && want.Digest == initial
	if want.Sequence == 0 && !matched {
		return Page{}, ErrCursor
	}
	out := Page{Metadata: m, Frames: []Frame{}, Next: want, Tail: "not-scanned", ReadAt: time.Now().UTC(), Basis: "verified original segment prefix at read time; gaps retain missing coverage; exit requires observed process evidence"}
	lock, err := privateFile(r, "writer.lock", os.O_RDONLY, 0)
	if err != nil {
		return Page{}, err
	}
	err = syscall.Flock(int(lock.Fd()), syscall.LOCK_SH|syscall.LOCK_NB)
	lock.Close()
	if errors.Is(err, syscall.EWOULDBLOCK) {
		out.WriterPresent = true
	} else if err != nil {
		return Page{}, ErrStorage
	}
	used := 0
	for {
		if err = ctx.Err(); err != nil {
			return Page{}, err
		}
		if matched && (len(out.Frames) >= maxFrames || used > maxBytes-MaxFrameBytes) {
			return out, nil
		}
		frame, e := scan.Next()
		if e != nil {
			if !matched {
				return Page{}, ErrCursor
			}
			switch {
			case e == io.EOF:
				out.Tail = "snapshot-eof"
			case errors.Is(e, ErrIncomplete):
				out.Tail = "incomplete"
			case errors.Is(e, ErrCorrupt), errors.Is(e, ErrInvalid):
				out.Tail = "corrupt"
			default:
				return Page{}, e
			}
			return out, nil
		}
		if !matched {
			if frame.Sequence == want.Sequence {
				if hex.EncodeToString(frame.Digest[:]) != want.Digest {
					return Page{}, ErrCursor
				}
				matched = true
			}
			continue
		}
		used += len(frame.Data)
		out.Frames = append(out.Frames, frame)
		out.Next = Cursor{Version: 1, SessionID: m.SessionID, MetadataSHA256: hash, Sequence: frame.Sequence, Digest: hex.EncodeToString(frame.Digest[:])}
	}
}

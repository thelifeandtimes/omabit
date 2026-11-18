//go:build linux

package recording

import (
	"context"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

var (
	ErrStorage = errors.New("recording storage ownership or layout is invalid")
	ErrLimit   = errors.New("recording session byte limit reached; original bytes retained")
	ErrReserve = errors.New("recording persistence reserve would be consumed; original bytes retained")
	ErrCursor  = errors.New("recording cursor no longer matches verified retained coverage")
)

// Limits bound encoded session bytes and metadata3 segment size. Host/ship
// quotas, incident pins and retention are not implemented here. Free-space
// admission is not an exclusive reservation against unrelated filesystem users.
type Limits struct {
	MaxSessionBytes  int64
	ReserveFreeBytes uint64
	// Zero selects16MiB (bounded by MaxSessionBytes); only metadata3 rotates.
	MaxSegmentBytes int64
}

type Store struct {
	root   *os.Root
	limits Limits
	free   func(*os.Root, uint64) error
}

func privateDirectory(info os.FileInfo) bool {
	s, ok := info.Sys().(*syscall.Stat_t)
	return ok && info.IsDir() && info.Mode().Perm() == 0700 && s.Uid == uint32(os.Geteuid())
}
func sameDirectory(root *os.Root, before os.FileInfo) bool {
	f, err := root.Open(".")
	if err != nil {
		return false
	}
	defer f.Close()
	after, err := f.Stat()
	return err == nil && privateDirectory(after) && os.SameFile(before, after)
}

// OpenStore pins an already authorized private directory. It does not create a
// root, select a runner/target, install a service or authorize any API caller.
func OpenStore(path string, limits Limits) (*Store, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || limits.MaxSessionBytes < int64(headerBytes+1024) || limits.MaxSessionBytes > 1<<30 || limits.ReserveFreeBytes == 0 {
		return nil, ErrStorage
	}
	info, err := os.Lstat(path)
	if err != nil || !privateDirectory(info) {
		return nil, ErrStorage
	}
	r, err := os.OpenRoot(path)
	if err != nil {
		return nil, ErrStorage
	}
	if !sameDirectory(r, info) {
		r.Close()
		return nil, ErrStorage
	}
	if limits.MaxSegmentBytes == 0 {
		limits.MaxSegmentBytes = min(16<<20, limits.MaxSessionBytes)
	}
	if limits.MaxSegmentBytes < int64(headerBytes+1024) || limits.MaxSegmentBytes > limits.MaxSessionBytes {
		r.Close()
		return nil, ErrStorage
	}
	return &Store{root: r, limits: limits, free: reserveSpace}, nil
}
func (s *Store) Close() error { return s.root.Close() }

func reserveSpace(root *os.Root, needed uint64) error {
	f, err := root.Open(".")
	if err != nil {
		return ErrStorage
	}
	defer f.Close()
	var stat syscall.Statfs_t
	if syscall.Fstatfs(int(f.Fd()), &stat) != nil || stat.Bsize <= 0 || stat.Bavail > ^uint64(0)/uint64(stat.Bsize) {
		return ErrStorage
	}
	if stat.Bavail*uint64(stat.Bsize) < needed {
		return ErrReserve
	}
	return nil
}
func (s *Store) admit(root *os.Root, extra uint64) error {
	if extra > ^uint64(0)-s.limits.ReserveFreeBytes {
		return ErrReserve
	}
	return s.free(root, s.limits.ReserveFreeBytes+extra)
}
func syncDirectory(root *os.Root) error {
	f, e := root.Open(".")
	if e != nil {
		return e
	}
	defer f.Close()
	return f.Sync()
}
func childDirectory(root *os.Root, name string) (*os.Root, error) {
	before, err := root.Lstat(name)
	if err != nil || !privateDirectory(before) {
		return nil, ErrStorage
	}
	r, err := root.OpenRoot(name)
	if err != nil {
		return nil, ErrStorage
	}
	if !sameDirectory(r, before) {
		r.Close()
		return nil, ErrStorage
	}
	return r, nil
}
func privateFile(root *os.Root, name string, flags int, max int64) (*os.File, error) {
	f, err := root.OpenFile(name, flags|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0600)
	if err != nil {
		return nil, err
	}
	i, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, ErrStorage
	}
	if !privateFileInfo(i, max) {
		f.Close()
		return nil, ErrStorage
	}
	return f, nil
}
func privateFileInfo(i os.FileInfo, max int64) bool {
	st, ok := i.Sys().(*syscall.Stat_t)
	return ok && i.Mode().IsRegular() && i.Mode().Perm() == 0600 && st.Uid == uint32(os.Geteuid()) && st.Nlink == 1 && i.Size() <= max
}
func boundFile(root *os.Root, name string, f *os.File, size int64) bool {
	i, e1 := f.Stat()
	path, e2 := root.Lstat(name)
	return e1 == nil && e2 == nil && privateFileInfo(i, size) && i.Size() == size && os.SameFile(i, path)
}

type SessionWriter struct {
	mu               sync.Mutex
	store            *Store
	root             *os.Root
	file, lock       *os.File
	encoder          *Encoder
	metadata         Metadata
	closed, poisoned bool
	closeErr         error
	segment          uint64
	closedBytes      int64
	epochUnixNano    int64
	terminal         bool
}

func transportKind(m Metadata, kind Kind) bool {
	return kind == Gap || kind == Exit || m.Transport == "docker-multiplex" && (kind == Stdout || kind == Stderr) || m.Transport == "pty-combined" && kind == PTYCombined
}

// Create never reuses an existing session directory or repairs a partial one.
// Metadata/header and directory entries sync before returning an attachment.
// A failed create leaves its original incomplete files for explicit diagnosis.
func (s *Store) Create(m Metadata) (_ *SessionWriter, err error) {
	if m.Version == 1 {
		return nil, ErrLegacyMetadata
	}
	b, err := m.Bytes()
	if err != nil {
		return nil, err
	}
	// Pin independent copies of successor metadata's nested identity pointers.
	// A caller retaining its Metadata value cannot rebind an admitted writer.
	m, err = ReadMetadata(b)
	if err != nil {
		return nil, err
	}
	if err = s.admit(s.root, uint64(len(b)+headerBytes+8192)); err != nil {
		return nil, err
	}
	if err = s.root.Mkdir(m.SessionID, 0700); err != nil {
		return nil, err
	}
	r, err := childDirectory(s.root, m.SessionID)
	if err != nil {
		return nil, err
	}
	w := &SessionWriter{root: r, store: s, metadata: m, segment: 1, epochUnixNano: m.AttachedAt.UnixNano()}
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
	w.lock, err = privateFile(r, "writer.lock", os.O_RDWR|os.O_CREATE|os.O_EXCL, 0)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(w.lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return nil, ErrStorage
	}
	meta, err := privateFile(r, "metadata.json", os.O_WRONLY|os.O_CREATE|os.O_EXCL, MaxMetadataBytes)
	if err != nil {
		return nil, err
	}
	err = errors.Join(write(meta, b), meta.Sync(), meta.Close())
	if err != nil {
		return nil, err
	}
	w.file, err = privateFile(r, "frames.obr", os.O_WRONLY|os.O_CREATE|os.O_EXCL, s.limits.MaxSessionBytes)
	if err != nil {
		return nil, err
	}
	digest, _ := m.Digest()
	id, _ := hex.DecodeString(m.SessionID)
	h := Header{MetadataSHA256: digest, Segment: 1, FirstSequence: 1, StartedUnixNano: m.AttachedAt.UnixNano()}
	copy(h.SessionID[:], id)
	w.encoder, err = NewEncoder(w.file, h)
	if err != nil {
		return nil, err
	}
	err = errors.Join(w.file.Sync(), syncDirectory(r), syncDirectory(s.root))
	if err != nil {
		return nil, err
	}
	return w, nil
}

// Append serializes original frames and syncs before returning success. It is
// a durable writer receipt, not proof of complete transport/runtime coverage.
// Caller timing must use the segment's single receive-clock coordinator.
func (w *SessionWriter) Append(kind Kind, data []byte, unixNano int64, elapsedNano uint64) (Frame, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed || w.poisoned || w.terminal {
		return Frame{}, ErrPoisoned
	}
	if !boundFile(w.root, segmentName(w.segment), w.file, w.encoder.CommittedOffset()) || !boundFile(w.root, "writer.lock", w.lock, 0) {
		w.poisoned = true
		return Frame{}, ErrStorage
	}
	if len(data) == 0 || len(data) > MaxFrameBytes || !transportKind(w.metadata, kind) || validateLifecycle(kind, data) != nil {
		return Frame{}, ErrInvalid
	}
	if kind == Exit {
		e, _ := ReadExit(data)
		if e.ContainerID != w.metadata.ContainerID {
			return Frame{}, ErrInvalid
		}
	}
	if unixNano <= 0 || elapsedNano < w.encoder.lastElapsed {
		return Frame{}, ErrInvalid
	}
	n := int64(frameHeaderBytes + len(data) + 32)
	rotate := w.metadata.Version == 3 && w.encoder.CommittedOffset()+n > w.store.limits.MaxSegmentBytes
	extra := n
	if rotate {
		extra += int64(headerBytes)
	}
	if extra > w.store.limits.MaxSessionBytes-w.closedBytes-w.encoder.CommittedOffset() {
		return Frame{}, ErrLimit
	}
	if err := w.store.admit(w.root, uint64(extra)+4096); err != nil {
		return Frame{}, err
	}
	if rotate {
		if n+int64(headerBytes) > w.store.limits.MaxSegmentBytes {
			return Frame{}, ErrLimit
		}
		if err := w.rotateLocked(); err != nil {
			return Frame{}, err
		}
	}
	f, err := w.encoder.Append(kind, data, unixNano, elapsedNano)
	if err == nil {
		err = w.file.Sync()
	}
	if err != nil {
		if w.encoder.poisoned || f.Sequence != 0 {
			w.poisoned = true
		}
		return Frame{}, err
	}
	if w.metadata.Version == 3 && kind == Exit {
		w.terminal = true
	}
	return f, nil
}
func (w *SessionWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return w.closeErr
	}
	w.closed = true
	var poison error
	if w.poisoned {
		poison = ErrPoisoned
	}
	w.closeErr = errors.Join(poison, w.file.Sync(), w.file.Close(), w.lock.Close(), w.root.Close())
	return w.closeErr
}

type Cursor struct {
	Version        int    `json:"version"`
	SessionID      string `json:"session_id"`
	MetadataSHA256 string `json:"metadata_sha256"`
	Sequence       uint64 `json:"sequence"`
	Digest         string `json:"digest"`
}
type Page struct {
	Metadata      Metadata  `json:"metadata"`
	Frames        []Frame   `json:"frames"`
	Next          Cursor    `json:"next"`
	Tail          string    `json:"tail"`
	WriterPresent bool      `json:"writer_present"`
	ReadAt        time.Time `json:"read_at"`
	Basis         string    `json:"basis"`
}

// ReadPage scans/checks the bounded original prefix; no index is trusted or
// repaired. Cursor mismatch never silently rewinds. EOF is only this file-size
// snapshot, not runtime exit. A growing writer can have a transient partial tail.
func (s *Store) ReadPage(ctx context.Context, id string, cursor *Cursor, maxFrames, maxBytes int) (Page, error) {
	var out Page
	if !lowerHex(id, 32) || maxFrames < 1 || maxFrames > 256 || maxBytes < MaxFrameBytes || maxBytes > 1<<20 {
		return out, ErrInvalid
	}
	r, err := childDirectory(s.root, id)
	if err != nil {
		return out, err
	}
	defer r.Close()
	meta, err := privateFile(r, "metadata.json", os.O_RDONLY, MaxMetadataBytes)
	if err != nil {
		return out, err
	}
	b, err := io.ReadAll(io.LimitReader(meta, MaxMetadataBytes+1))
	meta.Close()
	if err != nil {
		return out, err
	}
	m, err := ReadMetadata(b)
	if err != nil || m.SessionID != id {
		return out, ErrCorrupt
	}
	if m.Version == 3 {
		return s.readSegmentedPage(ctx, r, m, cursor, maxFrames, maxBytes)
	}
	file, err := privateFile(r, "frames.obr", os.O_RDONLY, s.limits.MaxSessionBytes)
	if err != nil {
		return out, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return out, err
	}
	if info.Size() > s.limits.MaxSessionBytes {
		return out, ErrLimit
	}
	d, err := NewDecoder(io.NewSectionReader(file, 0, info.Size()))
	if err != nil {
		return out, err
	}
	if d.Header.Segment != 1 || m.CheckHeader(d.Header) != nil {
		return out, ErrCorrupt
	}
	digest, _ := m.Digest()
	metadataHash := hex.EncodeToString(digest[:])
	firstDigest := d.LastDigest()
	want := Cursor{Version: 1, SessionID: id, MetadataSHA256: metadataHash, Digest: hex.EncodeToString(firstDigest[:])}
	if cursor != nil {
		want = *cursor
		if want.Version != 1 || want.SessionID != id || want.MetadataSHA256 != metadataHash || !lowerHex(want.Digest, 64) {
			return out, ErrCursor
		}
	}
	matched := want.Sequence == 0 && want.Digest == hex.EncodeToString(firstDigest[:])
	if want.Sequence == 0 && !matched {
		return out, ErrCursor
	}
	out = Page{Metadata: m, Frames: []Frame{}, Next: want, Tail: "not-scanned", ReadAt: time.Now().UTC(), Basis: "verified original bytes at read time; process exit requires separate observed exit evidence"}
	lock, err := privateFile(r, "writer.lock", os.O_RDONLY, 0)
	if err != nil {
		return Page{}, err
	}
	err = syscall.Flock(int(lock.Fd()), syscall.LOCK_SH|syscall.LOCK_NB)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		out.WriterPresent = true
	} else if err != nil {
		lock.Close()
		return Page{}, ErrStorage
	}
	lock.Close()
	used := 0
	for {
		if err := ctx.Err(); err != nil {
			return Page{}, err
		}
		if matched && (len(out.Frames) >= maxFrames || used > maxBytes-MaxFrameBytes) {
			return out, nil
		}
		frame, err := d.Next()
		if err != nil {
			if !matched {
				return Page{}, ErrCursor
			}
			switch {
			case errors.Is(err, io.EOF):
				out.Tail = "snapshot-eof"
			case errors.Is(err, ErrIncomplete):
				out.Tail = "incomplete"
			case errors.Is(err, ErrCorrupt):
				out.Tail = "corrupt"
			default:
				return Page{}, err
			}
			return out, nil
		}
		if frame.Kind == Exit {
			e, _ := ReadExit(frame.Data)
			if e.ContainerID != m.ContainerID {
				if !matched {
					return Page{}, ErrCursor
				}
				out.Tail = "corrupt"
				return out, nil
			}
		}
		if !transportKind(m, frame.Kind) {
			if !matched {
				return Page{}, ErrCursor
			}
			out.Tail = "corrupt"
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
		out.Next = Cursor{Version: 1, SessionID: id, MetadataSHA256: metadataHash, Sequence: frame.Sequence, Digest: hex.EncodeToString(frame.Digest[:])}
	}
}

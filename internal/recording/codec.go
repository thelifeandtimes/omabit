// Package recording defines the original-byte recording format under development.
// It does not start a runtime, open host paths, choose retention or authorize reads.
// Runner-owned storage and runtime integration must enforce those boundaries.
package recording

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"math"
)

const (
	magic            = "OMABIT-RECORDING/1\x00"
	MaxFrameBytes    = 64 << 10
	headerBytes      = len(magic) + 16 + 32 + 8 + 8 + 8 + 32
	frameHeaderBytes = 8 + 8 + 8 + 1 + 4
)

var (
	ErrInvalid    = errors.New("invalid recording format or binding")
	ErrIncomplete = errors.New("incomplete recording; preserve original bytes and report a gap")
	ErrCorrupt    = errors.New("recording checksum, sequence or timing mismatch")
	ErrPoisoned   = errors.New("recording writer failed; continue only in a new segment with gap evidence")
)

type Kind uint8

const (
	Stdout      Kind = 1
	Stderr      Kind = 2
	PTYCombined Kind = 3
	Gap         Kind = 4
	Exit        Kind = 5
)

// Header binds one segment to the exact immutable session metadata. Subsequent
// segments link to the last verified record/header digest of the previous one.
// The storage layer must also check segment/sequence continuity and any original
// partial tail. A checksum is corruption detection, not owner-state authenticity.
type Header struct {
	SessionID       [16]byte
	MetadataSHA256  [32]byte
	Segment         uint64
	FirstSequence   uint64
	StartedUnixNano int64
	PreviousDigest  [32]byte
}

func (h Header) Validate() error {
	if h.SessionID == [16]byte{} || h.MetadataSHA256 == [32]byte{} || h.Segment == 0 || h.FirstSequence == 0 || h.FirstSequence == math.MaxUint64 || h.StartedUnixNano <= 0 {
		return ErrInvalid
	}
	if h.Segment == 1 && (h.FirstSequence != 1 || h.PreviousDigest != [32]byte{}) || h.Segment > 1 && h.PreviousDigest == [32]byte{} {
		return ErrInvalid
	}
	return nil
}

func encodeHeader(h Header) []byte {
	b := make([]byte, headerBytes)
	n := copy(b, magic)
	n += copy(b[n:], h.SessionID[:])
	n += copy(b[n:], h.MetadataSHA256[:])
	for _, v := range []uint64{h.Segment, h.FirstSequence, uint64(h.StartedUnixNano)} {
		binary.BigEndian.PutUint64(b[n:], v)
		n += 8
	}
	copy(b[n:], h.PreviousDigest[:])
	return b
}

type Frame struct {
	Sequence    uint64
	UnixNano    int64
	ElapsedNano uint64
	Kind        Kind
	Data        []byte `json:"-"` // raw originals require an explicit export choice
	Digest      [32]byte
}

// Encoder belongs to one serialized capture coordinator; calls are not
// concurrent. It never rewrites bytes and becomes permanently poisoned on any
// short or failed write. The owner must sync files before claiming persistence.
type Encoder struct {
	w           io.Writer
	next        uint64
	lastElapsed uint64
	digest      [32]byte
	committed   int64
	poisoned    bool
}

func NewEncoder(w io.Writer, h Header) (*Encoder, error) {
	if w == nil || h.Validate() != nil {
		return nil, ErrInvalid
	}
	b := encodeHeader(h)
	if err := write(w, b); err != nil {
		return nil, err
	}
	return &Encoder{w: w, next: h.FirstSequence, digest: sha256.Sum256(b), committed: int64(len(b))}, nil
}

func write(w io.Writer, data []byte) error {
	n, err := w.Write(data)
	if err == nil && n != len(data) {
		return io.ErrShortWrite
	}
	return err
}

// Append preserves the given original bytes, including long lines, credentials,
// invalid UTF-8 and terminal controls. Kind is transport identity or a separately
// validated canonical lifecycle/gap payload; the codec performs no redaction/rendering.
func (e *Encoder) Append(kind Kind, data []byte, unixNano int64, elapsedNano uint64) (Frame, error) {
	if e.poisoned {
		return Frame{}, ErrPoisoned
	}
	if kind < Stdout || kind > Exit || len(data) == 0 || len(data) > MaxFrameBytes || unixNano <= 0 || elapsedNano < e.lastElapsed || e.next == math.MaxUint64 {
		return Frame{}, ErrInvalid
	}
	if validateLifecycle(kind, data) != nil {
		return Frame{}, ErrInvalid
	}
	head := make([]byte, frameHeaderBytes)
	binary.BigEndian.PutUint64(head, e.next)
	binary.BigEndian.PutUint64(head[8:], uint64(unixNano))
	binary.BigEndian.PutUint64(head[16:], elapsedNano)
	head[24] = byte(kind)
	binary.BigEndian.PutUint32(head[25:], uint32(len(data)))
	hash := sha256.New()
	hash.Write(e.digest[:])
	hash.Write(head)
	hash.Write(data)
	var digest [32]byte
	copy(digest[:], hash.Sum(nil))
	for _, part := range [][]byte{head, data, digest[:]} {
		if err := write(e.w, part); err != nil {
			e.poisoned = true
			return Frame{}, err
		}
	}
	frame := Frame{Sequence: e.next, UnixNano: unixNano, ElapsedNano: elapsedNano, Kind: kind, Data: append([]byte(nil), data...), Digest: digest}
	e.next++
	e.lastElapsed = elapsedNano
	e.digest = digest
	e.committed += int64(len(head) + len(data) + len(digest))
	return frame, nil
}

// CommittedOffset counts complete encoded frames, not a filesystem fsync claim.
func (e *Encoder) CommittedOffset() int64 { return e.committed }
func (e *Encoder) LastDigest() [32]byte   { return e.digest }

type Decoder struct {
	r           io.Reader
	Header      Header
	next        uint64
	lastElapsed uint64
	digest      [32]byte
	committed   int64
	failed      bool
}

func NewDecoder(r io.Reader) (*Decoder, error) {
	if r == nil {
		return nil, ErrInvalid
	}
	b := make([]byte, headerBytes)
	if _, err := io.ReadFull(r, b); err != nil {
		return nil, incomplete(err)
	}
	if string(b[:len(magic)]) != magic {
		return nil, ErrInvalid
	}
	var h Header
	n := len(magic)
	n += copy(h.SessionID[:], b[n:])
	n += copy(h.MetadataSHA256[:], b[n:])
	h.Segment = binary.BigEndian.Uint64(b[n:])
	n += 8
	h.FirstSequence = binary.BigEndian.Uint64(b[n:])
	n += 8
	h.StartedUnixNano = int64(binary.BigEndian.Uint64(b[n:]))
	n += 8
	copy(h.PreviousDigest[:], b[n:])
	if h.Validate() != nil {
		return nil, ErrInvalid
	}
	return &Decoder{r: r, Header: h, next: h.FirstSequence, digest: sha256.Sum256(b), committed: int64(len(b))}, nil
}

func incomplete(err error) error {
	// A partial record must never satisfy errors.Is(err, io.EOF), which ordinary
	// readers commonly interpret as successful completion of the whole stream.
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return ErrIncomplete
	}
	return errors.Join(ErrIncomplete, err)
}

// Next returns only a complete verified frame. A partial read preserves the last
// committed offset/digest; callers reopen there using trusted index/binding state
// or observe the growing original again. Never truncate an original to "repair"
// it or treat transport EOF as a verified runtime exit.
func (d *Decoder) Next() (Frame, error) {
	if d.failed {
		return Frame{}, ErrCorrupt
	}
	head := make([]byte, frameHeaderBytes)
	n, err := io.ReadFull(d.r, head)
	if err == io.EOF && n == 0 {
		return Frame{}, io.EOF
	}
	if err != nil {
		d.failed = true
		return Frame{}, incomplete(err)
	}
	frame := Frame{Sequence: binary.BigEndian.Uint64(head), UnixNano: int64(binary.BigEndian.Uint64(head[8:])), ElapsedNano: binary.BigEndian.Uint64(head[16:]), Kind: Kind(head[24])}
	size := binary.BigEndian.Uint32(head[25:])
	if size == 0 || size > MaxFrameBytes || frame.Kind < Stdout || frame.Kind > Exit || frame.Sequence != d.next || d.next == math.MaxUint64 || frame.UnixNano <= 0 || frame.ElapsedNano < d.lastElapsed {
		d.failed = true
		return Frame{}, ErrCorrupt
	}
	frame.Data = make([]byte, size)
	if _, err := io.ReadFull(d.r, frame.Data); err != nil {
		d.failed = true
		return Frame{}, incomplete(err)
	}
	if _, err := io.ReadFull(d.r, frame.Digest[:]); err != nil {
		d.failed = true
		return Frame{}, incomplete(err)
	}
	hash := sha256.New()
	hash.Write(d.digest[:])
	hash.Write(head)
	hash.Write(frame.Data)
	var expected [32]byte
	copy(expected[:], hash.Sum(nil))
	if frame.Digest != expected {
		d.failed = true
		return Frame{}, ErrCorrupt
	}
	if validateLifecycle(frame.Kind, frame.Data) != nil {
		d.failed = true
		return Frame{}, ErrCorrupt
	}
	d.next++
	d.lastElapsed = frame.ElapsedNano
	d.digest = frame.Digest
	d.committed += int64(len(head) + len(frame.Data) + len(frame.Digest))
	return frame, nil
}

func (d *Decoder) CommittedOffset() int64 { return d.committed }
func (d *Decoder) LastDigest() [32]byte   { return d.digest }

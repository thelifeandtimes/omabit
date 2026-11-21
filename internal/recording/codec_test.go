package recording

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"testing"
)

func testHeader() Header {
	return Header{SessionID: [16]byte{1}, MetadataSHA256: sha256.Sum256([]byte("exact owner metadata")), Segment: 1, FirstSequence: 1, StartedUnixNano: 1700000000000000000}
}

func TestOriginalFramesPreserveBytesAndClockBasis(t *testing.T) {
	var out bytes.Buffer
	h := testHeader()
	w, err := NewEncoder(&out, h)
	if err != nil {
		t.Fatal(err)
	}
	first := append(bytes.Repeat([]byte("trace\n"), 350), bytes.Repeat([]byte("x"), 12288)...)
	first = append(first, []byte("\n0wdiagnostic.atom Authorization: fake-token\r\x1b[2K\xff\x00")...)
	gap, _ := (GapEvent{Version: 1, Reason: "transport-interrupted"}).Bytes()
	exit, _ := (ExitEvent{Version: 1, ContainerID: string(bytes.Repeat([]byte("a"), 64)), StartedUnixNano: h.StartedUnixNano, FinishedUnixNano: h.StartedUnixNano + 1, Code: 7}).Bytes()
	want := [][]byte{first, []byte("stderr retained\n"), gap, exit}
	for i, data := range want {
		kind := []Kind{Stdout, Stderr, Gap, Exit}[i]
		// Wall clocks can move backwards. Monotonic observation order cannot.
		if _, err := w.Append(kind, data, h.StartedUnixNano-int64(i), uint64(i)*40000000); err != nil {
			t.Fatal(err)
		}
	}
	r, err := NewDecoder(bytes.NewReader(out.Bytes()))
	if err != nil || r.Header != h {
		t.Fatal(r, err)
	}
	for i, data := range want {
		frame, err := r.Next()
		if err != nil || frame.Sequence != uint64(i+1) || !bytes.Equal(frame.Data, data) || frame.Kind != []Kind{Stdout, Stderr, Gap, Exit}[i] || frame.ElapsedNano != uint64(i)*40000000 {
			t.Fatal(frame, err)
		}
	}
	if _, err := r.Next(); err != io.EOF {
		t.Fatal(err)
	}
	if r.CommittedOffset() != int64(out.Len()) || r.CommittedOffset() != w.CommittedOffset() || r.LastDigest() != w.LastDigest() {
		t.Fatal("verified coverage or digest differs")
	}
}

func TestInterruptedTailNeverBecomesCommittedOrRepaired(t *testing.T) {
	var out bytes.Buffer
	w, err := NewEncoder(&out, testHeader())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Append(Stdout, []byte("complete first trace\n"), 1700000000000000001, 1); err != nil {
		t.Fatal(err)
	}
	committed, digest := w.CommittedOffset(), w.LastDigest()
	if _, err := w.Append(Stderr, []byte("interrupted second trace\n"), 1700000000000000002, 2); err != nil {
		t.Fatal(err)
	}
	all := append([]byte(nil), out.Bytes()...)
	for cut := committed + 1; cut < int64(len(all)); cut++ {
		original := append([]byte(nil), all[:cut]...)
		r, err := NewDecoder(bytes.NewReader(original))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := r.Next(); err != nil {
			t.Fatal(err)
		}
		if _, err := r.Next(); !errors.Is(err, ErrIncomplete) || errors.Is(err, io.EOF) {
			t.Fatalf("partial tail at %d accepted: %v", cut, err)
		}
		if r.CommittedOffset() != committed || r.LastDigest() != digest || !bytes.Equal(original, all[:cut]) {
			t.Fatal("read changed original or committed unverified bytes")
		}
		if _, err := r.Next(); !errors.Is(err, ErrCorrupt) {
			t.Fatal("failed decoder reused without reobservation")
		}
	}
	for cut := 0; cut < headerBytes; cut++ {
		if _, err := NewDecoder(bytes.NewReader(all[:cut])); !errors.Is(err, ErrIncomplete) || errors.Is(err, io.EOF) {
			t.Fatal("partial header accepted", cut, err)
		}
	}
}

type failingWriter struct {
	bytes.Buffer
	limit int
	short bool
}

func (w *failingWriter) Write(data []byte) (int, error) {
	if w.limit < 0 {
		return w.Buffer.Write(data)
	}
	if len(data) <= w.limit {
		w.limit -= len(data)
		return w.Buffer.Write(data)
	}
	n, _ := w.Buffer.Write(data[:w.limit])
	w.limit = 0
	if w.short {
		return n, nil
	}
	return n, errors.New("injected disk pressure")
}

func TestFailedOrShortWriterRequiresNewSegment(t *testing.T) {
	for _, short := range []bool{false, true} {
		for _, capacity := range []int{0, 5, frameHeaderBytes, frameHeaderBytes + 3, frameHeaderBytes + 7 + 10} {
			out := &failingWriter{limit: -1, short: short}
			w, err := NewEncoder(out, testHeader())
			if err != nil {
				t.Fatal(err)
			}
			before, digest := w.CommittedOffset(), w.LastDigest()
			out.limit = capacity
			if _, err := w.Append(Stdout, []byte("payload"), 1700000000000000001, 1); err == nil {
				t.Fatal("injected failure accepted")
			}
			original := append([]byte(nil), out.Bytes()...)
			out.limit = -1
			if _, err := w.Append(Stdout, []byte("never append after failure"), 1700000000000000002, 2); err != ErrPoisoned {
				t.Fatal(err)
			}
			if w.CommittedOffset() != before || w.LastDigest() != digest || !bytes.Equal(original, out.Bytes()) {
				t.Fatal("failure silently repaired or advanced coverage")
			}
		}
	}
}

func TestBindingChainAndBoundedReadsRefuseCorruption(t *testing.T) {
	var out bytes.Buffer
	w, err := NewEncoder(&out, testHeader())
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err := w.Append(Stdout, []byte("same bytes"), 1700000000000000001+int64(i), uint64(i)); err != nil {
			t.Fatal(err)
		}
	}
	all := out.Bytes()
	for _, offset := range []int{len(magic), len(magic) + 16, headerBytes, headerBytes + 24, headerBytes + frameHeaderBytes, headerBytes + frameHeaderBytes + len("same bytes"), len(all) - 1} {
		changed := append([]byte(nil), all...)
		changed[offset] ^= 1
		r, err := NewDecoder(bytes.NewReader(changed))
		if err != nil {
			continue
		}
		for {
			_, err = r.Next()
			if err != nil {
				break
			}
		}
		if errors.Is(err, io.EOF) {
			t.Fatal("corrupt binding/chain accepted", offset)
		}
	}
	for _, size := range []uint32{0, MaxFrameBytes + 1, math.MaxUint32} {
		changed := append([]byte(nil), all...)
		binary.BigEndian.PutUint32(changed[headerBytes+25:], size)
		r, err := NewDecoder(bytes.NewReader(changed))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := r.Next(); err != ErrCorrupt {
			t.Fatal("unbounded frame length accepted", size, err)
		}
	}
	frameBytes := frameHeaderBytes + len("same bytes") + 32
	reordered := append([]byte(nil), all[:headerBytes]...)
	reordered = append(reordered, all[headerBytes+frameBytes:]...)
	reordered = append(reordered, all[headerBytes:headerBytes+frameBytes]...)
	r, err := NewDecoder(bytes.NewReader(reordered))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Next(); err != ErrCorrupt {
		t.Fatal("reordered original records accepted")
	}
}

func TestHeaderContinuationAndInvalidAppendHaveNoEffects(t *testing.T) {
	h := testHeader()
	for _, bad := range []Header{{}, func() Header { v := h; v.Segment = 2; return v }(), func() Header { v := h; v.FirstSequence = 2; return v }(), func() Header { v := h; v.FirstSequence = math.MaxUint64; return v }()} {
		var out bytes.Buffer
		if _, err := NewEncoder(&out, bad); err == nil || out.Len() != 0 {
			t.Fatal("invalid header wrote bytes", bad, err)
		}
	}
	var first bytes.Buffer
	w, err := NewEncoder(&first, h)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Append(Stdout, []byte("first"), h.StartedUnixNano, 10); err != nil {
		t.Fatal(err)
	}
	before := append([]byte(nil), first.Bytes()...)
	for _, tc := range []struct {
		kind    Kind
		data    []byte
		elapsed uint64
	}{{0, []byte("x"), 11}, {Stdout, nil, 11}, {Stdout, make([]byte, MaxFrameBytes+1), 11}, {Stderr, []byte("x"), 9}} {
		if _, err := w.Append(tc.kind, tc.data, h.StartedUnixNano, tc.elapsed); err != ErrInvalid || !bytes.Equal(before, first.Bytes()) {
			t.Fatal("invalid append changed original", err)
		}
	}
	h.Segment = 2
	h.FirstSequence = 2
	h.PreviousDigest = w.LastDigest()
	var second bytes.Buffer
	next, err := NewEncoder(&second, h)
	if err != nil {
		t.Fatal(err)
	}
	gap, _ := (GapEvent{Version: 1, Reason: "recorder-restarted"}).Bytes()
	frame, err := next.Append(Gap, gap, h.StartedUnixNano+100, 0)
	if err != nil || frame.Sequence != 2 {
		t.Fatal(frame, err)
	}
	r, err := NewDecoder(bytes.NewReader(second.Bytes()))
	if err != nil || r.Header.PreviousDigest != w.LastDigest() {
		t.Fatal(r, err)
	}
	if frame, err := r.Next(); err != nil || frame.Kind != Gap || frame.Sequence != 2 {
		t.Fatal(frame, err)
	}
}

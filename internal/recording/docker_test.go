package recording

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"testing"
)

func dockerFrame(kind byte, data []byte) []byte {
	out := make([]byte, 8, len(data)+8)
	out[0] = kind
	binary.BigEndian.PutUint32(out[4:], uint32(len(data)))
	return append(out, data...)
}

func TestDockerOriginalStreamPreservesBurstLongLineAndStreamIdentity(t *testing.T) {
	stdout := append(bytes.Repeat([]byte("trace\n"), 350), bytes.Repeat([]byte("long"), 40000)...)
	stdout = append(stdout, []byte("\r\x1b[2K0wdiagnostic.atom Authorization: fake-token\x00\xff\n")...)
	stderr := []byte("original stderr\n")
	input := append(dockerFrame(1, stdout), dockerFrame(2, stderr)...)
	var original bytes.Buffer
	encoder, err := NewEncoder(&original, testHeader())
	if err != nil {
		t.Fatal(err)
	}
	var tick uint64
	stats, err := CopyDockerStream(context.Background(), bytes.NewReader(input), func(kind Kind, data []byte) error {
		tick++
		_, err := encoder.Append(kind, data, testHeader().StartedUnixNano+int64(tick), tick)
		return err
	})
	if err != nil || !stats.AtFrameBoundary || stats.TransportFrames != 2 || stats.OriginalBytes != uint64(len(stdout)+len(stderr)) || stats.RecordedChunks < 4 {
		t.Fatal(stats, err)
	}
	decoder, err := NewDecoder(bytes.NewReader(original.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	var out, errout bytes.Buffer
	for {
		frame, err := decoder.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		switch frame.Kind {
		case Stdout:
			out.Write(frame.Data)
		case Stderr:
			errout.Write(frame.Data)
		default:
			t.Fatal("stream identity changed")
		}
	}
	if !bytes.Equal(out.Bytes(), stdout) || !bytes.Equal(errout.Bytes(), stderr) {
		t.Fatal("original stream bytes transformed or lost")
	}
}

func TestDockerTransportPartialPayloadPreservesPrefixAndReportsGap(t *testing.T) {
	data := bytes.Repeat([]byte("trace\n"), 20000)
	frame := dockerFrame(1, data)
	for _, cut := range []int{1, 7, 8, 9, 8 + MaxFrameBytes, 8 + MaxFrameBytes + 3, len(frame) - 1} {
		var out bytes.Buffer
		stats, err := CopyDockerStream(context.Background(), bytes.NewReader(frame[:cut]), func(kind Kind, data []byte) error {
			if kind != Stdout {
				t.Fatal(kind)
			}
			_, err := out.Write(data)
			return err
		})
		if !errors.Is(err, ErrTransportFrame) || errors.Is(err, io.EOF) || stats.AtFrameBoundary {
			t.Fatal("partial transport accepted as success", cut, stats, err)
		}
		want := max(0, cut-8)
		if out.Len() != want || !bytes.Equal(out.Bytes(), data[:want]) || stats.OriginalBytes != uint64(want) {
			t.Fatal("received prefix discarded or duplicated", cut, stats)
		}
	}
	for _, changed := range []int{0, 1, 2, 3} {
		bad := append([]byte(nil), frame...)
		bad[changed] = 9
		stats, err := CopyDockerStream(context.Background(), bytes.NewReader(bad), func(Kind, []byte) error { t.Fatal("invalid framing payload dispatched"); return nil })
		if err != ErrTransportFrame || stats.OriginalBytes != 0 {
			t.Fatal(stats, err)
		}
	}
}

func TestDockerRecordingFailureAndCancellationNeverReplay(t *testing.T) {
	input := dockerFrame(1, bytes.Repeat([]byte("x"), MaxFrameBytes+10))
	writes := 0
	stats, err := CopyDockerStream(context.Background(), bytes.NewReader(input), func(Kind, []byte) error { writes++; return errors.New("injected disk pressure") })
	if !errors.Is(err, ErrRecordingWrite) || writes != 1 || stats.KnownUnrecordedBytes != MaxFrameBytes || stats.OriginalBytes != 0 || stats.AtFrameBoundary {
		t.Fatal(stats, writes, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	stats, err = CopyDockerStream(ctx, bytes.NewReader(input), func(Kind, []byte) error { t.Fatal("cancelled capture wrote bytes"); return nil })
	if !errors.Is(err, context.Canceled) || stats.OriginalBytes != 0 {
		t.Fatal(stats, err)
	}
	stats, err = CopyDockerStream(context.Background(), bytes.NewReader(nil), func(Kind, []byte) error { t.Fatal("empty stream wrote bytes"); return nil })
	if err != nil || !stats.AtFrameBoundary || stats.TransportFrames != 0 {
		t.Fatal("transport EOF was not distinct from exit receipt", stats, err)
	}
}

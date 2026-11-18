//go:build linux

package recording

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"
)

func TestCaptureFitsLongOriginalTransportIntoSmallConfiguredSegments(t *testing.T) {
	s, _ := segmentStore(t, 1<<20, 4096)
	m := captureMetadata()
	payload := append(bytes.Repeat([]byte("multiline\n0wdiagnostic.atom\r\x1b[2K"), 4096), 0, 255)
	out, err := s.CaptureDocker(t.Context(), m, DockerCaptureSource{
		Attach: func(context.Context) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(capturePacket(1, payload))), nil
		},
		Ready: func(Metadata) error { return nil },
		ObserveExit: func(context.Context) (*ExitEvent, error) {
			return &ExitEvent{Version: 1, ContainerID: m.ContainerID, StartedUnixNano: m.AttachedAt.UnixNano(), FinishedUnixNano: m.AttachedAt.Add(time.Second).UnixNano(), Code: 0}, nil
		},
	})
	if err != nil || !out.ExitSaved || !out.ClosedDurably || out.GapReason != "" || out.Stats.OriginalBytes != uint64(len(payload)) || out.Stats.KnownUnrecordedBytes != 0 {
		t.Fatal("small segments lost long transport output", out, err)
	}
	page, err := s.ReadPage(t.Context(), m.SessionID, nil, 256, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	var original []byte
	var chunks uint64
	for _, frame := range page.Frames {
		if frame.Kind == Stdout {
			chunks++
			original = append(original, frame.Data...)
		}
	}
	if !bytes.Equal(original, payload) || chunks != out.Stats.RecordedChunks || page.Frames[len(page.Frames)-1].Segment < 16 {
		t.Fatal("long raw trace/stream/chunk/segment evidence changed")
	}
}

func TestCaptureSmallSegmentFailureCountsOnlyReceivedUnrecordedChunk(t *testing.T) {
	s, _ := segmentStore(t, 1<<20, 4096)
	capacity := 4096 - headerBytes - frameHeaderBytes - 32
	payload := bytes.Repeat([]byte{'x'}, capacity*4)
	m := captureMetadata()
	admissions := 0
	s.free = func(*os.Root, uint64) error {
		admissions++
		if admissions >= 3 {
			return ErrReserve
		}
		return nil
	}
	out, err := s.CaptureDocker(t.Context(), m, DockerCaptureSource{
		Attach: func(context.Context) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(capturePacket(1, payload))), nil
		},
		Ready:       func(Metadata) error { return nil },
		ObserveExit: func(context.Context) (*ExitEvent, error) { return nil, nil },
	})
	if !errors.Is(err, ErrCaptureIncomplete) || out.GapReason != "disk-pressure" || out.GapSaved || out.Stats.RecordedChunks != 1 || out.Stats.OriginalBytes != uint64(capacity) || out.Stats.KnownUnrecordedBytes != uint64(capacity) {
		t.Fatal("partial callback split overstated or hid lost bytes", out, err)
	}
	page, err := s.ReadPage(t.Context(), m.SessionID, nil, 256, 1<<20)
	if err != nil || len(page.Frames) != 1 || !bytes.Equal(page.Frames[0].Data, payload[:capacity]) {
		t.Fatal("pressure changed acknowledged original prefix", err)
	}
}

//go:build linux

package recording

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"
)

var ErrCaptureIncomplete = errors.New("recording coverage is incomplete; preserve originals and inspect its gap/status")

// DockerCaptureSource belongs to the trusted external recorder. Its factory
// must independently verify the selected daemon, full container/launch contract,
// non-TTY transport and complete metadata identity before this call. These Go
// callbacks are not caller-supplied RPC arguments or an authorization grant.
type DockerCaptureSource struct {
	// Attach acknowledges a read-only, output-only upgraded transport. Its
	// lifetime is the recorder's context, never a UI, SSH or Start request.
	Attach func(context.Context) (io.ReadCloser, error)
	// Ready acknowledges durable metadata and attachment to the lifecycle
	// coordinator. It does not establish process start or coverage. Start
	// admission remains a separate trusted lifecycle decision.
	Ready func(Metadata) error
	// ObserveExit supplies independently verified terminal-process evidence
	// for this exact container, or nil when its exit cannot be established.
	// EOF alone is never evidence. This callback gets a five-second bound.
	ObserveExit func(context.Context) (*ExitEvent, error)
}

type captureStream struct {
	io.ReadCloser
	once sync.Once
	err  error
}

func onceReadCloser(stream io.ReadCloser) io.ReadCloser { return &captureStream{ReadCloser: stream} }
func (s *captureStream) Close() error {
	s.once.Do(func() { s.err = s.ReadCloser.Close() })
	return s.err
}

type CaptureResult struct {
	SessionID              string    `json:"session_id"`
	AttachmentAcknowledged bool      `json:"attachment_acknowledged"`
	ReadyAcknowledged      bool      `json:"ready_acknowledged"`
	Stats                  CopyStats `json:"stats"`
	GapReason              string    `json:"gap_reason,omitempty"`
	GapSaved               bool      `json:"gap_saved"`
	ExitSaved              bool      `json:"exit_saved"`
	// A successful close includes fsync. It is not a completeness claim for
	// bytes never received, prior bootstrap or future process output.
	ClosedDurably bool `json:"closed_durably"`
}

// CaptureDocker serializes one new private current-format recording. It persists
// metadata before attaching and acknowledges readiness only after attachment.
// It never starts, signals, waits for, removes or retries a container. The
// independently supervised worker supplies its own lifetime and trusted source.
// This coordinator is not yet wired into installed runtime launch.
//
// Raw bytes are unchanged and fsynced. Cancel closes only this attachment, which
// interrupts a blocked read. Failure preserves the exact partial session; when
// a gap cannot be persisted, the returned finite status still reports it.
// Storage recovery/retention and reconnection require separate admission, never
// automatic replay or silent loss. A new session may link its predecessor.
func (s *Store) CaptureDocker(ctx context.Context, m Metadata, source DockerCaptureSource) (CaptureResult, error) {
	epoch := time.Now()
	return s.captureDocker(ctx, m, source, epoch, func() (int64, uint64) {
		now := time.Now()
		return now.UnixNano(), uint64(now.Sub(epoch))
	})
}

func (s *Store) captureDocker(ctx context.Context, m Metadata, source DockerCaptureSource, epoch time.Time, stamp func() (int64, uint64)) (out CaptureResult, err error) {
	if source.Attach == nil || source.Ready == nil || source.ObserveExit == nil || stamp == nil || m.Version != CurrentMetadataVersion || m.Transport != "docker-multiplex" {
		return out, ErrInvalid
	}
	if err = ctx.Err(); err != nil {
		return out, err
	}
	m.AttachedAt = epoch
	w, err := s.Create(m)
	if err != nil {
		return out, err
	}
	out.SessionID = w.metadata.SessionID
	defer func() {
		closeErr := w.Close()
		out.ClosedDurably = closeErr == nil
		err = errors.Join(err, closeErr)
	}()
	appendFrame := func(kind Kind, data []byte) error {
		wall, elapsed := stamp()
		_, e := w.Append(kind, data, wall, elapsed)
		return e
	}
	gap := func(reason string) {
		out.GapReason = reason
		data, _ := (GapEvent{Version: 1, Reason: reason}).Bytes()
		out.GapSaved = appendFrame(Gap, data) == nil
	}
	if ctx.Err() != nil {
		gap("bootstrap-not-attached")
		return out, ErrCaptureIncomplete
	}
	stream, err := source.Attach(ctx)
	if err != nil || stream == nil {
		if stream != nil {
			stream.Close()
		}
		gap("bootstrap-not-attached")
		return out, ErrCaptureIncomplete
	}
	out.AttachmentAcknowledged = true
	// Cancellation and normal completion may race. Close is serialized rather
	// than requiring arbitrary source readers to tolerate concurrent closes.
	stream = onceReadCloser(stream)
	stop := context.AfterFunc(ctx, func() { stream.Close() })
	defer func() { stop(); stream.Close() }()
	data, _ := w.metadata.Bytes()
	readyMetadata, _ := ReadMetadata(data) // independent nested identity pointers
	if err = ctx.Err(); err != nil {
		stream.Close()
		gap("transport-interrupted")
		return out, ErrCaptureIncomplete
	}
	if source.Ready(readyMetadata) != nil {
		stream.Close()
		gap("process-observation-unavailable")
		return out, ErrCaptureIncomplete
	}
	out.ReadyAcknowledged = true
	out.Stats, err = CopyDockerStream(ctx, stream, appendFrame)
	// Stop consuming only this output connection immediately on failure/EOF.
	// Storage diagnostics and independent exit observation must not leave an
	// unread attachment applying backpressure to the runtime.
	stream.Close()
	if err != nil {
		reason := "transport-interrupted"
		if errors.Is(err, ErrReserve) || errors.Is(err, ErrLimit) {
			reason = "disk-pressure"
		} else if errors.Is(err, ErrRecordingWrite) {
			reason = "write-failed"
		}
		gap(reason)
	}
	if ctx.Err() != nil {
		if out.GapReason == "" {
			gap("transport-interrupted")
		}
		return out, ErrCaptureIncomplete
	}
	observe, cancel := context.WithTimeout(ctx, 5*time.Second)
	exit, observeErr := source.ObserveExit(observe)
	expired := observe.Err() != nil
	cancel()
	if expired || observeErr != nil || exit == nil || exit.Validate() != nil || exit.ContainerID != w.metadata.ContainerID {
		if out.GapReason == "" {
			gap("process-observation-unavailable")
		}
		return out, ErrCaptureIncomplete
	}
	exitBytes, _ := exit.Bytes()
	if appendFrame(Exit, exitBytes) != nil {
		if out.GapReason == "" {
			gap("write-failed")
		}
		return out, ErrCaptureIncomplete
	}
	out.ExitSaved = true
	if out.GapReason != "" {
		return out, ErrCaptureIncomplete
	}
	return out, nil
}

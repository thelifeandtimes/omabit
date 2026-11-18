//go:build linux

package recording

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func capturePacket(stream byte, data []byte) []byte {
	packet := make([]byte, 8, 8+len(data))
	packet[0] = stream
	binary.BigEndian.PutUint32(packet[4:], uint32(len(data)))
	return append(packet, data...)
}

type countedCaptureStream struct {
	io.ReadCloser
	closed atomic.Int32
}

func (s *countedCaptureStream) Close() error { s.closed.Add(1); return s.ReadCloser.Close() }

func TestCapturePersistsReadyBeforeOriginalBytesAndIndependentExit(t *testing.T) {
	s, dir := segmentStore(t, 1<<20, 72<<10)
	m := segmentedMetadata()
	epoch := m.AttachedAt
	stdout := append(bytes.Repeat([]byte("trace\n"), 350), bytes.Repeat([]byte("long-trace"), 18000)...)
	stdout = append(stdout, []byte("\r\x1b[2K\x00\xff 0wdiagnostic.atom Authorization: private-original")...)
	stderr := []byte("distinct stderr\n")
	stream := &countedCaptureStream{ReadCloser: io.NopCloser(bytes.NewReader(append(capturePacket(1, stdout), capturePacket(2, stderr)...)))}
	attached, ready, observed := 0, 0, 0
	source := DockerCaptureSource{
		Attach: func(context.Context) (io.ReadCloser, error) {
			attached++
			page, err := s.ReadPage(t.Context(), m.SessionID, nil, 256, 1<<20)
			if err != nil || len(page.Frames) != 0 || !page.WriterPresent {
				t.Fatal("attachment preceded durable metadata", err)
			}
			return stream, nil
		},
		Ready: func(bound Metadata) error {
			ready++
			if attached != 1 || bound.Identity.Target != m.Identity.Target {
				t.Fatal("readiness did not follow exact attachment")
			}
			bound.Identity.Target.InstanceID = "changed"
			bound.Identity.ProjectNamespace.Inode++
			return nil
		},
		ObserveExit: func(ctx context.Context) (*ExitEvent, error) {
			observed++
			deadline, ok := ctx.Deadline()
			if !ok || time.Until(deadline) > 5*time.Second || ready != 1 {
				t.Fatal("unbounded or premature process observation")
			}
			return &ExitEvent{Version: 1, ContainerID: m.ContainerID, StartedUnixNano: epoch.UnixNano() + 1, FinishedUnixNano: epoch.UnixNano() + 9, Code: 7}, nil
		},
	}
	var tick uint64
	out, err := s.captureDocker(t.Context(), m, source, epoch, func() (int64, uint64) {
		tick++
		// Clock reversal changes wall time, never receive ordering.
		return epoch.UnixNano() - int64(tick), tick
	})
	if err != nil || !out.AttachmentAcknowledged || !out.ReadyAcknowledged || !out.ExitSaved || !out.ClosedDurably || out.GapReason != "" || out.Stats.OriginalBytes != uint64(len(stdout)+len(stderr)) || attached != 1 || ready != 1 || observed != 1 || stream.closed.Load() != 1 {
		t.Fatal("original capture did not finish independently", out, err)
	}
	page, err := s.ReadPage(t.Context(), m.SessionID, nil, 256, 1<<20)
	if err != nil || page.WriterPresent || page.Metadata.Identity.Target != m.Identity.Target || page.Metadata.Identity.ProjectNamespace.Inode != m.Identity.ProjectNamespace.Inode {
		t.Fatal("capture metadata was rebound or remained locked", err)
	}
	var gotOut, gotErr []byte
	for i, f := range page.Frames {
		if f.Sequence != uint64(i+1) || f.ElapsedNano != uint64(i+1) || f.UnixNano != epoch.UnixNano()-int64(i+1) {
			t.Fatal("receive timing/order changed", f)
		}
		switch f.Kind {
		case Stdout:
			gotOut = append(gotOut, f.Data...)
		case Stderr:
			gotErr = append(gotErr, f.Data...)
		case Exit:
			exit, err := ReadExit(f.Data)
			if err != nil || i != len(page.Frames)-1 || exit.Code != 7 {
				t.Fatal("exit was inferred from transport instead of supplied evidence")
			}
		default:
			t.Fatal("unexpected gap or transport kind")
		}
	}
	if !bytes.Equal(gotOut, stdout) || !bytes.Equal(gotErr, stderr) || page.Frames[len(page.Frames)-1].Segment < 3 {
		t.Fatal("capture changed raw bytes or failed to rotate")
	}
	body, _ := json.Marshal(out)
	if bytes.Contains(body, []byte("private-original")) {
		t.Fatal("capture status disclosed raw data")
	}
	before := fileSnapshot(t, filepath.Join(dir, m.SessionID))
	if _, err = s.CaptureDocker(t.Context(), m, source); err == nil || attached != 1 {
		t.Fatal("completed original was reused or reattached", err)
	}
	for name, want := range before {
		got, _ := os.ReadFile(filepath.Join(dir, m.SessionID, name))
		if !bytes.Equal(want, got) {
			t.Fatal("failed replay changed an original")
		}
	}
}

func TestCaptureSeparatesAttachmentCoverageStorageAndProcessFailures(t *testing.T) {
	for _, scenario := range []string{"attach", "ready", "eof-running", "partial-transport", "wrong-exit", "disk-pressure", "write-failed"} {
		t.Run(scenario, func(t *testing.T) {
			s, _ := openTestStore(t, 1<<20)
			m := segmentedMetadata()
			packet := capturePacket(1, []byte("preserved partial original"))
			if scenario == "partial-transport" {
				packet = packet[:len(packet)-3]
			}
			stream := &countedCaptureStream{ReadCloser: io.NopCloser(bytes.NewReader(packet))}
			attached, ready, observed := 0, 0, 0
			source := DockerCaptureSource{
				Attach: func(context.Context) (io.ReadCloser, error) {
					attached++
					if scenario == "attach" {
						return nil, errors.New("private attach diagnostic")
					}
					return stream, nil
				},
				Ready: func(Metadata) error {
					ready++
					if scenario == "ready" {
						return errors.New("private readiness diagnostic")
					}
					if scenario == "disk-pressure" || scenario == "write-failed" {
						s.free = func(*os.Root, uint64) error {
							if scenario == "disk-pressure" {
								return ErrReserve
							}
							return errors.New("private persistence diagnostic")
						}
					}
					return nil
				},
				ObserveExit: func(context.Context) (*ExitEvent, error) {
					observed++
					if scenario == "eof-running" {
						return nil, nil
					}
					exit := &ExitEvent{Version: 1, ContainerID: m.ContainerID, StartedUnixNano: m.AttachedAt.UnixNano(), FinishedUnixNano: m.AttachedAt.Add(time.Second).UnixNano(), Code: 0}
					if scenario == "wrong-exit" {
						exit.ContainerID = strings.Repeat("f", 64)
					}
					return exit, nil
				},
			}
			out, err := s.CaptureDocker(t.Context(), m, source)
			if !errors.Is(err, ErrCaptureIncomplete) || !out.ClosedDurably || out.SessionID != m.SessionID || attached != 1 || strings.Contains(err.Error(), "private") {
				t.Fatal("failed capture hid loss or retried/disclosed diagnostics", out, err)
			}
			if scenario == "attach" {
				if ready != 0 || observed != 0 || stream.closed.Load() != 0 || out.GapReason != "bootstrap-not-attached" {
					t.Fatal("failed attachment admitted runtime readiness", out)
				}
			} else if ready != 1 || stream.closed.Load() != 1 {
				t.Fatal("capture did not close exactly its own attachment", out)
			}
			page, e := s.ReadPage(t.Context(), m.SessionID, nil, 256, 1<<20)
			if e != nil || page.WriterPresent {
				t.Fatal("failed capture originals were unavailable", e)
			}
			if scenario == "disk-pressure" || scenario == "write-failed" {
				if out.GapSaved || out.ExitSaved || len(page.Frames) != 0 || out.Stats.KnownUnrecordedBytes == 0 {
					t.Fatal("storage refusal became successful or silent coverage", out)
				}
				if scenario == "disk-pressure" && out.GapReason != "disk-pressure" || scenario == "write-failed" && out.GapReason != "write-failed" {
					t.Fatal("storage reason was lost", out)
				}
			} else if !out.GapSaved || out.ExitSaved != (scenario == "partial-transport") {
				t.Fatal("EOF/failed start fabricated exit or hid a gap", out)
			}
		})
	}
}

func TestCaptureCancellationClosesOnlyItsTransportAndPersistsGap(t *testing.T) {
	s, _ := openTestStore(t, 1<<20)
	m := segmentedMetadata()
	reader, writer := io.Pipe()
	defer writer.Close()
	stream := &countedCaptureStream{ReadCloser: reader}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	ready := make(chan struct{})
	var observed atomic.Int32
	done := make(chan CaptureResult, 1)
	go func() {
		out, err := s.CaptureDocker(ctx, m, DockerCaptureSource{
			Attach: func(context.Context) (io.ReadCloser, error) { return stream, nil },
			Ready:  func(Metadata) error { close(ready); return nil },
			ObserveExit: func(context.Context) (*ExitEvent, error) {
				observed.Add(1)
				return nil, nil
			},
		})
		if !errors.Is(err, ErrCaptureIncomplete) {
			t.Error("cancel fabricated successful capture", err)
		}
		done <- out
	}()
	select {
	case <-ready:
	case <-time.After(time.Second):
		t.Fatal("capture did not acknowledge attachment")
	}
	cancel()
	select {
	case out := <-done:
		if !out.GapSaved || out.GapReason != "transport-interrupted" || out.ExitSaved || !out.ClosedDurably || observed.Load() != 0 || stream.closed.Load() != 1 {
			t.Fatal("cancel became process exit or leaked transport", out)
		}
	case <-time.After(time.Second):
		t.Fatal("cancel left a blocked stream reader")
	}
}

func TestCaptureAdmissionRefusesLegacyIncompleteOrCanceledBeforeEffects(t *testing.T) {
	for _, scenario := range []string{"legacy", "tty", "identity", "missing-ready", "canceled"} {
		t.Run(scenario, func(t *testing.T) {
			s, dir := openTestStore(t, 1<<20)
			m := segmentedMetadata()
			called := 0
			source := DockerCaptureSource{
				Attach: func(context.Context) (io.ReadCloser, error) { called++; return nil, nil },
				Ready:  func(Metadata) error { called++; return nil },
				ObserveExit: func(context.Context) (*ExitEvent, error) {
					called++
					return nil, nil
				},
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			switch scenario {
			case "legacy":
				m.Version = 2
			case "tty":
				m.Transport = "pty-combined"
			case "identity":
				m.Identity.Target.Generation = 0
			case "missing-ready":
				source.Ready = nil
			case "canceled":
				cancel()
			}
			out, err := s.CaptureDocker(ctx, m, source)
			if err == nil || called != 0 || out.SessionID != "" {
				t.Fatal("unadmitted capture attached or created a session", out, err)
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 0 {
				t.Fatal("refused admission changed storage", err)
			}
		})
	}
}

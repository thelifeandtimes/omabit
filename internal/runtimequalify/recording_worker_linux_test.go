//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

func recordingWorkerFixture(t *testing.T) (*fakeEngine, RecordingWorkerOptions, ManagedRecordingJob, string, RecordingWorkerIdentity, *fakeContainer) {
	t.Helper()
	f, state, op, selected := preparedRecordingFixture(t, true, false)
	m, _, err := prepareManagedRecording(t.Context(), state, op, func(recording.Metadata) error { return nil }, f.d, func(context.Context, string, string) (io.ReadCloser, error) {
		t.Fatal("fixture attached")
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	job, err := NewManagedRecordingJob(state, op, m, strings.Repeat("e", 64), "local-uid:1000", time.Now(), RecordingJobLimits{MaxSessionBytes: 1 << 20, MaxSegmentBytes: 64 << 10, ReserveFreeBytes: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err = os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	opts := RecordingWorkerOptions{JobsRoot: filepath.Join(root, "jobs"), RecordingsRoot: filepath.Join(root, "recordings"), MaxJobs: 8, JobReserveFreeBytes: 1 << 20}
	for _, p := range []string{opts.JobsRoot, opts.RecordingsRoot} {
		if err = os.Mkdir(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	jobs, err := OpenRecordingJobs(opts.JobsRoot, 8, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := jobs.Create(job)
	jobs.Close()
	if err != nil {
		t.Fatal(err)
	}
	worker := RecordingWorkerIdentity{ExecutableSHA256: job.ExecutableSHA256, PID: uint32(os.Getpid()), StartTicks: 1, HostBootID: m.Identity.HostBootID, Namespaces: m.Identity.RecorderNamespaces}
	return f, opts, job, digest, worker, selected
}

func recordingWorkerPacket(payload []byte) []byte {
	header := make([]byte, 8)
	header[0] = 2
	binary.BigEndian.PutUint32(header[4:], uint32(len(payload)))
	return append(header, payload...)
}

func TestRecordingWorkerAcknowledgesAfterSyncedMetadataAndRetainsRawExit(t *testing.T) {
	f, opts, job, digest, worker, selected := recordingWorkerFixture(t)
	payload := append(bytes.Repeat([]byte("bootstrap\n0wdiagnostic.atom\r\x1b[2K\n"), 350), 0, 255)
	stream := &recordingTestStream{Reader: bytes.NewReader(recordingWorkerPacket(payload))}
	acknowledged := 0
	ack := func(ready RecordingJobReady) error {
		acknowledged++
		jobs, err := OpenRecordingJobs(opts.JobsRoot, 8, 1<<20)
		if err != nil {
			t.Fatal(err)
		}
		defer jobs.Close()
		saved, err := jobs.ReadReady(job.ID, digest)
		if err != nil || saved != ready {
			t.Fatal("supervisor ack preceded saved readiness", err)
		}
		store, err := recording.OpenStore(opts.RecordingsRoot, recording.Limits{MaxSessionBytes: 1 << 20, MaxSegmentBytes: 64 << 10, ReserveFreeBytes: 1 << 20})
		if err != nil {
			t.Fatal(err)
		}
		defer store.Close()
		page, err := store.ReadPage(t.Context(), ready.SessionID, nil, 256, 1<<20)
		if err != nil || len(page.Frames) != 0 || !page.WriterPresent || recordingJobSHA(mustRecordingMetadata(t, page.Metadata)) != ready.MetadataSHA256 || page.Metadata.Identity.NativeShip != "" {
			t.Fatal("ack before durable metadata/header, or invented bootstrap identity", err)
		}
		// Inert native observation seam only; no lifecycle effect is sent.
		s := mapAt(selected.doc, "State")
		s["Status"], s["StartedAt"], s["FinishedAt"] = "exited", "2026-11-18T00:00:00Z", "2026-11-18T00:00:01Z"
		s["ExitCode"], s["OOMKilled"] = 137, true
		return nil
	}
	out, err := runManagedRecordingWorker(t.Context(), opts, job.ID, digest, ack, f.d, func(*qualifier) (RecordingWorkerIdentity, error) { return worker, nil }, func(context.Context, string, string) (io.ReadCloser, error) { return stream, nil })
	if err != nil || out.Code != "capture-finished" || !out.Capture.ExitSaved || !out.Capture.ClosedDurably || acknowledged != 1 || stream.closes != 1 {
		t.Fatal("worker capture/ack failed", out, err)
	}
	jobs, err := OpenRecordingJobs(opts.JobsRoot, 8, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	defer jobs.Close()
	allocation, allocationErr := jobs.ReadSession(job.ID, digest)
	if allocationErr != nil || allocation.SessionID != out.Capture.SessionID {
		t.Fatal("capture cannot be found through its original job", allocationErr)
	}
	saved, err := jobs.ReadResult(job.ID, digest)
	if err != nil || !reflect.DeepEqual(saved, out) {
		t.Fatal("saved result rebound", err)
	}
	if _, err = jobs.Claim(job.ID, digest, worker); !errors.Is(err, ErrRecordingJobClaimed) {
		t.Fatal("completed worker was replayed", err)
	}
	store, err := recording.OpenStore(opts.RecordingsRoot, recording.Limits{MaxSessionBytes: 1 << 20, MaxSegmentBytes: 64 << 10, ReserveFreeBytes: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	page, err := store.ReadPage(t.Context(), out.Capture.SessionID, nil, 256, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	var raw []byte
	for _, frame := range page.Frames {
		if frame.Kind == recording.Stderr {
			raw = append(raw, frame.Data...)
		}
	}
	if !bytes.Equal(raw, payload) || page.Metadata.Identity.NativeShip != "" || page.Metadata.Identity.NativeShipStatus != "pending-comet-mining" {
		t.Fatal("worker transformed bytes or inferred a name")
	}
	resultBytes, err := os.ReadFile(filepath.Join(opts.JobsRoot, job.ID, "result.json"))
	if err != nil || bytes.Contains(resultBytes, []byte("0wdiagnostic.atom")) {
		t.Fatal("raw output leaked into worker status", err)
	}
}

func TestRecordingWorkerRefusesChangedAdmissionAndDoesNotPromoteVisibleReadyFile(t *testing.T) {
	for _, scenario := range []string{"executable", "engine", "recording-storage", "ack-failed", "no-exit-proof", "canceled"} {
		t.Run(scenario, func(t *testing.T) {
			f, opts, job, digest, worker, _ := recordingWorkerFixture(t)
			if scenario == "executable" {
				worker.ExecutableSHA256 = strings.Repeat("f", 64)
			}
			if scenario == "engine" {
				f.engine.ID = "replacement-daemon"
			}
			if scenario == "recording-storage" {
				if err := os.Chmod(opts.RecordingsRoot, 0755); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if scenario == "canceled" {
				cancel()
			}
			attached, acked := 0, 0
			out, err := runManagedRecordingWorker(ctx, opts, job.ID, digest, func(RecordingJobReady) error {
				acked++
				if scenario == "ack-failed" {
					return errors.New("PRIVATE-ACK-CREDENTIAL-DIAGNOSTIC")
				}
				return nil
			}, f.d, func(*qualifier) (RecordingWorkerIdentity, error) { return worker, nil }, func(context.Context, string, string) (io.ReadCloser, error) {
				attached++
				return io.NopCloser(bytes.NewReader(recordingWorkerPacket([]byte("inert output\n")))), nil
			})
			if err == nil || strings.Contains(err.Error(), "PRIVATE-ACK-CREDENTIAL-DIAGNOSTIC") {
				t.Fatal("worker failure was successful or leaked diagnostics", out, err)
			}
			if scenario == "executable" || scenario == "canceled" {
				if attached != 0 || acked != 0 {
					t.Fatal("early refusal attached/acked")
				}
				return
			}
			switch scenario {
			case "engine":
				if out.Code != "admission-refused" || attached != 0 || acked != 0 {
					t.Fatal("changed engine admitted", out)
				}
			case "recording-storage":
				if out.Code != "storage-unavailable" || attached != 0 || acked != 0 {
					t.Fatal("bad storage admitted", out)
				}
			case "ack-failed":
				if out.Code != "capture-incomplete" || acked != 1 || out.Capture.ReadyAcknowledged || out.Capture.ExitSaved || !out.Capture.GapSaved {
					t.Fatal("visible ready file became supervisor acknowledgement", out)
				}
			case "no-exit-proof":
				if out.Code != "capture-incomplete" || !out.Capture.ReadyAcknowledged || out.Capture.ExitSaved || out.Capture.GapReason != "process-observation-unavailable" {
					t.Fatal("EOF became exit", out)
				}
			}
			jobs, openErr := OpenRecordingJobs(opts.JobsRoot, 8, 1<<20)
			if openErr != nil {
				t.Fatal(openErr)
			}
			defer jobs.Close()
			if scenario == "recording-storage" {
				allocation, e := jobs.ReadSession(job.ID, digest)
				if e != nil || allocation.SessionID == "" || out.Capture.SessionID != "" {
					t.Fatal("storage failure lost allocated session or invented capture", e)
				}
				if _, e := jobs.ReadReady(job.ID, digest); !errors.Is(e, ErrRecordingJobNotReady) {
					t.Fatal("allocation became successful readiness", e)
				}
			}
			if scenario == "ack-failed" {
				if _, e := jobs.ReadReady(job.ID, digest); e != nil {
					t.Fatal("test did not retain visible but unacknowledged marker", e)
				}
			}
			if _, e := jobs.Claim(job.ID, digest, worker); !errors.Is(e, ErrRecordingJobClaimed) {
				t.Fatal("failed worker claim was automatically reused", e)
			}
		})
	}
}

func TestRecordingWorkerCancellationClosesOnlyItsAttachment(t *testing.T) {
	f, opts, job, digest, worker, _ := recordingWorkerFixture(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	reader, writer := io.Pipe()
	defer writer.Close()
	callsBefore := len(f.calls)
	ready := make(chan struct{})
	result := make(chan RecordingJobResult, 1)
	go func() {
		out, _ := runManagedRecordingWorker(ctx, opts, job.ID, digest, func(RecordingJobReady) error { close(ready); return nil }, f.d, func(*qualifier) (RecordingWorkerIdentity, error) { return worker, nil }, func(context.Context, string, string) (io.ReadCloser, error) { return reader, nil })
		result <- out
	}()
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		t.Fatal("worker not ready")
	}
	cancel()
	select {
	case out := <-result:
		if out.Code != "capture-incomplete" || out.Capture.ExitSaved || !out.Capture.GapSaved || out.Capture.GapReason != "transport-interrupted" {
			t.Fatal("cancel fabricated an exit or omitted gap", out)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("worker cancellation did not detach blocked reader")
	}
	for _, call := range f.calls[callsBefore:] {
		if len(call) > 1 && (call[1] == "kill" || call[1] == "stop") {
			t.Fatal("recording cancellation signaled runtime", call)
		}
	}
}

func TestRecordingWorkerObservesOnlyItsOwnRunningImageAndNamespaces(t *testing.T) {
	q := &qualifier{d: defaults()}
	worker, err := observeRecordingWorkerIdentity(q)
	if err != nil || !worker.valid() || worker.PID != uint32(os.Getpid()) || worker.StartTicks == 0 {
		t.Fatal("missing actual self identity", err)
	}
	if worker.ExecutableSHA256 == strings.Repeat("0", 64) {
		t.Fatal("invented executable")
	}
}

func TestRecordingWorkerStartIdentityHandlesParenthesizedCommAndRefusesMissingTicks(t *testing.T) {
	fields := make([]string, 20)
	for i := range fields {
		fields[i] = "0"
	}
	fields[0] = "S"
	fields[19] = "123456"
	raw := []byte("123 (worker ) with spaces) " + strings.Join(fields, " "))
	if ticks, err := recordingWorkerStartTicks(raw); err != nil || ticks != 123456 {
		t.Fatal("Linux comm field broke process identity", ticks, err)
	}
	for _, bad := range [][]byte{nil, []byte("123 (worker) S"), []byte("123 (worker) " + strings.Join(append(fields[:19], "0"), " ")), bytes.Repeat([]byte{'x'}, 4097)} {
		if _, err := recordingWorkerStartTicks(bad); !errors.Is(err, ErrRecordingWorker) {
			t.Fatal("missing/out-of-range process identity accepted", err)
		}
	}
}

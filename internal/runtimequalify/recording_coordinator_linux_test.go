//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

func TestRecordingCoordinatorComposesExactSavedReadyAndLiveFencesWithoutStartOrRetry(t *testing.T) {
	for _, scenario := range []string{"pass", "pre-fence", "final-fence", "already-claimed", "lost-ack", "no-ready", "saved-result", "late-result", "closed-writer", "unexpected-output", "partial-frame", "changed-metadata", "changed-claim", "wrong-unit", "changed-unit", "worker-gone", "native-changed"} {
		t.Run(scenario, func(t *testing.T) {
			f, opts, job, digest, worker, _ := recordingWorkerFixture(t)
			worker.InvocationID = strings.Repeat("a", 32)
			jobs, err := OpenRecordingJobs(opts.JobsRoot, 8, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			defer jobs.Close()
			originalJob, _ := os.ReadFile(filepath.Join(opts.JobsRoot, job.ID, "job.json"))
			if scenario == "already-claimed" {
				lease, err := jobs.Claim(job.ID, digest, worker)
				if err != nil {
					t.Fatal(err)
				}
				lease.Close()
			}
			unitName, _ := RecordingWorkerUnitName(job)
			unit := RecordingSupervisorState{Unit: unitName, MainPID: worker.PID, InvocationID: worker.InvocationID, ControlGroup: "/user.slice/" + unitName}
			fences, submits, observations, nativeChecks := 0, 0, 0, 0
			prepare := func(selected ManagedRecordingJob) (recording.Metadata, error) {
				m, _, err := prepareManagedRecording(t.Context(), selected.State, selected.Operation, func(recording.Metadata) error { return nil }, f.d, func(context.Context, string, string) (io.ReadCloser, error) {
					t.Fatal("coordinator attempted an attachment")
					return nil, nil
				})
				return m, err
			}
			fence := func(context.Context) error {
				fences++
				if scenario == "pre-fence" || scenario == "final-fence" && fences == 2 {
					return errors.New("PRIVATE-JOURNAL-DETAIL")
				}
				return nil
			}
			var acknowledged RecordingJobReady
			submit := func(_ context.Context, selected ManagedRecordingJob) error {
				submits++
				if selected.ID != job.ID || submits != 1 {
					t.Fatal("submission rebound or retried")
				}
				lease, err := jobs.Claim(job.ID, digest, worker)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { lease.Close() })
				m, err := prepare(selected)
				if err != nil {
					t.Fatal(err)
				}
				s, err := recording.OpenStore(opts.RecordingsRoot, recording.Limits{MaxSessionBytes: job.Limits.MaxSessionBytes, MaxSegmentBytes: job.Limits.MaxSegmentBytes, ReserveFreeBytes: job.Limits.ReserveFreeBytes})
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { s.Close() })
				w, err := s.Create(m)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { w.Close() })
				if scenario != "no-ready" {
					if err := lease.Ready(m); err != nil {
						t.Fatal(err)
					}
					acknowledged, err = jobs.ReadReady(job.ID, digest)
					if err != nil {
						t.Fatal(err)
					}
				}
				switch scenario {
				case "lost-ack":
					return ErrRecordingSubmissionUncertain
				case "saved-result":
					_, err = lease.RecordResult("capture-incomplete", recording.CaptureResult{SessionID: m.SessionID}, time.Now())
				case "closed-writer":
					err = w.Close()
				case "unexpected-output":
					_, err = w.Append(recording.Stdout, []byte("unexpected pre-Start output"), m.AttachedAt.UnixNano(), 0)
				case "partial-frame":
					file, openErr := os.OpenFile(filepath.Join(opts.RecordingsRoot, m.SessionID, "frames.obr"), os.O_WRONLY|os.O_APPEND, 0)
					if openErr != nil {
						t.Fatal(openErr)
					}
					_, err = file.Write([]byte{1})
					file.Close()
				case "changed-metadata":
					m.RunnerID = "other"
					raw, _ := m.Bytes()
					err = os.WriteFile(filepath.Join(opts.RecordingsRoot, m.SessionID, "metadata.json"), raw, 0600)
				}
				if err != nil {
					t.Fatal(err)
				}
				return nil // inert model of successful notify/main submission
			}
			observe := func(_ context.Context, selected ManagedRecordingJob, claimed RecordingWorkerIdentity) (RecordingSupervisorState, error) {
				observations++
				if selected.ID != job.ID || claimed != worker {
					t.Fatal("live check rebound original worker")
				}
				if scenario == "worker-gone" {
					return unit, ErrRecordingWorker
				}
				got := unit
				if scenario == "wrong-unit" {
					got.Unit = "unrelated.service"
				}
				if scenario == "changed-unit" && observations == 2 {
					got.InvocationID = strings.Repeat("b", 32)
				}
				if observations == 2 && scenario == "changed-claim" {
					path := filepath.Join(opts.JobsRoot, job.ID, "claim.json")
					raw, err := os.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					var claim recordingJobClaim
					if json.Unmarshal(raw, &claim) != nil {
						t.Fatal("invalid fixture claim")
					}
					claim.Worker.StartTicks++
					raw, _ = json.Marshal(claim)
					if err := os.WriteFile(path, raw, 0600); err != nil {
						t.Fatal(err)
					}
				}
				if observations == 2 && scenario == "late-result" {
					if err := os.WriteFile(filepath.Join(opts.JobsRoot, job.ID, "result.json"), []byte("partial result"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				return got, nil
			}
			native := func(_ context.Context, selected ManagedRecordingJob) error {
				nativeChecks++
				if scenario == "native-changed" {
					f.engine.ID = "changed-daemon"
				}
				m, err := prepare(selected)
				if err != nil || !job.MatchesAdmission(m) {
					return ErrRecordingWorker
				}
				return nil
			}
			before := len(f.calls)
			got, err := submitRecordingWorker(t.Context(), opts, job, fence, submit, observe, native)
			if scenario == "pass" {
				if err != nil || got != acknowledged || submits != 1 || observations != 2 || nativeChecks != 1 || fences != 2 {
					t.Fatal("readiness was not completely composed", got, err, submits, observations, nativeChecks, fences)
				}
			} else if scenario == "lost-ack" {
				if err != ErrRecordingSubmissionUncertain || submits != 1 || observations != 0 || nativeChecks != 0 {
					t.Fatal("lost acknowledgement retried or became authority", err)
				}
			} else if err != ErrRecordingWorker {
				t.Fatal("unready/rebound worker admitted or private error leaked", scenario, err)
			}
			if (scenario == "pre-fence" || scenario == "already-claimed") && submits != 0 {
				t.Fatal("submission preceded original-job/fence admission")
			}
			for _, call := range f.calls[before:] {
				if len(call) > 1 && (call[1] == "start" || call[1] == "stop" || call[1] == "kill" || call[1] == "rm") {
					t.Fatal("coordinator controlled native lifecycle", call)
				}
			}
			currentJob, _ := os.ReadFile(filepath.Join(opts.JobsRoot, job.ID, "job.json"))
			if !bytes.Equal(originalJob, currentJob) {
				t.Fatal("coordinator changed the original job")
			}
		})
	}
}

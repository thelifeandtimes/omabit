//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

func TestRecordingWindowPagesOriginalsAndDerivesPrivateRedactionWithoutEffects(t *testing.T) {
	f, opts, job, digest, worker, selected := recordingWorkerFixture(t)
	payload := bytes.Repeat([]byte("trace 0wdiagnostic.atom\nAuthorization: Bearer fixture-secret\n"), 3500)
	stream := &recordingTestStream{Reader: bytes.NewReader(recordingWorkerPacket(payload))}
	result, err := runManagedRecordingWorker(t.Context(), opts, job.ID, digest, func(RecordingJobReady) error {
		s := mapAt(selected.doc, "State")
		s["Status"], s["Running"], s["Pid"] = "exited", false, 0
		s["StartedAt"], s["FinishedAt"] = "2026-11-18T00:00:00Z", "2026-11-18T00:00:01Z"
		s["ExitCode"], s["OOMKilled"] = 0, false
		return nil
	}, f.d, func(*qualifier) (RecordingWorkerIdentity, error) { return worker, nil }, func(context.Context, string, string) (io.ReadCloser, error) { return stream, nil })
	if err != nil || result.Code != "capture-finished" {
		t.Fatal("inert source setup failed", err)
	}
	before, err := os.ReadFile(filepath.Join(opts.JobsRoot, job.ID, "result.json"))
	if err != nil {
		t.Fatal(err)
	}
	in := recording.PageRequest{RunnerID: job.Operation.RunnerID, InstanceID: job.Operation.Target.InstanceID, JobID: job.ID, JobSHA256: digest, Representation: "original", IncludePrivateData: true}
	var got []byte
	exits, pages := 0, 0
	for {
		out, err := readManagedRecordingWindow(t.Context(), opts, in)
		if err != nil {
			t.Fatal(err)
		}
		if out.CompleteExport || out.Session.ConfigurationRevision != job.Operation.ConfigurationRevision || out.Redacted != nil || out.FrameCount > recording.MaxReadFrames || out.Status.ResultCode != "capture-finished" {
			t.Fatal("window invented coverage/representation")
		}
		for _, frame := range out.Original {
			if frame.Frame.Kind == recording.Stderr {
				got = append(got, frame.OriginalBytes...)
			}
			if frame.Frame.Kind == recording.Exit {
				exits++
			}
		}
		pages++
		if out.Tail == "snapshot-eof" {
			break
		}
		if pages > 10 || out.Next == nil || reflect.DeepEqual(out.Next, in.Cursor) {
			t.Fatal("cursor did not progress")
		}
		in.Cursor = out.Next
	}
	if !bytes.Equal(got, payload) || exits != 1 || pages < 2 {
		t.Fatal("original bytes, paging or separate exit lost", pages, exits)
	}
	in.Cursor = nil
	in.Representation = "redacted"
	out, err := readManagedRecordingWindow(t.Context(), opts, in)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(out)
	if out.Redacted == nil || bytes.Contains(raw, []byte("fixture-secret")) || !bytes.Contains(raw, []byte("0wdiagnostic.atom")) || len(out.Original) != 0 {
		t.Fatal("derived view leaked credential or erased diagnostic")
	}
	for _, change := range []string{"no-consent", "wrong-instance", "wrong-job-hash", "foreign-cursor"} {
		bad := in
		switch change {
		case "no-consent":
			bad.IncludePrivateData = false
		case "wrong-instance":
			bad.InstanceID = "other"
		case "wrong-job-hash":
			bad.JobSHA256 = recordingTestHex('e', 64)
		case "foreign-cursor":
			bad.Cursor = &recording.Cursor{Version: 1, SessionID: recordingTestHex('e', 32), MetadataSHA256: recordingTestHex('d', 64), Sequence: 1, Digest: recordingTestHex('c', 64)}
		}
		if _, err := readManagedRecordingWindow(t.Context(), opts, bad); err == nil {
			t.Fatal("bad selection read original", change)
		}
	}
	after, err := os.ReadFile(filepath.Join(opts.JobsRoot, job.ID, "result.json"))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("read changed original result", err)
	}
}

func recordingTestHex(b byte, n int) string { return string(bytes.Repeat([]byte{b}, n)) }

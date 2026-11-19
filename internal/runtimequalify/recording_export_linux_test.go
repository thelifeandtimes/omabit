//go:build linux && amd64

package runtimequalify

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

func TestRecordingExportUsesClosedJobAndPreservesOriginals(t *testing.T) {
	f, opts, job, digest, worker, selected := recordingWorkerFixture(t)
	in := recording.ExportRequest{RunnerID: job.Operation.RunnerID, InstanceID: job.Operation.Target.InstanceID, JobID: job.ID, JobSHA256: digest, Representation: "original", IncludePrivateData: true}
	if _, err := readManagedRecordingExportManifest(t.Context(), opts, in); err == nil {
		t.Fatal("unclaimed job became a complete original export")
	}
	payload := bytes.Repeat([]byte("raw 0wdiagnostic.atom\r\x1b[2K Authorization: private\x00\xff\n"), 7000)
	stream := &recordingTestStream{Reader: bytes.NewReader(recordingWorkerPacket(payload))}
	result, err := runManagedRecordingWorker(t.Context(), opts, job.ID, digest, func(RecordingJobReady) error {
		if _, err := readManagedRecordingExportManifest(t.Context(), opts, in); err == nil {
			t.Fatal("active job became a stationary export")
		}
		state := mapAt(selected.doc, "State")
		state["Status"], state["Running"], state["Pid"] = "exited", false, 0
		state["StartedAt"], state["FinishedAt"] = "2026-11-18T00:00:00Z", "2026-11-18T00:00:01Z"
		state["ExitCode"], state["OOMKilled"] = 0, false
		return nil
	}, f.d, func(*qualifier) (RecordingWorkerIdentity, error) { return worker, nil }, func(context.Context, string, string) (io.ReadCloser, error) { return stream, nil })
	if err != nil || result.Code != "capture-finished" {
		t.Fatal("inert fixture failed", err)
	}
	snapshot := func() map[string][]byte {
		out := map[string][]byte{}
		for _, dir := range []string{opts.JobsRoot, opts.RecordingsRoot} {
			if err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !d.IsDir() {
					data, err := os.ReadFile(path)
					if err != nil {
						return err
					}
					out[path] = data
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
		}
		return out
	}
	before := snapshot()
	manifest, err := readManagedRecordingExportManifest(t.Context(), opts, in)
	if err != nil || manifest.Validate(in) != nil || manifest.Manifest.Coverage.OutputBytes != uint64(len(payload)) || !manifest.Manifest.Coverage.ExitSaved || manifest.Manifest.RuntimeComplete || len(manifest.Manifest.Files) < 5 {
		t.Fatal("closed saved job coverage changed", manifest.Manifest.Coverage, err)
	}
	var archive bytes.Buffer
	if err := recording.WriteOriginalBundle(t.Context(), in, manifest, func(ctx context.Context, request recording.ExportPartRequest) (recording.ExportPart, error) {
		return readManagedRecordingExportPart(ctx, opts, request)
	}, &archive); err != nil {
		t.Fatal(err)
	}
	z, err := zip.NewReader(bytes.NewReader(archive.Bytes()), int64(archive.Len()))
	if err != nil || len(z.File) != len(manifest.Manifest.Files)+2 {
		t.Fatal("bundle did not finish", err)
	}
	file := manifest.Manifest.Files[0]
	part := recording.ExportPartRequest{ExportRequest: in, SessionID: manifest.Manifest.Metadata.SessionID, MetadataSHA256: manifest.Manifest.MetadataSHA256, FileName: file.Name, FileBytes: file.Bytes, ChunkSHA256: file.Chunks[0]}
	for _, change := range []string{"instance", "job", "digest", "session", "metadata", "chunk", "redacted", "no-consent"} {
		bad := part
		switch change {
		case "instance":
			bad.InstanceID = "other"
		case "job":
			bad.JobID = recordingTestHex('e', 32)
		case "digest":
			bad.JobSHA256 = recordingTestHex('e', 64)
		case "session":
			bad.SessionID = recordingTestHex('e', 32)
		case "metadata":
			bad.MetadataSHA256 = recordingTestHex('e', 64)
		case "chunk":
			bad.ChunkSHA256 = recordingTestHex('e', 64)
		case "redacted":
			bad.Representation = "redacted"
		case "no-consent":
			bad.IncludePrivateData = false
		}
		if out, err := readManagedRecordingExportPart(t.Context(), opts, bad); err == nil || out.Data != nil {
			t.Fatal("changed selection disclosed bytes", change)
		}
	}
	if !reflect.DeepEqual(before, snapshot()) {
		t.Fatal("export changed a job, claim, recording or result")
	}
	// Damage only this inert test capture. Even a saved successful worker
	// result must not hide a newly corrupt original prefix or forbid its export.
	path := filepath.Join(opts.RecordingsRoot, manifest.Manifest.Metadata.SessionID, "frames.obr")
	damaged, err := os.ReadFile(path)
	if err != nil || len(damaged) < 201 {
		t.Fatal("fixture too small", err)
	}
	damaged[200] ^= 1
	if err := os.WriteFile(path, damaged, 0600); err != nil {
		t.Fatal(err)
	}
	corrupt, err := readManagedRecordingExportManifest(t.Context(), opts, in)
	if err != nil || corrupt.Status.SessionStorage != "corrupt" || corrupt.Manifest.Coverage.Tail != "corrupt" || corrupt.Manifest.Coverage.VerifiedFrames != 0 || corrupt.Manifest.Coverage.ExitSaved || !corrupt.Manifest.RetainedFilesComplete || corrupt.Status.ResultCode != "capture-finished" {
		t.Fatal("saved outcome concealed new corruption or prevented original export", corrupt.Status, corrupt.Manifest.Coverage, err)
	}
}

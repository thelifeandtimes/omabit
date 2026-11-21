//go:build linux

package recording

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func exportSelection(m Metadata) ExportRequest {
	return ExportRequest{RunnerID: m.RunnerID, InstanceID: m.InstanceID, JobID: strings.Repeat("b", 32), JobSHA256: strings.Repeat("c", 64), Representation: "original", IncludePrivateData: true}
}

func bundleResult(m ExportManifest, in ExportRequest) ExportManifestResult {
	s := m.Metadata
	digest, _ := m.Digest()
	return ExportManifestResult{Session: SessionDescriptor{JobID: in.JobID, JobSHA256: in.JobSHA256, RunnerID: s.RunnerID, InstanceID: s.InstanceID, Mode: s.Mode, ConfigurationRevision: s.ConfigurationRevision, ContainerID: s.ContainerID, OperationID: s.OperationID, RequestID: s.RequestID, CreatedAt: s.AttachedAt}, Status: SavedStatus{State: "capture-incomplete", ResultCode: "capture-incomplete", SessionID: s.SessionID, SessionStorage: "header-available"}, Manifest: m, ManifestSHA256: digest}
}

func TestOriginalBundleStreamsAllSegmentBytesWithTimingAndGapEvidence(t *testing.T) {
	s, dir := segmentStore(t, 4<<20, 350<<10)
	m := captureMetadata()
	w, err := s.Create(m)
	if err != nil {
		t.Fatal(err)
	}
	payload := append(bytes.Repeat([]byte("long trace\n"), 1000), bytes.Repeat([]byte("0wdiagnostic.atom Authorization: private\x00\x1b[2K\xff"), 1000)...)
	for i := 0; i < 18; i++ {
		kind := Stdout
		if i%2 == 1 {
			kind = Stderr
		}
		if _, err := w.Append(kind, payload, m.AttachedAt.UnixNano()-int64(i), uint64(i)); err != nil {
			t.Fatal(err)
		}
	}
	gap, _ := (GapEvent{Version: 1, Reason: "transport-interrupted"}).Bytes()
	if _, err := w.Append(Gap, gap, m.AttachedAt.UnixNano(), 19); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ExportManifest(t.Context(), m.SessionID); err == nil {
		t.Fatal("active writer exported as a stationary session")
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	before := fileSnapshot(t, filepath.Join(dir, m.SessionID))
	manifest, err := s.ExportManifest(t.Context(), m.SessionID)
	if err != nil || manifest.Validate() != nil || len(manifest.Files) < 4 || !manifest.RetainedFilesComplete || manifest.RuntimeComplete || manifest.Coverage.VerifiedFrames != 19 || manifest.Coverage.GapFrames != 1 || manifest.Coverage.ExitSaved || manifest.Coverage.OutputBytes != uint64(18*len(payload)) || manifest.Coverage.FirstUnixNano != m.AttachedAt.UnixNano() {
		t.Fatal("full retained coverage misreported", manifest.Coverage, err)
	}
	selected := exportSelection(m)
	result := bundleResult(manifest, selected)
	var archive bytes.Buffer
	parts := 0
	fetch := func(ctx context.Context, in ExportPartRequest) (ExportPart, error) {
		parts++
		return s.ExportPart(ctx, in)
	}
	if err := WriteOriginalBundle(t.Context(), selected, result, fetch, &archive); err != nil {
		t.Fatal(err)
	}
	if parts < 5 {
		t.Fatal("fixture did not exercise chunk boundaries")
	}
	z, err := zip.NewReader(bytes.NewReader(archive.Bytes()), int64(archive.Len()))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range z.File {
		if file.Mode().Perm() != 0600 {
			t.Fatal("archive entry is not private", file.Name)
		}
		r, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatal(err)
		}
		if original, ok := before[file.Name]; ok && !bytes.Equal(original, got) {
			t.Fatal("encoded original bytes, stream or timing changed", file.Name)
		}
	}
	if len(z.File) != len(manifest.Files)+2 || !reflect.DeepEqual(before, fileSnapshot(t, filepath.Join(dir, m.SessionID))) {
		t.Fatal("bundle omitted a file or changed source evidence")
	}
	for _, change := range []string{"chunk", "file", "request", "cancel"} {
		t.Run(change, func(t *testing.T) {
			chosen := result
			chosen.Manifest.Files = append([]ExportFile(nil), manifest.Files...)
			if change == "file" {
				chosen.Manifest.Files[1].SHA256 = strings.Repeat("d", 64)
				chosen.ManifestSHA256, _ = chosen.Manifest.Digest()
			}
			var partial bytes.Buffer
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			err := WriteOriginalBundle(ctx, selected, chosen, func(ctx context.Context, in ExportPartRequest) (ExportPart, error) {
				part, err := fetch(ctx, in)
				if change == "chunk" && len(part.Data) > 0 {
					part.Data[0] ^= 1
				}
				if change == "request" {
					part.Request.JobID = strings.Repeat("e", 32)
				}
				if change == "cancel" {
					cancel()
				}
				return part, err
			}, &partial)
			if err == nil {
				t.Fatal("failed assembly became completed bundle")
			}
			if _, err := zip.NewReader(bytes.NewReader(partial.Bytes()), int64(partial.Len())); err == nil {
				t.Fatal("partial output received a ZIP completion directory")
			}
		})
	}
}

func TestOriginalExportRetainsTornOrCorruptTailWithoutRepair(t *testing.T) {
	for _, tail := range []string{"incomplete", "corrupt"} {
		t.Run(tail, func(t *testing.T) {
			s, dir := openTestStore(t, 1<<20)
			m := attachmentMetadata() // actual legacy metadata2 single-segment layout
			w, err := s.Create(m)
			if err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 2; i++ {
				if _, err := w.Append(Stdout, []byte("original\n"), m.AttachedAt.UnixNano(), uint64(i)); err != nil {
					t.Fatal(err)
				}
			}
			if err := w.Close(); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, m.SessionID, "frames.obr")
			data, _ := os.ReadFile(path)
			if tail == "incomplete" {
				data = data[:len(data)-1]
			} else {
				data[len(data)-1] ^= 1
			}
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			manifest, err := s.ExportManifest(t.Context(), m.SessionID)
			if err != nil || manifest.Coverage.Tail != tail || manifest.Coverage.VerifiedFrames != 1 || !manifest.RetainedFilesComplete || manifest.RuntimeComplete {
				t.Fatal("unverified tail promoted or hidden", manifest.Coverage, err)
			}
			hash := sha256.Sum256(data)
			if manifest.Files[1].SHA256 != hex.EncodeToString(hash[:]) {
				t.Fatal("broken original tail was not retained")
			}
			got, _ := os.ReadFile(path)
			if !bytes.Equal(got, data) {
				t.Fatal("export repaired original bytes")
			}
		})
	}
}

func TestOriginalExportRejectsUnsafeLayoutAndExactPartChanges(t *testing.T) {
	for _, change := range []string{"symlink", "hardlink", "mode", "extra-file", "changed-chunk", "foreign-session", "foreign-metadata", "redacted", "no-consent", "cancel"} {
		t.Run(change, func(t *testing.T) {
			s, dir := openTestStore(t, 1<<20)
			m := attachmentMetadata()
			w, err := s.Create(m)
			if err != nil {
				t.Fatal(err)
			}
			_, err = w.Append(Stdout, []byte("private\n"), m.AttachedAt.UnixNano(), 0)
			if err != nil {
				t.Fatal(err)
			}
			if err := w.Close(); err != nil {
				t.Fatal(err)
			}
			manifest, err := s.ExportManifest(t.Context(), m.SessionID)
			if err != nil {
				t.Fatal(err)
			}
			file := manifest.Files[1]
			in := ExportPartRequest{ExportRequest: exportSelection(m), SessionID: m.SessionID, MetadataSHA256: manifest.MetadataSHA256, FileName: file.Name, FileBytes: file.Bytes, ChunkIndex: 0, ChunkSHA256: file.Chunks[0]}
			path := filepath.Join(dir, m.SessionID, file.Name)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			switch change {
			case "symlink":
				if err := os.Rename(path, path+".saved"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(path+".saved", path); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err := os.Link(path, path+".link"); err != nil {
					t.Fatal(err)
				}
			case "mode":
				if err := os.Chmod(path, 0644); err != nil {
					t.Fatal(err)
				}
			case "extra-file":
				if err := os.WriteFile(filepath.Join(dir, m.SessionID, "unexpected"), []byte("extra"), 0600); err != nil {
					t.Fatal(err)
				}
				if _, err := s.ExportManifest(ctx, m.SessionID); err == nil {
					t.Fatal("extra retained file silently excluded")
				}
				return
			case "changed-chunk":
				in.ChunkSHA256 = strings.Repeat("e", 64)
			case "foreign-session":
				in.SessionID = strings.Repeat("e", 32)
			case "foreign-metadata":
				in.MetadataSHA256 = strings.Repeat("e", 64)
			case "redacted":
				in.Representation = "redacted"
			case "no-consent":
				in.IncludePrivateData = false
			case "cancel":
				cancel()
			}
			out, err := s.ExportPart(ctx, in)
			if err == nil || out.Data != nil {
				t.Fatal("unsafe selection released original bytes")
			}
		})
	}
}

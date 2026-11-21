//go:build linux

package recording

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func redactedBundleFixture(t *testing.T) (*Store, string, RedactedExportRequest, ExportManifestResult) {
	t.Helper()
	s, dir := segmentStore(t, 4<<20, 350<<10)
	m := captureMetadata()
	m.RunnerID, m.InstanceID, m.OperationID, m.RequestID = "runner-private-binding", "ship-private-binding", "operation-private-binding", "request-private-binding"
	m.Identity.Target.InstanceID = m.InstanceID
	w, err := s.Create(m)
	if err != nil {
		t.Fatal(err)
	}
	sequence := uint64(0)
	appendFrame := func(kind Kind, data []byte) {
		t.Helper()
		sequence++
		if _, err := w.Append(kind, data, m.AttachedAt.UnixNano()-int64(sequence), sequence); err != nil {
			t.Fatal(err)
		}
	}
	appendFrame(Stdout, []byte("leading partial\n0wdiagnostic.atom retained\nAuthorization: very-private-token\nCookie: hidden-cookie\nsafe after\n"))
	appendFrame(Stderr, []byte("leading stderr\nordinary stderr\n"))
	for i := 0; i < 61; i++ {
		appendFrame(Stdout, []byte("diagnostic background\n"))
	}
	appendFrame(Stdout, []byte("split q7-very-"))
	appendFrame(Stdout, []byte("private-literal complete\n0wafter-boundary.atom\n"))
	gap, _ := (GapEvent{Version: 1, Reason: "transport-interrupted"}).Bytes()
	appendFrame(Gap, gap)
	appendFrame(Stdout, []byte("partial after gap\nsafe post-gap\n"))
	exit, err := (ExitEvent{Version: 1, ContainerID: m.ContainerID, StartedUnixNano: m.AttachedAt.UnixNano(), FinishedUnixNano: m.AttachedAt.UnixNano() + 1, Code: 137, OOMKilled: true}).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	appendFrame(Exit, exit)
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	manifest, err := s.ExportManifest(t.Context(), m.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	original := exportSelection(m)
	in := RedactedExportRequest{RunnerID: original.RunnerID, InstanceID: original.InstanceID, JobID: original.JobID, JobSHA256: original.JobSHA256, Representation: "redacted", IncludePrivateData: true}
	return s, filepath.Join(dir, m.SessionID), in, bundleResult(manifest, original)
}

func TestRedactedBundleContainsDerivedTimingOmissionsAndSanitizedExitOnly(t *testing.T) {
	s, root, in, result := redactedBundleFixture(t)
	before := fileSnapshot(t, root)
	var output bytes.Buffer
	summary, err := WriteRedactedBundle(t.Context(), in, result, s.ExportPart, [][]byte{[]byte("q7-very-private-literal")}, &output)
	if err != nil || summary.Representation != "redacted" || summary.Frames != 68 || summary.Coverage.GapFrames != 1 || !summary.Coverage.ExitSaved || !summary.VerifiedPrefixTransformed || !summary.AllSourceFilesVerified || summary.RuntimeComplete || summary.UnparsedRawBytesIncluded || summary.OmittedPayloadBytes == 0 {
		t.Fatal("derived coverage wrong", err, summary)
	}
	z, err := zip.NewReader(bytes.NewReader(output.Bytes()), int64(output.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if len(z.File) != 3 {
		t.Fatal("unexpected raw/derived entry count")
	}
	var windows int
	var frames uint64
	var spans string
	var contentBytes uint64
	var footer RedactedBundleSummary
	foundExit, foundGap := false, false
	for _, file := range z.File {
		if file.Mode().Perm() != 0600 {
			t.Fatal("non-private derived entry")
		}
		r, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{in.RunnerID, in.InstanceID, result.Session.OperationID, result.Session.RequestID, result.Session.ContainerID, "very-private-token", "hidden-cookie", "q7-very-", "private-literal", "partial after gap"} {
			if bytes.Contains(data, []byte(forbidden)) {
				t.Fatal("derived export leaked excluded bindings or payload", file.Name, forbidden)
			}
		}
		if bytes.Contains(data, []byte{0x1b}) {
			t.Fatal("derived export contains terminal controls")
		}
		switch file.Name {
		case "manifest.json":
			var meta RedactedBundleManifest
			if json.Unmarshal(data, &meta) != nil || meta.Representation != "redacted" || !meta.ExactBindingsOmitted || meta.OriginalEncodedFilesIncluded || meta.ReviewRequired == "" {
				t.Fatal("wrong derived manifest")
			}
			contentBytes += uint64(len(data))
		case "redacted-frames.jsonl":
			contentBytes += uint64(len(data))
			h := sha256.Sum256(data)
			if hex.EncodeToString(h[:]) != summary.FramesSHA256 {
				t.Fatal("derived frame hash wrong")
			}
			decoder := json.NewDecoder(bytes.NewReader(data))
			for {
				var window RedactedBundleWindow
				err := decoder.Decode(&window)
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				windows++
				for _, frame := range window.Frames {
					frames++
					if frame.Timing.Sequence != frames {
						t.Fatal("derived timing/sequence missing")
					}
					for _, span := range frame.Spans {
						spans += span.Text
					}
					if frame.Exit != nil {
						foundExit = frame.Exit.ContainerReference == "container-1" && frame.Exit.OOMKilled && frame.Exit.Code == 137
					}
					if frame.Gap != nil {
						foundGap = frame.Gap.Reason == "transport-interrupted"
					}
				}
			}
		case "coverage.json":
			if json.Unmarshal(data, &footer) != nil {
				t.Fatal("bad final coverage")
			}
		default:
			t.Fatal("raw metadata/recording included in redacted ZIP", file.Name)
		}
	}
	if windows < 2 || frames != summary.Frames || !foundExit || !foundGap || !strings.Contains(spans, "0wdiagnostic.atom retained") || !strings.Contains(spans, "0wafter-boundary.atom") || !strings.Contains(spans, "safe post-gap") || footer != summary || contentBytes != summary.DerivedBytes {
		t.Fatal("omission/context/timing evidence lost", windows, frames, footer, summary)
	}
	if !reflect.DeepEqual(before, fileSnapshot(t, root)) {
		t.Fatal("redacted export changed original files")
	}
}

func TestRedactedBundleChoiceAuthorityAndSourceFailureNeverFallThrough(t *testing.T) {
	s, root, in, result := redactedBundleFixture(t)
	before := fileSnapshot(t, root)
	for _, variant := range []string{"original-choice", "no-consent", "multiline-literal", "whole-hash", "writer"} {
		t.Run(variant, func(t *testing.T) {
			request, copy := in, result
			copy.Manifest.Files = append([]ExportFile(nil), result.Manifest.Files...)
			var secrets [][]byte
			switch variant {
			case "original-choice":
				request.Representation = "original"
			case "no-consent":
				request.IncludePrivateData = false
			case "multiline-literal":
				secrets = [][]byte{[]byte("split\nsecret")}
			case "whole-hash":
				copy.Manifest.Files[1].SHA256 = strings.Repeat("f", 64)
			}
			var err error
			copy.ManifestSHA256, err = copy.Manifest.Digest()
			if err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			var writer io.Writer = &output
			if variant == "writer" {
				writer = timingShortWriter{}
			}
			calls := 0
			fetch := func(ctx context.Context, in ExportPartRequest) (ExportPart, error) {
				calls++
				return s.ExportPart(ctx, in)
			}
			got, err := WriteRedactedBundle(t.Context(), request, copy, fetch, secrets, writer)
			if err == nil || got != (RedactedBundleSummary{}) {
				t.Fatal("invalid redacted choice completed", variant, err)
			}
			if variant == "original-choice" || variant == "no-consent" || variant == "multiline-literal" {
				if calls != 0 || output.Len() != 0 {
					t.Fatal("refusal read/exported originals")
				}
			}
			if _, err := zip.NewReader(bytes.NewReader(output.Bytes()), int64(output.Len())); err == nil {
				t.Fatal("partial redacted archive completed")
			}
			if !reflect.DeepEqual(before, fileSnapshot(t, root)) {
				t.Fatal("redacted failure mutated originals")
			}
		})
	}
}

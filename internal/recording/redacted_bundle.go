package recording

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
)

const MaxRedactedBundleBytes = 256 << 20
const MaxRedactedRecordBytes = 4 << 20

// RedactedExportRequest cannot be supplied to an original formatter. Conversion
// to original reads is private to the verified transformation, never a fallback
// publishing original bytes for a redacted choice.
type RedactedExportRequest struct {
	RunnerID           string `json:"runner_id"`
	InstanceID         string `json:"instance_id"`
	JobID              string `json:"job_id"`
	JobSHA256          string `json:"job_sha256"`
	Representation     string `json:"representation"`
	IncludePrivateData bool   `json:"include_private_data"`
}

func (r RedactedExportRequest) original() ExportRequest {
	return ExportRequest{RunnerID: r.RunnerID, InstanceID: r.InstanceID, JobID: r.JobID, JobSHA256: r.JobSHA256, Representation: "original", IncludePrivateData: r.IncludePrivateData}
}
func (r RedactedExportRequest) Validate() error {
	if r.Representation != "redacted" || r.original().Validate() != nil {
		return ErrSelection
	}
	return nil
}

type RedactedBundleManifest struct {
	Version                      int          `json:"version"`
	Representation               string       `json:"representation"`
	Policy                       string       `json:"policy"`
	SourceSHA256                 string       `json:"source_sha256"`
	MetadataSHA256               string       `json:"metadata_sha256"`
	RuntimeVersion               string       `json:"runtime_version"`
	ImageManifestDigest          string       `json:"image_manifest_digest"`
	Transport                    string       `json:"transport"`
	ConfigurationRevision        uint64       `json:"configuration_revision"`
	AttachedUnixNano             int64        `json:"attached_unix_nano"`
	SourceFiles                  []TimingFile `json:"source_file_hashes"`
	SelectedSingleLineLiterals   int          `json:"selected_single_line_literals"`
	ExactBindingsOmitted         bool         `json:"exact_bindings_omitted"`
	OriginalEncodedFilesIncluded bool         `json:"original_encoded_files_included"`
	ReviewRequired               string       `json:"review_required"`
}

type RedactedExit struct {
	ContainerBindingOmitted bool   `json:"container_binding_omitted"`
	Version                 int    `json:"version"`
	ContainerReference      string `json:"container_reference"`
	StartedUnixNano         int64  `json:"started_unix_nano"`
	FinishedUnixNano        int64  `json:"finished_unix_nano"`
	Code                    int    `json:"code"`
	OOMKilled               bool   `json:"oom_killed"`
	Dead                    bool   `json:"dead"`
	DockerErrorPresent      bool   `json:"docker_error_present"`
}

type RedactedBundleFrame struct {
	Timing TimingEntry   `json:"timing"`
	Spans  []ViewSpan    `json:"spans,omitempty"`
	Gap    *GapEvent     `json:"gap,omitempty"`
	Exit   *RedactedExit `json:"exit,omitempty"`
}

type RedactedBundleWindow struct {
	Version        int                   `json:"version"`
	Policy         string                `json:"policy"`
	Representation string                `json:"representation"`
	OriginalBytes  uint64                `json:"verified_frame_payload_bytes"`
	OmittedBytes   uint64                `json:"omitted_stream_bytes"`
	Frames         []RedactedBundleFrame `json:"frames"`
}

type RedactedBundleSummary struct {
	ExitBindingsAliased       uint64         `json:"exit_bindings_aliased"`
	Version                   int            `json:"version"`
	Representation            string         `json:"representation"`
	SourceSHA256              string         `json:"source_sha256"`
	FramesSHA256              string         `json:"frames_sha256"`
	Frames                    uint64         `json:"frames"`
	OriginalPayloadBytes      uint64         `json:"verified_frame_payload_bytes"`
	OmittedPayloadBytes       uint64         `json:"omitted_stream_bytes"`
	DerivedBytes              uint64         `json:"derived_content_bytes"`
	Coverage                  ExportCoverage `json:"coverage"`
	AllSourceFilesVerified    bool           `json:"all_source_files_verified"`
	VerifiedPrefixTransformed bool           `json:"verified_prefix_transformed"`
	UnparsedRawBytesIncluded  bool           `json:"unparsed_raw_bytes_included"`
	ExactBindingsOmitted      bool           `json:"exact_bindings_omitted"`
	RuntimeComplete           bool           `json:"runtime_complete"`
}

// WriteRedactedBundle writes only derived escaped views and technical summaries.
// Each finite window withholds partial leading/trailing lines; stream continuity
// evidence comes only from verified prior bytes, and gaps reset it. Optional
// literals apply to stream payload only and must be single-line so a literal
// cannot straddle an emitted newline/window boundary. Raw originals stay private
// and unchanged. Owners must review finite-rule redaction before sharing.
func WriteRedactedBundle(ctx context.Context, in RedactedExportRequest, result ExportManifestResult, fetch func(context.Context, ExportPartRequest) (ExportPart, error), knownSecrets [][]byte, output io.Writer) (RedactedBundleSummary, error) {
	var zero RedactedBundleSummary
	if in.Validate() != nil || result.Validate(in.original()) != nil || output == nil || validateRedactionOptions(RedactionOptions{KnownSecrets: knownSecrets}) != nil {
		return zero, ErrSelection
	}
	secrets := make([][]byte, len(knownSecrets))
	for i, value := range knownSecrets {
		if bytes.ContainsAny(value, "\r\n") {
			return zero, ErrSelection
		}
		secrets[i] = append([]byte(nil), value...)
	}
	z := zip.NewWriter(output)
	summary := RedactedBundleSummary{Version: 1, Representation: "redacted", ExactBindingsOmitted: true}
	var frameWriter io.Writer
	frameHash := sha256.New()
	var pending []Frame
	var timings []TimingEntry
	var pendingBytes int
	lineStarts := map[Kind]bool{}
	writeEntry := func(name string, data []byte) error {
		if uint64(len(data)) > MaxRedactedBundleBytes-summary.DerivedBytes {
			return ErrLimit
		}
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetMode(0600)
		entry, err := z.CreateHeader(header)
		if err != nil {
			return err
		}
		if err = write(entry, data); err != nil {
			return err
		}
		if name != "coverage.json" {
			summary.DerivedBytes += uint64(len(data))
		}
		return nil
	}
	flush := func() error {
		if len(pending) == 0 {
			return nil
		}
		options := RedactionOptions{KnownSecrets: secrets}
		for _, kind := range []Kind{Stdout, Stderr, PTYCombined} {
			if lineStarts[kind] {
				options.VerifiedLineStarts = append(options.VerifiedLineStarts, kind)
			}
		}
		view, err := RenderRedactedFrames(pending, options)
		if err != nil {
			return err
		}
		window := RedactedBundleWindow{Version: 1, Policy: view.Policy, Representation: view.Representation, OriginalBytes: view.OriginalBytes, OmittedBytes: view.OmittedBytes}
		for i, frame := range view.Frames {
			derived := RedactedBundleFrame{Timing: timings[i], Spans: frame.Spans, Gap: frame.Gap}
			if frame.Exit != nil {
				e := frame.Exit
				derived.Exit = &RedactedExit{ContainerBindingOmitted: true, Version: e.Version, ContainerReference: "container-1", StartedUnixNano: e.StartedUnixNano, FinishedUnixNano: e.FinishedUnixNano, Code: e.Code, OOMKilled: e.OOMKilled, Dead: e.Dead, DockerErrorPresent: e.DockerErrorPresent}
			}
			if derived.Exit != nil {
				summary.ExitBindingsAliased++
			}
			window.Frames = append(window.Frames, derived)
		}
		data, err := json.Marshal(window)
		if err != nil {
			return err
		}
		data = append(data, '\n')
		if len(data) > MaxRedactedRecordBytes || uint64(len(data)) > MaxRedactedBundleBytes-summary.DerivedBytes {
			return ErrLimit
		}
		if err = write(frameWriter, data); err != nil {
			return err
		}
		frameHash.Write(data)
		summary.DerivedBytes += uint64(len(data))
		summary.OriginalPayloadBytes += view.OriginalBytes
		summary.OmittedPayloadBytes += view.OmittedBytes
		for _, frame := range pending {
			if frame.Kind == Gap || frame.Kind == Exit {
				clear(lineStarts)
			} else {
				lineStarts[frame.Kind] = len(frame.Data) > 0 && frame.Data[len(frame.Data)-1] == '\n'
			}
		}
		pending, timings, pendingBytes = nil, nil, 0
		return nil
	}
	verified, err := scanExportFrames(ctx, in.original(), result, fetch, func(source TimingSource, sourceSHA string) error {
		manifest := RedactedBundleManifest{Version: 1, Representation: "redacted", Policy: "credential-fields-and-selected-literals-v1", SourceSHA256: sourceSHA, MetadataSHA256: source.MetadataSHA256, RuntimeVersion: result.Manifest.Metadata.RuntimeVersion, ImageManifestDigest: result.Manifest.Metadata.ImageManifestDigest, Transport: source.Transport, ConfigurationRevision: source.Session.ConfigurationRevision, AttachedUnixNano: result.Manifest.Metadata.AttachedAt.UnixNano(), SourceFiles: source.Files, SelectedSingleLineLiterals: len(secrets), ExactBindingsOmitted: true, ReviewRequired: "Finite credential/single-line-literal payload rules cannot identify every secret or private application value. Review before sharing. Exact names/IDs/paths, raw metadata and encoded originals are omitted; technical strings, hashes and timing may remain sensitive and do not grant access or establish authenticity."}
		data, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			return err
		}
		if err = writeEntry("manifest.json", append(data, '\n')); err != nil {
			return err
		}
		header := &zip.FileHeader{Name: "redacted-frames.jsonl", Method: zip.Deflate}
		header.SetMode(0600)
		frameWriter, err = z.CreateHeader(header)
		return err
	}, func(entry TimingEntry, frame Frame) error {
		if len(pending) > 0 && (len(pending) == 64 || pendingBytes+len(frame.Data) > 128<<10) {
			if err := flush(); err != nil {
				return err
			}
		}
		pending = append(pending, frame)
		timings = append(timings, entry)
		pendingBytes += len(frame.Data)
		return nil
	})
	if err != nil {
		return zero, err
	}
	if err = flush(); err != nil {
		return zero, err
	}
	summary.SourceSHA256 = verified.sourceSHA
	summary.Frames = verified.coverage.VerifiedFrames
	summary.Coverage = verified.coverage
	summary.AllSourceFilesVerified, summary.VerifiedPrefixTransformed = true, true
	summary.FramesSHA256 = hex.EncodeToString(frameHash.Sum(nil))
	// The final coverage entry and ZIP directory appear only after all original
	// hashes/coverage and transformations agree. Any earlier output is partial.
	footer, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return zero, err
	}
	if err = writeEntry("coverage.json", append(footer, '\n')); err != nil {
		return zero, err
	}
	if err = ctx.Err(); err != nil {
		return zero, err
	}
	if err = z.Close(); err != nil {
		return zero, err
	}
	return summary, nil
}

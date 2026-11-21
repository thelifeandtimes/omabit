package recording

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// Parts fit the existing 1MiB runner transport after JSON base64 encoding.
const ExportChunkBytes = 256 << 10
const MaxExportChunks = ((1 << 30) / ExportChunkBytes) + 1024
const MaxExportManifestBytes = 900 << 10

type ExportRequest struct {
	RunnerID           string `json:"runner_id"`
	InstanceID         string `json:"instance_id"`
	JobID              string `json:"job_id"`
	JobSHA256          string `json:"job_sha256"`
	Representation     string `json:"representation"`
	IncludePrivateData bool   `json:"include_private_data"`
}

func (r ExportRequest) PageSelection() PageRequest {
	return PageRequest{RunnerID: r.RunnerID, InstanceID: r.InstanceID, JobID: r.JobID, JobSHA256: r.JobSHA256, Representation: r.Representation, IncludePrivateData: r.IncludePrivateData}
}
func (r ExportRequest) Validate() error {
	// This API exports exact encoded originals only. A redacted choice must
	// never fall through to originals; derived export packaging is separate.
	if r.Representation != "original" || r.PageSelection().Validate() != nil {
		return ErrSelection
	}
	return nil
}

type ExportFile struct {
	Name   string   `json:"name"`
	Bytes  int64    `json:"bytes"`
	SHA256 string   `json:"sha256"`
	Chunks []string `json:"chunk_sha256"`
}

// Coverage describes only the verified frame prefix. All retained file bytes,
// including an incomplete/corrupt tail, are independently hashed for export.
// An exit or an EOF cannot prove output before attachment was captured.
type ExportCoverage struct {
	VerifiedFrames uint64 `json:"verified_frames"`
	OutputBytes    uint64 `json:"output_bytes"`
	GapFrames      uint64 `json:"gap_frames"`
	ExitSaved      bool   `json:"exit_saved"`
	FirstUnixNano  int64  `json:"first_unix_nano"`
	LastUnixNano   int64  `json:"last_unix_nano"`
	Tail           string `json:"tail"`
}

type ExportManifest struct {
	Version               int            `json:"version"`
	Metadata              Metadata       `json:"metadata"`
	MetadataSHA256        string         `json:"metadata_sha256"`
	Representation        string         `json:"representation"`
	ChunkBytes            int            `json:"chunk_bytes"`
	Files                 []ExportFile   `json:"files"`
	Coverage              ExportCoverage `json:"coverage"`
	RetainedFilesComplete bool           `json:"retained_files_complete"`
	RuntimeComplete       bool           `json:"runtime_complete"`
	ReadAt                time.Time      `json:"read_at"`
	Basis                 string         `json:"basis"`
}

func exportName(number int) string {
	if number == 0 {
		return "metadata.json"
	}
	if number == 1 {
		return "frames.obr"
	}
	return fmt.Sprintf("frames-%08d.obr", number)
}

func validExportName(name string) bool {
	for i := 0; i <= 1024; i++ {
		if name == exportName(i) {
			return true
		}
	}
	return false
}

func (m ExportManifest) Validate() error {
	if m.Version != 1 || m.Metadata.Validate() != nil || m.Representation != "original" || m.ChunkBytes != ExportChunkBytes || !m.RetainedFilesComplete || m.RuntimeComplete || m.ReadAt.IsZero() || m.Basis == "" || len(m.Files) < 2 || len(m.Files) > 1025 {
		return ErrSelection
	}
	raw, _ := m.Metadata.Bytes()
	digest := sha256.Sum256(raw)
	if hex.EncodeToString(digest[:]) != m.MetadataSHA256 {
		return ErrSelection
	}
	var total int64
	chunks := 0
	for i, f := range m.Files {
		if f.Name != exportName(i) || f.Bytes <= 0 || f.Bytes > 1<<30 || !APIHex(f.SHA256, 64) || int64(len(f.Chunks)) != (f.Bytes+ExportChunkBytes-1)/ExportChunkBytes {
			return ErrSelection
		}
		if i == 0 && (f.Bytes != int64(len(raw)) || f.SHA256 != m.MetadataSHA256) {
			return ErrSelection
		}
		if i > 0 {
			total += f.Bytes
		}
		chunks += len(f.Chunks)
		for _, hash := range f.Chunks {
			if !APIHex(hash, 64) {
				return ErrSelection
			}
		}
	}
	if total > 1<<30 || chunks > MaxExportChunks || m.Metadata.Version < 3 && len(m.Files) != 2 {
		return ErrSelection
	}
	c := m.Coverage
	if c.GapFrames > c.VerifiedFrames || c.ExitSaved && c.VerifiedFrames == 0 || c.VerifiedFrames == 0 && (c.FirstUnixNano != 0 || c.LastUnixNano != 0 || c.OutputBytes != 0) || c.VerifiedFrames > 0 && (c.FirstUnixNano <= 0 || c.LastUnixNano <= 0) {
		return ErrSelection
	}
	switch c.Tail {
	case "snapshot-eof", "incomplete", "corrupt":
	default:
		return ErrSelection
	}
	b, err := json.Marshal(m)
	if err != nil || len(b) > MaxExportManifestBytes {
		return ErrLimit
	}
	return nil
}

func (m ExportManifest) Digest() (string, error) {
	if err := m.Validate(); err != nil {
		return "", err
	}
	b, _ := json.Marshal(m)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

type ExportManifestResult struct {
	Session        SessionDescriptor `json:"session"`
	Status         SavedStatus       `json:"status"`
	Manifest       ExportManifest    `json:"manifest"`
	ManifestSHA256 string            `json:"manifest_sha256"`
}

func (out ExportManifestResult) Validate(in ExportRequest) error {
	s, m := out.Session, out.Manifest.Metadata
	digest, err := out.Manifest.Digest()
	storage := out.Status.SessionStorage
	if in.Validate() != nil || err != nil || digest != out.ManifestSHA256 || s.JobID != in.JobID || s.JobSHA256 != in.JobSHA256 || s.RunnerID != in.RunnerID || s.InstanceID != in.InstanceID || s.ConfigurationRevision == 0 || s.CreatedAt.IsZero() || m.RunnerID != s.RunnerID || m.InstanceID != s.InstanceID || m.Mode != s.Mode || m.ConfigurationRevision != s.ConfigurationRevision || m.ContainerID != s.ContainerID || m.OperationID != s.OperationID || m.RequestID != s.RequestID || out.Status.SessionID != m.SessionID || out.Status.WriterPresent || storage != "header-available" && storage != "incomplete" && storage != "corrupt" || out.Status.State != out.Status.ResultCode || out.Status.ResultCode != "capture-finished" && out.Status.ResultCode != "capture-incomplete" {
		return ErrSelection
	}
	return nil
}

type ExportPartRequest struct {
	ExportRequest
	SessionID      string `json:"session_id"`
	MetadataSHA256 string `json:"metadata_sha256"`
	FileName       string `json:"file_name"`
	FileBytes      int64  `json:"file_bytes"`
	ChunkIndex     int    `json:"chunk_index"`
	ChunkSHA256    string `json:"chunk_sha256"`
}

func (r ExportPartRequest) Validate() error {
	if r.ExportRequest.Validate() != nil || !APIHex(r.SessionID, 32) || !APIHex(r.MetadataSHA256, 64) || !validExportName(r.FileName) || r.FileBytes <= 0 || r.FileBytes > 1<<30 || r.FileName == "metadata.json" && r.FileBytes > MaxMetadataBytes || r.ChunkIndex < 0 || int64(r.ChunkIndex) >= (r.FileBytes+ExportChunkBytes-1)/ExportChunkBytes || !APIHex(r.ChunkSHA256, 64) {
		return ErrSelection
	}
	return nil
}

type ExportPart struct {
	Version int               `json:"version"`
	Request ExportPartRequest `json:"request"`
	Data    []byte            `json:"data"`
	ReadAt  time.Time         `json:"read_at"`
}

func (p ExportPart) Validate() error {
	if p.Version != 1 || p.Request.Validate() != nil || p.ReadAt.IsZero() || int64(len(p.Data)) != min(int64(ExportChunkBytes), p.Request.FileBytes-int64(p.Request.ChunkIndex)*ExportChunkBytes) {
		return ErrSelection
	}
	h := sha256.Sum256(p.Data)
	if hex.EncodeToString(h[:]) != p.Request.ChunkSHA256 {
		return ErrSelection
	}
	return nil
}

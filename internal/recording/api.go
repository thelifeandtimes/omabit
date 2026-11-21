package recording

import (
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

const MaxCatalogSessions = 32
const MaxReadFrames = 64
const MaxReadBytes = 128 << 10

var ErrSelection = errors.New("invalid exact private recording selection")

func APIHex(s string, n int) bool {
	if len(s) != n || s != strings.ToLower(s) || s == strings.Repeat("0", n) {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

type CatalogRequest struct {
	RunnerID           string `json:"runner_id"`
	InstanceID         string `json:"instance_id"`
	AfterJobID         string `json:"after_job_id,omitempty"`
	IncludePrivateData bool   `json:"include_private_data"`
}

func (r CatalogRequest) Validate() error {
	if !protocol.ValidID(r.RunnerID) || !protocol.ValidID(r.InstanceID) || r.AfterJobID != "" && !APIHex(r.AfterJobID, 32) || !r.IncludePrivateData {
		return ErrSelection
	}
	return nil
}

type SessionDescriptor struct {
	JobID                 string    `json:"job_id"`
	JobSHA256             string    `json:"job_sha256"`
	RunnerID              string    `json:"runner_id"`
	Mode                  string    `json:"mode"`
	InstanceID            string    `json:"instance_id"`
	RuntimeID             string    `json:"runtime_id"`
	ConfigurationRevision uint64    `json:"configuration_revision"`
	ContainerID           string    `json:"container_id"`
	OperationID           string    `json:"operation_id"`
	RequestID             string    `json:"request_id"`
	CreatedAt             time.Time `json:"created_at"`
}

type Catalog struct {
	Version    int                 `json:"version"`
	RunnerID   string              `json:"runner_id"`
	InstanceID string              `json:"instance_id"`
	Sessions   []SessionDescriptor `json:"sessions"`
	NextJobID  string              `json:"next_job_id,omitempty"`
	ReadAt     time.Time           `json:"read_at"`
	Basis      string              `json:"basis"`
}

type PageRequest struct {
	RunnerID           string  `json:"runner_id"`
	InstanceID         string  `json:"instance_id"`
	JobID              string  `json:"job_id"`
	JobSHA256          string  `json:"job_sha256"`
	Cursor             *Cursor `json:"cursor,omitempty"`
	Representation     string  `json:"representation"`
	IncludePrivateData bool    `json:"include_private_data"`
}

func (r PageRequest) Validate() error {
	if !protocol.ValidID(r.RunnerID) || !protocol.ValidID(r.InstanceID) || !APIHex(r.JobID, 32) || !APIHex(r.JobSHA256, 64) || !r.IncludePrivateData || r.Representation != "redacted" && r.Representation != "original" {
		return ErrSelection
	}
	if r.Cursor != nil && (r.Cursor.Version != 1 || !APIHex(r.Cursor.SessionID, 32) || !APIHex(r.Cursor.MetadataSHA256, 64) || r.Cursor.Sequence == 0 || !APIHex(r.Cursor.Digest, 64)) {
		return ErrSelection
	}
	return nil
}

type SavedStatus struct {
	State          string `json:"state"`
	SessionID      string `json:"session_id,omitempty"`
	SessionIDBasis string `json:"session_id_basis"`
	SessionStorage string `json:"session_storage"`
	WriterPresent  bool   `json:"writer_present"`
	ResultCode     string `json:"result_code,omitempty"`
	ClosedDurably  bool   `json:"closed_durably"`
	ExitSaved      bool   `json:"exit_saved"`
	GapReason      string `json:"gap_reason,omitempty"`
}

// ReadWindow is operator-private even when derived redaction is selected. It is
// a bounded verified prefix, never a complete export or a current health claim.
// Original terminal bytes are base64 only. Escaped view spans require plain text.
type ReadWindow struct {
	Version        int                   `json:"version"`
	Session        SessionDescriptor     `json:"session"`
	Status         SavedStatus           `json:"status"`
	Metadata       *Metadata             `json:"metadata,omitempty"`
	MetadataSHA256 string                `json:"metadata_sha256,omitempty"`
	Representation string                `json:"representation"`
	From           *Cursor               `json:"from,omitempty"`
	Next           *Cursor               `json:"next,omitempty"`
	Tail           string                `json:"tail"`
	FrameCount     int                   `json:"frame_count"`
	Original       []OriginalFrameExport `json:"original,omitempty"`
	Redacted       *RedactedView         `json:"redacted,omitempty"`
	ReadAt         time.Time             `json:"read_at"`
	Basis          string                `json:"basis"`
	CompleteExport bool                  `json:"complete_export"`
}

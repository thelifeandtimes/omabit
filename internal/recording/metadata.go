package recording

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

const MaxMetadataBytes = 4096

// Metadata binds one recorder attachment, not a successful Start or proven
// coverage. The owner-private storage layer must validate the actual selected
// runner/configuration/operation and prepared container before capture. There
// are deliberately no argv, environment, session-cookie or secret-content fields.
type Metadata struct {
	Version               int       `json:"version"`
	SessionID             string    `json:"session_id"`
	PreviousSessionID     string    `json:"previous_session_id,omitempty"`
	RunnerID              string    `json:"runner_id"`
	Mode                  string    `json:"mode"`
	InstanceID            string    `json:"instance_id"`
	ProjectID             string    `json:"project_id,omitempty"`
	ConfigurationRevision uint64    `json:"configuration_revision"`
	OperationID           string    `json:"operation_id"`
	RequestID             string    `json:"request_id"`
	ContainerID           string    `json:"container_id"`
	LaunchContractSHA256  string    `json:"launch_contract_sha256"`
	ImageManifestDigest   string    `json:"image_manifest_digest"`
	RuntimeVersion        string    `json:"runtime_version"`
	Transport             string    `json:"transport"`
	AttachedAt            time.Time `json:"attached_at"`
}

func NewSessionID() (string, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", err
	}
	if id == [16]byte{} {
		return "", ErrInvalid
	}
	return hex.EncodeToString(id[:]), nil
}

func (m Metadata) Validate() error {
	if m.Version != 1 || !lowerHex(m.SessionID, 32) || m.SessionID == strings.Repeat("0", 32) || m.PreviousSessionID != "" && (!lowerHex(m.PreviousSessionID, 32) || m.PreviousSessionID == m.SessionID || m.PreviousSessionID == strings.Repeat("0", 32)) || m.ConfigurationRevision == 0 {
		return ErrInvalid
	}
	for _, v := range []string{m.RunnerID, m.Mode, m.InstanceID, m.OperationID, m.RequestID} {
		if !protocol.ValidID(v) {
			return ErrInvalid
		}
	}
	if !protocol.ValidID(strings.NewReplacer(".", "_", "+", "_").Replace(m.RuntimeVersion)) {
		return ErrInvalid
	}
	if m.ProjectID != "" && !protocol.ValidID(m.ProjectID) {
		return ErrInvalid
	}
	for _, v := range []string{m.ContainerID, m.LaunchContractSHA256} {
		if !lowerHex(v, 64) || v == strings.Repeat("0", 64) {
			return ErrInvalid
		}
	}
	if !strings.HasPrefix(m.ImageManifestDigest, "sha256:") || !lowerHex(strings.TrimPrefix(m.ImageManifestDigest, "sha256:"), 64) || m.ImageManifestDigest == "sha256:"+strings.Repeat("0", 64) {
		return ErrInvalid
	}
	if m.Transport != "docker-multiplex" && m.Transport != "pty-combined" {
		return ErrInvalid
	}
	if m.AttachedAt.Before(time.Unix(0, 1)) || m.AttachedAt.After(time.Unix(0, math.MaxInt64)) {
		return ErrInvalid
	}
	return nil
}

func (m Metadata) Bytes() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	m.AttachedAt = m.AttachedAt.UTC()
	b, err := json.Marshal(m)
	if err != nil || len(b) > MaxMetadataBytes {
		return nil, ErrInvalid
	}
	return b, nil
}

func ReadMetadata(data []byte) (Metadata, error) {
	var m Metadata
	if len(data) > MaxMetadataBytes || json.Unmarshal(data, &m) != nil {
		return Metadata{}, ErrInvalid
	}
	canonical, err := m.Bytes()
	if err != nil || !bytes.Equal(canonical, data) {
		return Metadata{}, ErrInvalid
	}
	return m, nil
}

func (m Metadata) Digest() ([32]byte, error) {
	b, err := m.Bytes()
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(b), nil
}

// CheckHeader establishes content binding only, never filesystem ownership or
// authorization. Each later segment may start a new monotonic clock epoch and
// must separately link verified preceding coverage/digests in durable storage.
func (m Metadata) CheckHeader(h Header) error {
	digest, err := m.Digest()
	if err != nil || h.Validate() != nil {
		return ErrInvalid
	}
	id, _ := hex.DecodeString(m.SessionID)
	if !bytes.Equal(id, h.SessionID[:]) || digest != h.MetadataSHA256 || h.Segment == 1 && h.StartedUnixNano != m.AttachedAt.UnixNano() {
		return ErrInvalid
	}
	return nil
}

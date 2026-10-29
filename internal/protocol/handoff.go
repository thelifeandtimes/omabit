package protocol

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"time"
)

// MaxPierBytes is the network host-handoff content limit: 100 GB in decimal
// bytes. Local imports have a separate limit; framing has its own allowance.
const MaxPierBytes int64 = 100_000_000_000
const MaxHandoffBytes int64 = MaxPierBytes + (32 << 20)
const HandoffHashHeader = "Urbit-Runner-Handoff-SHA256"

var handoffDataRoute = regexp.MustCompile(`^/v1/handoffs/[0-9a-f]{32}/data$`)

func HandoffDataPath(p string) bool { return handoffDataRoute.MatchString(p) }

type Handoff struct {
	ID                  string           `json:"id"`
	Direction           string           `json:"direction"`
	State               string           `json:"state"`
	SourceRunner        string           `json:"source_runner"`
	SourceInstance      string           `json:"source_instance"`
	DestinationRunner   string           `json:"destination_runner"`
	DestinationInstance string           `json:"destination_instance"`
	Ship                string           `json:"ship"`
	Mode                string           `json:"mode"`
	AmesPublished       bool             `json:"ames_published"`
	RollNeedsRuntime    bool             `json:"roll_requires_run,omitempty"`
	RuntimeSettings     *RuntimeSettings `json:"runtime_settings,omitempty"`
	SHA256              string           `json:"sha256,omitempty"`
	TreeDigest          string           `json:"tree_digest,omitempty"`
	Bytes               int64            `json:"bytes,omitempty"`
	CreatedAt           time.Time        `json:"created_at"`
	Message             string           `json:"message,omitempty"`
	Error               string           `json:"error,omitempty"`
	VerifyKey           string           `json:"verify_key"`
	Commitment          string           `json:"commitment,omitempty"`
}

// Commitment binds one stopped source to exactly one destination and byte stream.
// The ephemeral signing key stays in the source's private journal. The destination
// learns its verification key through the operator's pinned source connection.
func (h Handoff) CommitmentBytes() []byte {
	fields := []any{"omabit-stopped-handoff-v1", h.ID, h.SourceRunner, h.SourceInstance, h.DestinationRunner, h.DestinationInstance, h.Ship, h.Mode, h.AmesPublished, h.RollNeedsRuntime, h.SHA256, h.Bytes, h.TreeDigest, h.VerifyKey}
	if h.RuntimeSettings != nil {
		fields[0] = "omabit-stopped-handoff-v2"
		fields = append(fields, *h.RuntimeSettings)
	}
	b, _ := json.Marshal(fields)
	return b
}
func (h Handoff) ValidCommitment(proof string) bool {
	key, e := hex.DecodeString(h.VerifyKey)
	signature, e2 := hex.DecodeString(proof)
	return e == nil && e2 == nil && len(key) == ed25519.PublicKeySize && ed25519.Verify(key, h.CommitmentBytes(), signature)
}
func (h Handoff) SameTransfer(other Handoff) bool {
	return string(h.CommitmentBytes()) == string(other.CommitmentBytes())
}

func (h *Handoff) Fenced() bool { return h != nil && (h.Direction != "in" || h.State != "complete") }

type PrepareHandoffRequest struct {
	ExpectedRevision    uint64 `json:"expected_revision"`
	RequestID           string `json:"request_id"`
	DestinationRunner   string `json:"destination_runner"`
	DestinationInstance string `json:"destination_instance"`
	// The authenticated collector checks the pinned destination's capability
	// before allowing the source to fence a nondefault runtime configuration.
	DestinationRuntimeSettings bool `json:"destination_runtime_settings,omitempty"`
}

type ReceiveHandoffRequest struct {
	Offer     Handoff `json:"offer"`
	HTTPPort  uint16  `json:"http_port"`
	AmesPort  uint16  `json:"ames_port"`
	RequestID string  `json:"request_id"`
}

type FinishHandoffRequest struct {
	Commitment       string `json:"commitment,omitempty"`
	SHA256           string `json:"sha256"`
	RequestID        string `json:"request_id"`
	ExpectedRevision uint64 `json:"expected_revision"`
}

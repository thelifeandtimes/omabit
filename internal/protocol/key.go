package protocol

import "regexp"

const MaxBootKeyBytes = 2048

// Omabit's provisioning limit, in decimal bytes; not a Vere/Arvo format limit.
const MaxBootPillBytes int64 = 4_000_000_000
const PillUploadPath = "/v1/boot-pill"
const PillHashHeader = "Urbit-Runner-Pill-SHA256"

type BootPill struct {
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

var pillHashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

func ValidBootPill(p *BootPill) bool {
	return p != nil && pillHashPattern.MatchString(p.SHA256) && p.Bytes > 0 && p.Bytes <= MaxBootPillBytes
}

var bootKeyPattern = regexp.MustCompile(`^0w[0-9a-zA-Z.~-]+$`)

func ValidBootKey(key string) bool {
	return len(key) >= 18 && len(key) <= MaxBootKeyBytes && bootKeyPattern.MatchString(key)
}

// Key is transient request material, never part of the operation journal.
type CreateShipRequest struct {
	ID             string        `json:"id"`
	Label          string        `json:"label"`
	Ship           string        `json:"ship"`
	PierPath       string        `json:"pier_path"`
	Config         Configuration `json:"config"`
	RequestID      string        `json:"request_id"`
	Key            string        `json:"key,omitempty"`
	ConfirmNetwork bool          `json:"confirm_network"`
	Pill           *BootPill     `json:"pill,omitempty"`
}

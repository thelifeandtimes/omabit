package privatechain

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

const RegistrationKind = "register-private-chain-project"

// Primitive public inputs only. Neither keys nor runtime descriptors are accepted.
type RegisterRequest struct {
	RequestID                   string          `json:"request_id"`
	Label                       string          `json:"label"`
	Foundation                  Foundation      `json:"foundation"`
	Members                     []MemberRequest `json:"members"`
	ConfirmSharedAdministration bool            `json:"confirm_shared_administration"`
	ConfirmReservation          bool            `json:"confirm_reservation"`
	ReviewDigest                string          `json:"review_digest"`
}

func (r RegisterRequest) ReviewedPlan() (Plan, error) {
	p, err := Build(Request{Label: r.Label, Foundation: r.Foundation, Members: r.Members, ConfirmSharedAdministration: r.ConfirmSharedAdministration}, Reservations{})
	if err != nil {
		return Plan{}, err
	}
	digest, err := p.Digest()
	if err != nil || !protocol.ValidID(r.RequestID) || !r.ConfirmReservation || digest != r.ReviewDigest {
		return Plan{}, errors.New("private-chain registration requires a valid request ID, matching review digest and reservation acknowledgement")
	}
	for _, m := range p.Members {
		if !protocol.ValidPierName(m.InstanceID) {
			return Plan{}, errors.New("private-chain member names must be valid exclusive storage names")
		}
	}
	return p, nil
}
func DecodeRegistration(data []byte) (RegisterRequest, error) {
	var r RegisterRequest
	bad := errors.New("expected one bounded private-chain registration with public review metadata only")
	if len(data) > MaxFoundationBytes || !uniqueJSON(data) {
		return r, bad
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if d.Decode(&r) != nil || d.Decode(new(any)) != io.EOF {
		return RegisterRequest{}, bad
	}
	if _, err := r.ReviewedPlan(); err != nil {
		return RegisterRequest{}, bad
	}
	return r, nil
}

type RegistrationResult struct {
	SchemaVersion     int                `json:"schema_version"`
	RunnerID          string             `json:"runner_id"`
	Mode              string             `json:"mode"`
	Binding           Binding            `json:"binding"`
	ReviewDigest      string             `json:"review_digest"`
	ReservationOnly   bool               `json:"reservation_only"`
	RuntimeAuthorized bool               `json:"runtime_authorized"`
	Operation         protocol.Operation `json:"operation"`
}
type Reservation struct {
	Binding                 Binding   `json:"binding"`
	Revision                uint64    `json:"revision"`
	Phase                   string    `json:"phase"`
	Plan                    Plan      `json:"plan"`
	ReviewDigest            string    `json:"review_digest"`
	CreatedAt               time.Time `json:"created_at"`
	RegistrationOperationID string    `json:"registration_operation_id"`
}
type Inventory struct {
	SchemaVersion      int           `json:"schema_version"`
	RunnerID           string        `json:"runner_id"`
	Mode               string        `json:"mode"`
	ReadAt             time.Time     `json:"read_at"`
	Basis              string        `json:"basis"`
	RuntimeObservation string        `json:"runtime_observation"`
	RuntimeAuthorized  bool          `json:"runtime_authorized"`
	Projects           []Reservation `json:"projects"`
}

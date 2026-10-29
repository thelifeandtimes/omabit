package protocol

const ProjectRegistrationFeature = "project-registration-v1"

// Primitive public inputs; the runner reconstructs the canonical project plan.
type ProjectMemberInput struct {
	InstanceID string `json:"instance_id"`
	Ship       string `json:"ship"`
	HTTPPort   uint16 `json:"http_port"`
}
type ProjectRegisterRequest struct {
	RequestID                   string               `json:"request_id"`
	Label                       string               `json:"label"`
	Members                     []ProjectMemberInput `json:"members"`
	ConfirmSharedAdministration bool                 `json:"confirm_shared_administration"`
	ConfirmReservation          bool                 `json:"confirm_reservation"`
	ReviewDigest                string               `json:"review_digest"`
}
type ProjectRegistrationResult struct {
	SchemaVersion   int       `json:"schema_version"`
	RunnerID        string    `json:"runner_id"`
	Mode            string    `json:"mode"`
	ReviewDigest    string    `json:"review_digest"`
	ReservationOnly bool      `json:"reservation_only"`
	Operation       Operation `json:"operation"`
}

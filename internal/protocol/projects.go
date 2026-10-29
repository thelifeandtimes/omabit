package protocol

import "time"

const ProjectInventoryFeature = "project-reservations-v1"
const ProjectInventoryPath = "/v1/projects"

// ProjectInventory describes consistent saved claims, never current runtime state.
// No container IDs, session credentials or runtime mutation authority are returned.
type ProjectInventory struct {
	SchemaVersion      int                  `json:"schema_version"`
	RunnerID           string               `json:"runner_id"`
	Mode               string               `json:"mode"`
	ReadAt             time.Time            `json:"read_at"`
	Basis              string               `json:"basis"`
	RuntimeObservation string               `json:"runtime_observation"`
	Projects           []ProjectReservation `json:"projects"`
}

type ProjectReservation struct {
	AdditionalMembers []ProjectMemberReservation `json:"additional_members,omitempty"`
	ID                string                     `json:"id"`
	Revision          uint64                     `json:"revision"`
	Label             string                     `json:"label"`
	CreatedAt         time.Time                  `json:"created_at"`
	ReviewDigest      string                     `json:"review_digest"`
	Members           []ProjectMemberReservation `json:"members"`
}

type ProjectMemberReservation struct {
	InstanceID       string `json:"instance_id"`
	Ship             string `json:"ship"`
	HTTPPort         uint16 `json:"http_port"`
	InternalHTTPPort uint16 `json:"internal_http_port"`
}

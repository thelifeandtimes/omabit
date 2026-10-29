package protocol

const ProjectDependenciesFeature = "project-dependencies-v1"
const ProjectDependenciesPath = "/v1/project-dependencies"

type ProjectDependenciesRequest struct {
	ConfirmDownload bool `json:"confirm_download"`
}
type ProjectDependenciesResult struct {
	SchemaVersion int      `json:"schema_version"`
	RunnerID      string   `json:"runner_id"`
	Mode          string   `json:"mode"`
	Image         string   `json:"image"`
	Pill          BootPill `json:"pill"`
}

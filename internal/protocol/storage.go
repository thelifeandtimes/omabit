package protocol

type StorageRootRequest struct {
	ID        string `json:"id"`
	Path      string `json:"path"`
	RequestID string `json:"request_id"`
}
type ImportPierRequest struct {
	EnrollPierRequest
	Source string `json:"source"`
}

type ImportPierPreviewRequest struct {
	Source string `json:"source"`
}

type ExternalImportStatus struct {
	CopiedFiles int    `json:"copied_files"`
	CopiedBytes int64  `json:"copied_bytes"`
	Source      string `json:"source"`
	HoldingPath string `json:"holding_path,omitempty"`
	Stage       string `json:"stage"`
}

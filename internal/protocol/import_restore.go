package protocol

const ImportRestoreFeature = "restore-import"

type ImportRestorePreviewRequest struct {
	ExpectedRevision uint64 `json:"expected_revision"`
}

// Previews expose a digest and human-readable moves, never the private object
// inventories. The successful operation retains these paths after retirement.
type ImportRestoreResult struct {
	Source       string `json:"source"`
	RestoreTo    string `json:"restore_to"`
	Destination  string `json:"destination"`
	RetainAt     string `json:"retain_at"`
	ReviewDigest string `json:"review_digest"`
}
type ImportRestorePreview struct {
	ImportRestoreResult
	RunnerID         string `json:"runner_id"`
	InstanceID       string `json:"instance_id"`
	ConfigRevision   uint64 `json:"config_revision"`
	SourceFiles      int    `json:"source_files"`
	SourceBytes      int64  `json:"source_bytes"`
	DestinationFiles int    `json:"destination_files"`
	DestinationBytes int64  `json:"destination_bytes"`
	Stage            string `json:"stage"`
	Continuation     bool   `json:"continuation"`
}

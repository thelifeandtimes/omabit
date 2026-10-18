package protocol

const AmesPublicationFeature = "publish-ames"

// PublishAmesRequest explicitly upgrades a stopped, initialized legacy live
// enrollment to host UDP publication on its next Start. No port is inferred.
type PublishAmesRequest struct {
	RequestID        string `json:"request_id"`
	ExpectedRevision uint64 `json:"expected_revision"`
	AmesPort         uint16 `json:"ames_port"`
	ConfirmNetwork   bool   `json:"confirm_network"`
}

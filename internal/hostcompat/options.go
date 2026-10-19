package hostcompat

// VerificationOptions is accepted only by explicit local-root verification.
// FetchMissing permits download-only retrieval of the exact installed builds;
// it never permits installation, upgrades, key import or network on Inspect.
type VerificationOptions struct {
	FetchMissing bool `json:"fetch_missing"`
}

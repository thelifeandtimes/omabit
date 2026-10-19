package protocol

import (
	"encoding/json"
	"slices"
	"time"
	"unicode"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
)

const HostCompatibilityFeature = hostcompat.Feature
const HostCompatibilityPath = "/v1/host-compatibility"
const HostCompatibilitySchema = 1
const HostCompatibilityMaxAge = 30 * time.Second

// HostCompatibilityReport is public, read-only evidence, never authorization to
// run a repair or install command. ExpiresAt bounds both native and live evidence;
// Report.ObservedAt is the live sample, MetadataObservedAt the native sample.
type HostCompatibilityReport struct {
	SchemaVersion      int               `json:"schema_version"`
	ProtocolVersion    int               `json:"protocol_version"`
	RunnerID           string            `json:"runner_id"`
	Mode               string            `json:"mode"`
	Features           []string          `json:"features"`
	Report             hostcompat.Report `json:"report"`
	ExpiresAt          time.Time         `json:"expires_at"`
	MetadataObservedAt *time.Time        `json:"metadata_observed_at,omitempty"`
}

// PublicHostCompatibility removes private owner paths. Package/distro metadata
// and the policy's fixed diagnostics are public to authenticated observers.
// Inspection must not add raw subprocess output, repository URLs or credentials.
func PublicHostCompatibility(r hostcompat.Report) hostcompat.Report {
	if r.Engine != nil {
		engine := *r.Engine
		engine.DataRoot = ""
		r.Engine = &engine
	}
	return r
}

// ValidHostCompatibility bounds reports before they enter fleet observations.
// Unknown schemas/capabilities are unavailable evidence, not a failed inventory.
func ValidHostCompatibility(r HostCompatibilityReport) bool {
	if r.SchemaVersion != HostCompatibilitySchema || r.ProtocolVersion != Version || !ValidID(r.RunnerID) || !slices.Contains(r.Features, HostCompatibilityFeature) || len(r.Features) > 64 || r.Report.SchemaVersion != 1 || r.Report.PolicyID != hostcompat.PolicyID || r.Report.ObservedAt.IsZero() || !r.ExpiresAt.After(r.Report.ObservedAt) || r.ExpiresAt.Sub(r.Report.ObservedAt) > HostCompatibilityMaxAge {
		return false
	}
	if r.Report.Engine != nil && r.Report.Engine.DataRoot != "" {
		return false
	}
	if at := r.MetadataObservedAt; at != nil && (at.IsZero() || at.After(r.Report.ObservedAt) || r.ExpiresAt.Sub(*at) > HostCompatibilityMaxAge) {
		return false
	}
	if len(r.Report.Packages) > 64 || len(r.Report.Checks) > 128 || len(r.Report.Guidance) > 32 || len(r.Report.Distribution.IDLike) > 16 {
		return false
	}
	b, err := json.Marshal(r)
	if err != nil || len(b) > 64<<10 {
		return false
	}
	// Check every string, including strings introduced by future additive fields.
	var value any
	if json.Unmarshal(b, &value) != nil {
		return false
	}
	return boundedHostStrings(value)
}

func boundedHostStrings(value any) bool {
	switch v := value.(type) {
	case string:
		if len(v) > 1024 {
			return false
		}
		for _, c := range v {
			if unicode.IsControl(c) || unicode.Is(unicode.Cf, c) {
				return false
			}
		}
	case []any:
		if len(v) > 128 {
			return false
		}
		for _, entry := range v {
			if !boundedHostStrings(entry) {
				return false
			}
		}
	case map[string]any:
		for _, entry := range v {
			if !boundedHostStrings(entry) {
				return false
			}
		}
	}
	return true
}

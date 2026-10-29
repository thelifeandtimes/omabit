package hostcompat

import "errors"

// ValidEngineIdentity validates a bounded opaque Docker daemon ID. Current UUID
// and historical colon-separated forms are accepted without normalizing either.
// A data-root path or socket path is not a substitute for daemon identity.
func ValidEngineIdentity(id string) bool {
	if len(id) == 0 || len(id) > 128 {
		return false
	}
	for _, c := range id {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == ':' || c == '_') {
			return false
		}
	}
	return true
}

// CheckEngineIdentity is an additional requirement for new project resources.
// Existing non-project inventory keeps its earlier compatibility contract, so
// historical reports without IDs remain readable. This adds no adoption authority.
func CheckEngineIdentity(engine Engine, expected string) error {
	if !ValidEngineIdentity(expected) || !ValidEngineIdentity(engine.ID) || engine.ID != expected {
		return errors.New("private engine identity is unavailable or changed")
	}
	return nil
}

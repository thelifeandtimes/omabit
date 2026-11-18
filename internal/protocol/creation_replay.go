package protocol

import (
	"errors"
	"net/http"
)

// CreationReplayFeature promises that creation requests carrying the replay-only
// header can only return an existing actor-scoped, fingerprint-matching receipt.
// It must never be advertised by a server that can admit fresh work on that path.
const CreationReplayFeature = "creation-replay-only"
const CreationReplayHeader = "X-Omabit-Replay-Only"

// ReplayOnlyEndpoint lists only routes whose handlers implement guarded,
// actor-scoped receipt lookup. Accepting the header on an arbitrary mutation
// would let an older/unguarded handler silently dispatch fresh work.
func ReplayOnlyEndpoint(method, path string) bool {
	if method != http.MethodPost {
		return false
	}
	switch path {
	case "/v1/instances", "/v1/comets", "/v1/ships", "/v1/projects", PrivateChainProjectsPath:
		return true
	}
	if _, ok := ProjectActionID(path); ok {
		return true
	}
	if _, ok := ProjectStopReconcileID(path); ok {
		return true
	}
	_, _, _, ok := ProjectRecoveryActionIDs(path)
	return ok
}

// CreationReplayOnly accepts exactly one literal true or an absent header. A
// malformed/duplicated guard must never be silently treated as normal creation.
func CreationReplayOnly(header http.Header) (bool, error) {
	values, present := header[http.CanonicalHeaderKey(CreationReplayHeader)]
	if !present {
		return false, nil
	}
	if len(values) != 1 || values[0] != "true" {
		return false, errors.New("X-Omabit-Replay-Only must be a single true value")
	}
	return true, nil
}

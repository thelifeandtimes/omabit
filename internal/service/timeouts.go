package service

import (
	"slices"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// These are trusted catalog/adapter bounds, never tool arguments. Leave bounded
// room for the selected-connection identity/capability handshakes around a long
// RPC. The authoritative runner/transport retains its own shorter native bound.
func catalogTimeout(id string, capabilities []string) time.Duration {
	if id == "runtime.recording-timing.v1" || id == "runtime.recording-redacted-export-manifest.v1" || id == "runtime.recording-redacted-export-part.v1" {
		return 6 * time.Minute
	}
	if slices.Contains(capabilities, protocol.ProjectAllocationFeature) {
		return protocol.ProjectAllocationTimeout + 15*time.Second
	}
	if slices.Contains(capabilities, protocol.DeskStageFeature) {
		return protocol.DeskStageTimeout + 15*time.Second
	}
	if slices.Contains(capabilities, protocol.GrubberyAppSourceFeature) || slices.Contains(capabilities, protocol.GrubberyRuntimeRecoveryFeature) {
		return protocol.GrubberyAppTimeout + 15*time.Second
	}
	for _, feature := range []string{protocol.DeskCommitFeature, protocol.DeskReleaseFeature, protocol.DeskFrontendFeature, protocol.DeskReceiverFeature, protocol.DeskAdoptionFeature} {
		if slices.Contains(capabilities, feature) {
			return protocol.DeskDeploymentTimeout + 15*time.Second
		}
	}
	for _, feature := range []string{protocol.DeskCreateFeature, protocol.DeskDiscoveryFeature, protocol.DeskSourceFeature, protocol.DeskSourceExportFeature, protocol.DeskApplicationFeature} {
		if slices.Contains(capabilities, feature) {
			return protocol.DeskReadTimeout + 15*time.Second
		}
	}
	if id == "project.create-advance.v1" {
		return 105 * time.Second
	}
	return 90 * time.Second
}

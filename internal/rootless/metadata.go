// Package rootless prepares reviewed rootless Docker support artifacts. Preparing
// artifacts does not install packages, create accounts, or start an engine.
package rootless

const (
	// EngineVersion is the historical qualified baseline, retained for old wire
	// consumers. Engine admission is governed by hostcompat, not this constant.
	EngineVersion = "29.7.2"
	// LauncherVersion identifies the pinned artifact, independent of the engine.
	LauncherVersion = "29.7.2"
	SourceCommit    = "6a43e3d5afddf4111da0f864bbc7cae5d7e95001"
	LauncherSHA256  = "904c9b9e35f6927c0a5e65afb4d35b6bc9eb1278c878044501281fc728c9be46"
	AccountName     = "urbit-runner"
	HomeDir         = "/var/lib/urbit-runner"
	InstallDir      = "/usr/local/lib/urbit-runner/rootless/" + LauncherVersion
	UnitName        = "urbit-docker.service"
	SocketName      = "urbit-docker.sock"
)

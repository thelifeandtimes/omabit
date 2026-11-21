// Package buildmeta exposes only compiled provenance, never paths, environment,
// credentials or guesses about the installed desktop's release generation.
package buildmeta

import (
	"runtime/debug"
	"strconv"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func Current(executableSHA256 string) *protocol.ExecutableBuild {
	out := &protocol.ExecutableBuild{SHA256: executableSHA256}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return out
	}
	out.GoVersion = info.GoVersion
	out.ModuleVersion = info.Main.Version
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			out.SourceRevision = setting.Value
		case "vcs.modified":
			if value, err := strconv.ParseBool(setting.Value); err == nil {
				out.Modified = &value
			}
		}
	}
	return out
}

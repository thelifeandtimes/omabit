package managed

import "git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"

func validStoredResources(v storedInstance) bool {
	if !protocol.ValidContainerResources(v.Public.Config.MemoryGiB, v.Public.Config.CPUCores) || !protocol.ValidContainerResources(v.Runtime.MemoryGiB, v.Runtime.CPUCores) {
		return false
	}
	for _, c := range v.Runtime.Runtime.Containers {
		if !protocol.ValidContainerResources(c.MemoryGiB, c.CPUCores) {
			return false
		}
	}
	return true
}
func validateContainerResources(c protocol.Configuration) error {
	if !protocol.ValidContainerResources(c.MemoryGiB, c.CPUCores) {
		return bad("container RAM must be 1..16 GiB and CPU limit 1..16 cores; omitted values preserve defaults/settings")
	}
	return nil
}

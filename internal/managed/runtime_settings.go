package managed

import (
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"path/filepath"
)

// This is path discovery, not a claim that a socket is currently listening, nor
// permission to execute commands. Use the existing authenticated snapshot only.
func attachNativeControlPaths(snapshot *protocol.Snapshot) {
	if snapshot.Piers == nil || snapshot.Piers.Error != "" {
		return
	}
	for i := range snapshot.Instances {
		v := &snapshot.Instances[i]
		d := v.RuntimeStatus
		if d == nil || d.LaunchNativeControl == nil || !*d.LaunchNativeControl || !pierstore.ValidPath(v.PierPath) {
			continue
		}
		id, relative := pierstore.Split(v.PierPath)
		if id == "" {
			id = "default"
		}
		for _, root := range snapshot.Piers.Roots {
			if root.ID == id && filepath.IsAbs(root.Path) && filepath.Clean(root.Path) == root.Path {
				d.NativeControlSocket = filepath.Join(root.Path, relative, ".urb/conn.sock")
				break
			}
		}
	}
}

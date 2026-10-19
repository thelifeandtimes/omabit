package rootless

// SafePath contains only distribution-managed locations; never inherit a
// desktop user's PATH in an authorized installer or rootless service.
const SafePath = "/usr/bin:/usr/sbin:/bin:/sbin"

// ExecutablePaths enumerates fixed native aliases, not arbitrary search paths.
// Callers must validate root ownership, parent directories and permissions.
func ExecutablePaths(name string) []string {
	switch name {
	case "dockerd", "useradd", "usermod", "runuser", "nologin", "sshd", "getsubids", "iptables", "ip6tables":
		return []string{"/usr/bin/" + name, "/usr/sbin/" + name}
	case "docker", "rootlesskit", "slirp4netns", "newuidmap", "newgidmap", "systemctl", "loginctl", "id", "getent", "env", "test", "ssh-keygen", "stat", "timeout", "pacman-conf", "apt-config":
		return []string{"/usr/bin/" + name}
	}
	return nil
}

// ExecutableLink permits only known native symlink edges. A trusted root-owned
// symlink alone is not permission to execute an arbitrary target.
func ExecutableLink(name, from, to string) bool {
	for _, path := range ExecutablePaths(name) {
		if to == path && from != to {
			return true
		}
	}
	if name == "iptables" || name == "ip6tables" {
		if to == "/etc/alternatives/"+name {
			return from == "/usr/bin/"+name || from == "/usr/sbin/"+name
		}
		for _, dir := range []string{"/usr/bin/", "/usr/sbin/"} {
			for _, target := range []string{name + "-nft", name + "-legacy", "xtables-nft-multi", "xtables-legacy-multi"} {
				if to == dir+target {
					return true
				}
			}
		}
	}
	return false
}

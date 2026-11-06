package hostsetup

import (
	"slices"
	"sort"
	"strings"
)

// NativePackageForCheck is a compiled-in installation plan, never a command or
// package name supplied by the desktop. Other distributions retain their native
// manual prerequisite flow until their installation transactions are supported.
func NativePackageForCheck(r Report, c Check) string {
	if r.Compatibility == nil || r.Compatibility.Distribution.Provider != "pacman" ||
		!slices.Contains([]string{"arch", "omarchy"}, r.Compatibility.Distribution.ID) || c.Status != "missing" {
		return ""
	}
	return map[string]string{
		"binary-docker": "docker", "binary-dockerd": "docker",
		"binary-containerd": "containerd", "binary-runc": "runc",
		"binary-rootlesskit": "rootlesskit", "binary-slirp4netns": "slirp4netns",
		"binary-newuidmap": "shadow", "binary-newgidmap": "shadow", "binary-getsubids": "shadow",
		"binary-iptables": "iptables-nft", "binary-ssh": "openssh", "binary-sshd": "openssh", "binary-ssh-keygen": "openssh",
	}[c.ID]
}

func NativePackages(r Report) []string {
	var packages []string
	for _, c := range r.Checks {
		if name := NativePackageForCheck(r, c); name != "" && !slices.Contains(packages, name) {
			packages = append(packages, name)
		}
	}
	sort.Strings(packages)
	return packages
}

// An unresolved version is installable only when the engine itself is absent.
// An installed unsupported/unowned engine is never fixed by an implicit update.
func InstallableCheck(r Report, c Check) bool {
	if NativePackageForCheck(r, c) != "" {
		return true
	}
	if c.ID == "docker-version" && c.Status == "unknown" && r.Compatibility != nil && r.Compatibility.InstalledEngineVersion == "" {
		for _, binary := range r.Checks {
			if binary.ID == "binary-dockerd" && NativePackageForCheck(r, binary) == "docker" {
				return true
			}
		}
	}
	return false
}

func RequiredSetupCheck(c Check) bool {
	return c.ID == "owner-account" || c.ID == "platform" || c.ID == "systemd" || c.ID == "cgroup-v2" ||
		c.ID == "userns-limit" || c.ID == "docker-version" || strings.HasPrefix(c.ID, "binary-") ||
		(c.ID == "subuid" || c.ID == "subgid") && c.Status != "missing" ||
		(c.ID == "userns-clone" || c.ID == "account") && c.Status == "blocked"
}

package rootless

import (
	"embed"
	"fmt"
)

const PackageHookDir = "/usr/local/lib/urbit-runner/package-hooks"
const PackageRefreshPath = PackageHookDir + "/refresh-host-packages"
const PacmanPackageHookPath = "/etc/pacman.d/hooks/00-omabit-host-packages.hook"
const APTPackageHookPath = "/etc/apt/apt.conf.d/00-omabit-host-packages"

//go:embed assets/package-hooks/*
var packageHookAssets embed.FS

// PackageHookFiles returns only reviewed fixed paths and bytes. A privileged
// installer must separately validate native configuration and the current runner.
func PackageHookFiles(provider string) (map[string][]byte, error) {
	name, path := "", ""
	switch provider {
	case "pacman":
		name, path = "00-omabit-host-packages.hook", PacmanPackageHookPath
	case "deb":
		name, path = "00-omabit-host-packages", APTPackageHookPath
	default:
		return nil, fmt.Errorf("native package transaction hooks are unavailable for this provider")
	}
	files := make(map[string][]byte, 2)
	for target, asset := range map[string]string{path: name, PackageRefreshPath: "refresh-host-packages"} {
		data, err := packageHookAssets.ReadFile("assets/package-hooks/" + asset)
		if err != nil {
			return nil, err
		}
		files[target] = data
	}
	return files, nil
}

// PreviousPackageHookFiles is the exact-content migration allowlist. This first
// hook format has no predecessors; future reviewed versions append their frozen
// templates here, rather than inferring ownership from names or comments.
func PreviousPackageHookFiles(provider string) (map[string][][]byte, error) {
	if _, err := PackageHookFiles(provider); err != nil {
		return nil, err
	}
	return map[string][][]byte{}, nil
}

package hostinstall

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"slices"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

func (i *installer) nativeSSHUnit(ctx context.Context, data []byte) ([]byte, error) {
	path, err := i.resolveHelper(ctx, "/usr/bin/sshd")
	if err != nil {
		return nil, err
	}
	return bytes.ReplaceAll(data, []byte("/usr/bin/sshd"), []byte(path)), nil
}

// VerifyOptions controls explicit local verification, never package installation,
// repository configuration or key import. The zero value is offline.
type VerifyOptions struct{ FetchMissing bool }

// Normal setup/update has already received OS authorization. It may retrieve
// missing native evidence into core's private verified cache before executing
// Docker or changing services. Valid current receipts avoid any download.
func (i *installer) requireCompatibility(ctx context.Context) error {
	return i.requireCompatibilityWithOptions(ctx, VerifyOptions{FetchMissing: true})
}

// Use fresh local evidence, never a report supplied by the desktop or an earlier
// preflight. Cryptographic verification and cache policy remain core's authority.
func (i *installer) requireCompatibilityWithOptions(ctx context.Context, options VerifyOptions) error {
	if ctx == nil || i.euid != 0 {
		return errors.New("native package verification requires OS authorization and context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if i.inspectCompatibility == nil {
		return errors.New("native package verifier unavailable")
	}
	r := i.inspectCompatibility()
	if r.Provenance != "verified" {
		var err error
		if options.FetchMissing {
			if i.verifyCompatibilityWithOptions == nil {
				return errors.New("native package download verifier unavailable")
			}
			r, err = i.verifyCompatibilityWithOptions(ctx, hostcompat.VerificationOptions{FetchMissing: true})
		} else {
			if i.verifyCompatibility == nil {
				return errors.New("native package verifier unavailable")
			}
			r, err = i.verifyCompatibility(ctx)
		}
		if err != nil {
			return err
		}
	}
	if hostcompat.EngineVersionStatus(r.InstalledEngineVersion) != "compatible" {
		return errors.New("installed native engine version is unresolved or unsupported; review host compatibility guidance")
	}
	if r.Provenance != "verified" {
		return errors.New("native package provenance is not verified; refresh native authenticated package evidence before installation (metadata/MD5 alone are insufficient)")
	}
	return nil
}

// The CLI is native-package verified and runs as the selected owner, with a
// fixed API version and an already identity-checked private socket. Check both
// info and version; docker info alone does not establish API bounds.
func (i *installer) verifyEngineAPI(ctx context.Context, socket string, owner *ownerhost.Profile, infoData []byte) error {
	args := []string{"--host", "unix://" + socket, "version", "--format", "{{json .Server}}"}
	var data []byte
	var err error
	home := rootless.HomeDir
	if owner == nil {
		data, err = i.asAccount(ctx, "/usr/bin/docker", args...)
	} else {
		home = owner.Home
		data, err = i.ownerCommand(ctx, *owner, "/usr/bin/docker", args...)
	}
	if err != nil {
		return err
	}
	var version struct{ Version, APIVersion, MinAPIVersion string }
	var info struct {
		ServerVersion, CgroupDriver, CgroupVersion, DockerRootDir string
		SecurityOptions                                           []string
	}
	if len(data) > maxOutput || len(infoData) > maxOutput || json.Unmarshal(data, &version) != nil || json.Unmarshal(infoData, &info) != nil || version.Version != info.ServerVersion {
		return errors.New("engine API/version evidence is malformed or changed during verification")
	}
	return hostcompat.CheckEngine(hostcompat.Engine{Version: version.Version, APIVersion: version.APIVersion, MinAPIVersion: version.MinAPIVersion, Rootless: slices.Contains(info.SecurityOptions, "name=rootless"), CgroupDriver: info.CgroupDriver, CgroupVersion: info.CgroupVersion, DataRoot: info.DockerRootDir}, filepath.Join(home, ".local/share/urbit-runner/docker"), false)
}

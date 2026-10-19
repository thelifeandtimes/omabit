package hostinstall

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"

	serviceunits "git.thechristophgroup.org/thelifeandtimes/omabit/deploy/systemd"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

type GuardRepairResult struct {
	Updated  bool `json:"updated"`
	Reloaded bool `json:"reloaded"`
	Partial  bool `json:"partial"`
}

// RepairEngineGuard is a local-root repair seam, including when the previous
// exact-version guard prevents startup. It neither activates management nor
// starts/restarts the engine, and never changes piers, data roots or update fences.
// The caller must obtain explicit OS authorization, not a remote management grant.
// This existing entry point remains offline; missing evidence requires an
// explicitly opted-in RepairEngineGuardWithOptions call.
func RepairEngineGuard(ctx context.Context) (GuardRepairResult, error) {
	return RepairEngineGuardWithOptions(ctx, VerifyOptions{})
}

// RepairEngineGuardWithOptions optionally authorizes native download-only
// evidence retrieval. It retains the same guard-only, no-engine-restart contract.
func RepairEngineGuardWithOptions(ctx context.Context, options VerifyOptions) (GuardRepairResult, error) {
	i := newInstaller()
	return i.repairEngineGuardWithOptions(ctx, options)
}

func (i *installer) repairEngineGuard(ctx context.Context) (GuardRepairResult, error) {
	return i.repairEngineGuardWithOptions(ctx, VerifyOptions{})
}

func (i *installer) repairEngineGuardWithOptions(ctx context.Context, options VerifyOptions) (result GuardRepairResult, err error) {
	defer func() { result.Partial = err != nil && i.mutated }()
	if i.euid != 0 || i.goos != "linux" || i.goarch != "amd64" {
		return result, errors.New("guard repair requires OS authorization on Linux x86-64")
	}
	if err = i.requireCompatibilityWithOptions(ctx, options); err != nil {
		return result, err
	}
	var owner *ownerhost.Profile
	name, uid := rootless.AccountName, uint32(0)
	data, e := i.readInstalled(ctx, ownerhost.Path, 8192)
	if e == nil {
		var p ownerhost.Profile
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		var extra any
		if decoder.Decode(&p) != nil || decoder.Decode(&extra) != io.EOF || p.Validate() != nil {
			return result, errors.New("installed owner profile needs review")
		}
		u, e := i.lookupUser(p.Name)
		if e != nil || u.Username != p.Name || u.Uid != fmt.Sprint(p.UID) || u.Gid != fmt.Sprint(p.GID) || u.HomeDir != p.Home {
			return result, errors.New("installed owner identity changed")
		}
		owner, name, uid = &p, p.Name, p.UID
	} else if !errors.Is(e, os.ErrNotExist) {
		return result, e
	} else {
		u, e := i.lookupUser(name)
		if e != nil {
			return result, errors.New("no installed account to repair")
		}
		uid, e = accountID(u.Uid)
		gid, ge := accountID(u.Gid)
		if e != nil || ge != nil || u.HomeDir != rootless.HomeDir || u.Username != name {
			return result, errors.New("installed dedicated identity changed")
		}
		if err = i.checkAccountMode(ctx, u, uid, gid, false); err != nil {
			return result, err
		}
	}
	lockPath := "/etc/urbit-runner/.setup.lock"
	if _, e = i.readInstalled(ctx, lockPath, 0); e != nil && !errors.Is(e, os.ErrNotExist) {
		return result, e
	}
	if err = i.parents(ctx, lockPath, false); err != nil {
		return result, err
	}
	lock, err := os.OpenFile(i.path(lockPath), os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0600)
	if err != nil {
		return result, err
	}
	defer lock.Close()
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return result, errors.New("another host setup is running")
	}
	launcher, err := i.readInstalled(ctx, rootless.InstallDir+"/dockerd-rootless.sh", maxOutput)
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(launcher)) != rootless.LauncherSHA256 {
		return result, errors.New("installed pinned launcher needs review")
	}
	files, err := rootless.ServiceFiles()
	if err != nil {
		return result, err
	}
	result.Updated, err = i.updateEngineGuard(ctx, files, owner, false)
	if err != nil {
		return result, err
	}
	// Reload an already active manager only. A stopped engine stays stopped.
	state, err := i.command(ctx, "/usr/bin/systemctl", "show", fmt.Sprintf("user@%d.service", uid), "--property=ActiveState", "--value")
	if err != nil {
		return result, err
	}
	if strings.TrimSpace(string(state)) == "active" {
		_, err = i.command(ctx, "/usr/bin/systemctl", "--user", "--machine="+name+"@.host", "daemon-reload")
		result.Reloaded = err == nil
	}
	return result, err
}

// Prevalidate both artifacts before replacing either. Each atomic publication
// accepts old/current bytes, making an interrupted migration safely repeatable.
func (i *installer) updateEngineGuard(ctx context.Context, files map[string][]byte, owner *ownerhost.Profile, allowMissing bool) (bool, error) {
	if err := i.guardOverrides(ctx, owner); err != nil {
		return false, err
	}
	previous, err := rootless.PreviousServiceFiles()
	if err != nil {
		return false, err
	}
	dockerd, err := i.resolveHelper(ctx, "/usr/bin/dockerd")
	if err != nil {
		return false, err
	}
	type artifact struct {
		path    string
		data    []byte
		mode    os.FileMode
		replace bool
	}
	var changes []artifact
	for _, name := range []string{"check-engine.sh", rootless.UnitName} {
		target, ok := files[name]
		if !ok {
			return false, errors.New("missing reviewed engine service template")
		}
		known := [][]byte{target, previous[name]}
		if name == rootless.UnitName {
			known = append(known, bytes.ReplaceAll(previous[name], []byte("TimeoutStopSec=360"), []byte("TimeoutStopSec=120")))
		}
		if owner != nil {
			if name == "check-engine.sh" {
				// Initial owner setup could leave the exact unbound generated check.
				known = append(known, ownerEngineCheck(target, *owner), ownerEngineCheck(previous[name], *owner))
				target = ownerEngineCheck(target, *owner)
			} else {
				for index := range known {
					known[index] = ownerEngineUnit(known[index], *owner)
				}
				target = ownerEngineUnit(target, *owner)
			}
		}
		path, mode := rootless.InstallDir+"/"+name, os.FileMode(0755)
		if name == rootless.UnitName {
			path, mode = unitPath, 0644
			for _, template := range append([][]byte(nil), known...) {
				known = append(known, bytes.ReplaceAll(template, []byte("Environment=DOCKERD=/usr/bin/dockerd"), []byte("Environment=DOCKERD=/usr/sbin/dockerd")))
			}
			target = bytes.ReplaceAll(target, []byte("Environment=DOCKERD=/usr/bin/dockerd"), []byte("Environment=DOCKERD="+dockerd))
		}
		old, e := i.readInstalled(ctx, path, maxOutput)
		if e != nil {
			if !allowMissing || !errors.Is(e, os.ErrNotExist) {
				return false, e
			}
		} else {
			accepted := bytes.Equal(old, target)
			for _, candidate := range known {
				accepted = accepted || bytes.Equal(old, candidate)
			}
			if !accepted {
				return false, fmt.Errorf("%s has local changes; preserving it for review", path)
			}
		}
		changes = append(changes, artifact{path, target, mode, !bytes.Equal(old, target)})
	}
	updated := false
	for _, change := range changes {
		if err := i.installFile(ctx, change.path, change.data, change.mode, change.replace); err != nil {
			return updated, err
		}
		updated = updated || change.replace
	}
	return updated, nil
}

func (i *installer) guardOverrides(ctx context.Context, owner *ownerhost.Profile) error {
	home := rootless.HomeDir
	if owner != nil {
		home = owner.Home
	}
	for _, prefix := range []string{home + "/.config/systemd/user/", home + "/.local/share/systemd/user/", "/usr/lib/systemd/user/", "/run/systemd/user/"} {
		for _, suffix := range []string{"", ".d"} {
			path := prefix + rootless.UnitName + suffix
			if _, err := i.lstat(i.path(path)); !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("preserving unreviewed engine override %s", path)
			}
		}
	}
	dir := unitPath + ".d"
	info, err := i.lstat(i.path(dir))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	uid, _, ok := ownership(info)
	if err != nil || !ok || uid != 0 || !info.IsDir() || info.Mode().Perm()&0022 != 0 {
		return errors.New("engine override directory needs review")
	}
	entries, err := os.ReadDir(i.path(dir))
	if err != nil || len(entries) > 1 {
		return errors.New("engine overrides need review")
	}
	for _, entry := range entries {
		if entry.Name() != "shutdown.conf" {
			return errors.New("engine has a local service override; preserving it")
		}
		want, err := serviceunits.Files.ReadFile("urbit-docker.service.d/shutdown.conf")
		if err != nil {
			return err
		}
		got, err := i.readInstalled(ctx, dir+"/shutdown.conf", maxOutput)
		if err != nil || !bytes.Equal(got, want) {
			return errors.New("engine shutdown override has local changes; preserving it")
		}
	}
	return nil
}

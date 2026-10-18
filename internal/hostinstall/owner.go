package hostinstall

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"

	serviceunits "git.thechristophgroup.org/thelifeandtimes/omabit/deploy/systemd"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostsetup"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

func InstallOwnerHost(ctx context.Context, bundle string, public []byte, name string, progress func(string)) (HostResult, error) {
	return InstallOwnerHostWithOptions(ctx, bundle, public, name, HostOptions{}, progress)
}

type HostOptions struct{ OfflineUpgrade bool }

func InstallOwnerHostWithOptions(ctx context.Context, bundle string, public []byte, name string, options HostOptions, progress func(string)) (HostResult, error) {
	i := newInstaller()
	if i.euid != 0 || i.goos != "linux" || i.goarch != "amd64" {
		return HostResult{}, errors.New("owner setup requires OS authorization on Linux x86-64")
	}
	u, err := i.lookupUser(name)
	if err != nil {
		return HostResult{}, err
	}
	uid, e1 := accountID(u.Uid)
	gid, e2 := accountID(u.Gid)
	p := ownerhost.Profile{Version: 1, Name: u.Username, UID: uid, GID: gid, Home: u.HomeDir, Piers: ownerhost.DefaultPiers}
	if e1 != nil || e2 != nil {
		return HostResult{}, errors.New("invalid owner IDs")
	}
	if options.OfflineUpgrade {
		installed, err := ownerhost.Load()
		if err != nil || installed == nil || installed.Name != p.Name || installed.UID != p.UID || installed.GID != p.GID || installed.Home != p.Home {
			return HostResult{}, errors.New("offline upgrade requires the existing installed owner identity")
		}
		p = *installed // retain the installed pier root, including custom paths
	}
	f, err := os.Open("/proc/self/exe")
	if err != nil {
		return HostResult{}, err
	}
	defer f.Close()
	binary, err := io.ReadAll(io.LimitReader(f, (128<<20)+1))
	if err != nil {
		return HostResult{}, err
	}
	if options.OfflineUpgrade {
		return i.installOwnerOffline(ctx, bundle, public, binary, p, progress)
	}
	return i.installOwner(ctx, bundle, public, binary, p, probeHostRunner, progress)
}

func ownerManagerUnit(p ownerhost.Profile) []byte {
	return []byte(fmt.Sprintf(`[Unit]
Description=Omabit pier-owner ship management
ConditionUser=%s
Wants=urbit-docker.service
After=urbit-docker.service
StartLimitIntervalSec=60
StartLimitBurst=5

[Service]
Type=exec
WorkingDirectory=%%h
ExecStart=/usr/bin/env -i "HOME=%%h" PATH=/usr/bin:/bin LANG=C "XDG_RUNTIME_DIR=%%t" /usr/local/lib/urbit-runner/current/urbit-runner serve-development --acknowledge-development --resume-running
UMask=0077
NoNewPrivileges=yes
Restart=on-failure
RestartSec=5
KillMode=mixed
TimeoutStopSec=60

[Install]
WantedBy=default.target
`, p.Name))
}

func ownerEngineUnit(template []byte, p ownerhost.Profile) []byte {
	data := bytes.ReplaceAll(template, []byte("ConditionUser="+rootless.AccountName), []byte("ConditionUser="+p.Name))
	return bytes.ReplaceAll(data, []byte("Environment=HOME="+rootless.HomeDir), []byte(`Environment="HOME=%h"`))
}

func ownerEngineCheck(template []byte, p ownerhost.Profile) []byte {
	data := bytes.ReplaceAll(template, []byte("'"+rootless.AccountName+"'"), []byte("'"+p.Name+"'"))
	data = bytes.ReplaceAll(data, []byte("'"+rootless.HomeDir+"'"), []byte("'"+p.Home+"'"))
	return bytes.ReplaceAll(data, []byte("dedicated "+rootless.AccountName+" account"), []byte("selected pier-owning account"))
}

func (i *installer) ownerCommand(ctx context.Context, p ownerhost.Profile, path string, args ...string) ([]byte, error) {
	return i.command(ctx, "/usr/bin/runuser", append([]string{"-u", p.Name, "--", "/usr/bin/env", "-i", "HOME=" + p.Home, "PATH=/usr/bin:/bin", "XDG_RUNTIME_DIR=/run/user/" + strconv.FormatUint(uint64(p.UID), 10), path}, args...)...)
}

func freeSubIDs(text string, ids []uint64) (uint64, error) {
	type span struct{ a, b uint64 }
	var used []span
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, ":")
		if len(f) != 3 {
			return 0, errors.New("malformed subordinate IDs")
		}
		a, e1 := strconv.ParseUint(f[1], 10, 32)
		n, e2 := strconv.ParseUint(f[2], 10, 32)
		if e1 != nil || e2 != nil || n == 0 || a+n > 1<<32-1 {
			return 0, errors.New("invalid subordinate range")
		}
		used = append(used, span{a, a + n})
	}
	for _, id := range ids {
		used = append(used, span{id, id + 1})
	}
	for start := uint64(1 << 20); start+65536 < 1<<32-1; start += 65536 {
		free := true
		for _, s := range used {
			if start < s.b && s.a < start+65536 {
				free = false
				break
			}
		}
		if free {
			return start, nil
		}
	}
	return 0, errors.New("no free subordinate-ID range")
}

func (i *installer) ownerMappings(ctx context.Context, p ownerhost.Profile) error {
	nss, err := i.readInstalled(ctx, "/etc/nsswitch.conf", maxOutput)
	if err != nil {
		return fmt.Errorf("cannot establish identity authority: %w", err)
	}
	if !hostsetup.FileSubIDAuthority(string(nss)) {
		return errors.New("subordinate-ID NSS is not file-backed; configure reserved mappings through its authority")
	}
	local := hostsetup.LocalIdentityAuthority(string(nss))
	var reserved hostsetup.SubIDReservations
	if !local {
		data, err := i.readInstalled(ctx, hostsetup.SubIDReservationsPath, 8192)
		if err != nil {
			return errors.New("external or unknown NSS identity sources require root-owned subid-reservations.json and preconfigured reserved subuid/subgid ranges; enumeration is not authoritative")
		}
		reserved, err = hostsetup.ParseSubIDReservations(data)
		if err != nil {
			return err
		}
	}
	passwd, err := i.readInstalled(ctx, "/etc/passwd", maxOutput)
	if err != nil {
		return err
	}
	groups, err := i.readInstalled(ctx, "/etc/group", maxOutput)
	if err != nil {
		return err
	}
	uids, gids := hostsetup.LocalIDs(string(passwd), string(groups))
	if uids == nil || gids == nil {
		return errors.New("local ID inventory unavailable")
	}
	for _, item := range []struct {
		file, flag string
		ids        []uint64
	}{{"subuid", "--add-subuids", uids}, {"subgid", "--add-subgids", gids}} {
		path := "/etc/" + item.file
		data, err := i.readInstalled(ctx, path, maxOutput)
		if err != nil {
			return err
		}
		status, detail := hostsetup.ValidateSubIDs(p.Name, string(data), strconv.FormatUint(uint64(p.UID), 10), item.ids)
		if !local {
			span := reserved.UID
			if item.file == "subgid" {
				span = reserved.GID
			}
			status, detail = hostsetup.ValidateReservedSubIDs(p.Name, string(data), strconv.FormatUint(uint64(p.UID), 10), item.ids, span)
			if status != "pass" {
				return fmt.Errorf("configure reserved %s ranges explicitly: %s", item.file, detail)
			}
		}
		if status == "pass" {
			continue
		}
		if status != "missing" {
			return errors.New(detail)
		}
		start, err := freeSubIDs(string(data), item.ids)
		if err != nil {
			return err
		}
		i.mutated = true
		if _, err = i.command(ctx, "/usr/bin/usermod", item.flag, fmt.Sprintf("%d-%d", start, start+65535), p.Name); err != nil {
			return err
		}
		data, err = i.readInstalled(ctx, path, maxOutput)
		if err != nil {
			return err
		}
		if status, detail = hostsetup.ValidateSubIDs(p.Name, string(data), strconv.FormatUint(uint64(p.UID), 10), item.ids); status != "pass" {
			return errors.New(detail)
		}
	}
	return nil
}

func verifyOwnerInfo(data []byte, p ownerhost.Profile) error {
	var info struct {
		ServerVersion, CgroupDriver, CgroupVersion, DockerRootDir string
		SecurityOptions                                           []string
	}
	if len(data) > maxOutput || json.Unmarshal(data, &info) != nil || info.ServerVersion != rootless.EngineVersion || info.CgroupDriver != "systemd" || info.CgroupVersion != "2" || !slices.Contains(info.SecurityOptions, "name=rootless") || info.DockerRootDir != filepath.Join(p.Home, ".local/share/urbit-runner/docker") {
		return errors.New("explicit owner engine must be rootless with the pinned version, cgroup v2 and isolated user data root")
	}
	return nil
}

func (i *installer) installOwner(ctx context.Context, bundle string, public, binary []byte, p ownerhost.Profile, probe hostProbe, progress func(string)) (result HostResult, err error) {
	defer func() { result.Partial = err != nil && i.mutated }()
	if i.euid != 0 || i.goos != "linux" || i.goarch != "amd64" {
		return result, errors.New("owner setup requires root on Linux x86-64")
	}
	if err = p.Validate(); err != nil {
		return result, err
	}
	if len(binary) == 0 || len(binary) > 128<<20 {
		return result, errors.New("invalid runner executable")
	}
	line, err := access.OwnerAuthorizedKey(public, "operator")
	if err != nil {
		return result, err
	}
	grant, err := access.ParseLine(line)
	if err != nil {
		return result, err
	}
	result.UID, result.Owner, result.PierRoot, result.OperatorFingerprint = p.UID, p.Name, p.Piers, grant.Fingerprint
	for _, name := range append(append([]string{}, helpers...), hostHelpers...) {
		if err = i.helper(ctx, "/usr/bin/"+name); err != nil {
			return result, err
		}
	}
	if i.inspectOwner == nil {
		return result, errors.New("owner prerequisite inspector unavailable")
	}
	if err = requirePass(i.inspectOwner(p.Name), "platform", "systemd", "cgroup-v2", "userns-limit", "owner-account", "docker-version"); err != nil {
		return result, err
	}
	files, err := i.verified(bundle)
	if err != nil {
		return result, err
	}
	old, readErr := i.readInstalled(ctx, ownerhost.Path, 8192)
	if readErr == nil {
		var previous ownerhost.Profile
		if json.Unmarshal(old, &previous) != nil || previous != p {
			return result, errors.New("installed owner/path differs; explicit stopped migration is required")
		}
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return result, readErr
	} else {
		for _, path := range []string{rootless.HomeDir, managerUnit, unitPath, sshConfigPath, filepath.Join(p.Home, ".local/share/urbit-runner")} {
			if _, e := i.lstat(i.path(path)); !errors.Is(e, os.ErrNotExist) {
				return result, fmt.Errorf("existing installation at %s needs explicit review; no automatic account/pier migration", path)
			}
		}
	}
	for _, path := range []string{filepath.Join(p.Home, ".config/systemd/user/urbit-runner.service"), filepath.Join(p.Home, ".config/systemd/user/urbit-runner.service.d"), filepath.Join(p.Home, ".config/systemd/user/urbit-docker.service"), filepath.Join(p.Home, ".config/systemd/user/urbit-docker.service.d")} {
		if _, e := i.lstat(i.path(path)); !errors.Is(e, os.ErrNotExist) {
			return result, fmt.Errorf("preserving user service override %s", path)
		}
	}
	home, err := i.lstat(i.path(p.Home))
	if err != nil {
		return result, err
	}
	uid, _, ok := ownership(home)
	if !ok || !home.IsDir() || home.Mode()&os.ModeSymlink != 0 || uid != p.UID || home.Mode().Perm()&0022 != 0 {
		return result, errors.New("owner home must be owned by the selected user and not broadly writable")
	}
	if existing, e := i.lstat(i.path(p.Piers)); e == nil {
		u, g, ok := ownership(existing)
		if !ok || !existing.IsDir() || existing.Mode()&os.ModeSymlink != 0 || u != p.UID || g != p.GID || existing.Mode().Perm()&0022 != 0 {
			return result, errors.New("existing default pier root has incompatible ownership; preserving it")
		}
	} else if !errors.Is(e, os.ErrNotExist) {
		return result, e
	}
	if err = i.parents(ctx, ownerhost.Path, true); err != nil {
		return result, err
	}
	lockPath := "/etc/urbit-runner/.setup.lock"
	if _, e := i.readInstalled(ctx, lockPath, 0); e != nil && !errors.Is(e, os.ErrNotExist) {
		return result, e
	}
	lock, err := os.OpenFile(i.path(lockPath), os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0600)
	if err != nil {
		return result, err
	}
	defer lock.Close()
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return result, errors.New("another owner setup is running")
	}
	emit := func(step string) {
		if progress != nil {
			progress(step)
		}
	}
	if err = i.ownerMappings(ctx, p); err != nil {
		return result, err
	}
	if err = i.parents(ctx, p.Piers, true); err != nil {
		return result, err
	}
	info, e := i.lstat(i.path(p.Piers))
	if errors.Is(e, os.ErrNotExist) {
		if err = os.Mkdir(i.path(p.Piers), 0755); err != nil {
			return result, err
		}
		i.mutated = true
		if i.chownOwner == nil {
			return result, errors.New("owner directory preparation unavailable")
		}
		if err = i.chownOwner(i.path(p.Piers), p.UID, p.GID); err != nil {
			return result, err
		}
		info, e = i.lstat(i.path(p.Piers))
	}
	u, g, ok := ownership(info)
	if e != nil || !ok || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || u != p.UID || g != p.GID || info.Mode().Perm()&0022 != 0 {
		return result, errors.New("default pier root already exists with incompatible ownership; it was not recursively changed")
	}
	profile, _ := json.Marshal(p)
	if err = i.installFile(ctx, ownerhost.Path, append(profile, '\n'), 0644, false); err != nil {
		return result, err
	}
	emit("engine")
	for _, name := range []string{"dockerd-rootless.sh", "check-engine.sh", "daemon.json", "LICENSE", "NOTICE"} {
		mode := os.FileMode(0644)
		if strings.HasSuffix(name, ".sh") {
			mode = 0755
		}
		data, replace := files[name], false
		if name == "check-engine.sh" {
			data = ownerEngineCheck(data, p)
			// A partial owner install from the initial development implementation
			// may retain the exact verified legacy guard. Replace only that known
			// content, never an administrator's modified engine check.
			old, e := i.readInstalled(ctx, rootless.InstallDir+"/"+name, maxOutput)
			replace = e == nil && bytes.Equal(old, files[name])
		}
		if err = i.installFile(ctx, rootless.InstallDir+"/"+name, data, mode, replace); err != nil {
			return result, err
		}
	}
	if err = i.installFile(ctx, unitPath, ownerEngineUnit(files[rootless.UnitName], p), 0644, false); err != nil {
		return result, err
	}
	manager := func(args ...string) error {
		_, e := i.command(ctx, "/usr/bin/systemctl", append([]string{"--user", "--machine=" + p.Name + "@.host"}, args...)...)
		return e
	}
	if _, err = i.command(ctx, "/usr/bin/loginctl", "enable-linger", p.Name); err != nil {
		return result, err
	}
	i.mutated = true
	if _, err = i.command(ctx, "/usr/bin/systemctl", "start", fmt.Sprintf("user@%d.service", p.UID)); err != nil {
		return result, err
	}
	if err = manager("daemon-reload"); err != nil {
		return result, err
	}
	if err = manager("enable", "--now", rootless.UnitName); err != nil {
		return result, err
	}
	engine := fmt.Sprintf("/run/user/%d/%s", p.UID, rootless.SocketName)
	if err = i.checkSocket(ctx, engine, p.UID, p.GID); err != nil {
		return result, err
	}
	data, err := i.ownerCommand(ctx, p, "/usr/bin/docker", "--host", "unix://"+engine, "info", "--format", "{{json .}}")
	if err != nil {
		return result, err
	}
	if err = verifyOwnerInfo(data, p); err != nil {
		return result, err
	}
	socket := fmt.Sprintf("/run/user/%d/urbit-runner/runner.sock", p.UID)
	activation, err := i.prepareActivation(ctx, p.Name, socket, p.UID, binary, probe)
	if err != nil {
		return result, err
	}
	emit("access")
	keys, e := i.readInstalled(ctx, access.DefaultKeys, 65536)
	if e != nil && !errors.Is(e, os.ErrNotExist) {
		return result, e
	}
	found := false
	for _, raw := range strings.Split(string(keys), "\n") {
		raw = strings.TrimSpace(raw)
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		g, e := access.ParseLine(raw)
		if e != nil || !strings.Contains(raw, access.ShellPath+" owner-ssh ") {
			return result, errors.New("existing owner SSH grants are invalid")
		}
		if g.Fingerprint == grant.Fingerprint {
			if g.Role != "operator" {
				return result, errors.New("existing observer grant cannot be promoted implicitly")
			}
			found = true
		}
	}
	if !found {
		keys = append(bytes.TrimRight(keys, "\n"), []byte("\n"+line+"\n")...)
	}
	if len(keys) > 65536 {
		return result, errors.New("SSH grants are full")
	}
	key, e := i.readInstalled(ctx, hostKeyPath, 16384)
	if errors.Is(e, os.ErrNotExist) {
		if _, e := i.lstat(i.path(sshConfigPath)); e == nil {
			return result, errors.New("existing SSH host key is missing; refusing rotation")
		}
		if err = i.createHostKey(ctx); err != nil {
			return result, err
		}
	} else if e != nil || len(key) == 0 {
		return result, errors.New("SSH host key needs review")
	}
	keyInfo, err := i.lstat(i.path(hostKeyPath))
	if err != nil || keyInfo.Mode().Perm() != 0600 {
		return result, errors.New("SSH host key must be private")
	}
	pub, err := i.command(ctx, "/usr/bin/ssh-keygen", "-y", "-f", hostKeyPath)
	if err != nil {
		return result, err
	}
	if _, err = access.AuthorizedKey(pub, "operator"); err != nil {
		return result, err
	}
	if err = i.completeHostPublicKey(ctx, pub); err != nil {
		return result, err
	}
	result.HostPublicKey = strings.Join(strings.Fields(string(pub))[:2], " ")
	if err = i.installFile(ctx, access.DefaultKeys, keys, 0644, true); err != nil {
		return result, err
	}
	if err = i.installFile(ctx, sshConfigPath, []byte(access.OwnerSSHDConfig(SSHPort, p.Name)), 0644, false); err != nil {
		return result, err
	}
	if _, err = i.command(ctx, "/usr/bin/sshd", "-t", "-f", sshConfigPath); err != nil {
		return result, err
	}
	emit("services")
	if err = i.installFile(ctx, managerUnit, ownerManagerUnit(p), 0644, false); err != nil {
		return result, err
	}
	for _, item := range []struct{ asset, path string }{{"urbit-runner-ssh.service", sshUnit}, {"user-manager-shutdown.conf", fmt.Sprintf("/etc/systemd/system/user@%d.service.d/omabit-shutdown.conf", p.UID)}} {
		data, e := serviceunits.Files.ReadFile(item.asset)
		if e != nil {
			return result, e
		}
		if err = i.installFile(ctx, item.path, data, 0644, false); err != nil {
			return result, err
		}
	}
	if _, err = i.command(ctx, "/usr/bin/systemctl", "daemon-reload"); err != nil {
		return result, err
	}
	if err = manager("daemon-reload"); err != nil {
		return result, err
	}
	if _, err = i.command(ctx, "/usr/bin/systemctl", "enable", "--now", "urbit-runner-ssh.service"); err != nil {
		return result, err
	}
	emit("verify")
	status, err := i.activateManagement(ctx, socket, activation, manager)
	if err != nil {
		return result, err
	}
	result.RunnerID, result.Mode, result.SSHPort, result.Ready = status.RunnerID, status.Mode, SSHPort, true
	return result, nil
}

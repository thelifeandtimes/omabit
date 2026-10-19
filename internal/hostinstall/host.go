package hostinstall

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	serviceunits "git.thechristophgroup.org/thelifeandtimes/omabit/deploy/systemd"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

const SSHPort = 22222
const hostKeyPath = "/etc/urbit-runner/ssh_host_ed25519_key"
const sshConfigPath = "/etc/urbit-runner/sshd_config"
const managerUnit = "/etc/systemd/user/urbit-runner.service"
const sshUnit = "/etc/systemd/system/urbit-runner-ssh.service"

// HostResult contains only public connection data. The host key comes from the
// authorized local installer, so desktop setup never needs blind SSH key scanning.
type HostResult struct {
	Owner               string `json:"owner,omitempty"`
	PierRoot            string `json:"pier_root,omitempty"`
	Ready               bool   `json:"ready"`
	Partial             bool   `json:"partial"`
	UID                 uint32 `json:"uid"`
	RunnerID            string `json:"runner_id"`
	Mode                string `json:"mode"`
	SSHPort             uint   `json:"ssh_port"`
	HostPublicKey       string `json:"host_public_key"`
	OperatorFingerprint string `json:"operator_fingerprint"`
}

type hostProbe func(context.Context, string) (protocol.Hello, bool, error)

func InstallHost(ctx context.Context, bundle string, public []byte, progress func(string)) (HostResult, error) {
	i := newInstaller()
	if i.euid != 0 || i.goos != "linux" || i.goarch != "amd64" {
		return HostResult{}, errors.New("host setup requires OS authorization on Linux x86-64")
	}
	// Copy the running, OS-authorized executable, not a caller-supplied binary path.
	f, err := os.Open("/proc/self/exe")
	if err != nil {
		return HostResult{}, err
	}
	data, err := io.ReadAll(io.LimitReader(f, (128<<20)+1))
	f.Close()
	if err != nil || len(data) > 128<<20 {
		return HostResult{}, errors.New("management executable is unavailable or oversized")
	}
	return i.installHost(ctx, bundle, public, data, probeHostRunner, progress)
}

func probeHostRunner(ctx context.Context, socket string) (protocol.Hello, bool, error) {
	c := client.New(socket)
	defer c.Close()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	h, err := c.Hello(ctx)
	if err != nil {
		return h, false, err
	}
	if h.Mode != "development-managed" {
		return h, false, errors.New("existing runner has an unexpected mode")
	}
	s, err := c.Snapshot(ctx)
	if err != nil {
		return h, false, err
	}
	if s.RunnerID != h.RunnerID {
		return h, false, errors.New("runner identity changed during setup")
	}
	for _, v := range s.Instances {
		if v.ActiveOperationID != "" {
			return h, true, nil
		}
	}
	return h, false, nil
}

func (i *installer) readInstalled(ctx context.Context, path string, limit int64) ([]byte, error) {
	if err := i.parents(ctx, path, false); err != nil {
		return nil, err
	}
	info, err := i.lstat(i.path(path))
	if err != nil {
		return nil, err
	}
	uid, _, ok := ownership(info)
	if !ok || uid != 0 || !info.Mode().IsRegular() || info.Mode().Perm()&0022 != 0 {
		return nil, fmt.Errorf("preserving unsafe or unowned installation file %s", path)
	}
	f, err := os.OpenFile(i.path(path), os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	actual, err := f.Stat()
	if err != nil || !actual.Mode().IsRegular() || actual.Size() > limit {
		return nil, fmt.Errorf("invalid installation file %s", path)
	}
	if st, ok := actual.Sys().(*syscall.Stat_t); !ok || st.Nlink != 1 {
		return nil, errors.New("installation files must have exclusive links")
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil, fmt.Errorf("cannot read bounded installation file %s", path)
	}
	return data, ctx.Err()
}

func (i *installer) installFile(ctx context.Context, path string, data []byte, mode os.FileMode, replace bool) error {
	old, err := i.readInstalled(ctx, path, 128<<20)
	if err == nil {
		if bytes.Equal(old, data) {
			info, err := i.lstat(i.path(path))
			if err != nil {
				return err
			}
			if info.Mode().Perm() == mode {
				// Reestablish the publication barrier on retries, including after
				// an earlier directory-sync failure with complete bytes visible.
				return i.syncInstallDirectory(filepath.Dir(i.path(path)))
			}
			i.mutated = true
			return os.Chmod(i.path(path), mode)
		}
		if !replace {
			return fmt.Errorf("%s has local changes; preserving it for review", path)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := i.parents(ctx, path, true); err != nil {
		return err
	}
	if errors.Is(err, os.ErrNotExist) {
		return i.write(ctx, path, data, mode)
	}
	f, err := os.CreateTemp(filepath.Dir(i.path(path)), ".omabit-install-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, err = f.Write(data)
	err = errors.Join(err, i.chown(f), f.Chmod(mode), f.Sync(), f.Close())
	if err != nil {
		return err
	}
	if err := os.Rename(f.Name(), i.path(path)); err != nil {
		return err
	}
	i.mutated = true
	return i.syncInstallDirectory(filepath.Dir(i.path(path)))
}

func (i *installer) installHost(ctx context.Context, bundle string, public, binary []byte, probe hostProbe, progress func(string)) (result HostResult, err error) {
	defer func() { result.Partial = err != nil && i.mutated }()
	emit := func(step string) {
		if progress != nil {
			progress(step)
		}
	}
	if i.euid != 0 || i.goos != "linux" || i.goarch != "amd64" {
		return result, errors.New("host setup requires root on Linux x86-64")
	}
	if err := i.requireCompatibility(ctx); err != nil {
		return result, err
	}
	line, err := access.AuthorizedKey(public, "operator")
	if err != nil {
		return result, err
	}
	grant, err := access.ParseLine(line)
	if err != nil {
		return result, err
	}
	result.OperatorFingerprint = grant.Fingerprint
	if len(binary) == 0 || len(binary) > 128<<20 {
		return result, errors.New("invalid management executable")
	}
	for _, name := range hostHelpers {
		if err := i.helper(ctx, "/usr/bin/"+name); err != nil {
			return result, err
		}
	}
	files, err := i.verified(bundle)
	if err != nil {
		return result, err
	}
	lockPath := "/etc/urbit-runner/.setup.lock"
	if _, err := i.readInstalled(ctx, lockPath, 0); err != nil && !errors.Is(err, os.ErrNotExist) {
		return result, err
	}
	if err := i.parents(ctx, lockPath, true); err != nil {
		return result, err
	}
	lock, err := os.OpenFile(i.path(lockPath), os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return result, err
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return result, errors.New("another host setup is running; wait for it to finish")
	}
	emit("engine")
	account, lookupErr := i.lookupUser(rootless.AccountName)
	var absent user.UnknownUserError
	if errors.As(lookupErr, &absent) {
		if _, err := i.install(ctx, bundle); err != nil {
			return result, err
		}
		account, err = i.lookupUser(rootless.AccountName)
		if err != nil {
			return result, err
		}
	} else if lookupErr != nil {
		return result, lookupErr
	}
	uid, e1 := accountID(account.Uid)
	gid, e2 := accountID(account.Gid)
	if e1 != nil || e2 != nil || account.Username != rootless.AccountName || account.HomeDir != rootless.HomeDir {
		return result, errors.New("dedicated account identity does not match Omabit")
	}
	result.UID = uid
	if err := i.checkAccountMode(ctx, account, uid, gid, false); err != nil {
		return result, err
	}
	if err := requirePass(i.inspect(), "systemd", "cgroup-v2", "userns-limit", "account", "account-groups", "subuid", "subgid"); err != nil {
		return result, err
	}
	if err := i.noRootfulAccess(ctx); err != nil {
		return result, err
	}
	// Existing installations must retain the pinned launcher. Configuration and
	// unit edits remain the administrator's, rather than being overwritten blindly.
	for _, name := range []string{"dockerd-rootless.sh", "LICENSE", "NOTICE"} {
		data, err := i.readInstalled(ctx, rootless.InstallDir+"/"+name, maxOutput)
		if err != nil || !bytes.Equal(data, files[name]) {
			return result, fmt.Errorf("existing rootless artifact %s does not match the pinned launcher bundle", name)
		}
	}
	daemon, err := i.readInstalled(ctx, rootless.InstallDir+"/daemon.json", maxOutput)
	if err != nil {
		return result, err
	}
	if !bytes.Equal(daemon, files["daemon.json"]) && strings.TrimSpace(string(daemon)) != "{}" {
		return result, errors.New("dedicated Docker configuration has local changes; preserving it")
	}
	if _, err := i.updateEngineGuard(ctx, files, nil, false); err != nil {
		return result, err
	}
	userManager := func(args ...string) error {
		_, err := i.command(ctx, "/usr/bin/systemctl", append([]string{"--user", "--machine=" + rootless.AccountName + "@.host"}, args...)...)
		return err
	}
	i.mutated = true // service activation may proceed even if its acknowledgement is lost
	if err := userManager("daemon-reload"); err != nil {
		return result, err
	}
	if err := userManager("enable", rootless.UnitName); err != nil {
		return result, err
	}
	engineSocket := fmt.Sprintf("/run/user/%d/%s", uid, rootless.SocketName)
	if err := i.checkSocket(ctx, engineSocket, uid, gid); err != nil {
		return result, err
	}
	info, err := i.asAccount(ctx, "/usr/bin/docker", "--host", "unix://"+engineSocket, "info", "--format", "{{json .}}")
	if err != nil {
		return result, err
	}
	if err := verifyInfo(info); err != nil {
		return result, err
	}
	if err := i.verifyEngineAPI(ctx, engineSocket, nil, info); err != nil {
		return result, err
	}
	socket := fmt.Sprintf("/run/user/%d/urbit-runner/runner.sock", uid)
	activation, err := i.prepareActivation(ctx, rootless.AccountName, socket, uid, binary, probe)
	if err != nil {
		return result, err
	}
	emit("access")
	config, err := i.readInstalled(ctx, sshConfigPath, maxOutput)
	configExists := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return result, err
	}
	if configExists && string(config) != access.SSHDConfig(SSHPort) {
		return result, errors.New("dedicated SSH configuration has local changes; preserving it")
	}
	keys, err := i.readInstalled(ctx, access.DefaultKeys, 65536)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return result, err
	}
	found := false
	for _, text := range strings.Split(string(keys), "\n") {
		text = strings.TrimSpace(text)
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		g, err := access.ParseLine(text)
		if err != nil {
			return result, errors.New("existing SSH grants are invalid; preserving them")
		}
		if g.Fingerprint == grant.Fingerprint {
			if g.Role != "operator" {
				return result, errors.New("this key has a non-operator grant; refusing to change its role")
			}
			found = true
		}
	}
	if !found {
		keys = append(bytes.TrimRight(keys, "\n"), []byte("\n"+line+"\n")...)
	}
	if len(keys) > 65536 {
		return result, errors.New("SSH grant file is full")
	}
	key, err := i.readInstalled(ctx, hostKeyPath, 16384)
	if errors.Is(err, os.ErrNotExist) {
		if configExists {
			return result, errors.New("existing SSH host key is missing; refusing to rotate host identity")
		}
		if _, err := i.lstat(i.path(hostKeyPath + ".pub")); !errors.Is(err, os.ErrNotExist) {
			return result, errors.New("orphaned SSH host public key; inspect before regenerating identity")
		}
		if err := i.parents(ctx, hostKeyPath, true); err != nil {
			return result, err
		}
		if err := i.createHostKey(ctx); err != nil {
			return result, err
		}
		i.mutated = true
	} else if err != nil || len(key) == 0 {
		return result, errors.New("SSH host key is unavailable; preserving identity")
	}
	keyInfo, err := i.lstat(i.path(hostKeyPath))
	if err != nil || keyInfo.Mode().Perm() != 0600 {
		return result, errors.New("SSH host private key must have mode 0600")
	}
	pub, err := i.command(ctx, "/usr/bin/ssh-keygen", "-y", "-f", hostKeyPath)
	if err != nil {
		return result, err
	}
	if _, err := access.AuthorizedKey(pub, "operator"); err != nil {
		return result, errors.New("SSH host key must be Ed25519")
	}
	if err := i.completeHostPublicKey(ctx, pub); err != nil {
		return result, err
	}
	result.HostPublicKey = strings.Join(strings.Fields(string(pub))[:2], " ")
	if err := i.installFile(ctx, access.DefaultKeys, keys, 0644, true); err != nil {
		return result, err
	}
	if err := i.installFile(ctx, sshConfigPath, []byte(access.SSHDConfig(SSHPort)), 0644, false); err != nil {
		return result, err
	}
	if _, err := i.command(ctx, "/usr/bin/sshd", "-t", "-f", sshConfigPath); err != nil {
		return result, err
	}
	emit("services")
	for _, item := range []struct{ asset, path string }{
		{"urbit-runner.service", managerUnit}, {"urbit-runner-ssh.service", sshUnit},
		{"urbit-docker.service.d/shutdown.conf", "/etc/systemd/user/urbit-docker.service.d/shutdown.conf"},
		{"user-manager-shutdown.conf", fmt.Sprintf("/etc/systemd/system/user@%d.service.d/omabit-shutdown.conf", uid)},
	} {
		data, err := serviceunits.Files.ReadFile(item.asset)
		if err != nil {
			return result, err
		}
		if item.path == sshUnit {
			data, err = i.nativeSSHUnit(ctx, data)
			if err != nil {
				return result, err
			}
		}
		if err := i.installFile(ctx, item.path, data, 0644, false); err != nil {
			return result, err
		}
	}
	if err := userManager("daemon-reload"); err != nil {
		return result, err
	}
	if _, err := i.command(ctx, "/usr/bin/usermod", "--shell", access.ShellPath, rootless.AccountName); err != nil {
		return result, err
	}
	if err := i.installFile(ctx, rootless.InstallDir+"/daemon.json", files["daemon.json"], 0644, true); err != nil {
		return result, err
	}
	if _, err := i.command(ctx, "/usr/bin/systemctl", "daemon-reload"); err != nil {
		return result, err
	}
	if err := userManager("daemon-reload"); err != nil {
		return result, err
	}
	if _, err := i.command(ctx, "/usr/bin/systemctl", "enable", "--now", "urbit-runner-ssh.service"); err != nil {
		return result, err
	}
	emit("verify")
	status, err := i.activateManagement(ctx, socket, activation, userManager)
	if err != nil {
		return result, err
	}
	if activation.Stage == "complete" {
		if err := i.installPackageHooks(ctx, binary); err != nil {
			return result, err
		}
	}
	result.RunnerID, result.Mode, result.SSHPort, result.Ready = status.RunnerID, status.Mode, SSHPort, true
	return result, nil
}

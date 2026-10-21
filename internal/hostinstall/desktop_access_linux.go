package hostinstall

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"golang.org/x/sys/unix"
)

// DesktopAccessOptions changes only this key's grant and its forced-command SSH
// entry. It does not install binaries, restart services, or open the pier database.
type DesktopAccessOptions struct {
	RunnerID      string
	Role          string
	SSHConnection string // OpenSSH's client-address/port, server-address/port tuple.
	Revoke        bool
}

type DesktopAccessResult struct {
	Version     int    `json:"version"`
	RunnerID    string `json:"runner_id"`
	Mode        string `json:"mode"`
	Owner       string `json:"owner"`
	Fingerprint string `json:"fingerprint"`
	Role        string `json:"role"`
	Port        uint   `json:"port"`
	State       string `json:"state"`
	Partial     bool   `json:"partial"`
}

func AuthorizeDesktop(ctx context.Context, public []byte, options DesktopAccessOptions) (DesktopAccessResult, error) {
	i := newInstaller()
	if i.euid != 0 || i.goos != "linux" || i.goarch != "amd64" {
		return DesktopAccessResult{}, errors.New("desktop access requires local OS authorization on Linux x86-64")
	}
	p, err := ownerhost.Load()
	if err != nil || p == nil {
		return DesktopAccessResult{}, errors.New("an installed owner-user runner is required")
	}
	u, err := ownerhost.ForUser(p.Name)
	if err != nil || u.UID != p.UID || u.GID != p.GID || u.Home != p.Home {
		return DesktopAccessResult{}, errors.New("installed owner identity changed")
	}
	return i.desktopAccess(ctx, *p, public, options, probeHostRunner)
}

func sshContext(owner, connection string) (string, uint, error) {
	parts := strings.Fields(connection)
	if len(parts) != 4 || net.ParseIP(parts[0]) == nil || net.ParseIP(parts[2]) == nil {
		return "", 0, errors.New("require the authenticated SSH connection addresses")
	}
	for _, index := range []int{1, 3} {
		port, err := strconv.ParseUint(parts[index], 10, 16)
		if err != nil || port == 0 {
			return "", 0, errors.New("invalid SSH port")
		}
	}
	port, _ := strconv.ParseUint(parts[3], 10, 16)
	return "user=" + owner + ",host=" + parts[0] + ",addr=" + parts[0] + ",laddr=" + parts[2] + ",lport=" + parts[3], uint(port), nil
}

func supportsDesktopKey(config string, p ownerhost.Profile) error {
	values := map[string]string{}
	for _, line := range strings.Split(config, "\n") {
		key, value, ok := strings.Cut(line, " ")
		if ok {
			values[strings.ToLower(key)] = strings.TrimSpace(value)
		}
	}
	if values["pubkeyauthentication"] != "yes" || values["forcecommand"] != "none" || values["chrootdirectory"] != "none" || values["usedns"] != "no" {
		return errors.New("this SSH policy cannot use an owner forced-command key automatically; retain the dedicated runner endpoint or review the server policy")
	}
	if method := values["authenticationmethods"]; method != "any" && method != "publickey" {
		return errors.New("the SSH endpoint requires additional authentication; a dedicated key alone cannot reconnect unattended")
	}
	for _, path := range strings.Fields(values["authorizedkeysfile"]) {
		if path == ".ssh/authorized_keys" || path == "%h/.ssh/authorized_keys" || path == p.Home+"/.ssh/authorized_keys" {
			return nil
		}
	}
	return errors.New("SSH does not read the owner's .ssh/authorized_keys; existing authorization configuration was preserved")
}

func CheckDesktopSSH(ctx context.Context, p ownerhost.Profile, connection string) error {
	i := newInstaller()
	if i.euid != 0 {
		return errors.New("SSH authorization inspection requires OS authorization")
	}
	return i.checkDesktopSSH(ctx, p, connection)
}

func (i *installer) checkDesktopSSH(ctx context.Context, p ownerhost.Profile, connection string) error {
	if err := p.Validate(); err != nil {
		return err
	}
	context, _, err := sshContext(p.Name, connection)
	if err != nil {
		return err
	}
	sshd, err := i.resolveHelper(ctx, "/usr/bin/sshd")
	if err != nil {
		return err
	}
	config, err := i.command(ctx, sshd, "-T", "-C", context)
	if err != nil {
		return fmt.Errorf("inspect effective SSH authorization policy: %w", err)
	}
	return supportsDesktopKey(string(config), p)
}

// keyLines preserves all unrelated bytes. The personal file can contain arbitrary
// OpenSSH keys; Omabit's central grant file must still satisfy its strict parser.
func desktopKeyLines(data []byte, line string, central, revoke bool) ([]byte, error) {
	want, err := access.ParseLine(line)
	if err != nil || len(data) > 65536 {
		return nil, errors.New("invalid or oversized SSH grant file")
	}
	public := strings.Fields(line)[len(strings.Fields(line))-1]
	var result []byte
	found := false
	seen := map[string]bool{}
	for _, raw := range bytes.SplitAfter(data, []byte("\n")) {
		text := strings.TrimSpace(string(raw))
		if text != "" && !strings.HasPrefix(text, "#") {
			g, parseErr := access.ParseLine(text)
			if central {
				if parseErr != nil || seen[g.Fingerprint] {
					return nil, errors.New("existing management grants are invalid; preserving them")
				}
				seen[g.Fingerprint] = true
			}
			containsKey := false
			for _, field := range strings.Fields(text) {
				containsKey = containsKey || field == public
			}
			if containsKey || parseErr == nil && g.Fingerprint == want.Fingerprint {
				if parseErr != nil || g != want || text != line || found {
					return nil, errors.New("this public key already has a different SSH grant; preserve it and use a distinct desktop key")
				}
				found = true
				if revoke {
					continue
				}
			}
		}
		result = append(result, raw...)
	}
	if !found && !revoke {
		if len(result) > 0 && result[len(result)-1] != '\n' {
			result = append(result, '\n')
		}
		result = append(result, []byte(line+"\n")...)
	}
	if len(result) > 65536 {
		return nil, errors.New("SSH grant file is full")
	}
	return result, nil
}

func (i *installer) desktopAccess(ctx context.Context, p ownerhost.Profile, public []byte, options DesktopAccessOptions, probe hostProbe) (result DesktopAccessResult, err error) {
	defer func() { result.Partial = err != nil && i.mutated }()
	if err := p.Validate(); err != nil || !protocol.ValidID(options.RunnerID) {
		return result, errors.New("require the installed owner and expected runner identity")
	}
	line, err := access.OwnerAuthorizedKey(public, options.Role)
	if err != nil {
		return result, err
	}
	grant, _ := access.ParseLine(line)
	_, port, err := sshContext(p.Name, options.SSHConnection)
	if err != nil {
		return result, err
	}
	lockPath := "/etc/urbit-runner/.setup.lock"
	if _, err = i.readInstalled(ctx, lockPath, 0); err != nil && !errors.Is(err, os.ErrNotExist) {
		return result, err
	}
	lock, err := os.OpenFile(i.path(lockPath), os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return result, err
	}
	defer lock.Close()
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return result, errors.New("another host setup/access change is running")
	}
	id := sha256.Sum256([]byte(grant.Fingerprint))
	receipt := "/etc/urbit-runner/desktop-access/" + hex.EncodeToString(id[:]) + ".json"
	var saved DesktopAccessResult
	oldReceipt, receiptErr := i.readInstalled(ctx, receipt, 16384)
	if receiptErr == nil {
		if json.Unmarshal(oldReceipt, &saved) != nil || saved.Version != 1 || saved.Owner != p.Name || saved.RunnerID != options.RunnerID || saved.Fingerprint != grant.Fingerprint || saved.Role != grant.Role {
			return result, errors.New("desktop access receipt changed; preserving it for review")
		}
	} else if !errors.Is(receiptErr, os.ErrNotExist) {
		return result, receiptErr
	}
	h, _, err := probe(ctx, fmt.Sprintf("/run/user/%d/urbit-runner/runner.sock", p.UID))
	if err != nil && options.Revoke && receiptErr == nil {
		h = protocol.Hello{RunnerID: saved.RunnerID, Mode: saved.Mode}
		err = nil
	}
	if err != nil || h.RunnerID != options.RunnerID || h.Mode != "development-managed" {
		return result, errors.New("installed runner is unavailable or its identity changed")
	}
	if !options.Revoke {
		if e := i.checkDesktopSSH(ctx, p, options.SSHConnection); e != nil {
			return result, e
		}
	}
	keys, err := i.readInstalled(ctx, access.DefaultKeys, 65536)
	if err != nil {
		return result, err
	}
	updated, err := desktopKeyLines(keys, line, true, options.Revoke)
	if err != nil {
		return result, err
	}
	if _, e := os.Lstat(filepath.Join(i.path(p.Home), ".ssh")); errors.Is(e, os.ErrNotExist) && !options.Revoke {
		i.mutated = true
	}
	directory, err := openOwnerSSH(i.path(p.Home), p, !options.Revoke)
	if err != nil && !(options.Revoke && errors.Is(err, unix.ENOENT)) {
		return result, err
	}
	var personal []byte
	var mode os.FileMode = 0600
	if directory >= 0 {
		defer unix.Close(directory)
		personal, mode, err = readOwnerKeys(directory, p)
	} else {
		err = nil
	}
	if err != nil {
		return result, err
	}
	newPersonal, err := desktopKeyLines(personal, line, false, options.Revoke)
	if err != nil {
		return result, err
	}
	result = DesktopAccessResult{Version: 1, RunnerID: h.RunnerID, Mode: h.Mode, Owner: p.Name, Fingerprint: grant.Fingerprint, Role: grant.Role, Port: port, State: "pending"}
	data, _ := json.Marshal(result)
	if err = i.installFile(ctx, receipt, data, 0600, true); err != nil {
		return result, err
	}
	// Revocation first removes the authoritative per-request grant, including
	// existing sessions. A retry repairs only this exact personal key entry.
	if err = i.installFile(ctx, access.DefaultKeys, updated, 0644, true); err != nil {
		return result, err
	}
	if directory >= 0 {
		err = replaceOwnerKeys(directory, p, personal, newPersonal, mode)
	}
	if err != nil {
		return result, err
	}
	result.State = "authorized"
	if options.Revoke {
		result.State = "revoked"
	}
	data, _ = json.Marshal(result)
	err = i.installFile(ctx, receipt, data, 0600, true)
	return result, err
}

// Hold the owner home and .ssh as directory descriptors. Root never follows a
// user-controlled symlink or writes through a multiply-linked authorized_keys.
func openOwnerSSH(home string, p ownerhost.Profile, create bool) (int, error) {
	fd, err := unix.Openat2(unix.AT_FDCWD, home, &unix.OpenHow{Flags: unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC, Resolve: unix.RESOLVE_NO_SYMLINKS})
	if err != nil {
		return -1, err
	}
	defer unix.Close(fd)
	var st unix.Stat_t
	if unix.Fstat(fd, &st) != nil || st.Uid != p.UID || st.Mode&0022 != 0 {
		return -1, errors.New("owner home identity or permissions changed")
	}
	ssh, err := unix.Openat(fd, ".ssh", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if errors.Is(err, unix.ENOENT) && create {
		if err = unix.Mkdirat(fd, ".ssh", 0700); err != nil {
			return -1, err
		}
		ssh, err = unix.Openat(fd, ".ssh", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
		if err == nil {
			err = unix.Fchown(ssh, int(p.UID), int(p.GID))
		}
	}
	if err != nil {
		if ssh >= 0 {
			unix.Close(ssh)
		}
		return -1, err
	}
	if unix.Fstat(ssh, &st) != nil || st.Uid != p.UID || st.Mode&0022 != 0 {
		unix.Close(ssh)
		return -1, errors.New("preserving unsafe or foreign-owned .ssh directory")
	}
	return ssh, nil
}

func readOwnerKeys(directory int, p ownerhost.Profile) ([]byte, os.FileMode, error) {
	fd, err := unix.Openat(directory, "authorized_keys", unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if errors.Is(err, unix.ENOENT) {
		return nil, 0600, nil
	}
	if err != nil {
		return nil, 0, err
	}
	f := os.NewFile(uintptr(fd), "authorized_keys")
	defer f.Close()
	var st unix.Stat_t
	if unix.Fstat(fd, &st) != nil || st.Mode&unix.S_IFMT != unix.S_IFREG || st.Nlink != 1 || st.Mode&0022 != 0 || st.Uid != p.UID || st.Size > 65536 {
		return nil, 0, errors.New("preserving unsafe or foreign-owned authorized_keys")
	}
	data, err := io.ReadAll(io.LimitReader(f, 65537))
	if len(data) > 65536 {
		return nil, 0, errors.New("authorized_keys exceeds its bound")
	}
	return data, os.FileMode(st.Mode & 0777), err
}

func replaceOwnerKeys(directory int, p ownerhost.Profile, before, after []byte, mode os.FileMode) error {
	if bytes.Equal(before, after) {
		return nil
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return err
	}
	name := ".omabit-access-" + hex.EncodeToString(random[:])
	fd, err := unix.Openat(directory, name, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW, uint32(mode))
	if err != nil {
		return err
	}
	defer unix.Unlinkat(directory, name, 0)
	f := os.NewFile(uintptr(fd), name)
	_, err = f.Write(after)
	err = errors.Join(err, f.Chown(int(p.UID), int(p.GID)), f.Chmod(mode), f.Sync(), f.Close())
	if err != nil {
		return err
	}
	current, _, err := readOwnerKeys(directory, p)
	if err != nil || !bytes.Equal(current, before) {
		return errors.New("authorized_keys changed during setup; preserving the current file")
	}
	if err = unix.Renameat(directory, name, directory, "authorized_keys"); err != nil {
		return err
	}
	return unix.Fsync(directory)
}

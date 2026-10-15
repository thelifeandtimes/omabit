// Package hostinstall is a development-only, OS-authorized first-install helper.
// It is not a package installer, updater, rollback mechanism, or runtime qualifier.
package hostinstall

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostsetup"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

const unitPath = "/etc/systemd/user/" + rootless.UnitName

// Result reports completed steps, including on error. An attempted operation may
// have additional side effects when Partial is true. RootlessVerified means only
// the explicitly addressed server's self-reported API fields passed our checks.
type Result struct {
	AccountCreated   bool   `json:"account_created"`
	Account          string `json:"account,omitempty"`
	UID              uint32 `json:"uid,omitempty"`
	GID              uint32 `json:"gid,omitempty"`
	Service          string `json:"service,omitempty"`
	Socket           string `json:"socket,omitempty"`
	RootlessVerified bool   `json:"rootless_verified"`
	RuntimeQualified bool   `json:"runtime_qualified"`
	Partial          bool   `json:"partial"`
}

// Install requires explicit OS authorization and acknowledgement by its caller,
// plus euid 0 on linux/amd64. Packages must already be installed by the operator.
// No bundle code is executed as root. Failures preserve all created resources;
// this helper never takes over an existing installation or deletes data.
func Install(ctx context.Context, bundle string) (Result, error) {
	i := newInstaller()
	return i.install(ctx, bundle)
}

func newInstaller() installer {
	return installer{
		root: "/", euid: os.Geteuid(), goos: runtime.GOOS, goarch: runtime.GOARCH,
		verified: rootless.VerifiedFiles, inspect: hostsetup.Inspect,
		inspectOwner: hostsetup.InspectOwner,
		chownOwner:   func(path string, uid, gid uint32) error { return os.Chown(path, int(uid), int(gid)) },
		lookupUser:   user.Lookup, lookupGroup: user.LookupGroup, groupIDs: (*user.User).GroupIds,
		lstat: os.Lstat, chown: func(f *os.File) error { return f.Chown(0, 0) },
		run: runCommand,
	}
}

// All injection is private and per invocation. Tests use a temporary filesystem,
// synthetic ownership/NSS evidence, and a runner that never executes a process.
type installer struct {
	root         string
	euid         int
	goos, goarch string
	verified     func(string) (map[string][]byte, error)
	inspect      func() hostsetup.Report
	inspectOwner func(string) hostsetup.Report
	chownOwner   func(string, uint32, uint32) error
	lookupUser   func(string) (*user.User, error)
	lookupGroup  func(string) (*user.Group, error)
	groupIDs     func(*user.User) ([]string, error)
	lstat        func(string) (os.FileInfo, error)
	chown        func(*os.File) error
	run          func(context.Context, string, []string, []string) ([]byte, error)
	mutated      bool
}

func (i *installer) install(ctx context.Context, bundle string) (result Result, err error) {
	defer func() {
		if err != nil && i.mutated {
			result.Partial = true
			err = fmt.Errorf("partial first install: %w; resources preserved, manual review required (no automatic cleanup or force-rerun)", err)
		}
	}()
	if ctx == nil {
		return result, fmt.Errorf("installation requires a non-nil context")
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if i.euid != 0 || i.goos != "linux" || i.goarch != "amd64" {
		return result, fmt.Errorf("installation requires euid 0 on linux/amd64")
	}
	files, err := i.verified(bundle)
	if err != nil {
		return result, fmt.Errorf("verify bundle: %w", err)
	}
	// Only this fixed inventory is installed, exclusively from the verified bytes.
	artifacts := []struct {
		name, path string
		mode       os.FileMode
	}{
		{"dockerd-rootless.sh", rootless.InstallDir + "/dockerd-rootless.sh", 0755},
		{"check-engine.sh", rootless.InstallDir + "/check-engine.sh", 0755},
		{"daemon.json", rootless.InstallDir + "/daemon.json", 0644},
		{"LICENSE", rootless.InstallDir + "/LICENSE", 0644},
		{"NOTICE", rootless.InstallDir + "/NOTICE", 0644},
		{rootless.UnitName, unitPath, 0644},
	}
	for _, a := range artifacts {
		if _, ok := files[a.name]; !ok {
			return result, fmt.Errorf("verified bundle is missing %s", a.name)
		}
	}
	if err := i.absentIdentity(); err != nil {
		return result, err
	}
	// Refuse stale linger and unit overrides as well as the primary destinations.
	for _, path := range []string{
		rootless.HomeDir, rootless.InstallDir, unitPath, unitPath + ".d",
		"/usr/lib/systemd/user/" + rootless.UnitName,
		"/usr/lib/systemd/user/" + rootless.UnitName + ".d",
		"/run/systemd/user/" + rootless.UnitName,
		"/run/systemd/user/" + rootless.UnitName + ".d",
		"/var/lib/systemd/linger/" + rootless.AccountName,
	} {
		if err := i.parents(ctx, path, false); err != nil {
			return result, err
		}
		if err := i.absent(path); err != nil {
			return result, err
		}
	}
	for _, name := range helpers {
		if err := i.helper(ctx, "/usr/bin/"+name); err != nil {
			return result, err
		}
	}
	// The UID-specific runtime directory is checked later; its root-owned
	// parents can and must be validated before any installation writes.
	if err := i.parents(ctx, "/run/user/"+rootless.SocketName, false); err != nil {
		return result, err
	}
	report := i.inspect()
	if err := requirePass(report, "systemd", "cgroup-v2", "userns-limit", "binary-iptables"); err != nil {
		return result, err
	}
	// This optional sysctl is absent on some supported kernels. Other unknown
	// evidence (including malformed values) must not silently become a pass.
	if _, statErr := i.lstat(i.path("/proc/sys/kernel/unprivileged_userns_clone")); !errors.Is(statErr, os.ErrNotExist) {
		if statErr != nil {
			return result, fmt.Errorf("inspect optional user namespace control: %w", statErr)
		}
		if err := requirePass(report, "userns-clone"); err != nil {
			return result, err
		}
	}
	version, err := i.command(ctx, "/usr/bin/dockerd", "--version")
	if err != nil {
		return result, err
	}
	prefix := "Docker version " + rootless.EngineVersion + ", build "
	text := strings.TrimSpace(string(version))
	if !strings.HasPrefix(text, prefix) || len(strings.Fields(strings.TrimPrefix(text, prefix))) != 1 {
		return result, fmt.Errorf("/usr/bin/dockerd must report exactly Docker version %s", rootless.EngineVersion)
	}
	help, err := i.command(ctx, "/usr/bin/useradd", "--help")
	if err != nil {
		return result, err
	}
	if !slices.Contains(strings.Fields(string(help)), "--add-subids-for-system") {
		return result, fmt.Errorf("useradd must support --add-subids-for-system; install prerequisites manually")
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	for _, a := range artifacts {
		if err := i.parents(ctx, a.path, true); err != nil {
			return result, err
		}
		if err := i.write(ctx, a.path, files[a.name], a.mode); err != nil {
			return result, err
		}
	}
	if err := i.parents(ctx, rootless.HomeDir, true); err != nil {
		return result, err
	}
	// Recheck immediately before useradd, rather than reusing an NSS result from
	// before file installation. user.Lookup (unlike user.Current) is not cached.
	if err := i.absentIdentity(); err != nil {
		return result, err
	}
	if err := i.absent(rootless.HomeDir); err != nil {
		return result, err
	}
	_, err = i.command(ctx, "/usr/bin/useradd",
		"--system", "--user-group", "--create-home", "--home-dir", rootless.HomeDir,
		"--shell", "/usr/bin/nologin", "--groups", "", "--add-subids-for-system",
		"--key", "HOME_MODE=0700", "--key", "SUB_UID_COUNT=65536", "--key", "SUB_GID_COUNT=65536", rootless.AccountName)
	if err != nil {
		return result, err
	}
	result.AccountCreated, result.Account = true, rootless.AccountName
	account, err := i.lookupUser(rootless.AccountName)
	if err != nil {
		return result, fmt.Errorf("resolve newly created account through NSS: %w", err)
	}
	uid, uidErr := accountID(account.Uid)
	gid, gidErr := accountID(account.Gid)
	if uidErr != nil || gidErr != nil || account.Username != rootless.AccountName || account.HomeDir != rootless.HomeDir {
		return result, fmt.Errorf("new account must have valid non-root IDs and the expected name/home")
	}
	result.UID, result.GID = uid, gid
	if err := i.checkAccount(ctx, account, uid, gid); err != nil {
		return result, err
	}
	if err := requirePass(i.inspect(), "account", "account-groups", "subuid", "subgid"); err != nil {
		return result, err
	}
	if err := i.noRootfulAccess(ctx); err != nil {
		return result, err
	}
	uidText := strconv.FormatUint(uint64(uid), 10)
	for _, cmd := range [][]string{
		{"/usr/bin/loginctl", "enable-linger", rootless.AccountName},
		{"/usr/bin/systemctl", "start", "user@" + uidText + ".service"},
		{"/usr/bin/systemctl", "--user", "--machine=" + rootless.AccountName + "@.host", "daemon-reload"},
		{"/usr/bin/systemctl", "--user", "--machine=" + rootless.AccountName + "@.host", "enable", "--now", rootless.UnitName},
	} {
		if _, err := i.command(ctx, cmd[0], cmd[1:]...); err != nil {
			return result, err
		}
	}
	result.Service = rootless.UnitName
	socket := "/run/user/" + uidText + "/" + rootless.SocketName
	if err := i.checkSocket(ctx, socket, uid, gid); err != nil {
		return result, err
	}
	result.Socket = socket
	info, err := i.asAccount(ctx, "/usr/bin/docker", "--host", "unix://"+socket, "info", "--format", "{{json .}}")
	if err != nil {
		return result, err
	}
	if err := verifyInfo(info); err != nil {
		return result, err
	}
	result.RootlessVerified = true
	return result, nil
}

func (i *installer) absentIdentity() error {
	if _, err := i.lookupUser(rootless.AccountName); err == nil {
		return fmt.Errorf("refusing existing NSS account %s", rootless.AccountName)
	} else {
		var absent user.UnknownUserError
		if !errors.As(err, &absent) {
			return fmt.Errorf("NSS account lookup: %w", err)
		}
	}
	if _, err := i.lookupGroup(rootless.AccountName); err == nil {
		return fmt.Errorf("refusing existing NSS group %s", rootless.AccountName)
	} else {
		var absent user.UnknownGroupError
		if !errors.As(err, &absent) {
			return fmt.Errorf("NSS group lookup: %w", err)
		}
	}
	return nil
}

func accountID(text string) (uint32, error) {
	n, err := strconv.ParseUint(text, 10, 32)
	if err != nil || n == 0 || n == 1<<32-1 || strconv.FormatUint(n, 10) != text {
		return 0, fmt.Errorf("invalid non-root account ID")
	}
	return uint32(n), nil
}

func requirePass(report hostsetup.Report, ids ...string) error {
	for _, id := range ids {
		found := false
		for _, check := range report.Checks {
			if check.ID == id {
				if check.Status != "pass" {
					return fmt.Errorf("host prerequisite %s is %s: %s", id, check.Status, check.Detail)
				}
				found = true
			}
		}
		if !found {
			return fmt.Errorf("host prerequisite %s has no evidence", id)
		}
	}
	return nil
}

func (i *installer) checkAccount(ctx context.Context, account *user.User, uid, gid uint32) error {
	return i.checkAccountMode(ctx, account, uid, gid, true)
}

func (i *installer) checkAccountMode(ctx context.Context, account *user.User, uid, gid uint32, fresh bool) error {
	group, err := i.lookupGroup(rootless.AccountName)
	if err != nil || group.Name != rootless.AccountName || group.Gid != account.Gid {
		return fmt.Errorf("new account does not have the expected private NSS group")
	}
	groups, err := i.groupIDs(account)
	if err != nil || len(groups) == 0 {
		return fmt.Errorf("cannot establish new account's NSS group memberships")
	}
	for _, id := range groups {
		if id != account.Gid {
			return fmt.Errorf("new account has unexpected supplementary NSS group membership")
		}
	}
	for _, name := range []string{"root", "wheel", "sudo", "admin", "docker"} {
		group, err := i.lookupGroup(name)
		var absent user.UnknownGroupError
		if err != nil && !errors.As(err, &absent) {
			return fmt.Errorf("cannot check privileged NSS group %s: %w", name, err)
		}
		if err == nil && group.Gid == account.Gid {
			return fmt.Errorf("private GID aliases privileged NSS group %s", name)
		}
	}
	info, err := i.lstat(i.path(rootless.HomeDir))
	u, g, ok := ownership(info)
	if err != nil || !ok || !info.IsDir() || info.Mode().Perm() != 0700 || info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 || u != uid || g != gid {
		return fmt.Errorf("new home must be a non-symlink, account-owned private 0700 directory")
	}
	// A customized /etc/skel must not supply SSH access or user-unit overrides.
	if fresh {
		for _, path := range []string{".ssh", ".config/systemd", ".local/share/systemd"} {
			if err := i.absent(filepath.Join(rootless.HomeDir, path)); err != nil {
				return err
			}
		}
	}
	for _, database := range []string{"passwd", "shadow"} {
		data, err := i.readMetadata(ctx, "/etc/"+database)
		if err != nil {
			return err
		}
		matches := 0
		for _, line := range strings.Split(string(data), "\n") {
			fields := strings.Split(line, ":")
			if fields[0] != rootless.AccountName {
				continue
			}
			matches++
			if database == "passwd" {
				shellOK := len(fields) == 7 && (fields[6] == "/usr/bin/nologin" || !fresh && (fields[6] == "/usr/local/lib/urbit-runner/current/urbit-runner" || fields[6] == "/usr/local/lib/urbit-runner/checkpoints/4b/urbit-runner"))
				if len(fields) != 7 || fields[2] != account.Uid || fields[3] != account.Gid || fields[5] != rootless.HomeDir || !shellOK {
					return fmt.Errorf("new local account does not match NSS identity or nologin shell")
				}
			} else if len(fields) != 9 || !(strings.HasPrefix(fields[1], "!") || strings.HasPrefix(fields[1], "*")) {
				return fmt.Errorf("new account password is not locked")
			}
		}
		if matches != 1 {
			return fmt.Errorf("new account has ambiguous or missing local %s record", database)
		}
	}
	return nil
}

func (i *installer) noRootfulAccess(ctx context.Context) error {
	info, err := i.lstat(i.path("/run/docker.sock"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil || info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("cannot establish rootful socket metadata safely")
	}
	// test checks effective access, including ACLs, without connecting to Docker.
	if _, err := i.asAccount(ctx, "/usr/bin/test", "!", "-w", "/run/docker.sock"); err != nil {
		return fmt.Errorf("new account may access the existing rootful Docker socket (or access check failed): %w", err)
	}
	return nil
}

func (i *installer) checkSocket(ctx context.Context, socket string, uid, gid uint32) error {
	if err := i.parents(ctx, filepath.Dir(socket), false); err != nil {
		return err
	}
	info, err := i.lstat(i.path(filepath.Dir(socket)))
	u, g, ok := ownership(info)
	if err != nil || !ok || !info.IsDir() || info.Mode().Perm() != 0700 || u != uid || g != gid {
		return fmt.Errorf("runtime directory must be non-symlink, private 0700 and owned by the new account")
	}
	info, err = i.lstat(i.path(socket))
	u, g, ok = ownership(info)
	if err != nil || !ok || info.Mode()&os.ModeSocket == 0 || info.Mode().Perm()&0007 != 0 || u != uid || g != gid {
		return fmt.Errorf("dedicated socket must be a non-public socket owned by the new account; refusing engine connection")
	}
	return nil
}

func verifyInfo(data []byte) error {
	var info struct {
		ServerVersion   string
		SecurityOptions []string
		CgroupDriver    string
		CgroupVersion   string
		DockerRootDir   string
	}
	if len(data) > maxOutput || json.Unmarshal(data, &info) != nil {
		return fmt.Errorf("engine info is not a bounded JSON object")
	}
	if info.ServerVersion != rootless.EngineVersion || !slices.Contains(info.SecurityOptions, "name=rootless") ||
		info.CgroupDriver != "systemd" || info.CgroupVersion != "2" || info.DockerRootDir != rootless.HomeDir+"/.local/share/urbit-runner/docker" {
		return fmt.Errorf("engine self-report must match version %s, name=rootless, systemd cgroup v2, and the dedicated Docker data root; runtime remains unqualified", rootless.EngineVersion)
	}
	return nil
}

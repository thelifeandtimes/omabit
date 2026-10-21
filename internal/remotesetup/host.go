// Package remotesetup is the local, OS-authorized side of remote onboarding.
// Existing runners are paired by changing access only; installation is explicit.
package remotesetup

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostinstall"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type Request struct {
	Version   int    `json:"version"`
	ID        string `json:"id"`
	PublicKey string `json:"public_key"`
	Role      string `json:"role"`
	Owner     string `json:"owner,omitempty"`
	RunnerID  string `json:"runner_id,omitempty"`
	Revoke    bool   `json:"revoke,omitempty"`
}

type Result struct {
	Version     int    `json:"version"`
	ID          string `json:"id"`
	RunnerID    string `json:"runner_id"`
	Mode        string `json:"mode"`
	Owner       string `json:"owner"`
	Fingerprint string `json:"fingerprint"`
	Role        string `json:"role"`
	Ships       int    `json:"ships"`
	State       string `json:"state"`
}

func (r Request) Validate() error {
	id, err := hex.DecodeString(r.ID)
	if r.Version != 1 || err != nil || len(id) != 16 || hex.EncodeToString(id) != r.ID || (r.RunnerID != "" && !protocol.ValidID(r.RunnerID)) {
		return errors.New("invalid remote setup request identity")
	}
	if r.Revoke && r.RunnerID == "" {
		return errors.New("revocation requires the expected runner identity")
	}
	_, err = access.OwnerAuthorizedKey([]byte(r.PublicKey), r.Role)
	return err
}

// Inspect reads the installed profile and the live Unix RPC. Missing or unhealthy
// installed management is never classified as a fresh/disposable installation.
func Inspect(ctx context.Context, requestedOwner, loginUser string) (ownerhost.Profile, *protocol.Hello, int, error) {
	installed, err := ownerhost.Load()
	if err != nil {
		return ownerhost.Profile{}, nil, 0, err
	}
	if installed != nil {
		actual, err := ownerhost.ForUser(installed.Name)
		if err != nil || actual.UID != installed.UID || actual.GID != installed.GID || actual.Home != installed.Home || requestedOwner != "" && requestedOwner != installed.Name {
			return ownerhost.Profile{}, nil, 0, errors.New("preserve the installed pier owner; this request selects a different or changed account")
		}
		c := client.New(fmt.Sprintf("/run/user/%d/urbit-runner/runner.sock", installed.UID))
		defer c.Close()
		h, err := c.Hello(ctx)
		if err != nil {
			return *installed, nil, 0, fmt.Errorf("installed runner needs attention; it will not be recreated: %w", err)
		}
		s, err := c.Snapshot(ctx)
		if err != nil || s.RunnerID != h.RunnerID || h.Mode != "development-managed" {
			return *installed, nil, 0, errors.New("installed runner inventory/identity could not be verified")
		}
		return *installed, &h, len(s.Instances), nil
	}
	if requestedOwner == "" {
		requestedOwner = loginUser
	}
	p, err := ownerhost.ForUser(requestedOwner)
	if err != nil {
		return p, nil, 0, fmt.Errorf("choose an existing non-root Unix owner in Advanced: %w", err)
	}
	// A missing profile alone is not authority to recreate an old installation.
	for _, path := range []string{access.ShellPath, filepath.Join(p.Home, ".local/share/urbit-runner/managed"), "/var/lib/urbit-runner/managed"} {
		if _, e := os.Lstat(path); !errors.Is(e, os.ErrNotExist) {
			return p, nil, 0, errors.New("existing management resources lack an owner profile; preserve them for migration instead of fresh setup")
		}
	}
	return p, nil, 0, nil
}

type Confirm func(string) (bool, error)

// Bootstrap runs within the authenticated administrator's terminal. Passwords
// are handled by OpenSSH/sudo, never by JSON progress files or QML.
func Bootstrap(ctx context.Context, request Request, loginUser, sshConnection string, confirm Confirm, progress func(string)) (Result, error) {
	if os.Geteuid() != 0 || runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return Result{}, errors.New("remote setup requires OS authorization on Linux x86-64")
	}
	if err := request.Validate(); err != nil {
		return Result{}, err
	}
	if confirm == nil {
		return Result{}, errors.New("interactive setup approval is required")
	}
	dir, err := journalDirectory()
	if err != nil {
		return Result{}, err
	}
	journalPath := filepath.Join(dir, request.ID+".json")
	lock, err := os.OpenFile(filepath.Join(dir, request.ID+".lock"), os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return Result{}, err
	}
	defer lock.Close()
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return Result{}, errors.New("this remote setup is already running")
	}
	journal, err := readInstallJournal(journalPath)
	if err != nil {
		return Result{}, err
	}
	line, _ := access.OwnerAuthorizedKey([]byte(request.PublicKey), request.Role)
	grant, _ := access.ParseLine(line)
	helperDigest, err := Digest("/proc/self/exe")
	if err != nil {
		return Result{}, err
	}
	if journal != nil && !matchesJournal(journal, request, grant.Fingerprint, helperDigest) {
		return Result{}, errors.New("remote setup journal belongs to a different key, owner, runner or installer generation")
	}
	progress("Inspecting the existing owner and runner")
	p, h, ships, err := Inspect(ctx, request.Owner, loginUser)
	resume := err != nil && journal != nil && journal.Stage == "installing" && journal.Owner == p
	if err != nil && request.Revoke && p.Validate() == nil && request.RunnerID != "" {
		// AuthorizeDesktop independently requires its root-issued grant receipt
		// when management is offline. Revocation does not start the runner.
		h = &protocol.Hello{RunnerID: request.RunnerID, Mode: "development-managed"}
		err = nil
	}
	if err != nil && !resume {
		return Result{}, err
	}
	if request.RunnerID != "" && (h == nil || h.RunnerID != request.RunnerID) {
		return Result{}, errors.New("the previously selected runner identity changed; existing enrollment was preserved")
	}
	if journal != nil && (journal.Owner != p || journal.RunnerID != "" && (h == nil || journal.RunnerID != h.RunnerID)) {
		return Result{}, errors.New("the journaled owner or runner identity changed")
	}
	if !request.Revoke {
		if err = hostinstall.CheckDesktopSSH(ctx, p, sshConnection); err != nil {
			return Result{}, err
		}
	}
	if h == nil {
		if request.Revoke {
			return Result{}, errors.New("cannot revoke a missing runner")
		}
		if request.Role != "operator" {
			return Result{}, errors.New("fresh host installation requires operator access")
		}
		plan, err := PlanDependencies(ctx)
		if err != nil {
			return Result{}, err
		}
		message := "No owner-user runner is installed. Install Omabit for " + p.Name + " with pier root " + p.Piers + "?"
		if resume {
			message = "Continue this interrupted Omabit installation for " + p.Name + "? Its retained installer journals and data will be preserved."
		}
		if len(plan.Packages) > 0 {
			message += "\nMissing native packages: " + strings.Join(plan.Packages, ", ")
		}
		approved, err := confirm(message)
		if err != nil || !approved {
			return Result{}, errors.New("remote installation was not approved")
		}
		journal = &installJournal{Version: 1, ID: request.ID, Fingerprint: grant.Fingerprint, Role: request.Role, Owner: p, HelperSHA256: helperDigest, Stage: "installing"}
		if err = saveInstallJournal(journalPath, *journal); err != nil {
			return Result{}, err
		}
		if err := InstallDependencies(ctx, plan, progress); err != nil {
			return Result{}, err
		}
		progress("Preparing the content-pinned rootless helper")
		bundle, err := prepareOwnerBundle(ctx, p)
		if err != nil {
			return Result{}, err
		}
		progress("Installing the owner's management services")
		host, err := hostinstall.InstallOwnerHost(ctx, bundle, []byte(request.PublicKey), p.Name, progress)
		if err != nil {
			return Result{}, fmt.Errorf("installation incomplete; retain the same owner and retry: %w", err)
		}
		request.RunnerID = host.RunnerID
		journal.RunnerID = host.RunnerID
		if err = saveInstallJournal(journalPath, *journal); err != nil {
			return Result{}, err
		}
		p, h, ships, err = Inspect(ctx, p.Name, loginUser)
		if err != nil {
			return Result{}, err
		}
		if h.RunnerID != host.RunnerID {
			return Result{}, errors.New("runner identity changed after installation; preserve it for review")
		}
		journal.Stage = "installed"
		journal.RunnerID = h.RunnerID
		if err = saveInstallJournal(journalPath, *journal); err != nil {
			return Result{}, err
		}
	} else {
		if journal != nil && journal.Stage == "installing" {
			if journal.Owner != p {
				return Result{}, errors.New("owner changed after interrupted installation")
			}
			journal.Stage = "installed"
			journal.RunnerID = h.RunnerID
			if err = saveInstallJournal(journalPath, *journal); err != nil {
				return Result{}, err
			}
		}
		action := "Authorize this desktop to manage"
		if request.Role == "observer" {
			action = "Authorize this desktop to observe"
		}
		if request.Revoke {
			action = "Revoke this desktop's access to"
		}
		approved, err := confirm(fmt.Sprintf("Found runner %s, owned by %s, with %d ship(s).\n%s this runner? Existing ships and management will keep running.", h.RunnerID, p.Name, ships, action))
		if err != nil || !approved {
			return Result{}, errors.New("desktop access was not approved")
		}
	}
	progress("Configuring this desktop's restricted management key on the existing SSH endpoint")
	authorized, err := hostinstall.AuthorizeDesktop(ctx, []byte(request.PublicKey), hostinstall.DesktopAccessOptions{RunnerID: h.RunnerID, Role: request.Role, SSHConnection: sshConnection, Revoke: request.Revoke})
	if err != nil {
		return Result{}, err
	}
	return Result{Version: 1, ID: request.ID, RunnerID: authorized.RunnerID, Mode: authorized.Mode, Owner: authorized.Owner, Fingerprint: authorized.Fingerprint, Role: authorized.Role, Ships: ships, State: authorized.State}, nil
}

func prepareOwnerBundle(ctx context.Context, p ownerhost.Profile) (string, error) {
	stage, err := os.MkdirTemp("", "omabit-remote-rootless-")
	if err != nil {
		return "", err
	}
	// Fetches execute as the selected owner, exactly as local onboarding does.
	// Keep the bounded preparation directory for inspection rather than having
	// root recursively remove a tree writable by another UID.
	input, err := os.Open("/proc/self/exe")
	if err != nil {
		return "", err
	}
	defer input.Close()
	path := filepath.Join(stage, "urbit-runner")
	output, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0755)
	if err != nil {
		return "", err
	}
	n, copyErr := io.Copy(output, io.LimitReader(input, (128<<20)+1))
	err = errors.Join(copyErr, output.Chmod(0755), output.Sync(), output.Close())
	if err != nil || n > 128<<20 {
		return "", errors.New("cannot prepare owner-side helper")
	}
	if err = os.Chown(stage, int(p.UID), int(p.GID)); err != nil {
		return "", err
	}
	bundle := filepath.Join(stage, "rootless")
	runuser, environment := fixedExecutable("runuser"), fixedExecutable("env")
	if runuser == "" || environment == "" {
		return "", errors.New("native runuser/env are required for owner-side preparation")
	}
	if _, err = hostCommand(ctx, runuser, "-u", p.Name, "--", environment, "-i", "HOME="+p.Home, "PATH=/usr/bin:/usr/sbin:/bin:/sbin", "LANG=C", path, "prepare-rootless", "--output", bundle); err != nil {
		return "", fmt.Errorf("owner-side pinned helper preparation failed: %w", err)
	}
	return bundle, nil
}

type DependencyPlan struct {
	Provider string   `json:"provider"`
	Packages []string `json:"packages"`
}

var nativeDependencies = []struct{ executable, pacman, deb string }{
	{"docker", "docker", "docker.io"}, {"dockerd", "docker", "docker.io"}, {"containerd", "containerd", "containerd"},
	{"runc", "runc", "runc"}, {"rootlesskit", "rootlesskit", "rootlesskit"}, {"slirp4netns", "slirp4netns", "slirp4netns"},
	{"newuidmap", "shadow", "uidmap"}, {"newgidmap", "shadow", "uidmap"}, {"iptables", "iptables", "iptables"},
	{"sshd", "openssh", "openssh-server"}, {"ssh-keygen", "openssh", "openssh-client"}, {"gpg", "gnupg", "gnupg"},
	{"bsdtar", "libarchive", "libarchive-tools"}, {"dbus-daemon", "dbus", "dbus-user-session"},
}

func hostCommand(ctx context.Context, executable string, args ...string) ([]byte, error) {
	return runHostCommand(ctx, false, executable, args...)
}

func runHostCommand(ctx context.Context, visible bool, executable string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Env = []string{"PATH=/usr/bin:/usr/sbin:/bin:/sbin", "HOME=/root", "LANG=C", "LC_ALL=C", "DEBIAN_FRONTEND=noninteractive"}
	var output commandOutput
	var writer io.Writer = &output
	if visible {
		writer = io.MultiWriter(os.Stdout, &output)
	}
	cmd.Stdout = writer
	cmd.Stderr = writer
	err := cmd.Run()
	return output.Bytes(), err
}

type commandOutput struct{ bytes.Buffer }

func (b *commandOutput) Write(p []byte) (int, error) {
	n := len(p)
	if n > 65536 {
		p = p[n-65536:]
	}
	if b.Len()+len(p) > 65536 {
		b.Next(b.Len() + len(p) - 65536)
	}
	_, err := b.Buffer.Write(p)
	return n, err
}

func fixedExecutable(name string) string {
	for _, directory := range []string{"/usr/bin", "/usr/sbin", "/bin", "/sbin"} {
		path := filepath.Join(directory, name)
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			continue
		}
		if st, err := os.Stat(resolved); err == nil && st.Mode().IsRegular() && st.Mode().Perm()&0111 != 0 && st.Mode().Perm()&0022 == 0 {
			owner, ok := st.Sys().(*syscall.Stat_t)
			if !ok || owner.Uid != 0 {
				continue
			}
			trusted := true
			for parent := filepath.Dir(resolved); ; parent = filepath.Dir(parent) {
				info, e := os.Lstat(parent)
				if e != nil {
					trusted = false
					break
				}
				who, ok := info.Sys().(*syscall.Stat_t)
				if !ok || who.Uid != 0 || !info.IsDir() || info.Mode().Perm()&0022 != 0 {
					trusted = false
					break
				}
				if parent == "/" {
					break
				}
			}
			if trusted {
				return resolved
			}
		}
	}
	return ""
}

func PlanDependencies(ctx context.Context) (DependencyPlan, error) {
	if _, err := os.Stat("/run/systemd/system"); err != nil {
		return DependencyPlan{}, errors.New("this host must be booted with systemd")
	}
	if _, err := os.Stat("/sys/fs/cgroup/cgroup.controllers"); err != nil {
		return DependencyPlan{}, errors.New("this host requires cgroup v2")
	}
	plan := DependencyPlan{Packages: []string{}}
	if fixedExecutable("dockerd") != "" {
		report := hostcompat.InspectCurrent()
		if hostcompat.EngineVersionStatus(report.InstalledEngineVersion) != "compatible" {
			return plan, errors.New("the installed engine lacks a supported native version; preserve it and review host prerequisites before setup")
		}
	}
	if fixedExecutable("pacman") != "" {
		plan.Provider = "pacman"
	} else if fixedExecutable("apt-get") != "" {
		plan.Provider = "deb"
	} else {
		return plan, errors.New("automatic dependencies support Pacman and Debian/Ubuntu APT hosts")
	}
	seen := map[string]bool{}
	for _, item := range nativeDependencies {
		if fixedExecutable(item.executable) != "" {
			continue
		}
		name := item.pacman
		if plan.Provider == "deb" {
			name = item.deb
		}
		if !seen[name] {
			plan.Packages = append(plan.Packages, name)
			seen[name] = true
		}
	}
	if plan.Provider == "deb" && !seen["dbus-user-session"] {
		if _, err := os.Stat("/usr/lib/systemd/user/dbus.service"); errors.Is(err, os.ErrNotExist) {
			plan.Packages = append(plan.Packages, "dbus-user-session")
		}
	}
	return plan, ctx.Err()
}

func InstallDependencies(ctx context.Context, plan DependencyPlan, progress func(string)) error {
	if len(plan.Packages) == 0 {
		return nil
	}
	allowed := map[string]bool{}
	for _, item := range nativeDependencies {
		if plan.Provider == "pacman" {
			allowed[item.pacman] = true
		} else {
			allowed[item.deb] = true
		}
	}
	for _, name := range plan.Packages {
		if !allowed[name] {
			return errors.New("unexpected dependency package")
		}
	}
	progress("Installing the approved missing native dependencies")
	var program string
	var args []string
	switch plan.Provider {
	case "pacman":
		program = fixedExecutable("pacman")
		if err := checkEngineCandidate(ctx, plan); err != nil {
			return err
		}
		args = append([]string{"-S", "--needed", "--noconfirm", "--"}, plan.Packages...)
	case "deb":
		program = fixedExecutable("apt-get")
		if _, err := runHostCommand(ctx, true, program, "update"); err != nil {
			return fmt.Errorf("refresh configured APT metadata failed; review the host's configured repositories and resume: %w", err)
		}
		if err := checkEngineCandidate(ctx, plan); err != nil {
			return err
		}
		args = append([]string{"install", "--yes", "--no-install-recommends", "--"}, plan.Packages...)
	default:
		return errors.New("unsupported native dependency provider")
	}
	if _, err := runHostCommand(ctx, true, program, args...); err != nil {
		return fmt.Errorf("native dependency installation did not complete; inspect the native package manager and resume: %w", err)
	}
	// Installation is followed by the existing signature/payload and engine-policy
	// gates; package metadata alone never authorizes runtime execution.
	report := hostcompat.InspectCurrent()
	if hostcompat.EngineVersionStatus(report.InstalledEngineVersion) != "compatible" {
		return errors.New("configured packages do not provide a supported Docker version; use a supported native repository and resume setup")
	}
	return nil
}

func checkEngineCandidate(ctx context.Context, plan DependencyPlan) error {
	for _, name := range plan.Packages {
		if name != "docker" && name != "docker.io" {
			continue
		}
		var data []byte
		var err error
		if plan.Provider == "pacman" {
			data, err = hostCommand(ctx, fixedExecutable("pacman"), "-Si", "--", name)
		} else {
			data, err = hostCommand(ctx, fixedExecutable("apt-cache"), "policy", name)
		}
		if err != nil {
			return errors.New("cannot inspect the configured native Docker candidate")
		}
		version := ""
		for _, line := range strings.Split(string(data), "\n") {
			key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
			if ok && (strings.TrimSpace(key) == "Version" || strings.TrimSpace(key) == "Candidate") {
				version = strings.TrimSpace(value)
				break
			}
		}
		if hostcompat.EnginePackageVersionStatus(version) != "compatible" {
			return fmt.Errorf("configured Docker candidate %q is outside the supported engine policy; choose a supported native repository before setup", version)
		}
	}
	return nil
}

// Digest verifies a staged executable before it is offered for root execution.
func Digest(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, 128<<20+1))
	if err != nil || n > 128<<20 {
		return "", errors.New("invalid staged helper")
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func OwnerUID(name string) (uint32, error) { p, err := ownerhost.ForUser(name); return p.UID, err }

// Bound each privileged setup attempt; a subsequent attempt must re-inspect.
func Deadline(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, 20*time.Minute)
}

func UserSocket(p ownerhost.Profile) string {
	return "/run/user/" + strconv.FormatUint(uint64(p.UID), 10) + "/urbit-runner/runner.sock"
}

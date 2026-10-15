// Package onboarding owns local desktop setup policy. Privileged host operations
// remain in urbit-runner; enrollment and application credentials stay per-user.
package onboarding

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostinstall"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostsetup"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

type Status struct {
	ExecutionModel string                 `json:"execution_model"`
	PierRoot       string                 `json:"pier_root"`
	Ready          bool                   `json:"ready"`
	CanSetup       bool                   `json:"can_setup"`
	HostPresent    bool                   `json:"host_present"`
	Checks         []hostsetup.Check      `json:"checks"`
	Connection     *collection.Connection `json:"connection,omitempty"`
	BrowserBase    string                 `json:"browser_base,omitempty"`
	Hint           string                 `json:"hint,omitempty"`
}

type Event struct {
	Step    string  `json:"step"`
	Status  string  `json:"status"`
	Message string  `json:"message,omitempty"`
	Result  *Status `json:"result,omitempty"`
}

type receipt struct {
	RunnerID      string `json:"runner_id"`
	Connection    string `json:"connection"`
	HostPublicKey string `json:"host_public_key"`
}

type dependencies struct {
	inspect func() hostsetup.Report
	tool    func(string) bool
	prepare func(context.Context, string) (rootless.Manifest, error)
	key     func(context.Context, string) ([]byte, error)
	elevate func(context.Context, string, string, string, func(Event)) (hostinstall.HostResult, error)
	probe   func(context.Context, collection.Connection) error
	connect func(context.Context, string, client.SSHConfig, hostinstall.HostResult) (collection.Connection, error)
}

func defaults() dependencies {
	return dependencies{inspect: hostsetup.InspectCurrentOwner, tool: executable, prepare: rootless.Prepare, key: desktopKey, elevate: installHost, probe: probeConnection, connect: connectLocal}
}

func Directory() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "omabit", "setup"), nil
}

func Inspect(ctx context.Context, config string) (Status, error) {
	dir, err := Directory()
	if err != nil {
		return Status{}, err
	}
	return inspect(ctx, config, dir, defaults())
}

func inspect(ctx context.Context, config, dir string, d dependencies) (Status, error) {
	s := Status{CanSetup: true, Checks: []hostsetup.Check{}, ExecutionModel: "owner-user", PierRoot: ownerhost.DefaultPiers}
	for _, c := range d.inspect().Checks {
		if c.ID == "account" {
			s.HostPresent = c.Status == "pass"
		}
		required := c.ID == "owner-account" || c.ID == "platform" || c.ID == "systemd" || c.ID == "cgroup-v2" || c.ID == "userns-limit" || c.ID == "docker-version" || strings.HasPrefix(c.ID, "binary-")
		if (c.ID == "subuid" || c.ID == "subgid") && c.Status != "missing" {
			required = true
		}
		if c.ID == "userns-clone" && c.Status == "blocked" {
			required = true
		}
		if c.ID == "account" && c.Status == "blocked" {
			required = true
		}
		if required {
			s.Checks = append(s.Checks, c)
			if c.Status != "pass" {
				s.CanSetup = false
			}
		}
	}
	for _, name := range []string{"ssh", "ssh-keygen", "sshd", "pkexec"} {
		c := hostsetup.Check{ID: "binary-" + name, Status: "pass", Detail: name + " is available"}
		if !d.tool("/usr/bin/" + name) {
			c.Status = "missing"
			c.Detail = "Install " + name + " before local setup"
			s.CanSetup = false
		}
		s.Checks = append(s.Checks, c)
	}
	if !s.CanSetup {
		return s, nil
	}
	data, err := readPrivate(filepath.Join(dir, "ready.json"), 16384)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		s.Hint = err.Error()
		return s, nil
	}
	var saved receipt
	if json.Unmarshal(data, &saved) != nil || saved.RunnerID == "" || saved.Connection == "" {
		s.Hint = "Local setup state is invalid; existing enrollment was preserved."
		return s, nil
	}
	r, err := collection.Open(config)
	if err != nil {
		return s, err
	}
	c, err := r.Get(ctx, saved.Connection)
	if err != nil || c.RunnerID != saved.RunnerID {
		s.Hint = "The local enrollment changed. Check Manage runners before running setup again."
		return s, nil
	}
	check, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := d.probe(check, c); err != nil {
		s.Hint = "Local services or SSH access are unavailable. Setup can verify the installation again."
		return s, nil
	}
	s.Ready = true
	s.Connection = &c
	s.BrowserBase = "http://127.0.0.1"
	return s, nil
}

func Setup(ctx context.Context, config string, emit func(Event)) (Status, error) {
	if os.Geteuid() == 0 {
		return Status{}, errors.New("Run setup as your desktop user; only host installation needs authorization.")
	}
	dir, err := Directory()
	if err != nil {
		return Status{}, err
	}
	exe, err := os.Executable()
	if err != nil {
		return Status{}, err
	}
	runner := filepath.Join(filepath.Dir(exe), "urbit-runner")
	return setup(ctx, config, dir, runner, defaults(), emit)
}

func setup(ctx context.Context, config, dir, runner string, d dependencies, emit func(Event)) (Status, error) {
	step := func(name, message string) {
		if emit != nil {
			emit(Event{Step: name, Status: "running", Message: message})
		}
	}
	step("check", "Checking this computer")
	s, err := inspect(ctx, config, dir, d)
	if err != nil {
		return s, err
	}
	if s.Ready {
		if emit != nil {
			emit(Event{Step: "ready", Status: "complete", Message: "This computer is already connected.", Result: &s})
		}
		return s, nil
	}
	if !s.CanSetup {
		return s, errors.New("Install the listed prerequisites first. On Omarchy, use its package manager for docker, rootlesskit, slirp4netns and openssh; the supported Docker version is " + rootless.EngineVersion + ".")
	}
	if !executable(runner) {
		return s, errors.New("The bundled urbit-runner is missing. Rebuild and reinstall the Omabit panel.")
	}
	if err := privateDirectory(dir); err != nil {
		return s, err
	}
	unlock, err := lockSetup(dir)
	if err != nil {
		return s, err
	}
	defer unlock()
	step("key", "Preparing private desktop access")
	pub, err := d.key(ctx, filepath.Join(dir, "id_ed25519"))
	if err != nil {
		return s, err
	}
	line, err := access.AuthorizedKey(pub, "operator")
	if err != nil {
		return s, err
	}
	grant, err := access.ParseLine(line)
	if err != nil {
		return s, err
	}
	publicPath := filepath.Join(dir, "operator.pub")
	if err := writePrivate(publicPath, pub); err != nil {
		return s, err
	}
	step("download", "Downloading and verifying the rootless host helper")
	stage, err := os.MkdirTemp(dir, "bundle-")
	if err != nil {
		return s, err
	}
	defer os.RemoveAll(stage) // only this invocation's private staging directory
	bundle := filepath.Join(stage, "rootless")
	if _, err := d.prepare(ctx, bundle); err != nil {
		return s, fmt.Errorf("prepare host helper: %w", err)
	}
	step("authorize", "Authorize host setup in the system password prompt")
	host, err := d.elevate(ctx, runner, bundle, publicPath, emit)
	if err != nil {
		return s, err
	}
	if !host.Ready || host.UID == 0 || host.Mode != "development-managed" || !strings.HasPrefix(host.RunnerID, "runner-") || host.SSHPort != hostinstall.SSHPort || host.OperatorFingerprint != grant.Fingerprint {
		return s, errors.New("Host setup returned unexpected connection details; enrollment was preserved.")
	}
	if _, err := access.AuthorizedKey([]byte(host.HostPublicKey), "operator"); err != nil {
		return s, errors.New("Host setup returned an invalid SSH host identity.")
	}
	if data, err := readPrivate(filepath.Join(dir, "ready.json"), 16384); err == nil {
		var previous receipt
		if json.Unmarshal(data, &previous) != nil || previous.RunnerID != host.RunnerID || previous.HostPublicKey != host.HostPublicKey {
			return s, errors.New("Local runner or SSH host identity changed. Review the existing connection before replacing it.")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return s, err
	}
	step("connect", "Verifying your operator connection")
	if host.Owner != "" && host.UID != uint32(os.Geteuid()) {
		return s, errors.New("host setup selected a different pier owner")
	}
	ssh, err := writeSSH(dir, host.HostPublicKey, host.Owner)
	if err != nil {
		return s, err
	}
	connection, err := d.connect(ctx, config, ssh, host)
	if err != nil {
		return s, fmt.Errorf("connect local runner: %w", err)
	}
	if connection.RunnerID != host.RunnerID {
		return s, errors.New("SSH connection did not match the authorized runner identity")
	}
	saved := receipt{RunnerID: host.RunnerID, Connection: connection.Name, HostPublicKey: host.HostPublicKey}
	data, err := json.Marshal(saved)
	if err != nil {
		return s, err
	}
	if err := writePrivate(filepath.Join(dir, "ready.json"), data); err != nil {
		return s, err
	}
	s.Ready = true
	s.HostPresent = true
	s.Connection = &connection
	s.BrowserBase = "http://127.0.0.1"
	s.Hint = ""
	if emit != nil {
		emit(Event{Step: "ready", Status: "complete", Message: "This computer is ready. Add a comet to get started.", Result: &s})
	}
	return s, nil
}

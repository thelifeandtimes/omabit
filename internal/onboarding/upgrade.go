package onboarding

import (
	"bytes"
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostinstall"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

type UpgradeStatus struct {
	Version         int    `json:"version"`
	RunnerID        string `json:"runner_id"`
	TargetSHA256    string `json:"target_sha256"`
	InstalledSHA256 string `json:"installed_sha256"`
	TargetBuild     string `json:"target_build"`
	InstalledBuild  string `json:"installed_build"`
	Available       bool   `json:"available"`
	Pending         bool   `json:"pending"`
	Message         string `json:"message"`
}
type UpgradeEvent struct {
	Step    string         `json:"step"`
	Status  string         `json:"status"`
	Message string         `json:"message,omitempty"`
	Result  *UpgradeStatus `json:"result,omitempty"`
}
type upgradeBinding struct {
	SSHConfig        string            `json:"ssh_config"`
	Owner            ownerhost.Profile `json:"owner"`
	RunnerID         string            `json:"runner_id"`
	EnrollmentSHA256 string            `json:"enrollment_sha256"`
	AccessSHA256     string            `json:"access_sha256"`
	Fingerprint      string            `json:"fingerprint"`
	HostKey          string            `json:"host_key"`
}
type upgradeAttempt struct {
	PreviousSHA256 string         `json:"previous_sha256"`
	Version        int            `json:"version"`
	Binding        upgradeBinding `json:"binding"`
	TargetSHA256   string         `json:"target_sha256"`
	Complete       bool           `json:"complete"`
}
type upgradeDependencies struct {
	binding      func(context.Context, string, string) (upgradeBinding, error)
	probe        func(context.Context, upgradeBinding) error
	binary       func(string) ([]byte, string, error)
	installed    string
	prepare      func(context.Context, string) (rootless.Manifest, error)
	verifyBundle func(string) error
	elevate      func(context.Context, string, string, string, string, func(Event)) (hostinstall.HostResult, error)
}

func upgradeDefaults() upgradeDependencies {
	return upgradeDependencies{binding: loadUpgradeBinding, probe: probeUpgrade, binary: upgradeBinary,
		installed: "/usr/local/lib/urbit-runner/current/urbit-runner", prepare: rootless.Prepare,
		verifyBundle: func(path string) error { _, err := rootless.VerifiedFiles(path); return err }, elevate: installHostExpected}
}
func upgradeDigest(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }
func decodeUpgrade(data []byte, value any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		return err
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return errors.New("invalid trailing upgrade state")
	}
	return nil
}

// Read build metadata; never execute the installed or candidate binary for status.
func upgradeBinary(path string) ([]byte, string, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, "", err
	}
	if !st.Mode().IsRegular() || st.Size() > 128<<20 || st.Mode().Perm()&0022 != 0 {
		return nil, "", errors.New("runner executable is not a bounded, protected regular file")
	}
	data, err := io.ReadAll(io.LimitReader(f, (128<<20)+1))
	if err != nil || len(data) > 128<<20 {
		return nil, "", errors.New("cannot read runner executable")
	}
	info, err := buildinfo.Read(bytes.NewReader(data))
	if err != nil || info.Path != "git.thechristophgroup.org/thelifeandtimes/omabit/backend/cmd/urbit-runner" {
		return nil, "", errors.New("candidate is not an Omabit runner build")
	}
	label := "local build"
	dirty := false
	for _, v := range info.Settings {
		if v.Key == "vcs.revision" {
			label = v.Value
		}
		if v.Key == "vcs.modified" && v.Value == "true" {
			dirty = true
		}
	}
	if len(label) > 12 {
		label = label[:12]
	}
	if dirty {
		label += " (development)"
	}
	return data, label, nil
}
func loadUpgradeBinding(ctx context.Context, config, dir string) (upgradeBinding, error) {
	var b upgradeBinding
	p, err := ownerhost.Current()
	if err != nil || p == nil {
		return b, errors.New("no existing owner installation for this desktop user")
	}
	b.Owner = *p
	data, err := readPrivate(filepath.Join(dir, "ready.json"), 16384)
	if err != nil {
		return b, errors.New("the existing local setup receipt is unavailable; preserve the installation and review its connection")
	}
	var saved receipt
	if decodeUpgrade(data, &saved) != nil || saved.RunnerID == "" || saved.HostPublicKey == "" {
		return b, errors.New("invalid local setup receipt")
	}
	r, err := collection.Open(config)
	if err != nil {
		return b, err
	}
	c, err := r.Get(ctx, saved.Connection)
	if err != nil || c.RunnerID != saved.RunnerID || c.Mode != "development-managed" || c.Transport != "ssh" || c.SSH == nil || c.SSH.Host != "omabit-local" || c.SSH.ConfigFile != filepath.Join(dir, "ssh_config") {
		return b, errors.New("local enrollment changed; review it before upgrading services")
	}
	encoded, _ := json.Marshal(c)
	b.EnrollmentSHA256 = upgradeDigest(encoded)
	b.RunnerID = c.RunnerID
	b.HostKey = saved.HostPublicKey
	b.SSHConfig = c.SSH.ConfigFile
	var accessBytes []byte
	for _, name := range []string{"ready.json", "operator.pub", "ssh_config", "known_hosts", "id_ed25519"} {
		contents, err := readPrivate(filepath.Join(dir, name), 16384)
		if err != nil {
			return b, errors.New("existing desktop access files are unavailable; no keys were regenerated")
		}
		accessBytes = append(accessBytes, []byte(name+"\x00")...)
		accessBytes = append(accessBytes, contents...)
		if name == "operator.pub" {
			line, e := access.AuthorizedKey(contents, "operator")
			if e != nil {
				return b, e
			}
			g, e := access.ParseLine(line)
			if e != nil {
				return b, e
			}
			b.Fingerprint = g.Fingerprint
		}
	}
	b.AccessSHA256 = upgradeDigest(accessBytes)
	// Verify the selected local socket separately from configurable SSH aliases.
	return b, nil
}
func probeUpgrade(ctx context.Context, b upgradeBinding) error {
	c := collection.Connection{Transport: "unix", Socket: fmt.Sprintf("/run/user/%d/urbit-runner/runner.sock", b.Owner.UID), RunnerID: b.RunnerID, Mode: "development-managed"}
	cli := c.Client(true)
	defer cli.Close()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	h, err := cli.Hello(ctx)
	if err != nil || h.RunnerID != b.RunnerID {
		return errors.New("the local runner is unavailable or its identity changed")
	}
	ssh := collection.Connection{Transport: "ssh", SSH: &client.SSHConfig{Host: "omabit-local", ConfigFile: b.SSHConfig}, RunnerID: b.RunnerID, Mode: "development-managed"}.Client(true)
	defer ssh.Close()
	grant, err := ssh.Hello(ctx)
	if err != nil || grant.AccessRole != "operator" || grant.Actor != "ssh-key:"+b.Fingerprint {
		return errors.New("the original desktop operator access could not be verified")
	}
	snapshot, err := cli.Snapshot(ctx)
	if err != nil {
		return err
	}
	for _, ship := range snapshot.Instances {
		if ship.ActiveOperationID != "" {
			return errors.New("finish pending ship actions before upgrading services")
		}
	}
	return nil
}
func readUpgradeAttempt(dir string) (*upgradeAttempt, error) {
	data, err := readPrivate(filepath.Join(dir, "service-upgrade.json"), 16384)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var a upgradeAttempt
	if decodeUpgrade(data, &a) != nil || a.Version != 1 || len(a.PreviousSHA256) != 64 || len(a.TargetSHA256) != 64 || a.Binding.RunnerID == "" {
		return nil, errors.New("saved service update needs review; preserve its files")
	}
	return &a, nil
}
func writeUpgradeAttempt(dir string, a *upgradeAttempt) error {
	data, err := json.Marshal(a)
	if err != nil {
		return err
	}
	return writePrivate(filepath.Join(dir, "service-upgrade.json"), data)
}
func serviceUpgradeStatus(ctx context.Context, config, dir, runner string, d upgradeDependencies) (UpgradeStatus, error) {
	s := UpgradeStatus{Version: 1}
	b, err := d.binding(ctx, config, dir)
	if err != nil {
		return s, err
	}
	s.RunnerID = b.RunnerID
	a, err := readUpgradeAttempt(dir)
	if err != nil {
		return s, err
	}
	if a != nil && !a.Complete {
		if a.Binding != b {
			return s, errors.New("owner, enrollment or access changed since this update began; preserve the original attempt")
		}
		runner = filepath.Join(dir, "service-upgrade", "urbit-runner")
		s.Pending = true
	}
	data, label, err := d.binary(runner)
	if err != nil {
		return s, err
	}
	s.TargetSHA256 = upgradeDigest(data)
	s.TargetBuild = label
	if s.Pending && a.TargetSHA256 != s.TargetSHA256 {
		return s, errors.New("retained update executable changed; retry refused")
	}
	installed, label, err := d.binary(d.installed)
	if err != nil {
		return s, err
	}
	s.InstalledSHA256 = upgradeDigest(installed)
	s.InstalledBuild = label
	if s.Pending && s.InstalledSHA256 != a.PreviousSHA256 && s.InstalledSHA256 != a.TargetSHA256 {
		return s, errors.New("the host was changed by another update; preserve this attempt and review the installed generation")
	}
	s.Available = s.Pending || s.TargetSHA256 != s.InstalledSHA256
	s.Message = "Local services match this desktop bundle."
	if s.Pending {
		s.Message = "An update needs to finish. Retry uses its original retained installer."
	} else if s.Available {
		s.Message = "Update the local runner to match this desktop bundle. OS authorization is required."
	}
	return s, nil
}
func upgradePaths() (string, string, error) {
	if os.Geteuid() == 0 {
		return "", "", errors.New("run service updates as your desktop user; the host step requests OS authorization")
	}
	dir, err := Directory()
	if err != nil {
		return "", "", err
	}
	exe, err := os.Executable()
	if err != nil {
		return "", "", err
	}
	return dir, filepath.Join(filepath.Dir(exe), "urbit-runner"), nil
}
func InspectUpgrade(ctx context.Context, config string) (UpgradeStatus, error) {
	dir, runner, err := upgradePaths()
	if err != nil {
		return UpgradeStatus{}, err
	}
	return serviceUpgradeStatus(ctx, config, dir, runner, upgradeDefaults())
}
func UpgradeServices(ctx context.Context, config, expected, target string, emit func(UpgradeEvent)) (UpgradeStatus, error) {
	dir, runner, err := upgradePaths()
	if err != nil {
		return UpgradeStatus{}, err
	}
	return upgradeServices(ctx, config, dir, runner, expected, target, upgradeDefaults(), emit)
}
func upgradeServices(ctx context.Context, config, dir, runner, expected, target string, d upgradeDependencies, emit func(UpgradeEvent)) (UpgradeStatus, error) {
	var s UpgradeStatus
	if expected == "" || target == "" {
		return s, errors.New("check service updates first and supply the displayed runner and target digest")
	}
	if err := privateDirectory(dir); err != nil {
		return s, err
	}
	unlock, err := lockSetup(dir)
	if err != nil {
		return s, err
	}
	defer unlock()
	s, err = serviceUpgradeStatus(ctx, config, dir, runner, d)
	if err != nil {
		return s, err
	}
	if s.RunnerID != expected || s.TargetSHA256 != target {
		return s, errors.New("service update selection changed; check again before approving")
	}
	step := func(name, message string) {
		if emit != nil {
			emit(UpgradeEvent{Step: name, Status: "running", Message: message})
		}
	}
	b, err := d.binding(ctx, config, dir)
	if err != nil {
		return s, err
	}
	work := filepath.Join(dir, "service-upgrade")
	a, err := readUpgradeAttempt(dir)
	if err != nil {
		return s, err
	}
	if !s.Pending {
		if err = d.probe(ctx, b); err != nil {
			return s, err
		}
		if !s.Available {
			if emit != nil {
				emit(UpgradeEvent{Step: "upgrade", Status: "complete", Result: &s})
			}
			return s, nil
		}
		// No installer has been dispatched for an unrecorded staging directory.
		// Only this bounded staging location is retired; piers and host journals are unrelated.
		if err = os.RemoveAll(work); err != nil {
			return s, err
		}
		if err = privateDirectory(work); err != nil {
			return s, err
		}
		data, _, e := d.binary(runner)
		if e != nil {
			return s, e
		}
		if upgradeDigest(data) != target {
			return s, errors.New("candidate changed during preparation; check again")
		}
		f, e := os.OpenFile(filepath.Join(work, "urbit-runner"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
		if e != nil {
			return s, e
		}
		_, e = f.Write(data)
		e = errors.Join(e, f.Sync(), f.Close())
		if e != nil {
			return s, e
		}
		public, e := readPrivate(filepath.Join(dir, "operator.pub"), 8192)
		if e != nil {
			return s, e
		}
		if e = writePrivate(filepath.Join(work, "operator.pub"), public); e != nil {
			return s, e
		}
		step("prepare", "Downloading and verifying the matching service dependencies")
		if _, err = d.prepare(ctx, filepath.Join(work, "rootless")); err != nil {
			return s, err
		}
		a = &upgradeAttempt{Version: 1, Binding: b, TargetSHA256: target, PreviousSHA256: s.InstalledSHA256}
		if err = writeUpgradeAttempt(dir, a); err != nil {
			return s, err
		}
	}
	if err = privateDirectory(work); err != nil {
		return s, err
	}
	if err = d.verifyBundle(filepath.Join(work, "rootless")); err != nil {
		return s, err
	}
	pub, err := readPrivate(filepath.Join(work, "operator.pub"), 8192)
	if err != nil {
		return s, err
	}
	original, err := readPrivate(filepath.Join(dir, "operator.pub"), 8192)
	if err != nil || !bytes.Equal(pub, original) {
		return s, errors.New("retained operator key changed; preserving the update")
	}
	current, err := d.binding(ctx, config, dir)
	if err != nil || current != b {
		return s, errors.New("local setup changed during preparation; check again")
	}
	retained := filepath.Join(work, "urbit-runner")
	data, _, err := d.binary(retained)
	if err != nil || upgradeDigest(data) != target {
		return s, errors.New("retained installer changed; preserving the update")
	}
	step("authorize", "Approve the service update in the operating-system prompt")
	host, err := d.elevate(ctx, retained, filepath.Join(work, "rootless"), filepath.Join(work, "operator.pub"), expected, func(e Event) {
		if emit != nil {
			emit(UpgradeEvent{Step: e.Step, Status: e.Status, Message: e.Message})
		}
	})
	if err != nil {
		return s, err
	}
	if !host.Ready || host.RunnerID != expected || host.Owner != b.Owner.Name || host.UID != b.Owner.UID || host.PierRoot != b.Owner.Piers || host.Mode != "development-managed" || host.SSHPort != 22222 || strings.TrimSpace(host.HostPublicKey) != strings.TrimSpace(b.HostKey) || host.OperatorFingerprint != b.Fingerprint {
		return s, errors.New("updated host did not match the original installation; keep this attempt for review")
	}
	step("verify", "Checking the updated runner and original desktop access")
	installed, _, err := d.binary(d.installed)
	if err != nil || upgradeDigest(installed) != target {
		return s, errors.New("installed executable does not match the approved update")
	}
	if err = d.probe(ctx, b); err != nil {
		return s, err
	}
	a.Complete = true
	if err = writeUpgradeAttempt(dir, a); err != nil {
		return s, err
	}
	s, err = serviceUpgradeStatus(ctx, config, dir, runner, d)
	if err != nil {
		return s, err
	}
	if emit != nil {
		emit(UpgradeEvent{Step: "upgrade", Status: "complete", Message: "Local service update completed. Your fleet is ready.", Result: &s})
	}
	return s, nil
}

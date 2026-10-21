package onboarding

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type RemoteOptions struct {
	Host      string `json:"host"`
	Name      string `json:"name"`
	SSHConfig string `json:"ssh_config,omitempty"`
	LoginUser string `json:"login_user,omitempty"`
	Port      uint   `json:"port,omitempty"`
	Owner     string `json:"owner,omitempty"`
	Role      string `json:"role"`
}

type RemoteStatus struct {
	Version     int           `json:"version"`
	ID          string        `json:"id"`
	Options     RemoteOptions `json:"options"`
	Phase       string        `json:"phase"`
	Message     string        `json:"message"`
	RunnerID    string        `json:"runner_id,omitempty"`
	Mode        string        `json:"mode,omitempty"`
	Ships       int           `json:"ships"`
	Fingerprint string        `json:"fingerprint,omitempty"`
	Connection  string        `json:"connection,omitempty"`
	UpdatedAt   time.Time     `json:"updated_at"`
	Busy        bool          `json:"busy"`
	Revoking    bool          `json:"revoking"`
}

type remoteJob struct {
	RemoteStatus
	ConfigDir       string `json:"config_dir"`
	Helper          string `json:"helper"`
	HelperSHA256    string `json:"helper_sha256"`
	Revoke          bool   `json:"revoke,omitempty"`
	EndpointSHA256  string `json:"endpoint_sha256,omitempty"`
	sessionEndpoint *sshResolved
}

func remoteRoot() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "omabit", "remote")
	return dir, privateDirectory(dir)
}

func remoteID() (string, error) {
	var value [16]byte
	_, err := rand.Read(value[:])
	return hex.EncodeToString(value[:]), err
}
func validRemoteID(id string) bool {
	b, e := hex.DecodeString(id)
	return e == nil && len(b) == 16 && hex.EncodeToString(b) == id
}

func remoteName(host string) string {
	var result strings.Builder
	for _, r := range host {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' {
			result.WriteRune(r)
		} else {
			result.WriteByte('-')
		}
		if result.Len() == 48 {
			break
		}
	}
	return result.String()
}

func (o *RemoteOptions) validate() error {
	if len(o.SSHConfig) > 4096 {
		return errors.New("SSH configuration path exceeds its limit")
	}
	if err := (client.SSHConfig{Host: o.Host, ConfigFile: o.SSHConfig}).Validate(); err != nil {
		return err
	}
	if o.Name == "" {
		o.Name = remoteName(o.Host)
	}
	if o.Role == "" {
		o.Role = "operator"
	}
	if !protocol.ValidID(o.Name) || o.Port > 65535 || o.Role != "operator" && o.Role != "observer" {
		return errors.New("choose a connection name, valid SSH port and operator/observer role")
	}
	for _, user := range []string{o.LoginUser, o.Owner} {
		if user != "" && (!protocol.ValidID(user) || strings.HasPrefix(user, "-")) {
			return errors.New("invalid Unix account name")
		}
	}
	return nil
}

func readRemote(root, id string) (remoteJob, string, error) {
	if !validRemoteID(id) {
		return remoteJob{}, "", errors.New("invalid remote setup ID")
	}
	dir := filepath.Join(root, id)
	if _, err := os.Lstat(dir); err != nil {
		return remoteJob{}, "", err
	}
	if err := privateDirectory(dir); err != nil {
		return remoteJob{}, "", err
	}
	data, err := readPrivate(filepath.Join(dir, "state.json"), 65536)
	var job remoteJob
	if err == nil {
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		err = decoder.Decode(&job)
		var extra any
		if err == nil && decoder.Decode(&extra) != io.EOF {
			err = errors.New("extra remote setup state data")
		}
	}
	if err != nil {
		return job, dir, err
	}
	if job.Version != 1 || job.ID != id || !protocol.ValidID(job.Options.Name) || (job.Options.Role != "operator" && job.Options.Role != "observer") || job.Options.validate() != nil || !filepath.IsAbs(job.ConfigDir) || len(job.Message) > 8192 {
		return job, dir, errors.New("unsupported remote setup state; preserve it for review")
	}
	return job, dir, nil
}

func saveRemote(dir string, job *remoteJob) error {
	if len(job.Message) > 8192 {
		job.Message = job.Message[:8192]
	}
	job.UpdatedAt = time.Now().UTC()
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return writePrivate(filepath.Join(dir, "state.json"), data)
}

func RemoteState(id string) (RemoteStatus, error) {
	root, err := remoteRoot()
	if err != nil {
		return RemoteStatus{}, err
	}
	if id == "" {
		data, err := readPrivate(filepath.Join(root, "last.json"), 1024)
		if errors.Is(err, os.ErrNotExist) {
			return RemoteStatus{Version: 1, Phase: "idle"}, nil
		}
		if err != nil {
			return RemoteStatus{}, err
		}
		if json.Unmarshal(data, &id) != nil {
			return RemoteStatus{}, errors.New("invalid remote setup selection")
		}
	}
	job, dir, err := readRemote(root, id)
	if err != nil {
		return RemoteStatus{}, err
	}
	status := job.RemoteStatus
	status.Revoking = job.Revoke
	status.Busy, err = remoteLockBusy(dir)
	if err != nil {
		return RemoteStatus{}, err
	}
	if !status.Busy && (status.Phase == "opening_terminal" && time.Since(status.UpdatedAt) < 30*time.Second) {
		status.Busy = true
	}
	if !status.Busy && status.Phase != "connected" && status.Phase != "failed" && status.Phase != "revoked" {
		status.Phase = "interrupted"
		status.Message = "Setup stopped before completion. Resume to inspect and continue with the same dedicated key."
	}
	return status, nil
}

func remoteLockBusy(dir string) (bool, error) {
	path := filepath.Join(dir, ".lock")
	if _, err := readPrivate(path, 0); err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return false, err
	}
	defer f.Close()
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return true, nil
	}
	return false, err
}

// PrepareRemote creates or reuses an immutable target/key binding. A launch only
// opens an authentication terminal; enrollment happens after a separate RPC probe.
func PrepareRemote(ctx context.Context, options RemoteOptions, configDir, id string, revoke bool) (RemoteStatus, error) {
	executable, err := os.Executable()
	if err != nil {
		return RemoteStatus{}, err
	}
	return prepareRemote(ctx, options, configDir, id, revoke, filepath.Join(filepath.Dir(executable), "urbit-runner"))
}

func prepareRemote(ctx context.Context, options RemoteOptions, configDir, id string, revoke bool, helper string) (RemoteStatus, error) {
	if os.Geteuid() == 0 {
		return RemoteStatus{}, errors.New("run remote setup as your desktop user")
	}
	if configDir == "" {
		var err error
		configDir, err = collection.DefaultDir()
		if err != nil {
			return RemoteStatus{}, err
		}
	}
	if !filepath.IsAbs(configDir) || filepath.Clean(configDir) != configDir {
		return RemoteStatus{}, errors.New("collector directory must be an absolute clean path")
	}
	root, err := remoteRoot()
	if err != nil {
		return RemoteStatus{}, err
	}
	unlock, err := lockSetup(root)
	if err != nil {
		return RemoteStatus{}, err
	}
	defer unlock()
	var job remoteJob
	var dir string
	if id == "" {
		if err := options.validate(); err != nil {
			return RemoteStatus{}, err
		}
		entries, err := os.ReadDir(root)
		if err != nil {
			return RemoteStatus{}, err
		}
		for _, entry := range entries {
			if !validRemoteID(entry.Name()) {
				continue
			}
			old, _, err := readRemote(root, entry.Name())
			if err != nil {
				return RemoteStatus{}, err
			}
			if old.Options == options && old.ConfigDir == configDir {
				id = old.ID
				break
			}
		}
		if id == "" {
			if revoke {
				return RemoteStatus{}, errors.New("select an existing pairing to revoke")
			}
			id, err = remoteID()
			if err != nil {
				return RemoteStatus{}, err
			}
			dir = filepath.Join(root, id)
			if err = privateDirectory(dir); err != nil {
				return RemoteStatus{}, err
			}
			job = remoteJob{RemoteStatus: RemoteStatus{Version: 1, ID: id, Options: options, Phase: "prepared"}, ConfigDir: configDir}
			registry, err := collection.Open(configDir)
			if err != nil {
				return RemoteStatus{}, err
			}
			entries, err := registry.List(ctx)
			if err != nil {
				return RemoteStatus{}, err
			}
			for _, entry := range entries {
				if entry.Name == options.Name {
					job.RunnerID = entry.RunnerID
					job.Mode = entry.Mode
				}
			}
			if err = saveRemote(dir, &job); err != nil {
				return RemoteStatus{}, err
			}
		}
	}
	job, dir, err = readRemote(root, id)
	if err != nil {
		return RemoteStatus{}, err
	}
	if job.ConfigDir != configDir {
		return RemoteStatus{}, errors.New("pairing belongs to a different collector profile")
	}
	jobUnlock, err := lockSetup(dir)
	if err != nil {
		return RemoteStatus{}, err
	}
	defer jobUnlock()
	if job.Phase == "opening_terminal" && time.Since(job.UpdatedAt) < 30*time.Second {
		return RemoteStatus{}, errors.New("setup terminal is already opening")
	}
	if revoke && job.RunnerID == "" {
		return RemoteStatus{}, errors.New("pairing has no verified remote identity yet")
	}
	public, err := desktopKey(ctx, filepath.Join(dir, "id_ed25519"))
	if err != nil {
		return RemoteStatus{}, err
	}
	line, err := access.OwnerAuthorizedKey(public, job.Options.Role)
	if err != nil {
		return RemoteStatus{}, err
	}
	grant, _ := access.ParseLine(line)
	if job.Fingerprint != "" && job.Fingerprint != grant.Fingerprint {
		return RemoteStatus{}, errors.New("pairing key changed; existing state was preserved")
	}
	job.Fingerprint = grant.Fingerprint
	// A failed/lost-ack revocation must resume revocation, never silently grant
	// access again. A completed revoked pairing may be explicitly connected anew.
	job.Revoke = revoke || job.Revoke && job.Phase != "revoked"
	job.Revoking = job.Revoke
	digest, err := fileDigest(helper)
	if err != nil {
		return RemoteStatus{}, fmt.Errorf("matching bundled runner helper is required: %w", err)
	}
	// An interrupted attempt keeps its original executable generation until a
	// deliberate new setup can review a different installer; no silent upgrade.
	if job.HelperSHA256 != "" && job.HelperSHA256 != digest && job.Phase != "connected" && job.Phase != "revoked" {
		original, e := fileDigest(job.Helper)
		if e != nil || original != job.HelperSHA256 {
			return RemoteStatus{}, errors.New("resume this attempt with its original helper generation")
		}
		helper, digest = job.Helper, original
	}
	helper, err = retainRemoteHelper(dir, helper, digest)
	if err != nil {
		return RemoteStatus{}, err
	}
	job.Helper, job.HelperSHA256 = helper, digest
	job.Phase = "opening_terminal"
	job.Message = "Opening setup terminal for SSH authentication and remote authorization."
	if err = saveRemote(dir, &job); err != nil {
		return RemoteStatus{}, err
	}
	last, _ := json.Marshal(id)
	if err = writePrivate(filepath.Join(root, "last.json"), last); err != nil {
		return RemoteStatus{}, err
	}
	return job.RemoteStatus, nil
}

func retainRemoteHelper(dir, source, digest string) (string, error) {
	path := filepath.Join(dir, "installer-"+digest)
	if _, err := os.Lstat(path); err == nil {
		actual, e := fileDigest(path)
		if e != nil || actual != digest {
			return "", errors.New("retained remote installer changed; preserve it for review")
		}
		return path, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	in, err := os.Open(source)
	if err != nil {
		return "", err
	}
	defer in.Close()
	out, err := os.CreateTemp(dir, ".installer-")
	if err != nil {
		return "", err
	}
	defer os.Remove(out.Name())
	n, copyErr := io.Copy(out, io.LimitReader(in, (128<<20)+1))
	err = errors.Join(copyErr, out.Chmod(0500), out.Sync(), out.Close())
	if err != nil || n > 128<<20 {
		return "", errors.New("remote installer snapshot could not be retained")
	}
	actual, err := fileDigest(out.Name())
	if err != nil || actual != digest {
		return "", errors.New("remote installer changed while being retained")
	}
	if err = os.Link(out.Name(), path); err != nil {
		return "", err
	}
	if err = os.Remove(out.Name()); err != nil {
		return "", err
	}
	directory, err := os.Open(dir)
	if err != nil {
		return "", err
	}
	defer directory.Close()
	return path, directory.Sync()
}

func LaunchRemote(ctx context.Context, options RemoteOptions, configDir, id string, revoke bool) (RemoteStatus, error) {
	status, err := PrepareRemote(ctx, options, configDir, id, revoke)
	if err != nil {
		return status, err
	}
	exe, err := os.Executable()
	if err != nil {
		return status, err
	}
	args := []string{"launch", "terminal", "/usr/bin/env"}
	// Terminal launchers may delegate through a user service manager with a
	// different environment. Keep named-profile credentials in their own roots.
	for _, key := range []string{"XDG_CONFIG_HOME", "XDG_STATE_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME"} {
		value := os.Getenv(key)
		if key == "XDG_CONFIG_HOME" && value == "" {
			value, _ = os.UserConfigDir()
		}
		if value != "" {
			args = append(args, key+"="+value)
		}
	}
	args = append(args, exe)
	if configDir != "" {
		args = append(args, "--config-dir", configDir)
	}
	args = append(args, "remote-setup-run", "--id", status.ID)
	cmd := exec.Command("omarchy", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err = cmd.Start(); err != nil {
		root, _ := remoteRoot()
		job, dir, e := readRemote(root, status.ID)
		if e == nil {
			job.Phase = "failed"
			job.Message = "Could not open a terminal. Run omabit remote-setup-run --id " + status.ID + " in your terminal."
			_ = saveRemote(dir, &job)
		}
		return status, errors.New("could not open the authentication terminal")
	}
	go func() { _ = cmd.Wait() }()
	status.Busy = true
	return status, nil
}

package onboarding

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/remotesetup"
	"github.com/mattn/go-isatty"
)

func fileDigest(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || st.Size() > 128<<20 {
		return "", errors.New("invalid helper file")
	}
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }

func sshBase(o RemoteOptions) []string {
	args := []string{"-oForwardAgent=no", "-oForwardX11=no", "-oClearAllForwardings=yes", "-oPermitLocalCommand=no", "-oRemoteCommand=none", "-oConnectTimeout=10", "-oConnectionAttempts=1", "-oServerAliveInterval=15", "-oServerAliveCountMax=2"}
	if o.SSHConfig != "" {
		args = append(args, "-F", o.SSHConfig)
	}
	if o.LoginUser != "" {
		args = append(args, "-l", o.LoginUser)
	}
	if o.Port != 0 {
		args = append(args, "-p", strconv.FormatUint(uint64(o.Port), 10))
	}
	return args
}

type sshResolved struct {
	Host, User, Port, HostKeyAlias, ProxyJump, ProxyCommand string
	KnownHosts                                              []string
}

func bindRemoteEndpoint(job *remoteJob, r sshResolved) error {
	// Bind endpoint and routing, but permit normal administrator-key unlocking.
	r.KnownHosts = nil
	data, _ := json.Marshal(r)
	digest := sha256.Sum256(data)
	value := hex.EncodeToString(digest[:])
	if job.EndpointSHA256 != "" && job.EndpointSHA256 != value {
		return errors.New("the SSH alias now resolves to a different host, user, port or route; restore the original target before resuming")
	}
	job.EndpointSHA256 = value
	job.sessionEndpoint = &r
	return nil
}

func sessionSSHBase(job remoteJob) []string {
	args := sshBase(job.Options)
	if r := job.sessionEndpoint; r != nil {
		args = append(args, "-oHostName="+r.Host, "-l", r.User, "-p", r.Port)
		if r.HostKeyAlias != "" {
			args = append(args, "-oHostKeyAlias="+r.HostKeyAlias)
		}
		if r.ProxyJump != "" {
			args = append(args, "-oProxyJump="+r.ProxyJump)
		} else if r.ProxyCommand != "" {
			args = append(args, "-oProxyCommand="+r.ProxyCommand)
		} else {
			args = append(args, "-oProxyCommand=none", "-oProxyJump=none")
		}
	}
	return args
}

func masterSSHArgs(job remoteJob, control, known string) []string {
	return append(sessionSSHBase(job), "-M", "-S", control, "-oControlPersist=60", "-oBatchMode=no", "-oStrictHostKeyChecking=ask", "-oUpdateHostKeys=no", "-oGlobalKnownHostsFile=/dev/null", "-oUserKnownHostsFile="+known, "-fNT", "--", job.Options.Host)
}

func resolveSSH(ctx context.Context, o RemoteOptions) (sshResolved, error) {
	args := append(sshBase(o), "-G", "--", o.Host)
	cmd := exec.CommandContext(ctx, "/usr/bin/ssh", args...)
	var out boundedOutput
	cmd.Stdout = &out
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return sshResolved{}, fmt.Errorf("read SSH configuration for %s: %w", o.Host, err)
	}
	if out.overflow {
		return sshResolved{}, errors.New("SSH configuration output exceeded its bound")
	}
	r := sshResolved{}
	for _, line := range strings.Split(out.String(), "\n") {
		key, value, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		switch key {
		case "hostname":
			r.Host = value
		case "user":
			r.User = value
		case "port":
			r.Port = value
		case "hostkeyalias":
			if value != "none" {
				r.HostKeyAlias = value
			}
		case "proxyjump":
			if value != "none" {
				r.ProxyJump = value
			}
		case "proxycommand":
			if value != "none" {
				r.ProxyCommand = value
			}
		case "userknownhostsfile", "globalknownhostsfile":
			r.KnownHosts = append(r.KnownHosts, strings.Fields(value)...)
		}
	}
	port, err := strconv.ParseUint(r.Port, 10, 16)
	if err != nil || port == 0 || r.Host == "" || r.User == "" || strings.ContainsAny(r.Host+r.User+r.HostKeyAlias, "\r\n\x00") {
		return r, errors.New("SSH alias did not resolve to a host, user and port")
	}
	return r, nil
}

type boundedOutput struct {
	bytes.Buffer
	overflow bool
}

func (b *boundedOutput) Write(p []byte) (int, error) {
	n := len(p)
	left := (1 << 20) - b.Len()
	if len(p) > left {
		p = p[:left]
		b.overflow = true
	}
	_, err := b.Buffer.Write(p)
	return n, err
}

func seedKnownHosts(ctx context.Context, dir string, r sshResolved) error {
	path := filepath.Join(dir, "known_hosts")
	if _, err := readPrivate(path, 1<<20); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	lookup := r.Host
	if r.HostKeyAlias != "" {
		lookup = r.HostKeyAlias
	} else if r.Port != "22" {
		lookup = "[" + lookup + "]:" + r.Port
	}
	var contents []byte
	home, _ := os.UserHomeDir()
	for _, file := range r.KnownHosts {
		if strings.HasPrefix(file, "~/") {
			file = filepath.Join(home, file[2:])
		}
		if !filepath.IsAbs(file) || strings.ContainsAny(file, "%\x00\r\n") {
			continue
		}
		cmd := exec.CommandContext(ctx, "/usr/bin/ssh-keygen", "-F", lookup, "-f", file)
		var output boundedOutput
		cmd.Stdout = &output
		cmd.Stderr = io.Discard
		if err := cmd.Run(); err != nil {
			continue
		}
		if output.overflow {
			return errors.New("host identity lookup exceeded its bound")
		}
		contents = append(contents, output.Bytes()...)
	}
	return writePrivate(path, contents)
}

func managementConfig(dir string, job remoteJob, r sshResolved, owner string) (client.SSHConfig, error) {
	if !validRemoteID(job.ID) || !validSSHWord(owner) || !validSSHWord(r.Host) || !validSSHWord(r.Port) || r.HostKeyAlias != "" && !validSSHWord(r.HostKeyAlias) {
		return client.SSHConfig{}, errors.New("remote endpoint contains unsupported SSH configuration values")
	}
	alias := "omabit-" + job.ID
	var b strings.Builder
	fmt.Fprintf(&b, "Host %s\n  HostName %s\n  Port %s\n  User %s\n", alias, r.Host, r.Port, owner)
	fmt.Fprintf(&b, "  IdentityFile %s\n  IdentityAgent none\n  IdentitiesOnly yes\n  PreferredAuthentications publickey\n  PasswordAuthentication no\n  KbdInteractiveAuthentication no\n  BatchMode yes\n  UserKnownHostsFile %s\n  GlobalKnownHostsFile /dev/null\n  StrictHostKeyChecking yes\n  UpdateHostKeys no\n  ControlMaster no\n  ControlPath none\n  ControlPersist no\n  RequestTTY no\n  RemoteCommand none\n", sshPath(filepath.Join(dir, "id_ed25519")), sshPath(filepath.Join(dir, "known_hosts")))
	if r.HostKeyAlias != "" {
		fmt.Fprintf(&b, "  HostKeyAlias %s\n", r.HostKeyAlias)
	}
	if strings.ContainsAny(r.ProxyJump+r.ProxyCommand, "\x00\r\n") {
		return client.SSHConfig{}, errors.New("unsupported SSH proxy configuration")
	}
	if r.ProxyJump != "" {
		fmt.Fprintf(&b, "  ProxyJump %s\n", r.ProxyJump)
	} else if r.ProxyCommand != "" {
		fmt.Fprintf(&b, "  ProxyCommand %s\n", strings.ReplaceAll(r.ProxyCommand, "%n", job.Options.Host))
	}
	// Jump-host definitions remain the user's. Their credentials must pass the
	// fresh no-agent probe too; we never reuse our administrator control socket.
	if r.ProxyJump != "" || r.ProxyCommand != "" {
		original := job.Options.SSHConfig
		if original == "" {
			home, _ := os.UserHomeDir()
			original = filepath.Join(home, ".ssh/config")
		}
		fmt.Fprintf(&b, "\nHost *\n  BatchMode yes\n  Include %s\n  Include /etc/ssh/ssh_config\n", sshPath(original))
	}
	path := filepath.Join(dir, "ssh_config")
	if err := writePrivate(path, []byte(b.String())); err != nil {
		return client.SSHConfig{}, err
	}
	return client.SSHConfig{Host: alias, ConfigFile: path}, nil
}

func validSSHWord(s string) bool {
	return s != "" && !strings.ContainsAny(s, "\x00\r\n\t \"'\\%") && !strings.HasPrefix(s, "-")
}

func remoteCommand(ctx context.Context, job remoteJob, control, command string, input io.Reader, interactive bool) ([]byte, error) {
	args := append(sessionSSHBase(job), "-S", control, "-oControlMaster=no", "-oBatchMode=yes", "-oForkAfterAuthentication=no", "-oSessionType=default")
	if interactive {
		args = append(args, "-tt")
	} else {
		args = append(args, "-T")
	}
	args = append(args, "--", job.Options.Host, command)
	cmd := exec.CommandContext(ctx, "/usr/bin/ssh", args...)
	cmd.WaitDelay = 2 * time.Second
	if interactive {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return nil, cmd.Run()
	}
	var output boundedOutput
	cmd.Stdin = input
	cmd.Stdout = &output
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if output.overflow {
		return nil, errors.New("remote setup output exceeded its bound")
	}
	return output.Bytes(), err
}

func verifyRemote(ctx context.Context, config client.SSHConfig, result remotesetup.Result) (int, error) {
	// The worker can have an administrator agent, but the saved management path
	// must work in a clean desktop with no agent and no connection multiplexer.
	env := []string{}
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "SSH_AUTH_SOCK=") && !strings.HasPrefix(entry, "SSH_AGENT_PID=") {
			env = append(env, entry)
		}
	}
	c := client.NewSSHWithEnvironment(config, &client.Identity{RunnerID: result.RunnerID, Mode: result.Mode}, env)
	defer c.Close()
	h, err := c.Hello(ctx)
	if err != nil {
		return 0, fmt.Errorf("dedicated-key connection is not ready for unattended use: %w", err)
	}
	if h.AccessRole != result.Role || h.Actor != "ssh-key:"+result.Fingerprint {
		return 0, errors.New("management endpoint did not authenticate the dedicated desktop key and role")
	}
	s, err := c.Snapshot(ctx)
	if err != nil {
		return 0, err
	}
	if s.RunnerID != result.RunnerID {
		return 0, errors.New("runner identity changed while reading inventory")
	}
	return len(s.Instances), nil
}

func enrollRemote(ctx context.Context, job *remoteJob, config client.SSHConfig, result remotesetup.Result) (collection.Connection, error) {
	registry, err := collection.Open(job.ConfigDir)
	if err != nil {
		return collection.Connection{}, err
	}
	entries, err := registry.List(ctx)
	if err != nil {
		return collection.Connection{}, err
	}
	for _, old := range entries {
		if old.RunnerID != result.RunnerID {
			continue
		}
		if old.Mode != result.Mode {
			return collection.Connection{}, errors.New("existing runner mode changed; review it in Manage runners")
		}
		if old.Transport == "ssh" && old.SSH != nil && *old.SSH == config {
			return old, nil
		}
		return registry.Update(ctx, old.Name, old.RunnerID, old.Revision, collection.Connection{Name: old.Name, Transport: "ssh", SSH: &config})
	}
	return registry.EnrollSSHExpected(ctx, job.Options.Name, config, client.Identity{RunnerID: result.RunnerID, Mode: result.Mode})
}

func forgetRemote(ctx context.Context, job remoteJob, dir string) error {
	registry, err := collection.Open(job.ConfigDir)
	if err != nil {
		return err
	}
	entries, err := registry.List(ctx)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.RunnerID == job.RunnerID && entry.Transport == "ssh" && entry.SSH != nil && entry.SSH.Host == "omabit-"+job.ID && entry.SSH.ConfigFile == filepath.Join(dir, "ssh_config") {
			return registry.ForgetSSHExpected(ctx, entry)
		}
	}
	// The acknowledgement may have been lost after a previous successful forget.
	// An enrollment subsequently moved to another credential remains untouched.
	return nil
}

func RunRemote(ctx context.Context, id string) (err error) {
	if os.Geteuid() == 0 || !isatty.IsTerminal(os.Stdin.Fd()) {
		return errors.New("remote setup must run as your desktop user in an authentication terminal")
	}
	root, err := remoteRoot()
	if err != nil {
		return err
	}
	job, dir, err := readRemote(root, id)
	if err != nil {
		return err
	}
	unlock, err := lockSetup(dir)
	if err != nil {
		return err
	}
	defer unlock()
	defer func() {
		if err != nil {
			job.Phase = "failed"
			job.Message = err.Error()
			_ = saveRemote(dir, &job)
		}
	}()
	progress := func(phase, message string) error {
		job.Phase = phase
		job.Message = message
		fmt.Fprintln(os.Stdout, message)
		return saveRemote(dir, &job)
	}
	digest, err := fileDigest(job.Helper)
	if err != nil || digest != job.HelperSHA256 {
		return errors.New("the selected installer generation changed; preserve this attempt and resume with its original tools")
	}
	public, err := desktopKey(ctx, filepath.Join(dir, "id_ed25519"))
	if err != nil {
		return err
	}
	line, err := access.OwnerAuthorizedKey(public, job.Options.Role)
	if err != nil {
		return err
	}
	grant, _ := access.ParseLine(line)
	if grant.Fingerprint != job.Fingerprint {
		return errors.New("dedicated key identity changed")
	}
	if err = progress("authenticate", "Sign in to "+job.Options.Host+". OpenSSH will request your key/passphrase and confirm a new host if needed."); err != nil {
		return err
	}
	r, err := resolveSSH(ctx, job.Options)
	if err != nil {
		return err
	}
	if err = bindRemoteEndpoint(&job, r); err != nil {
		return err
	}
	if err = saveRemote(dir, &job); err != nil {
		return err
	}
	if err = seedKnownHosts(ctx, dir, r); err != nil {
		return err
	}
	// /tmp keeps the Unix control socket comfortably below the kernel path bound.
	controlDir, err := os.MkdirTemp("", "omabit-ssh-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(controlDir)
	control := filepath.Join(controlDir, "control")
	args := masterSSHArgs(job, control, filepath.Join(dir, "known_hosts"))
	master := exec.CommandContext(ctx, "/usr/bin/ssh", args...)
	master.Stdin = os.Stdin
	master.Stdout = os.Stdout
	master.Stderr = os.Stderr
	if err = master.Run(); err != nil {
		return fmt.Errorf("SSH authentication did not complete: %w", err)
	}
	defer func() {
		stop, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		args := append(sshBase(job.Options), "-S", control, "-O", "exit", "--", job.Options.Host)
		_ = exec.CommandContext(stop, "/usr/bin/ssh", args...).Run()
	}()
	if err = progress("prepare", "Preparing the verified setup helper on the remote host."); err != nil {
		return err
	}
	nonce, err := remoteID()
	if err != nil {
		return err
	}
	stage := "/tmp/omabit-setup-" + nonce
	remoteHelper := stage + "/urbit-runner"
	data, err := os.ReadFile(job.Helper)
	if err != nil || len(data) > 128<<20 {
		return errors.New("installer generation is unavailable")
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != job.HelperSHA256 {
		return errors.New("installer changed while staging")
	}
	command := "set -eu; umask 077; mkdir -- " + shellQuote(stage) + "; cat > " + shellQuote(remoteHelper) + "; chmod 700 " + shellQuote(remoteHelper) + "; printf '%s\\n' " + shellQuote(job.HelperSHA256+"  "+remoteHelper) + " | sha256sum -c - >/dev/null"
	if _, err = remoteCommand(ctx, job, control, command, bytes.NewReader(data), false); err != nil {
		return fmt.Errorf("stage the setup helper: %w", err)
	}
	request := remotesetup.Request{Version: 1, ID: job.ID, PublicKey: string(public), Role: job.Options.Role, Owner: job.Options.Owner, RunnerID: job.RunnerID, Revoke: job.Revoke}
	payload, _ := json.Marshal(request)
	if err = progress("authorize", "Review the detected runner and approve access or installation in the setup terminal."); err != nil {
		return err
	}
	// No password passes through our protocol. sudo and the remote helper inherit
	// the real terminal, while the result file contains public connection data only.
	command = "set -eu; if [ \"$(id -u)\" = 0 ]; then elevate=''; else elevate=sudo; fi; $elevate -- " + shellQuote(remoteHelper) + " remote-bootstrap --request " + shellQuote(base64.StdEncoding.EncodeToString(payload)) + " --login-user \"$(id -un)\" --ssh-connection \"$SSH_CONNECTION\" --result " + shellQuote(stage+"/result.json")
	// Root does not need sudo; avoid passing '--' as the executable in that case.
	command = strings.Replace(command, "then elevate='';", "then elevate=env;", 1)
	_, setupErr := remoteCommand(ctx, job, control, command, nil, true)
	output, resultErr := remoteCommand(ctx, job, control, "cat -- "+shellQuote(stage+"/result.json"), nil, false)
	var envelope struct {
		Result remotesetup.Result `json:"result"`
		Error  string             `json:"error"`
	}
	if resultErr != nil || json.Unmarshal(output, &envelope) != nil {
		return errors.New("remote setup did not return its acknowledgement; resume the same attempt to inspect and continue")
	}
	if envelope.Error != "" {
		return errors.New(envelope.Error)
	}
	if setupErr != nil {
		return fmt.Errorf("remote setup was interrupted; its result is retained for retry: %w", setupErr)
	}
	result := envelope.Result
	if result.Version != 1 || result.ID != job.ID || result.Fingerprint != job.Fingerprint || result.Role != job.Options.Role || result.RunnerID == "" || result.Mode != "development-managed" || job.RunnerID != "" && result.RunnerID != job.RunnerID {
		return errors.New("remote setup result did not match this request, key or runner")
	}
	job.RunnerID, job.Mode = result.RunnerID, result.Mode
	if err = saveRemote(dir, &job); err != nil {
		return err
	}
	if job.Revoke {
		if result.State != "revoked" {
			return errors.New("remote access revocation was not acknowledged")
		}
		if e := forgetRemote(ctx, job, dir); e != nil {
			return e
		}
		return progress("revoked", "This desktop's remote management access was revoked.")
	}
	if result.State != "authorized" {
		return errors.New("remote management key was not authorized")
	}
	config, err := managementConfig(dir, job, r, result.Owner)
	if err != nil {
		return err
	}
	if err = progress("verify", "Verifying a fresh unattended connection with the dedicated Omabit key."); err != nil {
		return err
	}
	ships, err := verifyRemote(ctx, config, result)
	if err != nil {
		return err
	}
	entry, err := enrollRemote(ctx, &job, config, result)
	if err != nil {
		return err
	}
	job.Connection = entry.Name
	job.Ships = ships
	return progress("connected", fmt.Sprintf("%s connected. Found %d ship(s).", entry.Name, ships))
}

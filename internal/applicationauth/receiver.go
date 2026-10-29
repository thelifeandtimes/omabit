package applicationauth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"golang.org/x/sys/unix"
)

var ErrArtifact = errors.New("application receiver artifact changed or could not be verified")

// Artifact holds the approved source bytes. Launches use a fresh private copy of
// these bytes, never the mutable checkout. This is artifact pinning, not a sandbox
// against malicious code with the same Unix identity as the user.
type Artifact struct {
	root, entry string
	target      collection.ApplicationTarget
	files       map[string][]byte
	names       []string
	environment []string
}

// OpenReceiver loads an explicit helper inventory. The trusted desktop resolver
// chooses root/entry/inventory; none are shell commands or receiver requests.
// Files are regular, single-link, owned source files under a non-symlink root.
// HOME/XDG/bus environment is explicit so the lab never inherits desktop state.
func OpenReceiver(root, entry string, inventory []string, environment map[string]string) (*Artifact, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || len(inventory) == 0 || len(inventory) > 128 {
		return nil, ErrArtifact
	}
	if resolved, err := filepath.EvalSymlinks(root); err != nil || resolved != root {
		return nil, ErrArtifact
	}
	a := &Artifact{root: root, entry: entry, files: map[string][]byte{}}
	total := 0
	for _, name := range inventory {
		if a.files[name] != nil || !artifactPath(name) {
			return nil, ErrArtifact
		}
		data, err := a.read(name)
		if err != nil {
			return nil, err
		}
		total += len(data)
		if total > 8<<20 {
			return nil, ErrArtifact
		}
		a.files[name] = data
		a.names = append(a.names, name)
	}
	if a.files[entry] == nil || !strings.HasSuffix(entry, ".py") || a.files["manifest.json"] == nil {
		return nil, ErrArtifact
	}
	var manifest struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(a.files["manifest.json"], &manifest) != nil {
		return nil, ErrArtifact
	}
	if _, err := os.Lstat(filepath.Join(root, "application-profile.json")); err == nil && a.files["application-profile.json"] == nil {
		return nil, ErrArtifact
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, ErrArtifact
	}
	profile := "default"
	if data, ok := a.files["application-profile.json"]; ok {
		var metadata struct {
			Version int    `json:"version"`
			Profile string `json:"profile"`
		}
		d := json.NewDecoder(bytes.NewReader(data))
		d.DisallowUnknownFields()
		if d.Decode(&metadata) != nil || d.Decode(new(any)) != io.EOF || metadata.Version != 1 {
			return nil, ErrArtifact
		}
		profile = metadata.Profile
	}
	if profile != "default" && !strings.HasSuffix(manifest.ID, "."+profile) {
		return nil, ErrArtifact
	}
	sort.Strings(a.names)
	h := sha256.New()
	_, _ = io.WriteString(h, "omabit-python-receiver-v1\x00"+entry+"\x00")
	for _, name := range a.names {
		sum := sha256.Sum256(a.files[name])
		_, _ = io.WriteString(h, name+"\x00"+hex.EncodeToString(sum[:])+"\x00")
	}
	a.target = collection.ApplicationTarget{PluginID: manifest.ID, Profile: profile, SHA256: hex.EncodeToString(h.Sum(nil))}
	if !a.target.Valid() {
		return nil, ErrArtifact
	}
	if !filepath.IsAbs(environment["HOME"]) {
		return nil, ErrArtifact
	}
	allowed := map[string]bool{"HOME": true, "XDG_CONFIG_HOME": true, "XDG_STATE_HOME": true, "XDG_DATA_HOME": true, "XDG_CACHE_HOME": true, "XDG_RUNTIME_DIR": true, "DBUS_SESSION_BUS_ADDRESS": true, "LANG": true}
	a.environment = []string{"PATH=/usr/bin:/bin"}
	var keys []string
	for key, value := range environment {
		if !allowed[key] || strings.ContainsAny(value, "\x00\r\n") || (strings.HasPrefix(key, "XDG_") && !filepath.IsAbs(value)) {
			return nil, ErrArtifact
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		a.environment = append(a.environment, key+"="+environment[key])
	}
	return a, nil
}

func artifactPath(name string) bool {
	return name != "" && name != receiverDescriptor && !filepath.IsAbs(name) && filepath.Clean(name) == name && name != ".." && !strings.HasPrefix(name, "../") && !strings.ContainsAny(name, "\x00\r\n\\") && (strings.HasSuffix(name, ".py") || strings.HasSuffix(name, ".json"))
}

func (a *Artifact) read(name string) ([]byte, error) {
	path := filepath.Join(a.root, name)
	resolved, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil || resolved != filepath.Dir(path) {
		return nil, ErrArtifact
	}
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, ErrArtifact
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, ErrArtifact
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || !info.Mode().IsRegular() || st.Uid != uint32(os.Geteuid()) || st.Nlink != 1 || info.Size() > 1<<20 {
		return nil, ErrArtifact
	}
	data, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if err != nil || len(data) > 1<<20 {
		return nil, ErrArtifact
	}
	return data, nil
}

func (a *Artifact) Target() collection.ApplicationTarget { return a.target }
func (a *Artifact) Verify(ctx context.Context) error {
	if a.files["application-profile.json"] == nil {
		if _, err := os.Lstat(filepath.Join(a.root, "application-profile.json")); !errors.Is(err, os.ErrNotExist) {
			return ErrArtifact
		}
	}
	for _, name := range a.names {
		if err := ctx.Err(); err != nil {
			return err
		}
		data, err := a.read(name)
		if err != nil || !bytes.Equal(data, a.files[name]) {
			return ErrArtifact
		}
	}
	return nil
}

func (a *Artifact) Deliver(ctx context.Context, data []byte) ([]byte, error) {
	return a.run(ctx, data, false)
}
func (a *Artifact) Reconcile(ctx context.Context, data []byte) ([]byte, error) {
	return a.run(ctx, data, true)
}

func (a *Artifact) run(ctx context.Context, data []byte, reconcile bool) ([]byte, error) {
	if len(data) > 16384 {
		return nil, ErrDelivery
	}
	if err := a.Verify(ctx); err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "omabit-receiver-")
	if err != nil {
		return nil, ErrArtifact
	}
	defer os.RemoveAll(dir)
	for _, name := range a.names {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return nil, ErrArtifact
		}
		if err := os.WriteFile(path, a.files[name], 0400); err != nil {
			return nil, ErrArtifact
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	entry := filepath.Join(dir, a.entry)
	// -I -S disables environment/user/system site startup hooks. Only this
	// approved module directory is added to the standard-library search path.
	bootstrap := "import sys,runpy; p=sys.argv.pop(1); sys.path.insert(0,p); e=sys.argv.pop(1); sys.argv[0]=e; runpy.run_path(e,run_name='__main__')"
	args := []string{"-I", "-S", "-B", "-c", bootstrap, filepath.Dir(entry), entry}
	if reconcile {
		args = append(args, "--reconcile")
	}
	cmd := exec.CommandContext(ctx, "/usr/bin/python3", args...)
	cmd.Env = append([]string(nil), a.environment...)
	cmd.Dir = dir
	cmd.Stdin = bytes.NewReader(data)
	cmd.Stderr = io.Discard
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	cmd.WaitDelay = time.Second
	output := &receiverOutput{cancel: cancel}
	cmd.Stdout = output
	err = cmd.Start()
	if err == nil {
		// Keep the group leader unreaped until its descendants are terminated.
		// WNOWAIT reserves the PID, so cleanup cannot hit a reused process group.
		var info unix.Siginfo
		var waitErr error
		for {
			waitErr = unix.Waitid(unix.P_PID, cmd.Process.Pid, &info, unix.WEXITED|unix.WNOWAIT, nil)
			if !errors.Is(waitErr, unix.EINTR) {
				break
			}
		}
		if waitErr == nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		} else {
			_ = cmd.Process.Kill()
		}
		err = errors.Join(waitErr, cmd.Wait())
	}
	if err != nil || output.overflow {
		clear(output.Bytes())
		return nil, ErrDelivery // Never return arbitrary Python errors or output.
	}
	return output.Bytes(), nil
}

type receiverOutput struct {
	buffer   bytes.Buffer
	cancel   context.CancelFunc
	overflow bool
}

func (b *receiverOutput) Write(p []byte) (int, error) {
	if len(p) > 4096-b.Len() {
		b.overflow = true
		b.cancel()
		return 0, ErrDelivery
	}
	return b.buffer.Write(p)
}

// Named storage avoids promoting bytes.Buffer.ReadFrom: io.Copy would otherwise
// bypass Write and its output limit when copying the child's stdout pipe.
func (b *receiverOutput) Bytes() []byte { return b.buffer.Bytes() }
func (b *receiverOutput) Len() int      { return b.buffer.Len() }

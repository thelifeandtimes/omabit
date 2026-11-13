// Package collection implements per-user runner enrollment and inventory.
package collection

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

const maxConnections = 32
const maxRegistryBytes = 256 << 10

type Connection struct {
	Name       string            `json:"name"`
	Transport  string            `json:"transport"`
	Socket     string            `json:"socket,omitempty"`
	SSH        *client.SSHConfig `json:"ssh,omitempty"`
	RunnerID   string            `json:"runner_id"`
	Mode       string            `json:"mode"`
	EnrolledAt time.Time         `json:"enrolled_at"`
	Revision   uint64            `json:"revision,omitempty"`
}

func (c Connection) Identity() client.Identity {
	return client.Identity{RunnerID: c.RunnerID, Mode: c.Mode}
}

func (c Connection) Client(pinned bool) *client.Client {
	identity := c.Identity()
	if c.Transport == "ssh" && c.SSH != nil {
		if pinned {
			return client.NewSSH(*c.SSH, &identity)
		}
		return client.NewSSH(*c.SSH, nil)
	}
	if pinned {
		return client.NewPinned(c.Socket, identity)
	}
	return client.New(c.Socket)
}

func (c Connection) Address() string {
	if c.SSH != nil {
		return c.SSH.Host
	}
	return c.Socket
}

func (c Connection) endpointKey() string {
	if c.SSH != nil {
		return "ssh:" + c.SSH.Host + ":" + c.SSH.ConfigFile
	}
	return "unix:" + c.Socket
}

func ValidSocket(path string) bool {
	return filepath.IsAbs(path) && filepath.Clean(path) == path && len(path) <= 4096 && !strings.ContainsAny(path, "\x00\r\n")
}

func (c Connection) validate() error {
	if !protocol.ValidID(c.Name) || !protocol.ValidID(c.RunnerID) || !protocol.ValidID(c.Mode) || c.EnrolledAt.IsZero() {
		return errors.New("invalid runner enrollment")
	}
	if c.Transport == "unix" && ValidSocket(c.Socket) && c.SSH == nil {
		return nil
	}
	if c.Transport == "ssh" && c.Socket == "" && c.SSH != nil {
		return c.SSH.Validate()
	}
	return errors.New("invalid runner transport")
}

type registryFile struct {
	Version     int          `json:"version"`
	Connections []Connection `json:"connections"`
}

// Registry contains no credentials or runner-owned state. Open creates only its
// own private directory; existing unsafe paths and unknown formats are refused.
type Registry struct{ dir string }

func DefaultDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "urbit-collector"), nil
}

func Open(dir string) (*Registry, error) {
	if !filepath.IsAbs(dir) || filepath.Clean(dir) != dir {
		return nil, errors.New("collector config directory must be absolute and clean")
	}
	// Check existing ancestors before creating anything below them. In particular,
	// a symlinked config path must not create a directory at its target first.
	for parent := dir; ; parent = filepath.Dir(parent) {
		if _, err := os.Lstat(parent); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return nil, err
		}
		resolved, err := filepath.EvalSymlinks(parent)
		if err != nil || resolved != parent {
			return nil, errors.New("collector config directory must not contain symlinks")
		}
		break
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil || resolved != dir {
		return nil, errors.New("collector config directory must not contain symlinks")
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return nil, err
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || !info.IsDir() || info.Mode().Perm() != 0700 || st.Uid != uint32(os.Geteuid()) {
		return nil, errors.New("collector config directory must be owned by this account with mode 0700")
	}
	return &Registry{dir: dir}, nil
}

func privateFile(path string, flags int) (*os.File, error) {
	f, err := os.OpenFile(path, flags|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0600)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err == nil {
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || st.Uid != uint32(os.Geteuid()) || st.Nlink != 1 {
			err = errors.New("collector registry files must be private, owned regular files without extra links")
		}
	}
	if err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

func (r *Registry) locked(ctx context.Context, fn func() error) error {
	f, err := privateFile(filepath.Join(r.dir, "registry.lock"), os.O_RDWR|os.O_CREATE)
	if err != nil {
		return err
	}
	defer f.Close() // releases the advisory lock, including after a process exits
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("collector registry lock: %w", err)
		}
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return fn()
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EINTR) {
			return err
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("collector registry lock: %w", ctx.Err())
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func (r *Registry) read() (registryFile, error) {
	f, err := privateFile(filepath.Join(r.dir, "runners.json"), os.O_RDONLY)
	if errors.Is(err, os.ErrNotExist) {
		return registryFile{Version: 1, Connections: []Connection{}}, nil
	}
	if err != nil {
		return registryFile{}, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxRegistryBytes+1))
	if err != nil || len(data) > maxRegistryBytes {
		return registryFile{}, errors.New("cannot read bounded collector registry")
	}
	var state registryFile
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&state); err != nil {
		return state, errors.New("invalid collector registry; refusing to reset it")
	}
	if decoder.Decode(new(any)) != io.EOF || state.Version != 1 || state.Connections == nil || len(state.Connections) > maxConnections {
		return state, errors.New("unsupported or invalid collector registry")
	}
	names, ids, sockets := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for i, c := range state.Connections {
		// Existing version-1 registries predate connection editing.
		if c.Revision == 0 {
			c.Revision = 1
			state.Connections[i] = c
		}
		if err := c.validate(); err != nil {
			return state, err
		}
		if names[c.Name] || ids[c.RunnerID] || sockets[c.endpointKey()] {
			return state, errors.New("duplicate enrollment in collector registry")
		}
		names[c.Name], ids[c.RunnerID], sockets[c.endpointKey()] = true, true, true
	}
	return state, nil
}

func (r *Registry) write(state registryFile) error {
	slices.SortFunc(state.Connections, func(a, b Connection) int { return strings.Compare(a.Name, b.Name) })
	return r.writeFile("runners.json", state)
}

func (r *Registry) writeFile(name string, state any) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(r.dir, ".runners-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if _, err := f.Write(append(data, '\n')); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(f.Name(), filepath.Join(r.dir, name)); err != nil {
		return err
	}
	dir, err := os.Open(r.dir)
	if err != nil {
		return fmt.Errorf("registry replacement completed but directory sync failed: %w", err)
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		return fmt.Errorf("registry replacement completed but durability is uncertain: %w", err)
	}
	return nil
}

func (r *Registry) List(ctx context.Context) ([]Connection, error) {
	var state registryFile
	err := r.locked(ctx, func() (err error) { state, err = r.read(); return err })
	return state.Connections, err
}

func (r *Registry) Get(ctx context.Context, name string) (Connection, error) {
	all, err := r.List(ctx)
	if err != nil {
		return Connection{}, err
	}
	for _, c := range all {
		if c.Name == name {
			return c, nil
		}
	}
	return Connection{}, errors.New("unknown enrolled runner")
}

// Enroll requires a live compatible runner. Network IO happens outside the lock;
// the locked read/modify/rename prevents concurrent enrollment from losing data.
func (r *Registry) Enroll(ctx context.Context, name, socket string) (Connection, error) {
	if !protocol.ValidID(name) || !ValidSocket(socket) {
		return Connection{}, errors.New("enrollment requires a valid name and absolute clean socket path")
	}
	return r.enroll(ctx, Connection{Name: name, Transport: "unix", Socket: socket})
}

func (r *Registry) EnrollSSH(ctx context.Context, name string, config client.SSHConfig) (Connection, error) {
	if !protocol.ValidID(name) {
		return Connection{}, errors.New("invalid connection name")
	}
	if err := config.Validate(); err != nil {
		return Connection{}, err
	}
	return r.enroll(ctx, Connection{Name: name, Transport: "ssh", SSH: &config})
}

// EnrollSSHExpected binds first enrollment to identity returned by authorized
// local host setup, rather than writing an unpinned enrollment and checking later.
func (r *Registry) EnrollSSHExpected(ctx context.Context, name string, config client.SSHConfig, expected client.Identity) (Connection, error) {
	if !protocol.ValidID(name) || !protocol.ValidID(expected.RunnerID) || !protocol.ValidID(expected.Mode) {
		return Connection{}, errors.New("invalid expected runner enrollment")
	}
	if err := config.Validate(); err != nil {
		return Connection{}, err
	}
	return r.enrollExpected(ctx, Connection{Name: name, Transport: "ssh", SSH: &config}, &expected)
}

func (r *Registry) enroll(ctx context.Context, entry Connection) (Connection, error) {
	return r.enrollExpected(ctx, entry, nil)
}

func (r *Registry) enrollExpected(ctx context.Context, entry Connection, expected *client.Identity) (Connection, error) {
	if expected != nil {
		entry.RunnerID, entry.Mode = expected.RunnerID, expected.Mode
	}
	c := entry.Client(expected != nil)
	defer c.Close()
	hello, err := c.Hello(ctx)
	if err != nil {
		return Connection{}, err
	}
	entry.RunnerID, entry.Mode, entry.EnrolledAt = hello.RunnerID, hello.Mode, time.Now().UTC()
	entry.Revision = 1
	if err := entry.validate(); err != nil {
		return Connection{}, err
	}
	err = r.locked(ctx, func() error {
		state, err := r.read()
		if err != nil {
			return err
		}
		for _, old := range state.Connections {
			if old.Name == entry.Name && old.endpointKey() == entry.endpointKey() && old.Identity() == entry.Identity() {
				entry = old
				return nil
			}
			if old.Name == entry.Name || old.RunnerID == entry.RunnerID || old.endpointKey() == entry.endpointKey() {
				return errors.New("enrollment conflict; existing names, runner identities, and sockets are never overwritten")
			}
		}
		if len(state.Connections) >= maxConnections {
			return errors.New("collector enrollment limit reached (32 runners)")
		}
		state.Connections = append(state.Connections, entry)
		return r.write(state)
	})
	return entry, err
}

// Update verifies the proposed endpoint against the original identity before an
// atomic, revision-checked replacement. Network IO never holds the registry lock.
// A rename/rebind cannot transfer enrollment to a different runner or mode.
func (r *Registry) Update(ctx context.Context, name, expectedID string, revision uint64, proposed Connection) (Connection, error) {
	return r.update(ctx, name, expectedID, revision, proposed, "")
}

// AcceptDevelopmentMode explicitly changes only the mode at the same pinned
// endpoint; ordinary edits continue to refuse mode changes.
func (r *Registry) AcceptDevelopmentMode(ctx context.Context, name, expectedID string, revision uint64) (Connection, error) {
	old, err := r.Get(ctx, name)
	if err != nil {
		return Connection{}, err
	}
	if old.Mode != "development-managed-fake" && old.Mode != "development-managed" {
		return Connection{}, errors.New("only the fake-to-development mode transition is supported")
	}
	return r.update(ctx, name, expectedID, revision, old, "development-managed")
}

func (r *Registry) update(ctx context.Context, name, expectedID string, revision uint64, proposed Connection, acceptedMode string) (Connection, error) {
	if !protocol.ValidID(name) || !protocol.ValidID(expectedID) || revision == 0 || revision == math.MaxUint64 {
		return Connection{}, errors.New("connection update requires the selected name, identity and positive revision")
	}
	old, err := r.Get(ctx, name)
	if err != nil {
		return Connection{}, err
	}
	if old.RunnerID != expectedID {
		return Connection{}, client.ErrIdentityMismatch
	}
	if old.Revision != revision {
		return Connection{}, errors.New("connection revision conflict; reload enrollment before editing")
	}
	if acceptedMode != "" && (proposed.Name != old.Name || proposed.endpointKey() != old.endpointKey() || proposed.Mode != old.Mode || proposed.Revision != old.Revision || !proposed.EnrolledAt.Equal(old.EnrolledAt)) {
		return Connection{}, errors.New("connection changed before mode verification")
	}
	proposed.RunnerID, proposed.Mode, proposed.EnrolledAt, proposed.Revision = old.RunnerID, old.Mode, old.EnrolledAt, old.Revision
	if acceptedMode != "" {
		proposed.Mode = acceptedMode
	}
	if err := proposed.validate(); err != nil {
		return Connection{}, err
	}
	c := proposed.Client(true)
	defer c.Close()
	hello, err := c.Hello(ctx)
	if err == nil {
		err = proposed.Identity().Check(hello.RunnerID, hello.Mode)
	}
	if err != nil {
		return Connection{}, err
	}
	err = r.locked(ctx, func() error {
		state, err := r.read()
		if err != nil {
			return err
		}
		index := -1
		for i, current := range state.Connections {
			if current.Name == name {
				if current.RunnerID != expectedID || current.Revision != revision || !current.EnrolledAt.Equal(old.EnrolledAt) || current.endpointKey() != old.endpointKey() || current.Mode != old.Mode {
					return errors.New("connection changed during verification; reload enrollment before editing")
				}
				index = i
			} else if current.Name == proposed.Name || current.RunnerID == proposed.RunnerID || current.endpointKey() == proposed.endpointKey() {
				return errors.New("connection update conflicts with another enrollment")
			}
		}
		if index < 0 {
			return errors.New("connection was removed during verification")
		}
		if old.Name == proposed.Name && old.endpointKey() == proposed.endpointKey() && old.Mode == proposed.Mode {
			return nil // successful no-op does not churn revisions
		}
		proposed.Revision++
		state.Connections[index] = proposed
		return r.write(state)
	})
	if err != nil {
		return Connection{}, err
	}
	return proposed, nil
}

// Forget removes only local enrollment and works while the runner is offline.
// An expected ID prevents an old command from removing a changed enrollment.
func (r *Registry) Forget(ctx context.Context, name, expectedID string) error {
	return r.locked(ctx, func() error {
		state, err := r.read()
		if err != nil {
			return err
		}
		for i, c := range state.Connections {
			if c.Name != name {
				continue
			}
			if c.RunnerID != expectedID {
				return client.ErrIdentityMismatch
			}
			state.Connections = slices.Delete(state.Connections, i, i+1)
			return r.write(state)
		}
		return errors.New("unknown enrolled runner")
	})
}

// ForgetSSHExpected removes only the exact enrollment observed by credential
// revocation. A concurrent rename/rebind is preserved, as is an already absent
// entry after a lost acknowledgement. No network request is required.
func (r *Registry) ForgetSSHExpected(ctx context.Context, expected Connection) error {
	if expected.validate() != nil || expected.Transport != "ssh" || expected.Revision == 0 {
		return errors.New("require an exact SSH enrollment for credential revocation")
	}
	return r.locked(ctx, func() error {
		state, err := r.read()
		if err != nil {
			return err
		}
		for index, current := range state.Connections {
			if current.Name != expected.Name {
				continue
			}
			if current.RunnerID != expected.RunnerID || current.Revision != expected.Revision || current.Mode != expected.Mode || !current.EnrolledAt.Equal(expected.EnrolledAt) || current.endpointKey() != expected.endpointKey() {
				return nil
			}
			state.Connections = slices.Delete(state.Connections, index, index+1)
			return r.write(state)
		}
		return nil
	})
}

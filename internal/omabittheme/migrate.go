package omabittheme

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/credentials"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shipaccess"
)

const LegacyPlugin = "thelifeandtimes.urbit-theme"

type LegacyShip struct {
	ID                     string `json:"id"`
	Ship                   string `json:"ship"`
	URL                    string `json:"url"`
	Automatic              bool   `json:"automatic"`
	Pending                bool   `json:"pending"`
	AuthenticationRequired bool   `json:"authenticationRequired"`
	LastPublished          string `json:"lastPublished"`
	LastTheme              string `json:"lastTheme"`
	LastError              string `json:"lastError"`
	Fingerprint            string `json:"fingerprint"`
}
type LegacyState struct {
	Version int          `json:"version"`
	Ships   []LegacyShip `json:"ships"`
}
type Migration struct {
	SourceDigest string            `json:"source_digest"`
	PausedDigest string            `json:"paused_digest"`
	Original     LegacyState       `json:"original"`
	Imported     map[string]string `json:"imported"`
	Cleared      map[string]bool   `json:"cleared"`
	Complete     bool              `json:"complete"`
}
type MigrationPlan struct {
	Digest string       `json:"source_digest"`
	Ships  []LegacyShip `json:"ships"`
	Issues []string     `json:"issues"`
}

func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func LegacyDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		base = filepath.Join(home, ".local/state")
	}
	return filepath.Join(base, "omarchy-urbit-theme"), nil
}

func readLegacy(dir string) (LegacyState, []byte, error) {
	f, err := private(filepath.Join(dir, "state.json"), os.O_RDONLY)
	if err != nil {
		return LegacyState{}, nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 262145))
	if err != nil || len(data) > 262144 {
		return LegacyState{}, nil, errors.New("invalid bounded legacy state")
	}
	var state LegacyState
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if d.Decode(&state) != nil || d.Decode(new(any)) != io.EOF || state.Version != 2 || state.Ships == nil || len(state.Ships) > 32 {
		return state, nil, errors.New("migration requires a valid legacy version-2 state with at most 32 ships")
	}
	seen := map[string]bool{}
	for _, s := range state.Ships {
		id, e := hex.DecodeString(s.ID)
		origin, err := eyre.Origin(s.URL)
		if e != nil || len(id) != 32 || err != nil || origin != s.URL || !eyre.ValidShip(s.Ship) || seen[s.URL] || len(s.LastTheme) > 256 || len(s.LastError) > 512 {
			return state, nil, errors.New("legacy enrollment cannot be safely imported; review URL and identity")
		}
		seen[s.URL] = true
	}
	return state, data, nil
}
func PlanMigration(ctx context.Context, r *collection.Registry, dir string) (MigrationPlan, error) {
	state, data, err := readLegacy(dir)
	if err != nil {
		return MigrationPlan{}, err
	}
	plan := MigrationPlan{Digest: digest(data), Ships: state.Ships, Issues: []string{}}
	ships, err := r.Ships(ctx)
	if err != nil {
		return plan, err
	}
	if len(ships)+len(state.Ships) > 32 {
		plan.Issues = append(plan.Issues, "collector capacity would be exceeded")
	}
	for _, old := range state.Ships {
		for _, s := range ships {
			if old.URL == s.Origin || s.Name == "legacy-"+old.ID[:16] {
				plan.Issues = append(plan.Issues, "an origin or migration name is already enrolled")
			}
		}
	}
	return plan, nil
}

// MigrationBackend isolates privileged-in-context integration from fixture tests.
// It contains no generic runner policy. Apply is explicit and digest-bound.
type MigrationBackend interface {
	DisableLegacy(context.Context) error
	LookupLegacy(context.Context, string) ([]byte, error)
	ClearLegacy(context.Context, string) error
	Store(*collection.Registry, collection.Ship) (credentials.Store, error)
}
type DesktopMigration struct{}

func (DesktopMigration) Store(r *collection.Registry, s collection.Ship) (credentials.Store, error) {
	return shipaccess.Store(r, s)
}
func (DesktopMigration) DisableLegacy(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "omarchy", "plugin", "disable", LegacyPlugin)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if cmd.Run() != nil {
		return errors.New("could not disable the legacy publisher")
	}
	return nil
}
func (DesktopMigration) LookupLegacy(ctx context.Context, origin string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/usr/bin/secret-tool", "lookup", "application", "omarchy-urbit-theme", "origin", origin)
	cmd.Stderr = io.Discard
	var out limitedOutput
	cmd.Stdout = &out
	if cmd.Run() != nil {
		return nil, errors.New("legacy keyring session unavailable; migration remains paused")
	}
	return out.Bytes(), nil
}
func (DesktopMigration) ClearLegacy(ctx context.Context, origin string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/usr/bin/secret-tool", "clear", "application", "omarchy-urbit-theme", "origin", origin)
	cmd.Stdout = io.Discard
	var stderr limitedOutput
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() == 1 && stderr.Len() == 0 && ctx.Err() == nil {
			return nil
		}
		return errors.New("legacy keyring deletion was not confirmed")
	}
	return nil
}

type limitedOutput struct{ bytes.Buffer }

func (b *limitedOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 16384 {
		return 0, errors.New("bounded credential output exceeded")
	}
	return b.Buffer.Write(p)
}

func legacyCredential(data []byte, ship LegacyShip) ([]byte, error) {
	defer clear(data)
	var old struct {
		URL   string `json:"url"`
		Ship  string `json:"ship"`
		Name  string `json:"cookieName"`
		Value string `json:"cookieValue"`
	}
	if json.Unmarshal(data, &old) != nil || old.URL != ship.URL || old.Ship != ship.Ship || old.Name != "urbauth-"+ship.Ship || old.Value == "" {
		return nil, errors.New("legacy credential does not match enrollment")
	}
	// The legacy schema has no expiry. The imported session is rechecked with
	// Eyre and gets a conservative local one-day deadline, never a fabricated age.
	return json.Marshal(map[string]any{"origin": old.URL, "ship": old.Ship, "cookie_name": old.Name, "cookie_value": old.Value, "expires": time.Now().UTC().Add(24 * time.Hour)})
}

func migratedPreference(source LegacyShip, entry collection.Ship) Preference {
	return Preference{ID: entry.ID, Name: entry.Name, Origin: entry.Origin, Ship: entry.Ship, Automatic: source.Automatic, Pending: source.Pending, Revision: 1, Fingerprint: source.Fingerprint, LastTheme: source.LastTheme, LastPublished: source.LastPublished, LastError: source.LastError, AuthenticationRequired: source.AuthenticationRequired}
}

// Check the largest expected journal (all imported preferences and progress
// entries) before disabling the old publisher or copying/clearing credentials.
// The final Update guard also protects against concurrent preference changes.
func preflightMigration(state *State) error {
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	var projected State
	if err := json.Unmarshal(raw, &projected); err != nil {
		return err
	}
	m := projected.Migration
	for n, source := range m.Original.Ships {
		id := m.Imported[source.ID]
		if id == "" {
			// Reserve a unique placeholder of the actual enrollment ID's length.
			for i := n + 1; ; i++ {
				id = fmt.Sprintf("%032x", i)
				if _, exists := projected.Ships[id]; !exists {
					break
				}
			}
		}
		if _, ok := projected.Ships[id]; !ok {
			entry := collection.Ship{ID: id, Name: "legacy-" + source.ID[:16], Origin: source.URL, Ship: source.Ship}
			projected.Ships[id] = migratedPreference(source, entry)
		}
		m.Imported[source.ID] = id
		m.Cleared[source.ID] = false // longer encoding than the eventual true
	}
	_, err = encodeState(&projected)
	return err
}

func ApplyMigration(ctx context.Context, r *collection.Registry, store *Store, dir, expected string, backend MigrationBackend) error {
	// Verify the legacy directory itself without changing its permissions.
	info, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	owner, ok := info.Sys().(*syscall.Stat_t)
	resolved, e := filepath.EvalSymlinks(dir)
	if !ok || !info.IsDir() || info.Mode().Perm() != 0700 || owner.Uid != uint32(os.Geteuid()) || e != nil || resolved != dir {
		return errors.New("legacy state directory must be private, owned, and non-symlinked")
	}
	// The original helper uses this exact lock; never race an in-flight publish.
	lock, err := private(filepath.Join(dir, "lock"), os.O_RDWR|os.O_CREATE)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err := lockLegacy(lock); err != nil {
		return err
	}
	old, data, err := readLegacy(dir)
	if err != nil {
		return err
	}
	if err := store.PruneForgotten(ctx, r); err != nil {
		return err
	}
	state, err := store.Snapshot(ctx)
	if err != nil {
		return err
	}
	if state.Migration == nil {
		plan, err := PlanMigration(ctx, r, dir)
		if err != nil {
			return err
		}
		if expected == "" || expected != plan.Digest || len(plan.Issues) > 0 {
			return errors.New("migration preview changed or has conflicts")
		}
		paused := old
		paused.Ships = append([]LegacyShip(nil), old.Ships...)
		for i := range paused.Ships {
			paused.Ships[i].Automatic = false
			paused.Ships[i].Pending = false
		}
		pausedData, _ := json.Marshal(paused)
		m := &Migration{SourceDigest: expected, PausedDigest: digest(pausedData), Original: old, Imported: map[string]string{}, Cleared: map[string]bool{}}
		if err := store.Update(ctx, func(s *State) error {
			if s.Migration != nil {
				return errors.New("another migration already exists")
			}
			s.Migration = m
			return preflightMigration(s)
		}); err != nil {
			return err
		}
		state.Migration = m
	}
	m := state.Migration
	if expected != m.SourceDigest {
		return errors.New("migration digest does not match journal")
	}
	if m.Complete {
		return nil
	}
	if digest(data) != m.SourceDigest && digest(data) != m.PausedDigest {
		return errors.New("legacy state changed after migration began")
	}
	if err := preflightMigration(&state); err != nil {
		return err
	}
	if err := backend.DisableLegacy(ctx); err != nil {
		return err
	}
	if digest(data) == m.SourceDigest {
		paused := m.Original
		paused.Ships = append([]LegacyShip(nil), m.Original.Ships...)
		for i := range paused.Ships {
			paused.Ships[i].Automatic = false
			paused.Ships[i].Pending = false
		}
		data, _ := json.Marshal(paused)
		if err := replaceLegacy(dir, data); err != nil {
			return err
		}
	}
	for _, source := range m.Original.Ships {
		name := "legacy-" + source.ID[:16]
		entry, err := r.EnrollShip(ctx, name, source.URL, source.Ship, "secret-service")
		if err != nil {
			return err
		}
		if err := store.Update(ctx, func(s *State) error {
			prior := s.Migration.Imported[source.ID]
			if prior != "" && prior != entry.ID {
				return errors.New("imported enrollment identity changed")
			}
			s.Migration.Imported[source.ID] = entry.ID
			return nil
		}); err != nil {
			return err
		}
		if entry.SessionRef == "" {
			raw, err := backend.LookupLegacy(ctx, source.URL)
			if err != nil {
				return err
			}
			secret, err := legacyCredential(raw, source)
			if err != nil {
				return err
			}
			client, err := eyre.New(entry.Origin, entry.Ship)
			if err != nil {
				clear(secret)
				return err
			}
			err = client.RestoreSession(secret)
			clear(secret)
			if err == nil {
				err = client.Check(ctx)
			}
			if err != nil {
				client.Close()
				return errors.New("legacy session rejected; sign in to the imported connection, then resume migration")
			}
			encoded, err := client.ExportSession()
			client.Close()
			if err != nil {
				return err
			}
			credentials, err := backend.Store(r, entry)
			if err != nil {
				clear(encoded)
				return err
			}
			ref := collection.NewSessionRef()
			copy := entry
			copy.SessionRef = ref
			entry, err = r.SaveShipSession(ctx, entry, ref, func(save context.Context) error {
				return credentials.Save(save, shipaccess.Key(copy), encoded)
			})
			clear(encoded)
			if err != nil {
				return err
			}
		}
		// Interrupted copies can reference the same remote cookie as the current
		// session. Retire those local copies without revoking the shared session.
		if len(entry.Retired) > 0 {
			credentialStore, err := backend.Store(r, entry)
			if err != nil {
				return err
			}
			for _, ref := range entry.Retired {
				copy := entry
				copy.SessionRef = ref
				local, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				err := credentialStore.Delete(local, shipaccess.Key(copy))
				cancel()
				if err != nil {
					return err
				}
				finish, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				err = r.CompleteCredentialCleanup(finish, entry.ID, []string{ref})
				cancel()
				if err != nil {
					return err
				}
			}
		}
		if err := store.Update(ctx, func(s *State) error {
			if previous := s.Migration.Imported[source.ID]; previous != "" && previous != entry.ID {
				return errors.New("imported enrollment identity changed")
			}
			if _, ok := s.Ships[entry.ID]; !ok {
				s.Ships[entry.ID] = migratedPreference(source, entry)
			}
			s.Migration.Imported[source.ID] = entry.ID
			return nil
		}); err != nil {
			return err
		}
		if !m.Cleared[source.ID] {
			if err := backend.ClearLegacy(ctx, source.URL); err != nil {
				return err
			}
			if err := store.Update(ctx, func(s *State) error { s.Migration.Cleared[source.ID] = true; return nil }); err != nil {
				return err
			}
		}
	}
	return store.Update(ctx, func(s *State) error { s.Migration.Complete = true; return nil })
}

func replaceLegacy(dir string, data []byte) error {
	f, err := os.CreateTemp(dir, ".omabit-migrate-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(f.Name(), filepath.Join(dir, "state.json")); err != nil {
		return err
	}
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

func lockLegacy(f *os.File) error {
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return errors.New("legacy publisher is busy; wait for its operation to finish")
	}
	return nil
}

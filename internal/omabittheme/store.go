package omabittheme

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
)

type Preference struct {
	ID                     string `json:"id"`
	Name                   string `json:"name"`
	Origin                 string `json:"origin"`
	Ship                   string `json:"ship"`
	Automatic              bool   `json:"automatic"`
	Pending                bool   `json:"pending"`
	Revision               uint64 `json:"revision"`
	Fingerprint            string `json:"fingerprint,omitempty"`
	LastTheme              string `json:"last_theme,omitempty"`
	LastPublished          string `json:"last_published,omitempty"`
	LastError              string `json:"last_error,omitempty"`
	AuthenticationRequired bool   `json:"authentication_required,omitempty"`
	Attempts               int    `json:"attempts,omitempty"`
	AttemptFingerprint     string `json:"attempt_fingerprint,omitempty"`
}
type State struct {
	Version   int                   `json:"version"`
	Ships     map[string]Preference `json:"ships"`
	Migration *Migration            `json:"migration,omitempty"`
}
type Store struct{ dir string }

func OpenStore(dir string) (*Store, error) {
	// Reuse the collector's private-directory policy, but keep all Omarchy-specific
	// preferences in a separate directory and schema, never its runner database.
	if _, err := collection.Open(dir); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}
func DefaultDir() (string, error) {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local/state")
	}
	if !filepath.IsAbs(base) {
		return "", errors.New("state directory must be absolute")
	}
	return filepath.Join(base, "omabit"), nil
}
func private(path string, flags int) (*os.File, error) {
	f, err := os.OpenFile(path, flags|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0600)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || st.Uid != uint32(os.Geteuid()) || st.Nlink != 1 {
		f.Close()
		return nil, errors.New("unsafe Omabit state file")
	}
	return f, nil
}
func (s *Store) Update(ctx context.Context, fn func(*State) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	lock, err := private(filepath.Join(s.dir, "theme.lock"), os.O_RDWR|os.O_CREATE)
	if err != nil {
		return err
	}
	defer lock.Close()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	for {
		if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err == nil {
			break
		} else if !errors.Is(err, syscall.EWOULDBLOCK) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}
	state := State{Version: 1, Ships: map[string]Preference{}}
	f, err := private(filepath.Join(s.dir, "themes.json"), os.O_RDONLY)
	if err == nil {
		data, e := io.ReadAll(io.LimitReader(f, 262145))
		f.Close()
		if e != nil || len(data) > 262144 {
			return errors.New("invalid bounded Omabit state")
		}
		d := json.NewDecoder(bytes.NewReader(data))
		d.DisallowUnknownFields()
		if d.Decode(&state) != nil || d.Decode(new(any)) != io.EOF || state.Version != 1 || state.Ships == nil || len(state.Ships) > 64 {
			return errors.New("invalid Omabit state; refusing reset")
		}
		for id, p := range state.Ships {
			if id != p.ID || len(id) != 32 || p.Revision == 0 || len(p.LastError) > 512 {
				return errors.New("invalid Omabit preference")
			}
		}
		if m := state.Migration; m != nil {
			if len(m.SourceDigest) != 64 || len(m.PausedDigest) != 64 || m.Imported == nil || m.Cleared == nil || m.Original.Version != 2 {
				return errors.New("invalid migration recovery journal")
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	before, _ := json.Marshal(state)
	if err := fn(&state); err != nil {
		return err
	}
	after, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	compact, _ := json.Marshal(state)
	if bytes.Equal(before, compact) {
		return nil
	}
	f, err = os.CreateTemp(s.dir, ".theme-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if _, err = f.Write(after); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(f.Name(), filepath.Join(s.dir, "themes.json")); err != nil {
		return err
	}
	directory, err := os.Open(s.dir)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
func (s *Store) Snapshot(ctx context.Context) (State, error) {
	var result State
	err := s.Update(ctx, func(state *State) error { result = *state; return nil })
	return result, err
}
func (s *Store) Auto(ctx context.Context, ship collection.Ship, enabled bool) error {
	return s.Update(ctx, func(state *State) error {
		p := state.Ships[ship.ID]
		if p.ID == "" {
			if len(state.Ships) >= 64 {
				return errors.New("theme preference capacity reached")
			}
			p = Preference{ID: ship.ID, Name: ship.Name, Origin: ship.Origin, Ship: ship.Ship}
		}
		if p.Origin != ship.Origin || p.Ship != ship.Ship {
			return collection.ErrShipConflict
		}
		p.Automatic = enabled
		p.Pending = enabled
		p.Revision++
		p.LastError = ""
		p.Attempts = 0
		if enabled {
			p.AuthenticationRequired = false
		}
		state.Ships[p.ID] = p
		return nil
	})
}

var ErrBusy = errors.New("another publication for this ship is already running")

func (s *Store) LockShip(id string) (func(), error) {
	if len(id) != 32 {
		return nil, errors.New("invalid ship identity")
	}
	for _, c := range id {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return nil, errors.New("invalid ship identity")
		}
	}
	f, err := private(filepath.Join(s.dir, "publish-"+id+".lock"), os.O_RDWR|os.O_CREATE)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrBusy
		}
		return nil, err
	}
	return func() { f.Close() }, nil
}

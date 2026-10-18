package collection

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// Ship is application enrollment, independent of any runner-managed instance.
// SessionRef is an opaque credential-store reference, never a session token.
type Ship struct {
	Name        string `json:"name"`
	ID          string `json:"id"`
	Origin      string `json:"origin"`
	Ship        string `json:"ship"`
	Credentials string `json:"credentials"`
	Revision    uint64 `json:"revision"`
	SessionRef  string `json:"session_ref,omitempty"`
	// Retired also journals a newly allocated reference before credential Save.
	// It is removed only by atomic promotion to SessionRef or confirmed cleanup.
	Retired    []string  `json:"retired_sessions,omitempty"`
	EnrolledAt time.Time `json:"enrolled_at"`
}
type shipsFile struct {
	Version int    `json:"version"`
	Ships   []Ship `json:"ships"`
}

var ErrShipConflict = errors.New("ship enrollment or session changed; inspect current state before retrying")

func NewSessionRef() string {
	var value [16]byte
	_, err := rand.Read(value[:])
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(value[:])
}
func validRef(s string) bool {
	b, err := hex.DecodeString(s)
	return err == nil && len(b) == 16 && s == strings.ToLower(s)
}

func (r *Registry) readShips() (shipsFile, error) {
	f, err := privateFile(filepath.Join(r.dir, "ships.json"), os.O_RDONLY)
	if errors.Is(err, os.ErrNotExist) {
		return shipsFile{Version: 1, Ships: []Ship{}}, nil
	}
	if err != nil {
		return shipsFile{}, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxRegistryBytes+1))
	if err != nil || len(data) > maxRegistryBytes {
		return shipsFile{}, errors.New("cannot read bounded ship registry")
	}
	var state shipsFile
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if d.Decode(&state) != nil || d.Decode(new(any)) != io.EOF || state.Version != 1 || state.Ships == nil || len(state.Ships) > maxConnections {
		return state, errors.New("invalid ship registry; refusing to reset it")
	}
	names, ids, origins := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, s := range state.Ships {
		origin, err := eyre.Origin(s.Origin)
		if err != nil || origin != s.Origin || !protocol.ValidID(s.Name) || !validRef(s.ID) || !eyre.ValidShip(s.Ship) || (s.Credentials != "secret-service" && s.Credentials != "file") || s.Revision == 0 || s.EnrolledAt.IsZero() || (s.SessionRef != "" && !validRef(s.SessionRef)) || names[s.Name] || ids[s.ID] || origins[s.Origin] {
			return state, errors.New("invalid or duplicate ship enrollment")
		}
		if len(s.Retired) > 32 {
			return state, errors.New("too many pending credential cleanups")
		}
		refs := map[string]bool{}
		for _, ref := range s.Retired {
			if !validRef(ref) || ref == s.SessionRef || refs[ref] {
				return state, errors.New("invalid retired session reference")
			}
			refs[ref] = true
		}
		names[s.Name], ids[s.ID], origins[s.Origin] = true, true, true
	}
	return state, nil
}

func (r *Registry) Ships(ctx context.Context) ([]Ship, error) {
	var state shipsFile
	err := r.locked(ctx, func() (err error) { state, err = r.readShips(); return err })
	return state.Ships, err
}
func (r *Registry) Ship(ctx context.Context, name string) (Ship, error) {
	all, err := r.Ships(ctx)
	if err != nil {
		return Ship{}, err
	}
	for _, s := range all {
		if s.Name == name {
			return s, nil
		}
	}
	return Ship{}, errors.New("unknown enrolled ship")
}

func (r *Registry) EnrollShip(ctx context.Context, name, rawOrigin, ship, credentials string) (Ship, error) {
	if !protocol.ValidID(name) || (credentials != "secret-service" && credentials != "file") {
		return Ship{}, errors.New("invalid ship name or credential adapter")
	}
	origin, err := eyre.Origin(rawOrigin)
	if err != nil {
		return Ship{}, err
	}
	c, err := eyre.New(origin, ship)
	if err != nil {
		return Ship{}, err
	}
	defer c.Close()
	if err := c.Probe(ctx); err != nil {
		return Ship{}, err
	}
	entry := Ship{Name: name, ID: NewSessionRef(), Origin: origin, Ship: ship, Credentials: credentials, Revision: 1, EnrolledAt: time.Now().UTC()}
	err = r.locked(ctx, func() error {
		state, err := r.readShips()
		if err != nil {
			return err
		}
		for _, old := range state.Ships {
			if old.Name == name && old.Origin == origin && old.Ship == ship && old.Credentials == credentials {
				entry = old
				return nil
			}
			if old.Name == name || old.Origin == origin {
				return ErrShipConflict
			}
		}
		if len(state.Ships) >= maxConnections {
			return errors.New("ship enrollment limit reached (32)")
		}
		state.Ships = append(state.Ships, entry)
		slices.SortFunc(state.Ships, func(a, b Ship) int { return strings.Compare(a.Name, b.Name) })
		return r.writeFile("ships.json", state)
	})
	return entry, err
}

// SetShipSession uses CAS so a slow login/logout cannot overwrite a newer one.
// New credentials must use SaveShipSession to journal their reference before IO.
// Watchers never write credentials back, so they cannot resurrect a forgotten or
// replaced session.
func (r *Registry) SetShipSession(ctx context.Context, expected Ship, ref string) (Ship, error) {
	if ref != "" && !validRef(ref) {
		return Ship{}, errors.New("invalid credential reference")
	}
	var result Ship
	err := r.locked(ctx, func() error {
		state, err := r.readShips()
		if err != nil {
			return err
		}
		for i, s := range state.Ships {
			if s.Name != expected.Name {
				continue
			}
			if s.ID != expected.ID || s.Revision != expected.Revision || s.Revision == ^uint64(0) {
				return ErrShipConflict
			}
			if ref == s.SessionRef || slices.Contains(s.Retired, ref) {
				return ErrShipConflict
			}
			if s.SessionRef != "" {
				if len(s.Retired) >= 32 {
					return errors.New("clear retired credentials before logging in again")
				}
				s.Retired = append(s.Retired, s.SessionRef)
			}
			s.SessionRef = ref
			s.Revision++
			state.Ships[i] = s
			result = s
			return r.writeFile("ships.json", state)
		}
		return ErrShipConflict
	})
	return result, err
}

// SaveShipSession durably records cleanup intent BEFORE save can persist a token.
// The existing retired_sessions schema already represents this intent, including
// an absent credential, so version-1 readers remain safe. Keep the registry lock
// across bounded local credential IO and publication: cleanup/forget cannot see
// the pending ref, delete it, then race a late Save. HTTP never holds this lock.
// A failed/abandoned save leaves the reference available to ship-logout on restart.
func (r *Registry) SaveShipSession(ctx context.Context, expected Ship, ref string, save func(context.Context) error) (Ship, error) {
	if !validRef(ref) || save == nil {
		return Ship{}, errors.New("invalid credential save intent")
	}
	var result Ship
	err := r.locked(ctx, func() error {
		state, err := r.readShips()
		if err != nil {
			return err
		}
		for i, s := range state.Ships {
			if s.Name != expected.Name {
				continue
			}
			if s.ID != expected.ID || s.Revision != expected.Revision || s.Revision == ^uint64(0) || ref == s.SessionRef || slices.Contains(s.Retired, ref) {
				return ErrShipConflict
			}
			if len(s.Retired) >= 32 {
				return errors.New("clear retired credentials before logging in again")
			}
			s.Retired = append(s.Retired, ref)
			state.Ships[i] = s
			if err := r.writeFile("ships.json", state); err != nil {
				return err
			}
			bounded, cancel := context.WithTimeout(ctx, 15*time.Second)
			err = save(bounded)
			cancel()
			if err != nil {
				return err // the credential may already exist; keep retry intent
			}
			s.Retired = slices.Delete(s.Retired, len(s.Retired)-1, len(s.Retired))
			if s.SessionRef != "" {
				s.Retired = append(s.Retired, s.SessionRef)
			}
			s.SessionRef = ref
			s.Revision++
			state.Ships[i] = s
			result = s
			return r.writeFile("ships.json", state)
		}
		return ErrShipConflict
	})
	return result, err
}

func (r *Registry) ForgetShip(ctx context.Context, name, expectedOrigin string) error {
	return r.ForgetShipExpected(ctx, name, expectedOrigin, "")
}

func (r *Registry) ForgetShipExpected(ctx context.Context, name, expectedOrigin, expectedID string) error {
	return r.locked(ctx, func() error {
		state, err := r.readShips()
		if err != nil {
			return err
		}
		for i, s := range state.Ships {
			if s.Name != name {
				continue
			}
			if s.Origin != expectedOrigin || (expectedID != "" && s.ID != expectedID) {
				return ErrShipConflict
			}
			if s.SessionRef != "" || len(s.Retired) > 0 {
				return errors.New("log out the ship before forgetting enrollment")
			}
			state.Ships = slices.Delete(state.Ships, i, i+1)
			return r.writeFile("ships.json", state)
		}
		return errors.New("unknown enrolled ship")
	})
}

func (r *Registry) CredentialsDir() string { return filepath.Join(r.dir, "credentials") }

// CompleteCredentialCleanup removes only known retired references. It never
// clears a newer current session or loses a concurrent retirement's retry intent.
func (r *Registry) CompleteCredentialCleanup(ctx context.Context, id string, refs []string) error {
	return r.locked(ctx, func() error {
		state, err := r.readShips()
		if err != nil {
			return err
		}
		for i, s := range state.Ships {
			if s.ID != id {
				continue
			}
			s.Retired = slices.DeleteFunc(s.Retired, func(ref string) bool { return slices.Contains(refs, ref) })
			state.Ships[i] = s
			return r.writeFile("ships.json", state)
		}
		return ErrShipConflict
	})
}

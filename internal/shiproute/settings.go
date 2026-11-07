package shiproute

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"sort"
	"time"
	"unicode"
	"unicode/utf8"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
)

// Settings retain user choices, not a snapshot of machine discovery. Automatic
// addresses are recomputed from the current verified listener/network on reads.
// Disabling one does not leave an old enabled copy after a port/network change.
type Settings struct {
	Routes            []Route  `json:"routes"`
	DisabledAutomatic []string `json:"disabled_automatic"`
}

func DefaultSettings() Settings { return Settings{Routes: []Route{}, DisabledAutomatic: []string{}} }
func automaticID(s string) bool {
	return s == "managed-http" || s == "tailscale-http" || s == "tailscale-serve"
}
func validAutomatic(r Route) bool {
	return r.ID == "managed-http" && r.Source == "managed" && r.Network == "local" ||
		r.ID == "tailscale-http" && r.Source == "tailscale" && r.Network == "tailnet" ||
		r.ID == "tailscale-serve" && r.Source == "tailscale" && (r.Network == "tailnet" || r.Network == "public")
}
func (s Settings) Validate() error {
	if s.Routes == nil || s.DisabledAutomatic == nil || len(s.Routes) > MaxRoutes-3 || len(s.DisabledAutomatic) > 3 {
		return errors.New("invalid route settings")
	}
	origins := map[string]bool{}
	for i, r := range s.Routes {
		if r.Validate() != nil || r.Source != "manual" && r.Source != "tunnel" || automaticID(r.ID) || origins[r.Origin] || i > 0 && s.Routes[i-1].ID >= r.ID {
			return errors.New("invalid, duplicate or reserved configured route")
		}
		origins[r.Origin] = true
	}
	for i, v := range s.DisabledAutomatic {
		if !automaticID(v) || i > 0 && s.DisabledAutomatic[i-1] >= v {
			return errors.New("invalid automatic route disable selection")
		}
	}
	return nil
}
func (s *Settings) Normalize() error {
	// Own the slices so normalizing a request cannot mutate retained history.
	s.Routes = append([]Route{}, s.Routes...)
	s.DisabledAutomatic = append([]string{}, s.DisabledAutomatic...)
	sort.Slice(s.Routes, func(i, j int) bool { return s.Routes[i].ID < s.Routes[j].ID })
	sort.Strings(s.DisabledAutomatic)
	return s.Validate()
}
func (s Settings) Digest() string {
	b, _ := json.Marshal(s)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func (b Binding) Key() string {
	v, _ := json.Marshal(b)
	h := sha256.Sum256(v)
	return hex.EncodeToString(h[:])
}

// Effective preserves explicit URL choices over duplicate discovered URLs.
// Neither a configured nor a discovered route is a reachability/identity claim.
func (s Settings) Effective(binding Binding, revision uint64, automatic []Route) (Registry, error) {
	out := Registry{Version: 1, Revision: revision, Binding: binding, Routes: append([]Route{}, s.Routes...)}
	if s.Validate() != nil || binding.Validate() != nil || revision > apppolicy.MaxRevision || len(automatic) > 3 {
		return Registry{}, errors.New("invalid route configuration or discovery")
	}
	seenIDs, seenOrigins := map[string]bool{}, map[string]bool{}
	for _, r := range s.Routes {
		seenOrigins[r.Origin] = true
	}
	for _, r := range automatic {
		if r.Validate() != nil || !validAutomatic(r) || seenIDs[r.ID] {
			return Registry{}, errors.New("invalid automatic route discovery")
		}
		seenIDs[r.ID] = true
		if seenOrigins[r.Origin] {
			continue
		}
		seenOrigins[r.Origin] = true
		r.Enabled = !slices.Contains(s.DisabledAutomatic, r.ID)
		out.Routes = append(out.Routes, r)
	}
	sort.Slice(out.Routes, func(i, j int) bool { return out.Routes[i].ID < out.Routes[j].ID })
	return out, out.Validate()
}

// Record is one immutable revision. The runner writes it and its current pointer
// transactionally; old ship incarnations retain their own history and are never
// adopted by a replacement that happens to have the same name or URL.
type Record struct {
	Version   int       `json:"version"`
	Binding   Binding   `json:"binding"`
	Revision  uint64    `json:"revision"`
	Settings  Settings  `json:"settings"`
	Actor     string    `json:"actor"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (r Record) Validate() error {
	if r.Version != 1 || r.Binding.Validate() != nil || r.Settings.Validate() != nil || r.Revision > apppolicy.MaxRevision {
		return errors.New("invalid route record")
	}
	if r.Revision == 0 {
		if r.Actor != "" || !r.UpdatedAt.IsZero() || r.Settings.Digest() != DefaultSettings().Digest() {
			return errors.New("invalid default route record")
		}
		return nil
	}
	if r.Actor == "" || len(r.Actor) > 128 || !utf8.ValidString(r.Actor) || r.UpdatedAt.IsZero() {
		return errors.New("route change lacks attribution")
	}
	for _, c := range r.Actor {
		if unicode.IsControl(c) {
			return errors.New("invalid route change actor")
		}
	}
	return nil
}
func DefaultRecord(binding Binding) Record {
	return Record{Version: 1, Binding: binding, Settings: DefaultSettings()}
}

type Update struct {
	Binding          Binding  `json:"binding"`
	ExpectedRevision uint64   `json:"expected_revision"`
	Settings         Settings `json:"settings"`
}

func (u Update) Validate() error {
	if u.Binding.Validate() != nil || u.ExpectedRevision >= apppolicy.MaxRevision || u.Settings.Validate() != nil {
		return errors.New("invalid route settings update")
	}
	return nil
}

// Package shiproute selects registered routes to one fleet ship. A successful
// probe is a reachability observation, never a session or permission grant.
package shiproute

import (
	"context"
	"errors"
	"net/netip"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
)

const MaxRoutes = 32

var id = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
var digest = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Binding deliberately includes a project member's generation. Two independent
// fake ships named ~zod must never share routes, approval, or credentials.
type Binding struct {
	RunnerID   string           `json:"runner_id"`
	Target     apppolicy.Target `json:"target"`
	Ship       string           `json:"ship"`
	Continuity string           `json:"continuity"` // verified pier identity, survives process restart
}

func (b Binding) Validate() error {
	if !id.MatchString(b.RunnerID) || !digest.MatchString(b.Continuity) || b.Target.Validate() != nil || b.Target.InstanceID == "" || b.Target.Desk != "" || b.Target.ProjectID != "" && b.Target.Generation == 0 || !eyre.ValidShip(b.Ship) {
		return errors.New("invalid route ship binding")
	}
	return nil
}

type Route struct {
	ID      string `json:"id"`
	Origin  string `json:"origin"`
	Network string `json:"network"` // local, tailnet, lan, public
	Scope   string `json:"scope"`   // host ID, tailnet DNS suffix, LAN profile ID, or empty
	Source  string `json:"source"`  // managed, tailscale, manual, tunnel
	Enabled bool   `json:"enabled"`
}

func (r Route) Validate() error {
	origin, err := eyre.Origin(r.Origin)
	if err != nil || origin != r.Origin || !id.MatchString(r.ID) {
		return errors.New("invalid route origin or ID")
	}
	u, _ := url.Parse(origin)
	addr, _ := netip.ParseAddr(u.Hostname())
	loopback := u.Hostname() == "localhost" || addr.IsValid() && addr.IsLoopback()
	switch r.Network {
	case "local":
		if !loopback || !id.MatchString(r.Scope) {
			return errors.New("local route requires an exact host scope and loopback origin")
		}
	case "tailnet":
		if !validTailnet(r.Scope) || !strings.HasSuffix(u.Hostname(), "."+r.Scope) || loopback {
			return errors.New("tailnet route requires its full MagicDNS name")
		}
	case "lan":
		if !id.MatchString(r.Scope) || loopback {
			return errors.New("LAN route requires an explicit network profile")
		}
	case "public":
		if r.Scope != "" || u.Scheme != "https" || loopback || addr.IsValid() && (addr.IsPrivate() || addr.IsLinkLocalUnicast()) {
			return errors.New("automatic public routes require HTTPS")
		}
	default:
		return errors.New("unknown route network")
	}
	switch r.Source {
	case "managed", "tailscale", "manual", "tunnel":
	default:
		return errors.New("unknown route source")
	}
	if r.Source == "tailscale" && r.Network != "tailnet" && r.Network != "public" || r.Source == "managed" && r.Network != "local" {
		return errors.New("route source and network disagree")
	}
	return nil
}

func validTailnet(s string) bool {
	if !strings.HasSuffix(s, ".ts.net") || strings.Count(s, ".") != 2 {
		return false
	}
	o, err := eyre.Origin("https://" + s)
	return err == nil && o == "https://"+s
}

type Registry struct {
	Version  int     `json:"version"`
	Revision uint64  `json:"revision"`
	Binding  Binding `json:"binding"`
	Routes   []Route `json:"routes"`
}

func (r Registry) Validate() error {
	if r.Version != 1 || r.Revision > apppolicy.MaxRevision || r.Binding.Validate() != nil || r.Routes == nil || len(r.Routes) > MaxRoutes {
		return errors.New("invalid ship route registry")
	}
	ids, origins := map[string]bool{}, map[string]bool{}
	for _, route := range r.Routes {
		if route.Validate() != nil || ids[route.ID] || origins[route.Origin] {
			return errors.New("invalid or duplicate ship route")
		}
		ids[route.ID], origins[route.Origin] = true, true
	}
	return nil
}

// Environment is the collector's current local network observation. An enrolled
// remote runner is not proof that its localhost belongs to this collector.
type Environment struct {
	HostID        string
	TailnetSuffix string
	LANProfiles   []string
}

func (r Registry) Candidates(env Environment) ([]Route, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	out := []Route{}
	for _, route := range r.Routes {
		if !route.Enabled {
			continue
		}
		eligible := route.Network == "public" || route.Network == "local" && route.Scope == env.HostID || route.Network == "tailnet" && route.Scope == env.TailnetSuffix
		if route.Network == "lan" {
			for _, network := range env.LANProfiles {
				eligible = eligible || route.Scope == network
			}
		}
		if eligible {
			out = append(out, route)
		}
	}
	rank := map[string]int{"local": 0, "tailnet": 1, "lan": 2, "public": 3}
	sort.Slice(out, func(i, j int) bool {
		if rank[out[i].Network] != rank[out[j].Network] {
			return rank[out[i].Network] < rank[out[j].Network]
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

type Attempt struct {
	RouteID string `json:"route_id"`
	Origin  string `json:"origin"`
	Status  string `json:"status"`
}

// Select only sends unauthenticated identity GETs. It can fall back between
// registered routes without issuing a login, handing off a cookie, replaying a
// write, or changing an existing plugin session. Those effects have their own
// durable approval/attempt records and must use a new origin-bound session.
func (r Registry) Select(ctx context.Context, env Environment, probe func(context.Context, string, string) error) (Route, []Attempt, error) {
	candidates, err := r.Candidates(env)
	if err != nil {
		return Route{}, nil, err
	}
	if probe == nil {
		probe = Probe
	}
	attempts := []Attempt{}
	for _, route := range candidates {
		if err := ctx.Err(); err != nil {
			return Route{}, attempts, err
		}
		bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := probe(bounded, route.Origin, r.Binding.Ship)
		cancel()
		status := "verified"
		if errors.Is(err, eyre.ErrIdentity) {
			status = "wrong-ship"
		} else if errors.Is(err, eyre.ErrRedirect) {
			status = "redirect"
		} else if err != nil {
			status = "unavailable"
		}
		attempts = append(attempts, Attempt{route.ID, route.Origin, status})
		if err == nil {
			return route, attempts, nil
		}
	}
	return Route{}, attempts, errors.New("no registered ship route is reachable")
}

func Probe(ctx context.Context, origin, ship string) error {
	client, err := eyre.New(origin, ship)
	if err != nil {
		return err
	}
	defer client.Close()
	return client.Probe(ctx)
}

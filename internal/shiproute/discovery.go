package shiproute

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"os/exec"
	"path/filepath"
	"reflect"
	"time"
)

// NetworkDiscovery retains local node information only, never peers, users,
// key material, authentication URLs or raw daemon errors.
type NetworkDiscovery struct {
	Tailscale     string `json:"tailscale"`
	DNSName       string `json:"dns_name,omitempty"`
	TailnetSuffix string `json:"tailnet_suffix,omitempty"`
	Serve         string `json:"serve,omitempty"`
}

func (n NetworkDiscovery) Validate() error {
	if n.Tailscale == "not-installed" || n.Tailscale == "unavailable" {
		if n.DNSName == "" && n.TailnetSuffix == "" && n.Serve == "" {
			return nil
		}
	}
	if n.Tailscale == "ready" {
		r := Route{ID: "tailscale-serve", Origin: "https://" + n.DNSName, Network: "tailnet", Scope: n.TailnetSuffix, Source: "tailscale"}
		if r.Validate() == nil && (n.Serve == "unavailable" || n.Serve == "not-matched" || n.Serve == "tailnet" || n.Serve == "public") {
			return nil
		}
	}
	return errors.New("invalid local network discovery")
}

type Discovery struct {
	Network NetworkDiscovery
	Route   *Route
}

func (d Discovery) Validate() error {
	if d.Network.Validate() != nil {
		return errors.New("invalid network observation")
	}
	if d.Route == nil {
		if d.Network.Serve == "tailnet" || d.Network.Serve == "public" {
			return errors.New("network observation lost its route")
		}
		return nil
	}
	r := d.Route
	u, err := url.Parse(r.Origin)
	if err != nil || r.Validate() != nil || r.ID != "tailscale-serve" || r.Source != "tailscale" || u.Scheme != "https" || d.Network.Tailscale != "ready" || d.Network.Serve != r.Network || u.Hostname() != d.Network.DNSName || r.Network == "tailnet" && r.Scope != d.Network.TailnetSuffix {
		return errors.New("network observation route mismatch")
	}
	return nil
}

type tailscaleRead func(context.Context, ...string) ([]byte, error)

// DiscoverTailscale only inspects the default local daemon. It invokes no shell,
// installs nothing and changes no network settings or authentication state.
func DiscoverTailscale(ctx context.Context, hostOrigin string) Discovery {
	binary, err := exec.LookPath("tailscale")
	if err != nil || !filepath.IsAbs(binary) {
		return Discovery{Network: NetworkDiscovery{Tailscale: "not-installed"}}
	}
	return discoverTailscale(ctx, hostOrigin, func(ctx context.Context, args ...string) ([]byte, error) { return readTailscale(ctx, binary, args...) })
}
func discoverTailscale(ctx context.Context, hostOrigin string, read tailscaleRead) Discovery {
	unavailable := Discovery{Network: NetworkDiscovery{Tailscale: "unavailable"}}
	if read == nil {
		return unavailable
	}
	bounded, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()
	status, err := read(bounded, "status", "--json", "--peers=false")
	if err != nil {
		return unavailable
	}
	before, err := ParseTailnetStatus(status)
	if err != nil {
		return unavailable
	}
	out := Discovery{Network: NetworkDiscovery{Tailscale: "ready", DNSName: before.DNSName, TailnetSuffix: before.Suffix, Serve: "unavailable"}}
	if out.Network.Validate() != nil {
		return unavailable
	}
	config, err := read(bounded, "serve", "status", "--json")
	if err != nil {
		return out
	}
	// Refuse a profile switch during observation rather than combining two
	// different tailnet identities. No raw daemon output escapes this function.
	afterData, err := read(bounded, "status", "--json", "--peers=false")
	if err != nil {
		return unavailable
	}
	after, err := ParseTailnetStatus(afterData)
	if err != nil || !reflect.DeepEqual(before, after) {
		return unavailable
	}
	out.Network.Serve = "not-matched"
	route, err := ServeRoute(status, config, hostOrigin)
	if err == nil {
		out.Route = &route
		out.Network.Serve = route.Network
	}
	return out
}

// Do not embed bytes.Buffer: its promoted ReadFrom would bypass Write's bound
// when os/exec copies the subprocess pipe with io.Copy.
type discoveryBuffer struct{ data bytes.Buffer }

func (b *discoveryBuffer) Write(p []byte) (int, error) {
	if b.data.Len()+len(p) > 256<<10 {
		return 0, errors.New("discovery response exceeds bound")
	}
	return b.data.Write(p)
}
func readTailscale(ctx context.Context, binary string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, binary, args...)
	// TS_SOCKET, debug and proxy variables must not redirect this local read.
	command.Env = []string{"PATH=/usr/local/bin:/usr/bin:/bin", "LANG=C.UTF-8"}
	command.WaitDelay = 250 * time.Millisecond
	var out discoveryBuffer
	command.Stdout = &out
	command.Stderr = io.Discard
	if command.Run() != nil {
		return nil, errors.New("local Tailscale observation unavailable")
	}
	return out.data.Bytes(), nil
}

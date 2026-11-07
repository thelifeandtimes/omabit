package shiproute

import (
	"encoding/json"
	"errors"
	"net"
	"net/netip"
	"strconv"
	"strings"
)

// TailnetStatus consumes only local state from `tailscale status --json
// --peers=false`. Missing/changed fields fail closed. Peer lists, users, keys,
// authentication URLs and other private details are not retained or returned.
type TailnetStatus struct {
	DNSName string
	Suffix  string
	IPs     []netip.Addr
}

func ParseTailnetStatus(data []byte) (TailnetStatus, error) {
	var raw struct {
		BackendState string
		TUN          bool
		Self         *struct {
			DNSName string
			Online  bool
		}
		CurrentTailnet *struct{ MagicDNSSuffix string }
		TailscaleIPs   []netip.Addr
	}
	if len(data) > 256<<10 || !uniqueJSON(data) || json.Unmarshal(data, &raw) != nil || raw.BackendState != "Running" || !raw.TUN || raw.Self == nil || !raw.Self.Online || raw.CurrentTailnet == nil || len(raw.TailscaleIPs) == 0 || len(raw.TailscaleIPs) > 8 {
		return TailnetStatus{}, errors.New("running local Tailscale network unavailable")
	}
	status := TailnetStatus{DNSName: strings.TrimSuffix(raw.Self.DNSName, "."), Suffix: raw.CurrentTailnet.MagicDNSSuffix, IPs: raw.TailscaleIPs}
	if !validTailnet(status.Suffix) || !strings.HasSuffix(status.DNSName, "."+status.Suffix) {
		return TailnetStatus{}, errors.New("local MagicDNS identity unavailable")
	}
	for _, ip := range status.IPs {
		if !ip.IsValid() || !(netip.MustParsePrefix("100.64.0.0/10").Contains(ip) || netip.MustParsePrefix("fd7a:115c:a1e0::/48").Contains(ip)) {
			return TailnetStatus{}, errors.New("unexpected Tailscale interface address")
		}
	}
	return status, nil
}

// TailnetRoute proposes the daemon's actual full DNS name and a known ship HTTP
// listener. It does not infer a hostname from an SSH alias, configure Tailscale,
// open a firewall, advertise a loopback-only service, or claim reachability.
func TailnetRoute(data []byte, listenAddress string, port uint16) (Route, error) {
	status, err := ParseTailnetStatus(data)
	if err != nil || port == 0 {
		return Route{}, errors.New("tailnet route discovery unavailable")
	}
	bound, err := netip.ParseAddr(listenAddress)
	if err != nil || bound.IsLoopback() || bound.Zone() != "" {
		return Route{}, errors.New("ship listener is not reachable on its tailnet interface")
	}
	bound = bound.Unmap()
	reachable := false
	for _, ip := range status.IPs {
		reachable = reachable || bound == ip || bound.IsUnspecified() && bound.Is4() == ip.Is4()
	}
	if !reachable {
		return Route{}, errors.New("ship listener does not bind its tailnet interface")
	}
	portText := strconv.Itoa(int(port))
	origin := "http://" + net.JoinHostPort(status.DNSName, portText)
	if port == 80 {
		origin = "http://" + status.DNSName
	}
	route := Route{ID: "tailscale-http", Origin: origin, Network: "tailnet", Scope: status.Suffix, Source: "tailscale", Enabled: true}
	return route, route.Validate()
}

package shiproute

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// A deliberately small subset of Tailscale's ServeConfig. Automatic discovery
// supports a node's HTTPS root proxy; service VIPs, path mounting and raw TCP
// forwarding require an explicit configured route. No daemon config is changed.
type serveConfig struct {
	TCP         map[string]*serveTCP
	Web         map[string]*serveWeb
	AllowFunnel map[string]bool
	Foreground  map[string]*serveConfig
	Services    json.RawMessage
	ETag        string
}
type serveTCP struct {
	HTTPS         bool
	HTTP          bool
	TCPForward    string
	TerminateTLS  string
	ProxyProtocol int
}
type serveWeb struct{ Handlers map[string]*serveHandler }
type serveHandler struct {
	Path          string
	Proxy         string
	Text          string
	Redirect      string
	AcceptAppCaps []string
}

// ServeRoute discovers one stable preferred origin (443, then the lowest port)
// that proxies the entire host origin. Funnel-enabled URLs are explicitly public,
// including when the same hostname is also usable inside the tailnet. The caller
// still has to probe from its own network before selecting or authenticating.
func ServeRoute(statusData, configData []byte, hostOrigin string) (Route, error) {
	status, err := ParseTailnetStatus(statusData)
	if err != nil {
		return Route{}, err
	}
	want, err := proxyOrigin(hostOrigin)
	if err != nil {
		return Route{}, err
	}
	if len(configData) > 256<<10 || !uniqueJSON(configData) {
		return Route{}, errors.New("invalid bounded Serve configuration")
	}
	var root *serveConfig
	d := json.NewDecoder(bytes.NewReader(configData))
	d.DisallowUnknownFields()
	if d.Decode(&root) != nil || d.Decode(new(any)) != io.EOF || root == nil || len(root.Foreground) > 16 {
		return Route{}, errors.New("unsupported Serve configuration")
	}
	configs := []*serveConfig{root}
	for _, c := range root.Foreground {
		if c == nil || len(c.Foreground) != 0 {
			return Route{}, errors.New("unsupported foreground Serve configuration")
		}
		configs = append(configs, c)
	}
	used := map[string]bool{}
	type candidate struct {
		port  int
		route Route
	}
	var found []candidate
	for _, c := range configs {
		if len(c.TCP) > 128 || len(c.Web) > 128 || len(c.AllowFunnel) > 128 {
			return Route{}, errors.New("Serve inventory exceeds bound")
		}
		for port := range c.TCP {
			if used[port] {
				return Route{}, errors.New("ambiguous foreground/background Serve port")
			}
			used[port] = true
		}
		for hostport, web := range c.Web {
			host, port, err := net.SplitHostPort(hostport)
			p, e := strconv.Atoi(port)
			if err != nil || e != nil || p < 1 || p > 65535 || strconv.Itoa(p) != port || host != status.DNSName {
				continue
			}
			tcp := c.TCP[port]
			if tcp == nil || !tcp.HTTPS || tcp.HTTP || tcp.TCPForward != "" || tcp.TerminateTLS != "" || tcp.ProxyProtocol != 0 || web == nil || len(web.Handlers) != 1 {
				continue
			}
			h := web.Handlers["/"]
			if h == nil || h.Proxy == "" || h.Path != "" || h.Text != "" || h.Redirect != "" || len(h.AcceptAppCaps) != 0 {
				continue
			}
			proxy, err := proxyOrigin(h.Proxy)
			if err != nil || proxy != want {
				continue
			}
			origin := "https://" + hostport
			if p == 443 {
				origin = "https://" + host
			}
			route := Route{ID: "tailscale-serve", Origin: origin, Network: "tailnet", Scope: status.Suffix, Source: "tailscale", Enabled: true}
			// Funnel state can be stored in another foreground/background section.
			for _, all := range configs {
				if all.AllowFunnel[hostport] {
					route.Network = "public"
					route.Scope = ""
				}
			}
			if route.Validate() != nil {
				return Route{}, errors.New("invalid discovered Serve origin")
			}
			found = append(found, candidate{p, route})
		}
	}
	if len(found) == 0 {
		return Route{}, errors.New("no HTTPS root proxy matches this ship listener")
	}
	sort.Slice(found, func(i, j int) bool {
		if found[i].port == found[j].port {
			return false
		}
		if found[i].port == 443 {
			return true
		}
		if found[j].port == 443 {
			return false
		}
		return found[i].port < found[j].port
	})
	return found[0].route, nil
}

func proxyOrigin(raw string) (string, error) {
	if !strings.Contains(raw, "://") {
		if _, err := strconv.ParseUint(raw, 10, 16); err == nil {
			raw = "127.0.0.1:" + raw
		}
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawFragment != "" || u.Opaque != "" || u.RawPath != "" || (u.Path != "" && u.Path != "/") {
		return "", errors.New("unsupported Serve proxy target")
	}
	// Existing managed listeners bind IPv4 loopback. Do not equate ::1, arbitrary
	// DNS names or alternate loopback IPs with that actual publication.
	if u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1" {
		return "", errors.New("Serve proxy is not the managed loopback listener")
	}
	port := u.Port()
	if port == "" {
		port = "80"
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 || strconv.Itoa(n) != port {
		return "", errors.New("invalid proxy port")
	}
	return "http://127.0.0.1:" + port, nil
}

// Duplicate keys are ambiguous discovery evidence. Bound nesting as well as bytes
// before decoding the small supported subset; unrelated private fields never
// appear in returned errors or route data.
func uniqueJSON(data []byte) bool {
	d := json.NewDecoder(bytes.NewReader(data))
	var value func(int) bool
	value = func(depth int) bool {
		if depth > 32 {
			return false
		}
		token, err := d.Token()
		if err != nil {
			return false
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return true
		}
		switch delim {
		case '{':
			keys := map[string]bool{}
			for d.More() {
				token, err := d.Token()
				key, ok := token.(string)
				if err != nil || !ok || keys[key] {
					return false
				}
				folded := strings.ToLower(key)
				if keys[folded] {
					return false
				}
				keys[folded] = true
				if !value(depth + 1) {
					return false
				}
			}
			end, err := d.Token()
			return err == nil && end == json.Delim('}')
		case '[':
			for d.More() {
				if !value(depth + 1) {
					return false
				}
			}
			end, err := d.Token()
			return err == nil && end == json.Delim(']')
		default:
			return false
		}
	}
	return value(0) && func() bool { _, err := d.Token(); return err == io.EOF }()
}

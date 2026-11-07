package shiproute

import (
	"encoding/json"
	"strings"
	"testing"
)

const serveStatus = `{"BackendState":"Running","TUN":true,"Self":{"Online":true,"DNSName":"box.example.ts.net."},"CurrentTailnet":{"MagicDNSSuffix":"example.ts.net"},"TailscaleIPs":["100.64.0.5"]}`

func serveFixture() map[string]any {
	return map[string]any{"TCP": map[string]any{"443": map[string]any{"HTTPS": true}}, "Web": map[string]any{"box.example.ts.net:443": map[string]any{"Handlers": map[string]any{"/": map[string]any{"Proxy": "http://127.0.0.1:8888"}}}}}
}
func TestServeDiscoveryMatchesWholeShipProxyAndPublicExposure(t *testing.T) {
	for _, kind := range []string{"background", "foreground", "funnel", "multiple", "bare-port", "localhost"} {
		t.Run(kind, func(t *testing.T) {
			cfg := serveFixture()
			if kind == "bare-port" || kind == "localhost" {
				target := "8888"
				if kind == "localhost" {
					target = "http://localhost:8888/"
				}
				cfg["Web"].(map[string]any)["box.example.ts.net:443"].(map[string]any)["Handlers"].(map[string]any)["/"].(map[string]any)["Proxy"] = target
			}
			if kind == "multiple" {
				cfg["TCP"].(map[string]any)["8443"] = map[string]any{"HTTPS": true}
				cfg["Web"].(map[string]any)["box.example.ts.net:8443"] = cfg["Web"].(map[string]any)["box.example.ts.net:443"]
			}
			if kind == "foreground" {
				cfg = map[string]any{"Foreground": map[string]any{"session": cfg}}
			}
			if kind == "funnel" {
				cfg["AllowFunnel"] = map[string]bool{"box.example.ts.net:443": true}
			}
			data, _ := json.Marshal(cfg)
			route, err := ServeRoute([]byte(serveStatus), data, "http://127.0.0.1:8888")
			if err != nil || route.Origin != "https://box.example.ts.net" || route.ID != "tailscale-serve" {
				t.Fatal(route, err)
			}
			if kind == "funnel" {
				if route.Network != "public" || route.Scope != "" {
					t.Fatal("Funnel mislabeled private", route)
				}
			} else if route.Network != "tailnet" || route.Scope != "example.ts.net" {
				t.Fatal(route)
			}
			s := DefaultSettings()
			s.DisabledAutomatic = []string{"tailscale-serve"}
			reg, err := s.Effective(fixture().Binding, 0, []Route{route})
			if err != nil || reg.Routes[0].Enabled {
				t.Fatal("automatic disable lost", reg, err)
			}
		})
	}
}
func TestServeDiscoveryRefusesAmbiguousOrPartialProxy(t *testing.T) {
	for _, kind := range []string{"wrong-port", "wrong-host", "path", "shadow-login", "plain-http", "file", "text", "redirect", "caps", "tcp-forward", "nested-foreground", "duplicate-port", "unknown-handler", "duplicate-key", "null", "oversized", "unsupported-target", "stopped"} {
		t.Run(kind, func(t *testing.T) {
			cfg := serveFixture()
			web := cfg["Web"].(map[string]any)
			handlers := web["box.example.ts.net:443"].(map[string]any)["Handlers"].(map[string]any)
			h := handlers["/"].(map[string]any)
			switch kind {
			case "wrong-port":
				h["Proxy"] = "http://127.0.0.1:8889"
			case "wrong-host":
				web["other.example.ts.net:443"] = web["box.example.ts.net:443"]
				delete(web, "box.example.ts.net:443")
			case "path":
				delete(handlers, "/")
				handlers["/app"] = h
			case "shadow-login":
				handlers["/~/login"] = map[string]any{"Text": "blocked"}
			case "plain-http":
				cfg["TCP"].(map[string]any)["443"] = map[string]bool{"HTTP": true}
			case "file":
				h["Path"] = "/PRIVATE"
			case "text":
				h["Text"] = "PRIVATE"
			case "redirect":
				h["Redirect"] = "https://PRIVATE"
			case "caps":
				h["AcceptAppCaps"] = []string{"PRIVATE"}
			case "tcp-forward":
				cfg["TCP"].(map[string]any)["443"].(map[string]any)["TCPForward"] = "PRIVATE"
			case "nested-foreground":
				cfg["Foreground"] = map[string]any{"s": map[string]any{"Foreground": map[string]any{"s": serveFixture()}}}
			case "duplicate-port":
				cfg["Foreground"] = map[string]any{"s": serveFixture()}
			case "unknown-handler":
				h["FutureRoutingRule"] = "PRIVATE"
			case "unsupported-target":
				h["Proxy"] = "https+insecure://127.0.0.1:8888"
			}
			data, _ := json.Marshal(cfg)
			status := serveStatus
			switch kind {
			case "duplicate-key":
				data = []byte(`{"TCP":{},"TCP":{"443":{"HTTPS":true}}}`)
			case "null":
				data = []byte(`null`)
			case "oversized":
				data = []byte(strings.Repeat(" ", 256<<10) + `{}`)
			case "stopped":
				status = strings.Replace(status, "Running", "Stopped", 1)
			}
			_, err := ServeRoute([]byte(status), data, "http://127.0.0.1:8888")
			if err == nil || strings.Contains(err.Error(), "PRIVATE") {
				t.Fatal("unsafe discovery result", err)
			}
		})
	}
}

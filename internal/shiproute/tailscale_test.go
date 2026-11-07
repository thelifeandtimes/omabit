package shiproute

import (
	"strings"
	"testing"
)

const tailnetJSON = `{"BackendState":"Running","TUN":true,"Self":{"Online":true,"DNSName":"actual-name.example.ts.net."},"CurrentTailnet":{"MagicDNSSuffix":"example.ts.net"},"TailscaleIPs":["100.64.0.8","fd7a:115c:a1e0::8"],"AuthURL":"never retained","Peer":{"ignored":"private data"}}`

func TestTailnetDiscoveryUsesActualDNSAndReachableListener(t *testing.T) {
	for _, bind := range []string{"0.0.0.0", "::", "100.64.0.8", "fd7a:115c:a1e0::8"} {
		r, err := TailnetRoute([]byte(tailnetJSON), bind, 8888)
		if err != nil || r.Origin != "http://actual-name.example.ts.net:8888" || r.Scope != "example.ts.net" {
			t.Fatal(r, err)
		}
	}
	for _, bind := range []string{"127.0.0.1", "::1", "192.168.1.5", "100.64.0.9", "localhost", "", "fe80::8%tailscale0"} {
		if _, err := TailnetRoute([]byte(tailnetJSON), bind, 8888); err == nil {
			t.Fatal("unreachable listener advertised", bind)
		}
	}
	r, err := TailnetRoute([]byte(tailnetJSON), "0.0.0.0", 80)
	if err != nil || r.Origin != "http://actual-name.example.ts.net" {
		t.Fatal("noncanonical default port", r, err)
	}
}

func TestTailnetDiscoveryRefusesIncompleteOrStoppedDaemon(t *testing.T) {
	for _, bad := range []string{
		`{}`, tailnetJSON + `{}`, strings.Repeat(" ", 256<<10+1),
		strings.Replace(tailnetJSON, `"Running"`, `"Stopped"`, 1),
		strings.Replace(tailnetJSON, `"TUN":true`, `"TUN":false`, 1),
		strings.Replace(tailnetJSON, `"Online":true`, `"Online":false`, 1),
		strings.Replace(tailnetJSON, `actual-name.example.ts.net.`, `other.example.net.`, 1),
		strings.Replace(tailnetJSON, `100.64.0.8`, `127.0.0.1`, 1),
	} {
		if _, err := TailnetRoute([]byte(bad), "0.0.0.0", 8888); err == nil {
			t.Fatal("invalid local network advertised")
		}
	}
}

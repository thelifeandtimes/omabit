package bootpill

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func trustPillTLS(t *testing.T, server *httptest.Server) *http.Transport {
	t.Helper()
	old := http.DefaultTransport
	transport := old.(*http.Transport).Clone()
	transport.TLSClientConfig = server.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
	http.DefaultTransport = transport
	t.Cleanup(func() { http.DefaultTransport = old; transport.CloseIdleConnections() })
	return transport
}

func pillDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestPillRejectsInsecureSourcesBeforeContact(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++ }))
	defer server.Close()
	for _, source := range []string{server.URL, "HTTP://example.invalid/pill", "ftp://example.invalid/pill", "file:///tmp/pill", "//example.invalid/pill", "https://user:secret@example.invalid/pill", "https://example.invalid/pill#fragment"} {
		if _, _, err := Resolve(context.Background(), source, pillDir(t)); err == nil {
			t.Fatalf("accepted disallowed source %q", source)
		}
	}
	if calls != 0 {
		t.Fatal("HTTP source contacted")
	}
}

func TestPillHTTPSRedirectAndTrustedExpectedDescriptor(t *testing.T) {
	body := "fixture custom pill"
	meta := protocol.BootPill{SHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(body))), Bytes: int64(len(body))}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" || r.Header.Get("Proxy-Authorization") != "" || r.Header.Get("Referer") != "" {
			t.Error("artifact download forwarded credentials")
		}
		switch r.URL.Path {
		case "/redirect":
			http.Redirect(w, r, "/pill", http.StatusFound)
		case "/downgrade":
			http.Redirect(w, r, "http://127.0.0.1/pill", http.StatusFound)
		default:
			io.WriteString(w, body)
		}
	}))
	defer server.Close()
	trustPillTLS(t, server)
	dir := pillDir(t)
	path, got, err := ResolveExpected(context.Background(), server.URL+"/redirect?token=synthetic", dir, &meta)
	if err != nil || got != meta {
		t.Fatalf("custom HTTPS failed: %v %+v", err, got)
	}
	if _, got, err := ResolveExpected(context.Background(), path, pillDir(t), &meta); err != nil || got != meta {
		t.Fatalf("custom local failed: %v %+v", err, got)
	}
	for _, source := range []string{server.URL + "/downgrade", server.URL + "/pill"} {
		wrong := meta
		wrong.SHA256 = strings.Repeat("a", 64)
		stage := pillDir(t)
		if _, _, err := ResolveExpected(context.Background(), source, stage, &wrong); err == nil {
			t.Fatal("accepted downgrade or changed content")
		}
		files, err := os.ReadDir(stage)
		if err != nil || len(files) != 0 {
			t.Fatal("failed artifact left staged bytes", err)
		}
	}
	wrong := meta
	wrong.Bytes--
	if _, _, err := ResolveExpected(context.Background(), server.URL+"/pill", pillDir(t), &wrong); err == nil {
		t.Fatal("trusted size ignored")
	}
}

func TestRecommendedAliasUsesFrozenReleaseAndRejectsSubstitution(t *testing.T) {
	calls := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if "https://"+r.Host+r.URL.Path != RecommendedURL {
			t.Error("provider alias not frozen", r.Host, r.URL.Path)
		}
		io.WriteString(w, "substituted TLS-served pill")
	}))
	defer server.Close()
	transport := trustPillTLS(t, server)
	transport.TLSClientConfig.ServerName = "example.com"
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
	}
	pin := RecommendedDescriptor()
	if pin.SHA256 != "85635c0b17a138a6473580ca362475faadf281e09cc9fa3746cfc698cc054f62" || pin.Bytes != 59351751 {
		t.Fatal("release descriptor changed")
	}
	for _, source := range []string{RecommendedAlias, RecommendedURL} {
		dir := pillDir(t)
		if _, _, err := Resolve(context.Background(), source, dir); err == nil || !strings.Contains(err.Error(), "differs") {
			t.Fatal("release substitution not rejected", err)
		}
		files, _ := os.ReadDir(dir)
		if len(files) != 0 {
			t.Fatal("substituted release left staged content")
		}
	}
	if calls != 2 {
		t.Fatal("frozen release not contacted", calls)
	}
}

//go:build linux || darwin

package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/service"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type httpRoundTripper func(*http.Request) (*http.Response, error)

func (f httpRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestAuthenticatedHTTPUsesSharedToolsAndRechecksCredential(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	path := filepath.Join(t.TempDir(), "credential.json")
	info, err := CreateHTTPToken(path)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var credential httpCredential
	if json.Unmarshal(raw, &credential) != nil {
		t.Fatal("credential encoding")
	}
	public, _ := json.Marshal(info)
	if bytes.Contains(public, []byte(credential.Token)) {
		t.Fatal("credential secret entered creation result")
	}
	authority, err := openHTTPAuthority(path)
	if err != nil {
		t.Fatal(err)
	}
	core, err := service.New(service.Options{RequireAllowed: true})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(nil)
	handler, closeSessions, err := NewHTTPHandler(core, server.Listener.Addr().String(), authority.Verify)
	if err != nil {
		t.Fatal(err)
	}
	server.Config.Handler = handler
	server.Start()
	defer server.Close()
	defer closeSessions()
	client := &http.Client{Transport: httpRoundTripper(func(r *http.Request) (*http.Response, error) {
		copy := r.Clone(r.Context())
		copy.Header = r.Header.Clone()
		copy.Header.Set("Authorization", "Bearer "+credential.Token)
		return http.DefaultTransport.RoundTrip(copy)
	})}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "omabit-http-test", Version: "1"}, nil).Connect(ctx, &mcp.StreamableClientTransport{Endpoint: server.URL + "/mcp", HTTPClient: client}, &mcp.ClientSessionOptions{ProtocolVersion: "2025-11-25"})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	tools, err := session.ListTools(ctx, nil)
	if err != nil || len(tools.Tools) == 0 {
		t.Fatal("authenticated discovery", err)
	}
	got, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit.catalog", Arguments: map[string]any{}})
	if err != nil || got.IsError || len(got.Content) != 1 {
		t.Fatal("shared HTTP catalog", err)
	}
	expected, _ := json.Marshal(core.Discovery())
	if got.Content[0].(*mcp.TextContent).Text != string(expected) {
		t.Fatal("HTTP catalog diverged from shared core")
	}
	denied, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit.runner.hello.v1", Arguments: map[string]any{"name": "not-allowed", "runner_id": "runner"}})
	if err != nil || !denied.IsError {
		t.Fatal("HTTP bypassed runner allowlist", err)
	}
	// Rename retains original secret bytes; the running server must refuse its old
	// path immediately, without reauthorization or session-ID-only access.
	if err := os.Rename(path, path+"-retained"); err != nil {
		t.Fatal(err)
	}
	if _, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit.catalog", Arguments: map[string]any{}}); err == nil {
		t.Fatal("existing session bypassed credential revocation")
	}
}

func TestHTTPRefusesUntrustedAuthorityOriginFramingAndMissingAuthentication(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credential.json")
	if _, err := CreateHTTPToken(path); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	var credential httpCredential
	json.Unmarshal(raw, &credential)
	authority, err := openHTTPAuthority(path)
	if err != nil {
		t.Fatal(err)
	}
	core, err := service.New(service.Options{RequireAllowed: true})
	if err != nil {
		t.Fatal(err)
	}
	h, closeSessions, err := NewHTTPHandler(core, "127.0.0.1:38417", authority.Verify)
	if err != nil {
		t.Fatal(err)
	}
	defer closeSessions()
	for _, tc := range []struct {
		name string
		want int
	}{{"missing", 401}, {"wrong", 401}, {"duplicate", 401}, {"host", 403}, {"origin", 403}, {"empty-origin", 403}, {"remote", 403}, {"query", 400}, {"cookie", 400}, {"path", 400}, {"future-protocol", 400}, {"standalone-get", 405}} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "http://127.0.0.1:38417/mcp", bytes.NewBufferString(`{}`))
			req.RemoteAddr = "127.0.0.1:40000"
			req.Header.Set("Authorization", "Bearer "+credential.Token)
			switch tc.name {
			case "missing":
				req.Header.Del("Authorization")
			case "wrong":
				req.Header.Set("Authorization", "Bearer invalid")
			case "duplicate":
				req.Header.Add("Authorization", "Bearer "+credential.Token)
			case "host":
				req.Host = "attacker.invalid"
			case "origin":
				req.Header.Set("Origin", "https://attacker.invalid")
			case "empty-origin":
				req.Header["Origin"] = []string{""}
			case "remote":
				req.RemoteAddr = "192.0.2.1:40000"
			case "query":
				req.URL.RawQuery = "token=forbidden"
			case "cookie":
				req.Header.Set("Cookie", "token=forbidden")
			case "path":
				req.URL.Path = "/other"
			case "future-protocol":
				req.Header.Set("MCP-Protocol-Version", "2026-07-28")
			case "standalone-get":
				req.Method = "GET"
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if w.Code != tc.want || bytes.Contains(w.Body.Bytes(), []byte(credential.Token)) {
				t.Fatal("HTTP admission/secret refusal", w.Code, w.Body)
			}
		})
	}
	closeSessions()
	req := httptest.NewRequest("POST", "http://127.0.0.1:38417/mcp", nil)
	req.RemoteAddr = "127.0.0.1:40000"
	req.Header.Set("Authorization", "Bearer "+credential.Token)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 503 {
		t.Fatal("closed server admitted request", w.Code)
	}
	for _, address := range []string{"0.0.0.0:1234", "localhost:1234", "192.0.2.1:1234", "127.0.0.1:00123", "127.0.0.1:65536"} {
		if localHTTPAddress(address) {
			t.Fatal("noncanonical/nonlocal listener", address)
		}
	}
}

func TestHTTPTokenOwnershipBindingsAndNoOverwrite(t *testing.T) {
	for _, scenario := range []string{"valid", "exposed", "symlink", "hardlink", "expired", "changed"} {
		t.Run(scenario, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "credential.json")
			if _, err := CreateHTTPToken(path); err != nil {
				t.Fatal(err)
			}
			raw, _ := os.ReadFile(path)
			var c httpCredential
			json.Unmarshal(raw, &c)
			a, err := openHTTPAuthority(path)
			if err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "exposed":
				os.Chmod(path, 0644)
			case "symlink":
				os.Rename(path, path+"-retained")
				if err := os.Symlink(path+"-retained", path); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err := os.Link(path, path+"-linked"); err != nil {
					t.Fatal(err)
				}
			case "expired":
				a.expires = time.Now().Add(-time.Second)
			case "changed":
				c.Token = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
				changed, _ := json.Marshal(c)
				os.WriteFile(path, append(changed, '\n'), 0600)
			}
			_, err = a.Verify(t.Context(), c.Token, nil)
			if (err == nil) != (scenario == "valid") {
				t.Fatal("credential admission", scenario, err)
			}
			if scenario == "valid" {
				if _, err := CreateHTTPToken(path); err == nil {
					t.Fatal("credential overwritten")
				}
				after, _ := os.ReadFile(path)
				if !bytes.Equal(raw, after) {
					t.Fatal("failed creation changed existing credential")
				}
				f, _ := os.Open(path)
				node, _ := f.Stat()
				f.Close()
				if node.Mode().Perm() != 0600 {
					t.Fatal("credential mode")
				}
			}
		})
	}
}

func TestHTTPNegotiatesEachAdvertisedProtocol(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	path := filepath.Join(t.TempDir(), "credential.json")
	if _, err := CreateHTTPToken(path); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	var credential httpCredential
	json.Unmarshal(raw, &credential)
	authority, err := openHTTPAuthority(path)
	if err != nil {
		t.Fatal(err)
	}
	core, err := service.New(service.Options{RequireAllowed: true})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewUnstartedServer(nil)
	handler, closeSessions, err := NewHTTPHandler(core, srv.Listener.Addr().String(), authority.Verify)
	if err != nil {
		t.Fatal(err)
	}
	srv.Config.Handler = handler
	srv.Start()
	defer srv.Close()
	defer closeSessions()
	client := &http.Client{Transport: httpRoundTripper(func(r *http.Request) (*http.Response, error) {
		copy := r.Clone(r.Context())
		copy.Header = r.Header.Clone()
		copy.Header.Set("Authorization", "Bearer "+credential.Token)
		return http.DefaultTransport.RoundTrip(copy)
	})}
	for _, version := range httpProtocolVersions {
		t.Run(version, func(t *testing.T) {
			session, err := mcp.NewClient(&mcp.Implementation{Name: "http-negotiation", Version: "1"}, nil).Connect(ctx, &mcp.StreamableClientTransport{Endpoint: srv.URL + "/mcp", HTTPClient: client}, &mcp.ClientSessionOptions{ProtocolVersion: version})
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()
			if got := session.InitializeResult().ProtocolVersion; got != version {
				t.Fatal("negotiated unexpected protocol", got)
			}
			result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit.catalog", Arguments: map[string]any{}})
			if err != nil || result.IsError {
				t.Fatal("negotiated catalog", err)
			}
		})
	}
}

func TestHTTPBoundsConcurrentAuthenticationRequests(t *testing.T) {
	core, err := service.New(service.Options{RequireAllowed: true})
	if err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{}, 16)
	release := make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	verifier := func(ctx context.Context, _ string, _ *http.Request) (*auth.TokenInfo, error) {
		entered <- struct{}{}
		select {
		case <-release:
			return &auth.TokenInfo{UserID: "fixture-owner", Expiration: time.Now().Add(time.Hour)}, nil
		case <-ctx.Done():
			return nil, auth.ErrInvalidToken
		}
	}
	h, closeSessions, err := NewHTTPHandler(core, "127.0.0.1:38417", verifier)
	if err != nil {
		t.Fatal(err)
	}
	defer closeSessions()
	request := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "http://127.0.0.1:38417/mcp", nil)
		r.RemoteAddr = "127.0.0.1:40000"
		r.Header.Set("Authorization", "Bearer fixture")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	results := make(chan int, 16)
	for n := 0; n < 16; n++ {
		go func() { results <- request().Code }()
	}
	for n := 0; n < 16; n++ {
		select {
		case <-entered:
		case <-time.After(3 * time.Second):
			t.Fatal("bounded verifier did not enter")
		}
	}
	if w := request(); w.Code != 503 {
		t.Fatal("authentication concurrency cap bypass", w.Code)
	}
	once.Do(func() { close(release) })
	for n := 0; n < 16; n++ {
		select {
		case code := <-results:
			if code != 405 {
				t.Fatal("admitted GET changed", code)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("bounded request did not release")
		}
	}
}

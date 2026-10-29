package eyre

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// This fixture models the relevant current Eyre behavior: login with an
// existing cookie replaces that session; cookie-free login creates another;
// logout without all/sid revokes only the request's session.
type independentSessionFixture struct {
	mu                    sync.Mutex
	sessions              map[string]bool
	logins, threadCalls   int
	threadStatus          int
	threadBody, mediaType string
	host, reuse           string
}

func independentSessionServer(t *testing.T) (*Client, *independentSessionFixture) {
	t.Helper()
	f := &independentSessionFixture{sessions: map[string]bool{}, host: "~zod", threadStatus: 200,
		threadBody: `"~` + fakeCode + `"`, mediaType: "application/json"}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.URL.Path == "/~/host" {
			fmt.Fprint(w, f.host)
			return
		}
		cookie, _ := r.Cookie("urbauth-~zod")
		if r.URL.Path == "/~/login" {
			if r.Method != "POST" || r.ParseForm() != nil || r.Form.Get("password") != fakeCode {
				w.WriteHeader(400)
				return
			}
			if cookie != nil {
				delete(f.sessions, cookie.Value)
				t.Error("new login received a previous client's cookie")
			}
			f.logins++
			token := fmt.Sprintf("0vSYNTHETIC-%d", f.logins)
			if f.reuse != "" {
				token = f.reuse
			}
			f.sessions[token] = true
			http.SetCookie(w, &http.Cookie{Name: "urbauth-~zod", Value: token, Path: "/", MaxAge: 3600})
			fmt.Fprint(w, token)
			return
		}
		if cookie == nil || !f.sessions[cookie.Value] {
			w.WriteHeader(403)
			return
		}
		switch r.URL.Path {
		case "/~/name":
			fmt.Fprint(w, "~zod")
		case "/~/logout":
			body, _ := io.ReadAll(r.Body)
			if len(body) != 0 || r.URL.RawQuery != "" {
				t.Error("logout must revoke only its own session")
			}
			delete(f.sessions, cookie.Value)
			w.WriteHeader(303)
		case "/spider/base/json/code/ship":
			f.threadCalls++
			body, _ := io.ReadAll(r.Body)
			if r.Method != "POST" || string(body) != "null" || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Accept") != "application/json" {
				t.Error("incorrect fixed code-thread request")
			}
			w.Header().Set("Content-Type", f.mediaType)
			if f.threadStatus == 302 {
				w.Header().Set("Location", "/must-not-follow")
			}
			w.WriteHeader(f.threadStatus)
			fmt.Fprint(w, f.threadBody)
		case "/~/scry/settings/desk/talon.json", "/~/scry/chat/dm.json":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{}`)
		default:
			t.Error("unexpected request path")
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(s.Close)
	c, err := New(s.URL, "~zod")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Close)
	if err := c.Login(context.Background(), []byte(fakeCode)); err != nil {
		t.Fatal(err)
	}
	return c, f
}

func TestIndependentSessionsContinueDirectlyAfterAuthClientCloses(t *testing.T) {
	ctx := context.Background()
	owner, f := independentSessionServer(t)
	theme, err := owner.NewIndependentSession(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer theme.Close()
	dms, err := owner.NewIndependentSession(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer dms.Close()
	if theme.session.Value == dms.session.Value || owner.session.Value == theme.session.Value || owner.session.Value == dms.session.Value {
		t.Fatal("sessions were shared between clients")
	}
	// Model private handoff into separate adapters. Close all issuing clients,
	// then create fresh direct clients from the three independent credentials.
	clients := []*Client{owner, theme, dms}
	for i, issued := range clients {
		data, err := issued.ExportSession()
		if err != nil {
			t.Fatal(err)
		}
		direct, err := New(issued.origin, issued.ship)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(direct.Close)
		issued.Close()
		if err := direct.RestoreSession(data); err != nil {
			t.Fatal(err)
		}
		clients[i] = direct
	}
	owner, theme, dms = clients[0], clients[1], clients[2]
	if _, err := theme.Scry(ctx, "settings", "/desk/talon"); err != nil {
		t.Fatal("direct Theme-shaped read failed", err)
	}
	if _, err := dms.Scry(ctx, "chat", "/dm"); err != nil {
		t.Fatal("direct DM-shaped read failed", err)
	}
	// Keep a copy before logout to prove server-side revocation, not just
	// clearing the issuing client's local cookie.
	themeCredential, err := theme.ExportSession()
	if err != nil {
		t.Fatal(err)
	}
	replay, err := New(theme.origin, theme.ship)
	if err != nil {
		t.Fatal(err)
	}
	defer replay.Close()
	if err := replay.RestoreSession(themeCredential); err != nil {
		t.Fatal(err)
	}
	if err := theme.Logout(ctx); err != nil {
		t.Fatal(err)
	}
	if err := replay.Check(ctx); !errors.Is(err, ErrAuth) {
		t.Fatal("server accepted a revoked session", err)
	}
	for _, survivor := range []*Client{owner, dms} {
		if err := survivor.Check(ctx); err != nil {
			t.Fatal("logging out one plugin revoked another session", err)
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.logins != 3 || f.threadCalls != 2 || len(f.sessions) != 2 {
		t.Fatal("unexpected authentication/revocation counts")
	}
}

func TestIndependentSessionRefusesInvalidOrUnavailableCodeThread(t *testing.T) {
	for _, tc := range []struct {
		name, body, media string
		status            int
		want              error
	}{
		{"missing", "private error", "application/json", 404, ErrIndependentSessionUnavailable},
		{"redirect", "private error", "application/json", 302, ErrRedirect},
		{"thread-error", "private stack " + fakeCode, "application/json", 500, ErrProtocol},
		{"wrong-media", `"~` + fakeCode + `"`, "text/html", 200, ErrProtocol},
		{"object", `{"secret":"` + fakeCode + `"}`, "application/json", 200, ErrProtocol},
		{"invalid-code", `"private-invalid-code"`, "application/json", 200, ErrProtocol},
		{"oversized", `"` + strings.Repeat("x", 257) + `"`, "application/json", 200, ErrProtocol},
	} {
		t.Run(tc.name, func(t *testing.T) {
			owner, f := independentSessionServer(t)
			f.mu.Lock()
			f.threadBody, f.mediaType, f.threadStatus = tc.body, tc.media, tc.status
			f.mu.Unlock()
			child, err := owner.NewIndependentSession(context.Background())
			if child != nil || !errors.Is(err, tc.want) {
				t.Fatal("unexpected failure outcome", err)
			}
			if strings.Contains(err.Error(), fakeCode) || strings.Contains(err.Error(), "private") {
				t.Fatal("secret thread output reached diagnostics")
			}
			if err := owner.Check(context.Background()); err != nil {
				t.Fatal("failed issuance damaged original session", err)
			}
			f.mu.Lock()
			defer f.mu.Unlock()
			if f.logins != 1 || f.threadCalls != 1 {
				t.Fatal("failed thread retried or created a login")
			}
		})
	}
}

func TestIndependentSessionRejectsReusedCookieWithoutRevokingParent(t *testing.T) {
	owner, f := independentSessionServer(t)
	f.mu.Lock()
	f.reuse = owner.session.Value
	f.mu.Unlock()
	child, err := owner.NewIndependentSession(context.Background())
	if child != nil || !errors.Is(err, ErrProtocol) {
		t.Fatal("parent cookie accepted as independent", err)
	}
	if err := owner.Check(context.Background()); err != nil {
		t.Fatal("reused parent cookie was revoked", err)
	}
}

func TestIndependentSessionChecksIdentityBeforeRequestingCode(t *testing.T) {
	owner, f := independentSessionServer(t)
	f.mu.Lock()
	f.host = "~nec"
	f.mu.Unlock()
	child, err := owner.NewIndependentSession(context.Background())
	if child != nil || !errors.Is(err, ErrIdentity) {
		t.Fatal("wrong identity accepted", err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.threadCalls != 0 || f.logins != 1 {
		t.Fatal("credentials requested after identity mismatch")
	}
}

package eyre

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const fakeCode = "lidlut-tabwed-pillex-ridrup" // public fake ~zod code from Urbit's tutorial
const testToken = "0vTEST-SESSION-MUST-NOT-APPEAR-IN-ERRORS"

func fakeEyre(t *testing.T, handler http.HandlerFunc) (*Client, *atomic.Bool) {
	t.Helper()
	valid := &atomic.Bool{}
	valid.Store(true)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" || r.Header.Get("Origin") != "" {
			t.Error("unexpected credentials/origin")
		}
		switch r.URL.Path {
		case "/~/host":
			fmt.Fprint(w, "~zod")
			return
		case "/~/login":
			if r.Method != "POST" || r.ParseForm() != nil || r.Form.Get("password") != fakeCode {
				w.WriteHeader(400)
				fmt.Fprint(w, "private authentication detail")
				return
			}
			http.SetCookie(w, &http.Cookie{Name: "urbauth-~zod", Value: testToken, Path: "/", MaxAge: 3600})
			fmt.Fprint(w, testToken)
			return
		}
		cookie, err := r.Cookie("urbauth-~zod")
		if err != nil || cookie.Value != testToken || !valid.Load() {
			w.WriteHeader(401)
			fmt.Fprint(w, "secret error "+testToken)
			return
		}
		if r.URL.Path == "/~/name" {
			fmt.Fprint(w, "~zod")
			return
		}
		if r.URL.Path == "/~/logout" {
			valid.Store(false)
			w.Header().Set("Location", "/~/login")
			w.WriteHeader(303)
			return
		}
		if handler != nil {
			handler(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"base":{"running":true}}`)
	}))
	t.Cleanup(srv.Close)
	c, err := New(srv.URL, "~zod")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Close)
	return c, valid
}

func TestOriginAndPathBoundaries(t *testing.T) {
	for _, raw := range []string{"http://example.com", "http://localhost:8080", "https://user:secret@example.com", "https://example.com/path", "https://example.com?password=secret", "https://example.com#fragment", "file:///tmp/session", "https://example.com:65536", "http://[::ffff:192.168.1.1]", "https://example.com/%2e%2e", "https://example.com/\n"} {
		if _, err := Origin(raw); err == nil {
			t.Fatalf("unsafe origin accepted: %q", raw)
		}
	}
	if got, err := Origin("https://EXAMPLE.com:443/"); err != nil || got != "https://example.com" {
		t.Fatalf("canonical origin: %s %v", got, err)
	}
	for _, p := range []string{"relative", "//other-host", "/../login", "/x/%2e%2e", "/x?secret", "/x#fragment", "/x\\y", "/x\ny"} {
		if ValidPath(p) {
			t.Fatalf("unsafe path accepted: %q", p)
		}
	}
}

func TestLoginPersistenceIdentityAndExpiry(t *testing.T) {
	c, valid := fakeEyre(t, nil)
	ctx := context.Background()
	code := []byte(fakeCode)
	if err := c.Login(ctx, code); err != nil {
		t.Fatal(err)
	}
	for _, b := range code {
		if b != 0 {
			t.Fatal("login input not cleared")
		}
	}
	secret, err := c.ExportSession()
	if err != nil {
		t.Fatal(err)
	}
	next, err := New(c.origin, "~zod")
	if err != nil {
		t.Fatal(err)
	}
	defer next.Close()
	if err := next.RestoreSession(append([]byte{}, secret...)); err != nil {
		t.Fatal(err)
	}
	if err := next.Check(ctx); err != nil {
		t.Fatal(err)
	}
	other, _ := fakeEyre(t, nil)
	if err := other.RestoreSession(append([]byte{}, secret...)); err == nil {
		t.Fatal("cookie accepted at another port")
	}
	wrong, _ := New(c.origin, "~nec")
	defer wrong.Close()
	if err := wrong.Probe(ctx); !errors.Is(err, ErrIdentity) {
		t.Fatalf("wrong identity: %v", err)
	}
	valid.Store(false)
	if err := next.Check(ctx); !errors.Is(err, ErrAuth) || strings.Contains(err.Error(), testToken) {
		t.Fatalf("expiry or redaction: %v", err)
	}
	var expired session
	json.Unmarshal(secret, &expired)
	expired.Expires = time.Now().Add(-time.Second)
	data, _ := json.Marshal(expired)
	if err := next.RestoreSession(data); !errors.Is(err, ErrAuth) {
		t.Fatalf("stored expiry ignored: %v", err)
	}
}

func TestRedirectsCannotReceiveCredentials(t *testing.T) {
	var leaked atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { leaked.Add(1) }))
	defer other.Close()
	for _, location := range []string{other.URL + "/login", "/elsewhere"} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/~/host" {
				fmt.Fprint(w, "~zod")
				return
			}
			w.Header().Set("Location", location)
			w.WriteHeader(307)
		}))
		c, err := New(srv.URL, "~zod")
		if err != nil {
			t.Fatal(err)
		}
		if err := c.Login(context.Background(), []byte(fakeCode)); !errors.Is(err, ErrRedirect) {
			t.Fatalf("redirect accepted: %v", err)
		}
		c.Close()
		srv.Close()
	}
	if leaked.Load() != 0 {
		t.Fatal("login followed a redirect")
	}
}

func TestScryAndLogout(t *testing.T) {
	c, valid := fakeEyre(t, nil)
	ctx := context.Background()
	if err := c.Login(ctx, []byte(fakeCode)); err != nil {
		t.Fatal(err)
	}
	if data, err := c.Scry(ctx, "hood", "/kiln/vats"); err != nil || !json.Valid(data) {
		t.Fatalf("scry: %s %v", data, err)
	}
	if err := c.Logout(ctx); err != nil || valid.Load() {
		t.Fatalf("logout: %v", err)
	}
	if err := c.Check(ctx); !errors.Is(err, ErrAuth) {
		t.Fatalf("logout retained session: %v", err)
	}
}

func TestHTTPSRequiresTrustedCertificate(t *testing.T) {
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "~zod") }))
	srv.Config.ErrorLog = log.New(io.Discard, "", 0)
	srv.StartTLS()
	defer srv.Close()
	c, err := New(srv.URL, "~zod")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := c.Probe(context.Background()); !errors.Is(err, ErrTransport) {
		t.Fatalf("untrusted TLS certificate accepted: %v", err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(srv.Certificate())
	c.http.Transport.(*http.Transport).TLSClientConfig = &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}
	if err := c.Probe(context.Background()); err != nil {
		t.Fatalf("trusted HTTPS origin: %v", err)
	}
}

// This requires explicit opt-in to the known development fake. Only owned HTTP
// sessions/channels are created; no Dojo/control socket or application poke.
func TestLiveFakeZod(t *testing.T) {
	if os.Getenv("OMABIT_TEST_FAKE_ZOD") != "1" {
		t.Skip("set OMABIT_TEST_FAKE_ZOD=1 for the known localhost development fake")
	}
	c, err := New("http://127.0.0.1:18081", "~zod")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := c.Login(ctx, []byte(fakeCode)); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := c.Logout(context.Background()); err != nil {
			t.Errorf("owned session cleanup: %v", err)
		}
	}()
	data, err := c.Scry(ctx, "hood", "/kiln/pikes")
	if err != nil {
		t.Fatal(err)
	}
	var desks map[string]json.RawMessage
	if err := json.Unmarshal(data, &desks); err != nil {
		t.Fatal("unexpected desk inventory shape")
	}
	if _, ok := desks["base"]; !ok {
		t.Fatal("base desk missing")
	}
	if err := c.Watch(ctx, "hood", "/kiln/updates", 1, func(event Event) error { return nil }); err != nil {
		t.Fatal(err)
	}
	t.Logf("PASS: live fake ~zod login, authenticated identity, %d desks, SSE initial fact/ack/delete; secrets omitted", len(desks))
}

package eyre

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestStoredSessionHealthNeverChangesSessionOrFollowsRedirect(t *testing.T) {
	for _, kind := range []string{"healthy", "expired", "rejected", "wrong-host", "wrong-name", "host-denied", "redirect-host", "redirect-name", "oversized-host", "oversized-name", "html", "unavailable", "timeout", "bad-json", "duplicate", "unknown", "case-alias", "zero-expiry", "wrong-origin", "wrong-cookie", "cancelled"} {
		t.Run(kind, func(t *testing.T) {
			var calls, authenticated, forwarded atomic.Int32
			other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { forwarded.Add(1) }))
			defer other.Close()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != "GET" || r.URL.RawQuery != "" || r.ContentLength > 0 || r.URL.Path != "/~/host" && r.URL.Path != "/~/name" {
					t.Error("unexpected effect or application request", r.Method, r.URL.Path)
					w.WriteHeader(400)
					return
				}
				auth := r.URL.Path == "/~/name"
				if auth {
					authenticated.Add(1)
					if r.Header.Get("Cookie") != "urbauth-~zod=private-fixture-token" {
						t.Error("wrong origin-bound cookie")
					}
				} else if r.Header.Get("Cookie") != "" {
					t.Error("unauthenticated probe received cookie")
				}
				if kind == "timeout" {
					<-r.Context().Done()
					return
				}
				if kind == "rejected" && auth || kind == "host-denied" && !auth {
					w.WriteHeader(403)
					return
				}
				if kind == "redirect-host" && !auth || kind == "redirect-name" && auth {
					http.Redirect(w, r, other.URL, 302)
					return
				}
				if kind == "oversized-host" && !auth || kind == "oversized-name" && auth {
					fmt.Fprint(w, strings.Repeat("PRIVATE", 100))
					return
				}
				if kind == "wrong-host" && !auth || kind == "wrong-name" && auth {
					fmt.Fprint(w, "~bus")
					return
				}
				if kind == "html" {
					fmt.Fprint(w, "<html>PRIVATE login page</html>")
					return
				}
				fmt.Fprint(w, "~zod")
			}))
			defer server.Close()
			s := session{Origin: server.URL, Ship: "~zod", Name: "urbauth-~zod", Value: "private-fixture-token", Expires: time.Now().Add(time.Hour)}
			switch kind {
			case "expired":
				s.Expires = time.Now().Add(-time.Second)
			case "zero-expiry":
				s.Expires = time.Time{}
			case "wrong-origin":
				s.Origin = other.URL
			case "wrong-cookie":
				s.Name = "urbauth-~bus"
			case "unavailable":
				server.Close()
			}
			data, _ := json.Marshal(s)
			switch kind {
			case "bad-json":
				data = []byte("PRIVATE invalid session")
			case "duplicate":
				data = append([]byte(`{"ship":"~bus",`), data[1:]...)
			case "unknown":
				data = append([]byte(`{"code":"PRIVATE",`), data[1:]...)
			case "case-alias":
				data = bytes.Replace(data, []byte(`"ship"`), []byte(`"Ship"`), 1)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if kind == "cancelled" {
				cancel()
			}
			state, err := StoredSessionHealth(ctx, server.URL, "~zod", data)
			wantState, wantCalls, wantAuth := "", int32(0), int32(0)
			var wantError error
			switch kind {
			case "healthy", "rejected":
				wantState, wantCalls, wantAuth = kind, 2, 1
			case "expired":
				wantState = "expired"
			case "unavailable":
				wantState = "unreachable"
			case "timeout":
				wantState, wantCalls = "unreachable", 1
			case "host-denied":
				wantState, wantCalls = "unreachable", 1
			case "wrong-host":
				wantError, wantCalls = ErrIdentity, 1
			case "wrong-name":
				wantError, wantCalls, wantAuth = ErrIdentity, 2, 1
			case "redirect-host":
				wantError, wantCalls = ErrRedirect, 1
			case "redirect-name":
				wantError, wantCalls, wantAuth = ErrRedirect, 2, 1
			case "oversized-host", "html":
				wantError, wantCalls = ErrProtocol, 1
			case "oversized-name":
				wantError, wantCalls, wantAuth = ErrProtocol, 2, 1
			case "cancelled":
				wantError = context.Canceled
			default:
				wantError = ErrStoredSession
			}
			if state != wantState || !errors.Is(err, wantError) || calls.Load() != wantCalls || authenticated.Load() != wantAuth || forwarded.Load() != 0 {
				t.Fatalf("state=%q err=%v requests=%d auth=%d forwarded=%d", state, err, calls.Load(), authenticated.Load(), forwarded.Load())
			}
			if !bytes.Equal(data, make([]byte, len(data))) {
				t.Fatal("retained input buffer was not cleared")
			}
			if err != nil && (strings.Contains(err.Error(), "PRIVATE") || strings.Contains(err.Error(), "private-fixture")) {
				t.Fatal("private data escaped health result")
			}
		})
	}
}

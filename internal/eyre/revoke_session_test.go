package eyre

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRevokeStoredExpiredSessionWithoutRevokingParent(t *testing.T) {
	ctx := context.Background()
	parent, f := independentSessionServer(t)
	child, err := parent.NewIndependentSession(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer child.Close()
	child.session.Expires = time.Now().Add(-time.Hour)
	data, err := child.ExportSession()
	if err != nil {
		t.Fatal(err)
	}
	copy := append([]byte(nil), data...)
	if err := child.RestoreSession(copy); !errors.Is(err, ErrAuth) {
		t.Fatal("normal expiry bypassed", err)
	}
	if _, err := child.Scry(ctx, "settings", "/desk/talon"); !errors.Is(err, ErrAuth) {
		t.Fatal("expired data request", err)
	}
	outcome, err := RevokeStoredSession(ctx, parent.origin, parent.ship, data)
	if err != nil || outcome != "revoked" {
		t.Fatal(outcome, err)
	}
	for _, b := range data {
		if b != 0 {
			t.Fatal("private input not cleared")
		}
	}
	if err := parent.Check(ctx); err != nil {
		t.Fatal("parent revoked", err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.sessions) != 1 {
		t.Fatal("expired server session retained")
	}
}

func TestRevokeStoredSessionDistinguishesUncertainAndRejected(t *testing.T) {
	for _, status := range []int{200, 204, 303, 401, 403, 302, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			requests := 0
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/~/host" {
					if r.Header.Get("Cookie") != "" {
						t.Error("public identity leaked cookie")
					}
					fmt.Fprint(w, "~zod")
					return
				}
				if r.URL.Path != "/~/logout" || r.URL.RawQuery != "" || r.Method != "POST" {
					t.Error("unexpected request")
				}
				requests++
				if r.Header.Get("Cookie") != "urbauth-~zod=fixture" {
					t.Error("missing exact cookie")
				}
				w.Header().Set("Location", "/should-not-follow")
				w.WriteHeader(status)
			}))
			defer s.Close()
			data, _ := json.Marshal(session{Origin: s.URL, Ship: "~zod", Name: "urbauth-~zod", Value: "fixture", Expires: time.Now().Add(-time.Hour)})
			out, err := RevokeStoredSession(context.Background(), s.URL, "~zod", data)
			if requests != 1 {
				t.Fatal("logout retried/followed", requests)
			}
			switch status {
			case 200, 204, 303:
				if err != nil || out != "revoked" {
					t.Fatal(out, err)
				}
			case 401, 403:
				if err != nil || out != "rejected" {
					t.Fatal(out, err)
				}
			default:
				if err == nil || out != "" {
					t.Fatal("uncertain logout became success", out, err)
				}
			}
		})
	}
}

func TestRevokeStoredSessionRefusesWrongIdentity(t *testing.T) {
	requests := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Header.Get("Cookie") != "" {
			t.Error("cookie exposed before identity check")
		}
		fmt.Fprint(w, "~nec")
	}))
	defer s.Close()
	data, _ := json.Marshal(session{Origin: s.URL, Ship: "~zod", Name: "urbauth-~zod", Value: "fixture", Expires: time.Now().Add(time.Hour)})
	if out, err := RevokeStoredSession(context.Background(), s.URL, "~zod", data); !errors.Is(err, ErrIdentity) || out != "" || requests != 1 {
		t.Fatal(out, err, requests)
	}
}

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/credentials"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestLostReplyGuidanceDoesNotDispatchRecovery(t *testing.T) {
	for _, format := range []string{"json", "text"} {
		t.Run(format, func(t *testing.T) {
			var mutations atomic.Int32
			var reads atomic.Int32
			socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/hello" {
					reply(t, w, protocol.Hello{ProtocolVersion: 1})
					return
				}
				if r.Method != http.MethodPost {
					reads.Add(1)
					t.Errorf("unexpected recovery request: %s", r.URL.Path)
					return
				}
				mutations.Add(1)
				conn, _, err := w.(http.Hijacker).Hijack()
				if err != nil {
					t.Error(err)
					return
				}
				conn.Close()
			})
			var stdout, stderr bytes.Buffer
			code := run(context.Background(), []string{"--socket", socket, "--format", format, "operate", "alpha", "stop", "--revision=2", "--request-id=original-intent"}, &stdout, &stderr)
			if code != 1 || stdout.Len() != 0 || mutations.Load() != 1 || reads.Load() != 0 {
				t.Fatalf("code=%d mutations=%d reads=%d out=%s err=%s", code, mutations.Load(), reads.Load(), &stdout, &stderr)
			}
			if format == "json" {
				var response guidedErrorResponse
				if err := json.Unmarshal(stderr.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				if response.Error.Code != "client_error" || response.Guidance == nil || response.Guidance.Kind != "reconcile_uncertain_request" {
					t.Fatalf("response=%+v", response)
				}
				if len(response.Guidance.Next) != 2 || response.Guidance.Next[0].Name != "snapshot" {
					t.Fatalf("unsafe/missing navigation: %+v", response.Guidance)
				}
			} else if !strings.Contains(stderr.String(), "Next (reconcile_uncertain_request)") || !strings.Contains(stderr.String(), "original request ID") {
				t.Fatal(stderr.String())
			}
		})
	}
}

func TestGuidanceUsesTypedErrorsAndPreservesIdentityRefusal(t *testing.T) {
	rejected := &client.RemoteError{Status: 409, Problem: protocol.APIError{Code: "request_conflict", Message: "outcome uncertain; remote text is not a type"}}
	if guidanceFor(rejected, rejected.Problem) != nil {
		t.Fatal("remote message text selected uncertain-outcome guidance")
	}
	stale := &client.RemoteError{Status: 409, Problem: protocol.APIError{Code: "revision_conflict", Message: "changed"}}
	if got := guidanceFor(stale, stale.Problem); got == nil || got.Kind != "review_current_revision" {
		t.Fatalf("stale: %+v", got)
	}
	// A connection identity failure can be wrapped during mutation dispatch.
	// Never suggest reading that replacement endpoint as the recovery path.
	err := &client.OutcomeUncertainError{Method: "POST", Path: "/v1/instances/alpha/operations", Err: fmt.Errorf("handshake: %w", client.ErrIdentityMismatch)}
	got := guidanceFor(err, protocol.APIError{})
	if got == nil || got.Kind != "resolve_runner_identity" {
		t.Fatalf("identity: %+v", got)
	}
	for _, next := range got.Next {
		if next.Name == "snapshot" || next.Name == "enroll" {
			t.Fatalf("unsafe identity recovery: %+v", got)
		}
	}
}

func TestShipGuidanceKeepsIdentityAndStoreFailuresDistinct(t *testing.T) {
	for _, tc := range []struct {
		err  error
		kind string
	}{
		{eyre.ErrIdentity, "resolve_ship_origin"}, {eyre.ErrRedirect, "resolve_ship_origin"},
		{credentials.ErrStore, "restore_credential_access"}, {credentials.ErrMissing, "review_ship_login"},
		{eyre.ErrAuth, "review_ship_login"},
		{errors.Join(eyre.ErrIdentity, eyre.ErrAuth), "resolve_ship_origin"},
		{errors.Join(credentials.ErrStore, credentials.ErrMissing), "restore_credential_access"},
	} {
		got := guidanceFor(fmt.Errorf("wrapped: %w", tc.err), protocol.APIError{})
		if got == nil || got.Kind != tc.kind {
			t.Fatalf("%v: %+v", tc.err, got)
		}
		for _, next := range got.Next {
			if next.Name != "ships" && next.Name != "help connections" && next.Name != "ship-login --help" {
				t.Fatalf("unexpected action: %+v", got)
			}
		}
	}
	if guidanceFor(errors.New(eyre.ErrAuth.Error()), protocol.APIError{}) != nil {
		t.Fatal("remote/message text chose a typed login action")
	}
	if guidanceFor(eyre.ErrTransport, protocol.APIError{}) != nil {
		t.Fatal("outage misclassified as expired login")
	}
}

func TestMissingShipSessionGuidanceIsOfflineAndKeepsEnrollment(t *testing.T) {
	for _, format := range []string{"json", "text"} {
		t.Run(format, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "registry")
			var stdout, stderr bytes.Buffer
			// Enrollment verifies public identity once. Missing credentials must
			// then fail locally without another HTTP request or a login attempt.
			var requests atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.Method != "GET" || r.URL.Path != "/~/host" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(500)
					return
				}
				fmt.Fprint(w, "~zod")
			}))
			defer srv.Close()
			if code := run(context.Background(), []string{"--config-dir", dir, "ship-enroll", "lab", "--url", srv.URL, "--ship", "~zod", "--credentials", "file"}, &stdout, &stderr); code != 0 {
				t.Fatalf("enroll %d: %s", code, &stderr)
			}
			stdout.Reset()
			stderr.Reset()
			code := run(context.Background(), []string{"--config-dir", dir, "--format", format, "ship-scry", "lab", "--app", "hood", "--path", "/kiln/pikes"}, &stdout, &stderr)
			if code != 1 || stdout.Len() != 0 {
				t.Fatalf("scry code=%d out=%s err=%s", code, &stdout, &stderr)
			}
			if format == "json" {
				var value guidedErrorResponse
				if json.Unmarshal(stderr.Bytes(), &value) != nil || value.Error.Code != "client_error" || value.Guidance == nil || value.Guidance.Kind != "review_ship_login" {
					t.Fatal(stderr.String())
				}
			} else if !strings.Contains(stderr.String(), "Next (review_ship_login)") {
				t.Fatal(stderr.String())
			}
			stdout.Reset()
			stderr.Reset()
			if code := run(context.Background(), []string{"--config-dir", dir, "ships"}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "~zod") || !strings.Contains(stdout.String(), srv.URL) {
				t.Fatalf("enrollment changed: %s %s", &stdout, &stderr)
			}
			if requests.Load() != 1 {
				t.Fatalf("missing-session guidance caused HTTP dispatch: %d", requests.Load())
			}
		})
	}
}

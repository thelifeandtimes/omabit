//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestHostHTTPClientPolicy(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != "GET" || r.URL.RequestURI() != "/~/host" || r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
			t.Errorf("unexpected probe request: %s %s", r.Method, r.URL)
		}
		w.Header().Set("Set-Cookie", "urbauth=DO-NOT-STORE")
		io.WriteString(w, "~zod\n")
	}))
	defer server.Close()
	for _, name := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY"} {
		t.Setenv(name, "http://127.0.0.1:1")
	}
	for range 2 {
		if err := probeHostHTTP(context.Background(), server.URL+"/~/host"); err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 2 {
		t.Fatal("host probes were redirected through a proxy")
	}
}

func TestHostHTTPClientRejectsUnexpectedResponses(t *testing.T) {
	for _, tc := range []struct {
		name     string
		status   int
		body     string
		encoding string
		pending  bool
	}{
		{"wrong-identity", 200, "~nec", "", false},
		{"oversize", 200, "~zod" + strings.Repeat(" ", 200), "", false},
		{"auth-required", 403, "SECRET", "", false},
		{"empty", 204, "", "", false},
		{"gzip", 200, "~zod", "gzip", false},
		{"missing", 404, "SECRET", "", true},
		{"starting", 503, "SECRET", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if tc.encoding != "" {
					w.Header().Set("Content-Encoding", tc.encoding)
				}
				w.WriteHeader(tc.status)
				io.WriteString(w, tc.body)
			}))
			defer server.Close()
			err := probeHostHTTP(context.Background(), server.URL+"/~/host")
			if err == nil || strings.Contains(err.Error(), "SECRET") || errors.Is(err, errHostNotReady) != tc.pending {
				t.Fatalf("response policy: %v", err)
			}
		})
	}
}

func TestHostHTTPNeverFollowsRedirects(t *testing.T) {
	var followed atomic.Int32
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		followed.Add(1)
		io.WriteString(w, "~zod")
	}))
	defer destination.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destination.URL+"/~/host", http.StatusFound)
	}))
	defer server.Close()
	if err := probeHostHTTP(context.Background(), server.URL+"/~/host"); err == nil || errors.Is(err, errHostNotReady) || followed.Load() != 0 {
		t.Fatalf("redirect followed or retried: %v", err)
	}
}

func TestHostHTTPEndpointValidationAndCancellation(t *testing.T) {
	for _, endpoint := range []string{
		"http://example.invalid:8080/~/host", "https://127.0.0.1:8080/~/host", "http://127.0.0.1:80/~/host",
		"http://127.0.0.1:08080/~/host", "http://user:secret@127.0.0.1:8080/~/host", "http://[::1]:8080/~/host",
		"http://127.0.0.1:8080/~/host?query", "http://127.0.0.1:8080/~/login", "http://127.0.0.1:8080/~/host#fragment",
	} {
		if err := probeHostHTTP(context.Background(), endpoint); err == nil || errors.Is(err, errHostNotReady) {
			t.Fatalf("invalid endpoint reached transport: %s, %v", endpoint, err)
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	if err := probeHostHTTP(ctx, server.URL+"/~/host"); !errors.Is(err, errHostNotReady) {
		t.Fatalf("cancellation: %v", err)
	}
}

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestShipCLIIndependentSessionsAndLogout(t *testing.T) {
	const code = "lidlut-tabwed-pillex-ridrup"
	const token = "0vSENSITIVESESSION"
	var mu sync.Mutex
	valid := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.URL.Path == "/~/host" {
			fmt.Fprint(w, "~zod")
			return
		}
		if r.URL.Path == "/~/login" {
			valid = true
			http.SetCookie(w, &http.Cookie{Name: "urbauth-~zod", Value: token, Path: "/", MaxAge: 3600})
			fmt.Fprint(w, token)
			return
		}
		cookie, err := r.Cookie("urbauth-~zod")
		if err != nil || cookie.Value != token || !valid {
			w.WriteHeader(401)
			return
		}
		if r.URL.Path == "/~/name" {
			fmt.Fprint(w, "~zod")
			return
		}
		if r.URL.Path == "/~/logout" {
			valid = false
			w.Header().Set("Location", "/~/login")
			w.WriteHeader(303)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"base":{"zest":"live"}}`)
	}))
	defer srv.Close()
	dir := filepath.Join(t.TempDir(), "collector")
	invoke := func(command string, args []string, input io.Reader, want int) string {
		t.Helper()
		var stdout, stderr bytes.Buffer
		fail := func(err error, code int) int { fmt.Fprintln(&stderr, err); return code }
		got := runShips(context.Background(), command, args, dir, "", "", "text", input, &stdout, fail)
		if got != want {
			t.Fatalf("%s: exit=%d stdout=%s stderr=%s", command, got, &stdout, &stderr)
		}
		if strings.Contains(stdout.String()+stderr.String(), token) || strings.Contains(stdout.String()+stderr.String(), code) {
			t.Fatal("CLI exposed credential")
		}
		return stdout.String()
	}
	invoke("ship-enroll", []string{"test", "--url", srv.URL, "--ship", "~zod", "--credentials", "file"}, nil, 0)
	invoke("ship-status", []string{"test"}, nil, 1)
	input := &neverRead{}
	invoke("ship-login", []string{"test", "--code-stdin", "--expected-ship-id", "stale-selection"}, input, 1)
	if input.read {
		t.Fatal("stale UI selection consumed a login code")
	}
	invoke("ship-login", []string{"test", "--code-stdin"}, strings.NewReader(code), 0)
	if text := invoke("ship-status", []string{"test"}, nil, 0); !strings.Contains(text, "authenticated") {
		t.Fatal(text)
	}
	invoke("ship-scry", []string{"test", "--app", "hood", "--path", "/kiln/pikes"}, nil, 0)
	data, err := os.ReadFile(filepath.Join(dir, "ships.json"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte(token)) || bytes.Contains(data, []byte(code)) {
		t.Fatal("registry contains credentials")
	}
	var registry any
	if err := json.Unmarshal(data, &registry); err != nil {
		t.Fatal(err)
	}
	// A failed deletion must leave cleanup intent that another invocation can
	// retry, rather than losing the only reference to the secret.
	stored, err := os.ReadDir(filepath.Join(dir, "credentials"))
	if err != nil || len(stored) != 1 {
		t.Fatal("expected one stored session")
	}
	credentialPath := filepath.Join(dir, "credentials", stored[0].Name())
	if err := os.Chmod(credentialPath, 0644); err != nil {
		t.Fatal(err)
	}
	invoke("ship-logout", []string{"test"}, nil, 1)
	invoke("ship-forget", []string{"test", "--url", srv.URL}, nil, 1)
	if err := os.Chmod(credentialPath, 0600); err != nil {
		t.Fatal(err)
	}
	invoke("ship-logout", []string{"test"}, nil, 0)
	invoke("ship-status", []string{"test"}, nil, 1)
	invoke("ship-forget", []string{"test", "--url", srv.URL}, nil, 0)
	files, err := os.ReadDir(filepath.Join(dir, "credentials"))
	if err != nil || len(files) != 0 {
		t.Fatalf("session material retained: %d %v", len(files), err)
	}
}

type neverRead struct{ read bool }

func (r *neverRead) Read(p []byte) (int, error) { r.read = true; return 0, io.EOF }

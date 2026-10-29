package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/applicationauth"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
)

func TestApplicationCLIApprovalRecoveryAndInstalledUpdate(t *testing.T) {
	ctx := context.Background()
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, key := range []string{"XDG_CONFIG_HOME", "XDG_STATE_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_RUNTIME_DIR", "DBUS_SESSION_BUS_ADDRESS"} {
		t.Setenv(key, "")
	}
	root := filepath.Join(home, ".config/omarchy/plugins/fixture.auth-lab")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	script := `import json,sys
v=json.load(sys.stdin)
reply={k:v[k] for k in ('version','request_id','connection_id','plugin_id','profile')}
if sys.argv[1:]==['--reconcile']:
    reply.update(status='reconciled',outcome='removed')
else:
    assert v['session']['cookie_value']=='fixture-cookie-2'
    reply.update(status='accepted')
print(json.dumps(reply))
`
	declaration, _ := json.Marshal(applicationauth.Declaration{Version: 1, Runtime: "python3", Authority: applicationauth.OwnerAuthority, Entry: "receiver.py", Files: []string{"manifest.json", "application-connection.json", "receiver.py"}})
	for name, data := range map[string]string{"manifest.json": `{"id":"fixture.auth-lab","name":"Fixture Auth Lab"}`, "application-profile.json": `{"version":1,"profile":"auth-lab"}`, "application-connection.json": string(declaration), "receiver.py": script} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	var mu sync.Mutex
	logins := 0
	sessions := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.URL.Path == "/~/host" {
			fmt.Fprint(w, "~zod")
			return
		}
		cookie, _ := r.Cookie("urbauth-~zod")
		if r.URL.Path == "/~/login" {
			if cookie != nil {
				t.Error("new login inherited parent cookie")
			}
			logins++
			token := fmt.Sprintf("fixture-cookie-%d", logins)
			sessions[token] = true
			http.SetCookie(w, &http.Cookie{Name: "urbauth-~zod", Value: token, Path: "/", MaxAge: 3600})
			return
		}
		if cookie == nil || !sessions[cookie.Value] {
			w.WriteHeader(403)
			return
		}
		switch r.URL.Path {
		case "/~/name":
			fmt.Fprint(w, "~zod")
		case "/~/logout":
			delete(sessions, cookie.Value)
			w.Header().Set("Location", "/~/login")
			w.WriteHeader(303)
		case "/spider/base/json/code/ship":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `"lidlut-tabwed-pillex-ridrup"`)
		default:
			t.Errorf("unexpected fixture request %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	dir := filepath.Join(t.TempDir(), "collector")
	registry, err := collection.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	ship, err := registry.EnrollShip(ctx, "lab", server.URL, "~zod", "file")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = loginShip(ctx, registry, ship, []byte("lidlut-tabwed-pillex-ridrup")); err != nil {
		t.Fatal(err)
	}
	ship, err = registry.Ship(ctx, "lab")
	if err != nil {
		t.Fatal(err)
	}
	invoke := func(want int, args ...string) string {
		t.Helper()
		var out, stderr bytes.Buffer
		actual := run(ctx, append([]string{"--config-dir", dir, "--format", "json"}, args...), &out, &stderr)
		if actual != want {
			t.Fatalf("%v exit=%d expected=%d stdout=%s stderr=%s", args, actual, want, &out, &stderr)
		}
		for _, secret := range []string{"fixture-cookie-", "lidlut-tabwed-pillex-ridrup", "session_ref"} {
			if strings.Contains(out.String()+stderr.String(), secret) {
				t.Fatal("private data in CLI output", secret)
			}
		}
		return out.String()
	}
	var plugins []applicationauth.InstalledPlugin
	if err := json.Unmarshal([]byte(invoke(0, "app-plugins")), &plugins); err != nil || len(plugins) != 1 {
		t.Fatal(plugins, err)
	}
	id := strings.Repeat("a", 32)
	args := []string{"app-connect", "lab", "--expected-ship-id", ship.ID, "--expected-ship-revision", strconv.FormatUint(ship.Revision, 10), "--plugin-id", plugins[0].Target.PluginID, "--profile", "auth-lab", "--artifact-sha256", plugins[0].Target.SHA256, "--request-id", id}
	invoke(2, args...) // No approval: no cache, journal or new session.
	if _, err := os.Stat(registry.ApplicationCacheDir()); !os.IsNotExist(err) {
		t.Fatal("unapproved helper retained")
	}
	stale := append([]string(nil), args...)
	stale[5] = "999"
	stale = append(stale, "--approve-owner-session")
	if text := invoke(1, stale...); !strings.Contains(text, "selection_changed") {
		t.Fatal(text)
	}
	args = append(args, "--approve-owner-session")
	var result applicationResult
	if err := json.Unmarshal([]byte(invoke(0, args...)), &result); err != nil || result.Connection == nil || result.Connection.Phase != "connected" {
		t.Fatal(result, err)
	}
	invoke(0, args...) // Same request cannot issue/deliver again.
	mu.Lock()
	if logins != 2 {
		t.Error("duplicate/stale approval issued a session", logins)
	}
	mu.Unlock()
	current := result.Connection
	if text := invoke(1, "app-disconnect", id, "--expected-revision", "1"); !strings.Contains(text, "selection_changed") {
		t.Fatal(text)
	}
	// Recovery preserves accepted sessions and uses no receiver.
	invoke(0, "app-recover", id, "--expected-revision", strconv.FormatUint(current.Revision, 10))
	if err := os.WriteFile(filepath.Join(root, "receiver.py"), []byte("raise Exception('new unapproved version executed')"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(invoke(0, "app-disconnect", id, "--expected-revision", strconv.FormatUint(current.Revision, 10))), &result); err != nil {
		t.Fatal(err)
	}
	if result.Connection.Phase != "closed" || !result.Connection.ReceiverReconciliation {
		t.Fatal(result)
	}
	if err := json.Unmarshal([]byte(invoke(0, "app-reconcile", id, "--expected-revision", strconv.FormatUint(result.Connection.Revision, 10))), &result); err != nil {
		t.Fatal(err)
	}
	if result.Connection.ReceiverReconciliation || result.Connection.ReceiverOutcome != "removed" {
		t.Fatal(result)
	}
	mu.Lock()
	if !sessions["fixture-cookie-1"] || sessions["fixture-cookie-2"] {
		t.Error("disconnect targeted wrong session")
	}
	mu.Unlock()
	text := invoke(0, "app-connections")
	if !strings.Contains(text, `"app-archive"`) {
		t.Fatal(text)
	}
	var out, stderr bytes.Buffer
	if exit := run(ctx, []string{"--config-dir", dir, "--format", "text", "app-connections"}, &out, &stderr); exit != 0 || !strings.Contains(out.String(), "fixture.auth-lab / auth-lab") {
		t.Fatal(exit, &out, &stderr)
	}
	archiveArgs := []string{"app-archive", id, "--expected-revision", strconv.FormatUint(result.Connection.Revision, 10)}
	invoke(0, archiveArgs...)
	invoke(0, archiveArgs...)
	if text := invoke(0, "app-connections"); strings.TrimSpace(text) != "[]" {
		t.Fatal(text)
	}
	if text := invoke(0, "app-connection", id); !strings.Contains(text, `"archived": true`) {
		t.Fatal(text)
	}
	if text := invoke(1, args...); !strings.Contains(text, `"code": "archived"`) {
		t.Fatal("archived approval replayed", text)
	}
	for _, cmd := range []string{"app-plugins", "app-connections", "app-connection", "app-archive", "app-connect", "app-disconnect", "app-recover", "app-reconcile"} {
		if text := invoke(0, cmd, "--help"); !strings.Contains(text, "Eyre does not isolate desks") {
			t.Fatal(text)
		}
	}
}

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/applicationauth"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"
)

func TestApplicationShipCLIReviewGrantConflictAndOfflineRevoke(t *testing.T) {
	ctx := context.Background()
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, k := range []string{"XDG_CONFIG_HOME", "XDG_STATE_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_RUNTIME_DIR", "DBUS_SESSION_BUS_ADDRESS"} {
		t.Setenv(k, "")
	}
	root := filepath.Join(home, ".config/omarchy/plugins/fixture.auth-lab")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	declaration, _ := json.Marshal(applicationauth.Declaration{Version: 1, Runtime: "python3", Authority: applicationauth.OwnerAuthority, Entry: "receiver.py", Files: []string{"manifest.json", "application-connection.json", "receiver.py"}})
	for name, data := range map[string]string{"manifest.json": `{"id":"fixture.auth-lab","name":"Fixture"}`, "application-profile.json": `{"version":1,"profile":"auth-lab"}`, "application-connection.json": string(declaration), "receiver.py": "raise Exception('approval must not execute me')\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	env, err := applicationEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	artifact, _, err := applicationauth.ResolveInstalled(filepath.Dir(root), "fixture.auth-lab", env)
	if err != nil {
		t.Fatal(err)
	}
	binding := shiproute.Binding{RunnerID: "runner-fixture", Target: apppolicy.Target{InstanceID: "alpha"}, Ship: "~zod", Continuity: strings.Repeat("c", 64)}
	selection := protocol.ShipRouteSelection{Target: binding.Target, RuntimeID: strings.Repeat("a", 64), ConfigRevision: 3}
	var offline, replaced atomic.Bool
	var calls atomic.Int32
	socket := fakeRunner(t, func(w http.ResponseWriter, q *http.Request) {
		calls.Add(1)
		if offline.Load() {
			w.WriteHeader(503)
			return
		}
		switch q.URL.Path {
		case "/hello":
			reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: binding.RunnerID, Mode: "simulation", AccessRole: "operator", Features: []string{protocol.AppPolicyFeature, protocol.ShipRoutesFeature}})
		case protocol.AppPolicyPath:
			reply(t, w, protocol.AppPolicyStatus{Version: 1, RunnerID: binding.RunnerID, Mode: "simulation", ReadAt: time.Now().UTC(), Policy: apppolicy.Default(), Targets: []protocol.AppPolicyTarget{{Target: binding.Target, Ship: binding.Ship, Label: "alpha", Ready: true}}})
		case "/v1/snapshot":
			reply(t, w, protocol.Snapshot{RunnerID: binding.RunnerID, Mode: "simulation", Instances: []protocol.Instance{{ID: "alpha", ConfigRevision: 3, ObservedState: "running", Resources: &protocol.ResourceMetrics{ContainerID: selection.RuntimeID}}}})
		case protocol.ShipRoutesPath:
			if q.Method != "POST" {
				t.Error("approval changed routes")
			}
			b := binding
			if replaced.Load() {
				b.Continuity = strings.Repeat("d", 64)
			}
			record := shiproute.DefaultRecord(b)
			registry, _ := record.Settings.Effective(b, 0, nil)
			reply(t, w, protocol.ShipRoutes{Version: 1, RunnerID: b.RunnerID, Mode: "simulation", Selection: selection, Record: record, Registry: registry, ReadAt: time.Now().UTC()})
		default:
			t.Error("approval performed an unexpected RPC", q.URL.Path)
			w.WriteHeader(400)
		}
	})
	dir := filepath.Join(t.TempDir(), "collector")
	r, err := collection.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.Enroll(ctx, "fixture", socket); err != nil {
		t.Fatal(err)
	}
	invoke := func(want int, args ...string) string {
		t.Helper()
		var out, stderr bytes.Buffer
		got := run(ctx, append([]string{"--config-dir", dir, "--format", "json"}, args...), &out, &stderr)
		if got != want {
			t.Fatalf("%v: exit %d want %d output=%s error=%s", args, got, want, &out, &stderr)
		}
		if strings.Contains(out.String()+stderr.String(), "session_ref") {
			t.Fatal("credential reference exposed")
		}
		return out.String()
	}
	pin := []string{"--runner", "fixture", "--expected-runner-id", binding.RunnerID}
	var review collection.ApplicationShipReview
	if err = json.Unmarshal([]byte(invoke(0, append(pin, "app-ship-review", "--instance", "alpha")...)), &review); err != nil || review.Routes.Record.Binding != binding || review.Approvals.Revision != 0 {
		t.Fatal(review, err)
	}
	allowed := true
	u := collection.ApplicationShipApprovalRequest{Version: 1, Binding: binding, Target: artifact.Target(), Allowed: &allowed, Selection: &selection}
	apply := func(want int, ack bool) {
		t.Helper()
		body, _ := json.Marshal(u)
		args := []string{"app-ship-approve", "--request-json", string(body)}
		if *u.Allowed {
			args = append(append([]string{}, pin...), args...)
		}
		if ack {
			args = append(args, "--allow-owner-session")
		}
		invoke(want, args...)
	}
	before := calls.Load()
	apply(2, false)
	if calls.Load() != before {
		t.Fatal("unacknowledged grant contacted runner")
	}
	state, err := r.ApplicationShipApprovals(ctx)
	if err != nil || state.Revision != 0 {
		t.Fatal("unacknowledged grant persisted", state, err)
	}
	apply(0, true)
	state, _ = r.ApplicationShipApprovals(ctx)
	if !state.Allows(binding, u.Target) || state.Revision != 1 {
		t.Fatal(state)
	}
	apply(1, true) // same stale revision is not permission to rewrite
	u.ExpectedRevision = 1
	replaced.Store(true)
	apply(1, true)
	state, _ = r.ApplicationShipApprovals(ctx)
	if state.Revision != 1 {
		t.Fatal("replacement inherited a grant")
	}
	replaced.Store(false)
	if err = os.WriteFile(filepath.Join(root, "receiver.py"), []byte("raise Exception('updated helper')\n"), 0600); err != nil {
		t.Fatal(err)
	}
	before = calls.Load()
	apply(1, true)
	if calls.Load() != before {
		t.Fatal("changed helper contacted runner")
	}
	// Offline/uninstalled revocation is local and leaves old connection journals alone.
	offline.Store(true)
	if err = os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	allowed = false
	u.Selection = nil
	before = calls.Load()
	apply(0, false)
	if calls.Load() != before {
		t.Fatal("revocation depended on unavailable runner")
	}
	state, _ = r.ApplicationShipApprovals(ctx)
	if state.Revision != 2 || state.Allows(binding, u.Target) {
		t.Fatal("revocation failed", state)
	}
	var listed collection.ApplicationShipApprovalView
	if err = json.Unmarshal([]byte(invoke(0, "app-ship-approvals")), &listed); err != nil || listed.Revision != 2 || len(listed.Entries) != 1 || listed.Entries[0].Allowed {
		t.Fatal(listed, err)
	}
	for _, name := range []string{"ships.json", "applications.json", "application-receivers"} {
		if _, err = os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatal("approval created credential/attempt state", name, err)
		}
	}
}

func TestApplicationShipCLIRefusesInvalidInputBeforeOpeningState(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "never-created")
	for _, args := range [][]string{{"app-ship-review"}, {"app-ship-approve"}, {"app-ship-approve", "--request-json", "{}"}, {"app-ship-approve", "--request-json", "{}", "--request-file", "/no-file"}, {"app-ship-approvals", "--allow-owner-session"}, {"app-ship-approvals", "extra"}} {
		var out, err bytes.Buffer
		if code := run(context.Background(), append([]string{"--config-dir", dir}, args...), &out, &err); code != 2 {
			t.Fatal(args, code, err.String())
		}
	}
	for _, cmd := range []string{"app-ship-review", "app-ship-approvals", "app-ship-approve"} {
		var out, err bytes.Buffer
		if code := run(context.Background(), []string{"--config-dir", dir, cmd, "--help"}, &out, &err); code != 0 || !strings.Contains(out.String(), "standing") {
			t.Fatal(cmd, code, out.String(), err.String())
		}
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("invalid input or help created state", err)
	}
}

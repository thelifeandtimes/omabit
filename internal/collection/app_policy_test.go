package collection

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type policyFixture struct {
	mu      sync.Mutex
	policy  apppolicy.Policy
	offline bool
	puts    int
	drop    bool
}

func policyRunner(t *testing.T, r *Registry, name string) *policyFixture {
	t.Helper()
	f := &policyFixture{policy: apppolicy.Default()}
	socket := filepath.Join(t.TempDir(), "runner.sock")
	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		w.Header().Set(protocol.Header, "1")
		if f.offline {
			http.Error(w, "offline", 503)
			return
		}
		if q.URL.Path == "/hello" {
			json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-" + name, Mode: "simulation", AccessRole: "operator", Features: []string{protocol.AppPolicyFeature}})
			return
		}
		if q.URL.Path != protocol.AppPolicyPath {
			t.Error(q.URL.Path)
			http.NotFound(w, q)
			return
		}
		if q.Method == "PUT" {
			var u apppolicy.Update
			if json.NewDecoder(q.Body).Decode(&u) != nil {
				t.Error("invalid request")
			}
			if err := f.policy.Apply(u); err != nil {
				http.Error(w, err.Error(), 409)
				return
			}
			f.puts++
			if f.drop {
				f.drop = false
				http.Error(w, "lost reply", 503)
				return
			}
			json.NewEncoder(w).Encode(f.policy)
			return
		}
		json.NewEncoder(w).Encode(protocol.AppPolicyStatus{Version: 1, RunnerID: "runner-" + name, Mode: "simulation", ReadAt: time.Now().UTC(), Policy: f.policy, Targets: []protocol.AppPolicyTarget{}})
	})}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })
	if _, err = r.Enroll(context.Background(), name, socket); err != nil {
		t.Fatal(err)
	}
	return f
}
func rolloutTargets(v AppDevelopmentStatus) []AppPolicyRollout {
	out := []AppPolicyRollout{}
	for _, r := range v.Runners {
		var revision *uint64
		if r.Status != nil {
			n := r.Status.Policy.Revision
			revision = &n
		}
		out = append(out, AppPolicyRollout{Name: r.Name, RunnerID: r.RunnerID, ExpectedRevision: revision})
	}
	return out
}
func TestAppPolicyGlobalDurableOfflineRolloutAndCAS(t *testing.T) {
	ctx := context.Background()
	r := registry(t)
	a := policyRunner(t, r, "a")
	b := policyRunner(t, r, "b")
	v, err := r.AppDevelopment(ctx)
	if err != nil || v.Global.Gates != apppolicy.Blocked() {
		t.Fatal(v, err)
	}
	g, err := r.SaveAppPolicyGlobal(ctx, 0, apppolicy.Gates{CreateDesk: apppolicy.Block, Commit: apppolicy.Allow, Publish: apppolicy.Block}, rolloutTargets(v))
	if err != nil {
		t.Fatal(err)
	}
	b.mu.Lock()
	b.offline = true
	b.mu.Unlock()
	if err = r.ApplyAppPolicyGlobal(ctx, g.Revision); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(r.dir)
	if err != nil {
		t.Fatal(err)
	}
	v, err = reopened.AppDevelopment(ctx)
	if err != nil || !v.Global.Runners[0].Applied || v.Global.Runners[1].Applied || v.Runners[1].Status != nil {
		t.Fatal(v, err)
	}
	b.mu.Lock()
	b.offline = false
	b.mu.Unlock()
	if err = reopened.ApplyAppPolicyGlobal(ctx, g.Revision); err != nil {
		t.Fatal(err)
	}
	b.mu.Lock()
	if b.policy.Global.Commit != apppolicy.Allow || b.puts != 1 {
		t.Error(b.policy, b.puts)
	}
	b.mu.Unlock()
	// A scoped change on a different collector causes a stale global rollout to
	// stop for review, while preserving its overrides and the original intent.
	v, err = r.AppDevelopment(ctx)
	if err != nil {
		t.Fatal(err)
	}
	g, err = r.SaveAppPolicyGlobal(ctx, g.Revision, apppolicy.Blocked(), rolloutTargets(v))
	if err != nil {
		t.Fatal(err)
	}
	a.mu.Lock()
	a.policy.Apply(apppolicy.Update{ExpectedRevision: a.policy.Revision, Scope: "runner", Gates: apppolicy.Gates{CreateDesk: apppolicy.Inherit, Commit: apppolicy.Allow, Publish: apppolicy.Inherit}})
	a.mu.Unlock()
	if err = r.ApplyAppPolicyGlobal(ctx, g.Revision); err != nil {
		t.Fatal(err)
	}
	v, err = r.AppDevelopment(ctx)
	if err != nil || v.Global.Runners[0].Applied || v.Global.Runners[0].Error == "" || !v.Global.Runners[1].Applied {
		t.Fatal(v, err)
	}
	a.mu.Lock()
	if a.policy.Global.Commit != apppolicy.Allow || a.policy.Runner.Commit != apppolicy.Allow {
		t.Error("overwrote concurrent edit")
	}
	a.mu.Unlock()
	// Enrollment after the rollout does not inherit a locally saved allow.
	c := policyRunner(t, r, "c")
	if err = r.ApplyAppPolicyGlobal(ctx, g.Revision); err != nil {
		t.Fatal(err)
	}
	c.mu.Lock()
	if c.puts != 0 || c.policy.Global != apppolicy.Blocked() {
		t.Error("new enrollment granted")
	}
	c.mu.Unlock()
	if _, err = r.SaveAppPolicyGlobal(ctx, 0, apppolicy.Blocked(), nil); err == nil {
		t.Fatal("stale local authoring accepted")
	}
}
func TestAppPolicyRolloutLostResponseAndUnreviewedOfflineRunner(t *testing.T) {
	ctx := context.Background()
	r := registry(t)
	a := policyRunner(t, r, "a")
	b := policyRunner(t, r, "b")
	b.mu.Lock()
	b.offline = true
	b.mu.Unlock()
	v, err := r.AppDevelopment(ctx)
	if err != nil {
		t.Fatal(err)
	}
	g, err := r.SaveAppPolicyGlobal(ctx, 0, apppolicy.Gates{CreateDesk: apppolicy.Block, Commit: apppolicy.Allow, Publish: apppolicy.Allow}, rolloutTargets(v))
	if err != nil {
		t.Fatal(err)
	}
	a.mu.Lock()
	a.drop = true
	a.mu.Unlock()
	if err = r.ApplyAppPolicyGlobal(ctx, g.Revision); err != nil {
		t.Fatal(err)
	}
	b.mu.Lock()
	b.offline = false
	b.mu.Unlock()
	if err = r.ApplyAppPolicyGlobal(ctx, g.Revision); err != nil {
		t.Fatal(err)
	}
	v, err = r.AppDevelopment(ctx)
	if err != nil || !v.Global.Runners[0].Applied || v.Global.Runners[1].Applied {
		t.Fatal(v, err)
	}
	a.mu.Lock()
	if a.puts != 1 {
		t.Error("lost response repeated mutation", a.puts)
	}
	a.mu.Unlock()
	b.mu.Lock()
	if b.puts != 0 {
		t.Error("unreviewed grant")
	}
	b.mu.Unlock()
}

func TestAppPolicyAcknowledgementSurvivesOfflineDisable(t *testing.T) {
	ctx := context.Background()
	r := registry(t)
	a := policyRunner(t, r, "a")
	v, err := r.AppDevelopment(ctx)
	if err != nil {
		t.Fatal(err)
	}
	g, err := r.SaveAppPolicyGlobal(ctx, 0, apppolicy.Gates{CreateDesk: apppolicy.Block, Commit: apppolicy.Allow, Publish: apppolicy.Allow}, rolloutTargets(v))
	if err != nil {
		t.Fatal(err)
	}
	if err = r.ApplyAppPolicyGlobal(ctx, g.Revision); err != nil {
		t.Fatal(err)
	}
	a.mu.Lock()
	a.offline = true
	a.mu.Unlock()
	v, err = r.AppDevelopment(ctx)
	if err != nil {
		t.Fatal(err)
	}
	g, err = r.SaveAppPolicyGlobal(ctx, g.Revision, apppolicy.Blocked(), rolloutTargets(v))
	if err != nil {
		t.Fatal(err)
	}
	if err = r.ApplyAppPolicyGlobal(ctx, g.Revision); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(r.dir)
	if err != nil {
		t.Fatal(err)
	}
	v, err = reopened.AppDevelopment(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ack := v.Runners[0].Acknowledged
	if ack == nil || ack.Revision != 1 || ack.Global.Commit != apppolicy.Allow || v.Global.Runners[0].Applied || v.Runners[0].Status != nil {
		t.Fatal("offline pending disable hid last acknowledged allow", v)
	}
}

func TestAppPolicyLegacyRolloutAddsClosedCreationWithoutWriting(t *testing.T) {
	ctx := context.Background()
	r, err := Open(filepath.Join(t.TempDir(), "collector"))
	if err != nil {
		t.Fatal(err)
	}
	legacy := json.RawMessage(`{"version":1,"revision":3,"gates":{"commit":"allow","publish":"block"},"runners":[{"name":"saved","runner_id":"runner-saved","expected_revision":2,"applied":false,"acknowledged":{"revision":2,"global":{"commit":"block","publish":"allow"}}}]}`)
	if err = r.locked(ctx, func() error { return r.writeFile("app-development.json", legacy) }); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(r.dir, "app-development.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	v, err := r.AppDevelopment(ctx)
	if err != nil || v.Global.Version != 2 || v.Global.Revision != 3 || v.Global.Gates.CreateDesk != apppolicy.Block || v.Global.Gates.Commit != apppolicy.Allow || v.Global.Runners[0].Acknowledged.Global.CreateDesk != apppolicy.Block || v.Global.Runners[0].Acknowledged.Global.Publish != apppolicy.Allow {
		t.Fatal(v, err)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("read rewrote legacy journal", err)
	}
	if _, err = r.SaveAppPolicyGlobal(ctx, 3, apppolicy.Gates{Commit: apppolicy.Allow, Publish: apppolicy.Block, CreateDesk: apppolicy.Allow}, nil); err != nil {
		t.Fatal(err)
	}
	v, err = r.AppDevelopment(ctx)
	if err != nil || v.Global.Version != 2 || v.Global.Revision != 4 || v.Global.Gates.CreateDesk != apppolicy.Allow {
		t.Fatal(v, err)
	}
	invalid := json.RawMessage(strings.Replace(string(legacy), `"version":1`, `"version":2`, 1))
	if err = r.locked(ctx, func() error { return r.writeFile("app-development.json", invalid) }); err != nil {
		t.Fatal(err)
	}
	if _, err = r.AppDevelopment(ctx); err == nil {
		t.Fatal("incomplete v2 journal was defaulted")
	}
}

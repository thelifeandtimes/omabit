package collection

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/appdev"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestAppPreflightSelectedRunnerReadOnlyAndTypedFailures(t *testing.T) {
	features := []string{protocol.AppPolicyFeature, protocol.DeskSettingsFeature, protocol.DeskStageFeature, protocol.DeskDiscoveryFeature, protocol.DeskCreateFeature, protocol.DeskCommitFeature, protocol.DeskApplicationFeature, protocol.DeskReleaseFeature}
	for _, tc := range []struct {
		name, status, omit, workflow, role, scope string
		target, wrongID, unresolved, knownBuild   bool
	}{
		{name: "before-project", status: "environment-compatible"},
		{name: "missing-policy-api", status: "missing-runner-api", omit: protocol.AppPolicyFeature},
		{name: "missing-stage-api", status: "missing-runner-api", omit: protocol.DeskStageFeature},
		{name: "missing-operation-adapter", status: "missing-operation-adapter", omit: protocol.DeskCommitFeature},
		{name: "workflow-adapter", status: "missing-workflow-adapter", workflow: "grubbery"},
		{name: "observer", status: "operator-role-required", role: "observer"},
		{name: "global-block", status: "policy-denied", target: true, scope: "global"},
		{name: "runner-block", status: "policy-denied", target: true, scope: "runner"},
		{name: "ship-block", status: "policy-denied", target: true, scope: "ship"},
		{name: "desk-block", status: "policy-denied", target: true, scope: "desk"},
		{name: "allowed-generation", status: "saved-policy-compatible", target: true, knownBuild: true},
		{name: "unresolved-generation", status: "target-unresolved", target: true, unresolved: true},
		{name: "changed-identity", wrongID: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			r := registry(t)
			var selected, other, writes, policyReads atomic.Int32
			h := protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-selected", Mode: "simulation", AccessRole: "operator", Features: slices.DeleteFunc(slices.Clone(features), func(f string) bool { return f == tc.omit })}
			if tc.role != "" {
				h.AccessRole = tc.role
			}
			if tc.knownBuild {
				modified := true
				h.Build = &protocol.ExecutableBuild{SourceRevision: "actual-runner-revision", Modified: &modified, SHA256: "actual-executable-hash"}
				h.StateSchema = 60
			}
			policy := apppolicy.Default()
			policy.Global = apppolicy.Gates{Commit: apppolicy.Allow, CreateDesk: apppolicy.Allow, Publish: apppolicy.Block}
			target := apppolicy.Target{InstanceID: "member-1", ProjectID: "project-1", Generation: 3, Creator: "actor-1"}
			g := apppolicy.Inherited()
			g.Commit = apppolicy.Block
			switch tc.scope {
			case "global":
				policy.Global.Commit = apppolicy.Block
			case "runner":
				policy.Runner = g
			case "ship":
				policy.Entries = append(policy.Entries, apppolicy.Entry{Target: target, Gates: g})
			case "desk":
				deskTarget := target
				deskTarget.Desk = "hello"
				policy.Entries = append(policy.Entries, apppolicy.Entry{Target: deskTarget, Gates: g})
			}
			socket := filepath.Join(t.TempDir(), "runner.sock")
			ln, err := net.Listen("unix", socket)
			if err != nil {
				t.Fatal(err)
			}
			var changed atomic.Bool
			srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
				selected.Add(1)
				w.Header().Set(protocol.Header, "1")
				if q.Method != "GET" {
					writes.Add(1)
					http.Error(w, "write prohibited", 405)
					return
				}
				if q.URL.Path == "/hello" {
					reply := h
					if changed.Load() {
						reply.RunnerID = "replacement"
					}
					_ = json.NewEncoder(w).Encode(reply)
					return
				}
				if q.URL.Path != protocol.AppPolicyPath {
					t.Errorf("unexpected endpoint %s", q.URL.Path)
					http.NotFound(w, q)
					return
				}
				policyReads.Add(1)
				_ = json.NewEncoder(w).Encode(protocol.AppPolicyStatus{Version: 1, RunnerID: h.RunnerID, Mode: h.Mode, ReadAt: time.Now().UTC(), Policy: policy, Targets: []protocol.AppPolicyTarget{{Target: target, Ready: !tc.unresolved}}})
			})}
			go srv.Serve(ln)
			t.Cleanup(func() { srv.Close() })
			if _, err = r.Enroll(ctx, "selected", socket); err != nil {
				t.Fatal(err)
			}
			otherSocket := fixture(t, "runner-other", func(w http.ResponseWriter, q *http.Request) { other.Add(1); http.Error(w, "unselected", 500) })
			if _, err = r.Enroll(ctx, "other", otherSocket); err != nil {
				t.Fatal(err)
			}
			// Make even the unselected Hello fail after enrollment. Any dial is a bug.
			if err = os.Remove(otherSocket); err != nil {
				t.Fatal(err)
			}
			selected.Store(0)
			changed.Store(tc.wrongID)
			before, err := os.ReadFile(filepath.Join(r.dir, "runners.json"))
			if err != nil {
				t.Fatal(err)
			}
			req := appdev.PreflightRequest{Name: "selected", RunnerID: h.RunnerID, Workflow: "gall", Effects: []string{"create-desk", "commit"}}
			if tc.workflow != "" {
				req.Workflow = tc.workflow
			}
			if tc.target {
				req.InstanceID = target.InstanceID
				req.Desk = "hello"
			}
			out, err := r.AppDevelopmentPreflight(ctx, req)
			if tc.wrongID {
				if err == nil {
					t.Fatal("identity replacement accepted")
				}
			} else {
				if err != nil || out.Status != tc.status {
					t.Fatalf("status=%s err=%v output=%+v", out.Status, err, out)
				}
				if tc.scope != "" && (out.Policy == nil || len(out.Policy.Blocked) != 1 || out.Policy.Blocked[0].Scope != tc.scope) {
					t.Fatalf("policy scope lost: %+v", out.Policy)
				}
				if tc.target && out.Policy != nil && out.Policy.Target.Generation != 3 {
					t.Fatal("generation lost")
				}
				if !tc.target && out.Policy != nil {
					t.Fatal("future target grant invented")
				}
				if (out.Hello.Build != nil) != tc.knownBuild {
					t.Fatal("runner build invented or lost")
				}
				if tc.knownBuild && out.Hello.Build.SourceRevision != "actual-runner-revision" {
					t.Fatal("client revision substituted")
				}
				if out.Assurance != "capability-and-saved-policy-only" {
					t.Fatal("assurance changed")
				}
				if out.Status == "saved-policy-compatible" && out.Policy.Effective.Publish.Value != apppolicy.Block {
					t.Fatal("unrequested publication grant changed")
				}
				if tc.omit == protocol.AppPolicyFeature && policyReads.Load() != 0 {
					t.Fatal("old runner contacted beyond Hello")
				}
			}
			after, err := os.ReadFile(filepath.Join(r.dir, "runners.json"))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) || writes.Load() != 0 || other.Load() != 0 {
				t.Fatal("preflight changed state or contacted unselected enrollment")
			}
		})
	}
}

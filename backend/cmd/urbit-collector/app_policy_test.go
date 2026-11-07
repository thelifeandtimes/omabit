package main

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestAppPolicyCLIReadOnlyAndFailClosed(t *testing.T) {
	var older atomic.Bool
	var invalid atomic.Int32
	socket := fakeRunner(t, func(w http.ResponseWriter, q *http.Request) {
		if q.Method != "GET" {
			t.Error("mutation", q.Method)
		}
		if q.URL.Path == "/hello" {
			h := protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", Features: []string{protocol.AppPolicyFeature}}
			if older.Load() {
				h.Features = nil
			}
			reply(t, w, h)
			return
		}
		if older.Load() {
			t.Error("queried unsupported runner")
		}
		value := protocol.AppPolicyStatus{Version: 1, RunnerID: "runner", Mode: "simulation", ReadAt: time.Now().UTC(), Policy: apppolicy.Default(), Targets: []protocol.AppPolicyTarget{{Target: apppolicy.Target{InstanceID: "alpha"}, Ready: true}}}
		switch invalid.Load() {
		case 1:
			value.Policy.Version = 2
		case 2:
			value.RunnerID = "other"
		case 3:
			value.Targets = append(value.Targets, protocol.AppPolicyTarget{Target: apppolicy.Target{InstanceID: "alpha", ProjectID: "other-project", Generation: 1, Creator: "launch"}, Ready: true})
		}
		reply(t, w, value)
	})
	var out, errs bytes.Buffer
	args := []string{"--socket", socket, "app-policy", "--instance", "alpha", "--desk", "app"}
	if run(context.Background(), args, &out, &errs) != 0 || !strings.Contains(out.String(), `"source": "global"`) {
		t.Fatal(out.String(), errs.String())
	}
	for _, tail := range [][]string{{"app-policy-set"}, {"app-policy", "--allow"}, {"desk-commit", "alpha"}, {"desk-publish", "alpha"}} {
		out.Reset()
		errs.Reset()
		if run(context.Background(), append([]string{"--socket", socket}, tail...), &out, &errs) == 0 {
			t.Fatal("agent edit/mutation exposed", tail)
		}
	}
	for _, issue := range []int32{1, 2, 3} {
		invalid.Store(issue)
		out.Reset()
		errs.Reset()
		if run(context.Background(), args, &out, &errs) == 0 {
			t.Fatal("unsupported or ambiguous policy accepted", issue, out.String())
		}
	}
	invalid.Store(0)
	older.Store(true)
	out.Reset()
	errs.Reset()
	if run(context.Background(), args, &out, &errs) == 0 || !strings.Contains(errs.String(), "app-development-policy-v1") {
		t.Fatal(out.String(), errs.String())
	}
}

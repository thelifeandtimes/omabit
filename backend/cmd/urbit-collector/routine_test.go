package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/routines"
)

func TestRoutineCLIRequiresAllConsentAndRevisionFields(t *testing.T) {
	writes := 0
	socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/hello" {
			reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-routine", Mode: "development-managed", AccessRole: "operator", Features: []string{"maintenance-schedules"}})
			return
		}
		if r.URL.Path != "/v1/instances/alpha/routine" {
			t.Error("wrong policy route")
		}
		if r.Method == "GET" {
			reply(t, w, protocol.RoutineResponse{RunnerID: "runner-routine", InstanceID: "alpha"})
			return
		}
		writes++
		var request routines.Update
		if json.NewDecoder(r.Body).Decode(&request) != nil || request.ExpectedRevision != 0 || request.ExpectedInstanceRevision != 2 || !request.Config.StopRunning || !request.Config.Resume {
			t.Error("policy consent/selection changed")
		}
		reply(t, w, routines.Policy{ID: "policy-one", InstanceID: "alpha", Revision: 1, InstanceRevision: 2, Config: request.Config})
	})
	dir := filepath.Join(t.TempDir(), "collector")
	invoke := func(want int, args ...string) {
		t.Helper()
		var out, errs bytes.Buffer
		if code := run(context.Background(), append([]string{"--config-dir", dir}, args...), &out, &errs); code != want {
			t.Fatalf("code=%d want=%d %s", code, want, &errs)
		}
	}
	invoke(0, "enroll", "host", "--socket", socket)
	base := []string{"--runner", "host", "--expected-runner-id", "runner-routine"}
	invoke(0, append(base, "routine", "alpha")...)
	args := append(append([]string{}, base...), "routine-set", "alpha", "--revision", "0", "--instance-revision", "2", "--enabled=true", "--action", "pack", "--days", "0,2", "--hour-utc", "2", "--minute-utc", "0", "--window-minutes", "15", "--stop-running=true")
	invoke(2, args...)
	if writes != 0 {
		t.Fatal("implicit resume policy accepted")
	}
	invoke(0, append(args, "--resume=true")...)
	if writes != 1 {
		t.Fatal("policy update count changed")
	}
}

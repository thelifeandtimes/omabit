package hostinstall

import (
	"context"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/managed"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestRC11OfflineVerifiedRetryStartsForwardDespitePostReleaseWork(t *testing.T) {
	for _, scenario := range []string{"held", "released-queued", "released-running", "released-uncertain", "complete-queued"} {
		t.Run(scenario, func(t *testing.T) {
			f, p, db, dir := offlineOwnerFixture(t)
			ctx := context.Background()
			rpc := f.i.updateRPC
			released := scenario != "held"
			if scenario != "complete-queued" {
				f.i.updateRPC = func(ctx context.Context, socket string, uid uint32, pid int, req *protocol.ManagementUpdateRequest) (protocol.ManagementUpdateStatus, error) {
					if req != nil && req.Action == "release" {
						if released {
							if _, err := rpc(ctx, socket, uid, pid, req); err != nil {
								t.Fatal(err)
							}
						}
						return protocol.ManagementUpdateStatus{}, errors.New("interrupted release acknowledgement")
					}
					return rpc(ctx, socket, uid, pid, req)
				}
			}
			_, err := f.i.installOwnerOffline(ctx, "/bundle", hostPublic(1), []byte("new"), p, nil)
			if scenario == "complete-queued" && err != nil || scenario != "complete-queued" && err == nil {
				t.Fatal(err)
			}
			if released {
				status := "queued"
				if scenario == "released-running" {
					status = "running"
				}
				if _, err := db.Exec("UPDATE operations SET body=json_remove(json_set(body,'$.status',?),'$.finished_at')", status); err != nil {
					t.Fatal(err)
				}
				if scenario == "released-uncertain" {
					if _, err := db.Exec(`UPDATE instances SET body=json_set(body,'$.runtime.maintenance_uncertain',json('true')) WHERE id='stopped'`); err != nil {
						t.Fatal(err)
					}
				}
			}
			var operation, instance string
			db.QueryRow("SELECT hex(body) FROM operations WHERE id='op-old'").Scan(&operation)
			db.QueryRow("SELECT hex(body) FROM instances WHERE id='stopped'").Scan(&instance)
			if _, err := f.i.command(ctx, "/usr/bin/systemctl", "--user", "--machine="+p.Name+"@.host", "stop", "urbit-runner.service"); err != nil {
				t.Fatal(err)
			}
			f.i.updateRPC = rpc
			inspections := 0
			stateCall := f.i.offlineStateCall
			f.i.offlineStateCall = func(ctx context.Context, lock *managed.OfflineStateLock, p ownerhost.Profile, req managed.OfflineUpdateRequest) (managed.OfflineUpdateState, error) {
				if req.Action != "inspect-continuation" || req.RunnerID != retryJournal(t, f).RunnerID || req.Token != retryJournal(t, f).Token || req.Schema != managed.StateSchemaVersion {
					t.Fatal("retry reapplied initial/migration prerequisites", req)
				}
				inspections++
				return stateCall(ctx, lock, p, req)
			}
			run := f.i.run
			f.i.run = func(ctx context.Context, path string, args, env []string) ([]byte, error) {
				if slices.Contains(args, "start") && slices.Contains(args, "urbit-runner.service") {
					lock, err := managed.LockOfflineState(dir, uint32(os.Geteuid()))
					if err != nil {
						t.Fatal("installer retained ownership while starting", err)
					}
					lock.Close()
				}
				return run(ctx, path, args, env)
			}
			f.calls = nil
			result, err := f.i.installOwnerOffline(ctx, "/bundle", hostPublic(1), []byte("new"), p, nil)
			if err != nil || !result.Ready || inspections != 1 {
				t.Fatal(result, err, inspections)
			}
			starts := 0
			for _, c := range f.calls {
				if slices.Contains(c.args, "stop") || slices.Contains(c.args, "restart") {
					t.Fatal("forward recovery canceled work", c)
				}
				if slices.Contains(c.args, "start") && slices.Contains(c.args, "urbit-runner.service") {
					starts++
				}
			}
			if starts != 1 {
				t.Fatal("new manager was not started", starts)
			}
			var afterOperation, afterInstance string
			db.QueryRow("SELECT hex(body) FROM operations WHERE id='op-old'").Scan(&afterOperation)
			db.QueryRow("SELECT hex(body) FROM instances WHERE id='stopped'").Scan(&afterInstance)
			if operation != afterOperation || instance != afterInstance {
				t.Fatal("installer changed jobs instead of leaving reconciliation to new manager")
			}
			if j := retryJournal(t, f); j.Stage != "complete" {
				t.Fatal(j.Stage)
			}
			f.calls = nil
			if r, err := f.i.installOwnerOffline(ctx, "/bundle", hostPublic(1), []byte("new"), p, nil); err != nil || !r.Ready {
				t.Fatal(r, err)
			}
			for _, c := range f.calls {
				if c.path == "/usr/bin/systemctl" {
					t.Fatal("live completion repeat changed service", c)
				}
			}
		})
	}
}

func TestRC11OfflineForwardRetryRejectsUnboundOrContradictoryState(t *testing.T) {
	for _, fault := range []string{"token", "runner", "schema", "build", "foreground", "held-with-job"} {
		t.Run(fault, func(t *testing.T) {
			f, p, db, _ := offlineOwnerFixture(t)
			ctx := context.Background()
			rpc := f.i.updateRPC
			f.i.updateRPC = func(ctx context.Context, socket string, uid uint32, pid int, req *protocol.ManagementUpdateRequest) (protocol.ManagementUpdateStatus, error) {
				if req != nil && req.Action == "release" {
					return protocol.ManagementUpdateStatus{}, errors.New("interrupted before release")
				}
				return rpc(ctx, socket, uid, pid, req)
			}
			if _, err := f.i.installOwnerOffline(ctx, "/bundle", hostPublic(1), []byte("new"), p, nil); err == nil {
				t.Fatal("missing injected interruption")
			}
			if _, err := f.i.command(ctx, "/usr/bin/systemctl", "--user", "--machine="+p.Name+"@.host", "stop", "urbit-runner.service"); err != nil {
				t.Fatal(err)
			}
			f.i.updateRPC = rpc
			switch fault {
			case "token":
				_, err := db.Exec("UPDATE meta SET value=json_set(value,'$.token',?) WHERE key='management_update_v1'", strings.Repeat("f", 64))
				if err != nil {
					t.Fatal(err)
				}
			case "runner":
				_, err := db.Exec("UPDATE meta SET value=? WHERE key='runner_id'", "runner-"+strings.Repeat("b", 32))
				if err != nil {
					t.Fatal(err)
				}
			case "schema":
				if _, err := db.Exec("PRAGMA user_version=19"); err != nil {
					t.Fatal(err)
				}
			case "build":
				f.put(access.ShellPath, "unexpected", 0755)
			case "foreground":
				f.i.activationStopped = func(context.Context, string, string) error { return errors.New("foreground listener") }
			case "held-with-job":
				if _, err := db.Exec(`UPDATE operations SET body=json_remove(json_set(body,'$.status','queued'),'$.finished_at')`); err != nil {
					t.Fatal(err)
				}
			}
			var before string
			db.QueryRow("SELECT value FROM meta WHERE key='management_update_v1'").Scan(&before)
			f.calls = nil
			if r, err := f.i.installOwnerOffline(ctx, "/bundle", hostPublic(1), []byte("new"), p, nil); err == nil || r.Ready {
				t.Fatal(r, err)
			}
			for _, c := range f.calls {
				if c.path == "/usr/bin/systemctl" {
					t.Fatal("invalid continuation started management", c)
				}
			}
			var after string
			db.QueryRow("SELECT value FROM meta WHERE key='management_update_v1'").Scan(&after)
			if before != after {
				t.Fatal("refusal altered retained fence")
			}
		})
	}
}

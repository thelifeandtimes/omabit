package hostinstall

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/managed"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func offlineOwnerFixture(t *testing.T) (*fixture, ownerhost.Profile, *sql.DB, string) {
	t.Helper()
	f, p, _ := ownerFixture(t)
	// The historical fixture has running ships on an independent engine; the
	// management-only bridge must not manufacture/start it by enabling a unit.
	f.engineStarted()
	profile, _ := json.Marshal(p)
	f.put(ownerhost.Path, string(profile)+"\n", 0644)
	f.put(managerUnit, string(ownerManagerUnit(p)), 0644)
	grant, err := access.OwnerAuthorizedKey(hostPublic(1), "operator")
	if err != nil {
		t.Fatal(err)
	}
	f.put(access.DefaultKeys, grant+"\n", 0644)
	f.put(hostKeyPath, "preserved-private-fixture", 0600)
	f.put(access.ShellPath, "original-schema19-fixture-executable", 0755)
	directory := f.i.path(filepath.Join(p.Home, ".local/share/urbit-runner/managed"))
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"runner.sqlite", ".lock"} {
		if err := os.WriteFile(filepath.Join(directory, name), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	db, err := sql.Open("sqlite", filepath.Join(directory, "runner.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("../managed/testdata/offline-schema19.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(data)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	f.i.offlineArtifactSchema = func(_ context.Context, binary []byte, owner ownerhost.Profile) (int, error) {
		if string(binary) != "original-schema19-fixture-executable" || owner != p {
			t.Fatal("unexpected provenance input")
		}
		return 19, nil
	}
	// Profile/process ownership evidence is synthetic (as in ownerFixture), but
	// the temp SQLite files and runtime flock are real and owned by this test UID.
	f.i.offlineLock = func(path string, uid uint32) (*managed.OfflineStateLock, error) {
		if path != directory || uid != p.UID {
			t.Fatal("wrong state lock", path, uid)
		}
		return managed.LockOfflineState(path, uint32(os.Geteuid()))
	}
	f.i.offlineStateCall = func(ctx context.Context, lock *managed.OfflineStateLock, _ ownerhost.Profile, req managed.OfflineUpdateRequest) (managed.OfflineUpdateState, error) {
		if req.Directory != filepath.Join(p.Home, ".local/share/urbit-runner/managed") {
			t.Fatal(req.Directory)
		}
		if req.Action == "inspect" {
			return lock.Inspect(ctx)
		}
		if req.Action == "inspect-continuation" {
			return lock.InspectContinuation(ctx, req.RunnerID, req.Token, req.Schema)
		}
		return lock.Fence(ctx, req.RunnerID, req.Token)
	}
	f.i.updateRPC = func(ctx context.Context, _ string, uid uint32, pid int, req *protocol.ManagementUpdateRequest) (protocol.ManagementUpdateStatus, error) {
		proof, err := f.i.activationEvidence(ctx, p.Name)
		if err != nil {
			return protocol.ManagementUpdateStatus{}, err
		}
		if uid != p.UID || pid != proof.PID {
			return protocol.ManagementUpdateStatus{}, errors.New("fixture peer mismatch")
		}
		var fence struct {
			Token string `json:"token"`
			Held  bool   `json:"held"`
		}
		var raw string
		if err := db.QueryRow("SELECT value FROM meta WHERE key='management_update_v1'").Scan(&raw); err != nil {
			return protocol.ManagementUpdateStatus{}, err
		}
		if err := json.Unmarshal([]byte(raw), &fence); err != nil {
			t.Fatal(err)
		}
		if req != nil {
			if req.Action != "release" || req.Token != fence.Token || req.RunnerID != "runner-"+strings.Repeat("a", 32) || req.ExpectedSHA256 != proof.SHA256 {
				return protocol.ManagementUpdateStatus{}, errors.New("unexpected offline release")
			}
			fence.Held = false
			data, _ := json.Marshal(fence)
			if _, err := db.Exec("UPDATE meta SET value=? WHERE key='management_update_v1'", string(data)); err != nil {
				t.Fatal(err)
			}
		}
		var schema int
		db.QueryRow("PRAGMA user_version").Scan(&schema)
		return protocol.ManagementUpdateStatus{RunnerID: "runner-" + strings.Repeat("a", 32), Mode: "development-managed", StateSchema: schema, MinStateSchema: 11, PID: pid, ExecutableSHA256: proof.SHA256, Quiesced: fence.Held, Token: fence.Token}, nil
	}
	return f, p, db, directory
}

func requireOfflineSchema(t *testing.T) {
	t.Helper()
	if managed.StateSchemaVersion < 20 {
		t.Fatal("offline upgrade requires the integrated schema20 fence")
	}
}

func TestRC11OfflineSchema19UpgradePreservesStateAndRepeatedOnlinePath(t *testing.T) {
	requireOfflineSchema(t)
	f, p, db, _ := offlineOwnerFixture(t)
	ctx := context.Background()
	before := map[string]string{}
	for _, table := range []string{"instances", "operations", "maintenance_policies"} {
		var body string
		if err := db.QueryRow("SELECT group_concat(hex(body),'|') FROM " + table).Scan(&body); err != nil {
			t.Fatal(err)
		}
		before[table] = body
	}
	result, err := f.i.installOwnerOffline(ctx, "/bundle", hostPublic(1), []byte("new-current-schema-fixture"), p, nil)
	if err != nil || !result.Ready || result.RunnerID != "runner-"+strings.Repeat("a", 32) {
		t.Fatal(result, err)
	}
	for table, want := range before {
		var got string
		if err := db.QueryRow("SELECT group_concat(hex(body),'|') FROM " + table).Scan(&got); err != nil || got != want {
			t.Fatal("changed historical bodies", table, err)
		}
	}
	var schema int
	db.QueryRow("PRAGMA user_version").Scan(&schema)
	if schema != managed.StateSchemaVersion {
		t.Fatal(schema)
	}
	old := executableGeneration{SHA256: digestBytes([]byte("original-schema19-fixture-executable")), MinSchema: 19, Schema: 19}
	if _, err := f.i.retainedBytes(ctx, old); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if r, err := f.i.installOwnerOffline(ctx, "/bundle", hostPublic(1), []byte("new-current-schema-fixture"), p, nil); err != nil || !r.Ready {
			t.Fatal(r, err)
		}
	}
	// Normal management-only updates retain their old default semantics. An
	// exact-version normal rerun verifies the new online manager without stopping.
	probe := func(context.Context, string) (protocol.Hello, bool, error) {
		return protocol.Hello{RunnerID: result.RunnerID, Mode: result.Mode}, false, nil
	}
	if r, err := f.i.installOwner(ctx, "/bundle", hostPublic(1), []byte("new-current-schema-fixture"), p, probe, nil); err != nil || !r.Ready {
		t.Fatal(r, err)
	}
	for _, c := range f.calls {
		if slices.Contains(c.args, "stop") {
			t.Fatal("offline bridge stopped a manager or ship", c)
		}
	}
}

func TestRC11OfflineRefusesProcessLockForegroundJobsAndProvenance(t *testing.T) {
	requireOfflineSchema(t)
	for _, fault := range []string{"lock", "foreground", "queued", "running", "uncertain", "identity", "future-schema", "unknown-binary"} {
		t.Run(fault, func(t *testing.T) {
			f, p, db, dir := offlineOwnerFixture(t)
			switch fault {
			case "lock":
				lock, err := managed.LockOfflineState(dir, uint32(os.Geteuid()))
				if err != nil {
					t.Fatal(err)
				}
				defer lock.Close()
			case "foreground":
				f.i.activationStopped = func(context.Context, string, string) error { return errors.New("foreground listener survived") }
			case "queued", "running":
				if _, err := db.Exec("UPDATE operations SET body=json_set(body,'$.status',?)", fault); err != nil {
					t.Fatal(err)
				}
			case "uncertain":
				if _, err := db.Exec(`UPDATE instances SET body=json_set(body,'$.runtime.maintenance_uncertain',json('true')) WHERE id='running'`); err != nil {
					t.Fatal(err)
				}
			case "identity":
				db.Exec("UPDATE meta SET value='invalid' WHERE key='runner_id'")
			case "future-schema":
				db.Exec("PRAGMA user_version=99")
			case "unknown-binary":
				f.i.offlineArtifactSchema = func(context.Context, []byte, ownerhost.Profile) (int, error) {
					return 0, errors.New("unrecognized artifact provenance")
				}
			}
			if r, err := f.i.installOwnerOffline(context.Background(), "/bundle", hostPublic(1), []byte("new"), p, nil); err == nil || r.Ready {
				t.Fatal(r, err)
			}
			var version int
			db.QueryRow("PRAGMA user_version").Scan(&version)
			want := 19
			if fault == "future-schema" {
				want = 99
			}
			if version != want {
				t.Fatal("refusal changed schema", version)
			}
			current, err := os.ReadFile(f.i.path(access.ShellPath))
			if err != nil || string(current) != "original-schema19-fixture-executable" {
				t.Fatal("refusal replaced original binary", err)
			}
			for _, c := range f.calls {
				if c.path == "/usr/bin/systemctl" {
					t.Fatal("refusal dispatched a service action", c)
				}
			}
		})
	}
}

func TestRC11OfflineResumeAcrossMigrationPublicationAndActivation(t *testing.T) {
	requireOfflineSchema(t)
	for _, boundary := range []string{"offline-prepared", "migration-ack", "offline-fenced", "executable", "starting", "started", "verified", "complete"} {
		t.Run(boundary, func(t *testing.T) {
			f, p, db, _ := offlineOwnerFixture(t)
			ctx := context.Background()
			interrupted := false
			chown, call, run := f.i.chown, f.i.offlineStateCall, f.i.run
			f.i.chown = func(file *os.File) error {
				info, err := file.Stat()
				if err != nil {
					t.Fatal(err)
				}
				if info.IsDir() {
					return chown(file)
				}
				data, err := os.ReadFile(file.Name())
				if err != nil {
					t.Fatal(err)
				}
				var j activationJournal
				journal := json.Unmarshal(data, &j) == nil && j.Offline
				executable := string(data) == "new" && filepath.Dir(file.Name()) == filepath.Dir(f.i.path(access.ShellPath))
				if !interrupted && (journal && j.Stage == boundary || executable && boundary == "executable") {
					interrupted = true
					return errors.New("publication boundary interrupted")
				}
				return chown(file)
			}
			f.i.offlineStateCall = func(ctx context.Context, lock *managed.OfflineStateLock, p ownerhost.Profile, req managed.OfflineUpdateRequest) (managed.OfflineUpdateState, error) {
				state, err := call(ctx, lock, p, req)
				if err == nil && !interrupted && boundary == "migration-ack" && req.Action == "fence" {
					interrupted = true
					return state, errors.New("lost committed migration acknowledgement")
				}
				return state, err
			}
			f.i.run = func(ctx context.Context, path string, args, env []string) ([]byte, error) {
				out, err := run(ctx, path, args, env)
				if err == nil && !interrupted && boundary == "started" && slices.Contains(args, "enable") && slices.Contains(args, "urbit-runner.service") {
					interrupted = true
					return out, errors.New("lost service start acknowledgement")
				}
				return out, err
			}
			if r, err := f.i.installOwnerOffline(ctx, "/bundle", hostPublic(1), []byte("new"), p, nil); err == nil || r.Ready || !interrupted {
				t.Fatal(r, err, interrupted)
			}
			var schema int
			db.QueryRow("PRAGMA user_version").Scan(&schema)
			if boundary != "offline-prepared" && schema != managed.StateSchemaVersion {
				t.Fatal("fence migration not committed", schema)
			}
			// A separate installer object reacquires all locks; no in-memory token,
			// lock ownership or PID from the interrupted invocation is reused.
			f.i.chown, f.i.offlineStateCall, f.i.run = chown, call, run
			next := f.i
			next.mutated = false
			if boundary == "started" {
				proof := f.i.activationEvidence
				f.i.activationEvidence = func(ctx context.Context, owner string) (serviceEvidence, error) {
					v, e := proof(ctx, owner)
					v.PID += 100
					v.Invocation = strings.Repeat("b", 32)
					return v, e
				}
				next.activationEvidence = f.i.activationEvidence
			}
			if r, err := next.installOwnerOffline(ctx, "/bundle", hostPublic(1), []byte("new"), p, nil); err != nil || !r.Ready {
				t.Fatal(r, err)
			}
			for _, c := range f.calls {
				if slices.Contains(c.args, "stop") {
					t.Fatal("recovery stopped management", c)
				}
			}
			if r, err := next.installOwnerOffline(ctx, "/bundle", hostPublic(1), []byte("other-generation"), p, nil); err == nil || r.Ready {
				t.Fatal("offline path became an implicit new update", r, err)
			}
		})
	}
}

func TestRC11OfflineRejectsNonELFOrUnrecognizedArtifactsWithoutExecution(t *testing.T) {
	for _, binary := range [][]byte{[]byte("#!/bin/sh\nexit 0\n"), []byte("fixture"), {0x7f, 'E', 'L', 'F'}} {
		if err := validateHistoricalExecutable(binary); err == nil {
			t.Fatal("invalid executable provenance accepted")
		}
	}
	p := ownerhost.Profile{UID: 942, GID: 943}
	cmd := historicalSchemaCommand(context.Background(), "/fixture/read-only-chroot", p)
	if cmd.Path != "/runner" || !slices.Equal(cmd.Args, []string{"/runner", "state-schema"}) || cmd.SysProcAttr.Chroot != "/fixture/read-only-chroot" || cmd.SysProcAttr.Credential.Uid != p.UID || cmd.SysProcAttr.Credential.Gid != p.GID || len(cmd.SysProcAttr.Credential.Groups) != 0 || cmd.SysProcAttr.Cloneflags&syscall.CLONE_NEWNET == 0 || len(cmd.ExtraFiles) != 0 {
		t.Fatal("schema probe lost unprivileged/read-only-root/network isolation", cmd)
	}
}

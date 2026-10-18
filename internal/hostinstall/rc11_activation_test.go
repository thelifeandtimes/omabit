package hostinstall

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostsetup"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/managed"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"golang.org/x/sys/unix"
)

// No live service/commands/proc/socket evidence. Keep the synthetic listener's
// process executable separate from the file so replacement cannot fake readiness.
func configureActivationFixture(f *fixture) {
	active, pid, build := false, 100, ""
	held, token := false, ""
	f.i.activationTimeout = time.Millisecond
	run := f.i.run
	f.i.run = func(ctx context.Context, path string, args, env []string) ([]byte, error) {
		out, err := run(ctx, path, args, env)
		if err != nil {
			return out, err
		}
		if path == "/usr/bin/systemctl" && slices.Contains(args, "urbit-runner.service") {
			if slices.Contains(args, "stop") {
				active = false
			}
			if slices.Contains(args, "enable") || slices.Contains(args, "start") {
				data, err := os.ReadFile(f.i.path(access.ShellPath))
				if err != nil {
					return nil, err
				}
				active, pid, build = true, pid+1, digestBytes(data)
			}
		}
		return out, nil
	}
	f.i.activationEvidence = func(context.Context, string) (serviceEvidence, error) {
		if !active {
			return serviceEvidence{}, errors.New("inactive fixture service")
		}
		return serviceEvidence{PID: pid, Invocation: strings.Repeat("a", 32), SHA256: build}, nil
	}
	f.i.activationStopped = func(context.Context, string, string) error {
		if active {
			return errors.New("fixture listener survived")
		}
		return nil
	}
	f.i.updateRPC = func(_ context.Context, _ string, uid uint32, wantPID int, req *protocol.ManagementUpdateRequest) (protocol.ManagementUpdateStatus, error) {
		if !active || uid != testUID || pid != wantPID {
			return protocol.ManagementUpdateStatus{}, errors.New("peer mismatch")
		}
		if req != nil {
			if req.RunnerID != "runner-"+strings.Repeat("a", 32) {
				return protocol.ManagementUpdateStatus{}, errors.New("identity mismatch")
			}
			switch req.Action {
			case "acquire":
				if token == req.Token && !held {
					return protocol.ManagementUpdateStatus{}, errors.New("update_token_released")
				}
				if held && token != req.Token {
					return protocol.ManagementUpdateStatus{}, errors.New("token conflict")
				}
				held, token = true, req.Token
			case "release":
				if token != req.Token || req.ExpectedSHA256 != build {
					return protocol.ManagementUpdateStatus{}, errors.New("release mismatch")
				}
				held = false
			}
		}
		return protocol.ManagementUpdateStatus{RunnerID: "runner-" + strings.Repeat("a", 32), Mode: "development-managed", PID: pid, ExecutableSHA256: build, StateSchema: managed.StateSchemaVersion, MinStateSchema: 11, Quiesced: held, Token: token}, nil
	}
}

func TestRC11ActivationBusyAtFinalAcquirePreservesManager(t *testing.T) {
	f, p, probe := ownerFixture(t)
	ctx := context.Background()
	if _, err := f.i.installOwner(ctx, "/bundle", hostPublic(1), []byte("old"), p, probe, nil); err != nil {
		t.Fatal(err)
	}
	rpc := f.i.updateRPC
	busy := false
	f.i.updateRPC = func(ctx context.Context, socket string, uid uint32, pid int, req *protocol.ManagementUpdateRequest) (protocol.ManagementUpdateStatus, error) {
		if req != nil && req.Action == "acquire" && busy {
			return protocol.ManagementUpdateStatus{}, errors.New("update_busy")
		}
		return rpc(ctx, socket, uid, pid, req)
	}
	f.calls = nil
	_, err := f.i.installOwner(ctx, "/bundle", hostPublic(1), []byte("new"), p, probe, func(step string) {
		if step == "access" {
			busy = true
		}
	})
	if err == nil || !strings.Contains(err.Error(), "update_busy") {
		t.Fatal(err)
	}
	for _, c := range f.calls {
		if slices.Contains(c.args, "stop") {
			t.Fatal("stopped accepted work", c)
		}
	}
	data, _ := os.ReadFile(f.i.path(access.ShellPath))
	if string(data) != "old" {
		t.Fatal(string(data))
	}
	busy = false
	if r, err := f.i.installOwner(ctx, "/bundle", hostPublic(1), []byte("new"), p, probe, nil); err != nil || !r.Ready {
		t.Fatal(r, err)
	}
}

func TestRC11ActivationRejectsForegroundAndHistoricalRunner(t *testing.T) {
	for _, kind := range []string{"foreground", "historical", "wrong-build", "wrong-peer", "exits-after-exec", "surviving-listener"} {
		t.Run(kind, func(t *testing.T) {
			f, p, probe := ownerFixture(t)
			ctx := context.Background()
			if _, err := f.i.installOwner(ctx, "/bundle", hostPublic(1), []byte("old"), p, probe, nil); err != nil {
				t.Fatal(err)
			}
			proof, rpc := f.i.activationEvidence, f.i.updateRPC
			switch kind {
			case "foreground":
				f.i.activationEvidence = func(context.Context, string) (serviceEvidence, error) {
					return serviceEvidence{}, errors.New("inactive service; foreground socket")
				}
			case "historical":
				f.i.updateRPC = func(context.Context, string, uint32, int, *protocol.ManagementUpdateRequest) (protocol.ManagementUpdateStatus, error) {
					return protocol.ManagementUpdateStatus{}, errors.New("404 historical rc11")
				}
			case "wrong-build":
				f.i.activationEvidence = func(ctx context.Context, owner string) (serviceEvidence, error) {
					v, err := proof(ctx, owner)
					v.SHA256 = strings.Repeat("b", 64)
					return v, err
				}
			case "wrong-peer":
				f.i.updateRPC = func(ctx context.Context, socket string, uid uint32, pid int, req *protocol.ManagementUpdateRequest) (protocol.ManagementUpdateStatus, error) {
					v, err := rpc(ctx, socket, uid, pid, req)
					v.PID++
					return v, err
				}
			case "exits-after-exec":
				f.i.activationEvidence = func(ctx context.Context, owner string) (serviceEvidence, error) {
					v, err := proof(ctx, owner)
					if v.SHA256 == digestBytes([]byte("new")) {
						return serviceEvidence{}, errors.New("exited")
					}
					return v, err
				}
			case "surviving-listener":
				f.i.activationStopped = func(context.Context, string, string) error { return errors.New("surviving foreground listener") }
			}
			f.calls = nil
			if r, err := f.i.installOwner(ctx, "/bundle", hostPublic(1), []byte("new"), p, probe, nil); err == nil || r.Ready {
				t.Fatal(r, err)
			}
			if kind != "exits-after-exec" && kind != "surviving-listener" {
				for _, c := range f.calls {
					if slices.Contains(c.args, "stop") {
						t.Fatal("preflight deactivated manager", c)
					}
				}
			}
		})
	}
}

func TestRC11CompatibleRollbackAndReleaseReplay(t *testing.T) {
	for _, failure := range []string{"start", "release-ack", "rollback-release-ack"} {
		t.Run(failure, func(t *testing.T) {
			f, p, probe := ownerFixture(t)
			ctx := context.Background()
			if _, err := f.i.installOwner(ctx, "/bundle", hostPublic(1), []byte("old"), p, probe, nil); err != nil {
				t.Fatal(err)
			}
			failed := false
			if failure != "release-ack" {
				f.fail = func(c call) error {
					if !failed && slices.Contains(c.args, "enable") && slices.Contains(c.args, "urbit-runner.service") {
						failed = true
						return errors.New("start failure")
					}
					return nil
				}
			}
			if failure != "start" {
				rpc := f.i.updateRPC
				lost := false
				f.i.updateRPC = func(ctx context.Context, socket string, uid uint32, pid int, req *protocol.ManagementUpdateRequest) (protocol.ManagementUpdateStatus, error) {
					v, err := rpc(ctx, socket, uid, pid, req)
					if !lost && req != nil && req.Action == "release" {
						lost = true
						return v, errors.New("lost release acknowledgement")
					}
					return v, err
				}
			}
			if r, err := f.i.installOwner(ctx, "/bundle", hostPublic(1), []byte("new"), p, probe, nil); err == nil || r.Ready {
				t.Fatal(r, err)
			}
			data, _ := os.ReadFile(f.i.path(access.ShellPath))
			want := "old"
			if failure == "release-ack" {
				want = "new"
			}
			if string(data) != want {
				t.Fatal(string(data))
			}
			for _, content := range []string{"old", "new"} {
				g := executableGeneration{SHA256: digestBytes([]byte(content)), MinSchema: 11, Schema: managed.StateSchemaVersion}
				if _, err := f.i.retainedBytes(ctx, g); err != nil {
					t.Fatal(err)
				}
			}
			f.calls = nil
			if _, err := f.i.installOwner(ctx, "/bundle", hostPublic(1), []byte("new"), p, probe, nil); err != nil {
				t.Fatal(err)
			}
			if failure != "start" {
				for _, c := range f.calls {
					if slices.Contains(c.args, "stop") {
						t.Fatal("release replay stopped reopened manager", c)
					}
				}
			}
		})
	}
}

func TestRC11RollbackRefusesPossiblyMigratedSchema(t *testing.T) {
	f, _, _ := ownerFixture(t)
	j := &activationJournal{Previous: &executableGeneration{SHA256: strings.Repeat("a", 64), MinSchema: 11, Schema: 19}, Target: executableGeneration{SHA256: strings.Repeat("b", 64), MinSchema: 11, Schema: 20}}
	_, err := f.i.rollbackManagement(context.Background(), "fixture", j, func(...string) error { t.Fatal("rollback dispatched"); return nil })
	if err == nil || !strings.Contains(err.Error(), "schema") {
		t.Fatal(err)
	}
}

func TestRC11ActivationJournalBoundaryResume(t *testing.T) {
	// Faults are ordinary injected persistence/command interruptions on temporary
	// files. They establish recovery ordering, not real power-loss qualification.
	for _, boundary := range []string{"prepared", "quiesced", "stopping", "stopped", "executable", "starting", "started", "verified", "complete"} {
		t.Run(boundary, func(t *testing.T) {
			f, p, probe := ownerFixture(t)
			ctx := context.Background()
			if _, err := f.i.installOwner(ctx, "/bundle", hostPublic(1), []byte("old"), p, probe, nil); err != nil {
				t.Fatal(err)
			}
			injected := errors.New("activation boundary interruption")
			interrupted := false
			chown := f.i.chown
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
				isJournal := json.Unmarshal(data, &j) == nil && j.Target.SHA256 == digestBytes([]byte("new"))
				isExecutable := string(data) == "new" && filepath.Dir(file.Name()) == filepath.Dir(f.i.path(access.ShellPath))
				if !interrupted && (isJournal && j.Stage == boundary || isExecutable && boundary == "executable") {
					interrupted = true
					return injected
				}
				return chown(file)
			}
			run := f.i.run
			callCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			f.i.run = func(ctx context.Context, path string, args, env []string) ([]byte, error) {
				data, err := run(ctx, path, args, env)
				if err == nil && !interrupted && slices.Contains(args, "urbit-runner.service") && (boundary == "stopped" && slices.Contains(args, "stop") || boundary == "started" && slices.Contains(args, "enable")) {
					interrupted = true
					cancel()
					return data, injected
				}
				return data, err
			}
			if result, err := f.i.installOwner(callCtx, "/bundle", hostPublic(1), []byte("new"), p, probe, nil); err == nil || result.Ready || !interrupted {
				t.Fatal(result, err, interrupted)
			}
			f.i.chown, f.i.run = chown, run
			f.calls = nil
			if result, err := f.i.installOwner(ctx, "/bundle", hostPublic(1), []byte("new"), p, probe, nil); err != nil || !result.Ready {
				t.Fatal(result, err)
			}
			data, err := os.ReadFile(f.i.path(activationJournalPath))
			var j activationJournal
			if err != nil || json.Unmarshal(data, &j) != nil || j.Stage != "complete" || j.Target.SHA256 != digestBytes([]byte("new")) {
				t.Fatal(j, err)
			}
			if boundary == "complete" || boundary == "started" || boundary == "verified" {
				for _, c := range f.calls {
					if slices.Contains(c.args, "stop") {
						t.Fatal("resume stopped already activated manager", c)
					}
				}
			}
		})
	}
}

func TestRC11PublicationSyncFailureLeavesCompleteRetryableFile(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	path := "/etc/urbit-runner/owner.json"
	if err := f.i.parents(ctx, path, true); err != nil {
		t.Fatal(err)
	}
	f.i.syncDirectory = func(string) error { return errors.New("directory barrier failed") }
	data := []byte(`{"complete":true}`)
	if err := f.i.installFile(ctx, path, data, 0644, false); err == nil {
		t.Fatal("ignored publication barrier")
	}
	actual, err := os.ReadFile(f.i.path(path))
	if err != nil || string(actual) != string(data) {
		t.Fatal(string(actual), err)
	}
	barrier := false
	f.i.syncDirectory = func(string) error { barrier = true; return nil }
	if err := f.i.installFile(ctx, path, data, 0644, false); err != nil || !barrier {
		t.Fatal("retry failed to reestablish publication barrier", err)
	}
}

func TestRC11ProfileStagingFailureAllowsOrdinarySetupRetry(t *testing.T) {
	f, p, probe := ownerFixture(t)
	chown := f.i.chown
	failed := false
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
		var profile struct {
			Name string `json:"name"`
		}
		if !failed && json.Unmarshal(data, &profile) == nil && profile.Name == p.Name {
			failed = true
			file.Truncate(12)
			return errors.New("partial staging interrupted")
		}
		return chown(file)
	}
	if _, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("runner"), p, probe, nil); err == nil || !failed {
		t.Fatal(err, failed)
	}
	if _, err := os.Lstat(f.i.path("/etc/urbit-runner/owner.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("partial profile became visible", err)
	}
	f.i.chown = chown
	if result, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("runner"), p, probe, nil); err != nil || !result.Ready {
		t.Fatal(result, err)
	}
}

func TestRC11HostKeyCompanionPublicationRetryPreservesSecret(t *testing.T) {
	f, p, probe := ownerFixture(t)
	f.i.publishFile = func(from, to string) error {
		if to == f.i.path(hostKeyPath+".pub") {
			return errors.New("public companion publication interrupted")
		}
		return unix.Renameat2(unix.AT_FDCWD, from, unix.AT_FDCWD, to, unix.RENAME_NOREPLACE)
	}
	if _, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("runner"), p, probe, nil); err == nil {
		t.Fatal("ignored publication error")
	}
	private, err := os.ReadFile(f.i.path(hostKeyPath))
	if err != nil {
		t.Fatal(err)
	}
	f.i.publishFile = nil
	if result, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("runner"), p, probe, nil); err != nil || !result.Ready {
		t.Fatal(result, err)
	}
	again, err := os.ReadFile(f.i.path(hostKeyPath))
	if err != nil || string(private) != string(again) {
		t.Fatal("rotated host identity", err)
	}
	if _, err := os.Stat(f.i.path(hostKeyPath + ".pub")); err != nil {
		t.Fatal(err)
	}
	generated := 0
	for _, c := range f.calls {
		if c.path == "/usr/bin/ssh-keygen" && !slices.Contains(c.args, "-y") {
			generated++
		}
	}
	if generated != 1 {
		t.Fatal("regenerated key", generated)
	}
}

func TestRC11IncompatibleSchemaAndTamperedGenerationPreserveHealthyManager(t *testing.T) {
	for _, scenario := range []string{"future-schema", "tampered-generation"} {
		t.Run(scenario, func(t *testing.T) {
			f, p, probe := ownerFixture(t)
			ctx := context.Background()
			if _, err := f.i.installOwner(ctx, "/bundle", hostPublic(1), []byte("old"), p, probe, nil); err != nil {
				t.Fatal(err)
			}
			if scenario == "future-schema" {
				rpc := f.i.updateRPC
				f.i.updateRPC = func(ctx context.Context, socket string, uid uint32, pid int, req *protocol.ManagementUpdateRequest) (protocol.ManagementUpdateStatus, error) {
					v, err := rpc(ctx, socket, uid, pid, req)
					v.StateSchema = managed.StateSchemaVersion + 1
					return v, err
				}
			} else {
				j, err := f.i.prepareActivation(ctx, p.Name, "/run/user/942/urbit-runner/runner.sock", p.UID, []byte("new"), probe)
				if err != nil {
					t.Fatal(err)
				}
				f.put(generationPath(j.Target), "changed", 0755)
			}
			f.calls = nil
			if result, err := f.i.installOwner(ctx, "/bundle", hostPublic(1), []byte("new"), p, probe, nil); err == nil || result.Ready {
				t.Fatal(result, err)
			}
			for _, c := range f.calls {
				if slices.Contains(c.args, "stop") {
					t.Fatal("deactivated healthy manager", c)
				}
			}
			data, err := os.ReadFile(f.i.path(access.ShellPath))
			if err != nil || string(data) != "old" {
				t.Fatal(string(data), err)
			}
		})
	}
}

func TestRC11FirstPublicationAbsentOrCompleteNoReplace(t *testing.T) {
	for _, fault := range []string{"ownership", "publish", "collision", "success"} {
		t.Run(fault, func(t *testing.T) {
			f := newFixture(t)
			ctx := context.Background()
			path := "/etc/urbit-runner/owner.json"
			if err := f.i.parents(ctx, path, true); err != nil {
				t.Fatal(err)
			}
			data := []byte(`{"version":1,"complete":true}`)
			if fault == "ownership" {
				f.i.chown = func(file *os.File) error { file.Truncate(2); return errors.New("interrupted staging") }
			}
			f.i.publishFile = func(from, to string) error {
				staged, err := os.ReadFile(from)
				if err != nil || string(staged) != string(data) {
					t.Fatal(string(staged), err)
				}
				if fault == "publish" {
					return errors.New("before publication")
				}
				if fault == "collision" {
					if err := os.WriteFile(to, []byte("foreign"), 0644); err != nil {
						t.Fatal(err)
					}
				}
				return unix.Renameat2(unix.AT_FDCWD, from, unix.AT_FDCWD, to, unix.RENAME_NOREPLACE)
			}
			err := f.i.write(ctx, path, data, 0644)
			actual, readErr := os.ReadFile(f.i.path(path))
			switch fault {
			case "ownership", "publish":
				if err == nil || !errors.Is(readErr, os.ErrNotExist) {
					t.Fatal(err, readErr, string(actual))
				}
			case "collision":
				if err == nil || string(actual) != "foreign" {
					t.Fatal(err, string(actual))
				}
			case "success":
				if err != nil || string(actual) != string(data) {
					t.Fatal(err, string(actual))
				}
			}
		})
	}
}

func TestRC11ExternalNSSRequiresReservedConfiguredRanges(t *testing.T) {
	for _, provider := range []string{"sss", "ldap", "compat", "systemd", "files [SUCCESS=return]"} {
		t.Run(provider, func(t *testing.T) {
			f, p, _ := ownerFixture(t)
			f.put("/etc/nsswitch.conf", "passwd: files "+provider+"\ngroup: files "+provider+"\n", 0644)
			f.put("/etc/subuid", "", 0644)
			f.put("/etc/subgid", "", 0644)
			if err := f.i.ownerMappings(context.Background(), p); err == nil {
				t.Fatal("trusted incomplete local inventory")
			}
			for _, c := range f.calls {
				if c.path == "/usr/bin/usermod" {
					t.Fatal("allocated across identity domain", c)
				}
			}
			reservation := hostsetup.SubIDReservations{Version: 1, Authority: "fixture centrally reserved outside NSS domains", UID: hostsetup.ReservedSubIDRange{Start: 2000000, Count: 65536}, GID: hostsetup.ReservedSubIDRange{Start: 2000000, Count: 65536}}
			data, _ := json.Marshal(reservation)
			f.put(hostsetup.SubIDReservationsPath, string(data), 0644)
			if err := f.i.ownerMappings(context.Background(), p); err == nil {
				t.Fatal("automatically allocated on external NSS")
			}
			for _, name := range []string{"subuid", "subgid"} {
				f.put(filepath.Join("/etc", name), p.Name+":2000000:65536\n", 0644)
			}
			if err := f.i.ownerMappings(context.Background(), p); err != nil {
				t.Fatal(err)
			}
			f.put("/etc/subgid", p.Name+":1048576:65536\n", 0644)
			if err := f.i.ownerMappings(context.Background(), p); err == nil {
				t.Fatal("accepted range outside reservation")
			}
		})
	}
}

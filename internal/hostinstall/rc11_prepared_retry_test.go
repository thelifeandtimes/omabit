package hostinstall

import (
	"context"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func preparedRetryFixture(t *testing.T, outcome string) (*fixture, ownerhost.Profile, hostProbe, string) {
	t.Helper()
	f, p, probe := ownerFixture(t)
	ctx := context.Background()
	if _, err := f.i.installOwner(ctx, "/bundle", hostPublic(1), []byte("old"), p, probe, nil); err != nil {
		t.Fatal(err)
	}
	rpc := f.i.updateRPC
	token := ""
	f.i.updateRPC = func(ctx context.Context, socket string, uid uint32, pid int, req *protocol.ManagementUpdateRequest) (protocol.ManagementUpdateStatus, error) {
		if req != nil && req.Action == "acquire" {
			token = req.Token
			switch outcome {
			case "held", "released":
				status, err := rpc(ctx, socket, uid, pid, req)
				if err != nil || !status.Quiesced || status.Token != token {
					t.Fatal(status, err)
				}
				if outcome == "released" {
					if _, err := rpc(ctx, socket, uid, pid, &protocol.ManagementUpdateRequest{Action: "release", RunnerID: req.RunnerID, Token: token, ExpectedSHA256: status.ExecutableSHA256}); err != nil {
						t.Fatal(err)
					}
				}
			case "foreign":
				other := *req
				other.Token = strings.Repeat("f", 64)
				if _, err := rpc(ctx, socket, uid, pid, &other); err != nil {
					t.Fatal(err)
				}
			case "not-acquired": // fail before the original RPC can commit
			default:
				t.Fatal(outcome)
			}
			return protocol.ManagementUpdateStatus{}, errors.New("acquire acknowledgement unavailable")
		}
		return rpc(ctx, socket, uid, pid, req)
	}
	if result, err := f.i.installOwner(ctx, "/bundle", hostPublic(1), []byte("new"), p, probe, nil); err == nil || result.Ready || token == "" {
		t.Fatal(result, err)
	}
	j := retryJournal(t, f)
	if j.Stage != "prepared" || j.Token != token || j.Previous == nil || j.Previous.SHA256 != digestBytes([]byte("old")) {
		t.Fatal(j)
	}
	f.i.updateRPC = rpc
	return f, p, probe, token
}

func TestRC11PreparedRetryReacquiresSameTokenBeforeTargetActivation(t *testing.T) {
	for _, outcome := range []string{"held", "not-acquired", "released", "foreign"} {
		for _, exited := range []bool{false, true} {
			t.Run(outcome+map[bool]string{false: "/live", true: "/exited"}[exited], func(t *testing.T) {
				f, p, probe, token := preparedRetryFixture(t, outcome)
				ctx := context.Background()
				before, err := f.i.activationEvidence(ctx, p.Name)
				if err != nil {
					t.Fatal(err)
				}
				if exited {
					if _, err := f.i.command(ctx, "/usr/bin/systemctl", "--user", "--machine="+p.Name+"@.host", "stop", "urbit-runner.service"); err != nil {
						t.Fatal(err)
					}
				}
				rpc := f.i.updateRPC
				acquired := false
				acquires := 0
				f.i.updateRPC = func(ctx context.Context, socket string, uid uint32, pid int, req *protocol.ManagementUpdateRequest) (protocol.ManagementUpdateStatus, error) {
					status, err := rpc(ctx, socket, uid, pid, req)
					if req != nil && req.Action == "acquire" {
						acquires++
						if req.Token != token {
							t.Fatal("replaced retained acquire token")
						}
						if exited && pid == before.PID || !exited && pid != before.PID {
							t.Fatal("wrong previous service invocation", pid, before.PID)
						}
						if err == nil {
							if !status.Quiesced || status.Token != token || status.ExecutableSHA256 != digestBytes([]byte("old")) {
								t.Fatal(status)
							}
							acquired = true
						}
					}
					return status, err
				}
				f.beforeCommand = func(c call) {
					if !slices.Contains(c.args, "urbit-runner.service") {
						return
					}
					data, err := os.ReadFile(f.i.path(access.ShellPath))
					if err != nil {
						t.Fatal(err)
					}
					if slices.Contains(c.args, "start") && (string(data) != "old" || retryJournal(t, f).Stage != "prepared" || acquired) {
						t.Fatal("started something other than unchanged previous generation", c, string(data))
					}
					if (slices.Contains(c.args, "stop") || slices.Contains(c.args, "enable")) && !acquired {
						t.Fatal("target activation preceded acquire acknowledgement", c)
					}
					if slices.Contains(c.args, "restart") {
						t.Fatal("restarted live previous manager", c)
					}
				}
				f.calls = nil
				result, err := f.i.installOwner(ctx, "/bundle", hostPublic(1), []byte("new"), p, probe, nil)
				want := outcome == "held" || outcome == "not-acquired"
				if want && (err != nil || !result.Ready) || !want && (err == nil || result.Ready) {
					t.Fatal(result, err)
				}
				if acquires != 1 {
					t.Fatal("same-token acquire not replayed exactly once", acquires)
				}
				starts, stops := 0, 0
				for _, c := range f.calls {
					if slices.Contains(c.args, "urbit-runner.service") {
						if slices.Contains(c.args, "start") {
							starts++
						}
						if slices.Contains(c.args, "stop") {
							stops++
						}
					}
				}
				if exited && starts != 1 || !exited && starts != 0 || want && stops != 1 || !want && stops != 0 {
					t.Fatal("unsafe dispatch counts", starts, stops)
				}
				if !want {
					j := retryJournal(t, f)
					if j.Stage != "prepared" || j.Token != token {
						t.Fatal(j)
					}
					data, err := os.ReadFile(f.i.path(access.ShellPath))
					if err != nil || string(data) != "old" {
						t.Fatal(string(data), err)
					}
					if _, err := f.i.activationEvidence(ctx, p.Name); err != nil {
						t.Fatal("refused acquire stopped previous", err)
					}
				}
			})
		}
	}
}

func TestRC11PreparedRetryRefusesNewStartupWorkWithoutCancellation(t *testing.T) {
	f, p, probe, token := preparedRetryFixture(t, "not-acquired")
	ctx := context.Background()
	if _, err := f.i.command(ctx, "/usr/bin/systemctl", "--user", "--machine="+p.Name+"@.host", "stop", "urbit-runner.service"); err != nil {
		t.Fatal(err)
	}
	busy := false
	f.beforeCommand = func(c call) {
		if slices.Contains(c.args, "start") && slices.Contains(c.args, "urbit-runner.service") {
			busy = true
		}
	}
	rpc := f.i.updateRPC
	f.i.updateRPC = func(ctx context.Context, socket string, uid uint32, pid int, req *protocol.ManagementUpdateRequest) (protocol.ManagementUpdateStatus, error) {
		if req != nil && req.Action == "acquire" {
			if req.Token != token {
				t.Fatal(req.Token)
			}
			if busy {
				return protocol.ManagementUpdateStatus{}, errors.New("update_busy: startup admitted work")
			}
		}
		return rpc(ctx, socket, uid, pid, req)
	}
	f.calls = nil
	if result, err := f.i.installOwner(ctx, "/bundle", hostPublic(1), []byte("new"), p, probe, nil); err == nil || result.Ready || !strings.Contains(err.Error(), "update_busy") {
		t.Fatal(result, err)
	}
	for _, c := range f.calls {
		if slices.Contains(c.args, "urbit-runner.service") && (slices.Contains(c.args, "stop") || slices.Contains(c.args, "enable")) {
			t.Fatal("canceled newly admitted work", c)
		}
	}
	if j := retryJournal(t, f); j.Stage != "prepared" || j.Token != token {
		t.Fatal(j)
	}
	busy = false
	f.beforeCommand = nil
	f.calls = nil
	if result, err := f.i.installOwner(ctx, "/bundle", hostPublic(1), []byte("new"), p, probe, nil); err != nil || !result.Ready {
		t.Fatal(result, err)
	}
	for _, c := range f.calls {
		if slices.Contains(c.args, "urbit-runner.service") && slices.Contains(c.args, "start") {
			t.Fatal("restarted already-live previous manager", c)
		}
	}
}

func TestRC11PreparedRetryFailuresPreservePreviousAndJournal(t *testing.T) {
	for _, fault := range []string{"installed-target", "retained-previous", "foreground", "start", "lost-start-ack", "readiness", "identity", "build", "schema", "acquire-ack"} {
		t.Run(fault, func(t *testing.T) {
			f, p, probe, token := preparedRetryFixture(t, "held")
			ctx := context.Background()
			if _, err := f.i.command(ctx, "/usr/bin/systemctl", "--user", "--machine="+p.Name+"@.host", "stop", "urbit-runner.service"); err != nil {
				t.Fatal(err)
			}
			j := retryJournal(t, f)
			rpc, run := f.i.updateRPC, f.i.run
			switch fault {
			case "installed-target":
				f.put(access.ShellPath, "new", 0755)
			case "retained-previous":
				f.put(generationPath(*j.Previous), "tampered", 0755)
			case "foreground":
				f.i.activationStopped = func(context.Context, string, string) error { return errors.New("surviving foreground listener") }
			case "start":
				f.fail = func(c call) error {
					if slices.Contains(c.args, "start") && slices.Contains(c.args, "urbit-runner.service") {
						return errors.New("start failed")
					}
					return nil
				}
			case "lost-start-ack":
				f.i.run = func(ctx context.Context, path string, args, env []string) ([]byte, error) {
					data, err := run(ctx, path, args, env)
					if err == nil && slices.Contains(args, "start") && slices.Contains(args, "urbit-runner.service") {
						return data, errors.New("start acknowledgement lost")
					}
					return data, err
				}
			}
			f.i.updateRPC = func(ctx context.Context, socket string, uid uint32, pid int, req *protocol.ManagementUpdateRequest) (protocol.ManagementUpdateStatus, error) {
				if fault == "readiness" {
					return protocol.ManagementUpdateStatus{}, errors.New("not serving yet")
				}
				status, err := rpc(ctx, socket, uid, pid, req)
				switch fault {
				case "identity":
					status.RunnerID = "runner-" + strings.Repeat("b", 32)
				case "build":
					status.ExecutableSHA256 = strings.Repeat("b", 64)
				case "schema":
					status.StateSchema++
				}
				if fault == "acquire-ack" && req != nil && req.Action == "acquire" {
					return status, errors.New("acquire acknowledgement lost again")
				}
				return status, err
			}
			f.calls = nil
			if result, err := f.i.installOwner(ctx, "/bundle", hostPublic(1), []byte("new"), p, probe, nil); err == nil || result.Ready {
				t.Fatal(result, err)
			}
			for _, c := range f.calls {
				if slices.Contains(c.args, "urbit-runner.service") && (slices.Contains(c.args, "stop") || slices.Contains(c.args, "enable")) {
					t.Fatal("failure activated target", c)
				}
			}
			if j := retryJournal(t, f); j.Stage != "prepared" || j.Token != token {
				t.Fatal(j)
			}
			if fault == "start" || fault == "lost-start-ack" || fault == "readiness" || fault == "acquire-ack" {
				f.fail, f.i.run, f.i.updateRPC = nil, run, rpc
				if result, err := f.i.installOwner(ctx, "/bundle", hostPublic(1), []byte("new"), p, probe, nil); err != nil || !result.Ready {
					t.Fatal("retry remained stuck", result, err)
				}
			}
		})
	}
}

func TestRC11PreparedRetryWaitsForPreviousReadiness(t *testing.T) {
	f, p, probe, _ := preparedRetryFixture(t, "held")
	ctx := context.Background()
	if _, err := f.i.command(ctx, "/usr/bin/systemctl", "--user", "--machine="+p.Name+"@.host", "stop", "urbit-runner.service"); err != nil {
		t.Fatal(err)
	}
	f.i.activationTimeout = time.Second
	rpc := f.i.updateRPC
	waiting, failedRead := false, false
	f.beforeCommand = func(c call) {
		if slices.Contains(c.args, "start") && slices.Contains(c.args, "urbit-runner.service") {
			waiting = true
		}
	}
	f.i.updateRPC = func(ctx context.Context, socket string, uid uint32, pid int, req *protocol.ManagementUpdateRequest) (protocol.ManagementUpdateStatus, error) {
		if waiting && !failedRead && req == nil {
			failedRead = true
			return protocol.ManagementUpdateStatus{}, errors.New("exec acknowledged before socket readiness")
		}
		return rpc(ctx, socket, uid, pid, req)
	}
	if result, err := f.i.installOwner(ctx, "/bundle", hostPublic(1), []byte("new"), p, probe, nil); err != nil || !result.Ready || !failedRead {
		t.Fatal(result, err)
	}
}

func TestRC11PreparedFreshInstallHasNoPreviousRecovery(t *testing.T) {
	f, p, probe := ownerFixture(t)
	if result, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("new"), p, probe, nil); err != nil || !result.Ready {
		t.Fatal(result, err)
	}
	if j := retryJournal(t, f); j.Previous != nil || j.Stage != "complete" {
		t.Fatal(j)
	}
	for _, c := range f.calls {
		if slices.Contains(c.args, "urbit-runner.service") && slices.Contains(c.args, "start") {
			t.Fatal("fresh install tried to revive a previous generation", c)
		}
	}
}

package hostinstall

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func retryJournal(t *testing.T, f *fixture) activationJournal {
	t.Helper()
	data, err := os.ReadFile(f.i.path(activationJournalPath))
	if err != nil {
		t.Fatal(err)
	}
	var j activationJournal
	if err := json.Unmarshal(data, &j); err != nil {
		t.Fatal(err)
	}
	return j
}

// Each command remains the existing synthetic service fixture. The complete
// owner installer entrypoint, durable journal and installed generations are real.
func verifiedRetryFixture(t *testing.T, rollback, released, completed bool) (*fixture, ownerhost.Profile, hostProbe) {
	t.Helper()
	f, p, probe := ownerFixture(t)
	ctx := context.Background()
	if _, err := f.i.installOwner(ctx, "/bundle", hostPublic(1), []byte("old"), p, probe, nil); err != nil {
		t.Fatal(err)
	}
	rpc := f.i.updateRPC
	if !completed {
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
	if rollback {
		failed := false
		f.fail = func(c call) error {
			if !failed && slices.Contains(c.args, "enable") && slices.Contains(c.args, "urbit-runner.service") {
				failed = true
				return errors.New("candidate activation failed")
			}
			return nil
		}
	}
	_, err := f.i.installOwner(ctx, "/bundle", hostPublic(1), []byte("new"), p, probe, nil)
	if err == nil && (!completed || rollback) || err != nil && completed && !rollback {
		t.Fatal(err)
	}
	want := "verified"
	if rollback {
		want = "rollback-verified"
	}
	if completed {
		want = "complete"
		if rollback {
			want = "rolled-back"
		}
	}
	if j := retryJournal(t, f); j.Stage != want {
		t.Fatal(j.Stage, want)
	}
	f.i.updateRPC, f.fail = rpc, nil
	return f, p, probe
}

func TestRC11VerifiedActivationRetryRestartsOnlyAbsentManager(t *testing.T) {
	for _, tc := range []struct {
		name                          string
		rollback, released, completed bool
	}{
		{"verified-held", false, false, false}, {"verified-released", false, true, false}, {"complete", false, true, true},
		{"rollback-held", true, false, false}, {"rollback-released", true, true, false}, {"rolled-back", true, true, true},
	} {
		for _, exited := range []bool{false, true} {
			if tc.rollback && tc.completed && !exited {
				continue
			} // an explicit new update retry, not availability recovery
			t.Run(tc.name+map[bool]string{false: "/live", true: "/exited"}[exited], func(t *testing.T) {
				f, p, probe := verifiedRetryFixture(t, tc.rollback, tc.released, tc.completed)
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
				f.calls = nil
				result, err := f.i.installOwner(ctx, "/bundle", hostPublic(1), []byte("new"), p, probe, nil)
				if err != nil || !result.Ready {
					t.Fatal(result, err)
				}
				starts := 0
				for _, c := range f.calls {
					if !slices.Contains(c.args, "urbit-runner.service") {
						continue
					}
					if slices.Contains(c.args, "stop") || slices.Contains(c.args, "restart") || slices.Contains(c.args, "enable") {
						t.Fatal("recovery disturbed released manager", c)
					}
					if slices.Contains(c.args, "start") {
						starts++
					}
				}
				if exited && starts != 1 || !exited && starts != 0 {
					t.Fatal("wrong recovery dispatch", starts)
				}
				after, err := f.i.activationEvidence(ctx, p.Name)
				if err != nil {
					t.Fatal(err)
				}
				if exited && after.PID == before.PID || !exited && after != before {
					t.Fatal("unexpected service identity", before, after)
				}
				wantBuild, wantStage := "new", "complete"
				if tc.rollback {
					wantBuild, wantStage = "old", "rolled-back"
				}
				data, err := os.ReadFile(f.i.path(access.ShellPath))
				if err != nil || string(data) != wantBuild || retryJournal(t, f).Stage != wantStage {
					t.Fatal(string(data), err)
				}
			})
		}
	}
}

func TestRC11VerifiedRetryRefusesUnexpectedGenerationAndForegroundListener(t *testing.T) {
	for _, fault := range []string{"installed", "retained", "foreground", "token", "identity", "peer", "build"} {
		t.Run(fault, func(t *testing.T) {
			f, p, probe := verifiedRetryFixture(t, false, false, false)
			ctx := context.Background()
			if _, err := f.i.command(ctx, "/usr/bin/systemctl", "--user", "--machine="+p.Name+"@.host", "stop", "urbit-runner.service"); err != nil {
				t.Fatal(err)
			}
			j := retryJournal(t, f)
			switch fault {
			case "installed":
				f.put(access.ShellPath, "unexpected", 0755)
			case "retained":
				f.put(generationPath(j.Target), "unexpected", 0755)
			case "foreground":
				f.i.activationStopped = func(context.Context, string, string) error { return errors.New("surviving foreground listener") }
			}
			rpc := f.i.updateRPC
			releases := 0
			f.i.updateRPC = func(ctx context.Context, socket string, uid uint32, pid int, req *protocol.ManagementUpdateRequest) (protocol.ManagementUpdateStatus, error) {
				if req != nil && req.Action == "release" {
					releases++
				}
				v, err := rpc(ctx, socket, uid, pid, req)
				switch fault {
				case "token":
					v.Token = strings.Repeat("f", 64)
				case "identity":
					v.RunnerID = "runner-" + strings.Repeat("b", 32)
				case "peer":
					v.PID++
				case "build":
					v.ExecutableSHA256 = strings.Repeat("f", 64)
				}
				return v, err
			}
			f.calls = nil
			if result, err := f.i.installOwner(ctx, "/bundle", hostPublic(1), []byte("new"), p, probe, nil); err == nil || result.Ready {
				t.Fatal(result, err)
			}
			if releases != 0 {
				t.Fatal("released before verifying recovery evidence")
			}
			for _, c := range f.calls {
				if slices.Contains(c.args, "urbit-runner.service") && (slices.Contains(c.args, "stop") || fault == "installed" || fault == "retained" || fault == "foreground") {
					t.Fatal("unsafe recovery action", c)
				}
			}
		})
	}
}

func TestRC11VerifiedRetryStartFailureRemainsResumable(t *testing.T) {
	f, p, probe := verifiedRetryFixture(t, false, true, false)
	ctx := context.Background()
	if _, err := f.i.command(ctx, "/usr/bin/systemctl", "--user", "--machine="+p.Name+"@.host", "stop", "urbit-runner.service"); err != nil {
		t.Fatal(err)
	}
	f.fail = func(c call) error {
		if slices.Contains(c.args, "start") && slices.Contains(c.args, "urbit-runner.service") {
			return errors.New("transient start failure")
		}
		return nil
	}
	if _, err := f.i.installOwner(ctx, "/bundle", hostPublic(1), []byte("new"), p, probe, nil); err == nil {
		t.Fatal("start failure ignored")
	}
	if j := retryJournal(t, f); j.Stage != "verified" {
		t.Fatal(j.Stage)
	}
	f.fail = nil
	if result, err := f.i.installOwner(ctx, "/bundle", hostPublic(1), []byte("new"), p, probe, nil); err != nil || !result.Ready {
		t.Fatal(result, err)
	}
}

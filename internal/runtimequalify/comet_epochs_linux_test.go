//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func cometEpochFixture(t *testing.T) *fakeEngine {
	f := cometFixture(t)
	f.d.profile = ProfileCometEpochs
	execute := f.d.execute
	f.d.execute = func(ctx context.Context, a, e []string) (string, error) {
		out, err := execute(ctx, a, e)
		if err != nil || a[4] != "container" {
			return out, err
		}
		for _, c := range f.containers {
			if len(a) <= 6 || c.id != a[6] {
				continue
			}
			if a[5] == "start" && c.kind == "comet-boot" {
				writeQualificationPierEpoch(t, filepath.Dir(a[3]), "comet", "0i0", false)
			}
			if a[5] == "wait" && isCometRoll(c.kind) {
				writeQualificationPierEpoch(t, filepath.Dir(a[3]), "comet", fmt.Sprintf("0i%d", int(c.kind[len(c.kind)-1]-'0')*100), true)
			}
		}
		return out, err
	}
	f.d.remove = func(*os.Root, string) error { t.Fatal("epoch preview tried to delete pier data"); return nil }
	return f
}

func TestCometEpochRollsIdentityRestartsAndPreview(t *testing.T) {
	f := cometEpochFixture(t)
	execute := f.d.execute
	dispatches := 0
	f.d.execute = func(ctx context.Context, a, e []string) (string, error) {
		if a[4] == "container" && a[5] == "create" && strings.Contains(a[7], "-comet-roll-") {
			dispatches++
			var stored Result
			data, err := os.ReadFile(filepath.Join(filepath.Dir(a[3]), "report.json"))
			if err != nil || json.Unmarshal(data, &stored) != nil || stored.Comet == nil || !stored.Comet.Uncertain || stored.EpochLifecycle == nil || len(stored.EpochLifecycle.Rolls) != dispatches || !stored.EpochLifecycle.Rolls[dispatches-1].Uncertain {
				t.Fatal("roll dispatched before durable intent")
			}
			if !reflect.DeepEqual(a[len(a)-4:], []string{"roll", "--loom", "31", "/urbit/comet"}) || !strings.Contains(strings.Join(a, " "), "--network none") {
				t.Fatal("wrong comet roll command or network")
			}
			for _, arg := range a {
				if arg == "-F" || arg == "-B" || arg == "-c" || arg == "--publish" || strings.Contains(arg, "boot.pill") {
					t.Fatal("creation or publication in utility")
				}
			}
		}
		return execute(ctx, a, e)
	}
	r, err := run(context.Background(), nil, f.d)
	if err != nil || !r.CheckpointPassed || !r.CleanupComplete || r.RuntimeQualified || len(r.Containers) != 9 || dispatches != 3 || len(f.containers) != 0 || len(f.networks) != 0 {
		t.Fatalf("comet epoch workflow %+v %v", r, err)
	}
	if r.Comet.Ship != fixtureComet || r.Comet.Uncertain || !r.Comet.ResumeCompleted || r.Comet.FinishedAt == nil {
		t.Fatal("comet identity or final outcome missing")
	}
	e := r.EpochLifecycle
	if len(e.Final.Entries) != 4 || len(e.Preview.Selected) != 1 || len(e.Preview.Retained) != 3 || e.Preview.DeletionSupported {
		t.Fatal("wrong final preview")
	}
	for _, c := range r.Containers {
		if !c.Removed || c.CreateUncertain || isNetworkedKind(c.Kind) && !c.HostHTTPVerified {
			t.Fatal("unresolved runtime or identity check", c)
		}
	}
	for i, roll := range e.Rolls {
		if roll.Uncertain || roll.PostRestart == nil || len(roll.PostRestart.Entries) != i+2 || roll.After.Entries[i+1].SnapshotBytes <= 0 {
			t.Fatal("incomplete roll evidence")
		}
	}
	entries, err := os.ReadDir(filepath.Join(r.Directory, "data/comet/.urb/log"))
	if err != nil || len(entries) != 4 {
		t.Fatal("preview changed the epoch set", err)
	}
	q := &qualifier{d: f.d, base: filepath.Dir(r.Directory), uid: f.i.uid, gid: f.i.gid}
	if err := q.priorReports(context.Background()); err != nil {
		t.Fatal("complete comet epoch history rejected", err)
	}
	assertReport(t, r)
}

func TestCometEpochFailuresDoNotContinueAndRemainQuarantined(t *testing.T) {
	for _, fault := range []string{"timeout", "no-epoch", "snapshot", "restart-identity", "final-identity", "admission", "restart-epoch"} {
		t.Run(fault, func(t *testing.T) {
			f := cometEpochFixture(t)
			execute := f.d.execute
			f.d.execute = func(ctx context.Context, a, e []string) (string, error) {
				if a[4] == "container" && a[5] == "wait" && fault == "timeout" {
					return "", errors.New("utility timeout")
				}
				out, err := execute(ctx, a, e)
				if err != nil {
					return out, err
				}
				if a[4] == "container" && a[5] == "wait" {
					path := filepath.Join(filepath.Dir(a[3]), "data/comet/.urb/log/0i100")
					if fault == "no-epoch" {
						if err := os.RemoveAll(path); err != nil {
							t.Fatal(err)
						}
					} else if fault == "snapshot" {
						if err := os.Remove(filepath.Join(path, "image.bin")); err != nil {
							t.Fatal(err)
						}
					}
				}
				if fault == "restart-epoch" && len(f.events) > 0 && f.events[len(f.events)-1] == "stop-comet-epoch-restart-1" {
					writeQualificationPierEpoch(t, filepath.Dir(a[3]), "comet", "0i50", true)
				}
				return out, err
			}
			read := f.d.hostIdentity
			f.d.hostIdentity = func(ctx context.Context, endpoint string) (string, error) {
				for _, c := range f.containers {
					if fault == "restart-identity" && c.kind == "comet-epoch-restart-1" || fault == "final-identity" && c.kind == "comet-resume" {
						return "~marzod-salbud-salwex-sogref--tasweb-navfun-batwet-sigfet", nil
					}
				}
				return read(ctx, endpoint)
			}
			if fault == "admission" {
				f.d.admission = func(string) (uint64, uint64, error) {
					for _, event := range f.events {
						if event == "rm-comet-boot" {
							return 0, 8 << 30, nil
						}
					}
					return 5 << 30, 8 << 30, nil
				}
			}
			r, err := run(context.Background(), nil, f.d)
			if err == nil || r.CheckpointPassed || !r.CleanupComplete || !r.Comet.Uncertain {
				t.Fatal("unsafe outcome accepted", fault, err)
			}
			phase := "comet-roll-1"
			if fault == "restart-identity" || fault == "restart-epoch" {
				phase = "comet-epoch-restart-1"
			}
			if fault == "final-identity" {
				phase = "comet-resume"
			}
			if len(r.Phases) < 2 || r.Phases[len(r.Phases)-2].Name != phase {
				t.Fatal("fixture failed before the intended boundary", fault, r.Phases, err)
			}
			for _, c := range r.Containers {
				if fault != "final-identity" && c.Kind == "comet-roll-2" || (fault == "timeout" || fault == "no-epoch" || fault == "snapshot" || fault == "admission") && c.Kind == "comet-epoch-restart-1" {
					t.Fatal("continued after a failed roll or restart")
				}
			}
			q := &qualifier{d: f.d, base: filepath.Dir(r.Directory), uid: f.i.uid, gid: f.i.gid}
			if err := q.priorReports(context.Background()); err == nil {
				t.Fatal("cleanup erased unresolved epoch history")
			}
		})
	}
}

func TestCometEpochHistoryRequiresAllEvidence(t *testing.T) {
	for _, field := range []string{"comet", "epoch_lifecycle", "final", "preview", "post_restart", "uncertain", "host_http_verified"} {
		t.Run(field, func(t *testing.T) {
			f := cometEpochFixture(t)
			r, err := run(context.Background(), nil, f.d)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(r.ReportPath)
			if err != nil {
				t.Fatal(err)
			}
			var data map[string]any
			if err := json.Unmarshal(raw, &data); err != nil {
				t.Fatal(err)
			}
			switch field {
			case "comet", "epoch_lifecycle":
				delete(data, field)
			case "final", "preview":
				delete(data["epoch_lifecycle"].(map[string]any), field)
			case "post_restart", "uncertain":
				delete(data["epoch_lifecycle"].(map[string]any)["rolls"].([]any)[0].(map[string]any), field)
			case "host_http_verified":
				delete(data["containers"].([]any)[3].(map[string]any), field)
			}
			raw, err = json.Marshal(data)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(r.ReportPath, raw, 0600); err != nil {
				t.Fatal(err)
			}
			q := &qualifier{d: f.d, base: filepath.Dir(r.Directory), uid: f.i.uid, gid: f.i.gid}
			if err := q.priorReport(r.RunID); err == nil {
				t.Fatal("missing evidence accepted", field)
			}
		})
	}
}

func TestCometEpochProfileDoesNotEnableOtherCommandsOrManagedEpochs(t *testing.T) {
	q := &qualifier{d: dependencies{profile: ProfileCometEpochs}}
	for _, kind := range []string{"boot", "roll-1", "epoch-restart-1", "comet-roll-4", "pack", "comet-meld", "epoch-recovery"} {
		if _, err := q.create(context.Background(), kind); err == nil || !strings.Contains(err.Error(), "not allowed") {
			t.Fatal("profile widened", kind, err)
		}
	}
	for _, profile := range []string{ProfileHTTP, ProfileComet} {
		q.d.profile, q.d.managed = profile, true
		for _, kind := range []string{"comet-roll-1", "comet-epoch-restart-1"} {
			if _, err := q.create(context.Background(), kind); err == nil || !strings.Contains(err.Error(), "not allowed") {
				t.Fatal("managed epoch enabled", err)
			}
		}
	}
	f := cometEpochFixture(t)
	f.d.managed = true
	if r, err := run(context.Background(), nil, f.d); err == nil || r.RunID != "" || len(f.calls) != 0 {
		t.Fatal("managed qualification dispatched")
	}
}

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

	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

func writeQualificationEpoch(t *testing.T, dir, name string, snapshot bool) {
	t.Helper()
	writeQualificationPierEpoch(t, dir, "fake-zod", name, snapshot)
}

func writeQualificationPierEpoch(t *testing.T, dir, pier, name string, snapshot bool) {
	t.Helper()
	writePierEpochPath(t, filepath.Join(dir, "data", pier), name, snapshot)
}

func writePierEpochPath(t *testing.T, pier, name string, snapshot bool) {
	t.Helper()
	path := filepath.Join(pier, ".urb/log", name)
	if err := os.MkdirAll(path, 0700); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{"epoc.txt": "2", "vere.txt": candidate.VereVersion, "data.mdb": "fixture, not an LMDB database", "lock.mdb": "fixture lock"}
	if snapshot {
		files["image.bin"] = "fixture, not a Vere memory image"
	}
	for file, body := range files {
		if err := os.WriteFile(filepath.Join(path, file), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func epochQualificationFixture(t *testing.T) *fakeEngine {
	f := fixture(t)
	f.d.profile = ProfileEpochs
	execute := f.d.execute
	f.d.execute = func(ctx context.Context, a, e []string) (string, error) {
		out, err := execute(ctx, a, e)
		if err != nil || a[4] != "container" {
			return out, err
		}
		if a[5] == "start" && f.events[len(f.events)-1] == "start-boot" {
			writeQualificationEpoch(t, filepath.Dir(a[3]), "0i0", false)
		}
		if a[5] == "wait" {
			for _, c := range f.containers {
				if c.id == a[6] && isRollKind(c.kind) {
					writeQualificationEpoch(t, filepath.Dir(a[3]), fmt.Sprintf("0i%d", int(c.kind[len(c.kind)-1]-'0')*100), true)
				}
			}
		}
		return out, err
	}
	return f
}

func TestEpochQualificationRollsRestartsAndPreviewsWithoutDeletion(t *testing.T) {
	f := epochQualificationFixture(t)
	execute := f.d.execute
	dispatches := 0
	f.d.execute = func(ctx context.Context, a, e []string) (string, error) {
		if a[4] == "container" && a[5] == "create" && strings.Contains(a[7], "-roll-") {
			dispatches++
			var stored Result
			data, err := os.ReadFile(filepath.Join(filepath.Dir(a[3]), "report.json"))
			if err != nil || json.Unmarshal(data, &stored) != nil || stored.EpochLifecycle == nil || len(stored.EpochLifecycle.Rolls) != dispatches || !stored.EpochLifecycle.Rolls[dispatches-1].Uncertain {
				t.Fatal("roll dispatched before durable uncertainty")
			}
			if !reflect.DeepEqual(a[len(a)-4:], []string{"roll", "--loom", "31", "/urbit/fake-zod"}) {
				t.Fatalf("unqualified roll arguments: %v", a)
			}
			for _, arg := range a {
				if arg == "-F" || arg == "-B" || arg == "-c" || arg == "--publish" || strings.Contains(arg, "boot.pill") {
					t.Fatal("roll mixed bootstrap or network arguments")
				}
			}
		}
		return execute(ctx, a, e)
	}
	var phases []string
	r, err := run(context.Background(), func(p string) { phases = append(phases, p) }, f.d)
	if err != nil || !r.CheckpointPassed || !r.CleanupComplete || r.RuntimeQualified || r.Network != nil || len(r.Containers) != 8 || dispatches != 3 {
		t.Fatalf("epoch lifecycle failed: %+v %v", r, err)
	}
	if !reflect.DeepEqual(phases, []string{"engine", "image", "pill", "probe", "boot", "epoch-baseline", "roll-1", "epoch-restart-1", "roll-2", "epoch-restart-2", "roll-3", "epoch-restart-3", "epoch-preview", "cleanup"}) {
		t.Fatalf("phase sequence: %v", phases)
	}
	e := r.EpochLifecycle
	if len(e.Initial.Entries) != 1 || len(e.Rolls) != 3 || len(e.Final.Entries) != 4 || e.Preview == nil || len(e.Preview.Selected) != 1 || len(e.Preview.Retained) != 3 || e.Preview.DeletionSupported {
		t.Fatalf("incorrect epoch evidence: %+v", e)
	}
	for i, roll := range e.Rolls {
		if roll.Uncertain || roll.FinishedAt == nil || roll.FinishedAt.Before(roll.StartedAt) || len(roll.After.Entries) != i+2 || roll.PostRestart == nil || roll.After.Entries[i+1].SnapshotBytes <= 0 {
			t.Fatalf("incomplete roll: %+v", roll)
		}
	}
	for _, c := range r.Containers {
		if !c.Removed || c.CreateUncertain || ((c.Kind == "boot" || isEpochRestartKind(c.Kind)) && !c.PrivateHTTPReachable) {
			t.Fatalf("container not qualified/removed: %+v", c)
		}
	}
	entries, err := os.ReadDir(filepath.Join(r.Directory, "data/fake-zod/.urb/log"))
	if err != nil || len(entries) != 4 {
		t.Fatal("preview deleted an epoch")
	}
	assertReport(t, r)
	q := &qualifier{d: f.d, base: filepath.Dir(r.Directory), uid: f.i.uid, gid: f.i.gid}
	if err := q.priorReports(context.Background()); err != nil {
		t.Fatalf("completed report rejected: %v", err)
	}
}

func TestRollFailureQuarantineSurvivesResourceCleanup(t *testing.T) {
	for _, fault := range []string{"timeout", "nonzero", "OOM", "no new epoch", "missing snapshot", "extra epoch", "final persistence"} {
		t.Run(fault, func(t *testing.T) {
			f := epochQualificationFixture(t)
			execute := f.d.execute
			f.d.execute = func(ctx context.Context, a, e []string) (string, error) {
				if a[4] == "container" && a[5] == "wait" && fault == "timeout" {
					return "", context.DeadlineExceeded
				}
				out, err := execute(ctx, a, e)
				if a[4] == "container" && a[5] == "wait" {
					for _, c := range f.containers {
						if fault == "nonzero" {
							mapAt(c.doc, "State")["ExitCode"] = 1
						}
						if fault == "OOM" {
							mapAt(c.doc, "State")["OOMKilled"] = true
						}
					}
					path := filepath.Join(filepath.Dir(a[3]), "data/fake-zod/.urb/log/0i100")
					switch fault {
					case "no new epoch":
						if err := os.RemoveAll(path); err != nil {
							t.Fatal(err)
						}
					case "missing snapshot":
						if err := os.Remove(filepath.Join(path, "image.bin")); err != nil {
							t.Fatal(err)
						}
					case "extra epoch":
						writeQualificationEpoch(t, filepath.Dir(a[3]), "0i50", true)
					}
				}
				return out, err
			}
			if fault == "final persistence" {
				f.d.persist = func(r Result) error {
					if e := r.EpochLifecycle; e != nil && len(e.Rolls) > 0 && !e.Rolls[0].Uncertain {
						return errors.New("journal failure")
					}
					return (&qualifier{r: r}).save()
				}
			}
			r, err := run(context.Background(), nil, f.d)
			if err == nil || r.CheckpointPassed || !r.CleanupComplete || r.EpochLifecycle == nil || len(r.EpochLifecycle.Rolls) != 1 || !r.EpochLifecycle.Rolls[0].Uncertain || r.EpochLifecycle.Rolls[0].FinishedAt != nil {
				t.Fatalf("roll quarantine lost: %+v %v", r, err)
			}
			for _, c := range r.Containers {
				if isEpochRestartKind(c.Kind) {
					t.Fatal("restarted after an uncertain roll")
				}
			}
			assertReport(t, r)
			calls := len(f.calls)
			f.d.randomID = func() (string, error) { return strings.Repeat("b", 32), nil }
			next, err := run(context.Background(), nil, f.d)
			if err == nil || next.RunID != "" || len(f.calls) != calls || !strings.Contains(err.Error(), "manual resolution") {
				t.Fatalf("quarantined history ignored: %+v %v", next, err)
			}
		})
	}
}

func TestEpochQualificationRechecksAdmissionForEveryBootAndRoll(t *testing.T) {
	for failAt := 2; failAt <= 8; failAt++ {
		t.Run(fmt.Sprint(failAt), func(t *testing.T) {
			f := epochQualificationFixture(t)
			checks := 0
			f.d.admission = func(string) (uint64, uint64, error) {
				checks++
				if checks == failAt {
					return 5<<30 - 1, 8 << 30, nil
				}
				return 5 << 30, 8 << 30, nil
			}
			r, err := run(context.Background(), nil, f.d)
			if err == nil || checks != failAt || r.CheckpointPassed || !r.CleanupComplete || !strings.Contains(err.Error(), "admission") {
				t.Fatalf("stale admission accepted: checks=%d %+v %v", checks, r, err)
			}
			if e := r.EpochLifecycle; e != nil {
				for _, roll := range e.Rolls {
					if roll.Uncertain {
						t.Fatal("capacity refusal quarantined an undispatched roll")
					}
				}
			}
			assertReport(t, r)
		})
	}
}

func TestEpochRestartFailureCannotPassQualification(t *testing.T) {
	f := epochQualificationFixture(t)
	execute := f.d.execute
	f.d.execute = func(ctx context.Context, a, e []string) (string, error) {
		out, err := execute(ctx, a, e)
		if a[4] == "container" && a[5] == "wait" {
			f.http = "503"
		}
		return out, err
	}
	r, err := run(context.Background(), nil, f.d)
	if err == nil || r.CheckpointPassed || !r.CleanupComplete || r.EpochLifecycle.Rolls[0].Uncertain || r.EpochLifecycle.Rolls[0].PostRestart != nil {
		t.Fatalf("restart failure hidden by utility success: %+v %v", r, err)
	}
	assertReport(t, r)
}

func TestRollIsNotAvailableToManagedOrOtherProfiles(t *testing.T) {
	for _, profile := range []string{ProfileLifecycle, ProfileHTTP, ProfileMeld} {
		f := fixture(t)
		f.d.profile = profile
		q := &qualifier{d: f.d}
		if _, err := q.create(context.Background(), "roll-1"); err == nil || len(f.calls) != 0 {
			t.Fatal("roll admitted outside epoch qualification")
		}
	}
	f := epochQualificationFixture(t)
	f.d.managed = true
	q := &qualifier{d: f.d}
	if err := q.roll(context.Background(), "roll-1"); err == nil {
		t.Fatal("managed roll enabled")
	}
}

func TestEpochHistoryRequiresResolvedAndConsistentEvidence(t *testing.T) {
	f := epochQualificationFixture(t)
	r, err := run(context.Background(), nil, f.d)
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(r.ReportPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, fault := range []string{"missing history", "missing uncertainty", "null uncertainty", "uncertain", "missing after", "missing snapshot", "missing restart", "bad preview", "too many containers", "wrong profile"} {
		t.Run(fault, func(t *testing.T) {
			var doc map[string]any
			if err := json.Unmarshal(original, &doc); err != nil {
				t.Fatal(err)
			}
			e := mapAt(doc, "epoch_lifecycle")
			roll := e["rolls"].([]any)[0].(map[string]any)
			switch fault {
			case "missing history":
				delete(doc, "epoch_lifecycle")
			case "missing uncertainty":
				delete(roll, "uncertain")
			case "null uncertainty":
				roll["uncertain"] = nil
			case "uncertain":
				roll["uncertain"] = true
			case "missing after":
				delete(roll, "after")
			case "missing snapshot":
				mapAt(roll, "after")["entries"].([]any)[1].(map[string]any)["snapshot_bytes"] = 0
			case "missing restart":
				delete(roll, "post_restart")
			case "bad preview":
				mapAt(e, "preview")["deletion_supported"] = true
			case "too many containers":
				doc["containers"] = append(doc["containers"].([]any), doc["containers"].([]any)[0])
			case "wrong profile":
				doc["profile"] = ProfileMeld
			}
			if err := os.WriteFile(r.ReportPath, []byte(encode(t, doc)), 0600); err != nil {
				t.Fatal(err)
			}
			q := &qualifier{d: f.d, base: filepath.Dir(r.Directory), uid: f.i.uid, gid: f.i.gid}
			if err := q.priorReports(context.Background()); err == nil {
				t.Fatal("malformed epoch history accepted")
			}
		})
	}
}

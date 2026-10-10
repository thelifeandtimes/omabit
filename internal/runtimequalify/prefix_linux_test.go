//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func prefixFixture(t *testing.T) *fakeEngine {
	f := pruneFixture(t)
	f.d.profile = ProfilePrefix
	return f
}

func TestRepeatedPrefixDeletionRetainsExactSuffixAndRecovers(t *testing.T) {
	f := prefixFixture(t)
	var removed []string
	f.d.remove = func(root *os.Root, name string) error {
		var stored Result
		data, err := os.ReadFile(filepath.Join(f.d.home, ".local/share/urbit-runner/qualification", strings.Repeat("a", 32), "report.json"))
		if err != nil || json.Unmarshal(data, &stored) != nil || len(stored.PrefixPrunes) == 0 {
			t.Fatal("prefix mutation preceded journal")
		}
		p := stored.PrefixPrunes[len(stored.PrefixPrunes)-1]
		if filepath.Base(root.Name()) == "chk" {
			if p.Recovery == nil || !p.Recovery.Uncertain || name != "image.bin" {
				t.Fatal("checkpoint removed without recovery intent")
			}
		} else if !p.Uncertain || len(p.Targets) == 0 {
			t.Fatal("unlink without whole-prefix intent")
		}
		removed = append(removed, filepath.Join(filepath.Base(root.Name()), name))
		return root.Remove(name)
	}
	r, err := run(context.Background(), nil, f.d)
	if err != nil || !r.CheckpointPassed || !r.CleanupComplete || r.RuntimeQualified || r.Prune != nil || r.Recovery != nil || len(r.Containers) != 16 || len(r.PrefixPrunes) != 2 {
		t.Fatalf("prefix qualification: %+v %v", r, err)
	}
	want := []string{"0i0/data.mdb", "0i0/epoc.txt", "0i0/lock.mdb", "0i0/vere.txt", "log/0i0", "0i100/data.mdb", "0i100/epoc.txt", "0i100/image.bin", "0i100/lock.mdb", "0i100/vere.txt", "log/0i100", "chk/image.bin", "0i200/data.mdb", "0i200/epoc.txt", "0i200/image.bin", "0i200/lock.mdb", "0i200/vere.txt", "log/0i200", "chk/image.bin"}
	if !reflect.DeepEqual(removed, want) {
		t.Fatalf("incorrect deletion set/order: %v", removed)
	}
	for i, p := range r.PrefixPrunes {
		_, selected, remaining := prefixShape(i + 1)
		if p.Uncertain || p.FinishedAt == nil || len(p.Targets) != selected || len(p.After.Entries) != remaining || p.PostRestart == nil || p.Recovery == nil || p.Recovery.Uncertain || !p.Recovery.CheckpointRemoved || p.Recovery.PostRestart == nil || !sameRetainedPrefix(p.Before, p.After, selected) {
			t.Fatalf("incomplete round: %+v", p)
		}
	}
	entries, err := os.ReadDir(filepath.Join(r.Directory, "data/fake-zod/.urb/log"))
	if err != nil || len(entries) != 2 || entries[0].Name() != "0i300" || entries[1].Name() != "0i400" {
		t.Fatal("newest two epochs not retained")
	}
	assertReport(t, r)
	q := &qualifier{d: f.d, base: filepath.Dir(r.Directory), uid: f.i.uid, gid: f.i.gid}
	if err := q.priorReports(context.Background()); err != nil {
		t.Fatalf("completed history rejected: %v", err)
	}
}

func TestPrefixPartialFailuresAreQuarantined(t *testing.T) {
	for _, fault := range []string{"between epochs", "snapshot unlink", "second prefix", "changed retained", "intent persistence", "completion persistence"} {
		t.Run(fault, func(t *testing.T) {
			f := prefixFixture(t)
			once := false
			f.d.persist = func(r Result) error {
				if len(r.PrefixPrunes) > 0 && !once {
					p := r.PrefixPrunes[len(r.PrefixPrunes)-1]
					if fault == "intent persistence" || (fault == "completion persistence" && !p.Uncertain) {
						once = true
						return errors.New("journal failure")
					}
				}
				return (&qualifier{r: r}).save()
			}
			calls := 0
			f.d.remove = func(root *os.Root, name string) error {
				base := filepath.Base(root.Name())
				calls++
				if (fault == "between epochs" && base == "0i100" && name == "data.mdb") || (fault == "snapshot unlink" && base == "0i100" && name == "image.bin") || (fault == "second prefix" && base == "0i200") {
					return errors.New("injected unlink failure")
				}
				err := root.Remove(name)
				if fault == "changed retained" && calls == 1 {
					log := filepath.Join(f.d.home, ".local/share/urbit-runner/qualification", strings.Repeat("a", 32), "data/fake-zod/.urb/log")
					requireFixtureRoot(t, root, filepath.Join(log, "0i0"))
					if err := os.WriteFile(filepath.Join(log, "0i300/data.mdb"), []byte("changed during prune"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				return err
			}
			r, err := run(context.Background(), nil, f.d)
			if err == nil || r.CheckpointPassed || !r.CleanupComplete || len(r.PrefixPrunes) == 0 {
				t.Fatalf("partial prefix accepted: %+v %v", r, err)
			}
			p := r.PrefixPrunes[len(r.PrefixPrunes)-1]
			if !p.Uncertain || p.FinishedAt != nil || p.PostRestart != nil || p.Recovery != nil {
				t.Fatal("partial deletion advanced to restart/recovery")
			}
			if fault == "intent persistence" && calls != 0 {
				t.Fatal("deletion preceded durable intent")
			}
			assertReport(t, r)
			before := len(f.calls)
			f.d.randomID = func() (string, error) { return strings.Repeat("b", 32), nil }
			next, err := run(context.Background(), nil, f.d)
			if err == nil || next.RunID != "" || len(f.calls) != before {
				t.Fatal("partial prefix automatically retried")
			}
		})
	}
}

func TestPrefixStaleSelectionAndFailedRecoveryBlockProgress(t *testing.T) {
	for _, fault := range []string{"stale first", "stale second", "first recovery", "second recovery"} {
		t.Run(fault, func(t *testing.T) {
			f := prefixFixture(t)
			execute := f.d.execute
			f.d.execute = func(ctx context.Context, a, e []string) (string, error) {
				if a[4] == "container" && a[5] == "wait" {
					for _, c := range f.containers {
						if c.id == a[6] && ((fault == "first recovery" && c.kind == "prefix-recovery-1") || (fault == "second recovery" && c.kind == "prefix-recovery-2")) {
							return "", context.DeadlineExceeded
						}
					}
				}
				return execute(ctx, a, e)
			}
			r, err := run(context.Background(), func(phase string) {
				if (fault == "stale first" && phase == "prefix-delete-1") || (fault == "stale second" && phase == "prefix-delete-2") {
					name := "0i0"
					if fault == "stale second" {
						name = "0i200"
					}
					path := filepath.Join(f.d.home, ".local/share/urbit-runner/qualification", strings.Repeat("a", 32), "data/fake-zod/.urb/log", name, "data.mdb")
					if err := os.WriteFile(path, []byte("changed after selection"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			}, f.d)
			if err == nil || r.CheckpointPassed || !r.CleanupComplete {
				t.Fatalf("failed prefix stage passed: %+v %v", r, err)
			}
			if strings.Contains(fault, "recovery") {
				p := r.PrefixPrunes[len(r.PrefixPrunes)-1]
				if p.Recovery == nil || !p.Recovery.Uncertain || !p.Recovery.CheckpointRemoved || p.Recovery.PostRestart != nil {
					t.Fatal("recovery uncertainty lost")
				}
				q := &qualifier{d: f.d, base: filepath.Dir(r.Directory), uid: f.i.uid, gid: f.i.gid}
				if err := q.priorReports(context.Background()); err == nil {
					t.Fatal("uncertain recovery accepted")
				}
			} else {
				expected := 0
				if fault == "stale second" {
					expected = 1
				}
				if len(r.PrefixPrunes) != expected {
					t.Fatal("stale selection began deletion")
				}
			}
			assertReport(t, r)
		})
	}
}

func TestPrefixReportCannotDropUncertaintyOrTargets(t *testing.T) {
	f := prefixFixture(t)
	r, err := run(context.Background(), nil, f.d)
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(r.ReportPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, fault := range []string{"missing marker", "uncertain", "missing target", "protected cutoff", "missing recovery marker", "unfinished recovery", "wrong profile", "missing container"} {
		t.Run(fault, func(t *testing.T) {
			var doc map[string]any
			if err := json.Unmarshal(original, &doc); err != nil {
				t.Fatal(err)
			}
			p := doc["prefix_prunes"].([]any)[0].(map[string]any)
			switch fault {
			case "missing marker":
				delete(p, "uncertain")
			case "uncertain":
				p["uncertain"] = true
			case "missing target":
				p["targets"] = p["targets"].([]any)[:1]
			case "protected cutoff":
				p["cutoff"] = "0i400"
			case "missing recovery marker":
				delete(mapAt(p, "recovery"), "uncertain")
			case "unfinished recovery":
				delete(mapAt(p, "recovery"), "finished_at")
			case "wrong profile":
				doc["profile"] = ProfilePrune
			case "missing container":
				doc["containers"] = doc["containers"].([]any)[:15]
			}
			if err := os.WriteFile(r.ReportPath, []byte(encode(t, doc)), 0600); err != nil {
				t.Fatal(err)
			}
			q := &qualifier{d: f.d, base: filepath.Dir(r.Directory), uid: f.i.uid, gid: f.i.gid}
			if err := q.priorReports(context.Background()); err == nil {
				t.Fatal("invalid prefix report accepted")
			}
		})
	}
}

func TestPrefixFailedFinalRestartCannotPass(t *testing.T) {
	f := prefixFixture(t)
	execute := f.d.execute
	f.d.execute = func(ctx context.Context, a, e []string) (string, error) {
		out, err := execute(ctx, a, e)
		if a[4] == "container" && a[5] == "wait" {
			for _, c := range f.containers {
				if c.id == a[6] && c.kind == "prefix-recovery-2" {
					f.http = "503"
				}
			}
		}
		return out, err
	}
	r, err := run(context.Background(), nil, f.d)
	if err == nil || r.CheckpointPassed || !r.CleanupComplete || len(r.PrefixPrunes) != 2 || r.PrefixPrunes[1].Recovery.Uncertain || r.PrefixPrunes[1].Recovery.PostRestart != nil {
		t.Fatalf("final restart failure hidden: %+v %v", r, err)
	}
	assertReport(t, r)
}

func TestPrefixDestructionAndFourthRollRemainScoped(t *testing.T) {
	for _, profile := range []string{ProfileHTTP, ProfileEpochs, ProfilePrune} {
		f := fixture(t)
		f.d.profile = profile
		q := &qualifier{d: f.d}
		if err := q.pruneWiderPrefix(context.Background(), 1); err == nil {
			t.Fatal("prefix deletion escaped qualification scope")
		}
		if _, err := q.create(context.Background(), "roll-4"); err == nil || len(f.calls) != 0 {
			t.Fatal("fourth roll accepted in old profile")
		}
	}
}

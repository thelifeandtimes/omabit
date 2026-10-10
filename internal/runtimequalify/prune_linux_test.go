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

// Root.Name is a display name, not a portable absolute path: nested roots in
// Go 1.25 retain the relative OpenRoot argument. Assert filesystem identity.
func requireFixtureRoot(t *testing.T, root *os.Root, expected string) {
	t.Helper()
	actual, err := root.Stat(".")
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.Stat(expected)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(actual, want) {
		t.Fatalf("operation root %q is not fixture directory %q", root.Name(), expected)
	}
}

func fixtureCheckpoint(t *testing.T, dir string) {
	t.Helper()
	path := filepath.Join(dir, "data/fake-zod/.urb/chk")
	if err := os.MkdirAll(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "image.bin"), []byte("fixture current checkpoint"), 0600); err != nil {
		t.Fatal(err)
	}
}

func pruneFixture(t *testing.T) *fakeEngine {
	f := epochQualificationFixture(t)
	f.d.profile = ProfilePrune
	execute := f.d.execute
	f.d.execute = func(ctx context.Context, a, e []string) (string, error) {
		out, err := execute(ctx, a, e)
		if err != nil || a[4] != "container" {
			return out, err
		}
		for _, c := range f.containers {
			if c.id != a[6] {
				continue
			}
			if a[5] == "start" && (c.kind == "boot" || isEpochRestartKind(c.kind) || c.kind == "post-prune" || c.kind == "post-recovery" || (isPrefixContainer(c.kind) && !isPrefixRecovery(c.kind))) {
				fixtureCheckpoint(t, filepath.Dir(a[3]))
			}
			if a[5] == "wait" && (c.kind == "epoch-recovery" || isPrefixRecovery(c.kind)) {
				if _, err := os.Lstat(filepath.Join(filepath.Dir(a[3]), "data/fake-zod/.urb/chk/image.bin")); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("replay did not start from missing checkpoint")
				}
				fixtureCheckpoint(t, filepath.Dir(a[3]))
			}
		}
		return out, err
	}
	return f
}

func TestPruneQualificationExactDeletionAndCheckpointRecovery(t *testing.T) {
	f := pruneFixture(t)
	var removed []string
	f.d.remove = func(root *os.Root, name string) error {
		path := root.Name()
		var stored Result
		// All roots are underneath the one new qualification run.
		dir := filepath.Join(f.d.home, ".local/share/urbit-runner/qualification", strings.Repeat("a", 32))
		data, err := os.ReadFile(filepath.Join(dir, "report.json"))
		if err != nil || json.Unmarshal(data, &stored) != nil {
			t.Fatal("missing destructive intent")
		}
		if filepath.Base(path) == "chk" {
			if stored.Recovery == nil || !stored.Recovery.Uncertain || name != "image.bin" {
				t.Fatal("checkpoint removed without durable scoped intent")
			}
		} else if stored.Prune == nil || !stored.Prune.Uncertain || stored.Prune.Cutoff != "0i0" || len(stored.Prune.Files) != 4 {
			t.Fatal("epoch file removed without exact durable manifest")
		}
		removed = append(removed, filepath.Join(filepath.Base(path), name))
		return root.Remove(name)
	}
	execute := f.d.execute
	f.d.execute = func(ctx context.Context, a, e []string) (string, error) {
		if a[4] == "container" && a[5] == "create" && strings.HasSuffix(a[7], "-epoch-recovery") {
			if !reflect.DeepEqual(a[len(a)-5:], []string{"play", "--full", "--loom", "31", "/urbit/fake-zod"}) {
				t.Fatalf("unqualified replay command: %v", a)
			}
		}
		return execute(ctx, a, e)
	}
	r, err := run(context.Background(), nil, f.d)
	if err != nil || !r.CheckpointPassed || !r.CleanupComplete || r.RuntimeQualified || len(r.Containers) != 11 || r.Prune == nil || r.Prune.Uncertain || r.Prune.PostRestart == nil || r.Recovery == nil || r.Recovery.Uncertain || !r.Recovery.CheckpointRemoved || r.Recovery.PostRestart == nil {
		t.Fatalf("prune/recovery failed: %+v %v", r, err)
	}
	want := []string{"0i0/data.mdb", "0i0/epoc.txt", "0i0/lock.mdb", "0i0/vere.txt", "log/0i0", "chk/image.bin"}
	if !reflect.DeepEqual(removed, want) {
		t.Fatalf("wrong deletion set: %v", removed)
	}
	if !sameRetainedEntries(r.Prune.Before, r.Prune.After) {
		t.Fatal("retained epochs changed during unlink")
	}
	entries, err := os.ReadDir(filepath.Join(r.Directory, "data/fake-zod/.urb/log"))
	if err != nil || len(entries) != 3 || entries[0].Name() != "0i100" {
		t.Fatal("three-epoch suffix not retained")
	}
	assertReport(t, r)
	q := &qualifier{d: f.d, base: filepath.Dir(r.Directory), uid: f.i.uid, gid: f.i.gid}
	if err := q.priorReports(context.Background()); err != nil {
		t.Fatalf("completed prune history rejected: %v", err)
	}
}

func TestPruneFailuresPreserveUncertaintyAndStopProgress(t *testing.T) {
	for _, fault := range []string{"partial unlink", "unexpected file", "changed selected file", "changed retained file", "intent persistence", "completion persistence"} {
		t.Run(fault, func(t *testing.T) {
			f := pruneFixture(t)
			once := false
			f.d.persist = func(r Result) error {
				if p := r.Prune; p != nil && !once {
					if fault == "intent persistence" {
						once = true
						return errors.New("intent journal failed")
					}
					if fault == "completion persistence" && !p.Uncertain {
						once = true
						return errors.New("completion journal failed")
					}
					if fault == "changed selected file" {
						once = true
						if err := os.WriteFile(filepath.Join(r.Directory, "data/fake-zod/.urb/log/0i0/data.mdb"), []byte("changed after intent"), 0600); err != nil {
							t.Fatal(err)
						}
					}
				}
				return (&qualifier{r: r}).save()
			}
			calls := 0
			f.d.remove = func(root *os.Root, name string) error {
				calls++
				if fault == "partial unlink" && calls == 2 {
					return errors.New("unlink failed")
				}
				err := root.Remove(name)
				if calls == 1 && (fault == "unexpected file" || fault == "changed retained file") {
					log := filepath.Join(f.d.home, ".local/share/urbit-runner/qualification", strings.Repeat("a", 32), "data/fake-zod/.urb/log")
					requireFixtureRoot(t, root, filepath.Join(log, "0i0"))
					path := filepath.Join(log, "0i0/unexpected")
					if fault == "changed retained file" {
						path = filepath.Join(log, "0i100/data.mdb")
					}
					if err := os.WriteFile(path, []byte("injected filesystem change"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				return err
			}
			r, err := run(context.Background(), nil, f.d)
			if err == nil || r.CheckpointPassed || !r.CleanupComplete || r.Prune == nil || !r.Prune.Uncertain || r.Prune.FinishedAt != nil || r.Recovery != nil {
				t.Fatalf("uncertain deletion continued: %+v %v", r, err)
			}
			if (fault == "intent persistence" || fault == "changed selected file") && calls != 0 {
				t.Fatal("removed a file before validating/persisting intent")
			}
			for _, c := range r.Containers {
				if c.Kind == "post-prune" {
					t.Fatal("restarted after ambiguous deletion")
				}
			}
			assertReport(t, r)
			before := len(f.calls)
			f.d.randomID = func() (string, error) { return strings.Repeat("b", 32), nil }
			next, err := run(context.Background(), nil, f.d)
			if err == nil || next.RunID != "" || len(f.calls) != before {
				t.Fatal("partial deletion was automatically retried")
			}
		})
	}
}

func TestPruneRefusesStaleInventoryBeforeMutation(t *testing.T) {
	f := pruneFixture(t)
	unlinks := 0
	f.d.remove = func(root *os.Root, name string) error { unlinks++; return root.Remove(name) }
	r, err := run(context.Background(), func(phase string) {
		if phase == "prune-prefix" {
			path := filepath.Join(f.d.home, ".local/share/urbit-runner/qualification", strings.Repeat("a", 32), "data/fake-zod/.urb/log/0i0/data.mdb")
			if err := os.WriteFile(path, []byte("changed after preview"), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}, f.d)
	if err == nil || r.Prune != nil || unlinks != 0 || !r.CleanupComplete {
		t.Fatalf("stale preview deleted data: %+v %v", r, err)
	}
}

func TestRecoveryFailureCannotBootOrClearQuarantine(t *testing.T) {
	for _, fault := range []string{"interrupted replay", "missing checkpoint", "completion persistence", "post-recovery HTTP"} {
		t.Run(fault, func(t *testing.T) {
			f := pruneFixture(t)
			execute := f.d.execute
			f.d.execute = func(ctx context.Context, a, e []string) (string, error) {
				recovery := false
				if a[4] == "container" && a[5] == "wait" {
					for _, c := range f.containers {
						recovery = recovery || (c.id == a[6] && c.kind == "epoch-recovery")
					}
				}
				if recovery && fault == "interrupted replay" {
					return "", context.DeadlineExceeded
				}
				out, err := execute(ctx, a, e)
				if recovery && fault == "missing checkpoint" {
					if err := os.Remove(filepath.Join(filepath.Dir(a[3]), "data/fake-zod/.urb/chk/image.bin")); err != nil {
						t.Fatal(err)
					}
				}
				if recovery && fault == "post-recovery HTTP" {
					f.http = "503"
				}
				return out, err
			}
			if fault == "completion persistence" {
				f.d.persist = func(r Result) error {
					if r.Recovery != nil && !r.Recovery.Uncertain {
						return errors.New("journal failure")
					}
					return (&qualifier{r: r}).save()
				}
			}
			r, err := run(context.Background(), nil, f.d)
			if err == nil || r.CheckpointPassed || !r.CleanupComplete || r.Recovery == nil || !r.Recovery.CheckpointRemoved || r.Recovery.PostRestart != nil {
				t.Fatalf("recovery failure accepted: %+v %v", r, err)
			}
			if fault != "post-recovery HTTP" {
				if !r.Recovery.Uncertain {
					t.Fatal("recovery uncertainty cleared")
				}
				for _, c := range r.Containers {
					if c.Kind == "post-recovery" {
						t.Fatal("booted after uncertain recovery")
					}
				}
				q := &qualifier{d: f.d, base: filepath.Dir(r.Directory), uid: f.i.uid, gid: f.i.gid}
				if err := q.priorReports(context.Background()); err == nil {
					t.Fatal("recovery quarantine ignored")
				}
			}
			assertReport(t, r)
		})
	}
}

func TestPruneHelpersRejectManagedAndUnscopedPaths(t *testing.T) {
	for _, profile := range []string{ProfileLifecycle, ProfileMeld, ProfileEpochs, ProfileHTTP} {
		q := &qualifier{d: dependencies{profile: profile}}
		if err := q.prunePrefix(context.Background()); err == nil {
			t.Fatal("destructive helper accepted another profile")
		}
	}
	q := &qualifier{d: dependencies{profile: ProfilePrune, managed: true}}
	if err := q.prunePrefix(context.Background()); err == nil {
		t.Fatal("managed prune enabled")
	}
	q = &qualifier{d: dependencies{profile: ProfilePrune}, r: Result{Profile: ProfilePrune, Directory: "/existing/pier", LimitsObserved: true}}
	if err := q.recoverEpoch(context.Background()); err == nil {
		t.Fatal("recovery accepted an arbitrary path")
	}
}

func TestPruneHistoryRequiresExplicitCompleteEvidence(t *testing.T) {
	f := pruneFixture(t)
	r, err := run(context.Background(), nil, f.d)
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(r.ReportPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, fault := range []string{"missing prune uncertainty", "uncertain prune", "incomplete manifest", "changed retained digest", "missing recovery uncertainty", "checkpoint not removed", "missing recovery result", "wrong profile", "missing replay container"} {
		t.Run(fault, func(t *testing.T) {
			var doc map[string]any
			if err := json.Unmarshal(original, &doc); err != nil {
				t.Fatal(err)
			}
			p, recovery := mapAt(doc, "prune"), mapAt(doc, "epoch_recovery")
			switch fault {
			case "missing prune uncertainty":
				delete(p, "uncertain")
			case "uncertain prune":
				p["uncertain"] = true
			case "incomplete manifest":
				p["removed_files"] = []string{}
			case "changed retained digest":
				mapAt(p, "after")["entries"].([]any)[0].(map[string]any)["digest"] = strings.Repeat("0", 64)
			case "missing recovery uncertainty":
				delete(recovery, "uncertain")
			case "checkpoint not removed":
				recovery["checkpoint_removed"] = false
			case "missing recovery result":
				delete(recovery, "after")
			case "wrong profile":
				doc["profile"] = ProfileEpochs
			case "missing replay container":
				var records []any
				for _, c := range doc["containers"].([]any) {
					if c.(map[string]any)["kind"] != "epoch-recovery" {
						records = append(records, c)
					}
				}
				doc["containers"] = records
			}
			if err := os.WriteFile(r.ReportPath, []byte(encode(t, doc)), 0600); err != nil {
				t.Fatal(err)
			}
			q := &qualifier{d: f.d, base: filepath.Dir(r.Directory), uid: f.i.uid, gid: f.i.gid}
			if err := q.priorReports(context.Background()); err == nil {
				t.Fatal("malformed destructive qualification history accepted")
			}
		})
	}
}

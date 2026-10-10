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
	"syscall"
	"testing"
)

func meldFixture(t *testing.T) *fakeEngine {
	f := fixture(t)
	f.d.profile = ProfileMeld
	return f
}

func TestMeldQualificationSequenceAndEvidence(t *testing.T) {
	f := meldFixture(t)
	execute := f.d.execute
	dispatches := 0
	f.d.execute = func(ctx context.Context, a, e []string) (string, error) {
		out, err := execute(ctx, a, e)
		if len(f.events) == 0 {
			return out, err
		}
		last := f.events[len(f.events)-1]
		path := filepath.Join(filepath.Dir(a[3]), "data/fake-zod/.urb/snapshot")
		if a[5] == "start" && last == "start-boot" {
			if err := os.WriteFile(path, make([]byte, 4096), 0600); err != nil {
				t.Fatal(err)
			}
		}
		if a[5] == "create" && last == "create-meld" {
			dispatches++
			data, err := os.ReadFile(filepath.Join(filepath.Dir(a[3]), "report.json"))
			var stored Result
			if err != nil || json.Unmarshal(data, &stored) != nil || stored.Meld == nil || !stored.Meld.Uncertain || stored.Meld.FinishedAt != nil {
				t.Fatal("utility dispatched before durable uncertainty")
			}
		}
		if a[5] == "wait" {
			if err := os.WriteFile(path, make([]byte, 1024), 0600); err != nil {
				t.Fatal(err)
			}
		}
		return out, err
	}
	var phases []string
	r, err := run(context.Background(), func(s string) { phases = append(phases, s) }, f.d)
	if err != nil || !r.CheckpointPassed || !r.CleanupComplete || r.RuntimeQualified || r.Network != nil || len(r.Containers) != 4 || dispatches != 1 {
		t.Fatalf("incomplete meld qualification: %+v %v", r, err)
	}
	if !reflect.DeepEqual(phases, []string{"engine", "image", "pill", "probe", "boot", "meld", "post-meld", "cleanup"}) {
		t.Fatalf("unexpected phases: %v", phases)
	}
	m := r.Meld
	if m == nil || m.Uncertain || m.StartedAt.IsZero() || m.FinishedAt == nil || m.FinishedAt.Before(m.StartedAt) || m.BeforePierBytes != 4096 || m.AfterPierBytes == nil || *m.AfterPierBytes != 1024 {
		t.Fatalf("missing measurement evidence: %+v", m)
	}
	for _, c := range r.Containers {
		if !c.Removed || c.CreateUncertain || ((c.Kind == "boot" || c.Kind == "post-meld") && !c.PrivateHTTPReachable) {
			t.Fatalf("missing cleanup/liveness: %+v", c)
		}
	}
	for _, call := range f.calls {
		if len(call) < 4 || call[1] != "create" {
			continue
		}
		if strings.HasSuffix(call[3], "-meld") && !strings.HasSuffix(call[3], "-post-meld") {
			if !reflect.DeepEqual(call[len(call)-4:], []string{"meld", "--loom", "31", "/urbit/fake-zod"}) {
				t.Fatalf("wrong utility command: %q", call)
			}
		}
		if strings.HasSuffix(call[3], "-post-meld") {
			for _, arg := range call {
				if arg == "-c" || arg == "-F" || arg == "-B" || strings.Contains(arg, "boot.pill") {
					t.Fatal("post-meld restart reinitialized or mounted the pill")
				}
			}
		}
	}
	assertReport(t, r)
	// Completed new reports coexist with accepted lifecycle/HTTP history.
	for i, profile := range []string{ProfileLifecycle, ProfileHTTP, ProfileMeld} {
		q := &qualifier{d: f.d, base: filepath.Dir(r.Directory), uid: f.i.uid, gid: f.i.gid}
		q.d.profile = profile
		if err := q.priorReports(context.Background()); err != nil {
			t.Fatalf("history compatibility %d: %v", i, err)
		}
	}
}

func TestMeldFreshAdmissionBeforeDispatch(t *testing.T) {
	for _, fault := range []string{"RAM", "disk", "unavailable"} {
		t.Run(fault, func(t *testing.T) {
			f := meldFixture(t)
			checks := 0
			f.d.admission = func(string) (uint64, uint64, error) {
				checks++
				if checks == 1 {
					return 5 << 30, 8 << 30, nil
				}
				switch fault {
				case "RAM":
					return 5<<30 - 1, 8 << 30, nil
				case "disk":
					return 5 << 30, 8<<30 - 1, nil
				default:
					return 0, 0, errors.New("unavailable")
				}
			}
			r, err := run(context.Background(), nil, f.d)
			if err == nil || !strings.Contains(err.Error(), "meld admission") || checks != 2 || r.Meld != nil || len(r.Containers) != 2 || !r.CleanupComplete {
				t.Fatalf("stale admission allowed meld: %+v %v", r, err)
			}
			assertReport(t, r)
		})
	}
}

func TestMeldFailuresRetainQuarantineAfterCleanup(t *testing.T) {
	for _, fault := range []string{"wait timeout", "wait nonzero", "exit nonzero", "OOM", "still running", "start cancellation", "missing pier", "final persistence"} {
		t.Run(fault, func(t *testing.T) {
			f := meldFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			execute := f.d.execute
			f.d.execute = func(ctx context.Context, a, e []string) (string, error) {
				if a[5] == "wait" && fault == "wait timeout" {
					return "", context.DeadlineExceeded
				}
				out, err := execute(ctx, a, e)
				if a[5] == "start" && fault == "start cancellation" {
					for _, c := range f.containers {
						if c.kind == "meld" {
							cancel()
							return "", context.Canceled
						}
					}
				}
				if a[5] == "wait" {
					for _, c := range f.containers {
						switch fault {
						case "exit nonzero":
							mapAt(c.doc, "State")["ExitCode"] = 1
						case "OOM":
							mapAt(c.doc, "State")["OOMKilled"] = true
						case "still running":
							mapAt(c.doc, "State")["Running"] = true
						}
					}
					if fault == "wait nonzero" {
						return "1\n", nil
					}
					if fault == "missing pier" {
						path := filepath.Join(filepath.Dir(a[3]), "data/fake-zod/.urb")
						if err := os.Rename(path, path+"-moved"); err != nil {
							t.Fatal(err)
						}
					}
				}
				return out, err
			}
			if fault == "final persistence" {
				f.d.persist = func(r Result) error {
					if r.Meld != nil && !r.Meld.Uncertain {
						return errors.New("injected final persistence failure")
					}
					return (&qualifier{r: r}).save()
				}
			}
			r, err := run(ctx, nil, f.d)
			if err == nil || r.CheckpointPassed || !r.CleanupComplete || r.Meld == nil || !r.Meld.Uncertain || r.Meld.FinishedAt != nil || len(f.containers) != 0 {
				t.Fatalf("lost quarantine: %+v %v", r, err)
			}
			count := 0
			for _, event := range f.events {
				if event == "create-meld" {
					count++
				}
				if event == "create-post-meld" {
					t.Fatal("restarted a pier after an uncertain meld")
				}
			}
			if count != 1 {
				t.Fatal("meld dispatched more than once")
			}
			assertReport(t, r)
			// A subsequent process cannot mistake cleanup for successful maintenance.
			calls := len(f.calls)
			f.d.randomID = func() (string, error) { return strings.Repeat("b", 32), nil }
			next, err := run(context.Background(), nil, f.d)
			if err == nil || !strings.Contains(err.Error(), "manual resolution") || next.RunID != "" || len(f.calls) != calls {
				t.Fatalf("new run ignored quarantined pier: %+v %v", next, err)
			}
		})
	}
}

func TestMeldManagementLockExclusion(t *testing.T) {
	t.Run("meld", func(t *testing.T) { testQualificationManagementLock(t, meldFixture(t)) })
	t.Run("epochs", func(t *testing.T) { testQualificationManagementLock(t, epochQualificationFixture(t)) })
	t.Run("prune", func(t *testing.T) { testQualificationManagementLock(t, pruneFixture(t)) })
	t.Run("prefix", func(t *testing.T) { testQualificationManagementLock(t, prefixFixture(t)) })
}

func testQualificationManagementLock(t *testing.T, f *fakeEngine) {
	q := &qualifier{d: f.d}
	if _, _, err := q.prevalidate(); err != nil {
		t.Fatal(err)
	}
	q.base = filepath.Join(f.d.home, ".local/share/urbit-runner/managed")
	unlock, err := q.lockStorage()
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	r, err := run(context.Background(), nil, f.d)
	if err == nil || !strings.Contains(err.Error(), "management stopped") || r.RunID != "" || len(f.calls) != 0 {
		t.Fatalf("active management was ignored: %+v %v", r, err)
	}
	unlock()
	lock, err := os.OpenFile(filepath.Join(q.base, ".lock"), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	checked := 0
	r, err = run(context.Background(), func(string) {
		checked++
		if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err == nil {
			t.Fatal("management lock not held throughout qualification/cleanup")
		}
	}, f.d)
	if err != nil || !r.CheckpointPassed || checked < 8 {
		t.Fatalf("exclusive qualification failed: %+v %v", r, err)
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal("management lock leaked")
	}
	_ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
}

func TestMeldPostRestartFailureCannotPass(t *testing.T) {
	f := meldFixture(t)
	execute := f.d.execute
	f.d.execute = func(ctx context.Context, a, e []string) (string, error) {
		out, err := execute(ctx, a, e)
		if a[5] == "wait" {
			f.http = "503"
		}
		return out, err
	}
	r, err := run(context.Background(), nil, f.d)
	if err == nil || r.CheckpointPassed || !r.CleanupComplete || r.Meld == nil || r.Meld.Uncertain || len(r.Containers) != 4 {
		t.Fatalf("utility success hid a failed restart: %+v %v", r, err)
	}
	assertReport(t, r)
}

func TestMeldHistoryRequiresExplicitResolvedEvidence(t *testing.T) {
	f := meldFixture(t)
	r, err := run(context.Background(), nil, f.d)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(r.ReportPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, fault := range []string{"missing meld", "missing uncertainty", "null uncertainty", "uncertain", "missing finish", "missing before", "missing after", "negative after", "low admission", "wrong profile", "network", "missing utility"} {
		t.Run(fault, func(t *testing.T) {
			var doc map[string]any
			if err := json.Unmarshal(data, &doc); err != nil {
				t.Fatal(err)
			}
			m := mapAt(doc, "meld")
			switch fault {
			case "missing meld":
				delete(doc, "meld")
			case "missing uncertainty":
				delete(m, "uncertain")
			case "null uncertainty":
				m["uncertain"] = nil
			case "uncertain":
				m["uncertain"] = true
			case "missing finish":
				delete(m, "finished_at")
			case "missing before":
				delete(m, "before_pier_bytes")
			case "missing after":
				delete(m, "after_pier_bytes")
			case "negative after":
				m["after_pier_bytes"] = -1
			case "low admission":
				m["mem_available_bytes"] = 1
			case "wrong profile":
				doc["profile"] = ProfileLifecycle
			case "network":
				doc["network"] = map[string]any{}
			case "missing utility":
				doc["containers"] = []any{}
			}
			if err := os.WriteFile(r.ReportPath, []byte(encode(t, doc)), 0600); err != nil {
				t.Fatal(err)
			}
			q := &qualifier{d: f.d, base: filepath.Dir(r.Directory), uid: f.i.uid, gid: f.i.gid}
			if err := q.priorReports(context.Background()); err == nil {
				t.Fatalf("malformed meld outcome accepted: %s", fault)
			}
		})
	}
}

func TestMeldQualifierRequiresDisposableProfile(t *testing.T) {
	for _, profile := range []string{ProfileLifecycle, ProfileHTTP} {
		f := fixture(t)
		f.d.profile = profile
		q := &qualifier{d: f.d}
		if err := q.meld(context.Background()); err == nil {
			t.Fatalf("meld admitted in %s", profile)
		}
		if _, err := q.create(context.Background(), "meld"); err == nil || len(f.calls) != 0 {
			t.Fatalf("meld container admitted in %s", profile)
		}
	}
	f := meldFixture(t)
	f.d.managed = true
	q := &qualifier{d: f.d}
	if err := q.meld(context.Background()); err == nil {
		t.Fatal("disposable meld workflow admitted a managed session")
	}
}

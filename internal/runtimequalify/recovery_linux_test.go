//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMissingStateIsNotEvidenceOfShutdown(t *testing.T) {
	for _, field := range []string{"State", "Running", "ExitCode", "OOMKilled"} {
		t.Run(field, func(t *testing.T) {
			f := fixture(t)
			f.editInspect = func(_ string, doc map[string]any) {
				if field == "State" {
					delete(doc, field)
				} else {
					delete(mapAt(doc, "State"), field)
				}
			}
			r, err := run(context.Background(), nil, f.d)
			if err == nil || r.CheckpointPassed || r.CleanupComplete || len(f.containers) != 1 {
				t.Fatalf("missing evidence treated as shutdown: %+v %v", r, err)
			}
			assertReport(t, r)
		})
	}
}

func TestCanceledAcceptedBootStartStopsDaemonJob(t *testing.T) {
	f := fixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	execute := f.d.execute
	f.d.execute = func(c context.Context, a, e []string) (string, error) {
		out, err := execute(c, a, e)
		if len(a) > 5 && a[5] == "start" && strings.HasSuffix(f.events[len(f.events)-1], "-boot") {
			cancel()
			return "", context.Canceled
		}
		return out, err
	}
	r, err := run(ctx, nil, f.d)
	if err == nil || !r.CleanupComplete || len(f.containers) != 0 {
		t.Fatalf("canceled live boot leaked: %+v %v", r, err)
	}
	events := strings.Join(f.events, ",")
	if !strings.Contains(events, "start-boot,stop-boot,rm-boot") {
		t.Fatalf("cleanup never contacted the daemon: %s", events)
	}
	assertReport(t, r)
}

func TestCleanupDeadlineRetainsExactUnresolvedName(t *testing.T) {
	f := fixture(t)
	execute := f.d.execute
	lost := false
	f.d.times.cleanup = 30 * time.Millisecond
	f.d.execute = func(ctx context.Context, a, e []string) (string, error) {
		if lost {
			<-ctx.Done()
			return "", ctx.Err()
		}
		out, err := execute(ctx, a, e)
		if len(a) > 5 && a[5] == "create" {
			lost = true
			return "", errors.New("daemon unavailable after acceptance")
		}
		return out, err
	}
	r, err := run(context.Background(), nil, f.d)
	if err == nil || r.CleanupComplete || r.CheckpointPassed || len(r.Containers) != 1 || r.Containers[0].Removed {
		t.Fatalf("uncertain cleanup lost its record: %+v %v", r, err)
	}
	if !strings.Contains(err.Error(), r.Containers[0].Name) {
		t.Fatal("missing exact manual-review name")
	}
	assertReport(t, r)
}

func TestPackExitEvidenceAndPostPackFailure(t *testing.T) {
	for _, fault := range []string{"nonzero", "OOM", "still running", "post-pack HTTP"} {
		t.Run(fault, func(t *testing.T) {
			f := fixture(t)
			execute := f.d.execute
			f.d.execute = func(ctx context.Context, a, e []string) (string, error) {
				out, err := execute(ctx, a, e)
				if len(a) > 5 && a[5] == "wait" {
					for _, c := range f.containers {
						state := mapAt(c.doc, "State")
						switch fault {
						case "nonzero":
							state["ExitCode"] = 1 // even if docker wait said zero
						case "OOM":
							state["OOMKilled"] = true
						case "still running":
							state["Running"] = true
						case "post-pack HTTP":
							f.http = "401"
						}
					}
				}
				return out, err
			}
			r, err := run(context.Background(), nil, f.d)
			if err == nil || r.CheckpointPassed || !r.CleanupComplete {
				t.Fatalf("pack failure accepted: %+v %v", r, err)
			}
			assertReport(t, r)
		})
	}
}

func TestFreshPierAndPillPathsCannotBeReused(t *testing.T) {
	for _, artifact := range []string{"pill", "pier"} {
		t.Run(artifact, func(t *testing.T) {
			f := fixture(t)
			execute := f.d.execute
			f.d.execute = func(ctx context.Context, a, e []string) (string, error) {
				out, err := execute(ctx, a, e)
				dir := filepath.Dir(a[3])
				if artifact == "pill" && len(a) > 5 && a[4] == "image" && a[5] == "inspect" {
					if err := os.WriteFile(filepath.Join(dir, "urbit-v4.6.pill"), []byte("existing"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				if artifact == "pier" && len(a) > 5 && a[5] == "rm" {
					if err := os.Mkdir(filepath.Join(dir, "data/fake-zod"), 0700); err != nil {
						t.Fatal(err)
					}
				}
				return out, err
			}
			r, err := run(context.Background(), nil, f.d)
			if err == nil || r.CheckpointPassed {
				t.Fatal("existing artifact overwritten or adopted")
			}
			if artifact == "pill" && f.downloads != 0 {
				t.Fatal("download attempted over existing pill")
			}
			for _, event := range f.events {
				if event == "create-boot" {
					t.Fatal("boot attempted with existing data")
				}
			}
			assertReport(t, r)
		})
	}
}

//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func cometMaintenanceFixture(t *testing.T) *fakeEngine {
	f := cometFixture(t)
	f.d.profile = ProfileCometMaintenance
	execute := f.d.execute
	f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
		out, err := execute(ctx, args, env)
		if err == nil && len(args) > 6 && args[4] == "container" && args[5] == "start" {
			for _, c := range f.containers {
				if c.id != args[6] {
					continue
				}
				file := filepath.Join(filepath.Dir(args[3]), "data/comet/.urb/state")
				size := -1
				switch c.kind {
				case "comet-boot":
					size = 4096
				case "comet-pack":
					size = 2048
				case "comet-meld":
					size = 1024
				}
				if size >= 0 {
					if err := os.WriteFile(file, []byte(strings.Repeat("x", size)), 0600); err != nil {
						t.Fatal(err)
					}
				}
				if c.kind == "comet-pack" || c.kind == "comet-meld" {
					if mapAt(c.doc, "HostConfig")["NetworkMode"] != "none" {
						t.Fatal("utility used network")
					}
				}
			}
		}
		return out, err
	}
	return f
}

func TestCometMaintenanceBothUtilitiesAndIdentityRestarts(t *testing.T) {
	f := cometMaintenanceFixture(t)
	r, err := run(context.Background(), nil, f.d)
	if err != nil || !r.CheckpointPassed || !r.CleanupComplete || len(r.Containers) != 7 || len(f.containers) != 0 || len(f.networks) != 0 || len(r.CometUtilities) != 2 || r.Comet.Ship != fixtureComet {
		t.Fatalf("maintenance qualification: %+v %v", r, err)
	}
	for i, m := range r.CometUtilities {
		if m.Uncertain || !m.RestartVerified || m.FinishedAt == nil || m.AfterPierBytes == nil || m.BeforePierBytes <= *m.AfterPierBytes || m.Kind != []string{"comet-pack", "comet-meld"}[i] {
			t.Fatal("missing utility evidence", m)
		}
	}
	q := &qualifier{d: f.d, base: filepath.Dir(r.Directory), uid: f.i.uid, gid: f.i.gid}
	if err := q.priorReport(r.RunID); err != nil {
		t.Fatal("completed report refused", err)
	}
	assertReport(t, r)
}

func TestCometUtilityFailureNeverContinuesAndCleanupKeepsQuarantine(t *testing.T) {
	for _, failed := range []string{"comet-pack", "comet-meld"} {
		t.Run(failed, func(t *testing.T) {
			f := cometMaintenanceFixture(t)
			execute := f.d.execute
			f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
				if len(args) > 6 && args[4] == "container" && args[5] == "wait" {
					for _, c := range f.containers {
						if c.id == args[6] && c.kind == failed {
							return "", errors.New("utility interrupted")
						}
					}
				}
				return execute(ctx, args, env)
			}
			r, err := run(context.Background(), nil, f.d)
			if err == nil || r.CheckpointPassed || !r.CleanupComplete || !r.Comet.Uncertain || len(r.CometUtilities) == 0 || !r.CometUtilities[len(r.CometUtilities)-1].Uncertain {
				t.Fatal("uncertain utility cleared", err)
			}
			for _, c := range r.Containers {
				if c.Kind == "comet-post-"+strings.TrimPrefix(failed, "comet-") {
					t.Fatal("restarted after failed utility")
				}
			}
			q := &qualifier{d: f.d, base: filepath.Dir(r.Directory), uid: f.i.uid, gid: f.i.gid}
			if err := q.priorReport(r.RunID); err == nil {
				t.Fatal("uncertain maintenance history trusted")
			}
		})
	}
}

func TestCometMaintenanceHistoryRejectsMissingEvidence(t *testing.T) {
	for _, field := range []string{"uncertain", "restart_verified", "after_pier_bytes", "before_pier_bytes", "finished_at"} {
		t.Run(field, func(t *testing.T) {
			f := cometMaintenanceFixture(t)
			r, err := run(context.Background(), nil, f.d)
			if err != nil {
				t.Fatal(err)
			}
			var data map[string]any
			raw, _ := os.ReadFile(r.ReportPath)
			json.Unmarshal(raw, &data)
			delete(data["comet_utilities"].([]any)[0].(map[string]any), field)
			raw, _ = json.Marshal(data)
			if err := os.WriteFile(r.ReportPath, raw, 0600); err != nil {
				t.Fatal(err)
			}
			q := &qualifier{d: f.d, base: filepath.Dir(r.Directory), uid: f.i.uid, gid: f.i.gid}
			if err := q.priorReport(r.RunID); err == nil {
				t.Fatal("missing evidence admitted", field)
			}
		})
	}
}

func TestCometUtilityCommandsStayPinnedAndProfileBound(t *testing.T) {
	for _, kind := range []string{"comet-pack", "comet-meld"} {
		entry, args := command(kind)
		if entry != "/bin/urbit" || len(args) != 4 || args[0] != strings.TrimPrefix(kind, "comet-") || args[3] != "/urbit/comet" {
			t.Fatal("wrong comet utility command")
		}
	}
	f, state, persist := managedCometFixture(t)
	for _, kind := range []string{"comet-pack", "comet-meld"} {
		if _, _, err := applyManaged(context.Background(), kind, state, persist, f.d); err == nil {
			t.Fatal("qualification enabled managed utility")
		}
	}
}

func TestCometMaintenanceAdmissionAndPostUtilityIdentity(t *testing.T) {
	for _, fault := range []string{"admission", "identity"} {
		t.Run(fault, func(t *testing.T) {
			f := cometMaintenanceFixture(t)
			if fault == "admission" {
				f.d.admission = func(string) (uint64, uint64, error) {
					for _, e := range f.events {
						if e == "rm-comet-boot" {
							return 0, 8 << 30, nil
						}
					}
					return 5 << 30, 8 << 30, nil
				}
			} else {
				read := f.d.hostIdentity
				f.d.hostIdentity = func(ctx context.Context, endpoint string) (string, error) {
					if strings.Contains(endpoint, "38103") {
						return "~marzod-salbud-salwex-sogref--tasweb-navfun-batwet-sigfet", nil
					}
					return read(ctx, endpoint)
				}
			}
			r, err := run(context.Background(), nil, f.d)
			if err == nil || r.CheckpointPassed || !r.CleanupComplete || r.Comet == nil || !r.Comet.Uncertain {
				t.Fatal("unsafe maintenance accepted", err)
			}
			for _, c := range r.Containers {
				if c.Kind == "comet-meld" || fault == "admission" && c.Kind == "comet-pack" {
					t.Fatal("continued beyond failed boundary")
				}
			}
		})
	}
}

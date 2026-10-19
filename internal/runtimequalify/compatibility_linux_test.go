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
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/routines"
)

func setFixtureEngineVersion(f *fakeEngine, version string) {
	f.engine.Version = version
	f.hostReport.InstalledEngineVersion = version
}

func TestCompatibilityActualAPIAndStableFamilies(t *testing.T) {
	for _, version := range []string{"27.2.0", "27.2.0+dfsg1", "27.5.1", "28.0.0", "29.7.1", "29.7.2", "29.9.1", "29.9.1+dfsg1"} {
		t.Run(version, func(t *testing.T) {
			f := fixture(t)
			setFixtureEngineVersion(f, version)
			q := &qualifier{d: f.d}
			if err := q.verifyEngine(context.Background()); err != nil {
				t.Fatal(err)
			}
			r := f.hostReport
			r.Engine = &f.engine
			r = hostcompat.Evaluate(r, f.engine.DataRoot)
			if !r.CanMutate || r.Qualification != "unqualified" || q.engineEvidence.Version != version || len(f.calls) != 0 {
				t.Fatalf("compatibility was confused with qualification/package version: %+v", r)
			}
		})
	}
	for name, edit := range map[string]func(*fakeEngine){
		"old family":            func(f *fakeEngine) { setFixtureEngineVersion(f, "26.1.5") },
		"pre API family":        func(f *fakeEngine) { setFixtureEngineVersion(f, "27.1.2") },
		"new family":            func(f *fakeEngine) { setFixtureEngineVersion(f, "30.0.0") },
		"prerelease":            func(f *fakeEngine) { setFixtureEngineVersion(f, "29.9.1-rc1") },
		"malformed":             func(f *fakeEngine) { setFixtureEngineVersion(f, "29.9.1;private") },
		"API max too old":       func(f *fakeEngine) { f.engine.APIVersion = "1.46" },
		"API min too new":       func(f *fakeEngine) { f.engine.MinAPIVersion = "1.48" },
		"API min missing":       func(f *fakeEngine) { f.engine.MinAPIVersion = "" },
		"rootful":               func(f *fakeEngine) { f.engine.Rootless = false },
		"cgroup v1":             func(f *fakeEngine) { f.engine.CgroupVersion = "1" },
		"wrong driver":          func(f *fakeEngine) { f.engine.CgroupDriver = "cgroupfs" },
		"wrong data root":       func(f *fakeEngine) { f.engine.DataRoot = "/var/lib/docker" },
		"metadata only":         func(f *fakeEngine) { f.hostReport.Packages[0].Provenance = "metadata-only" },
		"failed provenance":     func(f *fakeEngine) { f.hostReport.Packages[1].Provenance = "failed" },
		"incomplete stack":      func(f *fakeEngine) { f.hostReport.Packages = f.hostReport.Packages[:1] },
		"installed unsupported": func(f *fakeEngine) { f.hostReport.InstalledEngineVersion = "30.0.0" },
		"installed unknown":     func(f *fakeEngine) { f.hostReport.InstalledEngineVersion = "" },
		"probe unavailable": func(f *fakeEngine) {
			f.d.ProbeEngine = func(context.Context, string, string) (hostcompat.Engine, error) {
				return hostcompat.Engine{}, errors.New("private-token-untrusted-daemon-error")
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			f, state, _ := managedFixture(t)
			edit(f)
			persisted := false
			after, obs, err := applyManaged(context.Background(), "start", state, func(ManagedState) error { persisted = true; return nil }, f.d)
			if !errors.Is(err, ErrHostCompatibility) || obs.RecoveryRequired || persisted || len(f.calls) != 0 || !reflect.DeepEqual(state, after) {
				t.Fatalf("denial changed state or claimed damage: %+v %v", obs, err)
			}
			if strings.Contains(err.Error(), "private-token") {
				t.Fatal("untrusted diagnostic leaked")
			}
			if _, err := os.Stat(filepath.Join(f.d.home, ".local/share/urbit-runner/managed", state.ID)); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("denied work created a runtime workspace")
			}
		})
	}
}

func TestCompatibilityRestartPendingIsAdvisoryWithNormalPersistence(t *testing.T) {
	f, state, persist := managedFixture(t)
	f.hostReport.InstalledEngineVersion = "29.9.1"
	report := f.hostReport
	report.Engine = &f.engine
	report = hostcompat.Evaluate(report, f.engine.DataRoot)
	if !report.RestartPending || !report.CanMutate || report.Qualification != "unqualified" {
		t.Fatalf("compatible package/daemon drift was not advisory: %+v", report)
	}
	state, obs, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil || obs.State != "running" || obs.Health != "healthy" || obs.RecoveryRequired || !state.Initialized {
		t.Fatalf("compatible restart-pending Start failed: %+v %v", obs, err)
	}
	if state.Runtime.EngineVersion != f.engine.Version || state.Runtime.EngineVersion == f.hostReport.InstalledEngineVersion {
		t.Fatal("provisioning substituted installed-package evidence for the running daemon")
	}
	data, err := os.ReadFile(filepath.Join(state.Runtime.Directory, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var stored Result
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.EngineVersion != f.engine.Version || len(stored.Containers) != len(state.Runtime.Containers) {
		t.Fatal("normal persistence did not record running-engine evidence")
	}
	for _, record := range stored.Containers {
		if record.EngineVersion != f.engine.Version {
			t.Fatal("launch recorded installed version")
		}
	}
	state, obs, err = observeManaged(context.Background(), state, f.d)
	if err != nil || obs.State != "running" || obs.RecoveryRequired {
		t.Fatalf("advisory drift damaged observation: %+v %v", obs, err)
	}
	_, obs, err = applyManaged(context.Background(), "stop", state, persist, f.d)
	if err != nil || obs.State != "stopped" || obs.RecoveryRequired {
		t.Fatalf("advisory drift blocked Stop: %+v %v", obs, err)
	}
}

func TestCompatibilityStableBuildMetadataPreservesRawRuntimeEvidence(t *testing.T) {
	f, state, persist := managedFixture(t)
	setFixtureEngineVersion(f, "29.9.1+dfsg1")
	// Installed metadata may normalize the upstream component. It is not the
	// daemon's observed version, nor a project-specific minimum-version pin.
	f.hostReport.InstalledEngineVersion = "29.9.1"
	state, obs, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil || obs.State != "running" || obs.RecoveryRequired {
		t.Fatalf("stable native build metadata blocked runtime work: %+v %v", obs, err)
	}
	if state.Runtime.EngineVersion != f.engine.Version {
		t.Fatal("provisioning normalized away actual daemon evidence")
	}
	for _, record := range state.Runtime.Containers {
		if record.EngineVersion != f.engine.Version {
			t.Fatal("launch normalized actual engine version")
		}
	}
	// These reopen the managed session and therefore exercise recorded-version
	// validation independently of admitting the new create above.
	state, obs, err = observeManaged(context.Background(), state, f.d)
	if err != nil || obs.State != "running" || obs.RecoveryRequired {
		t.Fatalf("recorded native build rejected on observation: %+v %v", obs, err)
	}
	state, obs, err = applyManaged(context.Background(), "stop", state, persist, f.d)
	if err != nil || obs.State != "stopped" || obs.RecoveryRequired || state.Runtime.EngineVersion != f.engine.Version {
		t.Fatalf("Stop rejected or rewrote recorded native build: %+v %v", obs, err)
	}
}

func TestCompatibilityCanceledEvidenceDoesNotManufactureRecovery(t *testing.T) {
	for _, phase := range []string{"already-canceled", "during-probe", "probe-only-cancellation", "probe-deadline", "during-inspection"} {
		t.Run(phase, func(t *testing.T) {
			f, state, persist := managedFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			want := context.Canceled
			switch phase {
			case "already-canceled":
				cancel()
			case "during-probe":
				probe := f.d.ProbeEngine
				f.d.ProbeEngine = func(ctx context.Context, socket, root string) (hostcompat.Engine, error) {
					engine, err := probe(ctx, socket, root)
					cancel()
					return engine, err
				}
			case "probe-only-cancellation":
				f.d.ProbeEngine = func(context.Context, string, string) (hostcompat.Engine, error) {
					return hostcompat.Engine{}, fmt.Errorf("private-provider-text: %w", context.Canceled)
				}
			case "probe-deadline":
				want = context.DeadlineExceeded
				f.d.times.command = time.Millisecond
				f.d.ProbeEngine = func(ctx context.Context, _, _ string) (hostcompat.Engine, error) {
					<-ctx.Done()
					return hostcompat.Engine{}, ctx.Err()
				}
			case "during-inspection":
				f.d.InspectHost = func() hostcompat.Report { cancel(); return f.hostReport }
			}
			writes := 0
			after, obs, err := applyManaged(ctx, "start", state, func(s ManagedState) error { writes++; return persist(s) }, f.d)
			if !errors.Is(err, ErrHostCompatibility) || !errors.Is(err, want) || obs.RecoveryRequired || writes != 0 || len(f.calls) != 0 || !reflect.DeepEqual(state, after) {
				t.Fatalf("canceled evidence changed admission/recovery state: %+v %v", obs, err)
			}
			if strings.Contains(err.Error(), "private-provider-text") {
				t.Fatal("provider diagnostic leaked")
			}
			if !RecoveryRequiredOnCompatibility(after, true) {
				t.Fatal("prior caller recovery was cleared")
			}
		})
	}
}

func TestCompatibilityCanceledObservationPreservesOnlyExistingRecoveryEvidence(t *testing.T) {
	for _, outcome := range []string{"healthy", "uncertain-container", "uncertain-network", "known-terminal"} {
		t.Run(outcome, func(t *testing.T) {
			f, state, persist := managedFixture(t)
			state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
			if err != nil {
				t.Fatal(err)
			}
			for i := range state.Runtime.Containers {
				record := &state.Runtime.Containers[i]
				if !record.Removed {
					if outcome == "uncertain-container" {
						record.CreateUncertain = true
					}
					if outcome == "known-terminal" {
						record.LastState.Running = false
					}
				}
			}
			if outcome == "uncertain-network" {
				state.Runtime.Network.CreateUncertain = true
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			before, _ := json.Marshal(state)
			after, obs, err := observeManaged(ctx, state, f.d)
			encoded, _ := json.Marshal(after)
			if !errors.Is(err, ErrHostCompatibility) || !errors.Is(err, context.Canceled) || obs.RecoveryRequired != (outcome != "healthy") || string(before) != string(encoded) {
				t.Fatalf("cancel manufactured or erased recovery: %+v %v", obs, err)
			}
		})
	}
}

func TestCompatibilityEngineProbeNeverBypassesPrivateBoundary(t *testing.T) {
	for _, target := range []string{"socket", "client"} {
		t.Run(target, func(t *testing.T) {
			f := fixture(t)
			stat := f.d.stat
			f.d.stat = func(path string) (node, error) {
				n, err := stat(path)
				if target == "socket" && path == filepath.Join(f.d.runtimeDir, "urbit-docker.sock") || target == "client" && path == "/usr/bin/docker" {
					n.mode = os.ModeSymlink | 0777
				}
				return n, err
			}
			q := &qualifier{d: f.d}
			if err := q.verifyEngine(context.Background()); !errors.Is(err, ErrHostCompatibility) || len(f.events) != 0 || len(f.calls) != 0 {
				t.Fatalf("unsafe boundary reached a probe or executable: %v %v", err, f.events)
			}
		})
	}
}

func TestCompatibilityStoppedObservationRetainsWorkspaceOwnershipChecks(t *testing.T) {
	f, state, persist := managedFixture(t)
	state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	state, _, err = applyManaged(context.Background(), "stop", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(state.Runtime.Directory, 0777); err != nil {
		t.Fatal(err)
	}
	before := len(f.events)
	if _, _, err := observeManaged(context.Background(), state, f.d); err == nil || len(f.events) != before {
		t.Fatal("native probe bypassed prepared workspace ownership checks", err)
	}
}

func TestCompatibilityUnsupportedEngineAllowsBoundObserveAndExplicitStop(t *testing.T) {
	f, state, persist := managedFixture(t)
	state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	history := state.Runtime.EngineVersion
	setFixtureEngineVersion(f, "30.0.0")
	// A package update invalidating provenance does not strand an already-bound
	// runtime. Observation/Stop still require the actual API and all containment.
	f.hostReport.Packages[0].Provenance = "metadata-only"
	f.d.InspectHost = func() hostcompat.Report {
		t.Fatal("observation/Stop performed expensive native package inspection")
		return hostcompat.Report{}
	}
	state, obs, err := observeManaged(context.Background(), state, f.d)
	if err != nil || obs.State != "running" || obs.RecoveryRequired || state.Runtime.EngineVersion != history {
		t.Fatalf("observation stranded historical runtime: %+v %v", obs, err)
	}
	state.MaintenanceUncertain = true
	state, obs, err = applyManaged(context.Background(), "stop", state, persist, f.d)
	if err != nil || obs.State != "stopped" || !obs.RecoveryRequired || !state.MaintenanceUncertain || len(f.containers) != 0 || state.Runtime.EngineVersion != history {
		t.Fatalf("explicit Stop failed or cleared uncertainty/history: %+v %v", obs, err)
	}
}

func TestCompatibilityUnsupportedEngineRejectsUnboundOrUnsafeControls(t *testing.T) {
	for _, fault := range []string{"missing-id", "wrong-id", "wrong-label", "containment", "API", "rootful", "data-root"} {
		t.Run(fault, func(t *testing.T) {
			f, state, persist := managedFixture(t)
			state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
			if err != nil {
				t.Fatal(err)
			}
			setFixtureEngineVersion(f, "30.0.0")
			switch fault {
			case "missing-id", "wrong-id":
				for i := range state.Runtime.Containers {
					if !state.Runtime.Containers[i].Removed {
						state.Runtime.Containers[i].ID = ""
						if fault == "wrong-id" {
							state.Runtime.Containers[i].ID = strings.Repeat("f", 64)
						}
					}
				}
			case "wrong-label":
				f.editInspect = func(_ string, doc map[string]any) {
					mapAt(mapAt(doc, "Config"), "Labels")[f.runLabelName()] = "foreign"
				}
			case "containment":
				f.editInspect = func(_ string, doc map[string]any) { mapAt(doc, "HostConfig")["Privileged"] = true }
			case "API":
				f.engine.MinAPIVersion = "1.48"
			case "rootful":
				f.engine.Rootless = false
			case "data-root":
				f.engine.DataRoot = "/var/lib/docker"
			}
			before := len(f.calls)
			if _, _, err := observeManaged(context.Background(), state, f.d); err == nil {
				t.Fatal("unsafe observation admitted")
			}
			if _, _, err := applyManaged(context.Background(), "stop", state, persist, f.d); err == nil {
				t.Fatal("unsafe Stop admitted")
			}
			for _, call := range f.calls[before:] {
				if call[1] != "inspect" {
					t.Fatalf("unsafe control dispatched: %q", call)
				}
			}
			if len(f.containers) != 1 {
				t.Fatal("untrusted runtime removed")
			}
		})
	}
}

func TestCompatibilityVersionDriftPreservesHistoryAndRecordsNewLaunch(t *testing.T) {
	f, state, persist := managedFixture(t)
	state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal(state.Runtime.Containers)
	setFixtureEngineVersion(f, "29.9.1")
	state, _, err = observeManaged(context.Background(), state, f.d)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range state.Runtime.Containers {
		if c.EngineVersion != "29.7.2" {
			t.Fatal("observation relabeled launch history")
		}
	}
	if state.Runtime.EngineVersion != "29.7.2" || !strings.Contains(string(before), "29.7.2") {
		t.Fatal("lost original evidence")
	}
	state, _, err = applyManaged(context.Background(), "stop", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	state, obs, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil || obs.State != "running" || state.Runtime.EngineVersion != "29.7.2" {
		t.Fatalf("version drift rejected or relabeled history: %+v %v", obs, err)
	}
	for _, c := range state.Runtime.Containers {
		if c.EngineVersion != "29.9.1" {
			t.Fatalf("new launch used assumed version: %+v", c)
		}
	}
}

func TestCompatibilityFreshProvisioningAndMaintenanceRecordActualVersion(t *testing.T) {
	f, state, persist := managedFixture(t)
	setFixtureEngineVersion(f, "29.9.1")
	state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil || state.Runtime.EngineVersion != "29.9.1" {
		t.Fatal("fresh provisioning did not record the observed daemon", err)
	}
	state, _, err = applyManaged(context.Background(), "stop", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	for _, fault := range []string{"family", "provenance"} {
		if fault == "family" {
			setFixtureEngineVersion(f, "30.0.0")
		} else {
			setFixtureEngineVersion(f, "29.9.1")
			f.hostReport.Packages[0].Provenance = "metadata-only"
		}
		before, _ := json.Marshal(state)
		persisted := false
		after, obs, err := applyManaged(context.Background(), "pack", state, func(ManagedState) error { persisted = true; return nil }, f.d)
		encoded, _ := json.Marshal(after)
		if !errors.Is(err, ErrHostCompatibility) || obs.RecoveryRequired || persisted || string(before) != string(encoded) {
			t.Fatalf("denied maintenance changed pier/history: %s %+v %v", fault, obs, err)
		}
	}
	f.hostReport.Packages[0].Provenance = "verified"
	state, obs, err := applyManaged(context.Background(), "pack", state, persist, f.d)
	if err != nil || obs.State != "stopped" || state.MaintenanceUncertain {
		t.Fatal("untested stable patch blocked maintenance", err)
	}
	for _, c := range state.Runtime.Containers {
		if c.EngineVersion != "29.9.1" {
			t.Fatal("maintenance used assumed daemon version")
		}
	}
}

func TestCompatibilityUnsupportedFamilyCannotAutoStopOrStartupReplace(t *testing.T) {
	for _, action := range []string{"routine-pack", "startup-resume"} {
		t.Run(action, func(t *testing.T) {
			f, state, persist := managedFixture(t)
			state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
			if err != nil {
				t.Fatal(err)
			}
			if action == "routine-pack" {
				state.Routine = &routines.Run{Action: "pack", StopRunning: true, Resume: true, WasRunning: true, Deadline: time.Now().Add(time.Minute)}
			} else {
				for _, c := range f.containers {
					mapAt(c.doc, "State")["Running"] = false
				}
			}
			setFixtureEngineVersion(f, "30.0.0")
			before := len(f.calls)
			_, obs, err := applyManaged(context.Background(), action, state, persist, f.d)
			if !errors.Is(err, ErrHostCompatibility) || obs.RecoveryRequired != (action == "startup-resume") {
				t.Fatalf("automatic work was not denied as compatibility: %+v %v", obs, err)
			}
			for _, call := range f.calls[before:] {
				if call[1] != "inspect" {
					t.Fatalf("automatic control on unsupported family: %q", call)
				}
			}
			if len(f.containers) != 1 {
				t.Fatal("automatic work removed historical runtime")
			}
		})
	}
}

func TestCompatibilityLegacyAmesAndImportHistoryIndependentOfCurrentEngine(t *testing.T) {
	t.Run("Ames", func(t *testing.T) {
		f, state, _ := legacyAmesFixture(t, "~sampel-palnet", "planets", true, false)
		state.Runtime.EngineVersion = "27.2.0"
		setFixtureEngineVersion(f, "29.9.1")
		before, _ := json.Marshal(state)
		if err := validateAmesPublication(context.Background(), state, f.d); err != nil {
			t.Fatal(err)
		}
		after, _ := json.Marshal(state)
		if string(before) != string(after) {
			t.Fatal("publication rewrote history")
		}
		state.Runtime.EngineVersion = "untrusted"
		if err := validateAmesPublication(context.Background(), state, f.d); err == nil {
			t.Fatal("malformed historical version accepted")
		}
	})
	t.Run("qualified import", func(t *testing.T) {
		f, state, persist := qualifiedImportFixture(t)
		catalogue, err := importCatalogue(f.d)
		if err != nil {
			t.Fatal(err)
		}
		report, err := catalogue.importReport(state.Import.Source.RunID)
		if err != nil {
			t.Fatal(err)
		}
		report.EngineVersion = "27.2.0"
		catalogue.r = report
		if err := catalogue.save(); err != nil {
			t.Fatal(err)
		}
		before, err := os.ReadFile(report.ReportPath)
		if err != nil {
			t.Fatal(err)
		}
		setFixtureEngineVersion(f, "29.9.1")
		state, _, err = applyManaged(context.Background(), "import-fake", state, persist, f.d)
		if err != nil || state.Runtime.EngineVersion != "29.9.1" {
			t.Fatalf("historical import/current provisioning confused: %v", err)
		}
		after, err := os.ReadFile(report.ReportPath)
		if err != nil || string(before) != string(after) {
			t.Fatal("import rewrote qualified history", err)
		}
	})
}

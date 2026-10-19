//go:build linux && amd64

package runtimequalify

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Persist the entire driver state, close it, then reopen it for each simulated
// management restart. The fixture's report is separate Docker intent evidence;
// retaining only the returned state or report would miss a provisioning fence.
type bootAdmissionJournal struct {
	t      *testing.T
	path   string
	report func(ManagedState) error
}

func newBootAdmissionJournal(t *testing.T, report func(ManagedState) error) *bootAdmissionJournal {
	t.Helper()
	return &bootAdmissionJournal{t: t, path: filepath.Join(t.TempDir(), "managed-state.json"), report: report}
}

func (j *bootAdmissionJournal) save(state ManagedState) error {
	if err := j.report(state); err != nil {
		return err
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(j.path), ".managed-state-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	if err = errors.Join(err, f.Close()); err != nil {
		return err
	}
	if err = os.Rename(f.Name(), j.path); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(j.path))
	if err != nil {
		return err
	}
	return errors.Join(dir.Sync(), dir.Close())
}

func (j *bootAdmissionJournal) reopen() ManagedState {
	j.t.Helper()
	f, err := os.Open(j.path)
	if err != nil {
		j.t.Fatal(err)
	}
	var state ManagedState
	if err = errors.Join(json.NewDecoder(f).Decode(&state), f.Close()); err != nil {
		j.t.Fatal(err)
	}
	return state
}

func bootAdmissionFixture(t *testing.T, owned bool) (*fakeEngine, ManagedState, *bootAdmissionJournal) {
	t.Helper()
	f, state, persist := managedCometFixture(t)
	if owned {
		state.NativeShip, state.Comet.Ship = "~sampel-palnet", "~sampel-palnet"
		state.Storage.Path = "livenet/planets/sampel-palnet/pier"
		state.Boot, state.AmesPort = &ManagedBoot{}, 34343
		key := "0w" + strings.Repeat("a", 128)
		state.NativeKeyDigest = fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
		dir := filepath.Join(f.d.home, ".local/share/urbit-runner/managed/keys")
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, state.ID+".key"), []byte(key), 0600); err != nil {
			t.Fatal(err)
		}
		identity := f.d.hostIdentity
		f.d.hostIdentity = func(ctx context.Context, endpoint string) (string, error) {
			_, err := identity(ctx, endpoint) // retain the private endpoint assertion
			return "~sampel-palnet", err
		}
	}
	journal := newBootAdmissionJournal(t, persist)
	if err := journal.save(state); err != nil {
		t.Fatal(err)
	}
	return f, state, journal
}

func bootAdmissionRecord(state ManagedState) (ContainerRecord, bool) {
	for _, record := range state.Runtime.Containers {
		if record.Kind == "comet-boot" {
			return record, true
		}
	}
	return ContainerRecord{}, false
}

func bootAdmissionEventCount(f *fakeEngine, event string) int {
	count := 0
	for _, observed := range f.events {
		if observed == event {
			count++
		}
	}
	return count
}

func requireBootAdmissionRetry(t *testing.T, f *fakeEngine, original ManagedState, journal *bootAdmissionJournal) {
	t.Helper()
	state := journal.reopen()
	if state.Initialized || state.UncertainProvisioning() || state.Comet.StartedAt != nil || state.Comet.FinishedAt != nil || state.ID != original.ID || state.Storage.Path != original.Storage.Path || state.NativeShip != original.NativeShip || state.NativeKeyDigest != original.NativeKeyDigest {
		t.Fatal("never-dispatched boot lost its retryable identity/key contract")
	}
	mount := state.Storage.MountIdentity
	for range 2 {
		var err error
		state, _, err = applyManaged(context.Background(), "stop", state, journal.save, f.d)
		if err != nil || state.UncertainProvisioning() || state.Comet.StartedAt != nil {
			t.Fatal("Stop quarantined a proven undispatched boot", err)
		}
		state = journal.reopen()
	}
	state, obs, err := observeManaged(context.Background(), state, f.d)
	if err != nil || obs.State != "uninitialized" || obs.RecoveryRequired {
		t.Fatalf("clean preparation not observable as retryable after reopen: %+v %v", obs, err)
	}
	state, obs, err = applyManaged(context.Background(), "start", state, journal.save, f.d)
	if err != nil || !state.Initialized || obs.Health != "healthy" || state.UncertainProvisioning() || state.Storage.MountIdentity != mount {
		t.Fatalf("restored host proof did not allow first boot: %+v %v", obs, err)
	}
	state = journal.reopen()
	ship := fixtureComet
	if original.NativeShip != "" {
		ship = original.NativeShip
		if state.NativeKeyDigest != "" {
			t.Fatal("successful owned boot did not consume its key")
		}
	}
	if state.Comet.Ship != ship || state.ID != original.ID || state.Storage.Path != original.Storage.Path {
		t.Fatal("first successful boot changed the registered or generated identity")
	}
	started, finished, identity := *state.Comet.StartedAt, *state.Comet.FinishedAt, state.Storage.Identity
	for range 2 {
		state, _, err = applyManaged(context.Background(), "stop", state, journal.save, f.d)
		if err != nil {
			t.Fatal(err)
		}
		state = journal.reopen()
	}
	state, obs, err = applyManaged(context.Background(), "start", state, journal.save, f.d)
	if err != nil || obs.Health != "healthy" || state.Comet.Ship != ship || !state.Comet.StartedAt.Equal(started) || !state.Comet.FinishedAt.Equal(finished) || state.Storage.Identity != identity || state.Storage.MountIdentity != mount {
		t.Fatalf("replacement resumed a different identity or boot receipt: %+v %v", obs, err)
	}
	if bootAdmissionEventCount(f, "create-comet-boot") != 1 || bootAdmissionEventCount(f, "start-comet-boot") != 1 || bootAdmissionEventCount(f, "create-comet-resume") != 1 {
		t.Fatal("retry/repeated Stop caused duplicate identity boot", f.events)
	}
}

func TestManagedBootLateCompatibilityAdmissionRemainsRetryableAfterReopen(t *testing.T) {
	for _, owned := range []bool{false, true} {
		for _, hook := range []string{"probe", "network"} {
			for _, fault := range []string{"provenance", "installed-family"} {
				t.Run(fmt.Sprintf("owned=%t/%s/%s", owned, hook, fault), func(t *testing.T) {
					f, original, journal := bootAdmissionFixture(t, owned)
					execute := f.d.execute
					late := false
					f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
						out, err := execute(ctx, args, env)
						if err == nil && !late && (hook == "probe" && args[4] == "container" && args[5] == "rm" || hook == "network" && args[4] == "network" && args[5] == "create") {
							late = true
							if fault == "provenance" {
								f.hostReport.Packages[0].Provenance = "metadata-only"
							} else {
								f.hostReport.InstalledEngineVersion = "30.0.0"
							}
						}
						return out, err
					}
					state, obs, err := applyManaged(context.Background(), "start", original, journal.save, f.d)
					if !errors.Is(err, ErrHostCompatibility) || !late || obs.RecoveryRequired || state.UncertainProvisioning() || len(f.containers) != 0 {
						t.Fatalf("late host refusal stranded boot before any dispatch: %+v %v", obs, err)
					}
					if _, exists := bootAdmissionRecord(journal.reopen()); exists || bootAdmissionEventCount(f, "create-comet-boot") != 0 {
						t.Fatal("late admission failure reached a boot intent or dispatch")
					}
					f.d.execute = execute
					f.hostReport.Packages[0].Provenance = "verified"
					f.hostReport.InstalledEngineVersion = f.engine.Version
					requireBootAdmissionRetry(t, f, original, journal)
				})
			}
		}
	}
}

func TestManagedBootDispatchFenceIsAtomicAndConservativeAcrossReopen(t *testing.T) {
	for _, owned := range []bool{false, true} {
		for _, cut := range []string{"intent", "dispatch-rejected", "dispatch-ack-lost", "dispatch-canceled", "create-response", "start-response", "legacy-report-only"} {
			t.Run(fmt.Sprintf("owned=%t/%s", owned, cut), func(t *testing.T) {
				f, original, journal := bootAdmissionFixture(t, owned)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				interrupted := false
				save := func(state ManagedState) error {
					record, exists := bootAdmissionRecord(state)
					if !state.Initialized && state.Comet.Uncertain && (!exists || !record.CreateDispatched || state.Comet.StartedAt == nil) {
						t.Fatal("provisioning fence was published without its boot dispatch marker")
					}
					if exists && record.CreateDispatched && cut == "dispatch-rejected" && !interrupted {
						interrupted = true
						return errors.New("dispatch journal refused before commit")
					}
					if err := journal.save(state); err != nil {
						return err
					}
					if !interrupted && exists {
						switch {
						case !record.CreateDispatched && (cut == "intent" || cut == "legacy-report-only"):
							interrupted = true
							cancel()
						case record.CreateDispatched && cut == "dispatch-ack-lost":
							interrupted = true
							return errors.New("dispatch journal acknowledgment lost after commit")
						case record.CreateDispatched && cut == "dispatch-canceled":
							interrupted = true
							cancel()
						}
					}
					return nil
				}
				execute := f.d.execute
				f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
					bootCreate := len(args) > 7 && args[4] == "container" && args[5] == "create" && strings.HasSuffix(args[7], "-comet-boot")
					bootStart := false
					if len(args) > 6 && args[4] == "container" && args[5] == "start" {
						for _, c := range f.containers {
							bootStart = bootStart || c.kind == "comet-boot" && c.id == args[6]
						}
					}
					if bootCreate {
						durable := journal.reopen()
						record, exists := bootAdmissionRecord(durable)
						if !exists || !record.CreateDispatched || !record.CreateUncertain || !durable.Comet.Uncertain || durable.Comet.StartedAt == nil {
							t.Fatal("boot reached Docker before its complete durable dispatch fence")
						}
					}
					out, err := execute(ctx, args, env)
					if err == nil && !interrupted && (bootCreate && cut == "create-response" || bootStart && cut == "start-response") {
						interrupted = true
						return "", errors.New("fixture response lost after actual dispatch")
					}
					return out, err
				}
				if _, _, err := applyManaged(ctx, "start", original, save, f.d); err == nil || !interrupted {
					t.Fatal("did not reach the durable interruption boundary", err)
				}
				f.d.execute = execute
				if cut == "intent" || cut == "dispatch-rejected" {
					requireBootAdmissionRetry(t, f, original, journal)
					return
				}
				state := journal.reopen()
				if cut == "legacy-report-only" {
					// Old writers published the comet fence before any boot intent.
					// Absence of a boot record cannot prove that historical work never ran.
					state.Runtime.Containers = state.Runtime.Containers[:len(state.Runtime.Containers)-1]
					now := time.Now().UTC()
					state.Comet.Uncertain, state.Comet.StartedAt = true, &now
					if err := journal.save(state); err != nil {
						t.Fatal(err)
					}
					state = journal.reopen()
				}
				if !state.UncertainProvisioning() || state.Comet.StartedAt == nil || state.Initialized {
					t.Fatal("dispatched/legacy boot fence disappeared across reopen")
				}
				before, _ := json.Marshal(state.Comet)
				for range 2 {
					next, obs, err := applyManaged(context.Background(), "stop", state, journal.save, f.d)
					if (cut == "dispatch-ack-lost" || cut == "dispatch-canceled") && err == nil {
						t.Fatal("daemon absence resolved an uncertain dispatched create")
					}
					if (cut == "create-response" || cut == "start-response" || cut == "legacy-report-only") && err != nil {
						t.Fatal(err)
					}
					after, _ := json.Marshal(next.Comet)
					if !obs.RecoveryRequired || !next.UncertainProvisioning() || string(before) != string(after) {
						t.Fatal("Stop cleared or relabeled a dispatched/legacy boot fence", err)
					}
					state = journal.reopen()
				}
				calls := len(f.calls)
				if _, _, err := applyManaged(context.Background(), "start", state, journal.save, f.d); err == nil || len(f.calls) != calls {
					t.Fatal("dispatched/legacy identity bootstrap was retried")
				}
			})
		}
	}
}

//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestUndispatchedInitialBootRejectsLegacyAndContradictoryEvidence(t *testing.T) {
	for _, mode := range []string{"fake", "comet", "owned"} {
		t.Run(mode, func(t *testing.T) {
			var f *fakeEngine
			var state ManagedState
			var journal *bootAdmissionJournal
			if mode == "fake" {
				var persist func(ManagedState) error
				f, state, persist = managedFixture(t)
				journal = newBootAdmissionJournal(t, persist)
			} else {
				f, state, journal = bootAdmissionFixture(t, mode == "owned")
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			_, _, err := applyManaged(ctx, "start", state, func(s ManagedState) error {
				if err := journal.save(s); err != nil {
					return err
				}
				for _, c := range s.Runtime.Containers {
					if (c.Kind == "http-boot" || c.Kind == "comet-boot") && !c.CreateDispatched {
						cancel()
					}
				}
				return nil
			}, f.d)
			if err == nil {
				t.Fatal("boot intent interruption not reached")
			}
			state = journal.reopen()
			if !state.UndispatchedInitialBoot() {
				t.Fatal("real pre-dispatch checkpoint was not recognized")
			}
			original, err := json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
			boot := func(s *ManagedState) *ContainerRecord { return &s.Runtime.Containers[len(s.Runtime.Containers)-1] }
			for name, change := range map[string]func(*ManagedState){
				"legacy-no-version":      func(s *ManagedState) { boot(s).EngineVersion = "" },
				"malformed-version":      func(s *ManagedState) { boot(s).EngineVersion = "untrusted" },
				"wrong-kind":             func(s *ManagedState) { boot(s).Kind = "comet-resume" },
				"wrong-name":             func(s *ManagedState) { boot(s).Name = "foreign" },
				"dispatch-marker":        func(s *ManagedState) { boot(s).CreateDispatched = true },
				"uncertain-create":       func(s *ManagedState) { boot(s).CreateUncertain = true },
				"daemon-id":              func(s *ManagedState) { boot(s).ID = strings.Repeat("f", 64) },
				"daemon-state":           func(s *ManagedState) { boot(s).LastState = &ContainerState{} },
				"alive-evidence":         func(s *ManagedState) { boot(s).AliveObserved = true },
				"http-evidence":          func(s *ManagedState) { boot(s).HostHTTPVerified = true },
				"endpoint-evidence":      func(s *ManagedState) { boot(s).HostEndpoint = "http://127.0.0.1:18080/~/host" },
				"removed-intent":         func(s *ManagedState) { boot(s).Removed = true },
				"missing-intent":         func(s *ManagedState) { s.Runtime.Containers = s.Runtime.Containers[:1] },
				"retained-probe":         func(s *ManagedState) { s.Runtime.Containers[0].Removed = false },
				"uncertain-network":      func(s *ManagedState) { s.Runtime.Network.CreateUncertain = true },
				"unbound-network":        func(s *ManagedState) { s.Runtime.Network.ID = "" },
				"prior-storage-fence":    func(s *ManagedState) { s.Storage.Uncertain = true },
				"prior-pier-identity":    func(s *ManagedState) { s.Storage.Identity.Inode = 1 },
				"missing-mount-identity": func(s *ManagedState) { s.Storage.MountIdentity.Inode = 0 },
				"prior-provisioning-fence": func(s *ManagedState) {
					if s.Comet != nil {
						s.Comet.Uncertain = true
					} else {
						s.Fake = &ManagedFake{Ship: "~zod", Uncertain: true}
					}
				},
				"prior-started-evidence": func(s *ManagedState) {
					if s.Comet != nil {
						now := time.Now().UTC()
						s.Comet.StartedAt = &now
					} else {
						s.Initialized = true
					}
				},
				"prior-maintenance":     func(s *ManagedState) { s.MaintenanceUncertain = true },
				"prior-adoption":        func(s *ManagedState) { s.Adoption = &ManagedAdoption{} },
				"retained-runtime-logs": func(s *ManagedState) { s.Runtime.RetainedLogs = &protocol.RuntimeLogs{} },
				"prior-runtime-work":    func(s *ManagedState) { s.Runtime.CometUtilities = []CometUtilityRecord{{Kind: "comet-pack"}} },
				"advanced-stage":        func(s *ManagedState) { s.Runtime.Stage = "starting-runtime" },
			} {
				t.Run(name, func(t *testing.T) {
					var changed ManagedState
					if err := json.Unmarshal(original, &changed); err != nil {
						t.Fatal(err)
					}
					change(&changed)
					before, _ := json.Marshal(changed)
					if changed.UndispatchedInitialBoot() {
						t.Fatal("ambiguous or contrary evidence qualified for non-dispatch")
					}
					after, _ := json.Marshal(changed)
					if string(before) != string(after) {
						t.Fatal("read-only recognition mutated a fence")
					}
				})
			}
		})
	}
}

func TestBootDispatchJournalErrorRestoresMaintenanceStageBeforeCompletion(t *testing.T) {
	for _, committed := range []bool{false, true} {
		t.Run(map[bool]string{false: "before-commit", true: "lost-ack"}[committed], func(t *testing.T) {
			f, state, persist := stoppedManagedFixture(t)
			journal := newBootAdmissionJournal(t, persist)
			cut := false
			state, _, err := applyManaged(context.Background(), "pack", state, func(s ManagedState) error {
				if s.Runtime.Stage == "maintenance-dispatched-pack" {
					cut = true
					if committed {
						if err := journal.save(s); err != nil {
							return err
						}
					}
					return errors.New("dispatch journal failed")
				}
				return journal.save(s)
			}, f.d)
			if err == nil || !cut || state.Runtime.Stage != managedMaintenancePendingStage("pack") || !state.MaintenanceUncertain {
				t.Fatal("journal failure returned a contradictory maintenance stage", err)
			}
			last := state.Runtime.Containers[len(state.Runtime.Containers)-1]
			if !containerCreateUndispatched(last) || bootAdmissionEventCount(f, "create-pack") != 0 {
				t.Fatal("journal failure dispatched utility work")
			}
			// Model the completion write of the returned state, not the previous
			// callback; the managed integration suite exercises real SQLite finish.
			if err := journal.save(state); err != nil {
				t.Fatal(err)
			}
			state, obs, err := applyManaged(context.Background(), "stop", journal.reopen(), journal.save, f.d)
			if err != nil || obs.RecoveryRequired || state.UncertainMaintenance() {
				t.Fatal("rolled-back journal tuple was not resolvable", err)
			}
		})
	}
}

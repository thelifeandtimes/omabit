//go:build linux && amd64

package managed

import (
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"os"
	"reflect"
	"testing"
)

func TestVMRecoveryCleanupFindsRetirementThroughLaterCheckpoints(t *testing.T) {
	s, err := testOpenStore(privateDir(t))
	g02Check(t, err)
	defer s.close()
	p, req := recoveryStoreProject(t, s)
	op, _, err := s.acceptProjectAnchorRecovery("actor", p.ID, req)
	g02Check(t, err)
	j, err := loadProjectAnchorRecovery(s.db, op.ID)
	g02Check(t, err)
	j = completeRecoveryStoreFixture(t, s, j)
	final := j.Journal.Readiness.Clone()
	for _, phase := range []string{"retiring", "replacing", "complete"} {
		t.Run(phase, func(t *testing.T) {
			copy := recoveryTestClone(t, j)
			switch phase {
			case "retiring":
				retired := final.Replacement.Retirement.Clone()
				copy.Journal.Readiness = nil
				copy.Journal.Retirement = &retired
				copy.Journal.Phase = phase
				// Aggregate sequence is derived from the completed nested checkpoints.
				copy.Journal.Sequence = 5 + retired.Namespace.Drain.Sequence + retired.Namespace.Sequence + retired.Sequence
			case "replacing":
				replacement := final.Replacement.Clone()
				copy.Journal.Readiness = nil
				copy.Journal.Replacement = &replacement
				copy.Journal.Phase = phase
				copy.Journal.Sequence--
			}
			g02Check(t, copy.Journal.Validate(copy.Source))
			for _, member := range final.Replacement.Retirement.Namespace.Drain.Review.Context.Members {
				old := member.Context.Member
				result := map[string]any{"anchor_recovery_retirement": &vmRecoveryCleanupReceipt{job: &copy}}
				calls := 0
				docker := func(args ...string) json.RawMessage {
					calls++
					if !reflect.DeepEqual(args, []string{"container", "ls", "--all", "--quiet", "--no-trunc"}) {
						t.Fatal("cleanup attempted mutation", args)
					}
					return json.RawMessage("\n")
				}
				if !vmRecoveryRetirementCleanup(t, old.Container.ID, old.OperationID, old.Storage, result, docker) || calls != 1 || result["retired_container_absent"] != true || result["anchor_recovery_operation_id"] != op.ID {
					t.Fatal("missing exact retirement", phase, result, calls)
				}
				raw, err := json.Marshal(result)
				g02Check(t, err)
				if len(raw) > 1024 {
					t.Fatal("cleanup duplicated recovery journal", len(raw))
				}
				if vmRecoveryRetirementCleanup(t, old.Container.ID, "different-creator", old.Storage, result, docker) || calls != 1 {
					t.Fatal("foreign creator accepted")
				}
			}
		})
	}
}

func TestVMRecoveredCleanupUsesCompleteActualRecoveryOwnership(t *testing.T) {
	raw, err := os.ReadFile("testdata/recovery31-cleanup-ownership.json")
	g02Check(t, err)
	var fixture struct {
		RunnerID string `json:"runner_id"`
		Cases    []struct {
			Creation    projectnetwork.RecoveryMemberCreation `json:"creation"`
			ContainerID string                                `json:"container_id"`
			Inspection  json.RawMessage                       `json:"inspection"`
		} `json:"cases"`
	}
	g02Check(t, json.Unmarshal(raw, &fixture))
	if len(fixture.Cases) != 2 {
		t.Fatal("missing both actual recovered containers")
	}
	for _, item := range fixture.Cases {
		t.Run(item.Creation.InstanceID, func(t *testing.T) {
			row, err := vmRecoveredCleanupInspection(fixture.RunnerID, item.Creation, item.ContainerID, item.Inspection)
			g02Check(t, err)
			if !row.State.Running {
				t.Fatal("actual failed cleanup must precede Stop")
			}
			for _, key := range []string{"namespace-operation", "retirement-operation", "operation", "runner", "instance", "runtime", "container-generation"} {
				for _, wrong := range []string{"", "foreign"} {
					var rows []vmRecoveredCleanupContainer
					g02Check(t, json.Unmarshal(item.Inspection, &rows))
					label := "org.urbit-runner.project." + key
					if wrong == "" {
						delete(rows[0].Config.Labels, label)
					} else {
						rows[0].Config.Labels[label] = wrong
					}
					changed, err := json.Marshal(rows)
					g02Check(t, err)
					if _, err = vmRecoveredCleanupInspection(fixture.RunnerID, item.Creation, item.ContainerID, changed); err == nil {
						t.Fatal("accepted changed ownership", key, wrong)
					}
				}
			}
			for _, change := range []func(*vmRecoveredCleanupContainer){
				func(r *vmRecoveredCleanupContainer) { r.ID = "foreign" },
				func(r *vmRecoveredCleanupContainer) { r.Name += "-foreign" },
				func(r *vmRecoveredCleanupContainer) { r.Config.Labels["foreign"] = "true" },
			} {
				var rows []vmRecoveredCleanupContainer
				g02Check(t, json.Unmarshal(item.Inspection, &rows))
				change(&rows[0])
				changed, err := json.Marshal(rows)
				g02Check(t, err)
				if _, err = vmRecoveredCleanupInspection(fixture.RunnerID, item.Creation, item.ContainerID, changed); err == nil {
					t.Fatal("accepted foreign object")
				}
			}
		})
	}
}

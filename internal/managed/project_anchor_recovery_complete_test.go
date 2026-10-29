package managed

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

func recoveryTestClone[T any](t *testing.T, v T) T {
	t.Helper()
	b, err := json.Marshal(v)
	g02Check(t, err)
	var out T
	g02Check(t, json.Unmarshal(b, &out))
	return out
}

type recoveryStoreReadiness struct {
	state projectnetwork.AnchorRetirementObservation
	h     projectnetwork.AnchorRecoveryHistory
}

func (f *recoveryStoreReadiness) VerifyEngine(context.Context, string) error { return nil }
func (f *recoveryStoreReadiness) InspectRecoveryReadiness(context.Context, projectnetwork.AnchorReplacement) (projectnetwork.AnchorRetirementObservation, error) {
	o := f.state
	o.CapturedAt = time.Now().UTC()
	o.Runtime.ObservedAt = o.CapturedAt
	return o, nil
}
func (f *recoveryStoreReadiness) ReadHostIdentity(_ context.Context, endpoint string) (string, error) {
	for _, m := range f.h.Plan.Members {
		if endpoint == fmt.Sprintf("http://127.0.0.1:%d/~/host", m.HTTPPort) {
			return m.Ship, nil
		}
	}
	return "", fmt.Errorf("unexpected endpoint %s", endpoint)
}

// Pure synthetic observations drive the real SQLite transitions. They do not
// invoke Docker or touch piers. Completed history and creator receipts must then
// remain readable after reopening, including a second coordinated recovery.
func completeRecoveryStoreFixture(t *testing.T, s *store, j projectAnchorRecoveryJob) projectAnchorRecoveryJob {
	t.Helper()
	step := func(phase string, u projectnetwork.AnchorRecoveryJournal) {
		t.Helper()
		var err error
		j, err = s.advanceProjectAnchorRecovery(j.OperationID, j.Journal.Sequence, phase, u)
		g02Check(t, err)
	}
	for _, p := range []string{"reviewing", "reviewed", "draining"} {
		j = advanceRecoveryStoreFixture(t, s, j, p)
	}
	drainState := recoveryStoreDrainObservation(j)
	tickDrain := func() projectnetwork.AnchorDrainObservation {
		o := recoveryTestClone(t, drainState)
		o.CapturedAt = time.Now().UTC()
		o.Runtime.ObservedAt = o.CapturedAt
		return o
	}
	for i, m := range j.Journal.Drain.Members {
		before := tickDrain()
		next, err := j.Journal.Drain.Advance(j.Source, i, j.Journal.Drain.Sequence, "stop-dispatched", &before)
		g02Check(t, err)
		step("draining", projectnetwork.AnchorRecoveryJournal{Drain: &next})
		// Deliberately omit acknowledgement: exact exit observation resolves outcome.
		now := time.Now().UTC()
		for k := range drainState.Runtime.Members {
			if drainState.Runtime.Members[k].InstanceID == m.InstanceID {
				drainState.Runtime.Members[k].Running = false
				drainState.Runtime.Members[k].Namespace = projectnetwork.NamespaceIdentity{}
			}
		}
		for k := range drainState.Processes {
			if drainState.Processes[k].InstanceID == m.InstanceID {
				drainState.Processes[k].Process.FinishedAt = now
			}
		}
		after := tickDrain()
		next, err = j.Journal.Drain.Advance(j.Source, i, j.Journal.Drain.Sequence, "exited", &after)
		g02Check(t, err)
		step("draining", projectnetwork.AnchorRecoveryJournal{Drain: &next})
	}
	step("namespace", projectnetwork.AnchorRecoveryJournal{})
	before := tickDrain()
	ns, err := j.Journal.Namespace.Advance(j.Source, j.Journal.Namespace.Sequence, "observing", &before)
	g02Check(t, err)
	step("namespace", projectnetwork.AnchorRecoveryJournal{Namespace: &ns})
	after := tickDrain()
	ns, err = j.Journal.Namespace.Advance(j.Source, j.Journal.Namespace.Sequence, "observed", &after)
	g02Check(t, err)
	step("namespace", projectnetwork.AnchorRecoveryJournal{Namespace: &ns})
	step("retiring", projectnetwork.AnchorRecoveryJournal{})
	source := j.Journal.Retirement.Namespace.After
	state := projectnetwork.AnchorRetirementObservation{Runtime: recoveryTestClone(t, source.Runtime), AnchorProcess: source.AnchorProcess, Processes: recoveryTestClone(t, source.Processes), CapturedAt: source.CapturedAt}
	for _, p := range source.Processes {
		state.Piers = append(state.Piers, projectnetwork.RecoveryPier{InstanceID: p.InstanceID, Storage: p.Storage})
	}
	tick := func() projectnetwork.AnchorRetirementObservation {
		o := recoveryTestClone(t, state)
		o.CapturedAt = time.Now().UTC()
		o.Runtime.ObservedAt = o.CapturedAt
		return o
	}
	for i, m := range j.Journal.Retirement.Members {
		before := tick()
		next, err := j.Journal.Retirement.Advance(j.Source, i, j.Journal.Retirement.Sequence, "remove-dispatched", &before)
		g02Check(t, err)
		step("retiring", projectnetwork.AnchorRecoveryJournal{Retirement: &next})
		state.Runtime.Members = slices.DeleteFunc(state.Runtime.Members, func(v projectnetwork.MemberObservation) bool { return v.InstanceID == m.InstanceID })
		state.Processes = slices.DeleteFunc(state.Processes, func(v projectnetwork.AnchorDrainProcess) bool { return v.InstanceID == m.InstanceID })
		after := tick()
		next, err = j.Journal.Retirement.Advance(j.Source, i, j.Journal.Retirement.Sequence, "absent", &after)
		g02Check(t, err)
		step("retiring", projectnetwork.AnchorRecoveryJournal{Retirement: &next})
	}
	var members []projectnetwork.RecoveredMember
	for i, id := range j.Journal.IDs.Members {
		c, err := projectnetwork.SelectRecoveryMemberCreation(j.Source, *j.Journal.Retirement, id.InstanceID, id.OperationID)
		g02Check(t, err)
		members = append(members, projectnetwork.RecoveredMember{Creation: c, Container: projectnetwork.ResourceIdentity{ContractDigest: strings.Repeat(string(rune('a'+i)), 64)}, Phase: "planned"})
	}
	replacement, err := projectnetwork.NewAnchorReplacement(j.Source, *j.Journal.Retirement, j.OperationID, members)
	g02Check(t, err)
	step("replacing", projectnetwork.AnchorRecoveryJournal{Replacement: &replacement})
	for i, m := range j.Journal.Replacement.Members {
		before := tick()
		next, err := j.Journal.Replacement.Advance(j.Source, i, j.Journal.Replacement.Sequence, "create-dispatched", projectnetwork.AnchorReplacementResult{Observation: &before})
		g02Check(t, err)
		step("replacing", projectnetwork.AnchorRecoveryJournal{Replacement: &next})
		id := fmt.Sprintf("%064x", 1000+len(j.Source.Epochs)*100+i)
		next, err = j.Journal.Replacement.Advance(j.Source, i, j.Journal.Replacement.Sequence, "created", projectnetwork.AnchorReplacementResult{ContainerID: id})
		g02Check(t, err)
		step("replacing", projectnetwork.AnchorRecoveryJournal{Replacement: &next})
		state.Runtime.Members = append(state.Runtime.Members, projectnetwork.MemberObservation{InstanceID: m.Creation.InstanceID, ResourceObservation: projectnetwork.ResourceObservation{Resource: j.Journal.Replacement.Members[i].Container, Present: true}, AnchorID: state.Runtime.Anchor.Resource.ID})
		state.Processes = append(state.Processes, projectnetwork.AnchorDrainProcess{InstanceID: m.Creation.InstanceID, Storage: m.Creation.Storage})
		created := tick()
		next, err = j.Journal.Replacement.Advance(j.Source, i, j.Journal.Replacement.Sequence, "start-dispatched", projectnetwork.AnchorReplacementResult{Observation: &created})
		g02Check(t, err)
		step("replacing", projectnetwork.AnchorRecoveryJournal{Replacement: &next})
		for k := range state.Runtime.Members {
			if state.Runtime.Members[k].InstanceID == m.Creation.InstanceID {
				state.Runtime.Members[k].Running = true
				state.Runtime.Members[k].Namespace = m.Creation.Namespace
			}
		}
		for k := range state.Processes {
			if state.Processes[k].InstanceID == m.Creation.InstanceID {
				state.Processes[k].Process.StartedAt = time.Now().UTC()
			}
		}
		after := tick()
		next, err = j.Journal.Replacement.Advance(j.Source, i, j.Journal.Replacement.Sequence, "running", projectnetwork.AnchorReplacementResult{Observation: &after})
		g02Check(t, err)
		step("replacing", projectnetwork.AnchorRecoveryJournal{Replacement: &next})
	}
	if err = projectRetirementClear(s.db, j.ProjectID, ""); err == nil {
		t.Fatal("running replacements prematurely released recovery fence")
	}
	ready, err := projectnetwork.CollectAnchorRecoveryReadiness(context.Background(), j.Source, *j.Journal.Replacement, &recoveryStoreReadiness{state: state, h: j.Source}, func() time.Time { return time.Now().UTC() })
	g02Check(t, err)
	step("complete", projectnetwork.AnchorRecoveryJournal{Readiness: &ready})
	return j
}
func TestAnchorRecoveryStoreCompleteAndSecondRecovery(t *testing.T) {
	dir := privateDir(t)
	s, err := testOpenStore(dir)
	g02Check(t, err)
	p, req := recoveryStoreProject(t, s)
	old := currentStopHistory(t, s)
	for epoch := 0; epoch < 2; epoch++ {
		h, _, _, err := projectRecoveryHistory(s.db, p.ID, true)
		g02Check(t, err)
		req.RequestID = fmt.Sprintf("recover-%d", epoch)
		req.HistoryDigest = recoveryHistoryDigest(h)
		op, _, err := s.acceptProjectAnchorRecovery("actor", p.ID, req)
		g02Check(t, err)
		j, err := loadProjectAnchorRecovery(s.db, op.ID)
		g02Check(t, err)
		j = completeRecoveryStoreFixture(t, s, j)
		g02Check(t, validateProjectAnchorRecoveries(s.db))
		g02Check(t, projectRetirementClear(s.db, p.ID, ""))
		after, _, _, err := projectRecoveryHistory(s.db, p.ID, true)
		g02Check(t, err)
		if len(after.Epochs) != epoch+1 || len(after.Resumes) != 0 {
			t.Fatal("incomplete causal history")
		}
		for i, m := range j.Journal.Readiness.Replacement.Members {
			if m.StartAcknowledged {
				t.Fatal("lost acknowledgement invented")
			}
			g02Check(t, projectRecoveryMirror(s.db, m.Creation.OperationID, recoveryMemberReceipt(j, i)))
		}
		assertContextHistory(t, s, old)
		old = currentStopHistory(t, s)
		g02Check(t, s.close())
		s, err = testOpenStore(dir)
		g02Check(t, err)
		g02Check(t, s.reconcileInterrupted())
		reopened, _, _, err := projectRecoveryHistory(s.db, p.ID, true)
		g02Check(t, err)
		if !reflect.DeepEqual(after, reopened) {
			t.Fatal("completed history changed on reopen")
		}
		replay, hit, err := s.acceptProjectAnchorRecovery("actor", p.ID, req)
		g02Check(t, err)
		if !hit || replay.Status != "succeeded" {
			t.Fatal("completed recovery lost replay")
		}
	}
	g02Check(t, s.close())
}

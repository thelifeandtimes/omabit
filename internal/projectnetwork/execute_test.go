package projectnetwork

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

type preparationEngineFixture struct {
	t                                     *testing.T
	journal                               *Preparation
	events                                []string
	failCreate, failInspect, failEngine   int
	creations, inspections, verifications int
	createReturnedID                      bool
	cancel                                context.CancelFunc
}

func (f *preparationEngineFixture) VerifyEngine(ctx context.Context, id string) error {
	f.verifications++
	if id != f.journal.EngineID {
		f.t.Fatal("wrong engine selected")
	}
	f.events = append(f.events, "engine")
	if f.verifications == f.failEngine {
		return errors.New("untrusted-provider-text")
	}
	return ctx.Err()
}
func (f *preparationEngineFixture) Create(ctx context.Context, kind string, r Creation, bridgeID string) (string, error) {
	f.creations++
	f.events = append(f.events, "create-"+kind)
	stored := f.journal.Bridge
	if kind == "anchor" {
		stored = f.journal.Anchor
		if bridgeID != f.journal.Bridge.ID || f.journal.Bridge.Phase != "observed" {
			f.t.Fatal("anchor dispatched without observed bridge")
		}
	}
	if r != stored || r.Phase != "dispatched" {
		f.t.Fatal("engine create preceded durable intent")
	}
	id := strings.Repeat("a", 64)
	if kind == "anchor" {
		id = strings.Repeat("b", 64)
	}
	if f.cancel != nil {
		f.cancel()
	}
	if f.creations == f.failCreate {
		if f.createReturnedID {
			return id, errors.New("untrusted-provider-text")
		}
		return "", errors.New("untrusted-provider-text")
	}
	return id, nil
}
func (f *preparationEngineFixture) Inspect(ctx context.Context, kind string, r Creation, bridgeID string) (ResourceIdentity, error) {
	f.inspections++
	f.events = append(f.events, "inspect-"+kind)
	stored := f.journal.Bridge
	if kind == "anchor" {
		stored = f.journal.Anchor
	}
	if r != stored || r.Phase != "created" || r.ID == "" {
		f.t.Fatal("inspection preceded durable exact ID")
	}
	if f.inspections == f.failInspect {
		return ResourceIdentity{ID: strings.Repeat("e", 64), ContractDigest: r.ContractDigest}, nil
	}
	return ResourceIdentity{r.ID, r.ContractDigest}, ctx.Err()
}
func executionFixture(t *testing.T) (Plan, Preparation) {
	t.Helper()
	plan, err := Build(pair(), Reservations{})
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := plan.Digest()
	return plan, Preparation{ProjectID: "project-a", ProjectRevision: 1, Generation: 1, OperationID: "op-a", PlanDigest: digest, EngineID: "private-engine", Bridge: Creation{Name: "project-a-bridge", ContractDigest: strings.Repeat("1", 64), Phase: "planned"}, Anchor: Creation{Name: "project-a-anchor", ContractDigest: strings.Repeat("2", 64), Phase: "planned"}}
}

func TestExecuteProjectPreparationCommitsBeforeEachEffectAndInspection(t *testing.T) {
	plan, p := executionFixture(t)
	retained := p
	engine := &preparationEngineFixture{t: t, journal: &retained}
	var phases []string
	commit := func(sequence uint64, kind, phase, id string) error {
		next, err := retained.Advance(plan, sequence, kind, phase, id)
		if err == nil {
			retained = next
			phases = append(phases, kind+":"+phase)
		}
		return err
	}
	got, err := ExecutePreparation(context.Background(), plan, p, engine, commit)
	if err != nil || !reflect.DeepEqual(got, retained) || got.Sequence != 6 || got.Anchor.Phase != "observed" {
		t.Fatal(got, err)
	}
	if !reflect.DeepEqual(phases, []string{"bridge:dispatched", "bridge:created", "bridge:observed", "anchor:dispatched", "anchor:created", "anchor:observed"}) {
		t.Fatal(phases)
	}
	if engine.creations != 2 || engine.inspections != 2 || engine.verifications != 6 {
		t.Fatal(engine)
	}
	if _, err = ExecutePreparation(context.Background(), plan, retained, engine, commit); err == nil || engine.creations != 2 {
		t.Fatal("completed journal replayed engine effects")
	}
}

func TestExecuteProjectPreparationStopsAtEveryFailedCommit(t *testing.T) {
	for fail := uint64(0); fail < 6; fail++ {
		plan, p := executionFixture(t)
		retained := p
		engine := &preparationEngineFixture{t: t, journal: &retained}
		commit := func(sequence uint64, kind, phase, id string) error {
			if sequence == fail {
				return errors.New("synthetic commit failure")
			}
			next, err := retained.Advance(plan, sequence, kind, phase, id)
			if err == nil {
				retained = next
			}
			return err
		}
		_, err := ExecutePreparation(context.Background(), plan, p, engine, commit)
		if err == nil || retained.Sequence != fail {
			t.Fatal(fail, retained, err)
		}
		if engine.creations != []int{0, 1, 1, 1, 2, 2}[fail] || engine.inspections != []int{0, 0, 1, 1, 1, 2}[fail] {
			t.Fatal("effect crossed failed persistence", fail, engine.events)
		}
	}
}

func TestExecuteProjectPreparationRetainsUncertainOutcomesWithoutRetry(t *testing.T) {
	for _, mode := range []string{"lost-reply", "error-with-id", "inspection", "engine-changed", "canceled-after-create"} {
		t.Run(mode, func(t *testing.T) {
			plan, p := executionFixture(t)
			retained := p
			engine := &preparationEngineFixture{t: t, journal: &retained}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch mode {
			case "lost-reply":
				engine.failCreate = 1
			case "error-with-id":
				engine.failCreate = 1
				engine.createReturnedID = true
			case "inspection":
				engine.failInspect = 1
			case "engine-changed":
				engine.failEngine = 2
			case "canceled-after-create":
				engine.cancel = cancel
			}
			commit := func(sequence uint64, kind, phase, id string) error {
				next, err := retained.Advance(plan, sequence, kind, phase, id)
				if err == nil {
					retained = next
				}
				return err
			}
			got, err := ExecutePreparation(ctx, plan, p, engine, commit)
			if err == nil || strings.Contains(err.Error(), "untrusted-provider-text") || !reflect.DeepEqual(got, retained) || engine.creations != 1 {
				t.Fatal(got, err, engine.events)
			}
			want := "created"
			if mode == "lost-reply" {
				want = "dispatched"
			}
			if got.Bridge.Phase != want || got.Anchor.Phase != "planned" {
				t.Fatal(got)
			}
			if _, err = ExecutePreparation(context.Background(), plan, retained, engine, commit); err == nil || engine.creations != 1 {
				t.Fatal("uncertain create replayed")
			}
		})
	}
}

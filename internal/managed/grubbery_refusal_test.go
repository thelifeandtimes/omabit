package managed

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

type appRejectedBackend struct {
	t     *testing.T
	calls int
}

func (b *appRejectedBackend) Admit(_ context.Context, p grubbery.Plan, s grubbery.Step) (grubbery.Review, error) {
	raw, _ := json.Marshal(p.Binding)
	return grubbery.Review{Version: 1, PlanDigest: p.Digest, StepID: s.ID, BindingDigest: stopEvidenceDigest(raw), PolicyRevision: p.Binding.PolicyRevision, NativeEvidenceDigest: strings.Repeat("a", 64)}, nil
}
func (b *appRejectedBackend) Dispatch(context.Context, grubbery.Plan, grubbery.Step, grubbery.Request) (string, error) {
	b.calls++
	return strings.Repeat("b", 64), grubbery.ErrNativeRejected
}
func (b *appRejectedBackend) ObserveAfter(context.Context, grubbery.Plan, grubbery.Step) (*grubbery.Proof, error) {
	b.t.Fatal("known rejected write requested after-evidence")
	return nil, nil
}
func TestGrubberyJournalKnownNativeRefusalReopensWithoutAdoption(t *testing.T) {
	dir := privateDir(t)
	s, err := testOpenStore(dir)
	g02Check(t, err)
	r, files := grubberyFixture(t, s)
	call := grubbery.Call{RunnerID: r.Plan.Binding.RunnerID, Mode: r.Plan.Binding.Mode, Actor: r.Plan.Binding.Actor, Target: r.Plan.Binding.Target, AppID: r.Plan.Binding.AppID, RequestID: r.Plan.RequestID, PlanDigest: r.Plan.Digest, StepID: r.Plan.Steps[0].ID}
	b := &appRejectedBackend{t: t}
	e := grubbery.Execution{Store: grubberyJournal{s}, Backend: b, LoadSource: func(context.Context, string) ([]deskwork.SourceFile, error) { return files, nil }}
	_, err = e.Begin(t.Context(), call, r.Plan)
	g02Check(t, err)
	refused, err := e.Advance(t.Context(), call)
	g02Check(t, err)
	if state, _ := refused.State(call.StepID); state != "refused" || b.calls != 1 {
		t.Fatal("known refusal not retained")
	}
	g02Check(t, s.close())
	s, err = testOpenStore(dir)
	g02Check(t, err)
	defer s.close()
	e = grubbery.Execution{Store: grubberyJournal{s}}
	for _, f := range []func(context.Context, grubbery.Call) (grubbery.Record, error){e.Advance, e.Reconcile} {
		got, err := f(t.Context(), call)
		g02Check(t, err)
		if !reflect.DeepEqual(got, refused) {
			t.Fatal("reopen turned refusal into uncertainty or adoption")
		}
	}
	abandoned, err := e.Abandon(t.Context(), call)
	g02Check(t, err)
	if !abandoned.Abandoned() {
		t.Fatal("known no-effect abandonment failed")
	}
	if !reflect.DeepEqual(refused.Events, abandoned.Events[:len(refused.Events)]) {
		t.Fatal("abandonment replaced original refusal")
	}
	var leases int
	g02Check(t, s.db.QueryRow("SELECT count(*) FROM grubbery_app_leases").Scan(&leases))
	if leases != 0 {
		t.Fatal("no-effect refusal lease not released")
	}
}

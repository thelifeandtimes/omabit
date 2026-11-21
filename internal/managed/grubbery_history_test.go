package managed

import (
	"reflect"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

func TestGrubberyTrustedHistoryUsesCreationOrderAndRetainsUnfinishedStages(t *testing.T) {
	s := testStore(t)
	base, files := grubberyFixture(t, s)
	first := appNewFixture(t, base, files, base.Plan.Before, strings.Repeat("f", 64))
	j := grubberyJournal{s}
	_, err := j.Create(t.Context(), first)
	g02Check(t, err)
	call := grubbery.Call{RunnerID: first.Plan.Binding.RunnerID, Mode: first.Plan.Binding.Mode, Actor: first.Plan.Binding.Actor, Target: first.Plan.Binding.Target, AppID: first.Plan.Binding.AppID, RequestID: first.Plan.RequestID, PlanDigest: first.Plan.Digest, StepID: first.Plan.Steps[0].ID}
	first, err = (grubbery.Execution{Store: j}).Abandon(t.Context(), call)
	g02Check(t, err)
	second := appNewFixture(t, base, files, base.Plan.Before, strings.Repeat("1", 64))
	second, err = j.Create(t.Context(), second)
	g02Check(t, err)
	history, err := j.History(t.Context(), second.Plan.Binding)
	g02Check(t, err)
	if !reflect.DeepEqual(history, []grubbery.Record{first, second}) {
		t.Fatal("history used request ID order or dropped abandonment")
	}
	used, err := j.StageInUse(t.Context(), second.Plan.StageID)
	g02Check(t, err)
	if !used {
		t.Fatal("unfinished operation lost stage protection")
	}
	foreign := second.Plan.Binding
	foreign.RunnerID = "other"
	if _, err = j.History(t.Context(), foreign); err == nil {
		t.Fatal("history crossed runner identity")
	}
	foreign = second.Plan.Binding
	foreign.AppID = "other"
	history, err = j.History(t.Context(), foreign)
	g02Check(t, err)
	if len(history) != 0 {
		t.Fatal("history crossed application namespace")
	}
	call.RequestID, call.PlanDigest, call.StepID = second.Plan.RequestID, second.Plan.Digest, second.Plan.Steps[0].ID
	_, err = (grubbery.Execution{Store: j}).Abandon(t.Context(), call)
	g02Check(t, err)
	used, err = j.StageInUse(t.Context(), second.Plan.StageID)
	g02Check(t, err)
	if used {
		t.Fatal("terminal operations prevented reviewed cache discard")
	}
}

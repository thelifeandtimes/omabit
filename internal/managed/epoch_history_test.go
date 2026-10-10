package managed

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func TestEpochHistoryUsesCompletedBoundRollsOutsideActivityWindow(t *testing.T) {
	s := testStore(t)
	if _, _, err := s.register("operator", registration("alpha", "register", 18080)); err != nil {
		t.Fatal(err)
	}
	v, err := loadInstance(s.db, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	base := time.Now().UTC().Add(-time.Hour)
	add := func(name, kind, status, instance, privateID, epoch string, stale bool) {
		t.Helper()
		start, finish := base.Add(time.Second), base.Add(2*time.Second)
		op := protocol.Operation{ID: name, InstanceID: instance, Kind: kind, Status: status, Actor: "operator", RequestID: name, CreatedAt: base, FinishedAt: &finish}
		if stale {
			start = base.Add(-time.Minute)
		}
		runtime := runtimequalify.ManagedState{ID: privateID, Maintenance: &runtimequalify.ManagedMaintenance{Kind: "roll", CreatedEpoch: epoch, StartedAt: start, FinishedAt: &finish, EpochsAfter: []string{"0i0", epoch}}}
		body, _ := json.Marshal(op)
		data, _ := json.Marshal(runtime)
		if _, err := s.db.Exec("INSERT INTO operations(id,actor,request_id,fingerprint,body,runtime) VALUES(?,?,?,?,?,?)", name, "operator", name, name, body, data); err != nil {
			t.Fatal(err)
		}
		base = base.Add(time.Minute)
	}
	add("roll-old", "roll", "succeeded", "alpha", v.Runtime.ID, "0i100", false)
	add("roll-new", "roll", "succeeded", "alpha", v.Runtime.ID, "0i200", false)
	add("foreign", "roll", "succeeded", "alpha", "foreign-pier", "0i300", false)
	add("failed", "roll", "failed", "alpha", v.Runtime.ID, "0i400", false)
	add("old-report", "roll", "succeeded", "alpha", v.Runtime.ID, "0i500", true)
	add("other-instance", "roll", "succeeded", "beta", v.Runtime.ID, "0i600", false)
	for i := 0; i < 130; i++ {
		add(fmt.Sprintf("later-%d", i), "pack", "succeeded", "alpha", v.Runtime.ID, "0i200", false)
	}
	snapshot, err := s.snapshot()
	if err != nil {
		t.Fatal(err)
	}
	i := snapshot.Instances[0]
	if len(snapshot.Operations) != 128 || len(i.EpochHistory) != 2 || i.EpochHistory[0].Name != "0i200" || i.EpochHistory[1].Name != "0i100" || i.LastRoll == nil || i.LastRoll.CreatedEpoch != "0i200" {
		t.Fatalf("incorrect roll history: %+v", i)
	}
	if i.Epochs != nil {
		t.Fatal("historical metadata became a fresh inventory")
	}
	again, err := loadInstance(s.db, "alpha")
	if err != nil || again.Public.LastRoll != nil || len(again.Public.EpochHistory) != 0 {
		t.Fatal("snapshot metadata mutated stored state", err)
	}
}

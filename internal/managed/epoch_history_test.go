package managed

import (
	"encoding/json"
	"fmt"
	"reflect"
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
	if _, _, err := s.register("operator", registration("beta", "register-beta", 18081)); err != nil {
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
	if len(snapshot.Instances) != 2 || len(snapshot.Operations) != 128 || len(i.EpochHistory) != 2 || i.EpochHistory[0].Name != "0i200" || i.EpochHistory[1].Name != "0i100" || i.LastRoll == nil || i.LastRoll.CreatedEpoch != "0i200" {
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

// Compare the batched reader to the independent per-instance reader, including
// evidence from replaced runtimes and the per-instance limit before validation.
func TestEpochHistoriesMatchIndependentReaders(t *testing.T) {
	s := testStore(t)
	all := []storedInstance{
		{Public: protocol.Instance{ID: "alpha"}, Runtime: runtimequalify.ManagedState{ID: "alpha-runtime"}},
		{Public: protocol.Instance{ID: "beta"}, Runtime: runtimequalify.ManagedState{ID: "beta-runtime"}},
		{Public: protocol.Instance{ID: "gamma"}, Runtime: runtimequalify.ManagedState{ID: "gamma-runtime"}},
		{Public: protocol.Instance{ID: "missing"}, Runtime: runtimequalify.ManagedState{ID: "missing-runtime"}},
	}
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	add := func(instance, runtime, epoch, kind, status string, stale bool, rawRuntime []byte) {
		t.Helper()
		base = base.Add(time.Minute)
		start, finish := base.Add(time.Second), base.Add(2*time.Second)
		if stale {
			start = base.Add(-time.Second)
		}
		name := fmt.Sprintf("operation-%d", base.Unix())
		op := protocol.Operation{ID: name, Actor: "operator", RequestID: name, InstanceID: instance, Kind: kind, Status: status, CreatedAt: base, FinishedAt: &finish}
		body, err := json.Marshal(op)
		if err != nil {
			t.Fatal(err)
		}
		if rawRuntime == nil {
			rawRuntime, err = json.Marshal(runtimequalify.ManagedState{ID: runtime, Maintenance: &runtimequalify.ManagedMaintenance{Kind: "roll", CreatedEpoch: epoch, StartedAt: start, FinishedAt: &finish}})
			if err != nil {
				t.Fatal(err)
			}
		}
		if _, err := s.db.Exec("INSERT INTO operations(id,actor,request_id,fingerprint,body,runtime) VALUES(?,?,?,?,?,?)", name, "operator", name, name, body, rawRuntime); err != nil {
			t.Fatal(err)
		}
	}
	// Excluded runtime JSON must never be decoded, even when malformed.
	add("unselected", "", "", "roll", "succeeded", false, []byte("not-json"))
	add("alpha", "", "", "roll", "succeeded", false, []byte("not-json"))
	add("gamma", "gamma-runtime", "0i1", "roll", "succeeded", false, nil)
	for n := 1; n <= 128; n++ {
		add("alpha", "alpha-runtime", fmt.Sprintf("0i%d", n), "roll", "succeeded", false, nil)
		// The old valid gamma record is beyond the 128-record evidence window;
		// filtering by current runtime must not accidentally bring it back.
		add("gamma", "replaced-runtime", fmt.Sprintf("0i%d", n+1), "roll", "succeeded", false, nil)
	}
	add("alpha", "alpha-runtime", "0i999", "roll", "failed", false, nil)
	add("beta", "beta-runtime", "0i200", "roll", "succeeded", false, nil)
	add("beta", "beta-runtime", "0i200", "roll", "succeeded", false, nil)
	add("beta", "foreign-runtime", "0i300", "roll", "succeeded", false, nil)
	add("beta", "beta-runtime", "0i400", "roll", "succeeded", true, nil)
	add("beta", "beta-runtime", "0i500", "pack", "succeeded", false, nil)
	add("beta", "beta-runtime", "0i0", "roll", "succeeded", false, nil)
	add("beta", "beta-runtime", "bad-epoch", "roll", "succeeded", false, nil)
	add("beta", "", "", "roll", "succeeded", false, []byte(`{"id":"beta-runtime"}`))
	want := append([]storedInstance(nil), all...)
	for i := range want {
		if err := loadEpochHistory(s.db, &want[i]); err != nil {
			t.Fatal(err)
		}
	}
	if err := loadEpochHistories(s.db, all); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(all, want) {
		t.Fatalf("batch differs from independent evidence readers: got %+v, want %+v", all, want)
	}
	if len(all[0].Public.EpochHistory) != 128 || all[0].Public.LastRoll.CreatedEpoch != "0i128" || len(all[1].Public.EpochHistory) != 1 || len(all[2].Public.EpochHistory) != 0 || all[3].Public.LastRoll != nil {
		t.Fatalf("incorrect limits, ordering or validation: %+v", all)
	}
}

func TestEpochHistoriesMultipleBatchesClearDecoration(t *testing.T) {
	s := testStore(t)
	all := make([]storedInstance, 513)
	for i := range all {
		all[i] = storedInstance{Public: protocol.Instance{ID: fmt.Sprintf("instance-%d", i), EpochHistory: []protocol.EpochCreation{{Name: "0i1"}}, LastRoll: &protocol.MaintenanceReport{CreatedEpoch: "0i1"}}, Runtime: runtimequalify.ManagedState{ID: fmt.Sprintf("runtime-%d", i)}}
	}
	now := time.Now().UTC()
	for _, i := range []int{0, 511, 512} {
		id := fmt.Sprintf("operation-%d", i)
		op := protocol.Operation{ID: id, InstanceID: all[i].Public.ID, Kind: "roll", Status: "succeeded", CreatedAt: now, FinishedAt: &now}
		runtime := runtimequalify.ManagedState{ID: all[i].Runtime.ID, Maintenance: &runtimequalify.ManagedMaintenance{Kind: "roll", CreatedEpoch: "0i2", StartedAt: now, FinishedAt: &now}}
		body, _ := json.Marshal(op)
		data, _ := json.Marshal(runtime)
		if _, err := s.db.Exec("INSERT INTO operations(id,actor,request_id,fingerprint,body,runtime) VALUES(?,?,?,?,?,?)", id, "actor", id, id, body, data); err != nil {
			t.Fatal(err)
		}
	}
	if err := loadEpochHistories(s.db, all); err != nil {
		t.Fatal(err)
	}
	for i := range all {
		if i == 0 || i == 511 || i == 512 {
			if all[i].Public.LastRoll == nil || all[i].Public.LastRoll.CreatedEpoch != "0i2" || len(all[i].Public.EpochHistory) != 1 {
				t.Fatalf("missing evidence at batch boundary %d", i)
			}
		} else if all[i].Public.LastRoll != nil || all[i].Public.EpochHistory != nil {
			t.Fatalf("retained stale decoration %d", i)
		}
	}
	if err := loadEpochHistories(s.db, nil); err != nil {
		t.Fatal(err)
	}
}

func TestEpochHistoriesRejectMalformedSelectedRuntime(t *testing.T) {
	for _, runtime := range []string{"not-json", `{"id":"current","maintenance":"not-a-report"}`} {
		t.Run(runtime, func(t *testing.T) {
			s := testStore(t)
			body := []byte(`{"instance_id":"alpha","kind":"roll","status":"succeeded"}`)
			if _, err := s.db.Exec("INSERT INTO operations(id,actor,request_id,fingerprint,body,runtime) VALUES(?,?,?,?,?,?)", "op", "actor", "request", "fingerprint", body, runtime); err != nil {
				t.Fatal(err)
			}
			v := storedInstance{Public: protocol.Instance{ID: "alpha"}, Runtime: runtimequalify.ManagedState{ID: "current"}}
			if err := loadEpochHistory(s.db, &v); err == nil {
				t.Fatal("single reader accepted malformed selected evidence")
			}
			if err := loadEpochHistories(s.db, []storedInstance{v, {Public: protocol.Instance{ID: "beta"}}}); err == nil {
				t.Fatal("batch reader accepted malformed selected evidence")
			}
		})
	}
}

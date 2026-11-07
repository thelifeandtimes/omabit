package managed

import (
	"encoding/json"
	"reflect"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

func TestDeskSettingsPreservePolicyAndReceiptsAcrossReopen(t *testing.T) {
	dir := privateDir(t)
	s, err := testOpenStore(dir)
	g02Check(t, err)
	_, _, err = s.register("fixture", registration("alpha", "register", 18080))
	g02Check(t, err)
	receipt, _, err := s.createDeskOperation("agent", deskIntentFixture())
	g02Check(t, err)
	before := originalDeskRows(t, s.db)
	for _, target := range []apppolicy.Target{{InstanceID: "alpha"}, {InstanceID: "alpha", Desk: "example"}} {
		value, err := s.deskSetting(target)
		g02Check(t, err)
		if value.Revision != 0 {
			t.Fatal("defaults persisted as user settings")
		}
		update := deskwork.SettingUpdate{Target: target, Ship: value.Ship, Desk: value.Desk}
		if target.Desk == "" {
			update.Ship.Purpose = "distribution"
			update.Ship.Instructions = "Use the app's release checks"
		} else {
			update.Desk.Publication = "treaty"
			update.Desk.Checks = []string{"Frontend smoke test", "Receiver readiness"}
		}
		saved, err := s.putDeskSetting("desktop", update)
		g02Check(t, err)
		if saved.Revision != 1 || saved.Actor != "desktop" {
			t.Fatal(saved)
		}
		_, err = s.putDeskSetting("other-desktop", update)
		requireFault(t, err, "desk_settings_changed")
		update.ExpectedRevision = 1
		saved, err = s.putDeskSetting("other-desktop", update)
		g02Check(t, err)
		g02Check(t, s.close())
		s, err = testOpenStore(dir)
		g02Check(t, err)
		got, err := s.deskSetting(target)
		g02Check(t, err)
		if !reflect.DeepEqual(got, saved) {
			t.Fatal("settings lost", got, saved)
		}
	}
	defer s.close()
	if !reflect.DeepEqual(before, originalDeskRows(t, s.db)) {
		t.Fatal("settings changed policy or registry")
	}
	got, err := loadDeskOperation(s.db, receipt.Intent.ID)
	g02Check(t, err)
	if !reflect.DeepEqual(got, receipt) {
		t.Fatal("settings changed desk operation")
	}
}
func TestDeskSettingsRefuseCorruptionWithoutRepair(t *testing.T) {
	for _, query := range []string{
		"DELETE FROM desk_settings_history", "UPDATE desk_settings SET body='{}'", "UPDATE desk_settings_history SET body='{}'",
		"UPDATE desk_settings_history SET revision=7", "UPDATE desk_settings SET target='other'", "DROP TABLE desk_settings_history",
	} {
		t.Run(query, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			g02Check(t, err)
			_, _, err = s.register("fixture", registration("alpha", "register", 18080))
			g02Check(t, err)
			target := apppolicy.Target{InstanceID: "alpha", Desk: "example"}
			v := deskwork.DefaultSetting(target)
			_, err = s.putDeskSetting("desktop", deskwork.SettingUpdate{Target: target, Desk: v.Desk})
			g02Check(t, err)
			g02Check(t, s.close())
			db := g02RawDB(t, dir)
			_, err = db.Exec("PRAGMA foreign_keys=OFF")
			g02Check(t, err)
			_, err = db.Exec(query)
			g02Check(t, err)
			g02Check(t, db.Close())
			if reopened, err := testOpenStore(dir); err == nil {
				reopened.close()
				t.Fatal("corrupt settings accepted")
			}
		})
	}
}
func TestDeskSettingsUnknownTargetAndUnresolvedGenerationRefused(t *testing.T) {
	s := testStore(t)
	for _, target := range []apppolicy.Target{{InstanceID: "unknown"}, {InstanceID: "member", ProjectID: "network", Desk: "example"}} {
		v := deskwork.DefaultSetting(target)
		_, err := s.putDeskSetting("desktop", deskwork.SettingUpdate{Target: target, Ship: v.Ship, Desk: v.Desk})
		if err == nil {
			b, _ := json.Marshal(target)
			t.Fatal("unknown target accepted", string(b))
		}
	}
}

package managed

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

func sourceBaselineFixture() deskwork.SourceBaseline {
	m, _ := deskwork.NewManifest([]deskwork.File{{Path: "hello.txt", SHA256: strings.Repeat("a", 64)}})
	return deskwork.SourceBaseline{Version: 1, Binding: deskwork.SourceBinding{Target: apppolicy.Target{InstanceID: "alpha", Desk: "example"}, RuntimeID: strings.Repeat("b", 64), ConfigRevision: 1, SourceName: "example"}, Manifest: m, ObservedAt: time.Now().UTC()}
}
func TestDeskSourceBaselineReopenAndBindingIsolation(t *testing.T) {
	dir := privateDir(t)
	s, err := testOpenStore(dir)
	g02Check(t, err)
	_, _, err = s.register("fixture", registration("alpha", "register", 18080))
	g02Check(t, err)
	b := sourceBaselineFixture()
	g02Check(t, s.saveSourceBaseline(b))
	g02Check(t, s.close())
	s, err = testOpenStore(dir)
	g02Check(t, err)
	defer s.close()
	got, err := s.sourceBaseline(b.Binding)
	g02Check(t, err)
	if got == nil || !reflect.DeepEqual(*got, b) {
		t.Fatal("baseline lost", got)
	}
	for _, change := range []func(*deskwork.SourceBinding){
		func(b *deskwork.SourceBinding) { b.RuntimeID = strings.Repeat("c", 64) },
		func(b *deskwork.SourceBinding) { b.ConfigRevision++ },
		func(b *deskwork.SourceBinding) { b.SettingsRevision++ },
		func(b *deskwork.SourceBinding) { b.SourceName = "other" },
		func(b *deskwork.SourceBinding) {
			b.Target.ProjectID = "network"
			b.Target.Generation = 2
			b.Target.Creator = "other"
		},
	} {
		binding := b.Binding
		change(&binding)
		got, err := s.sourceBaseline(binding)
		g02Check(t, err)
		if got != nil {
			t.Fatal("baseline followed changed binding", binding)
		}
	}
	setting := deskwork.DefaultSetting(b.Binding.Target)
	setting.Desk.SourceName = "other"
	_, err = s.putDeskSetting("setup", deskwork.SettingUpdate{Target: b.Binding.Target, Desk: setting.Desk})
	g02Check(t, err)
	if err = s.saveSourceBaseline(b); err == nil {
		t.Fatal("baseline saved against changed workflow")
	}
}
func TestDeskSourceBaselineCorruptionRefused(t *testing.T) {
	for _, kind := range []string{"missing-table", "bad-body", "wrong-key", "wrong-hash", "unknown-field"} {
		t.Run(kind, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			g02Check(t, err)
			_, _, err = s.register("fixture", registration("alpha", "register", 18080))
			g02Check(t, err)
			b := sourceBaselineFixture()
			g02Check(t, s.saveSourceBaseline(b))
			switch kind {
			case "missing-table":
				_, err = s.db.Exec("DROP TABLE desk_source_baselines")
			case "bad-body":
				_, err = s.db.Exec("UPDATE desk_source_baselines SET body='{}'")
			case "wrong-key":
				_, err = s.db.Exec("UPDATE desk_source_baselines SET target='other'")
			case "wrong-hash":
				b.Manifest.Files[0].SHA256 = strings.Repeat("d", 64)
				body, _ := json.Marshal(b)
				_, err = s.db.Exec("UPDATE desk_source_baselines SET body=?", body)
			case "unknown-field":
				body, _ := json.Marshal(b)
				body = append(body[:len(body)-1], []byte(`,"allow_commit":true}`)...)
				_, err = s.db.Exec("UPDATE desk_source_baselines SET body=?", body)
			}
			g02Check(t, err)
			g02Check(t, s.close())
			if reopened, err := testOpenStore(dir); err == nil {
				reopened.close()
				t.Fatal("corrupt baseline accepted")
			}
		})
	}
}

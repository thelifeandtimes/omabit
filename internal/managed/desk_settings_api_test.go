package managed

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestDeskSettingsAPIIsSeparateFromPermissionAndRuntimeEffects(t *testing.T) {
	b := &deskBackendFixture{}
	r, _ := deskServerFixture(t, b)
	target := apppolicy.Target{InstanceID: "alpha", Desk: "example"}
	before, err := loadAppPolicy(r.s.db)
	g02Check(t, err)
	read := deskHTTPRequest(t, r, "POST", protocol.DeskSettingsPath, target)
	var value protocol.DeskSetting
	if read.Code != 200 || json.Unmarshal(read.Body.Bytes(), &value) != nil || value.Setting.Revision != 0 {
		t.Fatal(read.Code, read.Body.String())
	}
	update := deskwork.SettingUpdate{Target: target, Desk: value.Setting.Desk}
	update.Desk.Instructions = "Run the application tests before publishing"
	update.Desk.Publication = "source"
	saved := deskHTTPRequest(t, r, "PUT", protocol.DeskSettingsPath, update)
	if saved.Code != 200 || json.Unmarshal(saved.Body.Bytes(), &value) != nil || value.Setting.Revision != 1 || value.Setting.Actor == "" {
		t.Fatal(saved.Code, saved.Body.String())
	}
	stale := deskHTTPRequest(t, r, "PUT", protocol.DeskSettingsPath, update)
	if stale.Code != 409 {
		t.Fatal("stale edit accepted", stale.Code)
	}
	after, err := loadAppPolicy(r.s.db)
	g02Check(t, err)
	if !reflect.DeepEqual(before, after) || b.calls.Load() != 0 || b.reads.Load() != 0 {
		t.Fatal("metadata changed policy or touched runtime")
	}
}
func TestDeskSettingsAPIRejectsObserversAndUnknownFields(t *testing.T) {
	r, _ := deskServerFixture(t, &deskBackendFixture{})
	target := apppolicy.Target{InstanceID: "alpha"}
	v := deskwork.DefaultSetting(target)
	update := deskwork.SettingUpdate{Target: target, Ship: v.Ship}
	body, _ := json.Marshal(update)
	denied := httptest.NewRecorder()
	r.serveDeskSettings(denied, httptest.NewRequest("PUT", protocol.DeskSettingsPath, bytes.NewReader(body)), "observer", "observer")
	if denied.Code != 403 {
		t.Fatal("observer edited settings", denied.Code)
	}
	read := httptest.NewRecorder()
	body, _ = json.Marshal(target)
	r.serveDeskSettings(read, httptest.NewRequest("POST", protocol.DeskSettingsPath, bytes.NewReader(body)), "observer", "observer")
	if read.Code != 200 {
		t.Fatal("observer cannot read descriptive settings", read.Code)
	}
	for _, raw := range []string{`{"target":{"instance_id":"alpha"},"ship":{"purpose":"standard","instructions":""},"allow_commit":true}`, `{} {}`, `{"instance_id":"alpha","desk":"../base"}`} {
		out := httptest.NewRecorder()
		r.serveDeskSettings(out, httptest.NewRequest("PUT", protocol.DeskSettingsPath, bytes.NewBufferString(raw)), "desktop", "operator")
		if out.Code != 400 {
			t.Fatal("invalid setting accepted", out.Code, out.Body.String())
		}
	}
}

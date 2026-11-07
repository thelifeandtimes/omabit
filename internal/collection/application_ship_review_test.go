package collection

import (
	"encoding/json"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestApplicationShipApprovalRequestStrict(t *testing.T) {
	b := shipApprovalBinding()
	allow := true
	u := ApplicationShipApprovalRequest{Version: 1, Binding: b, Target: ApplicationTarget{PluginID: "fixture.plugin", Profile: "default", SHA256: strings.Repeat("a", 64)}, Allowed: &allow, Selection: &protocol.ShipRouteSelection{Target: b.Target, ConfigRevision: 1, RuntimeID: strings.Repeat("b", 64)}}
	body, _ := json.Marshal(u)
	if _, err := DecodeApplicationShipApprovalRequest(body); err != nil {
		t.Fatal(err)
	}
	for _, data := range []string{`{"version":1,` + string(body[1:]), `{"VERSION":1,` + string(body[1:]), `{"unknown":1,` + string(body[1:]), string(body) + " {}", strings.Replace(string(body), `"allowed":true`, `"allowed":null`, 1), strings.Replace(string(body), `"allowed":true,`, "", 1), strings.Repeat(" ", 16385) + string(body)} {
		if _, err := DecodeApplicationShipApprovalRequest([]byte(data)); err == nil {
			t.Fatal("ambiguous input accepted", data)
		}
	}
	u.Selection = nil
	body, _ = json.Marshal(u)
	if _, err := DecodeApplicationShipApprovalRequest(body); err == nil {
		t.Fatal("allow lacks a verified selection")
	}
	allow = false
	body, _ = json.Marshal(u)
	if _, err := DecodeApplicationShipApprovalRequest(body); err != nil {
		t.Fatal("offline revoke requires a live selection", err)
	}
}

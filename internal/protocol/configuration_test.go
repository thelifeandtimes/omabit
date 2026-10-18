package protocol

import (
	"encoding/json"
	"testing"
)

func TestConfigurationNativePresence(t *testing.T) {
	for _, tc := range []struct {
		field            string
		preserve, native bool
	}{
		{"", true, false}, {`,"native_control":false`, false, false}, {`,"native_control":true`, false, true},
	} {
		var req ConfigureRequest
		if err := json.Unmarshal([]byte(`{"expected_revision":1,"config":{"http_port":18080,"ames_port":34343,"loom":31`+tc.field+`}}`), &req); err != nil {
			t.Fatal(err)
		}
		if req.PreserveNativeControl != tc.preserve || req.Config.NativeControl != tc.native {
			t.Fatal(req)
		}
		data, err := json.Marshal(req)
		if err != nil {
			t.Fatal(err)
		}
		var again ConfigureRequest
		if err := json.Unmarshal(data, &again); err != nil || again != req {
			t.Fatal(string(data), again, err)
		}
	}
	for _, wire := range []string{`{"unknown":true}`, `{"config":{"native_contorl":true}}`, `{"config":{"native_control":"false"}}`} {
		var req ConfigureRequest
		if json.Unmarshal([]byte(wire), &req) == nil {
			t.Fatal("invalid request accepted", wire)
		}
	}
}

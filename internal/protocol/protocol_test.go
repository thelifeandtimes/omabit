package protocol

import (
	"encoding/json"
	"testing"
	"time"
)

func TestWireFormat(t *testing.T) {
	if Version != 1 || Header != "Urbit-Runner-Protocol" {
		t.Fatalf("unexpected protocol version/header: %d, %q", Version, Header)
	}
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	revision := uint64(0)
	config := Configuration{HTTPPort: 8080, AmesPort: 34343, Loom: 31}
	instance := Instance{
		ID: "demo", Label: "Demo", Config: config, ConfigRevision: 2, AppliedRevision: 1,
		DesiredState: "running", ObservedState: "stopped",
	}
	operation := Operation{
		ID: "op", RequestID: "request", InstanceID: "demo", Kind: "start", Actor: "local",
		Status: "pending", ConfigRevision: 2, CreatedAt: now,
	}
	incident := Incident{ID: "incident", InstanceID: "demo", Code: "demo", Message: "demo only", Active: true, ObservedAt: now}
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{"hello", Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "demo", Features: []string{"snapshot"}}, `{"protocol_version":1,"runner_id":"runner","mode":"demo","features":["snapshot"]}`},
		{"configuration", config, `{"http_port":8080,"ames_port":34343,"loom":31}`},
		{"instance", instance, `{"id":"demo","label":"Demo","config":{"http_port":8080,"ames_port":34343,"loom":31},"config_revision":2,"applied_revision":1,"desired_state":"running","observed_state":"stopped"}`},
		{"operation", operation, `{"id":"op","request_id":"request","instance_id":"demo","kind":"start","actor":"local","status":"pending","config_revision":2,"created_at":"2026-09-22T12:00:00Z"}`},
		{"incident", incident, `{"id":"incident","instance_id":"demo","code":"demo","message":"demo only","active":true,"observed_at":"2026-09-22T12:00:00Z"}`},
		{"snapshot", Snapshot{RunnerID: "runner", Mode: "demo", Instances: []Instance{}, Operations: []Operation{}, Incidents: []Incident{}}, `{"runner_id":"runner","mode":"demo","instances":[],"operations":[],"incidents":[]}`},
		{"configure", ConfigureRequest{ExpectedRevision: 2, Config: config}, `{"expected_revision":2,"config":{"http_port":8080,"ames_port":34343,"loom":31,"native_control":false}}`},
		{"legacy_configure", ConfigureRequest{ExpectedRevision: 2, Config: config, PreserveNativeControl: true}, `{"expected_revision":2,"config":{"http_port":8080,"ames_port":34343,"loom":31}}`},
		{"operate", OperationRequest{RequestID: "request", Kind: "start", ExpectedRevision: 2}, `{"request_id":"request","kind":"start","expected_revision":2}`},
		{"error", ErrorResponse{Error: APIError{Code: "conflict", Message: "stale revision"}}, `{"error":{"code":"conflict","message":"stale revision"}}`},
		{"zero_revision", APIError{Code: "conflict", Message: "stale revision", CurrentRevision: &revision}, `{"code":"conflict","message":"stale revision","current_revision":0}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != tt.want {
				t.Fatalf("wire JSON = %s; want %s", data, tt.want)
			}
		})
	}
}

func TestOptionalFields(t *testing.T) {
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		value any
		key   string
		want  string
	}{
		{Instance{ActiveOperationID: "op"}, "active_operation_id", `"op"`},
		{Operation{FinishedAt: &now}, "finished_at", `"2026-09-22T12:00:00Z"`},
	} {
		data, err := json.Marshal(tt.value)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(data, &fields); err != nil {
			t.Fatal(err)
		}
		if string(fields[tt.key]) != tt.want {
			t.Errorf("%s = %s; want %s", tt.key, fields[tt.key], tt.want)
		}
	}
}

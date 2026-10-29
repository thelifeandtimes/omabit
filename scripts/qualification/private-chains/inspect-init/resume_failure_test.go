package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestResumeFailureProjectionNeverExportsArbitraryValues(t *testing.T) {
	report := map[string]any{"create_dispatched": true, "start_dispatched": true, "cleanup_stop_dispatched": true, "ready": map[string]any{"private": "PRIVATE-SENTINEL"}, "error_type": "PRIVATE-SENTINEL", "error_sha256": "PRIVATE-SENTINEL", "rpc_probe": map[string]any{"read": true, "transaction_refused": true, "mining_refused": true}, "final_exit": map[string]any{"Running": false, "OOMKilled": false, "ExitCode": float64(0)}, "unknown": "PRIVATE-SENTINEL"}
	result := resumeFailureSummary(report)
	raw, _ := json.Marshal(result)
	if strings.Contains(string(raw), "PRIVATE-SENTINEL") || result["ready_recorded"] != true || result["peer_recovery_recorded"] != false || result["read_only_probe_passed"] != true || result["error_type"] != "unclassified" || result["final_exit_code"] != float64(0) {
		t.Fatal(string(raw))
	}
	report["ready"] = nil
	report["error_type"] = "RuntimeError"
	report["error_sha256"] = strings.Repeat("a", 64)
	report["final_exit"] = map[string]any{"ExitCode": -1.0, "Running": "PRIVATE-SENTINEL"}
	result = resumeFailureSummary(report)
	if result["ready_recorded"] != false || result["error_type"] != "RuntimeError" || result["error_sha256"] != strings.Repeat("a", 64) || result["final_exit_code"] != nil || result["final_running"] != nil {
		t.Fatal(result)
	}
}
func TestResumeLogProjectionFixedCodesAndLocationsOnly(t *testing.T) {
	raw := []byte("PRIVATE-SENTINEL MODULE_NOT_FOUND /gateway/docker_gateway_resume.cjs:12:9 /gateway/gateway_resume_contract.cjs:4:5 /private/SECRET:1:2 private_gateway_resume_refused")
	result := resumeLogSummary(raw, diagnosticSummary(raw)["sha256"])
	encoded, _ := json.Marshal(result)
	if strings.Contains(string(encoded), "PRIVATE-SENTINEL") || strings.Contains(string(encoded), "SECRET") || result["receipt_hash_matched"] != true || result["opaque_resume_refusal"] != true {
		t.Fatal(string(encoded))
	}
	locations := result["source_locations"].([]string)
	if len(locations) != 2 {
		t.Fatal(result)
	}
	if resumeLogSummary(raw, strings.Repeat("a", 64))["receipt_hash_matched"] != false {
		t.Fatal("changed log accepted")
	}
}

func TestGatewayPeerFailureProjection(t *testing.T) {
	report := map[string]any{"error_type": "RuntimeError", "error_sha256": strings.Repeat("b", 64), "ships": map[string]any{"zod": map[string]any{"container": strings.Repeat("a", 64), "ready": false, "cookie": "PRIVATE-SENTINEL"}, "PRIVATE-SENTINEL": map[string]any{}}, "application_state": map[string]any{"value": "PRIVATE-SENTINEL"}, "stop_records": []any{map[string]any{"passed": true, "private": "PRIVATE-SENTINEL"}}, "cleanup_errors": []any{map[string]any{"private": "PRIVATE-SENTINEL"}}}
	out := gatewayPeerFailureSummary(report)
	raw, _ := json.Marshal(out)
	if strings.Contains(string(raw), "PRIVATE-SENTINEL") || out["error_sha256"] != strings.Repeat("b", 64) || out["application_state_recorded"] != true || out["cleanup_error_count"] != 1 {
		t.Fatal(string(raw))
	}
	ships := out["ships"].(map[string]any)
	if len(ships) != 1 || ships["zod"].(map[string]any)["ready"] != false || out["stops"].(map[string]int)["passed"] != 1 {
		t.Fatal(out)
	}
	empty := gatewayPeerFailureSummary(map[string]any{})
	if empty["error_sha256"] != nil || empty["application_state_recorded"] != false || len(empty["ships"].(map[string]any)) != 0 {
		t.Fatal(empty)
	}
}

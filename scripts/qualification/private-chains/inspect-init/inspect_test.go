package main

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func previousState(t *testing.T) map[string]any { t.Helper(); return syntheticState() }
func TestAuditSyntheticState(t *testing.T) {
	p := auditState(previousState(t))
	if p["scry_observations"] != 7 || len(list(p["snapshots"])) != 6 || len(list(p["writes"])) != 4 || p["raw_values_exported"] != false {
		t.Fatal("projection differs")
	}
}
func TestAuditRejectsChangedEvidence(t *testing.T) {
	for _, kind := range []string{"state-value", "duplicate-write", "cursor", "missing-snapshot"} {
		t.Run(kind, func(t *testing.T) {
			a := previousState(t)
			switch kind {
			case "state-value":
				o := obj(list(a["observations"])[0])
				obj(o["state"])["entries"] = []any{"changed"}
			case "duplicate-write":
				w := list(a["writes"])
				obj(w[1])["entry"] = obj(w[0])["entry"]
			case "cursor":
				for _, v := range list(a["observations"]) {
					o := obj(v)
					if o["via"] == "subscription" {
						o["cursor"] = float64(0)
						break
					}
				}
			case "missing-snapshot":
				a["observations"] = list(a["observations"])[1:]
			}
			defer func() {
				if recover() == nil {
					t.Error("changed evidence accepted")
				}
			}()
			auditState(a)
		})
	}
}

func TestDiagnosticSummaryDoesNotExportPrivateValues(t *testing.T) {
	secret := "0wPRIVATE-fixture-network-key"
	value := "private-fixture-application-value"
	result := diagnosticSummary([]byte("AssertionError: " + secret + " " + value + " at /gateway.cjs:91:4"))
	raw, e := json.Marshal(result)
	if e != nil {
		t.Fatal(e)
	}
	for _, token := range []string{secret, value} {
		if bytes.Contains(raw, []byte(token)) {
			t.Fatal("private diagnostic body exported")
		}
	}
	if result["raw_message_exported"] != false || len(result["sha256"].(string)) != 64 {
		t.Fatal("missing opaque diagnostic evidence")
	}
	if !reflect.DeepEqual(result["source_locations"], []string{"/gateway.cjs:91:4"}) {
		t.Fatal("source location missing")
	}
}

func TestEvaluatorSummaryAllowsOnlyFixedScalars(t *testing.T) {
	baseline := map[string]any{"timeout_ms": 120000, "elapsed_ms": 120001, "status": nil, "signal": "SIGTERM", "error_code": "ETIMEDOUT", "stdout_bytes": 0, "stderr_bytes": 108, "raw_output_exported": false}
	raw, _ := json.Marshal(baseline)
	if evaluatorSummary(raw)["metadata"] == nil {
		t.Fatal("valid fixed metadata missing")
	}
	for _, kind := range []string{"extra", "private-code", "private-signal", "output-limit", "missing-field", "fractional-time"} {
		t.Run(kind, func(t *testing.T) {
			var v map[string]any
			json.Unmarshal(raw, &v)
			switch kind {
			case "extra":
				v["private-value"] = "0wPRIVATE-test-key"
			case "private-code":
				v["error_code"] = "0wPRIVATE-test-key"
			case "private-signal":
				v["signal"] = "0wPRIVATE-test-key"
			case "output-limit":
				v["stdout_bytes"] = 2097152
			case "missing-field":
				delete(v, "status")
			case "fractional-time":
				v["elapsed_ms"] = 1.5
			}
			data, _ := json.Marshal(v)
			r := evaluatorSummary(data)
			if r["metadata"] != nil {
				t.Fatal("unexpected metadata accepted")
			}
			encoded, _ := json.Marshal(r)
			if bytes.Contains(encoded, []byte("0wPRIVATE")) {
				t.Fatal("private text exported")
			}
		})
	}
}

func TestPeerFailureExcludesRawState(t *testing.T) {
	private := "private-application-value-0wPRIVATE"
	peers := map[string]any{"error": "RuntimeError: unexpected state: " + private, "application_state": map[string]any{"entries": []string{private}}, "ships": map[string]any{"zod": map[string]any{"ready": true, "container": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "peer_jael_life": float64(1), "extra": private}}, "attempts": []any{map[string]any{"state": "acknowledged", "nonce": private}, map[string]any{"state": private}}, "stop_records": []any{map[string]any{"passed": true, "extra": private}}}
	r := peerFailureSummary(peers)
	raw, _ := json.Marshal(r)
	if bytes.Contains(raw, []byte(private)) {
		t.Fatal("private error/state exported")
	}
	if r["reason"] != "application-state-mismatch" || r["raw_state_exported"] != false {
		t.Fatal("reason/projection differs")
	}
	counts := r["messages"].(map[string]int)
	if counts["acknowledged"] != 1 || counts["unknown"] != 1 {
		t.Fatal("message projection differs")
	}
	for _, message := range []string{"RuntimeError: fresh keyed container readiness deadline", "RuntimeError: private HTTP request failed; curl_exit=7", "unclassified " + private} {
		v := peerFailureSummary(map[string]any{"error": message})
		b, _ := json.Marshal(v)
		if bytes.Contains(b, []byte(private)) {
			t.Fatal("unknown diagnostic exported")
		}
		if v["reason"] == "unclassified" && !strings.HasPrefix(message, "unclassified") {
			t.Fatal("known reason missing")
		}
	}
}

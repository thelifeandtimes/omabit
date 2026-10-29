package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func syntheticResume() (map[string]any, map[string]any, map[string]any, map[string]any, string) {
	s, _ := syntheticResults()
	s["anchor"] = strings.Repeat("b", 64)
	original := obj(s["gateway_proof"])
	proofSHA := strings.Repeat("c", 64)
	observation := map[string]any{"chain_id": "0x539", "genesis": obj(s["ready"])["genesis"], "head": "0x50", "head_hash": "0x" + strings.Repeat("d", 64), "public_keys": original["publicKeys"]}
	base := func() map[string]any {
		return map[string]any{"version": float64(1), "mode": resumeMode, "project": s["project"], "operation": strings.Repeat("e", 32), "previous_gateway": s["anchor"], "read_only_rpc": true, "initialization_performed": false}
	}
	ready := base()
	ready["observation"] = observation
	result := base()
	result["passed"] = true
	result["signal_shutdown"] = true
	result["before"] = observation
	result["after"] = observation
	result["method_counts"] = map[string]any{"eth_chainId": float64(3), "eth_blockNumber": float64(4), "eth_getBlockByNumber": float64(4), "eth_getCode": float64(18), "eth_call": float64(8)}
	r := base()
	r["ok"] = true
	r["container"] = strings.Repeat("f", 64)
	r["initial_result_sha256"] = proofSHA
	r["create_dispatched"] = true
	r["start_dispatched"] = true
	r["stop_dispatched"] = true
	r["cleanup_errors"] = []any{}
	r["chain_identity"] = []any{float64(23), float64(345)}
	r["rpc_probe"] = map[string]any{"read": true, "transaction_refused": true, "mining_refused": true}
	r["ready"] = ready
	r["result"] = result
	r["final_exit"] = map[string]any{"Running": false, "ExitCode": float64(0), "OOMKilled": false}
	r["diagnostic_logs_sha256"] = strings.Repeat("a", 64)
	g := map[string]any{"ok": true, "ship_operations": true, "gateway_resume": map[string]any{"ok": true, "project": s["project"], "operation": r["operation"], "previous_gateway": s["anchor"], "initialization_performed": false, "cleanup_errors": []any{}}}
	// Separate JSON receipt objects as on disk, so mutation tests cannot accidentally
	// change all shared references together.
	var all []map[string]any
	raw, _ := json.Marshal([]any{s, r, g, original})
	_ = json.Unmarshal(raw, &all)
	return all[0], all[1], all[2], all[3], proofSHA
}

func TestResumeAuditProjectsOnlyPublicEvidence(t *testing.T) {
	s, r, g, p, h := syntheticResume()
	secret := "0wPRIVATE-synthetic-value"
	r["extra"] = secret
	obj(r["rpc_probe"])["extra"] = secret
	obj(r["final_exit"])["extra"] = secret
	obj(r["result"])["extra"] = secret
	out := auditResume(s, r, g, p, h)
	raw, e := json.Marshal(out)
	if e != nil || bytes.Contains(raw, []byte(secret)) || out["ok"] != true || out["raw_state_exported"] != false {
		t.Fatal("invalid public projection")
	}
}

func TestResumeAuditRejectsChangedProofs(t *testing.T) {
	for _, kind := range []string{"project", "operation", "previous", "same-container", "proof-hash", "proof-body", "create", "start", "stop", "initialization", "cleanup", "error", "chain-id", "head", "head-hash", "genesis", "keys", "after", "ready", "mining", "write-rpc", "missing-contract-check", "exit", "oom", "guest", "guest-operation", "inode", "mode", "signal", "missing-keys"} {
		t.Run(kind, func(t *testing.T) {
			s, r, g, p, h := syntheticResume()
			result := obj(r["result"])
			before := obj(result["before"])
			switch kind {
			case "project":
				r["project"] = strings.Repeat("d", 32)
			case "operation":
				r["operation"] = strings.Repeat("d", 32)
			case "previous":
				r["previous_gateway"] = strings.Repeat("d", 64)
			case "same-container":
				r["container"] = s["anchor"]
			case "proof-hash":
				h = strings.Repeat("d", 64)
			case "proof-body":
				p["extra"] = true
			case "create":
				r["create_dispatched"] = false
			case "start":
				r["start_dispatched"] = false
			case "stop":
				r["stop_dispatched"] = false
			case "initialization":
				result["initialization_performed"] = true
			case "cleanup":
				r["cleanup_errors"] = []any{"unknown"}
			case "error":
				r["error_sha256"] = strings.Repeat("d", 64)
			case "chain-id":
				before["chain_id"] = "0x1"
			case "head":
				before["head"] = "0x51"
			case "head-hash":
				before["head_hash"] = "bad"
			case "genesis":
				before["genesis"] = "0x" + strings.Repeat("d", 64)
			case "keys":
				obj(list(before["public_keys"])[0])["auth"] = "0x" + strings.Repeat("d", 64)
			case "after":
				obj(result["after"])["head"] = "0x51"
			case "ready":
				obj(obj(r["ready"])["observation"])["head"] = "0x51"
			case "mining":
				obj(r["rpc_probe"])["mining_refused"] = false
			case "write-rpc":
				obj(result["method_counts"])["eth_sendTransaction"] = float64(1)
			case "missing-contract-check":
				obj(result["method_counts"])["eth_getCode"] = float64(1)
			case "exit":
				obj(r["final_exit"])["ExitCode"] = float64(1)
			case "oom":
				obj(r["final_exit"])["OOMKilled"] = true
			case "guest":
				g["ok"] = false
			case "guest-operation":
				obj(g["gateway_resume"])["operation"] = strings.Repeat("d", 32)
			case "inode":
				r["chain_identity"] = []any{float64(1), float64(1.5)}
			case "mode":
				result["mode"] = "initializer"
			case "signal":
				result["signal_shutdown"] = false
			case "missing-keys":
				before["public_keys"] = []any{}
			}
			defer func() {
				if recover() == nil {
					t.Error("changed proof accepted")
				}
			}()
			auditResume(s, r, g, p, h)
		})
	}
}

func TestResumeAuditMarkerCannotBeDowngraded(t *testing.T) {
	if resumeRequired("console=ttyS0") || !resumeRequired("console=ttyS0 "+resumeMarker) {
		t.Fatal("marker selection differs")
	}
	for _, value := range []string{resumeMarker + " " + resumeMarker, "omabit.gateway-resume-audit=false", "omabit.gateway-resume-audit="} {
		t.Run(value, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("invalid selection accepted")
				}
			}()
			resumeRequired(value)
		})
	}
}

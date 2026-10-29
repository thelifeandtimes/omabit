package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"golang.org/x/sys/unix"
)

const resumeMode = "private-chain-docker-gateway-resume-v1"
const resumeMarker = "omabit.gateway-resume-audit=required-v1"

func resumeRequired(cmdline string) bool {
	count := 0
	for _, word := range strings.Fields(cmdline) {
		if strings.HasPrefix(word, "omabit.gateway-resume-audit=") {
			need(word == resumeMarker, "unknown resume audit mode")
			count++
		}
	}
	need(count <= 1, "duplicate resume audit mode")
	return count == 1
}

// Fixed paths derived from validated fixture IDs only. No raw content leaves the
// reader. Refuse special files, links and oversized or changing public receipts.
func readPublicReceipt(path string) (map[string]any, string) {
	fd, e := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	must(e)
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	var st unix.Stat_t
	must(unix.Fstat(fd, &st))
	need(st.Mode&unix.S_IFMT == unix.S_IFREG && st.Nlink == 1 && st.Size <= 4*1024*1024, "invalid public receipt file")
	raw, e := io.ReadAll(io.LimitReader(f, 4*1024*1024+1))
	must(e)
	need(int64(len(raw)) == st.Size, "public receipt size changed")
	var value map[string]any
	must(json.Unmarshal(raw, &value))
	digest := sha256.Sum256(raw)
	return value, hex.EncodeToString(digest[:])
}

func positiveInteger(v any) float64 {
	n, ok := v.(float64)
	need(ok && n > 0 && n < 9007199254740992 && n == float64(int64(n)), "invalid positive metadata")
	return n
}

func auditResume(summary, report, guest, original map[string]any, originalSHA string) map[string]any {
	project := hexValue(summary["project"], 32, "")
	previous := hexValue(summary["anchor"], 64, "")
	operation := hexValue(report["operation"], 32, "")
	container := hexValue(report["container"], 64, "")
	need(previous != container, "resume container reused")
	need(report["version"] == float64(1) && report["ok"] == true && report["project"] == project && report["previous_gateway"] == previous, "resume identity/result differs")
	need(report["initial_result_sha256"] == hexValue(originalSHA, 64, "") && reflect.DeepEqual(summary["gateway_proof"], original), "original gateway proof differs")
	for _, field := range []string{"create_dispatched", "start_dispatched", "stop_dispatched"} {
		need(report[field] == true, "resume dispatch missing")
	}
	need(report["initialization_performed"] == false && len(list(report["cleanup_errors"])) == 0, "resume initialization or cleanup differs")
	need(report["error_type"] == nil && report["error_sha256"] == nil, "resume contains failure")
	chain := list(report["chain_identity"])
	need(len(chain) == 2, "chain identity missing")
	chainPublic := []any{positiveInteger(chain[0]), positiveInteger(chain[1])}
	probe := obj(report["rpc_probe"])
	for _, field := range []string{"read", "transaction_refused", "mining_refused"} {
		need(probe[field] == true, "read-only RPC probe missing")
	}
	ready, result := obj(report["ready"]), obj(report["result"])
	for _, v := range []map[string]any{ready, result} {
		need(v["version"] == float64(1) && v["mode"] == resumeMode && v["project"] == project && v["operation"] == operation && v["previous_gateway"] == previous, "resume receipt binding differs")
		need(v["read_only_rpc"] == true && v["initialization_performed"] == false, "resume receipt permits initialization or mutations")
	}
	need(result["passed"] == true && result["signal_shutdown"] == true, "resume shutdown proof missing")
	before := obj(result["before"])
	need(reflect.DeepEqual(before, result["after"]) && reflect.DeepEqual(before, ready["observation"]), "resume observations differ")
	need(before["chain_id"] == "0x539" && before["head"] == "0x50" && before["genesis"] == obj(summary["ready"])["genesis"], "resume chain differs")
	need(reflect.DeepEqual(before["public_keys"], original["publicKeys"]), "resume public keys changed")
	originalHeadMatched := false
	if originalHead := obj(original["gateway"])["headHash"]; originalHead != nil {
		need(before["head_hash"] == hexValue(originalHead, 64, "0x"), "original retained head hash changed")
		originalHeadMatched = true
	}
	publicKeys := []any{}
	for n, key := range list(before["public_keys"]) {
		need(n < 2, "unexpected public key")
		k := obj(key)
		ship := []string{"~zod", "~nec"}[n]
		need(k["number"] == float64(n) && k["ship"] == ship, "public key binding differs")
		publicKeys = append(publicKeys, map[string]any{"number": n, "ship": ship, "auth": hexValue(k["auth"], 64, "0x"), "crypt": hexValue(k["crypt"], 64, "0x")})
	}
	need(len(publicKeys) == 2, "missing public key")
	counts := obj(result["method_counts"])
	minimum := map[string]float64{"eth_chainId": 3, "eth_blockNumber": 4, "eth_getBlockByNumber": 4, "eth_getCode": 18, "eth_call": 8}
	allowed := map[string]bool{}
	for _, s := range []string{"web3_clientVersion", "net_version", "eth_chainId", "eth_blockNumber", "eth_getLogs", "eth_getBlockByNumber", "eth_getBlockByHash", "eth_getTransactionReceipt", "eth_getTransactionByHash", "eth_call", "eth_getCode"} {
		allowed[s] = true
	}
	publicCounts := map[string]any{}
	for name, value := range counts {
		need(allowed[name], "resume used unqualified RPC method")
		publicCounts[name] = positiveInteger(value)
	}
	for name, n := range minimum {
		need(positiveInteger(counts[name]) >= n, "resume chain checks missing")
	}
	need(guest["ok"] == true && guest["ship_operations"] == true, "guest resume case did not pass")
	g := obj(guest["gateway_resume"])
	need(g["ok"] == true && g["project"] == project && g["operation"] == operation && g["previous_gateway"] == previous && g["initialization_performed"] == false && len(list(g["cleanup_errors"])) == 0, "guest resume summary differs")
	return map[string]any{"ok": true, "mode": resumeMode, "project": project, "operation": operation, "previous_gateway": previous, "container": container, "chain_identity": chainPublic, "original_result_sha256": originalSHA, "initialization_performed": false, "original_head_hash_matched": originalHeadMatched, "read_only_rpc": true, "rpc_probe": map[string]any{"read": true, "transaction_refused": true, "mining_refused": true}, "observation": map[string]any{"chain_id": "0x539", "genesis": hexValue(before["genesis"], 64, "0x"), "head": "0x50", "head_hash": hexValue(before["head_hash"], 64, "0x"), "public_keys": publicKeys}, "method_counts": publicCounts, "clean_shutdown": cleanExit(report["final_exit"]), "diagnostic_logs_sha256": hexValue(report["diagnostic_logs_sha256"], 64, ""), "raw_state_exported": false}
}

func inspectResume(summary, peers map[string]any, requirePeers bool) map[string]any {
	project := hexValue(summary["project"], 32, "")
	if _, err := os.Lstat("/newroot/opt/omabit-lab/results/gateway-resume.json"); os.IsNotExist(err) {
		guest, guestSHA := readResult("result.json")
		raw, _ := json.Marshal(guest["error"])
		root := "/newroot/home/lab/omabit-private-chain-" + project
		return map[string]any{"ok": false, "required": true, "reason": "resume-report-not-created", "guest_result_sha256": guestSHA, "diagnostic": diagnosticSummary(raw), "directories": map[string]any{"project": directoryMetadata(root), "chain": directoryMetadata(root + "/chain")}, "raw_state_exported": false}
	}
	report, digest := readResult("gateway-resume.json")
	guest, guestSHA := readResult("result.json")
	if report["ok"] != true {
		// Preserve failed-trial evidence without reflecting arbitrary messages/state.
		raw, _ := json.Marshal(report)
		return map[string]any{"ok": false, "required": true, "report_sha256": digest, "guest_result_sha256": guestSHA, "diagnostic": diagnosticSummary(raw), "failure_boundary": resumeFailureSummary(report), "gateway_log": resumeFailureLog(report), "peer_failure": inspectGatewayPeerFailure(), "raw_state_exported": false}
	}
	root := "/newroot/home/lab/omabit-private-chain-" + project
	original, originalSHA := readPublicReceipt(root + "/result.json")
	out := auditResume(summary, report, guest, original, originalSHA)
	if requirePeers {
		need(out["original_head_hash_matched"] == true, "peer recovery requires original head hash")
		out["peer_recovery"] = inspectGatewayPeers(peers, report)
	}
	var st unix.Stat_t
	must(unix.Lstat(root+"/chain", &st))
	need(st.Mode&unix.S_IFMT == unix.S_IFDIR && st.Mode&0777 == 0700 && st.Uid == 1000, "retained chain directory differs")
	chain := list(report["chain_identity"])
	need(chain[0] == float64(st.Dev) && chain[1] == float64(st.Ino), "retained chain object changed")
	operation := hexValue(report["operation"], 32, "")
	work := "/newroot/home/lab/omabit-private-resume-" + project + "-" + operation
	retainedReady, readySHA := readPublicReceipt(work + "/resume-ready.json")
	retainedResult, resultSHA := readPublicReceipt(work + "/resume-result.json")
	need(reflect.DeepEqual(retainedReady, report["ready"]) && reflect.DeepEqual(retainedResult, report["result"]), "retained resume receipts differ")
	out["report_sha256"] = digest
	out["guest_result_sha256"] = guestSHA
	out["ready_sha256"] = readySHA
	out["result_sha256"] = resultSHA
	return out
}

// Public metadata for the exact two fixture directories only; never export a
// symlink target, arbitrary path, file listing or file content.
func directoryMetadata(path string) map[string]any {
	var st unix.Stat_t
	if err := unix.Lstat(path, &st); err != nil {
		return map[string]any{"present": false}
	}
	resolved, err := filepath.EvalSymlinks(path)
	return map[string]any{"present": true, "directory": st.Mode&unix.S_IFMT == unix.S_IFDIR, "symlink": st.Mode&unix.S_IFMT == unix.S_IFLNK, "canonical": err == nil && resolved == path, "mode": st.Mode & 0777, "uid": st.Uid, "gid": st.Gid, "device": st.Dev, "inode": st.Ino}
}

// Failure diagnostics are fixed projections, never a successful lifecycle proof.
func resumeFailureSummary(report map[string]any) map[string]any {
	out := map[string]any{"raw_message_exported": false}
	for _, field := range []string{"create_dispatched", "start_dispatched", "stop_dispatched", "cleanup_stop_dispatched", "initialization_performed"} {
		out[field] = report[field] == true
	}
	for _, field := range []string{"ready", "result", "peer_recovery"} {
		_, ok := report[field].(map[string]any)
		out[field+"_recorded"] = ok
	}
	kind, _ := report["error_type"].(string)
	switch kind {
	case "RuntimeError", "ValueError", "TimeoutExpired", "FileNotFoundError", "PermissionError":
		out["error_type"] = kind
	default:
		out["error_type"] = "unclassified"
	}
	if value, ok := report["error_sha256"].(string); ok && len(value) == 64 {
		if _, err := hex.DecodeString(value); err == nil && value == strings.ToLower(value) {
			out["error_sha256"] = value
		}
	}
	if probe, ok := report["rpc_probe"].(map[string]any); ok {
		out["read_only_probe_passed"] = probe["read"] == true && probe["transaction_refused"] == true && probe["mining_refused"] == true
	}
	if state, ok := report["final_exit"].(map[string]any); ok {
		if b, ok := state["Running"].(bool); ok {
			out["final_running"] = b
		}
		if b, ok := state["OOMKilled"].(bool); ok {
			out["final_oom"] = b
		}
		if n, ok := state["ExitCode"].(float64); ok && n >= 0 && n <= 255 && n == float64(int(n)) {
			out["final_exit_code"] = n
		}
	}
	return out
}

func resumeLogSummary(raw []byte, expected any) map[string]any {
	out := diagnosticSummary(raw)
	out["receipt_hash_matched"] = out["sha256"] == expected
	out["opaque_resume_refusal"] = strings.Contains(string(raw), "private_gateway_resume_refused")
	return out
}

func resumeFailureLog(report map[string]any) map[string]any {
	const path = "/newroot/opt/omabit-lab/results/gateway-resume.log"
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return map[string]any{"available": false}
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	var st unix.Stat_t
	if unix.Fstat(fd, &st) != nil || st.Mode&unix.S_IFMT != unix.S_IFREG || st.Nlink != 1 || st.Size < 0 || st.Size > 4*1024*1024 {
		return map[string]any{"available": false}
	}
	raw, err := io.ReadAll(io.LimitReader(f, 4*1024*1024+1))
	if err != nil || int64(len(raw)) != st.Size {
		return map[string]any{"available": false}
	}
	out := resumeLogSummary(raw, report["diagnostic_logs_sha256"])
	out["available"] = true
	return out
}

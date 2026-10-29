package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
)

func need(ok bool, reason string) {
	if !ok {
		panic(reason)
	}
}
func obj(v any) map[string]any { m, ok := v.(map[string]any); need(ok, "object required"); return m }
func list(v any) []any         { a, ok := v.([]any); need(ok, "array required"); return a }
func hash(v any) string {
	b, e := json.Marshal(v)
	must(e)
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}
func readResult(name string) (map[string]any, string) {
	f, e := os.Open(filepath.Join("/newroot/opt/omabit-lab/results", name))
	must(e)
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, 4*1024*1024+1))
	must(e)
	need(len(b) <= 4*1024*1024, "result size exceeded")
	var m map[string]any
	must(json.Unmarshal(b, &m))
	s := sha256.Sum256(b)
	return m, hex.EncodeToString(s[:])
}
func emit(v any) {
	b, e := json.Marshal(v)
	must(e)
	fmt.Println("OMABIT_PRIVATE_INSPECTION " + string(b))
}
func inspectResults(requireResume, requirePeers, requirePublication, requireInitial bool) {
	if _, err := os.Stat("/newroot/opt/omabit-lab/results/private-chain.json"); os.IsNotExist(err) {
		inspectBootstrap()
		return
	}
	summary, summarySHA := readResult("private-chain.json")
	project := hexValue(summary["project"], 32, "")
	if summary["ok"] != true {
		raw, _ := json.Marshal(map[string]any{"error": summary["error"], "cleanup_errors": summary["cleanup_errors"]})
		out := map[string]any{"schema": 1, "private_chain_sha256": summarySHA, "project": project, "ok": false, "read_only_disk": true, "raw_state_exported": false, "diagnostic": diagnosticSummary(raw)}
		f, e := os.Open("/newroot/home/lab/omabit-private-chain-" + project + "/failure.txt")
		if e == nil {
			b, _ := io.ReadAll(io.LimitReader(f, 4096))
			f.Close()
			out["gateway_failure"] = diagnosticSummary(b)
		}
		inspectGatewayLog(out, project)
		inspectEvaluatorLog(out, project)
		if peers, sha := optionalPeers(); peers != nil {
			out["peers_sha256"] = sha
			out["peer_failure"] = peerFailureSummary(peers)
		}
		emit(out)
		return
	}
	peers, peersSHA := readResult("private-chain-peers.json")
	out := auditResults(summary, peers)
	out["private_chain_sha256"] = summarySHA
	out["peers_sha256"] = peersSHA
	if requireResume {
		resume := inspectResume(summary, peers, requirePeers)
		out["gateway_resume"] = resume
		out["ok"] = resume["ok"] == true
	}
	if requirePublication && out["ok"] == true {
		out["gateway_publication"] = inspectPublication()
	}
	if requireInitial && out["ok"] == true {
		out["initial_publication"] = inspectInitialPublication(summary, peers)
	}
	emit(out)
}
func optionalPeers() (m map[string]any, s string) {
	defer func() {
		if recover() != nil {
			m = nil
		}
	}()
	return readResult("private-chain-peers.json")
}

func auditState(app map[string]any) map[string]any {
	for _, k := range []string{"ok", "state_survived", "queued_update_survived", "existing_subscription_survived", "fresh_subscription_reconciled"} {
		need(app[k] == true, "application recovery gate missing")
	}
	need(app["writes_replayed"] == false, "write replay flag differs")
	writes := list(app["writes"])
	need(len(writes) == 4, "write count differs")
	phases := []string{"connected", "disconnected-before-restart", "existing-subscription-after-restart", "fresh-subscription-after-restart"}
	entries := []any{}
	writeMeta := []any{}
	seenEntries := map[any]bool{}
	for n, v := range writes {
		w := obj(v)
		need(w["state"] == "acknowledged" && w["http_status"] == float64(200) && w["phase"] == phases[n], "write outcome differs")
		entry, ok := w["entry"].(string)
		need(ok && !seenEntries[entry], "write identity repeated")
		seenEntries[entry] = true
		entries = append(entries, entry)
		writeMeta = append(writeMeta, map[string]any{"phase": w["phase"], "entry_sha256": hash(entry), "receipt_sha256": hash(w["receipt"]), "response_sha256": hexValue(w["response_sha256"], 64, "")})
	}
	channels := list(app["channels"])
	for _, v := range channels {
		channel, ok := v.(string)
		need(ok && regexp.MustCompile(`^/~/channel/omabit-state-[0-9a-f]{30,32}$`).MatchString(channel), "channel format differs")
	}
	need(len(channels) == 2 && channels[0] != channels[1] && reflect.DeepEqual(channels, app["channels_deleted"]), "channel cleanup differs")
	observations := list(app["observations"])
	need(len(observations) == 13, "observation count differs")
	snapshots := []any{}
	scries := 0
	subscriptions := 0
	last := map[any]float64{}
	expectedRevisions := []float64{0, 1, 2, 3, 3, 4}
	for _, v := range observations {
		o := obj(v)
		state := obj(o["state"])
		rev, ok := state["revision"].(float64)
		need(ok && rev >= 0 && rev <= 4 && rev == float64(int(rev)), "state revision invalid")
		need(reflect.DeepEqual(state["entries"], entries[:int(rev)]), "durable application state differs")
		if o["via"] == "scry" {
			scries++
			continue
		}
		need(o["via"] == "subscription" && subscriptions < 6 && rev == expectedRevisions[subscriptions], "subscription revision differs")
		channel := channels[0]
		if subscriptions >= 4 {
			channel = channels[1]
		}
		need(o["channel"] == channel, "subscription binding changed")
		events := list(o["events"])
		facts := 0
		ack := false
		cursor, seen := last[channel]
		if !seen {
			cursor = -1
		}
		for _, pair := range events {
			values := list(pair)
			need(len(values) == 2, "event pair differs")
			id, ok := values[0].(float64)
			need(ok && id > cursor, "event replay/order differs")
			cursor = id
			event := obj(values[1])
			need(event["response"] != "quit" && event["err"] == nil, "subscription failed")
			if event["response"] == "diff" {
				need(event["id"] == float64(1) && reflect.DeepEqual(event["json"], state), "snapshot event differs")
				facts++
			}
			if event["response"] == "subscribe" {
				need(event["id"] == float64(1) && event["ok"] == "ok", "subscribe acknowledgment differs")
				ack = true
			}
		}
		need(facts == 1 && o["cursor"] == cursor, "snapshot/cursor missing")
		if subscriptions == 0 || subscriptions == 4 {
			need(ack, "subscription acknowledgment missing")
		}
		last[channel] = cursor
		snapshots = append(snapshots, map[string]any{"revision": rev, "state_sha256": hash(state), "channel": channel, "cursor": cursor, "events": len(events)})
		subscriptions++
	}
	need(scries == 7 && subscriptions == 6, "state observation totals differ")
	need(app["resume_channel"] == channels[0] && app["resume_cursor"] == obj(snapshots[1])["cursor"], "saved resume cursor differs")
	return map[string]any{"writes": writeMeta, "snapshots": snapshots, "scry_observations": scries, "channels_deleted": channels, "fixture_sha256": hexValue(app["fixture_sha256"], 64, ""), "resume_cursor": app["resume_cursor"], "raw_values_exported": false}
}

func inspectBootstrap() {
	report, digest := readResult("result.json")
	need(report["ship_operations"] == false, "bootstrap reader cannot admit ship operations")
	out := map[string]any{"schema": 1, "ok": false, "bootstrap_only": true, "guest_result_sha256": digest, "ship_operations": false, "raw_state_exported": false, "read_only_disk": true}
	raw, _ := json.Marshal(report)
	out["bootstrap_diagnostic"] = diagnosticSummary(raw)
	emit(out)
}

// Failure text can contain a rejected Hoon expression with a private network key.
// Only digests, byte counts, fixed error codes and known source locations leave
// the stopped private disk. Never return the message or arbitrary stack text.
func diagnosticSummary(b []byte) map[string]any {
	digest := sha256.Sum256(b)
	text := string(b)
	codes := []string{}
	for _, code := range []string{"ERR_DLOPEN_FAILED", "MODULE_NOT_FOUND", "ENOENT", "EACCES", "AssertionError", "SyntaxError", "TypeError"} {
		if regexp.MustCompile(`\b` + code + `\b`).MatchString(text) {
			codes = append(codes, code)
		}
	}
	return map[string]any{"sha256": hex.EncodeToString(digest[:]), "bytes": len(b), "codes": codes, "source_locations": regexp.MustCompile(`/(?:gateway[.]cjs|gateway/(?:docker_gateway_resume|gateway_resume_contract)[.]cjs):[0-9]+:[0-9]+`).FindAllString(text, 16), "raw_message_exported": false}
}

func inspectGatewayLog(out map[string]any, project string) {
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(project) {
		return
	}
	root := "/newroot/home/lab/omabit-private-chain-" + project
	_, secretErr := os.Stat(root + "/secrets")
	out["secret_directory_present"] = secretErr == nil
	f, err := os.Open("/newroot/opt/omabit-lab/results/gateway.log")
	if err != nil {
		return
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, 8192))
	if err != nil {
		return
	}
	out["gateway_log_diagnostic"] = diagnosticSummary(raw)

}

// Read only the fixed scalar fields emitted by the gateway's evaluator wrapper.
// Unknown fields or values produce an opaque diagnostic, never raw JSON.
func evaluatorSummary(raw []byte) map[string]any {
	opaque := diagnosticSummary(raw)
	var v map[string]any
	if json.Unmarshal(raw, &v) != nil || len(v) != 8 {
		return opaque
	}
	for _, name := range []string{"timeout_ms", "elapsed_ms", "status", "signal", "error_code", "stdout_bytes", "stderr_bytes", "raw_output_exported"} {
		if _, ok := v[name]; !ok {
			return opaque
		}
	}
	if v["raw_output_exported"] != false {
		return opaque
	}
	if v["timeout_ms"] != float64(30000) && v["timeout_ms"] != float64(120000) {
		return opaque
	}
	for name, max := range map[string]float64{"elapsed_ms": 300000, "stdout_bytes": 1048576, "stderr_bytes": 1048576} {
		n, ok := v[name].(float64)
		if !ok || n < 0 || n > max || n != float64(int64(n)) {
			return opaque
		}
	}
	if v["status"] != nil {
		n, ok := v["status"].(float64)
		if !ok || n < 0 || n > 255 || n != float64(int64(n)) {
			return opaque
		}
	}
	if v["signal"] != nil && v["signal"] != "SIGTERM" && v["signal"] != "SIGPIPE" && v["signal"] != "SIGKILL" {
		return opaque
	}
	if v["error_code"] != nil && v["error_code"] != "ETIMEDOUT" && v["error_code"] != "ENOBUFS" && v["error_code"] != "EACCES" && v["error_code"] != "ENOENT" && v["error_code"] != "EIO" {
		return opaque
	}
	opaque["metadata"] = v
	return opaque
}
func inspectEvaluatorLog(out map[string]any, project string) {
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(project) {
		return
	}
	f, e := os.Open("/newroot/home/lab/omabit-private-chain-" + project + "/evaluator-failure.json")
	if e != nil {
		return
	}
	defer f.Close()
	raw, e := io.ReadAll(io.LimitReader(f, 4097))
	if e != nil || len(raw) > 4096 {
		return
	}
	out["evaluator_failure"] = evaluatorSummary(raw)
}

// A transport exception may embed an application snapshot. Preserve only an
// opaque error fingerprint, known reason and bounded public progress counts.
func peerFailureSummary(peers map[string]any) map[string]any {
	text, _ := peers["error"].(string)
	out := map[string]any{"diagnostic": diagnosticSummary([]byte(text)), "reason": "unclassified", "raw_state_exported": false}
	reasons := map[string]string{"RuntimeError: fresh keyed container readiness deadline": "readiness-deadline", "RuntimeError: keyed container exited before readiness": "container-exited", "RuntimeError: private peer key observation deadline": "peer-key-deadline", "RuntimeError: private SSE observation deadline": "subscription-deadline", "RuntimeError: private SSE reconnect refused": "subscription-refused", "RuntimeError: administrative identity mismatch": "identity-mismatch"}
	if reason, ok := reasons[text]; ok {
		out["reason"] = reason
	} else if strings.HasPrefix(text, "RuntimeError: unexpected state: ") {
		out["reason"] = "application-state-mismatch"
	} else if regexp.MustCompile(`^RuntimeError: private HTTP request failed; curl_exit=[0-9]{1,3}$`).MatchString(text) {
		out["reason"] = "http-probe-failed"
	}
	ships := map[string]any{}
	if value, ok := peers["ships"].(map[string]any); ok {
		for _, name := range []string{"zod", "nec"} {
			if ship, ok := value[name].(map[string]any); ok {
				row := map[string]any{"ready": ship["ready"] == true, "peer_key_observed": ship["peer_jael_life"] == float64(1)}
				if id, ok := ship["container"].(string); ok && regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(id) {
					row["container"] = id
				}
				ships[name] = row
			}
		}
	}
	out["ships"] = ships
	counts := map[string]int{"acknowledged": 0, "rejected": 0, "pending": 0, "unknown": 0}
	if attempts, ok := peers["attempts"].([]any); ok {
		out["attempts"] = len(attempts)
		for _, v := range attempts {
			a, ok := v.(map[string]any)
			if !ok {
				counts["unknown"]++
				continue
			}
			state, _ := a["state"].(string)
			if _, ok := counts[state]; !ok {
				state = "unknown"
			}
			counts[state]++
		}
	}
	out["messages"] = counts
	total, passed := 0, 0
	if stops, ok := peers["stop_records"].([]any); ok {
		total = len(stops)
		for _, v := range stops {
			if s, ok := v.(map[string]any); ok && s["passed"] == true {
				passed++
			}
		}
	}
	out["stops"] = map[string]int{"total": total, "passed": passed}
	return out
}

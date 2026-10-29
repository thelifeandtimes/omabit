package main

import (
	"reflect"
	"regexp"
	"strings"
)

const peerAuditMarker = "omabit.gateway-peer-audit=required-v1"

func gatewayPeersRequired(cmdline string, resume bool) bool {
	count := 0
	for _, word := range strings.Fields(cmdline) {
		if strings.HasPrefix(word, "omabit.gateway-peer-audit=") {
			need(word == peerAuditMarker, "unknown gateway peer audit mode")
			count++
		}
	}
	need(count <= 1 && (count == 0 || resume), "invalid gateway peer audit selection")
	return count == 1
}

func auditGatewayState(state map[string]any, baseline []any, checkpoint map[string]any) map[string]any {
	need(len(baseline) == 4, "gateway state baseline differs")
	for _, name := range []string{"ok", "state_survived", "queued_update_survived", "existing_subscription_survived", "fresh_subscription_reconciled"} {
		need(state[name] == true, "gateway state recovery gate missing")
	}
	need(state["writes_replayed"] == false, "gateway state write replayed")
	entries := append([]any{}, baseline...)
	writes := list(state["writes"])
	need(len(writes) == 3, "gateway recovery write count differs")
	phases := []string{"queued-before-gateway-replacement", "existing-subscription-after-restart", "fresh-subscription-after-restart"}
	publicWrites := []any{}
	for n, v := range writes {
		w := obj(v)
		entry, ok := w["entry"].(string)
		need(ok && regexp.MustCompile(`^omabit-state-[0-9a-f]{24}$`).MatchString(entry), "gateway state write identity differs")
		for _, prior := range entries {
			need(prior != entry, "gateway recovery write repeated")
		}
		need(w["phase"] == phases[n] && w["state"] == "acknowledged" && w["http_status"] == float64(200), "gateway write outcome unresolved")
		entries = append(entries, entry)
		publicWrites = append(publicWrites, map[string]any{"phase": phases[n], "entry_sha256": hash(entry), "receipt_sha256": hash(w["receipt"]), "response_sha256": hexValue(w["response_sha256"], 64, "")})
	}
	need(reflect.DeepEqual(entries[:5], checkpoint["entries"]), "retained checkpoint entries differ")
	channels := list(state["channels"])
	need(len(channels) == 2 && channels[0] != channels[1] && reflect.DeepEqual(channels, state["channels_deleted"]), "gateway recovery channel cleanup differs")
	for _, v := range channels {
		s, ok := v.(string)
		need(ok && regexp.MustCompile(`^/~/channel/omabit-state-[0-9a-f]{32}$`).MatchString(s), "gateway recovery channel differs")
	}
	need(state["resume_channel"] == channels[0] && checkpoint["channel"] == channels[0] && state["resume_cursor"] == checkpoint["cursor"], "gateway resume cursor binding differs")
	observations := list(state["observations"])
	need(len(observations) == 10, "gateway recovery observation count differs")
	snapshots := []any{}
	scries, subscriptions := 0, 0
	last := map[any]float64{}
	revisions := []float64{4, 5, 6, 6, 7}
	for _, v := range observations {
		o := obj(v)
		current := obj(o["state"])
		rev, ok := current["revision"].(float64)
		need(ok && rev >= 4 && rev <= 7 && rev == float64(int(rev)) && reflect.DeepEqual(current["entries"], entries[:int(rev)]), "gateway durable state changed")
		if o["via"] == "scry" {
			scries++
			continue
		}
		need(o["via"] == "subscription" && subscriptions < 5 && rev == revisions[subscriptions], "gateway subscription state differs")
		channel := channels[0]
		if subscriptions >= 3 {
			channel = channels[1]
		}
		need(o["channel"] == channel, "gateway subscription binding differs")
		cursor, seen := last[channel]
		if !seen {
			cursor = -1
		}
		events := list(o["events"])
		facts := 0
		ack := false
		for _, pair := range events {
			p := list(pair)
			need(len(p) == 2, "gateway SSE event pair differs")
			id, ok := p[0].(float64)
			need(ok && id > cursor && id == float64(int64(id)), "gateway SSE cursor repeated")
			cursor = id
			e := obj(p[1])
			need(e["response"] != "quit" && e["err"] == nil, "gateway subscription rejected")
			if e["response"] == "diff" {
				need(e["id"] == float64(1) && reflect.DeepEqual(e["json"], current), "gateway subscription fact differs")
				facts++
			}
			if e["response"] == "subscribe" {
				need(e["id"] == float64(1) && e["ok"] == "ok", "gateway subscribe acknowledgment differs")
				ack = true
			}
		}
		need(facts == 1 && o["cursor"] == cursor, "gateway snapshot/cursor missing")
		if subscriptions == 0 || subscriptions == 3 {
			need(ack, "gateway subscribe acknowledgment missing")
		}
		if subscriptions == 0 {
			need(cursor == checkpoint["cursor"], "saved pre-restart cursor differs")
		}
		last[channel] = cursor
		snapshots = append(snapshots, map[string]any{"revision": rev, "state_sha256": hash(current), "channel": channel, "cursor": cursor, "events": len(events)})
		subscriptions++
	}
	need(scries == 5 && subscriptions == 5, "gateway state observation totals differ")
	return map[string]any{"baseline_revision": 4, "final_revision": 7, "baseline_sha256": hash(baseline), "writes": publicWrites, "scry_observations": scries, "snapshots": snapshots, "channels_deleted": channels, "writes_replayed": false, "raw_values_exported": false}
}

func auditGatewayPeers(original, report, resumed, checkpoint map[string]any) map[string]any {
	project := hexValue(original["project"], 32, "")
	oldGateway, newGateway := hexValue(resumed["previous_gateway"], 64, ""), hexValue(resumed["container"], 64, "")
	operation := hexValue(resumed["operation"], 32, "")
	need(report["version"] == float64(1) && report["ok"] == true && report["project"] == project && report["previous_gateway"] == oldGateway && report["gateway"] == newGateway && report["operation"] == operation, "gateway peer identity/result differs")
	need(report["transport_passed"] == true && report["original_sessions_survived"] == true && report["cross_ship_sessions_refused"] == true && report["writes_replayed"] == false && len(list(report["cleanup_errors"])) == 0, "gateway peer recovery gate missing")
	need(report["error_type"] == nil && report["error_sha256"] == nil, "gateway peer recovery contains failure")
	need(checkpoint["version"] == float64(1) && checkpoint["ready"] == true && checkpoint["project"] == project && checkpoint["previous_gateway"] == oldGateway, "gateway client checkpoint differs")
	previousNS, ok := report["previous_namespace"].(string)
	need(ok && previousNS == original["network_namespace"] && previousNS == checkpoint["previous_namespace"], "prior gateway namespace differs")
	currentNS, ok := report["network_namespace"].(string)
	need(ok && regexp.MustCompile(`^net:\[[0-9]{1,20}\]$`).MatchString(currentNS), "gateway namespace malformed")
	guestNS, ok := report["guest_namespace"].(string)
	need(ok && regexp.MustCompile(`^net:\[[0-9]{1,20}\]$`).MatchString(guestNS) && currentNS != guestNS && previousNS != guestNS, "gateway shares guest namespace")
	need(oldGateway != newGateway && report["historical_namespace_number_repeated"] == (currentNS == previousNS), "gateway generation evidence differs")
	members := obj(report["ships"])
	previousMembers := obj(original["ships"])
	savedMembers := obj(checkpoint["ships"])
	need(len(members) == 2 && len(savedMembers) == 2, "gateway member set differs")
	publicMembers := map[string]any{}
	memberIDs := map[string]bool{}
	for _, ship := range []string{"zod", "nec"} {
		m, old, saved := obj(members[ship]), obj(previousMembers[ship]), obj(savedMembers[ship])
		id, oldID := hexValue(m["container"], 64, ""), hexValue(m["previous_container"], 64, "")
		need(id != oldID && id != newGateway && id != oldGateway && !memberIDs[id] && oldID == old["container"] && oldID == saved["container"] && m["ready"] == true, "gateway member binding differs")
		memberIDs[id] = true
		pier := list(m["pier_identity"])
		need(len(pier) == 2 && reflect.DeepEqual(pier, saved["pier_identity"]), "gateway pier object differs")
		publicMembers[ship] = map[string]any{"container": id, "previous_container": oldID, "pier_identity": []any{positiveInteger(pier[0]), positiveInteger(pier[1])}}
	}
	stops := list(report["stop_records"])
	need(len(stops) == 2, "gateway peer shutdown count differs")
	publicStops := []any{}
	seen := map[string]bool{}
	for _, v := range stops {
		stop := obj(v)
		id := hexValue(stop["container"], 64, "")
		need(memberIDs[id] && !seen[id] && stop["project"] == project && stop["mode"] == "sigterm" && stop["passed"] == true && stop["dispatched"] == true && stop["acknowledged"] == true && stop["exit"] == float64(0) && stop["running"] == false && stop["oom"] == false, "gateway peer stop differs")
		seen[id] = true
		publicStops = append(publicStops, map[string]any{"container": id, "mode": "sigterm", "passed": true, "exit": 0})
	}
	baseline := list(report["baseline_entries"])
	originalWrites := list(obj(original["application_state"])["writes"])
	need(len(baseline) == 4 && len(originalWrites) == 4, "original baseline write count differs")
	for n, v := range baseline {
		need(v == obj(originalWrites[n])["entry"], "original state baseline changed")
	}
	state := auditGatewayState(obj(report["application_state"]), baseline, checkpoint)
	narrow := obj(resumed["peer_recovery"])
	for _, name := range []string{"ok", "project", "previous_gateway", "gateway", "operation", "original_sessions_survived", "cross_ship_sessions_refused", "writes_replayed"} {
		need(narrow[name] == report[name], "resume controller peer proof differs")
	}
	return map[string]any{"ok": true, "project": project, "operation": operation, "previous_gateway": oldGateway, "gateway": newGateway, "previous_namespace": previousNS, "network_namespace": currentNS, "guest_namespace": guestNS, "historical_namespace_number_repeated": currentNS == previousNS, "members": publicMembers, "stops": publicStops, "original_sessions_survived": true, "cross_ship_sessions_refused": true, "state": state, "raw_state_exported": false}
}

func inspectGatewayPeers(original, resumed map[string]any) map[string]any {
	report, reportSHA := readResult("gateway-peer-recovery.json")
	project := hexValue(original["project"], 32, "")
	checkpoint, checkpointSHA := readPublicReceipt("/newroot/opt/omabit-lab/results/gateway-clients-" + project + "/checkpoint.json")
	operation := hexValue(resumed["operation"], 32, "")
	selected, selectionSHA := readPublicReceipt("/newroot/home/lab/omabit-private-resume-" + project + "-" + operation + "/selection.json")
	need(obj(selected["guest_namespaces"])["net"] == report["guest_namespace"], "guest namespace selection differs")
	out := auditGatewayPeers(original, report, resumed, checkpoint)
	out["selection_sha256"] = selectionSHA
	out["report_sha256"] = reportSHA
	out["checkpoint_sha256"] = checkpointSHA
	return out
}

// Diagnostic-only projection. A failure report never establishes recovery.
func gatewayPeerFailureSummary(report map[string]any) map[string]any {
	out := resumeFailureSummary(report)
	prior := peerFailureSummary(report)
	out["ships"] = prior["ships"]
	out["stops"] = prior["stops"]
	if ns, ok := report["network_namespace"].(string); ok && regexp.MustCompile(`^net:\[[0-9]{1,20}\]$`).MatchString(ns) {
		out["network_namespace"] = ns
	}
	for _, field := range []string{"original_sessions_survived", "cross_ship_sessions_refused", "transport_passed", "writes_replayed"} {
		out[field] = report[field] == true
	}
	_, out["application_state_recorded"] = report["application_state"].(map[string]any)
	if rows, ok := report["cleanup_errors"].([]any); ok {
		out["cleanup_error_count"] = len(rows)
	}
	return out
}

func inspectGatewayPeerFailure() (out map[string]any) {
	out = map[string]any{"available": false, "raw_state_exported": false}
	defer func() {
		if recover() != nil {
			out = map[string]any{"available": false, "raw_state_exported": false}
		}
	}()
	report, digest := readPublicReceipt("/newroot/opt/omabit-lab/results/gateway-peer-recovery.json")
	out = gatewayPeerFailureSummary(report)
	out["available"] = true
	out["report_sha256"] = digest
	out["raw_state_exported"] = false
	return out
}

package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func syntheticGatewayPeers() (map[string]any, map[string]any, map[string]any, map[string]any) {
	_, original := syntheticResults()
	project := original["project"]
	oldGateway, newGateway := strings.Repeat("d", 64), strings.Repeat("e", 64)
	operation := strings.Repeat("f", 32)
	baseline := []any{}
	for _, w := range list(obj(original["application_state"])["writes"]) {
		baseline = append(baseline, obj(w)["entry"])
	}
	entries := append([]any{}, baseline...)
	phases := []string{"queued-before-gateway-replacement", "existing-subscription-after-restart", "fresh-subscription-after-restart"}
	writes := []any{}
	for n, p := range phases {
		entry := "omabit-state-" + strings.Repeat(string(rune('a'+n)), 24)
		entries = append(entries, entry)
		writes = append(writes, map[string]any{"entry": entry, "phase": p, "state": "acknowledged", "http_status": float64(200), "receipt": "synthetic-private-receipt", "response_sha256": strings.Repeat("a", 64)})
	}
	channels := []any{"/~/channel/omabit-state-" + strings.Repeat("a", 32), "/~/channel/omabit-state-" + strings.Repeat("b", 32)}
	state := func(n int) map[string]any { return map[string]any{"revision": float64(n), "entries": entries[:n]} }
	observations := []any{}
	subscription := func(n, channel, cursor int, initial bool) {
		events := []any{}
		if initial {
			events = append(events, []any{float64(0), map[string]any{"response": "subscribe", "id": float64(1), "ok": "ok"}})
		}
		events = append(events, []any{float64(cursor), map[string]any{"response": "diff", "id": float64(1), "json": state(n)}})
		observations = append(observations, map[string]any{"via": "subscription", "state": state(n), "channel": channels[channel], "cursor": float64(cursor), "events": events})
	}
	scry := func(n int) { observations = append(observations, map[string]any{"via": "scry", "state": state(n)}) }
	subscription(4, 0, 1, true)
	scry(5)
	scry(5)
	subscription(5, 0, 2, false)
	scry(6)
	subscription(6, 0, 3, false)
	subscription(6, 1, 1, true)
	scry(7)
	subscription(7, 1, 2, false)
	scry(7)
	app := map[string]any{"ok": true, "state_survived": true, "queued_update_survived": true, "existing_subscription_survived": true, "fresh_subscription_reconciled": true, "writes_replayed": false, "writes": writes, "observations": observations, "channels": channels, "channels_deleted": channels, "resume_channel": channels[0], "resume_cursor": float64(1)}
	original["ships"] = map[string]any{}
	members, savedMembers := map[string]any{}, map[string]any{}
	stops := []any{}
	for n, ship := range []string{"zod", "nec"} {
		oldID, newID := strings.Repeat(string(rune('a'+n)), 64), strings.Repeat(string(rune('1'+n)), 64)
		obj(original["ships"])[ship] = map[string]any{"container": oldID}
		pier := []any{float64(1), float64(n + 2)}
		members[ship] = map[string]any{"container": newID, "previous_container": oldID, "pier_identity": pier, "ready": true}
		savedMembers[ship] = map[string]any{"container": oldID, "pier_identity": pier}
		stops = append(stops, map[string]any{"container": newID, "project": project, "mode": "sigterm", "passed": true, "dispatched": true, "acknowledged": true, "exit": float64(0), "running": false, "oom": false})
	}
	checkpoint := map[string]any{"version": float64(1), "ready": true, "project": project, "previous_gateway": oldGateway, "previous_namespace": original["network_namespace"], "ships": savedMembers, "entries": entries[:5], "channel": channels[0], "cursor": float64(1)}
	report := map[string]any{"version": float64(1), "ok": true, "project": project, "previous_gateway": oldGateway, "gateway": newGateway, "operation": operation, "transport_passed": true, "original_sessions_survived": true, "cross_ship_sessions_refused": true, "writes_replayed": false, "cleanup_errors": []any{}, "previous_namespace": original["network_namespace"], "network_namespace": "net:[5678]", "guest_namespace": "net:[1]", "historical_namespace_number_repeated": false, "ships": members, "stop_records": stops, "baseline_entries": baseline, "application_state": app}
	narrow := map[string]any{}
	for _, n := range []string{"ok", "project", "previous_gateway", "gateway", "operation", "original_sessions_survived", "cross_ship_sessions_refused", "writes_replayed"} {
		narrow[n] = report[n]
	}
	resumed := map[string]any{"previous_gateway": oldGateway, "container": newGateway, "operation": operation, "peer_recovery": narrow}
	var result []map[string]any
	raw, _ := json.Marshal([]any{original, report, resumed, checkpoint})
	_ = json.Unmarshal(raw, &result)
	return result[0], result[1], result[2], result[3]
}

func TestGatewayPeerAuditExcludesPrivateState(t *testing.T) {
	original, report, resumed, checkpoint := syntheticGatewayPeers()
	private := "0wPRIVATE-cookie-value"
	report["extra"] = private
	obj(report["ships"])["zod"].(map[string]any)["extra"] = private
	checkpoint["cookie"] = private
	out := auditGatewayPeers(original, report, resumed, checkpoint)
	raw, e := json.Marshal(out)
	for _, secret := range []string{private, "synthetic-private-value", "synthetic-private-receipt"} {
		if bytes.Contains(raw, []byte(secret)) {
			t.Fatal("private state exported")
		}
	}
	if e != nil || out["ok"] != true || obj(out["state"])["final_revision"] != 7 {
		t.Fatal("recovery projection differs")
	}
}

func TestGatewayPeerAuditRejectsChangedRecovery(t *testing.T) {
	for _, kind := range []string{"project", "gateway", "operation", "sessions", "crossed-cookie", "replay", "cleanup", "failed", "checkpoint", "previous-namespace", "host-namespace", "namespace-repeat-flag", "same-gateway", "member-old", "member-duplicate", "pier", "member-unready", "stop-exit", "stop-duplicate", "stop-mode", "baseline", "checkpoint-state", "pending-write", "duplicate-write", "write-count", "write-phase", "state-revision", "state-value", "cursor", "resume-cursor", "channel-delete", "channel-binding", "event-ack", "event-fact", "event-replay", "observation-count", "controller"} {
		t.Run(kind, func(t *testing.T) {
			original, r, resumed, c := syntheticGatewayPeers()
			s := obj(r["application_state"])
			switch kind {
			case "project":
				r["project"] = strings.Repeat("c", 32)
			case "gateway":
				r["gateway"] = strings.Repeat("c", 64)
			case "operation":
				r["operation"] = strings.Repeat("c", 32)
			case "sessions":
				r["original_sessions_survived"] = false
			case "crossed-cookie":
				r["cross_ship_sessions_refused"] = false
			case "replay":
				s["writes_replayed"] = true
			case "cleanup":
				r["cleanup_errors"] = []any{"failed"}
			case "failed":
				r["error_sha256"] = strings.Repeat("a", 64)
			case "checkpoint":
				c["ready"] = false
			case "previous-namespace":
				c["previous_namespace"] = "net:[4567]"
			case "host-namespace":
				r["network_namespace"] = r["guest_namespace"]
			case "namespace-repeat-flag":
				r["historical_namespace_number_repeated"] = true
			case "same-gateway":
				resumed["container"] = resumed["previous_gateway"]
				r["gateway"] = resumed["previous_gateway"]
			case "member-old":
				obj(obj(r["ships"])["zod"])["previous_container"] = strings.Repeat("c", 64)
			case "member-duplicate":
				obj(obj(r["ships"])["zod"])["container"] = obj(obj(r["ships"])["nec"])["container"]
			case "pier":
				obj(obj(r["ships"])["zod"])["pier_identity"] = []any{float64(1), float64(99)}
			case "member-unready":
				obj(obj(r["ships"])["zod"])["ready"] = false
			case "stop-exit":
				obj(list(r["stop_records"])[0])["exit"] = float64(1)
			case "stop-duplicate":
				r["stop_records"] = []any{list(r["stop_records"])[0], list(r["stop_records"])[0]}
			case "stop-mode":
				obj(list(r["stop_records"])[0])["mode"] = "forced"
			case "baseline":
				list(r["baseline_entries"])[0] = "changed"
			case "checkpoint-state":
				list(c["entries"])[4] = "changed"
			case "pending-write":
				obj(list(s["writes"])[0])["state"] = "pending"
			case "duplicate-write":
				obj(list(s["writes"])[1])["entry"] = obj(list(s["writes"])[0])["entry"]
			case "write-count":
				s["writes"] = list(s["writes"])[1:]
			case "write-phase":
				obj(list(s["writes"])[0])["phase"] = "resend"
			case "state-revision":
				obj(obj(list(s["observations"])[0])["state"])["revision"] = float64(8)
			case "state-value":
				obj(obj(list(s["observations"])[0])["state"])["entries"] = []any{"changed"}
			case "cursor":
				obj(list(s["observations"])[3])["cursor"] = float64(1)
			case "resume-cursor":
				s["resume_cursor"] = float64(2)
			case "channel-delete":
				s["channels_deleted"] = []any{}
			case "channel-binding":
				obj(list(s["observations"])[3])["channel"] = list(s["channels"])[1]
			case "event-ack":
				obj(list(s["observations"])[0])["events"] = list(obj(list(s["observations"])[0])["events"])[1:]
			case "event-fact":
				obj(list(list(obj(list(s["observations"])[3])["events"])[0])[1])["json"] = map[string]any{}
			case "event-replay":
				list(list(obj(list(s["observations"])[3])["events"])[0])[0] = float64(1)
			case "observation-count":
				s["observations"] = list(s["observations"])[1:]
			case "controller":
				obj(resumed["peer_recovery"])["ok"] = false
			}
			defer func() {
				if recover() == nil {
					t.Error("changed recovery accepted")
				}
			}()
			auditGatewayPeers(original, r, resumed, c)
		})
	}
}

func TestGatewayPeerAuditSelectionRequiresResume(t *testing.T) {
	if gatewayPeersRequired("", true) || !gatewayPeersRequired(peerAuditMarker, true) {
		t.Fatal("peer selection differs")
	}
	for _, value := range []string{peerAuditMarker + " " + peerAuditMarker, "omabit.gateway-peer-audit=false"} {
		t.Run(value, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("invalid peer marker accepted")
				}
			}()
			gatewayPeersRequired(value, true)
		})
	}
	defer func() {
		if recover() == nil {
			t.Fatal("peer audit without resume accepted")
		}
	}()
	gatewayPeersRequired(peerAuditMarker, false)
}

func TestGatewayPeerAuditAllowsRecycledHistoricalNamespaceNumber(t *testing.T) {
	original, report, resumed, checkpoint := syntheticGatewayPeers()
	report["network_namespace"] = report["previous_namespace"]
	report["historical_namespace_number_repeated"] = true
	out := auditGatewayPeers(original, report, resumed, checkpoint)
	if out["ok"] != true || out["historical_namespace_number_repeated"] != true {
		t.Fatal(out)
	}
}

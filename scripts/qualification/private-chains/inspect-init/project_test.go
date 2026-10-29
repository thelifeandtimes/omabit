package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func syntheticState() map[string]any {
	entries := []any{"synthetic-private-value-one", "synthetic-private-value-two", "synthetic-private-value-three", "synthetic-private-value-four"}
	channels := []any{"/~/channel/omabit-state-" + strings.Repeat("a", 30), "/~/channel/omabit-state-" + strings.Repeat("b", 30)}
	writes := []any{}
	for n, phase := range []string{"connected", "disconnected-before-restart", "existing-subscription-after-restart", "fresh-subscription-after-restart"} {
		writes = append(writes, map[string]any{"state": "acknowledged", "http_status": float64(200), "phase": phase, "entry": entries[n], "receipt": fmt.Sprintf("synthetic-receipt-%d", n), "response_sha256": strings.Repeat("a", 64)})
	}
	state := func(rev int) map[string]any {
		return map[string]any{"revision": float64(rev), "entries": append([]any{}, entries[:rev]...)}
	}
	observations := []any{}
	for n, rev := range []int{0, 1, 2, 3, 3, 4} {
		channel := channels[0]
		cursor := n + 1
		if n >= 4 {
			channel = channels[1]
			cursor = n - 3
		}
		events := []any{}
		s := state(rev)
		if n == 0 || n == 4 {
			events = append(events, []any{float64(0), map[string]any{"response": "subscribe", "id": float64(1), "ok": "ok"}})
		}
		events = append(events, []any{float64(cursor), map[string]any{"response": "diff", "id": float64(1), "json": s}})
		observations = append(observations, map[string]any{"via": "subscription", "state": s, "channel": channel, "cursor": float64(cursor), "events": events}, map[string]any{"via": "scry", "state": state(rev)})
	}
	observations = append(observations, map[string]any{"via": "scry", "state": state(4)})
	return map[string]any{"ok": true, "state_survived": true, "queued_update_survived": true, "existing_subscription_survived": true, "fresh_subscription_reconciled": true, "writes_replayed": false, "writes": writes, "channels": channels, "channels_deleted": channels, "observations": observations, "resume_channel": channels[0], "resume_cursor": float64(2), "fixture_sha256": strings.Repeat("b", 64)}
}
func syntheticResults() (map[string]any, map[string]any) {
	project := strings.Repeat("a", 32)
	keys, cases := []any{}, []any{}
	for n, ship := range []string{"~zod", "~nec"} {
		keys = append(keys, map[string]any{"ship": ship, "number": float64(n), "auth": "0x" + strings.Repeat("a", 64), "crypt": "0x" + strings.Repeat("b", 64)})
		cases = append(cases, map[string]any{"ship": ship, "correct": true, "wrongLifeRefused": true, "wrongPublicKeyRefused": true})
	}
	summary := map[string]any{"ok": true, "project": project, "ready": map[string]any{"chainId": "0x539", "genesis": "0x" + strings.Repeat("c", 64), "externalRpcMutationsAllowed": false, "publicKeys": keys}, "gateway_proof": map[string]any{"passed": true, "publicKeys": keys, "cases": cases, "gateway": map[string]any{"signalShutdown": true, "head": "0x50", "chainCalls": []any{map[string]any{"method": "eth_blockNumber"}}}}, "gateway_stop": map[string]any{"ExitCode": float64(0), "OOMKilled": false, "Running": false}}
	attempts, stops := []any{}, []any{}
	for n := 0; n < 7; n++ {
		s := "acknowledged"
		if n == 6 {
			s = "rejected"
		}
		attempts = append(attempts, map[string]any{"state": s})
	}
	for n := 0; n < 3; n++ {
		mode := "lens-exit"
		if n == 2 {
			mode = "sigterm"
		}
		stops = append(stops, map[string]any{"passed": true, "acknowledged": true, "dispatched": true, "exit": float64(0), "oom": false, "running": false, "project": project, "container": strings.Repeat(string(rune('a'+n)), 64), "mode": mode})
	}
	peers := map[string]any{"project": project, "ok": true, "transport_passed": true, "restart_same_pier": true, "restart_kept_sessions": true, "cross_ship_sessions_refused": true, "attempts": attempts, "stop_records": stops, "application_state": syntheticState(), "restart": map[string]any{"original": strings.Repeat("b", 64), "replacement": strings.Repeat("c", 64), "pier_device": float64(1), "pier_inode": float64(2)}, "network_namespace": "net:[1234]"}
	return summary, peers
}
func TestSuccessProjectionDropsUnknownPrivateFields(t *testing.T) {
	summary, peers := syntheticResults()
	secret := "0wPRIVATE-sensitive-content"
	summary["unexpected"] = secret
	peers["unexpected"] = secret
	for _, v := range list(peers["stop_records"]) {
		obj(v)["unexpected"] = secret
	}
	obj(peers["restart"])["unexpected"] = secret
	proof := obj(summary["gateway_proof"])
	for _, name := range []string{"publicKeys", "cases"} {
		for _, v := range list(proof[name]) {
			obj(v)["unexpected"] = secret
		}
	}
	for _, v := range list(obj(peers["application_state"])["writes"]) {
		obj(v)["receipt"] = secret
	}
	result := auditResults(summary, peers)
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte(secret)) || bytes.Contains(raw, []byte("synthetic-private-value")) {
		t.Fatal("raw application content exported")
	}
	if result["ok"] != true || len(list(result["stops"])) != 3 || len(list(obj(result["state"])["snapshots"])) != 6 {
		t.Fatal("public success evidence missing")
	}
}
func TestProjectionRejectsInvalidPublicValues(t *testing.T) {
	for _, kind := range []string{"channel", "container", "project", "genesis", "key", "method", "namespace", "stop"} {
		t.Run(kind, func(t *testing.T) {
			summary, peers := syntheticResults()
			secret := "0wPRIVATE-sensitive-content"
			switch kind {
			case "channel":
				obj(peers["application_state"])["channels"] = []any{secret, "other"}
			case "container":
				obj(list(peers["stop_records"])[0])["container"] = secret
			case "project":
				peers["project"] = strings.Repeat("b", 32)
			case "genesis":
				obj(summary["ready"])["genesis"] = secret
			case "key":
				obj(list(obj(summary["gateway_proof"])["publicKeys"])[0])["auth"] = secret
			case "method":
				obj(list(obj(obj(summary["gateway_proof"])["gateway"])["chainCalls"])[0])["method"] = secret
			case "namespace":
				peers["network_namespace"] = secret
			case "stop":
				obj(list(peers["stop_records"])[0])["exit"] = float64(1)
			}
			defer func() {
				v := recover()
				if v == nil {
					t.Error("invalid public metadata admitted")
				}
				if strings.Contains(fmt.Sprint(v), secret) {
					t.Error("rejection reflected private value")
				}
			}()
			auditResults(summary, peers)
		})
	}
}

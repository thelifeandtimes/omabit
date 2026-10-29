package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func initialPublicationFixture(t *testing.T) (map[string]any, map[string]any, map[string]any) {
	t.Helper()
	summary, peers := syntheticResults()
	published, client := publicationFixture(t)
	summary["anchor"] = published["container"]
	summary["publication"] = published["publication"]
	obj(summary["publication"])["initial_start"] = true
	ready := obj(summary["ready"])
	ready["headHash"] = "0x" + strings.Repeat("e", 64)
	ready["block"] = "0x50"
	ready["initialNetwork"] = "published-v1"
	gateway := obj(obj(summary["gateway_proof"])["gateway"])
	gateway["headHash"] = ready["headHash"]
	gateway["initialKeySetup"] = ready
	baseline := []any{}
	for _, w := range list(obj(peers["application_state"])["writes"]) {
		baseline = append(baseline, obj(w)["entry"])
	}
	entries := append([]any{}, baseline...)
	writes := []any{}
	for n, phase := range []string{"initial-published-connected", "initial-published-fresh-subscription"} {
		entry := "omabit-state-" + strings.Repeat(string(rune('c'+n)), 24)
		entries = append(entries, entry)
		writes = append(writes, map[string]any{"phase": phase, "entry": entry, "receipt": "omabit-receipt-" + strings.Repeat(string(rune('e'+n)), 32), "state": "acknowledged", "http_status": float64(200), "response_sha256": strings.Repeat("a", 64)})
	}
	channels := []any{"/~/channel/omabit-state-" + strings.Repeat("a", 32), "/~/channel/omabit-state-" + strings.Repeat("b", 32)}
	observations := []any{}
	for n, rev := range []int{4, 5, 5, 5, 6, 6, 6} {
		state := map[string]any{"revision": float64(rev), "entries": entries[:rev]}
		if n == 1 || n == 4 || n == 6 {
			observations = append(observations, map[string]any{"via": "scry", "state": state})
			continue
		}
		channel, cursor := 0, 2
		if n >= 3 {
			channel = 1
		}
		events := []any{}
		if n == 0 || n == 3 {
			cursor = 1
			events = append(events, []any{float64(0), map[string]any{"response": "subscribe", "id": float64(1), "ok": "ok"}})
		}
		events = append(events, []any{float64(cursor), map[string]any{"response": "diff", "id": float64(1), "json": state}})
		observations = append(observations, map[string]any{"via": "subscription", "state": state, "channel": channels[channel], "cursor": float64(cursor), "events": events})
	}
	app := map[string]any{"ok": true, "initial_publication": true, "writes_replayed": false, "writes": writes, "channels": channels, "channels_deleted": channels, "observations": observations}
	pub := obj(client["publication"])
	requests := []any{}
	add := func(ship, path, method string, status int, stream, cross bool) {
		port := 18381
		if ship == "nec" {
			port = 18382
		}
		requests = append(requests, map[string]any{"ship": ship, "port": float64(port), "path": path, "method": method, "status": float64(status), "stream": stream, "crossed_session": cross})
	}
	for _, ship := range []string{"zod", "nec"} {
		add(ship, "/~/host", "GET", 200, false, false)
		add(ship, "/~/scry/hood/kiln/pikes.json", "GET", 200, false, false)
		add(ship, "/~/scry/hood/kiln/pikes.json", "GET", 403, false, true)
	}
	for _, channel := range channels {
		add("zod", "/spider/base/ted-eval/khan-eval/noun", "POST", 200, false, false)
		add("nec", "/~/scry/omabit-state/state.json", "GET", 200, false, false)
		for i := 0; i < 2; i++ {
			add("nec", channel.(string), "GET", 200, true, false)
		}
		for i := 0; i < 4; i++ {
			add("nec", channel.(string), "PUT", 204, false, false)
		}
	}
	add("nec", "/~/scry/omabit-state/state.json", "GET", 200, false, false)
	pub["requests"] = requests
	report := map[string]any{"version": float64(1), "ok": true, "project": summary["project"], "gateway": summary["anchor"], "initial_start": true, "original_sessions_reused": true, "cleanup_errors": []any{}, "baseline_entries": baseline, "cookie_sha256": map[string]any{"zod": strings.Repeat("a", 64), "nec": strings.Repeat("b", 64)}, "application_state": app, "publication": pub}
	// Independent JSON values model distinct persisted receipts (no map aliases).
	raw, e := json.Marshal([]any{summary, peers, report})
	if e != nil {
		t.Fatal(e)
	}
	var rows []map[string]any
	if e = json.Unmarshal(raw, &rows); e != nil {
		t.Fatal(e)
	}
	return rows[0], rows[1], rows[2]
}

func TestInitialPublicationValidatesDirectClientWithoutExportingState(t *testing.T) {
	s, p, r := initialPublicationFixture(t)
	r["unknown"] = "PRIVATE-session-value"
	if auditResults(s, p)["ok"] != true {
		t.Fatal("base proof differs")
	}
	out := auditInitialPublication(s, p, r)
	raw, e := json.Marshal(out)
	if e != nil || out["ok"] != true || out["requests"] != 23 || out["channel_puts"] != 8 || obj(out["application_state"])["final_revision"] != 6 {
		t.Fatal("initial publication projection differs")
	}
	for _, private := range []string{"PRIVATE-session-value", "synthetic-private-value", "omabit-state-", "omabit-receipt-", "cookie_sha256"} {
		if bytes.Contains(raw, []byte(private)) {
			t.Fatal("private state exported")
		}
	}
}

func TestInitialPublicationRejectsIncompleteOrReboundProof(t *testing.T) {
	for _, kind := range []string{"project", "anchor", "not-initial", "session", "cleanup", "error", "mode", "head", "zero-head", "ready-copy", "baseline", "duplicate-write", "pending-write", "receipt", "duplicate-receipt", "writes-replayed", "missing-observation", "wrong-state", "wrong-via", "wrong-channel", "missing-delete", "missing-ack", "wrong-fact", "cursor", "replay-event", "unknown-event", "cookie", "request-channel", "missing-put", "stream", "cross-session", "wildcard", "rpc", "network-owner"} {
		t.Run(kind, func(t *testing.T) {
			s, p, r := initialPublicationFixture(t)
			app := obj(r["application_state"])
			writes := list(app["writes"])
			obs := list(app["observations"])
			requests := list(obj(r["publication"])["requests"])
			switch kind {
			case "project":
				r["project"] = strings.Repeat("d", 32)
			case "anchor":
				r["gateway"] = strings.Repeat("d", 64)
			case "not-initial":
				r["initial_start"] = false
			case "session":
				r["original_sessions_reused"] = false
			case "cleanup":
				r["cleanup_errors"] = []any{"failure"}
			case "error":
				r["error_type"] = "RuntimeError"
			case "mode":
				obj(s["ready"])["initialNetwork"] = "none"
			case "head":
				obj(obj(s["gateway_proof"])["gateway"])["headHash"] = "0x" + strings.Repeat("d", 64)
			case "zero-head":
				obj(s["ready"])["headHash"] = "0x" + strings.Repeat("0", 64)
			case "ready-copy":
				obj(obj(obj(s["gateway_proof"])["gateway"])["initialKeySetup"])["block"] = "0x51"
			case "baseline":
				list(r["baseline_entries"])[0] = "changed"
			case "duplicate-write":
				obj(writes[1])["entry"] = obj(writes[0])["entry"]
			case "pending-write":
				obj(writes[0])["state"] = "pending"
			case "receipt":
				obj(writes[0])["receipt"] = "private"
			case "duplicate-receipt":
				obj(writes[1])["receipt"] = obj(writes[0])["receipt"]
			case "writes-replayed":
				app["writes_replayed"] = true
			case "missing-observation":
				app["observations"] = obs[:6]
			case "wrong-state":
				obj(obj(obs[6])["state"])["revision"] = float64(5)
			case "wrong-via":
				obj(obs[1])["via"] = "subscription"
			case "wrong-channel":
				obj(obs[2])["channel"] = list(app["channels"])[1]
			case "missing-delete":
				app["channels_deleted"] = []any{}
			case "missing-ack":
				obj(obs[0])["events"] = list(obj(obs[0])["events"])[1:]
			case "wrong-fact":
				obj(list(list(obj(obs[0])["events"])[1])[1])["json"] = map[string]any{}
			case "cursor":
				obj(obs[2])["cursor"] = float64(1)
			case "replay-event":
				list(list(obj(obs[2])["events"])[0])[0] = float64(1)
			case "unknown-event":
				obj(list(list(obj(obs[2])["events"])[0])[1])["response"] = "quit"
			case "cookie":
				obj(r["cookie_sha256"])["nec"] = obj(r["cookie_sha256"])["zod"]
			case "request-channel":
				obj(requests[8])["path"] = "/~/channel/omabit-state-" + strings.Repeat("d", 32)
			case "missing-put":
				obj(r["publication"])["requests"] = append(requests[:10], requests[11:]...)
			case "stream":
				obj(requests[8])["stream"] = false
			case "cross-session":
				obj(requests[2])["status"] = float64(200)
			case "wildcard":
				obj(list(obj(obj(obj(obj(s["publication"])["first_observation"])["gateway"])["HostConfig"])["PortBindings"].(map[string]any)["18081/tcp"])[0])["HostIp"] = "0.0.0.0"
			case "rpc":
				obj(requests[0])["path"] = "/rpc"
			case "network-owner":
				obj(obj(obj(obj(s["publication"])["first_observation"])["network"])["Labels"])["org.omabit.qualification.private-chain"] = strings.Repeat("d", 32)
			}
			defer func() {
				if recover() == nil {
					t.Fatal("invalid initial proof accepted")
				}
			}()
			auditInitialPublication(s, p, r)
		})
	}
}

func TestInitialPublicationAuditSelection(t *testing.T) {
	marker := "omabit.initial-publication-audit=required-v1"
	if initialPublicationRequired("", false) || !initialPublicationRequired(marker, false) {
		t.Fatal("initial audit selection differs")
	}
	for _, row := range []struct {
		cmd   string
		other bool
	}{{marker, true}, {marker + " " + marker, false}, {"omabit.initial-publication-audit=unknown", false}} {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatal("ambiguous audit selection accepted")
				}
			}()
			initialPublicationRequired(row.cmd, row.other)
		}()
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("mixed isolation accepted")
			}
		}()
		isolationRequired("omabit.private-isolation-audit=required-v1", initialPublicationRequired(marker, false))
	}()
}

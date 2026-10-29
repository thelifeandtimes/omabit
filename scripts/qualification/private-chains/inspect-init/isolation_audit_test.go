package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func cloneIsolation(v any) any {
	b, e := json.Marshal(v)
	must(e)
	var out any
	must(json.Unmarshal(b, &out))
	return out
}
func syntheticIsolation() (map[string]any, map[string]map[string]any, map[string]map[string]any) {
	summaries, peers := map[string]map[string]any{}, map[string]map[string]any{}
	ready, live, extensions := map[string]any{}, map[string]any{}, map[string]any{}
	entries := map[string][]any{}
	for n, label := range []string{"a", "b"} {
		summary, peer := syntheticResults()
		project := strings.Repeat(label, 32)
		anchor := fmt.Sprintf("%064x", n*10+1)
		summary["project"] = project
		summary["anchor"] = anchor
		peer["project"] = project
		peer["network_namespace"] = fmt.Sprintf("net:[%d]", 100+n)
		for _, raw := range list(peer["stop_records"]) {
			obj(raw)["project"] = project
		}
		for k, raw := range list(obj(summary["gateway_proof"])["publicKeys"]) {
			key := obj(raw)
			key["auth"] = fmt.Sprintf("0x%064x", n*10+k*2+1)
			key["crypt"] = fmt.Sprintf("0x%064x", n*10+k*2+2)
		}
		// Renaming synthetic values in one baseline preserves all of its snapshots.
		bytes, e := json.Marshal(peer)
		must(e)
		bytes = []byte(strings.ReplaceAll(string(bytes), "synthetic-private-value", label+"-synthetic-private-value"))
		must(json.Unmarshal(bytes, &peer))
		summaries[label] = summary
		peers[label] = peer
		ready[label] = summary["ready"]
		container := func(id, name, network string, cmd []any) map[string]any {
			return map[string]any{"Id": id, "Name": name, "State": map[string]any{"Running": true, "OOMKilled": false}, "HostConfig": map[string]any{"NetworkMode": network, "PortBindings": nil}, "Config": map[string]any{"Labels": map[string]any{"org.omabit.qualification.private-chain": project}, "Cmd": cmd}}
		}
		ships, peerShips := map[string]any{}, map[string]any{}
		for k, ship := range []string{"zod", "nec"} {
			id := fmt.Sprintf("%064x", n*10+k+2)
			suffix := "-boot"
			if k == 1 {
				suffix = "-resume"
			}
			ships[ship] = container(id, "/omabit-private-"+project+"-"+ship+suffix, "container:"+anchor, []any{"--http-port", fmt.Sprint(18081 + k), "-p", fmt.Sprint(31337 + k)})
			peerShips[ship] = map[string]any{"container": id}
		}
		peer["ships"] = peerShips
		objects := map[string]any{}
		for k, name := range []string{"root", "chain", "zod", "nec"} {
			objects[name] = []any{float64(1), float64(n*10 + k + 1)}
		}
		live[label] = map[string]any{"project": project, "namespace": peer["network_namespace"], "anchor": container(anchor, "/omabit-private-"+project+"-gateway", "none", nil), "ships": ships, "objects": objects}
		for _, w := range list(obj(peer["application_state"])["writes"]) {
			entries[label] = append(entries[label], obj(w)["entry"])
		}
		entry := label + "-synthetic-private-extension"
		entries[label] = append(entries[label], entry)
		channel := "/~/channel/omabit-state-" + strings.Repeat(label, 32)
		state := func(rev int) any { return map[string]any{"revision": float64(rev), "entries": entries[label][:rev]} }
		subscription := func(rev, cursor int, initial bool) any {
			events := []any{}
			if initial {
				events = append(events, []any{float64(0), map[string]any{"id": float64(1), "response": "subscribe", "ok": "ok"}})
			}
			events = append(events, []any{float64(cursor), map[string]any{"id": float64(1), "response": "diff", "json": state(rev)}})
			return map[string]any{"via": "subscription", "channel": channel, "cursor": float64(cursor), "state": state(rev), "events": events}
		}
		extensions[label] = map[string]any{"writes": []any{map[string]any{"entry": entry, "receipt": "private-receipt", "phase": "project-" + label + "-isolation", "state": "acknowledged", "http_status": float64(200), "response_sha256": strings.Repeat("a", 64)}}, "channels": []any{channel}, "channels_deleted": []any{channel}, "observations": []any{subscription(4, 1, true), map[string]any{"via": "scry", "state": state(5)}, subscription(5, 2, false)}}
	}
	crossed := []any{}
	for n := 0; n < 4; n++ {
		src, dst := "a", "b"
		if n >= 2 {
			src, dst = "b", "a"
		}
		crossed = append(crossed, map[string]any{"source": src, "target": dst, "ship": []string{"zod", "nec"}[n%2], "dispatched": true, "status": float64(403)})
	}
	states := []any{}
	for n, label := range []string{"a", "b", "a", "b", "b", "a", "a"} {
		rev := []int{4, 4, 5, 4, 5, 5, 5}[n]
		states = append(states, map[string]any{"project": label, "phase": []string{"initial", "initial", "after-a", "after-a", "after-b", "after-b", "after-b-stopped"}[n], "state": map[string]any{"revision": float64(rev), "entries": entries[label][:rev]}})
	}
	receipt := map[string]any{"ok": true, "verified": true, "first": true, "second": true, "cleanup_errors": []any{}, "ready": ready, "before": cloneIsolation(live), "after": cloneIsolation(live), "after_second_stopped": map[string]any{"a": cloneIsolation(live["a"])}, "extensions": extensions, "cross_sessions": crossed, "states": states}
	return receipt, summaries, peers
}

func TestIsolationAuditRequiresTwoConcurrentProjectsAndNoStateLeak(t *testing.T) {
	receipt, summaries, peers := syntheticIsolation()
	out := auditIsolation(receipt, summaries, peers)
	raw, e := json.Marshal(out)
	must(e)
	if out["ok"] != true || out["private_project_isolation"] != true || strings.Contains(string(raw), "synthetic-private") || strings.Contains(string(raw), "private-receipt") {
		t.Fatal("missing proof or raw application data exported")
	}
	if !reflect.DeepEqual(obj(out["extensions"])["a"], auditIsolationExtension(obj(obj(receipt["extensions"])["a"]), "a", list(obj(list(receipt["states"])[2])["state"].(map[string]any)["entries"]))) {
		t.Fatal("extension proof differs")
	}
}

func TestIsolationAuditRefusesMissingOrCrossedEvidence(t *testing.T) {
	changes := map[string]func(map[string]any){
		"missing-live-pair": func(v map[string]any) { delete(obj(v["before"]), "b") },
		"stopped-peer": func(v map[string]any) {
			obj(obj(obj(obj(v["before"])["b"])["ships"])["nec"])["State"] = map[string]any{"Running": false, "OOMKilled": false}
		},
		"shared-namespace": func(v map[string]any) {
			obj(obj(v["before"])["b"])["namespace"] = obj(obj(v["before"])["a"])["namespace"]
		},
		"shared-chain-dir": func(v map[string]any) {
			obj(obj(obj(v["before"])["b"])["objects"])["chain"] = obj(obj(obj(v["before"])["a"])["objects"])["chain"]
		},
		"accepted-cookie":          func(v map[string]any) { obj(list(v["cross_sessions"])[0])["status"] = float64(200) },
		"missing-cookie-direction": func(v map[string]any) { v["cross_sessions"] = list(v["cross_sessions"])[:2] },
		"crossed-state":            func(v map[string]any) { obj(list(v["states"])[3])["state"] = obj(list(v["states"])[2])["state"] },
		"unacknowledged-write":     func(v map[string]any) { obj(list(obj(obj(v["extensions"])["b"])["writes"])[0])["state"] = "pending" },
		"unclosed-channel":         func(v map[string]any) { obj(obj(v["extensions"])["a"])["channels_deleted"] = []any{} },
		"replayed-event": func(v map[string]any) {
			list(list(obj(list(obj(obj(v["extensions"])["a"])["observations"])[2])["events"])[0])[0] = float64(1)
		},
		"first-lost-after-second-stop": func(v map[string]any) { v["after_second_stopped"] = map[string]any{} },
		"changed-after-pair": func(v map[string]any) {
			obj(obj(obj(v["after"])["b"])["objects"])["chain"] = []any{float64(1), float64(90)}
		},
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			v, s, p := syntheticIsolation()
			change(v)
			defer func() {
				if recover() == nil {
					t.Fatal("incomplete isolation admitted")
				}
			}()
			auditIsolation(v, s, p)
		})
	}
}

func TestIsolationMarkerIsMandatoryAndExclusive(t *testing.T) {
	if isolationRequired("console=ttyS0", false) || !isolationRequired("omabit.private-isolation-audit=required-v1", false) {
		t.Fatal("selection differs")
	}
	for _, cmd := range []string{"omabit.private-isolation-audit=bad", "omabit.private-isolation-audit=required-v1 omabit.private-isolation-audit=required-v1"} {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatal("invalid selection accepted")
				}
			}()
			isolationRequired(cmd, false)
		}()
	}
	defer func() {
		if recover() == nil {
			t.Fatal("mixed audit accepted")
		}
	}()
	isolationRequired("omabit.private-isolation-audit=required-v1", true)
}

package main

import (
	"reflect"
	"regexp"
	"strings"
)

func isolationRequired(cmdline string, other bool) bool {
	found := false
	for _, field := range strings.Fields(cmdline) {
		if strings.HasPrefix(field, "omabit.private-isolation-audit=") {
			need(!found && field == "omabit.private-isolation-audit=required-v1" && !other, "invalid private isolation audit selection")
			found = true
		}
	}
	return found
}

func inspectIsolation() {
	receipt, digest := readResult("private-chain-isolation.json")
	if receipt["ok"] != true {
		emit(map[string]any{"schema": 1, "ok": false, "isolation_sha256": digest, "raw_state_exported": false, "read_only_disk": true})
		return
	}
	summaries, peers := map[string]map[string]any{}, map[string]map[string]any{}
	hashes := map[string]any{}
	for _, label := range []string{"a", "b"} {
		var s, p string
		summaries[label], s = readResult("private-chain-" + label + ".json")
		peers[label], p = readResult("private-chain-peers-" + label + ".json")
		hashes[label] = map[string]any{"gateway": s, "peers": p}
	}
	out := auditIsolation(receipt, summaries, peers)
	out["isolation_sha256"] = digest
	out["receipts_sha256"] = hashes
	emit(out)
}

func liveIsolation(value map[string]any, summaries, peers map[string]map[string]any, labels []string) map[string]any {
	need(len(value) == len(labels), "live project count differs")
	out := map[string]any{}
	ids, objects, namespaces := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, label := range labels {
		row := obj(value[label])
		summary, peer := summaries[label], peers[label]
		project := hexValue(summary["project"], 32, "")
		anchor := obj(row["anchor"])
		anchorID := hexValue(anchor["Id"], 64, "")
		need(row["project"] == project && summary["anchor"] == anchorID, "live project owner differs")
		ns, ok := row["namespace"].(string)
		need(ok && regexp.MustCompile(`^net:\[[0-9]{1,20}\]$`).MatchString(ns) && peer["network_namespace"] == ns && !namespaces[ns], "concurrent namespace overlaps")
		namespaces[ns] = true
		need(anchor["Name"] == "/omabit-private-"+project+"-gateway" && obj(anchor["HostConfig"])["NetworkMode"] == "none", "live gateway attachment differs")
		live := func(container map[string]any) {
			id := hexValue(container["Id"], 64, "")
			need(!ids[id], "live container identity overlaps")
			ids[id] = true
			state := obj(container["State"])
			need(state["Running"] == true && state["OOMKilled"] == false, "concurrent container not running")
			bindings := obj(container["HostConfig"])["PortBindings"]
			need(bindings == nil || len(obj(bindings)) == 0, "unexpected concurrent publication")
			need(obj(obj(container["Config"])["Labels"])["org.omabit.qualification.private-chain"] == project, "live container project label differs")
		}
		live(anchor)
		ships := obj(row["ships"])
		need(len(ships) == 2, "live member count differs")
		members := map[string]any{}
		for n, ship := range []string{"zod", "nec"} {
			container := obj(ships[ship])
			live(container)
			need(container["Id"] == obj(obj(peer["ships"])[ship])["container"] && obj(container["HostConfig"])["NetworkMode"] == "container:"+anchorID, "live peer membership differs")
			suffix := "-boot"
			if ship == "nec" {
				suffix = "-resume"
			}
			need(container["Name"] == "/omabit-private-"+project+"-"+ship+suffix, "live member name differs")
			cmd := list(obj(container["Config"])["Cmd"])
			ports := map[string]string{"--http-port": []string{"18081", "18082"}[n], "-p": []string{"31337", "31338"}[n]}
			for key, expected := range ports {
				count := 0
				for i, arg := range cmd {
					if arg == key {
						need(i+1 < len(cmd) && cmd[i+1] == expected, "duplicate identity port differs")
						count++
					}
				}
				need(count == 1, "duplicate identity port missing")
			}
			members[ship] = container["Id"]
		}
		directories := obj(row["objects"])
		need(len(directories) == 4, "project directory count differs")
		publicObjects := map[string]any{}
		for _, name := range []string{"root", "chain", "zod", "nec"} {
			pair := list(directories[name])
			need(len(pair) == 2, "directory identity shape differs")
			for _, number := range pair {
				v, ok := number.(float64)
				need(ok && v > 0 && v < 9007199254740992 && v == float64(int64(v)), "directory identity invalid")
			}
			digest := hash(pair)
			need(!objects[digest], "concurrent directory object overlaps")
			objects[digest] = true
			publicObjects[name] = pair
		}
		out[label] = map[string]any{"project": project, "gateway": anchorID, "namespace": ns, "members": members, "objects": publicObjects}
	}
	return out
}

func auditIsolationExtension(value map[string]any, label string, entries []any) map[string]any {
	writes := list(value["writes"])
	need(len(writes) == 1, "isolation write count differs")
	write := obj(writes[0])
	need(write["phase"] == "project-"+label+"-isolation" && write["state"] == "acknowledged" && write["http_status"] == float64(200) && write["entry"] == entries[4], "isolation write acknowledgment differs")
	channels := list(value["channels"])
	need(len(channels) == 1 && reflect.DeepEqual(channels, value["channels_deleted"]), "isolation channel cleanup differs")
	channel, ok := channels[0].(string)
	need(ok && regexp.MustCompile(`^/~/channel/omabit-state-[0-9a-f]{32}$`).MatchString(channel), "isolation channel differs")
	observations := list(value["observations"])
	need(len(observations) == 3, "isolation observation count differs")
	cursor := float64(-1)
	for n, raw := range observations {
		observation := obj(raw)
		revision := 5
		if n == 0 {
			revision = 4
		}
		state := map[string]any{"revision": float64(revision), "entries": entries[:revision]}
		need(reflect.DeepEqual(observation["state"], state), "isolation application state differs")
		if n == 1 {
			need(observation["via"] == "scry", "isolation write scry missing")
			continue
		}
		need(observation["via"] == "subscription" && observation["channel"] == channel, "isolation subscription differs")
		facts, acks := 0, 0
		for _, rawEvent := range list(observation["events"]) {
			pair := list(rawEvent)
			need(len(pair) == 2, "isolation event shape differs")
			id, ok := pair[0].(float64)
			need(ok && id > cursor && id == float64(int64(id)), "isolation event replay/order differs")
			cursor = id
			event := obj(pair[1])
			need(event["err"] == nil && event["response"] != "quit", "isolation subscription failed")
			if event["response"] == "diff" {
				need(event["id"] == float64(1) && reflect.DeepEqual(event["json"], state), "isolation snapshot differs")
				facts++
			}
			if event["response"] == "subscribe" {
				need(event["id"] == float64(1) && event["ok"] == "ok", "isolation subscribe rejected")
				acks++
			}
		}
		need(facts == 1 && observation["cursor"] == cursor && (n != 0 || acks == 1), "isolation subscription evidence missing")
	}
	return map[string]any{"write_entry_sha256": hash(write["entry"]), "receipt_sha256": hash(write["receipt"]), "response_sha256": hexValue(write["response_sha256"], 64, ""), "state_sha256": hash(entries), "channel": channel, "cursor": cursor}
}

func auditIsolation(receipt map[string]any, summaries, peers map[string]map[string]any) map[string]any {
	need(receipt["ok"] == true && receipt["verified"] == true && receipt["first"] == true && receipt["second"] == true && len(list(receipt["cleanup_errors"])) == 0, "private project isolation incomplete")
	projects, extensions := map[string]any{}, map[string]any{}
	entries := map[string][]any{}
	seenKeys, seenEntries := map[string]bool{}, map[string]bool{}
	ready := obj(receipt["ready"])
	need(len(ready) == 2, "private readiness count differs")
	for _, label := range []string{"a", "b"} {
		summary, peer := summaries[label], peers[label]
		projects[label] = auditResults(summary, peer)
		need(reflect.DeepEqual(ready[label], summary["ready"]), "private readiness receipt differs")
		for _, raw := range list(obj(ready[label])["publicKeys"]) {
			for _, name := range []string{"auth", "crypt"} {
				key := hexValue(obj(raw)[name], 64, "0x")
				need(!seenKeys[key], "private projects reuse network keys")
				seenKeys[key] = true
			}
		}
		for _, raw := range list(obj(peer["application_state"])["writes"]) {
			entries[label] = append(entries[label], obj(raw)["entry"])
		}
		ext := obj(obj(receipt["extensions"])[label])
		writes := list(ext["writes"])
		need(len(writes) == 1, "extension write count differs")
		entries[label] = append(entries[label], obj(writes[0])["entry"])
		for _, value := range entries[label] {
			entry, ok := value.(string)
			need(ok && !seenEntries[entry], "project write identities overlap")
			seenEntries[entry] = true
		}
		extensions[label] = auditIsolationExtension(ext, label, entries[label])
	}
	need(summaries["a"]["project"] != summaries["b"]["project"] && obj(ready["a"])["genesis"] == obj(ready["b"])["genesis"], "private chain foundation/ownership differs")
	before := liveIsolation(obj(receipt["before"]), summaries, peers, []string{"a", "b"})
	after := liveIsolation(obj(receipt["after"]), summaries, peers, []string{"a", "b"})
	need(reflect.DeepEqual(before, after), "live project ownership changed during comparison")
	remaining := liveIsolation(obj(receipt["after_second_stopped"]), summaries, peers, []string{"a"})
	need(reflect.DeepEqual(remaining["a"], before["a"]), "first project changed after second stopped")
	crossed := list(receipt["cross_sessions"])
	need(len(crossed) == 4, "cross-project session coverage differs")
	for n, raw := range crossed {
		row := obj(raw)
		src, dst := "a", "b"
		if n >= 2 {
			src, dst = "b", "a"
		}
		need(row["source"] == src && row["target"] == dst && row["ship"] == []string{"zod", "nec"}[n%2] && row["dispatched"] == true && (row["status"] == float64(401) || row["status"] == float64(403)), "cross-project session accepted/missing")
	}
	states := list(receipt["states"])
	need(len(states) == 7, "independent state coverage differs")
	labels := []string{"a", "b", "a", "b", "b", "a", "a"}
	phases := []string{"initial", "initial", "after-a", "after-a", "after-b", "after-b", "after-b-stopped"}
	revisions := []int{4, 4, 5, 4, 5, 5, 5}
	for n, raw := range states {
		row := obj(raw)
		expected := map[string]any{"revision": float64(revisions[n]), "entries": entries[labels[n]][:revisions[n]]}
		need(row["project"] == labels[n] && row["phase"] == phases[n] && reflect.DeepEqual(row["state"], expected), "mutation crossed project or shutdown boundary")
	}
	return map[string]any{"schema": 1, "ok": true, "private_project_isolation": true, "read_only_disk": true, "raw_state_exported": false, "projects": projects, "live_ownership": before, "extensions": extensions, "cross_sessions_refused": 4, "independent_state_observations": 7, "first_survived_second_shutdown": true}
}

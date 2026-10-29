package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"reflect"
	"regexp"
	"strings"

	"golang.org/x/sys/unix"
)

func initialPublicationRequired(cmdline string, other bool) bool {
	found := false
	for _, field := range strings.Fields(cmdline) {
		if strings.HasPrefix(field, "omabit.initial-publication-audit=") {
			need(!found && !other && field == "omabit.initial-publication-audit=required-v1", "invalid initial publication audit selection")
			found = true
		}
	}
	return found
}

func auditInitialState(state map[string]any, baseline []any) map[string]any {
	need(len(baseline) == 4 && state["ok"] == true && state["initial_publication"] == true && state["writes_replayed"] == false, "initial application gates missing")
	entries := append([]any{}, baseline...)
	writes := list(state["writes"])
	need(len(writes) == 2, "initial application write count differs")
	receipts := map[string]bool{}
	for n, raw := range writes {
		w := obj(raw)
		entry, ok := w["entry"].(string)
		need(ok && regexp.MustCompile(`^omabit-state-[0-9a-f]{24}$`).MatchString(entry), "initial write identity differs")
		for _, previous := range entries {
			need(previous != entry, "initial write replayed")
		}
		receipt, ok := w["receipt"].(string)
		need(ok && regexp.MustCompile(`^omabit-receipt-[0-9a-f]{32}$`).MatchString(receipt) && !receipts[receipt], "initial write receipt differs")
		receipts[receipt] = true
		need(w["phase"] == []string{"initial-published-connected", "initial-published-fresh-subscription"}[n] && w["state"] == "acknowledged" && w["http_status"] == float64(200), "initial write outcome differs")
		hexValue(w["response_sha256"], 64, "")
		entries = append(entries, entry)
	}
	channels := list(state["channels"])
	need(len(channels) == 2 && channels[0] != channels[1] && reflect.DeepEqual(channels, state["channels_deleted"]), "initial channel cleanup differs")
	for _, value := range channels {
		channel, ok := value.(string)
		need(ok && regexp.MustCompile(`^/~/channel/omabit-state-[0-9a-f]{32}$`).MatchString(channel), "initial channel identity differs")
	}
	observations := list(state["observations"])
	need(len(observations) == 7, "initial application observation count differs")
	revisions := []int{4, 5, 5, 5, 6, 6, 6}
	via := []string{"subscription", "scry", "subscription", "subscription", "scry", "subscription", "scry"}
	last := map[any]float64{}
	for n, raw := range observations {
		observation := obj(raw)
		current := map[string]any{"revision": float64(revisions[n]), "entries": entries[:revisions[n]]}
		need(observation["via"] == via[n] && reflect.DeepEqual(observation["state"], current), "initial application state differs")
		if via[n] == "scry" {
			continue
		}
		channel := channels[0]
		if n >= 3 {
			channel = channels[1]
		}
		need(observation["channel"] == channel, "initial subscription binding differs")
		cursor, seen := last[channel]
		if !seen {
			cursor = -1
		}
		facts, acks := 0, 0
		for _, rawEvent := range list(observation["events"]) {
			pair := list(rawEvent)
			need(len(pair) == 2, "initial event shape differs")
			id, ok := pair[0].(float64)
			need(ok && id > cursor && id < 9007199254740992 && id == float64(int64(id)), "initial event replay/order differs")
			cursor = id
			event := obj(pair[1])
			need(event["err"] == nil, "initial subscription error")
			switch event["response"] {
			case "diff":
				need(event["id"] == float64(1) && reflect.DeepEqual(event["json"], current), "initial snapshot differs")
				facts++
			case "subscribe":
				need(event["id"] == float64(1) && event["ok"] == "ok", "initial subscribe rejected")
				acks++
			default:
				need(false, "unexpected initial subscription event")
			}
		}
		expectedAcks := 0
		if !seen {
			expectedAcks = 1
		}
		need(facts == 1 && acks == expectedAcks && observation["cursor"] == cursor, "initial snapshot/cursor missing")
		last[channel] = cursor
	}
	return map[string]any{"ok": true, "baseline_revision": 4, "final_revision": 6, "writes": 2, "subscriptions": 4, "scries": 3, "channels_deleted": 2, "writes_replayed": false, "state_sha256": hash(entries), "raw_values_exported": false}
}

func auditInitialPublication(summary, peers, report map[string]any) map[string]any {
	need(summary["ok"] == true && peers["ok"] == true, "initial publication requires completed peers")
	project, anchor := hexValue(summary["project"], 32, ""), hexValue(summary["anchor"], 64, "")
	need(report["version"] == float64(1) && report["ok"] == true && report["project"] == project && peers["project"] == project && report["gateway"] == anchor, "initial publication identity differs")
	need(report["initial_start"] == true && report["original_sessions_reused"] == true && report["error_type"] == nil && report["error_sha256"] == nil && len(list(report["cleanup_errors"])) == 0, "initial publication gates missing")
	ready := obj(summary["ready"])
	gateway := obj(obj(summary["gateway_proof"])["gateway"])
	head := hexValue(ready["headHash"], 64, "0x")
	need(head != "0x"+strings.Repeat("0", 64) && ready["block"] == "0x50" && ready["initialNetwork"] == "published-v1" && gateway["headHash"] == head && reflect.DeepEqual(gateway["initialKeySetup"], ready), "initial publication readiness differs")
	p, client := obj(summary["publication"]), obj(report["publication"])
	need(p["initial_start"] == true, "initial publication mode missing")
	out := auditPublicationView(project, anchor, p, client, 8)
	cookies := obj(report["cookie_sha256"])
	need(len(cookies) == 2 && hexValue(cookies["zod"], 64, "") != hexValue(cookies["nec"], 64, ""), "initial session bindings differ")
	baseline := list(report["baseline_entries"])
	previous := list(obj(peers["application_state"])["writes"])
	need(len(baseline) == 4 && len(previous) == 4, "initial application baseline missing")
	for n, v := range baseline {
		need(v == obj(previous[n])["entry"], "initial application baseline differs")
	}
	state := obj(report["application_state"])
	out["application_state"] = auditInitialState(state, baseline)
	channels := list(state["channels"])
	streams, puts := map[any]int{}, map[any]int{}
	for _, raw := range list(client["requests"]) {
		request := obj(raw)
		path, ok := request["path"].(string)
		need(ok, "initial request path missing")
		if !strings.HasPrefix(path, "/~/channel/") {
			continue
		}
		need(path == channels[0] || path == channels[1], "initial request channel differs")
		if request["method"] == "GET" {
			streams[path]++
		} else {
			puts[path]++
		}
	}
	for _, channel := range channels {
		need(streams[channel] >= 2 && puts[channel] == 4, "initial channel request coverage differs")
	}
	out["initial_start"] = true
	out["original_sessions_reused"] = true
	out["head_hash"] = head
	return out
}

// Only the two owned, bounded guest session files are hashed. Their contents
// and digests do not enter the public projection.
func verifyInitialCookies(root string, cookies map[string]any) {
	fd, err := unix.Open(root, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	must(err)
	defer unix.Close(fd)
	var dir unix.Stat_t
	must(unix.Fstat(fd, &dir))
	need(dir.Uid == 0 && dir.Mode&0777 == 0700, "private session directory differs")
	for _, ship := range []string{"zod", "nec"} {
		func() {
			child, err := unix.Openat(fd, ship+".cookie", unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
			must(err)
			f := os.NewFile(uintptr(child), ship)
			defer f.Close()
			var st unix.Stat_t
			must(unix.Fstat(child, &st))
			need(st.Mode&unix.S_IFMT == unix.S_IFREG && st.Mode&0777 == 0600 && st.Uid == 0 && st.Nlink == 1 && st.Size > 0 && st.Size <= 8192, "private session file differs")
			raw, err := io.ReadAll(io.LimitReader(f, 8193))
			must(err)
			need(int64(len(raw)) == st.Size, "private session size changed")
			digest := sha256.Sum256(raw)
			need(hex.EncodeToString(digest[:]) == cookies[ship], "private session content changed")
		}()
	}
}

func inspectInitialPublication(summary, peers map[string]any) map[string]any {
	report, digest := readPublicReceipt("/newroot/opt/omabit-lab/results/initial-publication.json")
	out := auditInitialPublication(summary, peers, report)
	project := hexValue(summary["project"], 32, "")
	verifyInitialCookies("/newroot/opt/omabit-lab/results/initial-publication-clients-"+project, obj(report["cookie_sha256"]))
	out["report_sha256"] = digest
	out["session_files_matched"] = true
	return out
}

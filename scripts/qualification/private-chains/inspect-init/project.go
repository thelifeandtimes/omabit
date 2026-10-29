package main

import (
	"fmt"
	"reflect"
	"regexp"
)

func hexValue(v any, digits int, prefix string) string {
	s, ok := v.(string)
	need(ok && regexp.MustCompile("^"+regexp.QuoteMeta(prefix)+fmt.Sprintf("[0-9a-f]{%d}$", digits)).MatchString(s), "public identifier differs")
	return s
}
func cleanExit(value any) map[string]any {
	v := obj(value)
	need(v["Running"] == false && v["ExitCode"] == float64(0) && v["OOMKilled"] == false, "unclean gateway shutdown")
	return map[string]any{"Running": false, "ExitCode": 0, "OOMKilled": false}
}
func auditResults(summary, peers map[string]any) map[string]any {
	project := hexValue(summary["project"], 32, "")
	need(peers["project"] == project, "peer project differs")
	need(summary["ok"] == true && peers["ok"] == true && peers["transport_passed"] == true && peers["restart_same_pier"] == true && peers["restart_kept_sessions"] == true && peers["cross_ship_sessions_refused"] == true, "peer gates missing")
	attempts := list(peers["attempts"])
	need(len(attempts) == 7, "message attempt count differs")
	positive, negative := 0, 0
	for _, v := range attempts {
		a := obj(v)
		switch a["state"] {
		case "acknowledged":
			positive++
		case "rejected":
			negative++
		default:
			panic("unresolved message")
		}
	}
	need(positive == 6 && negative == 1, "message outcomes differ")
	stops := list(peers["stop_records"])
	need(len(stops) == 3, "shutdown count differs")
	lens, term := 0, 0
	publicStops := []any{}
	for _, v := range stops {
		s := obj(v)
		need(s["passed"] == true && s["acknowledged"] == true && s["dispatched"] == true && s["exit"] == float64(0) && s["oom"] == false && s["running"] == false && s["project"] == project, "unclean peer shutdown")
		switch s["mode"] {
		case "lens-exit":
			lens++
		case "sigterm":
			term++
		default:
			panic("shutdown mode differs")
		}
		publicStops = append(publicStops, map[string]any{"passed": true, "acknowledged": true, "dispatched": true, "exit": 0, "oom": false, "running": false, "project": project, "container": hexValue(s["container"], 64, ""), "mode": s["mode"]})
	}
	need(lens == 2 && term == 1, "shutdown modes differ")
	restart := obj(peers["restart"])
	old, newID := hexValue(restart["original"], 64, ""), hexValue(restart["replacement"], 64, "")
	need(old != newID, "receiver container not replaced")
	publicRestart := map[string]any{"original": old, "replacement": newID}
	for _, name := range []string{"pier_device", "pier_inode"} {
		n, ok := restart[name].(float64)
		need(ok && n > 0 && n < 9007199254740992 && n == float64(int64(n)), "pier identity differs")
		publicRestart[name] = n
	}
	state := auditState(obj(peers["application_state"]))
	proof := obj(summary["gateway_proof"])
	gateway := obj(proof["gateway"])
	ready := obj(summary["ready"])
	need(proof["passed"] == true && gateway["signalShutdown"] == true && gateway["head"] == "0x50" && ready["chainId"] == "0x539" && ready["externalRpcMutationsAllowed"] == false, "chain proof differs")
	publicKeys, cases := []any{}, []any{}
	keys := list(proof["publicKeys"])
	validation := list(proof["cases"])
	need(len(keys) == 2 && len(validation) == 2 && reflect.DeepEqual(ready["publicKeys"], proof["publicKeys"]), "key validation set differs")
	for n, ship := range []string{"~zod", "~nec"} {
		key := obj(keys[n])
		verdict := obj(validation[n])
		need(key["ship"] == ship && key["number"] == float64(n) && verdict["ship"] == ship && verdict["correct"] == true && verdict["wrongLifeRefused"] == true && verdict["wrongPublicKeyRefused"] == true, "key verdict differs")
		publicKeys = append(publicKeys, map[string]any{"ship": ship, "number": n, "auth": hexValue(key["auth"], 64, "0x"), "crypt": hexValue(key["crypt"], 64, "0x")})
		cases = append(cases, map[string]any{"ship": ship, "correct": true, "wrongLifeRefused": true, "wrongPublicKeyRefused": true})
	}
	calls := map[string]int{}
	for _, v := range list(gateway["chainCalls"]) {
		c := obj(v)
		method, ok := c["method"].(string)
		need(ok, "RPC method missing")
		switch method {
		case "eth_blockNumber", "eth_getBlockByNumber", "eth_getLogs":
			calls[method]++
		default:
			panic("unqualified RPC method")
		}
	}
	ns, ok := peers["network_namespace"].(string)
	need(ok && regexp.MustCompile(`^net:\[[0-9]{1,20}\]$`).MatchString(ns), "network namespace evidence differs")
	return map[string]any{"schema": 1, "project": project, "ok": true, "read_only_disk": true, "raw_state_exported": false, "messages": map[string]int{"acknowledged": positive, "intentional_nack": negative}, "stops": publicStops, "restart": publicRestart, "state": state, "gateway": map[string]any{"chain_id": "0x539", "genesis": hexValue(ready["genesis"], 64, "0x"), "head": "0x50", "public_keys": publicKeys, "key_validation": cases, "readonly_rpc_counts": calls, "clean_shutdown": cleanExit(summary["gateway_stop"])}, "network_namespace": ns, "cross_ship_sessions_refused": true}
}

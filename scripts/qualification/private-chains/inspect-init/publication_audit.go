package main

import (
	"fmt"
	"net/netip"
	"reflect"
	"regexp"
	"strings"
)

func publicationRequired(cmdline string, peers bool) bool {
	count := 0
	for _, word := range strings.Fields(cmdline) {
		if strings.HasPrefix(word, "omabit.private-publication-audit=") {
			need(word == "omabit.private-publication-audit=required-v1", "unknown private publication audit mode")
			count++
		}
	}
	need(count <= 1 && (count == 0 || peers), "invalid private publication audit selection")
	return count == 1
}

func auditPublicationNetwork(v map[string]any, project, network, anchor string) {
	n, g := obj(v["network"]), obj(v["gateway"])
	name := "omabit-private-" + project
	need(n["Id"] == network && n["Name"] == name && n["Driver"] == "bridge" && n["Scope"] == "local" && n["Internal"] == false && n["EnableIPv4"] == true && n["EnableIPv6"] == false && n["Ingress"] == false, "published bridge identity differs")
	need(obj(n["Labels"])["org.omabit.qualification.private-chain"] == project, "published bridge owner differs")
	expectedOptions := map[string]any{"com.docker.network.bridge.host_binding_ipv4": "127.0.0.1", "com.docker.network.bridge.gateway_mode_ipv4": "nat"}
	need(reflect.DeepEqual(n["Options"], expectedOptions), "published bridge options differ")
	ipam := obj(n["IPAM"])
	need(ipam["Driver"] == "default" && (ipam["Options"] == nil || len(obj(ipam["Options"])) == 0), "published bridge IPAM differs")
	configs := list(ipam["Config"])
	need(len(configs) == 1, "published bridge address count differs")
	config := obj(configs[0])
	need(len(config) == 2, "published bridge address fields differ")
	subnetString, ok := config["Subnet"].(string)
	need(ok, "published subnet missing")
	gatewayString, ok := config["Gateway"].(string)
	need(ok, "published gateway missing")
	subnet, err := netip.ParsePrefix(subnetString)
	need(err == nil && subnet.Addr().Is4() && subnet == subnet.Masked(), "published subnet differs")
	address, err := netip.ParseAddr(gatewayString)
	need(err == nil && subnet.Contains(address) && address != subnet.Addr(), "published gateway address differs")
	private := false
	for _, block := range []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"} {
		p := netip.MustParsePrefix(block)
		private = private || p.Contains(subnet.Addr()) && p.Bits() <= subnet.Bits()
	}
	need(private, "published subnet is not private")
	members := obj(n["Containers"])
	need(len(members) == 1 && members[anchor] != nil, "published bridge member differs")
	cfg, host, live, state := obj(g["Config"]), obj(g["HostConfig"]), obj(g["NetworkSettings"]), obj(g["State"])
	image := "docker.io/tloncorp/vere@sha256:47fbcf0710ded939931ce3c6408864cee5e8848c5765da0b51a90ca0b1ab1afd"
	need(g["Id"] == anchor && cfg["Image"] == image && (g["Image"] == "sha256:d6cc5107ed7fa92174e336af9c7f25ee433073bd9961e1c3ed23a225b866b245" || g["Image"] == strings.Split(image, "@")[1]), "published gateway image differs")
	need(obj(cfg["Labels"])["org.omabit.qualification.private-chain"] == project && host["NetworkMode"] == network && state["Running"] == true && state["OOMKilled"] == false, "published gateway state differs")
	wanted := map[string]any{"18081/tcp": []any{map[string]any{"HostIp": "127.0.0.1", "HostPort": "18381"}}, "18082/tcp": []any{map[string]any{"HostIp": "127.0.0.1", "HostPort": "18382"}}}
	need(reflect.DeepEqual(host["PortBindings"], wanted), "published host ports differ")
	ports := map[string]any{}
	for key, value := range obj(live["Ports"]) {
		if value != nil {
			if rows, ok := value.([]any); !ok || len(rows) > 0 {
				ports[key] = value
			}
		}
	}
	need(reflect.DeepEqual(ports, wanted), "active published ports differ")
	networks := obj(live["Networks"])
	need(len(networks) == 1 && obj(networks[name])["NetworkID"] == network, "published gateway network differs")
}

func auditPublication(resume, peers map[string]any) map[string]any {
	need(resume["ok"] == true && peers["ok"] == true, "publication requires passed retained recovery")
	project, anchor := hexValue(resume["project"], 32, ""), hexValue(resume["container"], 64, "")
	p, client := obj(resume["publication"]), obj(peers["publication"])
	return auditPublicationView(project, anchor, p, client, 7)
}

// Shared transport checks; initialization creates both channels, while replacement
// resumes one saved channel. The caller binds the different mutation counts.
func auditPublicationView(project, anchor string, p, client map[string]any, expectedPuts int) map[string]any {
	need(expectedPuts == 7 || expectedPuts == 8, "invalid publication channel coverage")
	network := hexValue(p["network"], 64, "")
	ports := map[string]any{"18081": float64(18381), "18082": float64(18382)}
	need(p["loopback_only"] == true && p["name"] == "omabit-private-"+project && reflect.DeepEqual(p["ports"], ports) && client["network"] == network && reflect.DeepEqual(client["ports"], ports) && client["direct_client"] == true, "publication review binding differs")
	for _, key := range []string{"first_observation", "last_observation"} {
		auditPublicationNetwork(obj(p[key]), project, network, anchor)
	}
	refusals := list(client["refusals"])
	need(len(refusals) == 5, "unpublished endpoint observations missing")
	seen := map[string]bool{}
	for _, value := range refusals {
		v := obj(value)
		kind, ok := v["kind"].(string)
		need(ok && !seen[kind] && v["refused"] == true, "unpublished refusal differs")
		seen[kind] = true
		port := positiveInteger(v["port"])
		need(port <= 65535, "unpublished port differs")
		switch kind {
		case "rpc":
			need(v["host"] == "127.0.0.1" && port == 25432, "RPC refusal differs")
		case "admin-zod", "admin-nec":
			need(v["host"] == "127.0.0.1" && port != 18381 && port != 18382, "admin refusal differs")
		case "ipv6-zod":
			need(v["host"] == "::1" && port == 18381, "IPv6 refusal differs")
		case "ipv6-nec":
			need(v["host"] == "::1" && port == 18382, "IPv6 refusal differs")
		default:
			need(false, "unknown unpublished endpoint")
		}
	}
	requests := list(client["requests"])
	need(len(requests) > 0 && len(requests) <= 200, "direct request count differs")
	hostReads, inventory, crossed := map[string]bool{}, map[string]bool{}, map[string]bool{}
	posts, streams, scries, puts := 0, 0, 0, 0
	for _, value := range requests {
		q := obj(value)
		ship, ok := q["ship"].(string)
		need(ok && (ship == "zod" || ship == "nec"), "direct client ship differs")
		port := float64(18381)
		if ship == "nec" {
			port = 18382
		}
		need(q["port"] == port, "direct client endpoint differs")
		path, ok := q["path"].(string)
		need(ok, "direct path missing")
		status := positiveInteger(q["status"])
		need(status <= 599, "direct HTTP status differs")
		method, stream, cross := q["method"], q["stream"], q["crossed_session"]
		switch {
		case path == "/~/host":
			need(method == "GET" && stream == false && cross == false, "host request differs")
			if status == 200 {
				hostReads[ship] = true
			}
		case path == "/~/scry/hood/kiln/pikes.json":
			need(method == "GET" && stream == false, "inventory request differs")
			if cross == true {
				need(status == 401 || status == 403, "crossed published session accepted")
				crossed[ship] = true
			} else {
				need(cross == false && status == 200, "original published session refused")
				inventory[ship] = true
			}
		case path == "/~/scry/omabit-state/state.json":
			need(ship == "nec" && method == "GET" && stream == false && cross == false && status == 200, "published state scry differs")
			scries++
		case path == "/spider/base/ted-eval/khan-eval/noun":
			need(ship == "zod" && method == "POST" && stream == false && cross == false && status == 200, "published write differs")
			posts++
		case regexp.MustCompile(`^/~/channel/omabit-state-[0-9a-f]{32}$`).MatchString(path):
			need(ship == "nec" && cross == false, "published channel differs")
			if method == "GET" {
				need(stream == true && status == 200, "published stream absent")
				streams++
			} else {
				need(method == "PUT" && stream == false && (status == 200 || status == 204), "published channel mutation differs")
				puts++
			}
		default:
			need(false, "unexpected direct client request")
		}
	}
	// Four cursor acknowledgments, two deletes, and the caller-bound subscriptions.
	if !(len(hostReads) == 2 && len(inventory) == 2 && len(crossed) == 2 && posts == 2 && streams >= 4 && scries >= 3 && puts == expectedPuts) {
		panic(publicationCoverageFailure{len(hostReads), len(inventory), len(crossed), posts, streams, scries, puts})
	}
	return map[string]any{"ok": true, "project": project, "gateway": anchor, "network": network, "ports": ports, "direct_client": true, "requests": len(requests), "peer_writes": posts, "stream_reads": streams, "state_scries": scries, "channel_puts": puts, "unpublished_refusals": 5, "relogin_performed": false, "raw_state_exported": false}
}

func inspectPublication() map[string]any {
	resume, resumeSHA := readResult("gateway-resume.json")
	peers, peersSHA := readResult("gateway-peer-recovery.json")
	out := auditPublication(resume, peers)
	out["resume_sha256"] = resumeSHA
	out["peers_sha256"] = peersSHA
	return out
}

// Fixed bounded counters explain coverage refusal without disclosing request
// bodies, sessions, channel names or any application values.
type publicationCoverageFailure struct{ Hosts, Inventories, Crossed, Writes, Streams, Scries, ChannelPuts int }

func (publicationCoverageFailure) Error() string { return "direct publication coverage incomplete" }
func readerFailureProjection(value any) map[string]any {
	out := diagnosticSummary([]byte(fmt.Sprint(value)))
	if v, ok := value.(publicationCoverageFailure); ok {
		fields := map[string]int{"hosts": v.Hosts, "inventories": v.Inventories, "crossed": v.Crossed, "writes": v.Writes, "streams": v.Streams, "scries": v.Scries, "channel_puts": v.ChannelPuts}
		valid := true
		for _, n := range fields {
			valid = valid && n >= 0 && n <= 200
		}
		if valid {
			out["publication_coverage"] = fields
		}
	}
	return out
}

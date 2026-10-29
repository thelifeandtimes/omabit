//go:build linux && amd64

package runtimequalify

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

func privateNetworkFixture(t *testing.T) (privatechain.Plan, privatechain.Workspace, privatechain.Gateway, privateGatewayNetworkContract, map[string]any) {
	t.Helper()
	raw, err := os.ReadFile("../privatechain/testdata/qualified-foundation.json")
	if err != nil {
		t.Fatal(err)
	}
	f, err := privatechain.DecodeFoundation(raw)
	if err != nil {
		t.Fatal(err)
	}
	p, err := privatechain.Build(privatechain.Request{Label: "Private network fixture", Foundation: f, ConfirmSharedAdministration: true,
		Members: []privatechain.MemberRequest{{InstanceID: "sender", Ship: "~zod", HTTPPort: 18381}, {InstanceID: "receiver", Ship: "~nec", HTTPPort: 18382}}}, privatechain.Reservations{})
	if err != nil {
		t.Fatal(err)
	}
	id := func(n uint64) pierstore.DirectoryIdentity {
		return pierstore.DirectoryIdentity{Device: 12, Inode: n, Birth: "1000000000:1"}
	}
	b := privatechain.Binding{Mode: privatechain.Mode, RunnerID: "runner-private", ProjectID: "private-project-network", Generation: 1, FoundationDigest: p.FoundationDigest}
	w, err := privatechain.NewWorkspace(p, b, 1, "workspace-operation", id(9))
	if err != nil {
		t.Fatal(err)
	}
	w.Phase, w.Root = "observed", id(10)
	w.Observation = &privatechain.WorkspaceObservation{ObservedAt: time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC), Parent: id(9), Root: id(10), PiersRoot: id(11), Mounts: []pierstore.DirectoryIdentity{id(12), id(13)}, Complete: true, Private: true, Empty: true, Synced: true}
	i := privatechain.RuntimeInputInspection{Format: privatechain.RuntimeInputFormat, ManifestSHA256: strings.Repeat("1", 64), FoundationDigest: p.FoundationDigest, Files: 12352, Bytes: 834035190}
	g, err := privatechain.NewGateway(p, w, "gateway-operation", "engine-private-fixture", i)
	if err != nil {
		t.Fatal(err)
	}
	c, err := newPrivateGatewayNetworkContract(p, w, g)
	if err != nil {
		t.Fatal(err)
	}
	n := map[string]any{
		"Id": strings.Repeat("a", 64), "Name": "urbit-private-project-network-g1-bridge", "Driver": "bridge", "Scope": "local",
		"Internal": false, "EnableIPv4": true, "EnableIPv6": false, "Attachable": false, "Ingress": false, "ConfigOnly": false,
		"ConfigFrom": map[string]any{"Network": ""}, "Options": bridgeOptions(), "Labels": c.labels(),
		"IPAM":       map[string]any{"Driver": "default", "Options": nil, "Config": []any{map[string]any{"Subnet": "172.22.0.0/16", "Gateway": "172.22.0.1"}}},
		"Containers": map[string]any{},
	}
	return p, w, g, c, n
}

func privateNetworkBytes(t *testing.T, n map[string]any) []byte {
	t.Helper()
	b, err := json.Marshal([]any{n})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func privateNetworkAttach(n map[string]any) map[string]any {
	m := map[string]any{"Name": "urbit-private-project-network-g1-gateway", "EndpointID": strings.Repeat("c", 64), "IPv4Address": "172.22.0.2/16", "IPv6Address": ""}
	n["Containers"] = map[string]any{strings.Repeat("b", 64): m}
	return m
}

func TestPrivateGatewayNetworkContractSelection(t *testing.T) {
	p, w, g, c, _ := privateNetworkFixture(t)
	args := c.networkArgs()
	if strings.Join(args[:8], " ") != "network create --driver bridge --internal=false --ipv4=true --ipv6=false --label" || args[len(args)-1] != "urbit-private-project-network-g1-bridge" {
		t.Fatal(args)
	}
	if !reflect.DeepEqual(args, c.networkArgs()) {
		t.Fatal("command is nondeterministic")
	}
	for _, forbidden := range []string{"--attachable", "--internal=true", "--ipv6=true", "--publish", "--network", "host"} {
		for _, a := range args {
			if a == forbidden {
				t.Fatal("unexpected network authority", a)
			}
		}
	}
	labels := c.labels()
	labels[projectLabel+".operation"] = "changed"
	if c.labels()[projectLabel+".operation"] != "gateway-operation" {
		t.Fatal("labels alias caller memory")
	}
	for name, change := range map[string]func(*privatechain.Gateway){
		"other-workspace":      func(g *privatechain.Gateway) { g.Selection.WorkspaceOperationID = "other" },
		"other-project":        func(g *privatechain.Gateway) { g.Selection.Binding.ProjectID = "private-project-other" },
		"other-plan":           func(g *privatechain.Gateway) { g.Selection.PlanDigest = strings.Repeat("f", 64) },
		"input-authority":      func(g *privatechain.Gateway) { g.Selection.Inputs.RuntimeAuthorized = true },
		"unobserved-workspace": func(g *privatechain.Gateway) { g.Selection.WorkspaceDigest = "" },
	} {
		t.Run(name, func(t *testing.T) {
			bad := g
			change(&bad)
			if _, err := newPrivateGatewayNetworkContract(p, w, bad); err == nil {
				t.Fatal("unbound selection admitted")
			}
		})
	}
	other := g
	other.Selection.OperationID = "next-operation"
	d, err := newPrivateGatewayNetworkContract(p, w, other)
	if err != nil || reflect.DeepEqual(c.labels(), d.labels()) {
		t.Fatal("operation identity lost", err)
	}
}

func TestPrivateGatewayNetworkObservedAllocation(t *testing.T) {
	_, _, _, c, n := privateNetworkFixture(t)
	id := n["Id"].(string)
	before, err := c.inspectNetwork(privateNetworkBytes(t, n), id, "")
	if err != nil || before.Subnet.String() != "172.22.0.0/16" || before.Gateway.String() != "172.22.0.1" || before.Address.IsValid() {
		t.Fatal(before, err)
	}
	privateNetworkAttach(n)
	after, err := c.inspectNetwork(privateNetworkBytes(t, n), id, strings.Repeat("b", 64))
	if err != nil || after.Subnet != before.Subnet || after.Gateway != before.Gateway || after.Address.String() != "172.22.0.2" || after.EndpointID != strings.Repeat("c", 64) {
		t.Fatal(after, err)
	}
	if _, err := c.inspectNetwork(privateNetworkBytes(t, n), id, ""); err == nil {
		t.Fatal("attached network treated as empty")
	}
	n["Containers"] = nil
	if _, err := c.inspectNetwork(privateNetworkBytes(t, n), id, strings.Repeat("b", 64)); err == nil {
		t.Fatal("missing gateway attachment admitted")
	}
}

func TestPrivateGatewayNetworkRejectsChangedOwnershipAndAllocation(t *testing.T) {
	mutations := map[string]func(map[string]any){
		"id":     func(n map[string]any) { n["Id"] = strings.Repeat("d", 64) },
		"name":   func(n map[string]any) { n["Name"] = "urbit-private-project-other-g1-bridge" },
		"driver": func(n map[string]any) { n["Driver"] = "overlay" },
		"scope":  func(n map[string]any) { n["Scope"] = "swarm" },
		"labels": func(n map[string]any) {
			n["Labels"].(map[string]string)[projectLabel+".project"] = "private-project-other"
		},
		"old-operation": func(n map[string]any) { n["Labels"].(map[string]string)[projectLabel+".operation"] = "old-operation" },
		"inputs": func(n map[string]any) {
			n["Labels"].(map[string]string)[projectLabel+".inputs"] = strings.Repeat("e", 64)
		},
		"extra-label": func(n map[string]any) { n["Labels"].(map[string]string)["foreign"] = "true" },
		"public-binding": func(n map[string]any) {
			n["Options"].(map[string]string)["com.docker.network.bridge.host_binding_ipv4"] = "0.0.0.0"
		},
		"routed": func(n map[string]any) {
			n["Options"].(map[string]string)["com.docker.network.bridge.gateway_mode_ipv4"] = "routed"
		},
		"config-source":      func(n map[string]any) { n["ConfigFrom"] = map[string]string{"Network": "foreign"} },
		"ipam-driver":        func(n map[string]any) { n["IPAM"].(map[string]any)["Driver"] = "external" },
		"ipam-options":       func(n map[string]any) { n["IPAM"].(map[string]any)["Options"] = map[string]string{"foreign": "true"} },
		"missing-allocation": func(n map[string]any) { n["IPAM"].(map[string]any)["Config"] = nil },
		"multiple-subnets": func(n map[string]any) {
			i := n["IPAM"].(map[string]any)
			i["Config"] = append(i["Config"].([]any), map[string]string{"Subnet": "10.0.0.0/24", "Gateway": "10.0.0.1"})
		},
		"aux-address": func(n map[string]any) {
			n["IPAM"].(map[string]any)["Config"].([]any)[0].(map[string]any)["AuxiliaryAddresses"] = "foreign"
		},
		"foreign-member": func(n map[string]any) {
			n["Containers"].(map[string]any)[strings.Repeat("d", 64)] = map[string]string{}
		},
	}
	for _, field := range []string{"Internal", "EnableIPv4", "EnableIPv6", "Attachable", "Ingress", "ConfigOnly"} {
		mutations["missing-"+field] = func(n map[string]any) { delete(n, field) }
		mutations["changed-"+field] = func(n map[string]any) { n[field] = !n[field].(bool) }
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			_, _, _, c, n := privateNetworkFixture(t)
			mutate(n)
			if _, err := c.inspectNetwork(privateNetworkBytes(t, n), strings.Repeat("a", 64), ""); err == nil {
				t.Fatal("altered network admitted")
			}
		})
	}
	for _, pair := range [][2]string{
		{"172.22.0.1/16", "172.22.0.1"}, {"172.22.0.0/16", "172.23.0.1"}, {"172.22.0.0/16", "172.22.0.0"}, {"172.22.0.0/16", "172.22.255.255"},
		{"172.0.0.0/8", "172.22.0.1"}, {"100.64.0.0/10", "100.64.0.1"}, {"192.168.0.0/15", "192.168.0.1"}, {"127.0.0.0/8", "127.0.0.1"},
		{"8.0.0.0/8", "8.0.0.1"}, {"::ffff:172.22.0.0/112", "::ffff:172.22.0.1"}, {"fd00::/64", "fd00::1"}, {"10.0.0.0/31", "10.0.0.1"},
	} {
		t.Run(pair[0]+"-"+pair[1], func(t *testing.T) {
			_, _, _, c, n := privateNetworkFixture(t)
			n["IPAM"].(map[string]any)["Config"] = []any{map[string]string{"Subnet": pair[0], "Gateway": pair[1]}}
			if _, err := c.inspectNetwork(privateNetworkBytes(t, n), strings.Repeat("a", 64), ""); err == nil {
				t.Fatal("unsupported address allocation admitted")
			}
		})
	}
}

func TestPrivateGatewayNetworkRejectsWrongAttachment(t *testing.T) {
	for field, values := range map[string][]string{
		"Name":        {"urbit-private-project-other-g1-gateway", ""},
		"EndpointID":  {"", strings.Repeat("0", 64), strings.Repeat("C", 64)},
		"IPv4Address": {"172.22.0.2/24", "172.22.0.1/16", "172.22.0.0/16", "172.22.255.255/16", "172.23.0.2/16", "::ffff:172.22.0.2/112"},
		"IPv6Address": {"fd00::2/64"},
	} {
		for _, v := range values {
			t.Run(field+"-"+v, func(t *testing.T) {
				_, _, _, c, n := privateNetworkFixture(t)
				member := privateNetworkAttach(n)
				member[field] = v
				if _, err := c.inspectNetwork(privateNetworkBytes(t, n), strings.Repeat("a", 64), strings.Repeat("b", 64)); err == nil {
					t.Fatal("changed attachment admitted")
				}
			})
		}
	}
	_, _, _, c, n := privateNetworkFixture(t)
	privateNetworkAttach(n)
	for _, gateway := range []string{strings.Repeat("a", 64), strings.Repeat("d", 64), strings.Repeat("0", 64), "short"} {
		if _, err := c.inspectNetwork(privateNetworkBytes(t, n), strings.Repeat("a", 64), gateway); err == nil {
			t.Fatal("unknown or invalid gateway admitted")
		}
	}
	for _, data := range [][]byte{nil, []byte("null"), []byte("[]"), []byte("[{},{}]"), append(privateNetworkBytes(t, n), []byte("[]")...), []byte(strings.Repeat(" ", 1<<20+1))} {
		if _, err := c.inspectNetwork(data, strings.Repeat("a", 64), strings.Repeat("b", 64)); err == nil {
			t.Fatal("invalid inspection response admitted")
		}
	}
}

func TestPrivateGatewayNetworkAcceptsRFC1918AllocationEdges(t *testing.T) {
	for _, pair := range [][2]string{{"10.0.0.0/8", "10.0.0.1"}, {"172.16.0.0/12", "172.31.255.254"}, {"192.168.0.0/16", "192.168.255.254"}, {"192.168.255.252/30", "192.168.255.253"}} {
		t.Run(pair[0], func(t *testing.T) {
			_, _, _, c, n := privateNetworkFixture(t)
			n["IPAM"].(map[string]any)["Config"] = []any{map[string]string{"Subnet": pair[0], "Gateway": pair[1]}}
			if _, err := c.inspectNetwork(privateNetworkBytes(t, n), strings.Repeat("a", 64), ""); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPrivateGatewayNetworkRefusesAmbiguousInspection(t *testing.T) {
	_, _, _, c, n := privateNetworkFixture(t)
	raw := string(privateNetworkBytes(t, n))
	for name, changed := range map[string]string{
		"duplicate-attachment": strings.Replace(raw, `"Containers":{}`, `"Containers":{"foreign":{}},"Containers":{}`, 1),
		"case-alias":           strings.Replace(raw, `"Internal":false`, `"Internal":true,"internal":false`, 1),
		"escaped-key":          strings.Replace(raw, `"EnableIPv6":false`, `"EnableIPv6":true,"EnableIPv\u0036":false`, 1),
		"duplicate-gateway":    strings.Replace(raw, `"Gateway":"172.22.0.1"`, `"Gateway":"8.8.8.8","Gateway":"172.22.0.1"`, 1),
		"extra-document":       raw + `[]`,
		"deep-metadata":        strings.Replace(raw, `"Driver":"bridge"`, `"Future":`+strings.Repeat(`[`, 18)+`0`+strings.Repeat(`]`, 18)+`,"Driver":"bridge"`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if changed == raw {
				t.Fatal("fixture failed to mutate response")
			}
			if _, err := c.inspectNetwork([]byte(changed), strings.Repeat("a", 64), ""); err == nil {
				t.Fatal("ambiguous inspection admitted")
			}
		})
	}
	n["FutureDockerMetadata"] = map[string]any{"Version": "new", "Enabled": false}
	if _, err := c.inspectNetwork(privateNetworkBytes(t, n), strings.Repeat("a", 64), ""); err != nil {
		t.Fatal("unambiguous extra metadata rejected", err)
	}
}

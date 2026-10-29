package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

func publicationFixture(t *testing.T) (map[string]any, map[string]any) {
	t.Helper()
	raw, e := os.ReadFile("testdata/publication.json")
	if e != nil {
		t.Fatal(e)
	}
	var rows []map[string]any
	if e = json.Unmarshal(raw, &rows); e != nil {
		t.Fatal(e)
	}
	return rows[0], rows[1]
}
func TestPublicationProjectsOnlyPublicEvidence(t *testing.T) {
	r, p := publicationFixture(t)
	secret := "PRIVATE-synthetic-cookie"
	r["unknown"] = secret
	p["cookie"] = secret
	out := auditPublication(r, p)
	raw, e := json.Marshal(out)
	if e != nil || bytes.Contains(raw, []byte(secret)) || out["ok"] != true || out["peer_writes"] != 2 {
		t.Fatal("invalid public projection")
	}
}
func TestPublicationRefusesExposureAndIncompleteDirectProof(t *testing.T) {
	for _, kind := range []string{"project", "network", "ipv6", "extra-member", "options", "public-subnet", "host", "wildcard", "rpc", "active-port", "second-network", "no-direct", "crossed", "extra-write", "no-stream", "no-refusal", "accepted-admin", "login", "wrong-endpoint", "failed-recovery"} {
		t.Run(kind, func(t *testing.T) {
			r, p := publicationFixture(t)
			pub := obj(r["publication"])
			ob := obj(pub["first_observation"])
			n, g := obj(ob["network"]), obj(ob["gateway"])
			c := obj(p["publication"])
			q := list(c["requests"])
			switch kind {
			case "project":
				r["project"] = strings.Repeat("d", 32)
			case "network":
				n["Id"] = strings.Repeat("d", 64)
			case "ipv6":
				n["EnableIPv6"] = true
			case "extra-member":
				obj(n["Containers"])[strings.Repeat("d", 64)] = map[string]any{}
			case "options":
				obj(n["Options"])["com.docker.network.bridge.gateway_mode_ipv4"] = "routed"
			case "public-subnet":
				obj(list(obj(n["IPAM"])["Config"])[0])["Subnet"] = "0.0.0.0/0"
			case "host":
				obj(g["HostConfig"])["NetworkMode"] = "host"
			case "wildcard":
				obj(list(obj(obj(g["HostConfig"])["PortBindings"])["18081/tcp"])[0])["HostIp"] = "0.0.0.0"
			case "rpc":
				obj(obj(g["HostConfig"])["PortBindings"])["25432/tcp"] = []any{map[string]any{"HostIp": "127.0.0.1", "HostPort": "25432"}}
			case "active-port":
				delete(obj(obj(g["NetworkSettings"])["Ports"]), "18081/tcp")
			case "second-network":
				obj(obj(g["NetworkSettings"])["Networks"])["other"] = map[string]any{}
			case "no-direct":
				c["direct_client"] = false
			case "crossed":
				obj(q[2])["status"] = float64(200)
			case "extra-write":
				c["requests"] = append(q, q[6])
			case "no-stream":
				obj(q[8])["stream"] = false
			case "no-refusal":
				c["refusals"] = list(c["refusals"])[:4]
			case "accepted-admin":
				obj(list(c["refusals"])[1])["refused"] = false
			case "login":
				obj(q[0])["path"] = "/~/login"
			case "wrong-endpoint":
				obj(q[0])["port"] = float64(18382)
			case "failed-recovery":
				p["ok"] = false
			}
			defer func() {
				if recover() == nil {
					t.Fatal("invalid publication passed")
				}
			}()
			auditPublication(r, p)
		})
	}
}
func TestPublicationAuditSelection(t *testing.T) {
	m := "omabit.private-publication-audit=required-v1"
	if publicationRequired("", true) || !publicationRequired(m, true) {
		t.Fatal("selection differs")
	}
	for _, row := range []struct {
		s string
		p bool
	}{{m, false}, {m + " " + m, true}, {"omabit.private-publication-audit=unknown", true}} {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatal("invalid selection passed")
				}
			}()
			publicationRequired(row.s, row.p)
		}()
	}
}

func TestPublicationCoverageFailureExposesBoundedCountsOnly(t *testing.T) {
	r, p := publicationFixture(t)
	c := obj(p["publication"])
	requests := list(c["requests"])
	var keep []any
	for _, value := range requests {
		q := obj(value)
		if q["method"] != "PUT" {
			keep = append(keep, value)
		}
	}
	c["requests"] = keep
	c["secret"] = "PRIVATE-synthetic-cookie"
	defer func() {
		failure := recover()
		if failure == nil {
			t.Fatal("missing coverage passed")
		}
		out := readerFailureProjection(failure)
		coverage, ok := out["publication_coverage"].(map[string]int)
		if !ok || coverage["channel_puts"] != 0 || coverage["writes"] != 2 {
			t.Fatal(out)
		}
		raw, e := json.Marshal(out)
		if e != nil || bytes.Contains(raw, []byte("PRIVATE-synthetic-cookie")) {
			t.Fatal("private diagnostic escaped")
		}
	}()
	auditPublication(r, p)
}
func TestPublicationCoverageProjectionRefusesUnboundedOrUntypedDetails(t *testing.T) {
	for _, value := range []any{publicationCoverageFailure{Streams: 201}, publicationCoverageFailure{Hosts: -1}, "private-application-value"} {
		out := readerFailureProjection(value)
		if out["publication_coverage"] != nil {
			t.Fatal("unbounded/untyped diagnostic accepted")
		}
		raw, _ := json.Marshal(out)
		if bytes.Contains(raw, []byte("private-application-value")) {
			t.Fatal("private diagnostic escaped")
		}
	}
}

func TestPublicationRequiresChannelAcknowledgementsAsWellAsLifecycle(t *testing.T) {
	for _, count := range []int{3, 6, 8} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			r, p := publicationFixture(t)
			c := obj(p["publication"])
			requests := list(c["requests"])
			var keep []any
			var put any
			for _, q := range requests {
				if obj(q)["method"] == "PUT" {
					put = q
				} else {
					keep = append(keep, q)
				}
			}
			for range count {
				keep = append(keep, put)
			}
			c["requests"] = keep
			defer func() {
				if recover() == nil {
					t.Fatal("missing or extra channel request accepted")
				}
			}()
			auditPublication(r, p)
		})
	}
}

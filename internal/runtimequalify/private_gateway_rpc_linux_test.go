//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

func privateRPCFixture(t *testing.T) (privatechain.Plan, privateGatewayRPCObservation) {
	t.Helper()
	p, _, _, _, _ := privateNetworkFixture(t)
	hash := "0x" + strings.Repeat("a", 64)
	o := privateGatewayRPCObservation{Format: "omabit-private-gateway-observation-v1", Head: 80, HeadHash: hash, Foundation: p.Foundation,
		Startup: privateGatewayStartupObservation{Format: "omabit-private-gateway-startup-v1", Ready: true, Head: "0x50", HeadHash: hash, Durable: true, Reopened: true}, Namespace: privateGatewayRPCNamespace{Device: "4", Inode: "123"}}
	for i, m := range p.Members {
		o.PublicKeys = append(o.PublicKeys, privatechain.GatewayPublicKey{Ship: m.Ship, Auth: "0x" + strings.Repeat(string(rune('b'+i*2)), 64), Crypt: "0x" + strings.Repeat(string(rune('c'+i*2)), 64), Life: 1, Suite: 1})
	}
	return p, o
}
func privateRPCBytes(t *testing.T, o privateGatewayRPCObservation) []byte {
	t.Helper()
	b, err := json.Marshal(o)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func TestPrivateGatewayRPCParse(t *testing.T) {
	p, expected := privateRPCFixture(t)
	got, err := parsePrivateGatewayRPC(p, privateRPCBytes(t, expected))
	if err != nil || !reflect.DeepEqual(got, expected) {
		t.Fatal(got, err)
	}
}
func TestPrivateGatewayRPCRejectsChangedEvidence(t *testing.T) {
	for name, change := range map[string]func(*privateGatewayRPCObservation){
		"format":             func(o *privateGatewayRPCObservation) { o.Format = "different" },
		"height":             func(o *privateGatewayRPCObservation) { o.Head++ },
		"head":               func(o *privateGatewayRPCObservation) { o.HeadHash = "0x" + strings.Repeat("0", 64) },
		"genesis":            func(o *privateGatewayRPCObservation) { o.Foundation.Genesis = "0x" + strings.Repeat("f", 64) },
		"chain":              func(o *privateGatewayRPCObservation) { o.Foundation.ChainID++ },
		"contract":           func(o *privateGatewayRPCObservation) { o.Foundation.Contracts[8].CodeSHA256 = strings.Repeat("f", 64) },
		"status-format":      func(o *privateGatewayRPCObservation) { o.Startup.Format = "different" },
		"ready":              func(o *privateGatewayRPCObservation) { o.Startup.Ready = false },
		"durable":            func(o *privateGatewayRPCObservation) { o.Startup.Durable = false },
		"reopened":           func(o *privateGatewayRPCObservation) { o.Startup.Reopened = false },
		"status-head":        func(o *privateGatewayRPCObservation) { o.Startup.Head = "0x51" },
		"status-hash":        func(o *privateGatewayRPCObservation) { o.Startup.HeadHash = "0x" + strings.Repeat("f", 64) },
		"missing-key":        func(o *privateGatewayRPCObservation) { o.PublicKeys = o.PublicKeys[:1] },
		"wrong-ship":         func(o *privateGatewayRPCObservation) { o.PublicKeys[0].Ship = "~bud" },
		"life":               func(o *privateGatewayRPCObservation) { o.PublicKeys[0].Life++ },
		"suite":              func(o *privateGatewayRPCObservation) { o.PublicKeys[0].Suite++ },
		"duplicate-key":      func(o *privateGatewayRPCObservation) { o.PublicKeys[1].Auth = o.PublicKeys[0].Crypt },
		"zero-key":           func(o *privateGatewayRPCObservation) { o.PublicKeys[0].Auth = "0x" + strings.Repeat("0", 64) },
		"upper-key":          func(o *privateGatewayRPCObservation) { o.PublicKeys[0].Auth = "0x" + strings.Repeat("F", 64) },
		"zero-namespace":     func(o *privateGatewayRPCObservation) { o.Namespace.Inode = "0" },
		"aliased-namespace":  func(o *privateGatewayRPCObservation) { o.Namespace.Inode = "0123" },
		"overflow-namespace": func(o *privateGatewayRPCObservation) { o.Namespace.Inode = "18446744073709551616" },
	} {
		t.Run(name, func(t *testing.T) {
			p, o := privateRPCFixture(t)
			o.Foundation.Contracts = append([]privatechain.Contract(nil), o.Foundation.Contracts...)
			change(&o)
			got, err := parsePrivateGatewayRPC(p, privateRPCBytes(t, o))
			if !errors.Is(err, errPrivateGatewayRPC) || !reflect.DeepEqual(got, privateGatewayRPCObservation{}) {
				t.Fatal("accepted invalid evidence", got, err)
			}
		})
	}
}
func TestPrivateGatewayRPCRejectsMalformedOutput(t *testing.T) {
	p, o := privateRPCFixture(t)
	raw := string(privateRPCBytes(t, o))
	for name, value := range map[string]string{"empty": "", "null": "null", "trailing": raw + "{}", "duplicate": strings.Replace(raw, "{", "{\"head\":0,", 1), "case-alias": strings.Replace(raw, "{", "{\"HEAD\":0,", 1), "unknown": strings.Replace(raw, "{", "{\"secret\":\"never export\",", 1), "oversize": raw + strings.Repeat(" ", 16384), "nested-unknown": strings.Replace(raw, "\"durable\":true", "\"durable\":true,\"private\":true", 1)} {
		t.Run(name, func(t *testing.T) {
			if _, err := parsePrivateGatewayRPC(p, []byte(value)); !errors.Is(err, errPrivateGatewayRPC) {
				t.Fatal("accepted malformed output", err)
			}
		})
	}
}
func TestPrivateGatewayRPCFixedCommand(t *testing.T) {
	p, _, _, _, _ := privateNetworkFixture(t)
	id := strings.Repeat("b", 64)
	args, err := privateGatewayRPCArgs(p, id)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"container", "exec", id, "/node/usr/lib/ld-linux-x86-64.so.2", "--library-path", "/node/usr/lib", "/node/usr/bin/node", "--eval"}
	if len(args) != 10 || !reflect.DeepEqual(args[:8], want) || !strings.HasPrefix(args[8], privateGatewayProbe+"\nmain().catch(") {
		t.Fatal("unexpected execution contract")
	}
	var input struct {
		Foundation privatechain.Foundation `json:"foundation"`
		Members    []struct {
			Ship   string `json:"ship"`
			Number int    `json:"number"`
		} `json:"members"`
	}
	if err := json.Unmarshal([]byte(args[9]), &input); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(input.Foundation, p.Foundation) || len(input.Members) != 2 {
		t.Fatal("changed public input")
	}
	for i, m := range input.Members {
		if m.Ship != p.Members[i].Ship || (m.Ship == "~nec" && m.Number != 1) || (m.Ship == "~zod" && m.Number != 0) {
			t.Fatal("changed member order or number")
		}
	}
	for _, bad := range []string{"", "gateway-name", strings.Repeat("0", 64), strings.Repeat("A", 64)} {
		if _, err := privateGatewayRPCArgs(p, bad); err == nil {
			t.Fatal("accepted nonexact container")
		}
	}
	p.PublishRPC = true
	if _, err := privateGatewayRPCArgs(p, id); err == nil {
		t.Fatal("accepted invalid plan")
	}
}
func TestPrivateGatewayRPCNoAuthorityFromNilOrCancelledCaller(t *testing.T) {
	p, _ := privateRPCFixture(t)
	var q *qualifier
	if _, err := q.privateGatewayRPC(context.Background(), p, strings.Repeat("b", 64)); !errors.Is(err, errPrivateGatewayRPC) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := q.privateGatewayRPC(ctx, p, strings.Repeat("b", 64)); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestPrivateGatewayRPCOwnedExecutorAndFailures(t *testing.T) {
	for _, mode := range []string{"valid", "client-error", "malformed", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			x := newPrivateResourceFixture(t)
			p, o := privateRPCFixture(t)
			id := strings.Repeat("b", 64)
			want, err := privateGatewayRPCArgs(p, id)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			x.e.q.d.execute = func(ctx context.Context, args, env []string) (string, error) {
				calls++
				deadline, ok := ctx.Deadline()
				if !ok || time.Until(deadline) > 15*time.Second || len(args) != len(want)+4 || args[0] != "--host" || args[1] != "unix://"+x.e.q.socket || args[2] != "--config" || args[3] != x.e.q.config || !reflect.DeepEqual(args[4:], want) {
					t.Fatal("unbounded, ambient or changed probe command")
				}
				if len(env) != 4 || env[0] != "HOME="+x.e.q.d.home {
					t.Fatal("ambient environment")
				}
				if mode == "client-error" {
					return "private raw error must not escape", errors.New("private raw diagnostic")
				}
				if mode == "malformed" {
					return "private invalid JSON", nil
				}
				if mode == "cancelled" {
					cancel()
				}
				return string(privateRPCBytes(t, o)), nil
			}
			got, err := x.e.q.privateGatewayRPC(ctx, p, id)
			if calls != 1 {
				t.Fatal("probe was skipped or retried", calls)
			}
			if mode == "valid" {
				if err != nil || !reflect.DeepEqual(got, o) {
					t.Fatal("valid probe failed", err)
				}
			} else if !errors.Is(err, errPrivateGatewayRPC) || !reflect.DeepEqual(got, privateGatewayRPCObservation{}) || strings.Contains(err.Error(), "raw") {
				t.Fatal("invalid result or diagnostic escaped", err)
			}
		})
	}
}

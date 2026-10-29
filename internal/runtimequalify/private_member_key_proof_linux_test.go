//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

func privateKeyProofFixture(t *testing.T) (privateMemberContract, privateMemberKeyProofObservation) {
	t.Helper()
	c := privateMemberContractFixture(t, "sender", "boot")
	g := c.selection.Gateway.Observation
	o := privateGatewayRPCObservation{Format: "omabit-private-gateway-observation-v1", Head: g.Head, HeadHash: g.HeadHash, Foundation: g.Foundation, PublicKeys: g.PublicKeys, Namespace: privateGatewayRPCNamespace{Device: strconv.FormatUint(g.Namespace.Device, 10), Inode: strconv.FormatUint(g.Namespace.Inode, 10)}, Startup: privateGatewayStartupObservation{Format: "omabit-private-gateway-startup-v1", Ready: true, Head: "0x50", HeadHash: g.HeadHash, Durable: true, Reopened: true}}
	ship := ""
	for _, m := range c.selection.Plan.Members {
		if m.InstanceID == c.selection.InstanceID {
			ship = m.Ship
		}
	}
	return c, privateMemberKeyProofObservation{Format: "omabit-private-member-key-proof-v1", BindingSHA256: c.digest, Ship: ship, KeySHA256: strings.Repeat("a", 64), Verified: true, Gateway: o}
}
func keyProofBytes(t *testing.T, o privateMemberKeyProofObservation) []byte {
	t.Helper()
	raw, err := json.Marshal(o)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func TestPrivateMemberKeyProofExactBinding(t *testing.T) {
	c, want := privateKeyProofFixture(t)
	got, err := parsePrivateMemberKeyProof(c, keyProofBytes(t, want))
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatal("bound proof refused", err)
	}
	args, err := privateMemberKeyProofArgs(c)
	if err != nil {
		t.Fatal(err)
	}
	prefix := []string{"container", "exec", c.selection.Gateway.ContainerID, "/node/usr/lib/ld-linux-x86-64.so.2", "--library-path", "/node/usr/lib", "/node/usr/bin/node", "--eval"}
	if len(args) != 10 || !reflect.DeepEqual(args[:8], prefix) || !strings.Contains(args[8], privateGatewayProbe) || !strings.Contains(args[8], privateMemberKeyProbe) {
		t.Fatal("execution contract changed")
	}
	var input struct {
		Gateway struct {
			Members []struct {
				Ship string `json:"ship"`
			}
		} `json:"gateway"`
		Ship          string `json:"ship"`
		BindingSHA256 string `json:"binding_sha256"`
	}
	if json.Unmarshal([]byte(args[9]), &input) != nil || input.Ship != want.Ship || input.BindingSHA256 != c.digest || len(input.Gateway.Members) != 2 {
		t.Fatal("selection lost")
	}
	if strings.Contains(args[9], c.keyPath) || strings.Contains(args[9], c.pillPath) {
		t.Fatal("host input path exported")
	}
}
func TestPrivateMemberKeyProofRejectsChangedBinding(t *testing.T) {
	for _, kind := range []string{"resume", "contract", "key-path", "instance", "format", "ship", "binding", "hash", "verified", "head", "namespace", "public-key", "life", "foundation", "unready"} {
		t.Run(kind, func(t *testing.T) {
			c, o := privateKeyProofFixture(t)
			o.Gateway.PublicKeys = append(o.Gateway.PublicKeys[:0:0], o.Gateway.PublicKeys...)
			o.Gateway.Foundation.Contracts = append(o.Gateway.Foundation.Contracts[:0:0], o.Gateway.Foundation.Contracts...)
			switch kind {
			case "resume":
				c = privateMemberContractFixture(t, "sender", "resume")
			case "contract":
				c.digest = strings.Repeat("b", 64)
			case "key-path":
				c.keyPath += "-changed"
			case "instance":
				c.selection.InstanceID = "receiver"
			case "format":
				o.Format = "other"
			case "ship":
				o.Ship = "~bud"
			case "binding":
				o.BindingSHA256 = strings.Repeat("b", 64)
			case "hash":
				o.KeySHA256 = strings.Repeat("0", 64)
			case "verified":
				o.Verified = false
			case "head":
				o.Gateway.HeadHash = "0x" + strings.Repeat("f", 64)
				o.Gateway.Startup.HeadHash = o.Gateway.HeadHash
			case "namespace":
				o.Gateway.Namespace.Inode = "999"
			case "public-key":
				o.Gateway.PublicKeys[0].Auth = "0x" + strings.Repeat("f", 64)
			case "life":
				o.Gateway.PublicKeys[0].Life++
			case "foundation":
				o.Gateway.Foundation.Contracts[0].CodeSHA256 = strings.Repeat("f", 64)
			case "unready":
				o.Gateway.Startup.Ready = false
			}
			got, err := parsePrivateMemberKeyProof(c, keyProofBytes(t, o))
			if !errors.Is(err, errPrivateMemberKeyProof) || !reflect.DeepEqual(got, privateMemberKeyProofObservation{}) {
				t.Fatal("changed binding accepted", err)
			}
		})
	}
}
func TestPrivateMemberKeyProofMalformedOutput(t *testing.T) {
	c, o := privateKeyProofFixture(t)
	raw := string(keyProofBytes(t, o))
	for _, value := range []string{"", "null", raw + "{}", raw + strings.Repeat(" ", 24576), strings.Replace(raw, "{", "{\"ship\":\"~nec\",", 1), strings.Replace(raw, "{", "{\"SHIP\":\"~nec\",", 1), strings.Replace(raw, "{", "{\"key\":\"never accept\",", 1), strings.Replace(raw, "\"durable\":true", "\"durable\":true,\"secret\":true", 1)} {
		if _, err := parsePrivateMemberKeyProof(c, []byte(value)); !errors.Is(err, errPrivateMemberKeyProof) {
			t.Fatal("invalid JSON accepted")
		}
	}
}
func TestPrivateMemberKeyProofNoAmbientAuthority(t *testing.T) {
	c, _ := privateKeyProofFixture(t)
	var q *qualifier
	if _, err := q.privateMemberKeyProof(context.Background(), c); !errors.Is(err, errPrivateMemberKeyProof) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := q.privateMemberKeyProof(ctx, c); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
func TestPrivateMemberKeyProofEmbeddedHostRefusal(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node is required for embedded program guard qualification")
	}
	c, _ := privateKeyProofFixture(t)
	args, err := privateMemberKeyProofArgs(c)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "--eval", args[8], args[9])
	cmd.Env = []string{"HOME=" + t.TempDir(), "LANG=C.UTF-8"}
	out, err := cmd.CombinedOutput()
	if err == nil || ctx.Err() != nil || string(out) != "private_member_key_proof_refused\n" {
		t.Fatal("embedded program failed host-refusal contract", err, string(out))
	}
}
func TestPrivateMemberKeyProofOwnedExecutorSingleAttempt(t *testing.T) {
	for _, mode := range []string{"valid", "error", "malformed", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			x := newPrivateResourceFixture(t)
			c, o := privateKeyProofFixture(t)
			want, err := privateMemberKeyProofArgs(c)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			x.e.q.d.execute = func(ctx context.Context, args, env []string) (string, error) {
				calls++
				deadline, ok := ctx.Deadline()
				if !ok || time.Until(deadline) > 90*time.Second || len(args) != len(want)+4 || args[0] != "--host" || args[1] != "unix://"+x.e.q.socket || args[2] != "--config" || args[3] != x.e.q.config || !reflect.DeepEqual(args[4:], want) {
					t.Fatal("ambient, unbounded or rebound execution")
				}
				if len(env) != 4 || env[0] != "HOME="+x.e.q.d.home {
					t.Fatal("ambient environment")
				}
				if mode == "error" {
					return "secret diagnostic", errors.New("private error")
				}
				if mode == "malformed" {
					return "secret malformed JSON", nil
				}
				if mode == "cancelled" {
					cancel()
				}
				return string(keyProofBytes(t, o)), nil
			}
			got, err := x.e.q.privateMemberKeyProof(ctx, c)
			if calls != 1 {
				t.Fatal("observation skipped or retried", calls)
			}
			if mode == "valid" {
				if err != nil || !reflect.DeepEqual(got, o) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, errPrivateMemberKeyProof) || !reflect.DeepEqual(got, privateMemberKeyProofObservation{}) || strings.Contains(err.Error(), "secret") {
				t.Fatal("invalid result escaped", err)
			}
		})
	}
}

//go:build linux && amd64 && omabit_private_member_key_inspection

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

func keyAuditSourceFixture(t *testing.T) (privatechain.Plan, privatechain.Workspace, privatechain.Gateway) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "internal", "managed", "testdata", "historical-upgrades", "schema43.json"))
	if err != nil {
		t.Fatal(err)
	}
	if digest(raw) != "004391518e998771b9e739c094d7b6f80b0746dae7624ddfe697f7e9307bdbdc" {
		t.Fatal("original synthetic43 fixture changed")
	}
	var saved struct {
		Synthetic bool `json:"synthetic"`
		Schema    int  `json:"schema"`
		Tables    []struct {
			Name    string              `json:"name"`
			Columns []string            `json:"columns"`
			Rows    [][]json.RawMessage `json:"rows"`
		} `json:"tables"`
	}
	if json.Unmarshal(raw, &saved) != nil || !saved.Synthetic || saved.Schema != 43 {
		t.Fatal("synthetic43 fixture required")
	}
	bodies := map[string][][]byte{}
	for _, table := range saved.Tables {
		if table.Name != "private_projects" && table.Name != "private_project_workspaces" && table.Name != "private_gateway_observations" {
			continue
		}
		column := -1
		for i, name := range table.Columns {
			if name == "body" {
				column = i
			}
		}
		if column < 0 {
			t.Fatal("missing body")
		}
		for _, row := range table.Rows {
			var value struct {
				Blob string `json:"blob"`
			}
			if json.Unmarshal(row[column], &value) != nil {
				t.Fatal("blob decode")
			}
			decoded, err := base64.StdEncoding.DecodeString(value.Blob)
			if err != nil {
				t.Fatal(err)
			}
			bodies[table.Name] = append(bodies[table.Name], decoded)
		}
	}
	var g privatechain.Gateway
	// The recovered result belongs to the complete workspace. The unrelated
	// ready row in the historical fixture deliberately has an interrupted layout.
	for _, raw := range bodies["private_gateway_observations"] {
		var candidate struct {
			Phase  string                `json:"phase"`
			Result *privatechain.Gateway `json:"result"`
		}
		if json.Unmarshal(raw, &candidate) != nil {
			t.Fatal("gateway decode")
		}
		if candidate.Phase == "observed" && candidate.Result != nil {
			g = *candidate.Result
			break
		}
	}
	var w privatechain.Workspace
	var p privatechain.Plan
	for _, raw := range bodies["private_project_workspaces"] {
		var candidate privatechain.Workspace
		if json.Unmarshal(raw, &candidate) != nil {
			t.Fatal("workspace decode")
		}
		if candidate.Binding == g.Selection.Binding {
			w = candidate
			break
		}
	}
	for _, raw := range bodies["private_projects"] {
		var candidate struct {
			Binding privatechain.Binding `json:"binding"`
			Plan    privatechain.Plan    `json:"plan"`
		}
		if json.Unmarshal(raw, &candidate) != nil {
			t.Fatal("project decode")
		}
		if candidate.Binding == g.Selection.Binding {
			p = candidate.Plan
			break
		}
	}
	if g.Phase != "ready" || g.Validate(p, w) != nil {
		t.Fatal("ready synthetic43 fixture unavailable")
	}
	return p, w, g
}
func keyAuditFixture(t *testing.T) (keyAuditRecord, []byte, privatechain.Plan, privatechain.Workspace, privatechain.Gateway, map[string][]byte) {
	t.Helper()
	p, w, g := keyAuditSourceFixture(t)
	o := g.Observation.Clone()
	o.ObservedAt = time.Time{}
	record := keyAuditRecord{Version: 1, PlanDigest: w.PlanDigest, GatewaySHA256: hash(g), WorkspaceSHA256: hash(w)}
	keys := map[string][]byte{}
	for i, m := range p.Members {
		key := []byte("0w" + strings.Repeat(string(rune('a'+i)), 24) + "\n")
		keys[m.Ship] = key
		for j, control := range []string{"valid", "wrong-public-key", "wrong-life"} {
			proof := keyAuditProof{Format: "omabit-private-member-key-proof-v1", BindingSHA256: fmt.Sprintf("%064x", i*3+j+1), Ship: m.Ship, KeySHA256: digest(key), Verified: true}
			proof.Gateway = keyAuditRPC{Format: "omabit-private-gateway-observation-v1", Head: o.Head, HeadHash: o.HeadHash, Foundation: o.Foundation, PublicKeys: o.PublicKeys}
			proof.Gateway.Startup.Format = "omabit-private-gateway-startup-v1"
			proof.Gateway.Startup.Ready = true
			proof.Gateway.Startup.Head = "0x50"
			proof.Gateway.Startup.HeadHash = o.HeadHash
			proof.Gateway.Startup.Durable = true
			proof.Gateway.Startup.Reopened = true
			proof.Gateway.Namespace.Device = fmt.Sprint(o.Namespace.Device)
			proof.Gateway.Namespace.Inode = fmt.Sprint(o.Namespace.Inode)
			raw, err := json.Marshal(proof)
			if err != nil {
				t.Fatal(err)
			}
			record.Cases = append(record.Cases, keyAuditCase{InstanceID: m.InstanceID, Ship: m.Ship, Control: control, Proof: raw, BeforeSHA256: hash(o), AfterSHA256: hash(o), HostKeyVerified: true, Seconds: 1.5})
		}
	}
	log := []byte("--- PASS: TestPrivateMemberKeyProofVM\ntwo valid keys, four explicit negative verdicts\n")
	return record, log, p, w, g, keys
}
func keyAuditBytes(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func TestMemberKeyAuditRequiresTaggedMarker(t *testing.T) {
	if inspectionGuestMarker != "offline-managed-member-key-inspect-v1" || memberKeyAudit == nil {
		t.Fatal("key audit not mandatory in tagged reader")
	}
}
func TestMemberKeyAuditIndependentBytesAndPublicProjection(t *testing.T) {
	r, log, p, w, g, keys := keyAuditFixture(t)
	reads := 0
	out := checkMemberKeyProofs(keyAuditBytes(t, r), log, p, w, g, func(ship string) []byte {
		reads++
		key, ok := keys[ship]
		if !ok {
			t.Fatal("unselected key")
		}
		return append([]byte{}, key...)
	})
	if out["ok"] != true || out["explicit_negative_verdicts"] != 4 || out["positive_verdicts"] != 2 || out["key_contents_read"] != true || out["key_contents_exported"] != false || reads != 2 {
		t.Fatal("projection changed")
	}
	raw := string(keyAuditBytes(t, out))
	for _, key := range keys {
		if strings.Contains(raw, string(key)) || strings.Contains(raw, digest(key)) {
			t.Fatal("key bytes or hash exported")
		}
	}
}
func TestMemberKeyAuditRejectsChangedProofOrRetainedKey(t *testing.T) {
	for _, kind := range []string{"missing", "extra", "member", "control", "binding-reused", "key-hash", "namespace", "head", "foundation", "public-key", "startup", "verified", "host-key", "before", "after", "duration", "total-duration", "boots", "source", "log", "log-secret", "changed-key", "invalid-key", "duplicate-json", "unknown-json", "noncanonical"} {
		t.Run(kind, func(t *testing.T) {
			r, log, p, w, g, keys := keyAuditFixture(t)
			var proof keyAuditProof
			if json.Unmarshal(r.Cases[0].Proof, &proof) != nil {
				t.Fatal("fixture proof")
			}
			switch kind {
			case "missing":
				r.Cases = r.Cases[:5]
			case "extra":
				r.Cases = append(r.Cases, r.Cases[0])
			case "member":
				r.Cases[0].InstanceID = "other"
			case "control":
				r.Cases[0].Control = "wrong-life"
			case "binding-reused":
				proof.BindingSHA256 = fmt.Sprintf("%064x", 2)
			case "key-hash":
				proof.KeySHA256 = strings.Repeat("f", 64)
			case "namespace":
				proof.Gateway.Namespace.Inode = "999"
			case "head":
				proof.Gateway.HeadHash = "0x" + strings.Repeat("1", 64)
			case "foundation":
				proof.Gateway.Foundation.Contracts[0].CodeSHA256 = strings.Repeat("f", 64)
			case "public-key":
				proof.Gateway.PublicKeys[0].Auth = "0x" + strings.Repeat("f", 64)
			case "startup":
				proof.Gateway.Startup.Ready = false
			case "verified":
				proof.Verified = false
			case "host-key":
				r.Cases[0].HostKeyVerified = false
			case "before":
				r.Cases[0].BeforeSHA256 = strings.Repeat("f", 64)
			case "after":
				r.Cases[0].AfterSHA256 = strings.Repeat("f", 64)
			case "duration":
				r.Cases[0].Seconds = 0
			case "total-duration":
				for i := range r.Cases {
					r.Cases[i].Seconds = 121
				}
			case "boots":
				r.ShipBoots = 1
			case "source":
				r.GatewaySHA256 = strings.Repeat("f", 64)
			case "log":
				log = []byte("failed")
			case "log-secret":
				log = append(log, keys[p.Members[0].Ship]...)
			case "changed-key":
				keys[p.Members[0].Ship] = []byte("0w" + strings.Repeat("f", 24) + "\n")
			case "invalid-key":
				keys[p.Members[0].Ship] = []byte("invalid")
			}
			r.Cases[0].Proof = keyAuditBytes(t, proof)
			raw := keyAuditBytes(t, r)
			if kind == "duplicate-json" {
				raw = append([]byte(`{"version":1,`), raw[1:]...)
			}
			if kind == "unknown-json" {
				raw = append([]byte(`{"private":true,`), raw[1:]...)
			}
			if kind == "noncanonical" {
				raw = append(raw, '\n')
			}
			defer func() {
				if recover() == nil {
					t.Fatal("changed proof accepted")
				}
			}()
			checkMemberKeyProofs(raw, log, p, w, g, func(ship string) []byte { return append([]byte{}, keys[ship]...) })
		})
	}
}

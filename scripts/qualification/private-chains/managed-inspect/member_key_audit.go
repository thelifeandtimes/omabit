//go:build linux && amd64 && omabit_private_member_key_inspection

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func init() {
	inspectionGuestMarker = "offline-managed-member-key-inspect-v1"
	memberKeyAudit = inspectMemberKeyProofs
}

type keyAuditRPC struct {
	Format     string                          `json:"format"`
	Head       uint64                          `json:"head"`
	HeadHash   string                          `json:"head_hash"`
	Foundation privatechain.Foundation         `json:"foundation"`
	PublicKeys []privatechain.GatewayPublicKey `json:"public_keys"`
	Startup    struct {
		Format   string `json:"format"`
		Ready    bool   `json:"ready"`
		Head     string `json:"head"`
		HeadHash string `json:"head_hash"`
		Durable  bool   `json:"durable"`
		Reopened bool   `json:"reopened"`
	} `json:"startup"`
	Namespace struct {
		Device string `json:"device"`
		Inode  string `json:"inode"`
	} `json:"namespace"`
}
type keyAuditProof struct {
	Format        string      `json:"format"`
	BindingSHA256 string      `json:"binding_sha256"`
	Ship          string      `json:"ship"`
	KeySHA256     string      `json:"key_sha256"`
	Verified      bool        `json:"verified"`
	Gateway       keyAuditRPC `json:"gateway"`
}
type keyAuditCase struct {
	InstanceID      string          `json:"instance_id"`
	Ship            string          `json:"ship"`
	Control         string          `json:"control"`
	Proof           json.RawMessage `json:"proof"`
	BeforeSHA256    string          `json:"before_sha256"`
	AfterSHA256     string          `json:"after_sha256"`
	HostKeyVerified bool            `json:"host_key_verified"`
	Seconds         float64         `json:"seconds"`
}
type keyAuditRecord struct {
	Version         int            `json:"version"`
	PlanDigest      string         `json:"plan_digest"`
	GatewaySHA256   string         `json:"gateway_sha256"`
	WorkspaceSHA256 string         `json:"workspace_sha256"`
	Cases           []keyAuditCase `json:"cases"`
	ShipBoots       int            `json:"ship_boots"`
}

func keyAuditDecode(raw []byte, value any) {
	decode(raw, value)
	canonical, err := json.Marshal(value)
	must(err)
	need(bytes.Equal(canonical, raw), "key-proof-canonical-json")
}
func keyAuditHash(value string) bool {
	return regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(value) && value != strings.Repeat("0", 64)
}
func checkMemberKeyProofs(raw, log []byte, p privatechain.Plan, w privatechain.Workspace, g privatechain.Gateway, readKey func(string) []byte) map[string]any {
	must(g.Validate(p, w))
	need(g.Phase == "ready" && len(p.Members) == 2, "key-proof-ready-source")
	var record keyAuditRecord
	keyAuditDecode(raw, &record)
	need(record.Version == 1 && record.PlanDigest == w.PlanDigest && record.GatewaySHA256 == hash(g) && record.WorkspaceSHA256 == hash(w) && record.ShipBoots == 0 && len(record.Cases) == 6, "key-proof-inventory")
	need(bytes.Contains(log, []byte("--- PASS: TestPrivateMemberKeyProofVM")) && bytes.Contains(log, []byte("two valid keys, four explicit negative verdicts")), "key-proof-runtime-passed")
	baseline := g.Observation.Clone()
	baseline.ObservedAt = time.Time{}
	observationHash := hash(baseline)
	seenBindings := map[string]bool{}
	seenKeys := map[string]bool{}
	total := float64(0)
	for i, m := range p.Members {
		key := readKey(m.Ship)
		defer clear(key)
		keyText := strings.TrimSuffix(string(key), "\n")
		need(len(key) <= protocol.MaxBootKeyBytes+1 && protocol.ValidBootKey(keyText), "key-proof-retained-key-format")
		keyHash := digest(key)
		need(!seenKeys[keyHash], "key-proof-distinct-members")
		seenKeys[keyHash] = true
		need(!bytes.Contains(log, []byte(keyText)), "key-proof-no-key-text-in-log")
		for j, control := range []string{"valid", "wrong-public-key", "wrong-life"} {
			row := record.Cases[i*3+j]
			need(row.InstanceID == m.InstanceID && row.Ship == m.Ship && row.Control == control && row.HostKeyVerified && row.BeforeSHA256 == observationHash && row.AfterSHA256 == observationHash && row.Seconds > 0 && row.Seconds <= 720, "key-proof-case-binding")
			total += row.Seconds
			var proof keyAuditProof
			keyAuditDecode(row.Proof, &proof)
			need(proof.Format == "omabit-private-member-key-proof-v1" && proof.Ship == m.Ship && proof.KeySHA256 == keyHash && proof.Verified && keyAuditHash(proof.BindingSHA256) && !seenBindings[proof.BindingSHA256], "key-proof-exact-file-hash")
			seenBindings[proof.BindingSHA256] = true
			live := proof.Gateway
			need(live.Format == "omabit-private-gateway-observation-v1" && live.Head == baseline.Head && live.HeadHash == baseline.HeadHash && reflect.DeepEqual(live.Foundation, baseline.Foundation) && reflect.DeepEqual(live.PublicKeys, baseline.PublicKeys), "key-proof-public-chain")
			need(live.Startup.Format == "omabit-private-gateway-startup-v1" && live.Startup.Ready && live.Startup.Head == "0x50" && live.Startup.HeadHash == baseline.HeadHash && live.Startup.Durable && live.Startup.Reopened, "key-proof-durable-startup")
			need(live.Namespace.Device == strconv.FormatUint(baseline.Namespace.Device, 10) && live.Namespace.Inode == strconv.FormatUint(baseline.Namespace.Inode, 10), "key-proof-runtime-namespace")
		}
	}
	need(total <= 720, "key-proof-total-deadline")
	return map[string]any{"schema": 1, "ok": true, "cases": 6, "members": 2, "positive_verdicts": 2, "explicit_negative_verdicts": 4, "retained_key_hashes_match": true, "key_text_absent_from_log": true, "gateway_state_unchanged": true, "host_key_verifier_passed": true, "receipt_sha256": digest(raw), "log_sha256": digest(log), "key_contents_read": true, "key_contents_exported": false, "raw_state_exported": false, "ship_boots": 0, "member_launch_qualified": false}
}
func inspectMemberKeyProofs(p privatechain.Plan, w privatechain.Workspace, g privatechain.Gateway) {
	raw := file(root+"/qualification-private-member-key.json", 256<<10, 1000, 0600)
	log := file("/newroot/opt/omabit-lab/results/managed-member-key.log", 8<<20, 0, 0)
	summary := checkMemberKeyProofs(raw, log, p, w, g, func(ship string) []byte {
		need(ship == "~zod" || ship == "~nec", "key-proof-selected-ship")
		return file(root+"/"+w.Path+"/secrets/"+strings.TrimPrefix(ship, "~")+".key", protocol.MaxBootKeyBytes+1, 1000, 0600)
	})
	encoded, err := json.Marshal(summary)
	must(err)
	fmt.Println("OMABIT_MEMBER_KEY_INSPECTION " + string(encoded))
}

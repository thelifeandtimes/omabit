//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"time"
)

//go:embed private_member_key_probe.cjs
var privateMemberKeyProbe string

var errPrivateMemberKeyProof = errors.New("private member bootstrap key proof is incomplete or changed")

type privateMemberKeyProofObservation struct {
	Format        string                       `json:"format"`
	BindingSHA256 string                       `json:"binding_sha256"`
	Ship          string                       `json:"ship"`
	KeySHA256     string                       `json:"key_sha256"`
	Verified      bool                         `json:"verified"`
	Gateway       privateGatewayRPCObservation `json:"gateway"`
}

func privateMemberKeyProofArgs(c privateMemberContract) ([]string, error) {
	canonical, err := newPrivateMemberContract(c.selection, c.root, c.imageID)
	if err != nil || c.selection.Mode != "boot" || !reflect.DeepEqual(c, canonical) {
		return nil, errPrivateMemberKeyProof
	}
	gatewayArgs, err := privateGatewayRPCArgs(c.selection.Plan, c.selection.Gateway.ContainerID)
	if err != nil {
		return nil, errPrivateMemberKeyProof
	}
	ship := ""
	for _, m := range c.selection.Plan.Members {
		if m.InstanceID == c.selection.InstanceID {
			ship = m.Ship
		}
	}
	input := struct {
		Gateway       json.RawMessage `json:"gateway"`
		Ship          string          `json:"ship"`
		BindingSHA256 string          `json:"binding_sha256"`
	}{json.RawMessage(gatewayArgs[len(gatewayArgs)-1]), ship, c.digest}
	raw, err := json.Marshal(input)
	if err != nil || len(raw) > 12288 {
		return nil, errPrivateMemberKeyProof
	}
	// Public selection only in argv. The key stays inside its exact owned gateway;
	// Vere receives the expression through stdin, never a shell or process argv.
	program := "const gatewayProofHelpers=(()=>{const module={exports:{}};\n" + privateGatewayProbe + "\nreturn module.exports;})();\n" + privateMemberKeyProbe + "\nmain(gatewayProofHelpers).catch(()=>{process.stderr.write('private_member_key_proof_refused\\n');process.exitCode=1;});\n"
	return append(append([]string{}, gatewayArgs[:8]...), program, string(raw)), nil
}

func parsePrivateMemberKeyProof(c privateMemberContract, raw []byte) (privateMemberKeyProofObservation, error) {
	bad := func() (privateMemberKeyProofObservation, error) {
		return privateMemberKeyProofObservation{}, errPrivateMemberKeyProof
	}
	if _, err := privateMemberKeyProofArgs(c); err != nil || len(raw) > 24576 || !privateGatewayUnambiguousJSON(raw) {
		return bad()
	}
	var o privateMemberKeyProofObservation
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&o) != nil || o.Format != "omabit-private-member-key-proof-v1" || o.BindingSHA256 != c.digest || !privateGatewayID(o.KeySHA256) || !o.Verified {
		return bad()
	}
	ship := ""
	for _, m := range c.selection.Plan.Members {
		if m.InstanceID == c.selection.InstanceID {
			ship = m.Ship
		}
	}
	if o.Ship != ship {
		return bad()
	}
	gatewayBytes, err := json.Marshal(o.Gateway)
	if err != nil {
		return bad()
	}
	live, err := parsePrivateGatewayRPC(c.selection.Plan, gatewayBytes)
	if err != nil {
		return bad()
	}
	expected := c.selection.Gateway.Observation
	if live.Head != expected.Head || live.HeadHash != expected.HeadHash || !reflect.DeepEqual(live.PublicKeys, expected.PublicKeys) || live.Namespace.Device != strconv.FormatUint(expected.Namespace.Device, 10) || live.Namespace.Inode != strconv.FormatUint(expected.Namespace.Inode, 10) {
		return bad()
	}
	return o, nil
}

// This is an observation primitive, not admission or dispatch. The eventual
// driver must bracket it with fresh complete gateway/engine/membership checks,
// open the host MemberKeyVerifier against this hash, and retain accepted intent
// before any member effect. A saved gateway receipt alone cannot authorize it.
// Resume must not call this bootstrap-only path. No automatic retry is allowed.
func (q *qualifier) privateMemberKeyProof(ctx context.Context, c privateMemberContract) (privateMemberKeyProofObservation, error) {
	if err := ctx.Err(); err != nil {
		return privateMemberKeyProofObservation{}, err
	}
	args, err := privateMemberKeyProofArgs(c)
	if q == nil || err != nil {
		return privateMemberKeyProofObservation{}, errPrivateMemberKeyProof
	}
	out, err := q.docker(ctx, 90*time.Second, args...)
	if err != nil {
		return privateMemberKeyProofObservation{}, errPrivateMemberKeyProof
	}
	return parsePrivateMemberKeyProof(c, []byte(out))
}

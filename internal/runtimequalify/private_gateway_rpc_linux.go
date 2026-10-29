//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

//go:embed private_gateway_probe.cjs
var privateGatewayProbe string

var errPrivateGatewayRPC = errors.New("private gateway live chain observation is incomplete or changed")

type privateGatewayRPCObservation struct {
	Format     string                           `json:"format"`
	Head       uint64                           `json:"head"`
	HeadHash   string                           `json:"head_hash"`
	Foundation privatechain.Foundation          `json:"foundation"`
	PublicKeys []privatechain.GatewayPublicKey  `json:"public_keys"`
	Startup    privateGatewayStartupObservation `json:"startup"`
	Namespace  privateGatewayRPCNamespace       `json:"namespace"`
}
type privateGatewayStartupObservation struct {
	Format   string `json:"format"`
	Ready    bool   `json:"ready"`
	Head     string `json:"head"`
	HeadHash string `json:"head_hash"`
	Durable  bool   `json:"durable"`
	Reopened bool   `json:"reopened"`
}
type privateGatewayRPCNamespace struct {
	Device string `json:"device"`
	Inode  string `json:"inode"`
}

func privateGatewayRPCArgs(p privatechain.Plan, containerID string) ([]string, error) {
	if _, err := p.Digest(); err != nil || !privateGatewayID(containerID) {
		return nil, errPrivateGatewayRPC
	}
	type member struct {
		Ship   string `json:"ship"`
		Number int    `json:"number"`
	}
	input := struct {
		Foundation privatechain.Foundation `json:"foundation"`
		Members    []member                `json:"members"`
	}{Foundation: p.Foundation}
	for _, m := range p.Members {
		number := 0
		if m.Ship == "~nec" {
			number = 1
		}
		input.Members = append(input.Members, member{Ship: m.Ship, Number: number})
	}
	raw, err := json.Marshal(input)
	if err != nil || len(raw) > 8192 {
		return nil, errPrivateGatewayRPC
	}
	// No shell, ambient host endpoint, user-supplied script, or output of errors
	// from the container. The program emits only validated public observations.
	program := privateGatewayProbe + "\nmain().catch(()=>{process.stderr.write('private_gateway_probe_refused\\n');process.exitCode=1;});\n"
	return []string{"container", "exec", containerID, "/node/usr/lib/ld-linux-x86-64.so.2", "--library-path", "/node/usr/lib", "/node/usr/bin/node", "--eval", program, string(raw)}, nil
}

func parsePrivateGatewayRPC(p privatechain.Plan, raw []byte) (privateGatewayRPCObservation, error) {
	bad := func() (privateGatewayRPCObservation, error) {
		return privateGatewayRPCObservation{}, errPrivateGatewayRPC
	}
	if _, err := p.Digest(); err != nil || len(raw) > 16384 || !privateGatewayUnambiguousJSON(raw) {
		return bad()
	}
	var o privateGatewayRPCObservation
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&o); err != nil {
		return bad()
	}
	validHash := func(h string) bool {
		return strings.HasPrefix(h, "0x") && privateGatewayID(strings.TrimPrefix(h, "0x"))
	}
	foundation, err := o.Foundation.Digest()
	if err != nil || foundation != p.FoundationDigest || o.Format != "omabit-private-gateway-observation-v1" || o.Head != 80 || !validHash(o.HeadHash) ||
		o.Startup.Format != "omabit-private-gateway-startup-v1" || !o.Startup.Ready || !o.Startup.Durable || !o.Startup.Reopened ||
		o.Startup.Head != "0x50" || o.Startup.HeadHash != o.HeadHash || len(o.PublicKeys) != len(p.Members) {
		return bad()
	}
	for _, value := range []string{o.Namespace.Device, o.Namespace.Inode} {
		n, err := strconv.ParseUint(value, 10, 64)
		if err != nil || n == 0 || strconv.FormatUint(n, 10) != value {
			return bad()
		}
	}
	seen := make(map[string]bool)
	for i, key := range o.PublicKeys {
		if key.Ship != p.Members[i].Ship || key.Life != 1 || key.Suite != 1 {
			return bad()
		}
		for _, value := range []string{key.Auth, key.Crypt} {
			if !validHash(value) || seen[value] {
				return bad()
			}
			seen[value] = true
		}
	}
	return o, nil
}

// The complete observer must bracket this call with fresh exact container,
// engine, namespace, sealed program and filesystem checks. A parsed RPC result
// alone does not confer CompleteContract, Private, Synced or RPCReadOnly.
// In particular, startup flags are not trusted without the qualified initializer.
func (q *qualifier) privateGatewayRPC(ctx context.Context, p privatechain.Plan, containerID string) (privateGatewayRPCObservation, error) {
	if err := ctx.Err(); err != nil {
		return privateGatewayRPCObservation{}, err
	}
	args, err := privateGatewayRPCArgs(p, containerID)
	if q == nil || err != nil {
		return privateGatewayRPCObservation{}, errPrivateGatewayRPC
	}
	out, err := q.docker(ctx, 15*time.Second, args...)
	if err != nil {
		return privateGatewayRPCObservation{}, errPrivateGatewayRPC
	}
	return parsePrivateGatewayRPC(p, []byte(out))
}

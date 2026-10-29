// Package projectnetwork defines the explicit fresh-fake project contract.
// Planning has no filesystem, engine, runner or network side effects. A plan is
// not admission to launch: persistence and fresh host/resource checks follow.
package projectnetwork

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

const FakeGalaxies = "fake-galaxies-v1"

// MemberRequest names a new instance and its explicitly reserved runner-loopback
// Eyre port. Imported/existing instances cannot join through this contract.
type MemberRequest struct {
	InstanceID string `json:"instance_id"`
	Ship       string `json:"ship"`
	HTTPPort   uint16 `json:"http_port"`
}

type Request struct {
	Label                       string          `json:"label"`
	Members                     []MemberRequest `json:"members"`
	ConfirmSharedAdministration bool            `json:"confirm_shared_administration"`
}

// Reservations is supplied from a consistent runner store observation, including
// both desired ports and ports still owned by older runtime generations. Build
// does not certify that the caller's observation is fresh or that a port is free.
type Reservations struct {
	Instances map[string]bool
	HTTPPorts map[uint16]bool
}

type Member struct {
	MemberRequest
	InternalHTTPPort uint16 `json:"internal_http_port"`
}

type Plan struct {
	Mode                 string   `json:"mode"`
	Label                string   `json:"label"`
	Members              []Member `json:"members"`
	BindAddress          string   `json:"bind_address"`
	SharedAdministration bool     `json:"shared_administration"`
}

// Build validates one complete initial reservation set. Internal ports are
// deterministic for the reviewed member set; adding/removing members later is
// explicit whole-project maintenance, never an implicit reallocation at Start.
func Build(req Request, reserved Reservations) (Plan, error) {
	invalid := func(detail string) (Plan, error) { return Plan{}, errors.New(detail) }
	if !req.ConfirmSharedAdministration {
		return invalid("acknowledge that project members share administrative loopback access")
	}
	if req.Label == "" || !utf8.ValidString(req.Label) || len(req.Label) > 128 || strings.IndexFunc(req.Label, unicode.IsControl) >= 0 {
		return invalid("project label must contain 1..128 UTF-8 bytes without control characters")
	}
	// The initial transport contract is the finite 256-galaxy identity class.
	// This is not an arbitrary reservation of host ports or hidden spare slots.
	if len(req.Members) > 256 {
		return invalid("a fake project allows up to 256 distinct fresh galaxy members")
	}
	members := slices.Clone(req.Members)
	slices.SortFunc(members, func(a, b MemberRequest) int { return strings.Compare(a.InstanceID, b.InstanceID) })
	ids, ships, ports := map[string]bool{}, map[string]bool{}, map[uint16]bool{}
	result := Plan{Mode: FakeGalaxies, Label: req.Label, BindAddress: "127.0.0.1", SharedAdministration: true, Members: make([]Member, 0, len(members))}
	for i, m := range members {
		if !protocol.ValidID(m.InstanceID) || ids[m.InstanceID] {
			return invalid("member instance IDs must be valid and unique within the project")
		}
		if reserved.Instances[m.InstanceID] {
			return invalid("member instance ID is already registered or reserved")
		}
		// Require the canonical spelling rather than silently normalizing a
		// reviewed identity. Other fake classes need their own transport gate.
		if len(m.Ship) != 4 || m.Ship[0] != '~' || !protocol.ValidFakeShip(m.Ship) || ships[m.Ship] {
			return invalid("member ships must be distinct canonical fake galaxies")
		}
		if m.HTTPPort < 1024 || ports[m.HTTPPort] || reserved.HTTPPorts[m.HTTPPort] {
			return invalid("member HTTP ports must be unprivileged, distinct and unreserved")
		}
		ids[m.InstanceID], ships[m.Ship], ports[m.HTTPPort] = true, true, true
		result.Members = append(result.Members, Member{MemberRequest: m, InternalHTTPPort: uint16(8080 + i)})
	}
	return result, nil
}

// Digest binds the validated, deterministic complete set for later reviewed
// publication. It is neither a project identity nor evidence of a live endpoint.
func (p Plan) Digest() (string, error) {
	request := Request{Label: p.Label, ConfirmSharedAdministration: p.SharedAdministration}
	for _, member := range p.Members {
		request.Members = append(request.Members, member.MemberRequest)
	}
	rebuilt, err := Build(request, Reservations{})
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	canonical, _ := json.Marshal(rebuilt)
	if string(data) != string(canonical) {
		return "", fmt.Errorf("project plan does not match its canonical endpoint contract")
	}
	sum := sha256.Sum256(append([]byte("omabit-project-plan-v1\x00"), data...))
	return hex.EncodeToString(sum[:]), nil
}

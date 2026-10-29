// Package privatechain defines offline review data for local keyed development
// projects. It does not migrate fake projects, grant runtime authority, open a
// chain, generate a key, reserve a port or call a runner.
package privatechain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

const Mode = "private-local-keyed-v1"
const Profile = "azimuth-local-1337-v1"
const FirstStop = "hood-lens-acknowledged-exit-v1"

// Foundation is public review metadata. A matching chain ID alone is insufficient:
// a runtime must freshly observe the genesis and every contract on the selected
// project-owned chain. Head height is deliberately absent because it advances.
type Foundation struct {
	Version   int        `json:"version"`
	Profile   string     `json:"profile"`
	ChainID   uint64     `json:"chain_id"`
	Genesis   string     `json:"genesis"`
	Contracts []Contract `json:"contracts"`
}
type Contract struct {
	Name       string `json:"name"`
	Address    string `json:"address"`
	CodeSHA256 string `json:"code_sha256"`
}

func hash(value string, prefix bool) bool {
	if prefix {
		if !strings.HasPrefix(value, "0x") {
			return false
		}
		value = value[2:]
	}
	if len(value) != 64 || strings.ToLower(value) != value || value == strings.Repeat("0", 64) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func canonicalFoundation(f Foundation) (Foundation, error) {
	if f.Version != 1 || f.Profile != Profile || f.ChainID != 1337 || !hash(f.Genesis, true) || len(f.Contracts) != len(qualifiedContracts) {
		return Foundation{}, errors.New("expected version1 private local chain1337 foundation and nonzero canonical genesis")
	}
	f.Contracts = slices.Clone(f.Contracts)
	slices.SortFunc(f.Contracts, func(a, b Contract) int { return strings.Compare(a.Name, b.Name) })
	seen := map[string]bool{}
	for _, c := range f.Contracts {
		expected, ok := qualifiedContracts[c.Name]
		if !ok || seen[c.Name] || c.Address != expected.Address || c.CodeSHA256 != expected.CodeSHA256 {
			return Foundation{}, errors.New("foundation contract set differs from the qualified local profile")
		}
		seen[c.Name] = true
	}
	return f, nil
}

func (f Foundation) Digest() (string, error) {
	canonical, err := canonicalFoundation(f)
	if err != nil {
		return "", err
	}
	return digest("omabit-private-chain-foundation-v1", canonical), nil
}

func digest(domain string, value any) string {
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(append([]byte(domain+"\x00"), data...))
	return hex.EncodeToString(sum[:])
}

type MemberRequest struct {
	InstanceID string `json:"instance_id"`
	Ship       string `json:"ship"`
	HTTPPort   uint16 `json:"http_port"`
}
type Request struct {
	Label                       string          `json:"label"`
	Foundation                  Foundation      `json:"foundation"`
	Members                     []MemberRequest `json:"members"`
	ConfirmSharedAdministration bool            `json:"confirm_shared_administration"`
}
type Reservations struct {
	Instances map[string]bool
	HTTPPorts map[uint16]bool
}
type Member struct {
	MemberRequest
	InternalHTTPPort uint16 `json:"internal_http_port"`
	InternalAmesPort uint16 `json:"internal_ames_port"`
	FreshKeyRequired bool   `json:"fresh_key_required"`
}
type Plan struct {
	Mode                 string     `json:"mode"`
	Label                string     `json:"label"`
	Foundation           Foundation `json:"foundation"`
	FoundationDigest     string     `json:"foundation_digest"`
	Members              []Member   `json:"members"`
	BindAddress          string     `json:"bind_address"`
	GatewayPort          uint16     `json:"internal_gateway_port"`
	PublishRPC           bool       `json:"publish_rpc"`
	PublishAmes          bool       `json:"publish_ames"`
	SharedAdministration bool       `json:"shared_administration"`
	FirstStop            string     `json:"first_stop"`
}

// Build currently describes the qualified two-member ~zod/~nec happy path only.
// Host port choices are requested reservations, not proof of endpoint publication.
func Build(req Request, reserved Reservations) (Plan, error) {
	bad := func(message string) (Plan, error) { return Plan{}, errors.New(message) }
	if !req.ConfirmSharedAdministration {
		return bad("acknowledge shared administrative loopback access")
	}
	if req.Label == "" || !utf8.ValidString(req.Label) || len(req.Label) > 128 || strings.IndexFunc(req.Label, unicode.IsControl) >= 0 {
		return bad("project label must contain 1..128 UTF-8 bytes without control characters")
	}
	foundation, err := canonicalFoundation(req.Foundation)
	if err != nil {
		return Plan{}, err
	}
	if len(req.Members) != 2 {
		return bad("the initial private profile requires exactly fresh ~zod and ~nec members")
	}
	members := slices.Clone(req.Members)
	slices.SortFunc(members, func(a, b MemberRequest) int { return strings.Compare(a.InstanceID, b.InstanceID) })
	ids, ships, ports := map[string]bool{}, map[string]bool{}, map[uint16]bool{}
	p := Plan{Mode: Mode, Label: req.Label, Foundation: foundation, FoundationDigest: digest("omabit-private-chain-foundation-v1", foundation), BindAddress: "127.0.0.1", GatewayPort: 25432, SharedAdministration: true, FirstStop: FirstStop, Members: make([]Member, 0, 2)}
	for _, m := range members {
		if !protocol.ValidID(m.InstanceID) || ids[m.InstanceID] || reserved.Instances[m.InstanceID] {
			return bad("member instance ID is invalid, duplicated or already reserved")
		}
		number := uint16(0)
		switch m.Ship {
		case "~zod":
		case "~nec":
			number = 1
		default:
			return bad("the initial private profile supports canonical ~zod and ~nec only")
		}
		if ships[m.Ship] {
			return bad("private project ship identities must be distinct")
		}
		if m.HTTPPort < 1024 || ports[m.HTTPPort] || reserved.HTTPPorts[m.HTTPPort] {
			return bad("member HTTP ports must be unprivileged, distinct and unreserved")
		}
		ids[m.InstanceID], ships[m.Ship], ports[m.HTTPPort] = true, true, true
		p.Members = append(p.Members, Member{MemberRequest: m, InternalHTTPPort: 18081 + number, InternalAmesPort: 31337 + number, FreshKeyRequired: true})
	}
	return p, nil
}

func (p Plan) Digest() (string, error) {
	request := Request{Label: p.Label, Foundation: p.Foundation, ConfirmSharedAdministration: p.SharedAdministration}
	for _, m := range p.Members {
		request.Members = append(request.Members, m.MemberRequest)
	}
	canonical, err := Build(request, Reservations{})
	if err != nil {
		return "", err
	}
	if !reflect.DeepEqual(p, canonical) {
		return "", errors.New("private-chain plan differs from its canonical reviewed contract")
	}
	return digest("omabit-private-chain-plan-v1", p), nil
}

// Binding distinguishes projects even when both use the same foundation/genesis
// and ship names. Generation tracks chain lifecycle, not chain head height. Its
// validation is pure data validation; the runner must supply fresh owned evidence.
type Binding struct {
	Mode             string `json:"mode"`
	RunnerID         string `json:"runner_id"`
	ProjectID        string `json:"project_id"`
	Generation       uint64 `json:"generation"`
	FoundationDigest string `json:"foundation_digest"`
}

func (b Binding) Validate() error {
	if b.Mode != Mode || !protocol.ValidID(b.RunnerID) || !protocol.ValidID(b.ProjectID) || b.Generation == 0 || !hash(b.FoundationDigest, false) {
		return errors.New("explicit private-chain project, runner, generation and foundation binding required")
	}
	return nil
}
func Match(expected, observed Binding, foundation Foundation) error {
	if err := expected.Validate(); err != nil {
		return err
	}
	if err := observed.Validate(); err != nil {
		return err
	}
	actual, err := foundation.Digest()
	if err != nil {
		return err
	}
	if expected != observed || actual != expected.FoundationDigest {
		return fmt.Errorf("private-chain project or foundation observation differs")
	}
	return nil
}

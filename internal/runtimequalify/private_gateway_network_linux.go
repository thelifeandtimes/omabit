//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"encoding/json"
	"io"
	"net/netip"
	"reflect"
	"strconv"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

// This is a command/inspection contract, not runtime authority. The concrete
// driver must separately check its rootless engine, retained workspace, input
// generation and journal before an effect. An ordinary bridge permits rootless
// HTTP forwarding; it does not establish an air gap. Qualification uses a no-NIC
// VM. No caller may infer production network isolation from this contract.
type privateGatewayNetworkContract struct {
	selection privatechain.GatewaySelection
}

func newPrivateGatewayNetworkContract(p privatechain.Plan, w privatechain.Workspace, g privatechain.Gateway) (privateGatewayNetworkContract, error) {
	if err := g.Validate(p, w); err != nil {
		return privateGatewayNetworkContract{}, err
	}
	return privateGatewayNetworkContract{selection: g.Selection}, nil
}

func (c privateGatewayNetworkContract) networkName() string {
	return "urbit-" + c.selection.Binding.ProjectID + "-g" + strconv.FormatUint(c.selection.Binding.Generation, 10) + "-bridge"
}

func (c privateGatewayNetworkContract) gatewayName() string {
	return "urbit-" + c.selection.Binding.ProjectID + "-g" + strconv.FormatUint(c.selection.Binding.Generation, 10) + "-gateway"
}

func (c privateGatewayNetworkContract) labels() map[string]string {
	s := c.selection
	return map[string]string{
		projectLabel: "true", projectLabel + ".mode": privatechain.Mode,
		projectLabel + ".runner": s.Binding.RunnerID, projectLabel + ".project": s.Binding.ProjectID,
		projectLabel + ".generation": strconv.FormatUint(s.Binding.Generation, 10),
		projectLabel + ".operation":  s.OperationID, projectLabel + ".plan": s.PlanDigest,
		projectLabel + ".workspace": s.WorkspaceDigest, projectLabel + ".inputs": s.Inputs.ManifestSHA256,
	}
}

func (c privateGatewayNetworkContract) networkArgs() []string {
	args := appendLabels([]string{"network", "create", "--driver", "bridge", "--internal=false", "--ipv4=true", "--ipv6=false"}, c.labels())
	return append(args, "--opt", "com.docker.network.bridge.host_binding_ipv4=127.0.0.1", "--opt", "com.docker.network.bridge.gateway_mode_ipv4=nat", c.networkName())
}

type privateGatewayNetwork struct {
	ID                               string `json:"Id"`
	Name, Driver, Scope              string
	Internal, EnableIPv4, EnableIPv6 *bool
	Attachable, Ingress, ConfigOnly  *bool
	Options, Labels                  map[string]string
	ConfigFrom                       struct{ Network string }
	IPAM                             struct {
		Driver  string
		Options map[string]string
		Config  []map[string]string
	}
	Containers map[string]struct {
		Name, EndpointID, IPv4Address, IPv6Address string
	}
}

// Allocation is freshly observed metadata for later endpoint/route comparisons.
// It is not a saved authorization or a replacement for those comparisons.
type privateGatewayAllocation struct {
	NetworkID, ContainerID string
	Subnet                 netip.Prefix
	Gateway                netip.Addr
	Address                netip.Addr
	EndpointID             string
}

func privateGatewayID(id string) bool {
	return containerIDPattern.MatchString(id) && id != strings.Repeat("0", 64)
}

// Inspect exactly one Docker network response. An empty gateway ID requires an
// empty network; a supplied ID requires exactly that attachment. A container
// merely bearing a matching label/name is never adopted.
func (c privateGatewayNetworkContract) inspectNetwork(data []byte, networkID, gatewayID string) (privateGatewayAllocation, error) {
	bad := func() (privateGatewayAllocation, error) { return privateGatewayAllocation{}, errUntrustedNetwork }
	if !privateGatewayID(networkID) || gatewayID != "" && (!privateGatewayID(gatewayID) || gatewayID == networkID) || len(data) > 1<<20 || !privateGatewayUnambiguousJSON(data) {
		return bad()
	}
	var rows []privateGatewayNetwork
	if json.Unmarshal(data, &rows) != nil || len(rows) != 1 {
		return bad()
	}
	n := rows[0]
	boolean := func(p *bool, want bool) bool { return p != nil && *p == want }
	if n.ID != networkID || n.Name != c.networkName() || n.Driver != "bridge" || n.Scope != "local" ||
		!boolean(n.Internal, false) || !boolean(n.EnableIPv4, true) || !boolean(n.EnableIPv6, false) ||
		!boolean(n.Attachable, false) || !boolean(n.Ingress, false) || !boolean(n.ConfigOnly, false) || n.ConfigFrom.Network != "" ||
		!reflect.DeepEqual(n.Options, bridgeOptions()) || !reflect.DeepEqual(n.Labels, c.labels()) ||
		n.IPAM.Driver != "default" || len(n.IPAM.Options) != 0 || len(n.IPAM.Config) != 1 || len(n.IPAM.Config[0]) != 2 {
		return bad()
	}
	config := n.IPAM.Config[0]
	subnet, err := netip.ParsePrefix(config["Subnet"])
	if err != nil || !subnet.Addr().Is4() || subnet != subnet.Masked() || subnet.String() != config["Subnet"] || subnet.Bits() > 30 {
		return bad()
	}
	private := false
	for _, block := range []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"} {
		p := netip.MustParsePrefix(block)
		private = private || subnet.Bits() >= p.Bits() && p.Contains(subnet.Addr())
	}
	if !private {
		return bad()
	}
	gateway, err := netip.ParseAddr(config["Gateway"])
	if err != nil || gateway.String() != config["Gateway"] || !privateGatewayHost(subnet, gateway) {
		return bad()
	}
	a := privateGatewayAllocation{NetworkID: networkID, ContainerID: gatewayID, Subnet: subnet, Gateway: gateway}
	if gatewayID == "" {
		if len(n.Containers) != 0 {
			return bad()
		}
		return a, nil
	}
	member, ok := n.Containers[gatewayID]
	if !ok || len(n.Containers) != 1 || member.Name != c.gatewayName() || !privateGatewayID(member.EndpointID) || member.IPv6Address != "" {
		return bad()
	}
	address, err := netip.ParsePrefix(member.IPv4Address)
	if err != nil || address.String() != member.IPv4Address || address.Bits() != subnet.Bits() || !privateGatewayHost(subnet, address.Addr()) || address.Addr() == gateway {
		return bad()
	}
	a.Address, a.EndpointID = address.Addr(), member.EndpointID
	return a, nil
}

func privateGatewayHost(subnet netip.Prefix, address netip.Addr) bool {
	if !address.Is4() || !subnet.Contains(address) || address == subnet.Addr() {
		return false
	}
	// The address immediately following the broadcast leaves the prefix. This
	// also handles a final-octet rollover without uint32 arithmetic.
	return subnet.Contains(address.Next())
}

// encoding/json otherwise accepts repeated fields (including case aliases),
// letting a later empty Containers or trusted Labels value hide an earlier one.
// Docker adds metadata over time, so unknown fields remain acceptable, but no
// object may have ambiguous fields. Bound nesting before decoding typed rows.
func privateGatewayUnambiguousJSON(data []byte) bool {
	d := json.NewDecoder(bytes.NewReader(data))
	var visit func(int) bool
	visit = func(depth int) bool {
		if depth > 16 {
			return false
		}
		token, err := d.Token()
		if err != nil {
			return false
		}
		open, nested := token.(json.Delim)
		if !nested {
			return true
		}
		if open != '{' && open != '[' {
			return false
		}
		seen := map[string]bool{}
		for d.More() {
			if open == '{' {
				token, err := d.Token()
				key, ok := token.(string)
				if err != nil || !ok || seen[strings.ToLower(key)] {
					return false
				}
				seen[strings.ToLower(key)] = true
			}
			if !visit(depth + 1) {
				return false
			}
		}
		end, err := d.Token()
		return err == nil && (open == '{' && end == json.Delim('}') || open == '[' && end == json.Delim(']'))
	}
	return visit(0) && d.Decode(new(any)) == io.EOF
}

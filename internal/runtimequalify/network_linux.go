//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	errUntrustedNetwork = errors.New("qualification network identity or isolation mismatch")
	errBindingPending   = errors.New("loopback HTTP binding is not assigned yet")
)

type bridge struct {
	ID                                          string `json:"Id"`
	Name, Driver, Scope                         string
	Internal, EnableIPv4                        *bool
	EnableIPv6, Attachable, Ingress, ConfigOnly bool
	Options, Labels                             map[string]string
	Containers                                  map[string]json.RawMessage
}

func isHTTPKind(kind string) bool { return kind == "http-boot" || kind == "http-restart" }
func isCometKind(kind string) bool {
	return kind == "comet-boot" || kind == "comet-resume" || kind == "comet-verify"
}
func isCometRestart(kind string) bool { return kind == "comet-post-pack" || kind == "comet-post-meld" }
func cometProfile(profile string) bool {
	return profile == ProfileComet || profile == ProfileCometMaintenance || cometEpochProfile(profile)
}
func isNetworkedKind(kind string) bool {
	return isHTTPKind(kind) || isCometKind(kind) || isCometRestart(kind) || isCometEpochRestart(kind) || isCometPrefixRuntime(kind)
}
func networkProfile(profile string) bool {
	return profile == ProfileHTTP || cometProfile(profile)
}

func bridgeOptions() map[string]string {
	return map[string]string{
		"com.docker.network.bridge.host_binding_ipv4": "127.0.0.1",
		"com.docker.network.bridge.gateway_mode_ipv4": "nat",
	}
}

func (q *qualifier) inspectNetwork(ctx context.Context) (bridge, error) {
	r := q.r.Network
	if !networkProfile(q.d.profile) || r == nil || r.Removed {
		return bridge{}, errUntrustedNetwork
	}
	out, err := q.docker(ctx, q.d.times.command, "network", "inspect", r.Name)
	if err != nil {
		return bridge{}, err
	}
	var list []bridge
	if json.Unmarshal([]byte(out), &list) != nil || len(list) != 1 {
		return bridge{}, errUntrustedNetwork
	}
	n := list[0]
	if !containerIDPattern.MatchString(n.ID) || n.Name != r.Name ||
		(r.ID != "" && r.ID != n.ID) || n.Labels[q.resourceLabel()] != "true" || n.Labels[q.resourceRunLabel()] != q.r.RunID ||
		n.Driver != "bridge" || n.Scope != "local" || n.Internal == nil || *n.Internal ||
		n.EnableIPv4 == nil || !*n.EnableIPv4 || n.EnableIPv6 || n.Attachable || n.Ingress || n.ConfigOnly ||
		!reflect.DeepEqual(n.Options, bridgeOptions()) {
		return bridge{}, errUntrustedNetwork
	}
	// Never disconnect or adopt an unrelated container, even on our own bridge.
	for id := range n.Containers {
		owned := false
		for _, c := range q.r.Containers {
			owned = owned || (c.ID == id && isNetworkedKind(c.Kind))
		}
		if !owned {
			return bridge{}, errors.Join(errUntrustedNetwork, errors.New("unrecognized network attachment"))
		}
	}
	r.ID, r.CreateUncertain = n.ID, false
	return n, nil
}

func (q *qualifier) createNetwork(ctx context.Context) error {
	if !networkProfile(q.d.profile) || q.r.Network != nil {
		return errors.New("network creation is only available once in the HTTP profile")
	}
	q.r.Network = &NetworkRecord{Name: q.resourceName("http")}
	if err := q.save(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	q.r.Network.CreateDispatched, q.r.Network.CreateUncertain = true, true
	if err := q.save(); err != nil {
		q.r.Network.CreateDispatched, q.r.Network.CreateUncertain = false, false
		return err
	}
	out, err := q.docker(ctx, q.d.times.command, "network", "create",
		"--driver", "bridge", "--internal=false", "--ipv4=true", "--ipv6=false",
		"--label", q.resourceLabel()+"=true", "--label", q.resourceRunLabel()+"="+q.r.RunID,
		"--opt", "com.docker.network.bridge.host_binding_ipv4=127.0.0.1",
		"--opt", "com.docker.network.bridge.gateway_mode_ipv4=nat", q.r.Network.Name)
	if id := strings.TrimSpace(out); containerIDPattern.MatchString(id) {
		q.r.Network.ID = id
	}
	if err = errors.Join(err, q.save()); err != nil {
		return err
	}
	_, err = q.inspectNetwork(ctx)
	return errors.Join(err, q.save())
}

func (q *qualifier) cleanupNetwork() error {
	r := q.r.Network
	if r == nil || r.Removed {
		return nil
	}
	if !r.CreateDispatched {
		r.Removed = true
		return q.save()
	}
	for _, c := range q.r.Containers {
		if !c.Removed || c.CreateUncertain {
			return errors.New("network retained while container cleanup is unresolved")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), q.d.times.cleanup)
	defer cancel()
	var last error
	for ctx.Err() == nil {
		attempt, done := context.WithTimeout(ctx, q.d.times.cleanupCommand)
		n, err := q.inspectNetwork(attempt)
		done()
		if errors.Is(err, errUntrustedNetwork) {
			return err
		}
		if err == nil {
			if err = q.save(); err != nil {
				return err
			}
			if len(n.Containers) != 0 {
				err = errors.New("network still has known attachments; waiting without disconnecting them")
			} else if _, err = q.docker(ctx, q.d.times.cleanupCommand, "network", "rm", n.ID); err == nil {
				r.Removed = true
				return q.save()
			}
		}
		// A missing network after a timed-out create/rm is still an uncertain
		// daemon-side outcome. Never infer completion from one empty inventory.
		last = err
		timer := time.NewTimer(q.d.times.poll)
		select {
		case <-ctx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
	return errors.Join(last, errors.New("network cleanup deadline exceeded; manual resolution required"))
}

type portBinding struct {
	HostIP   string `json:"HostIp"`
	HostPort string `json:"HostPort"`
}

func decodeLoopbackBinding(raw json.RawMessage, dynamic bool) (portBinding, error) {
	var list []portBinding
	if json.Unmarshal(raw, &list) != nil || len(list) != 1 || list[0].HostIP != "127.0.0.1" {
		return portBinding{}, errors.New("HTTP port must have exactly one IPv4 loopback binding")
	}
	b := list[0]
	if dynamic && (b.HostPort == "" || b.HostPort == "0") {
		return b, nil
	}
	n, err := strconv.Atoi(b.HostPort)
	if err != nil || n < 1024 || n > 65535 || strconv.Itoa(n) != b.HostPort {
		return portBinding{}, errors.New("HTTP host port must be a canonical unprivileged TCP port")
	}
	return b, nil
}

func (q *qualifier) networkContainment(c container, kind string) error {
	if !isNetworkedKind(kind) {
		if c.HostConfig.NetworkMode != "none" || len(c.HostConfig.PortBindings) != 0 {
			return errors.New("isolated container cannot have network access or published ports")
		}
		for _, binding := range c.NetworkSettings.Ports {
			if string(bytes.TrimSpace(binding)) != "null" {
				return errors.New("isolated container has an unexpected port binding")
			}
		}
		for name := range c.NetworkSettings.Networks {
			if name != "none" {
				return errors.New("isolated container joined an unexpected network")
			}
		}
		return nil
	}
	n := q.r.Network
	allowed := isHTTPKind(kind) && q.d.profile == ProfileHTTP || isCometKind(kind) && q.d.profile == ProfileComet ||
		(isCometKind(kind) || isCometRestart(kind)) && q.d.profile == ProfileCometMaintenance && !q.d.managed ||
		(isCometKind(kind) || isCometEpochRestart(kind) && cometEpochContainer(q.d.profile, kind)) && cometEpochProfile(q.d.profile) && !q.d.managed ||
		isCometPrefixRuntime(kind) && q.d.profile == ProfileCometPrefix && !q.d.managed
	if !allowed || n == nil || n.Removed || n.CreateUncertain || !containerIDPattern.MatchString(n.ID) ||
		(c.HostConfig.NetworkMode != n.ID && c.HostConfig.NetworkMode != n.Name) {
		return errUntrustedNetwork
	}
	wantPorts := 1
	amesKey := strconv.Itoa(int(q.d.amesPort)) + "/udp"
	if q.publishAmes(kind) {
		wantPorts++
		if err := q.amesBinding(c.HostConfig.PortBindings[amesKey]); err != nil {
			return err
		}
	}
	if len(c.HostConfig.PortBindings) != wantPorts {
		return errors.New("container port bindings differ from the registered HTTP/Ames configuration")
	}
	binding, err := decodeLoopbackBinding(c.HostConfig.PortBindings["8080/tcp"], true)
	if err != nil {
		return err
	}
	if q.d.hostPort != 0 && binding.HostPort != strconv.Itoa(int(q.d.hostPort)) {
		return errors.New("managed HTTP port differs from the recorded configuration")
	}
	if len(c.NetworkSettings.Networks) > 1 {
		return errors.New("HTTP container joined additional networks")
	}
	for name, raw := range c.NetworkSettings.Networks {
		var endpoint struct{ NetworkID string }
		if (name != n.Name && name != n.ID) || json.Unmarshal(raw, &endpoint) != nil ||
			(endpoint.NetworkID != "" && endpoint.NetworkID != n.ID) || (*c.State.Running && endpoint.NetworkID != n.ID) {
			return errUntrustedNetwork
		}
	}
	if *c.State.Running && len(c.NetworkSettings.Networks) != 1 {
		return errUntrustedNetwork
	}
	for port, raw := range c.NetworkSettings.Ports {
		if string(bytes.TrimSpace(raw)) == "null" {
			continue // image EXPOSE metadata is not a host publication
		}
		if port != "8080/tcp" {
			if q.publishAmes(kind) && port == amesKey {
				if err := q.amesBinding(raw); err != nil {
					return err
				}
				continue
			}
			return errors.New("HTTP container published an unexpected port")
		}
		if _, err := decodeLoopbackBinding(raw, false); err != nil {
			return err
		}
	}
	if q.publishAmes(kind) && *c.State.Running {
		if err := q.amesBinding(c.NetworkSettings.Ports[amesKey]); err != nil {
			return err
		}
	}
	return nil
}

func (q *qualifier) amesBinding(raw json.RawMessage) error {
	var bindings []portBinding
	if json.Unmarshal(raw, &bindings) != nil || len(bindings) != 1 || bindings[0].HostIP != "0.0.0.0" || bindings[0].HostPort != strconv.Itoa(int(q.d.amesPort)) {
		return errors.New("Ames UDP binding differs from the registered host port")
	}
	return nil
}

func (q *qualifier) publishedEndpoint(c container, kind string) (string, error) {
	if !isNetworkedKind(kind) {
		return "", errors.New("host endpoint is only available for the HTTP checkpoint")
	}
	if err := q.containment(c, kind); err != nil {
		return "", err
	}
	raw := c.NetworkSettings.Ports["8080/tcp"]
	if len(raw) == 0 || string(bytes.TrimSpace(raw)) == "null" {
		return "", errBindingPending
	}
	b, err := decodeLoopbackBinding(raw, false)
	if err != nil {
		return "", err
	}
	if q.d.hostPort != 0 && b.HostPort != strconv.Itoa(int(q.d.hostPort)) {
		return "", errors.New("published port differs from the recorded managed port")
	}
	return fmt.Sprintf("http://%s/~/host", net.JoinHostPort("127.0.0.1", b.HostPort)), nil
}

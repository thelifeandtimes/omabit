package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"omarchy-urbit/internal/fleet"
)

type Host struct {
	Name string `json:"name"`
	SSH  string `json:"ssh,omitempty"`
}
type Config struct {
	Schema  int             `json:"schema"`
	Default string          `json:"default_host"`
	Hosts   map[string]Host `json:"hosts"`
}

func LoadClient(p fleet.Paths) (Config, error) {
	c := Config{Schema: fleet.Schema, Default: "local", Hosts: map[string]Host{"local": {Name: "local"}}}
	e := fleet.ReadJSON(filepath.Join(p.Config, "client.json"), &c)
	if os.IsNotExist(e) {
		return c, nil
	}
	if e == nil && (c.Schema != fleet.Schema || c.Hosts == nil) {
		e = fmt.Errorf("unsupported client config")
	}
	return c, e
}
func AddHost(p fleet.Paths, name, ssh string, def bool) (Config, error) {
	lock, e := fleet.AcquireLock(filepath.Join(p.Config, "client.lock"))
	if e != nil {
		return Config{}, e
	}
	defer lock.Close()
	c, e := LoadClient(p)
	if e != nil {
		return c, e
	}
	if e = fleet.ValidateName(name); e != nil {
		return c, e
	}
	if name == "local" || ssh == "" {
		return c, fmt.Errorf("remote alias cannot be local; --ssh is required")
	}
	if e = ValidateSSH(ssh); e != nil {
		return c, e
	}
	c.Hosts[name] = Host{Name: name, SSH: ssh}
	if def {
		c.Default = name
	}
	return c, fleet.AtomicJSON(filepath.Join(p.Config, "client.json"), c)
}

type Client struct {
	Paths fleet.Paths
	Host  Host
	Agent bool
}

func NewClient(p fleet.Paths, name string, agent bool) (Client, error) {
	c, e := LoadClient(p)
	if e != nil {
		return Client{}, e
	}
	if name == "" {
		name = c.Default
	}
	h, ok := c.Hosts[name]
	if !ok {
		return Client{}, fmt.Errorf("host alias %q not registered", name)
	}
	return Client{Paths: p, Host: h, Agent: agent}, nil
}
func (c Client) Call(ctx context.Context, q fleet.Request) (fleet.Response, error) {
	if c.Host.SSH == "" {
		return Local(ctx, c.Paths, c.Agent, q)
	}
	return Remote(ctx, c.Host.SSH, c.Agent, q)
}

type HostInventory struct {
	Name      string          `json:"name"`
	Location  string          `json:"location"`
	Reachable bool            `json:"reachable"`
	Stale     bool            `json:"stale"`
	Error     string          `json:"error,omitempty"`
	Snapshot  json.RawMessage `json:"snapshot,omitempty"`
}

// A failed refresh retains the last observation but explicitly marks it stale.
// It never writes stopped state into an unreachable host's inventory.
func Inventory(ctx context.Context, p fleet.Paths, agent bool) ([]HostInventory, error) {
	conf, e := LoadClient(p)
	if e != nil {
		return nil, e
	}
	names := []string{}
	for n := range conf.Hosts {
		names = append(names, n)
	}
	sort.Strings(names)
	result := []HostInventory{}
	for _, n := range names {
		h := conf.Hosts[n]
		entry := HostInventory{Name: n, Location: "local"}
		if h.SSH != "" {
			entry.Location = "remote"
		}
		c := Client{p, h, agent}
		v, err := c.Call(ctx, fleet.Request{Schema: fleet.Schema, Method: "ship.list"})
		cache := filepath.Join(p.State, "inventory", n+".json")
		if err == nil && v.OK {
			entry.Reachable = true
			c.Decorate(&v)
			entry.Snapshot = v.Result
			if e = fleet.AtomicJSON(cache, v.Result); e != nil {
				return nil, e
			}
		} else {
			entry.Stale = true
			if err != nil {
				entry.Error = err.Error()
			} else if v.Error != nil {
				entry.Error = v.Error.Message
			}
			var saved json.RawMessage
			if fleet.ReadJSON(cache, &saved) == nil {
				entry.Snapshot = saved
			}
		}
		result = append(result, entry)
	}
	return result, nil
}

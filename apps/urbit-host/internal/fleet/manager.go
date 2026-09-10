package fleet

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Manager struct {
	mu      sync.Mutex
	DB      DB
	Engine  *Engine
	dbPath  string
	serial  sync.Mutex
	work    sync.WaitGroup
	closing bool
}

func NewManager(e *Engine) (*Manager, error) {
	m := &Manager{DB: NewDB(), Engine: e, dbPath: filepath.Join(e.Paths.State, "host.json")}
	if err := ReadJSON(m.dbPath, &m.DB); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if m.DB.Schema != Schema {
		return nil, fmt.Errorf("unsupported database schema")
	}
	if m.DB.Instances == nil || m.DB.Groups == nil || m.DB.Operations == nil || m.DB.Requests == nil || m.DB.Fingerprints == nil || m.DB.Notices == nil {
		return nil, fmt.Errorf("invalid database maps")
	}
	for id, op := range m.DB.Operations {
		if op.State == "running" || op.State == "queued" {
			op.State = "interrupted"
			op.Finished = Now()
			op.Message = "host service restarted; inspect instance before an explicit retry"
			m.DB.Operations[id] = op
		}
	}
	for id, i := range m.DB.Instances {
		i.Busy = ""
		m.DB.Instances[id] = i
	}
	if err := m.persist(); err != nil {
		return nil, err
	}
	return m, nil
}
func (m *Manager) persist() error { return AtomicJSON(m.dbPath, m.DB) }

// Transaction snapshots metadata, not pier contents. A failed commit must not
// leave accepted in-memory state that disappears when the daemon restarts.
func (m *Manager) transaction(f func() error) error {
	old, _ := json.Marshal(m.DB)
	if err := f(); err != nil {
		json.Unmarshal(old, &m.DB)
		return err
	}
	if err := m.persist(); err != nil {
		json.Unmarshal(old, &m.DB)
		return err
	}
	return nil
}
func (m *Manager) find(target string) (Instance, error) {
	if i, ok := m.DB.Instances[target]; ok {
		return i, nil
	}
	var found []Instance
	for _, i := range m.DB.Instances {
		if i.Label == target && i.Lifecycle != "purged" {
			found = append(found, i)
		}
	}
	if len(found) != 1 {
		return Instance{}, E("not_found", "no uniquely matching instance; use the full instance ID")
	}
	return found[0], nil
}
func (m *Manager) groupFor(i Instance) *Group {
	if i.Group == "" {
		return nil
	}
	g := m.DB.Groups[i.Group]
	return &g
}
func (m *Manager) Snapshot() ([]Instance, []Group) {
	m.mu.Lock()
	defer m.mu.Unlock()
	is := []Instance{}
	gs := []Group{}
	for _, i := range m.DB.Instances {
		is = append(is, i)
	}
	for _, g := range m.DB.Groups {
		gs = append(gs, g)
	}
	sort.Slice(is, func(a, b int) bool { return is[a].Created < is[b].Created })
	sort.Slice(gs, func(a, b int) bool { return gs[a].Created < gs[b].Created })
	return is, gs
}
func validateCreate(r *Request) error {
	if err := ValidateName(r.Label); err != nil {
		return err
	}
	if r.Kind != "fake" && r.Kind != "comet" && r.Kind != "keyed" {
		return E("invalid_request", "kind must be fake, comet, or keyed")
	}
	r.Identity = strings.TrimPrefix(strings.TrimSpace(r.Identity), "~")
	if r.Kind == "fake" {
		if _, err := Galaxy(r.Identity); err != nil {
			return err
		}
		if r.Group == "" {
			return E("invalid_request", "fake ships require a group")
		}
		if r.Key != "" {
			return E("invalid_request", "fake ships do not accept networking keys")
		}
	}
	if r.Kind == "comet" && (r.Identity != "" || r.Key != "" || r.Group != "") {
		return E("invalid_request", "comets accept no identity, key or development group")
	}
	if r.Kind == "keyed" {
		if r.Group != "" {
			return E("invalid_request", "real ships cannot join fake groups")
		}
		if r.Key == "" {
			return E("invalid_request", "networking key required; failed keyed boots never fall back to comet")
		}
		if len(r.Identity) < 3 || len(r.Identity) > 63 || strings.Trim(r.Identity, "abcdefghijklmnopqrstuvwxyz-") != "" {
			return E("invalid_request", "invalid identity syntax; Vere performs canonical key/identity validation")
		}
	}
	if len(r.Key) > 16384 || len(r.Hoon) > 65536 {
		return E("invalid_request", "secret or expression too large")
	}
	if r.Channel == "" {
		r.Channel = "latest"
	}
	if r.Channel != "latest" && r.Channel != "edge" && r.Channel != "canary" {
		return E("invalid_request", "unknown GroundSeg release channel")
	}
	if r.Loom == 0 {
		r.Loom = 31
	}
	if r.Loom < 30 || r.Loom > 34 {
		return E("invalid_request", "pilot loom exponent must be 30 through 34; this is a runtime setting, not a fleet resource limit")
	}
	if r.Image != "" {
		if err := ValidateImage(r.Image); err != nil {
			return err
		}
	}
	if r.Pill != "" {
		u, err := url.Parse(r.Pill)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
			return E("invalid_request", "pill override must be an HTTPS URL without credentials")
		}
	}
	return nil
}
func authorize(role string, r Request, i *Instance) error {
	if role == "owner" {
		if i != nil && i.Real() {
			switch r.Method {
			case "ship.start", "ship.stop", "ship.restart", "ship.archive", "ship.exec":
				if r.Confirm != i.ID {
					return E("confirmation_required", "repeat with --confirm INSTANCE_ID for this real-ship operation")
				}
			}
		}
		return nil
	}
	if role != "agent" {
		return E("forbidden", "unknown role")
	}
	switch r.Method {
	case "doctor", "ship.list", "ship.inspect", "group.list", "operation.inspect", "operation.list", "metrics", "notices.list":
		return nil
	case "group.create":
		return nil
	case "ship.create":
		if r.Kind == "fake" {
			return nil
		}
	case "ship.start", "ship.stop", "ship.restart", "ship.archive", "ship.purge", "ship.code", "ship.exec", "ship.logs", "ship.dojo":
		if i != nil && !i.Real() {
			return nil
		}
	}
	return E("forbidden", "agent endpoint permits development management, but not real-ship mutation, code retrieval, logs or Dojo access")
}
func (m *Manager) Handle(ctx context.Context, role string, r Request) Response {
	v, err := m.handle(ctx, role, r)
	if err != nil {
		return Failure(err)
	}
	return Success(v)
}
func (m *Manager) handle(ctx context.Context, role string, r Request) (any, error) {
	raw, _ := json.Marshal(r)
	rawSum := sha256.Sum256(append([]byte(role+"\x00"), raw...))
	fingerprint := hex.EncodeToString(rawSum[:])
	if r.Schema != Schema {
		return nil, E("schema_mismatch", "expected schema 1")
	}
	if r.Method == "ship.create" {
		if r.Channel == "" {
			r.Channel = m.Engine.Host.Channel
		}
		if r.Image == "" {
			r.Image = m.Engine.Host.ImageOverride
		}
		if err := validateCreate(&r); err != nil {
			return nil, err
		}
	}
	m.mu.Lock()
	var inst *Instance
	if strings.HasPrefix(r.Method, "ship.") && r.Method != "ship.create" && r.Method != "ship.list" {
		i, err := m.find(r.Target)
		if err != nil {
			m.mu.Unlock()
			return nil, err
		}
		inst = &i
	}
	if err := authorize(role, r, inst); err != nil {
		m.mu.Unlock()
		return nil, err
	}
	if m.closing {
		m.mu.Unlock()
		return nil, E("unavailable", "host service is shutting down")
	}
	switch r.Method {
	case "group.create":
		if err := ValidateName(r.Label); err != nil {
			m.mu.Unlock()
			return nil, err
		}
		for _, g := range m.DB.Groups {
			if g.Name == r.Label {
				m.mu.Unlock()
				return g, nil
			}
		}
		base := 20000 + len(m.DB.Groups)*256
		if base+255 >= 40000 {
			m.mu.Unlock()
			return nil, E("capacity", "pilot group HTTP port range exhausted")
		}
		id := ID()
		g := Group{ID: id, Name: r.Label, Network: "ou-net-" + id, Namespace: "ou-ns-" + id, HTTPBase: base, Created: Now()}
		err := m.transaction(func() error { m.DB.Groups[id] = g; return nil })
		m.mu.Unlock()
		return g, err
	case "group.list":
		m.mu.Unlock()
		_, gs := m.Snapshot()
		return gs, nil
	case "operation.inspect":
		op, ok := m.DB.Operations[r.Target]
		m.mu.Unlock()
		if !ok {
			return nil, E("not_found", "operation not found")
		}
		return op, nil
	case "operation.list":
		ops := []Operation{}
		for _, o := range m.DB.Operations {
			ops = append(ops, o)
		}
		m.mu.Unlock()
		sort.Slice(ops, func(a, b int) bool { return ops[a].Started < ops[b].Started })
		return ops, nil
	case "notices.list":
		a := []Notice{}
		for _, n := range m.DB.Notices {
			if !n.Dismissed {
				a = append(a, n)
			}
		}
		m.mu.Unlock()
		return a, nil
	case "notices.dismiss":
		n, ok := m.DB.Notices[r.Target]
		if !ok {
			m.mu.Unlock()
			return nil, E("not_found", "notice not found")
		}
		n.Dismissed = true
		err := m.transaction(func() error { m.DB.Notices[n.ID] = n; return nil })
		m.mu.Unlock()
		return n, err
	case "ship.create", "ship.start", "ship.stop", "ship.restart", "ship.archive", "ship.purge":
		v, err := m.submitLocked(role, r, inst, fingerprint)
		m.mu.Unlock()
		return v, err
	}
	var g *Group
	if inst != nil {
		g = m.groupFor(*inst)
	}
	m.mu.Unlock()
	switch r.Method {
	case "doctor":
		checks := map[string]any{"service_version": Version, "host": m.Engine.Host, "docker": false, "tailscale": "checked-by-client-pilot-script", "backups": "out-of-scope", "cross_host_devnet": "not-integrated", "real_ships_pilot_gate": !m.Engine.Host.AllowReal}
		b, err := m.Engine.run(ctx, "info", "--format", "{{json .ServerVersion}}")
		if err != nil {
			checks["docker_error"] = err.Error()
		} else {
			checks["docker"] = true
			checks["docker_version"] = strings.TrimSpace(string(b))
		}
		return checks, nil
	case "ship.list":
		is, gs := m.Snapshot()
		a := []Observation{}
		for _, i := range is {
			var g *Group
			for _, x := range gs {
				if x.ID == i.Group {
					copy := x
					g = &copy
				}
			}
			a = append(a, m.Engine.Observe(ctx, i, g))
		}
		return map[string]any{"host_id": m.Engine.Host.ID, "host_name": m.Engine.Host.Name, "observed_at": Now(), "instances": a}, nil
	case "ship.inspect":
		return m.Engine.Observe(ctx, *inst, g), nil
	case "ship.logs":
		if inst.Archived != "" {
			return nil, E("unavailable", "archived container has been removed")
		}
		if _, err := m.Engine.CheckContainer(ctx, *inst); err != nil {
			return nil, err
		}
		b, err := m.Engine.run(ctx, "logs", "--tail", "200", inst.Container)
		return map[string]string{"logs": string(b)}, err
	case "ship.dojo":
		s, err := m.Engine.CheckContainer(ctx, *inst)
		if err != nil {
			return nil, err
		}
		if s != "running" {
			return nil, E("unavailable", "ship is not running")
		}
		return map[string]any{"container": inst.Container, "command": []string{"docker", "exec", "-it", inst.Container, "tmux", "attach-session", "-t", "urbit"}}, nil
	case "ship.code":
		code, err := m.Engine.Control(ctx, *inst, "+code")
		if err != nil {
			return nil, err
		}
		code = strings.Trim(code, "\"\n\r ")
		parts := strings.Split(code, "-")
		if len(parts) != 4 {
			return nil, E("unsupported_runtime", "unexpected code format")
		}
		for _, p := range parts {
			if len(p) != 6 || strings.Trim(p, "abcdefghijklmnopqrstuvwxyz") != "" {
				return nil, E("unsupported_runtime", "unexpected code format")
			}
		}
		return map[string]string{"code": code}, nil
	case "ship.exec":
		if strings.TrimSpace(r.Hoon) == "" || len(r.Hoon) > 65536 {
			return nil, E("invalid_request", "a Hoon expression up to 64 KiB is required")
		}
		out, err := m.Engine.Control(ctx, *inst, r.Hoon)
		return map[string]string{"output": out}, err
	case "metrics":
		is, _ := m.Snapshot()
		stats, err := m.Engine.Stats(ctx, is)
		if err != nil {
			return nil, err
		}
		return BudgetMetrics(stats, m.Engine.Host), nil
	case "updates.check":
		return m.CheckUpdates(ctx)
	default:
		return nil, E("unknown_method", "unknown RPC method")
	}
}
func (m *Manager) submitLocked(role string, r Request, inst *Instance, fp string) (Operation, error) {
	if r.ID == "" {
		return Operation{}, E("invalid_request", "mutation requires request_id")
	}
	if len(r.ID) > 128 {
		return Operation{}, E("invalid_request", "request_id too long")
	}
	// A request ID belongs to a role and exactly one payload, including the secret
	// digest (not its plaintext). Replays return the original operation.
	// Fingerprint the original request, before applying mutable host defaults.
	if old, ok := m.DB.Requests[r.ID]; ok {
		if m.DB.Fingerprints[r.ID] != fp {
			return Operation{}, E("conflict", "request_id reused with a different payload or role")
		}
		return m.DB.Operations[old], nil
	}
	var i Instance
	if r.Method == "ship.create" {
		if r.Kind != "fake" && !m.Engine.Host.AllowReal {
			return Operation{}, E("pilot_gate", "real-network ships are disabled until the disposable-ship pilot is validated; see docs/PILOT.md")
		}
		for _, old := range m.DB.Instances {
			if old.Label == r.Label && old.Lifecycle != "purged" {
				return Operation{}, E("conflict", "label already exists, including archived instances")
			}
			if r.Kind == "keyed" && old.Real() && old.Identity == r.Identity && old.Lifecycle != "purged" {
				return Operation{}, E("conflict", "this real identity is already registered on this host")
			}
		}
		id := ID()
		i = Instance{ID: id, Label: r.Label, Kind: r.Kind, Identity: r.Identity, Channel: r.Channel, ImageOverride: r.Image, Pill: r.Pill, Loom: r.Loom, Container: "ou-ship-" + id, HTTPPort: 8080, Desired: "running", Lifecycle: "creating", Created: Now()}
		if i.Kind == "fake" {
			var g Group
			for _, x := range m.DB.Groups {
				if x.Name == r.Group || x.ID == r.Group {
					g = x
					break
				}
			}
			if g.ID == "" {
				return Operation{}, E("not_found", "create the development group first")
			}
			idx, _ := Galaxy(i.Identity)
			i.Group = g.ID
			i.HTTPPort = 8080 + idx
			i.AmesPort = 31337 + idx
			for _, old := range m.DB.Instances {
				if old.Group == g.ID && old.Identity == i.Identity && old.Archived == "" && old.Lifecycle != "purged" {
					return Operation{}, E("conflict", "fake identity already exists in this development group")
				}
			}
		} else {
			port, err := m.freePort()
			if err != nil {
				return Operation{}, err
			}
			i.HostPort = port
			i.AmesPort = port + 10000
		}
	} else {
		i = *inst
		if i.Busy != "" {
			return Operation{}, E("busy", "another lifecycle operation is active")
		}
		if i.Lifecycle == "purged" {
			return Operation{}, E("conflict", "instance is permanently purged")
		}
		if r.Method == "ship.purge" {
			if i.Archived == "" {
				return Operation{}, E("conflict", "archive the pier before purging")
			}
			if r.Confirm != i.ID {
				return Operation{}, E("confirmation_required", "permanent purge requires --confirm INSTANCE_ID")
			}
		} else if i.Archived != "" {
			return Operation{}, E("conflict", "archived instances cannot be started or modified")
		}
		if i.Lifecycle == "archiving" && r.Method != "ship.archive" {
			return Operation{}, E("conflict", "finish the interrupted archive first")
		}
	}
	op := Operation{ID: ID(), RequestID: r.ID, Action: r.Method, Instance: i.ID, Role: role, State: "queued", Started: Now()}
	i.Busy = op.ID
	if err := m.transaction(func() error {
		m.DB.Instances[i.ID] = i
		m.DB.Operations[op.ID] = op
		m.DB.Requests[r.ID] = op.ID
		m.DB.Fingerprints[r.ID] = fp
		return nil
	}); err != nil {
		return Operation{}, err
	}
	key := r.Key
	r.Key = ""
	m.work.Add(1)
	go func() { defer m.work.Done(); m.execute(op, i, key) }()
	return op, nil
}
func (m *Manager) freePort() (int, error) {
	for p := 41000; p < 50000; p++ {
		used := false
		for _, i := range m.DB.Instances {
			if i.HostPort == p && i.Lifecycle != "purged" {
				used = true
			}
		}
		if used {
			continue
		}
		l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p))
		if err != nil {
			continue
		}
		l.Close()
		u, err := net.ListenPacket("udp", fmt.Sprintf(":%d", p+10000))
		if err != nil {
			continue
		}
		u.Close()
		return p, nil
	}
	return 0, E("capacity", "no free pilot ports")
}
func (m *Manager) setInstance(i Instance) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.transaction(func() error { m.DB.Instances[i.ID] = i; return nil })
}
func (m *Manager) execute(op Operation, i Instance, key string) {
	m.serial.Lock()
	defer m.serial.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	m.mu.Lock()
	op.State = "running"
	err := m.transaction(func() error { m.DB.Operations[op.ID] = op; return nil })
	g := m.groupFor(i)
	m.mu.Unlock()
	if err == nil {
		switch op.Action {
		case "ship.create", "ship.start":
			if i.Image == "" {
				i.Image = i.ImageOverride
				if i.Image == "" {
					i.Image, err = ResolveVersion(ctx, m.Engine.Host.VersionURL, i.Channel)
				}
				if err == nil && i.ImageOverride != "" {
					err = m.Engine.EnsureImage(ctx, i.Image)
					if err == nil {
						var b []byte
						b, err = m.Engine.run(ctx, "image", "inspect", "--format", "{{.Id}}", i.Image)
						if err == nil {
							pin := strings.TrimSpace(string(b))
							if len(pin) != 71 || !strings.HasPrefix(pin, "sha256:") {
								err = fmt.Errorf("image ID could not be pinned")
							} else {
								i.Image = pin
							}
						}
					}
				}
				if err == nil {
					err = m.setInstance(i)
				}
			}
			var secret string
			if err == nil && i.Kind == "keyed" {
				var root string
				root, err = CheckPath(m.Engine.Host.DataRoot, filepath.Join(i.RelativeDir(), ".boot-secret", "network.key"))
				secret = filepath.Dir(root)
				if err == nil {
					err = os.MkdirAll(secret, 0700)
				}
				if err == nil && key != "" {
					err = os.WriteFile(filepath.Join(secret, "network.key"), []byte(key), 0600)
				}
				key = ""
			}
			if err == nil {
				err = m.Engine.Create(ctx, i, g, secret)
			}
			if err == nil {
				i.Desired = "running"
				i.Lifecycle = "active"
			}
		case "ship.stop":
			stopCtx, c := context.WithTimeout(ctx, 120*time.Second)
			err = m.Engine.Stop(stopCtx, i)
			c()
			if err == nil {
				i.Desired = "stopped"
			}
		case "ship.restart":
			stopCtx, c := context.WithTimeout(ctx, 120*time.Second)
			err = m.Engine.Stop(stopCtx, i)
			c()
			if err == nil {
				err = m.Engine.Start(ctx, i)
			}
			if err == nil {
				i.Desired = "running"
			}
		case "ship.archive":
			err = m.archive(ctx, &i)
		case "ship.purge":
			var p string
			p, err = CheckPath(m.Engine.Host.DataRoot, filepath.Join("archive", i.ID))
			if err == nil {
				err = os.RemoveAll(p)
			}
			if err == nil {
				i.Lifecycle = "purged"
				i.Desired = "stopped"
			}
		}
	}
	m.mu.Lock()
	i.Busy = ""
	op.Finished = Now()
	if err != nil {
		op.State = "failed"
		op.Message = err.Error()
	} else {
		op.State = "succeeded"
		op.Message = "lifecycle action completed; runtime readiness is reported separately"
	}
	if saveErr := m.transaction(func() error { m.DB.Instances[i.ID] = i; m.DB.Operations[op.ID] = op; return nil }); saveErr != nil {
		log.Printf("operation %s persistence failed: %v", op.ID, saveErr)
	}
	m.mu.Unlock()
}
func (m *Manager) archive(ctx context.Context, i *Instance) error {
	i.Lifecycle = "archiving"
	if err := m.setInstance(*i); err != nil {
		return err
	}
	exists, err := m.Engine.Exists(ctx, i.Container)
	if err != nil {
		return err
	}
	if exists {
		stopCtx, c := context.WithTimeout(ctx, 120*time.Second)
		err = m.Engine.Stop(stopCtx, *i)
		c()
		if err != nil {
			return err
		}
		if err = m.Engine.Remove(ctx, *i); err != nil {
			return err
		}
	}
	src, err := CheckPath(m.Engine.Host.DataRoot, i.RelativeDir())
	if err != nil {
		return err
	}
	dst, err := CheckPath(m.Engine.Host.DataRoot, filepath.Join("archive", i.ID))
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
		return err
	}
	_, srcErr := os.Lstat(src)
	_, dstErr := os.Lstat(dst)
	if srcErr == nil && os.IsNotExist(dstErr) {
		if err = os.Rename(src, dst); err != nil {
			return err
		}
	} else if os.IsNotExist(srcErr) && dstErr == nil { /* replay after rename, before metadata commit */
	} else if os.IsNotExist(srcErr) && os.IsNotExist(dstErr) {
		if err = os.Mkdir(dst, 0700); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("archive paths ambiguous; refusing overwrite")
	}
	if secret, e := CheckPath(m.Engine.Host.DataRoot, filepath.Join("archive", i.ID, ".boot-secret", "network.key")); e == nil {
		if e = os.Remove(secret); e != nil && !os.IsNotExist(e) {
			return e
		}
	} else {
		return e
	}
	for _, parent := range []string{filepath.Dir(src), filepath.Dir(dst)} {
		f, e := os.Open(parent)
		if os.IsNotExist(e) && parent == filepath.Dir(src) {
			continue
		}
		if e != nil {
			return e
		}
		e = f.Sync()
		f.Close()
		if e != nil {
			return e
		}
	}
	i.Archived = Now()
	i.Desired = "stopped"
	i.Lifecycle = "archived"
	return nil
}
func (m *Manager) CheckUpdates(ctx context.Context) ([]Notice, error) {
	is, _ := m.Snapshot()
	refs := map[string]string{}
	notices := []Notice{}
	for _, i := range is {
		if i.Archived != "" || i.Image == "" || i.ImageOverride != "" {
			continue
		}
		ref, ok := refs[i.Channel]
		if !ok {
			var err error
			ref, err = ResolveVersion(ctx, m.Engine.Host.VersionURL, i.Channel)
			if err != nil {
				return nil, err
			}
			refs[i.Channel] = ref
		}
		if ref == i.Image {
			continue
		}
		sum := sha256.Sum256([]byte(i.ID + ref))
		id := hex.EncodeToString(sum[:16])
		n := Notice{ID: id, Instance: i.ID, Image: ref, Message: "A different runtime image is available on " + i.Channel + ". Review before applying; no container was changed.", Created: Now()}
		m.mu.Lock()
		old, exists := m.DB.Notices[id]
		if exists {
			n = old
		}
		err := m.transaction(func() error { m.DB.Notices[id] = n; return nil })
		m.mu.Unlock()
		if err != nil {
			return nil, err
		}
		if !n.Dismissed {
			notices = append(notices, n)
		}
	}
	return notices, nil
}
func (m *Manager) Close() { m.mu.Lock(); m.closing = true; m.mu.Unlock(); m.work.Wait() }

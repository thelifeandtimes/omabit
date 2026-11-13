package main

// This guarded assembled test uses the actual receiver artifacts and their
// long-lived consumers. Its launcher owns a private Secret Service and network.
import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/applicationauth"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"
)

type reconnectChannel struct {
	token   string
	frames  []any
	active  int
	deleted bool
}

type reconnectPeer struct {
	t           *testing.T
	mu          sync.Mutex
	sessions    map[string]bool
	origins     map[string]string
	channels    map[string]*reconnectChannel
	profile     json.RawMessage
	talon       map[string]any
	logins      int
	logouts     int
	sequence    int
	rejectToken string
}

func (p *reconnectPeer) active(token string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, c := range p.channels {
		if c.token == token && c.active > 0 && !c.deleted {
			return true
		}
	}
	return false
}

func (p *reconnectPeer) serve(w http.ResponseWriter, r *http.Request, offline *atomic.Bool) {
	if offline.Load() {
		w.WriteHeader(503)
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if r.URL.Path == "/~/host" {
		if r.Header.Get("Cookie") != "" {
			p.t.Error("public identity probe received credential")
		}
		fmt.Fprint(w, "~zod")
		return
	}
	cookie, _ := r.Cookie("urbauth-~zod")
	if r.URL.Path == "/~/login" {
		if cookie != nil || r.Method != "POST" || r.ParseForm() != nil || r.Form.Get("password") != "lidlut-tabwed-pillex-ridrup" {
			p.t.Error("invalid private login request")
			w.WriteHeader(400)
			return
		}
		p.logins++
		token := fmt.Sprintf("reconnect-private-cookie-%d", p.logins)
		p.sessions[token] = true
		p.origins[token] = r.Host
		http.SetCookie(w, &http.Cookie{Name: "urbauth-~zod", Value: token, Path: "/", MaxAge: 3600})
		return
	}
	if cookie != nil && p.origins[cookie.Value] != r.Host {
		p.t.Error("credential crossed its original URL")
		w.WriteHeader(403)
		return
	}
	if cookie == nil || !p.sessions[cookie.Value] {
		w.WriteHeader(401)
		return
	}
	if r.URL.Path == "/~/name" && p.rejectToken == cookie.Value {
		w.WriteHeader(403)
		return
	}
	respond := func(v any) { w.Header().Set("Content-Type", "application/json"); reply(p.t, w, v) }
	switch r.URL.Path {
	case "/~/name":
		fmt.Fprint(w, "~zod")
		return
	case "/~/logout":
		p.logouts++
		delete(p.sessions, cookie.Value)
		w.Header().Set("Location", "/~/login")
		w.WriteHeader(303)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/~/scry/") {
		switch {
		case strings.Contains(r.URL.Path, "/settings/"):
			respond(map[string]any{"desk": map[string]any{"appearance": map[string]any{"current": string(p.profile)}, "ui-prefs": p.talon}})
		case r.URL.Path == "/~/scry/groups-ui/v4/heads.json":
			respond(map[string]any{"dms": []any{}})
		case strings.Contains(r.URL.Path, "/chat/"):
			respond([]any{})
		default:
			respond(map[string]any{})
		}
		return
	}
	if !strings.HasPrefix(r.URL.Path, "/~/channel/") {
		p.t.Error("unexpected private Eyre path", r.Method, r.URL.Path)
		w.WriteHeader(404)
		return
	}
	c := p.channels[r.URL.Path]
	if r.Method == "PUT" {
		var messages []map[string]any
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&messages); err != nil {
			p.t.Error(err)
			w.WriteHeader(400)
			return
		}
		if c == nil {
			c = &reconnectChannel{token: cookie.Value}
			p.channels[r.URL.Path] = c
		}
		if c.token != cookie.Value {
			p.t.Error("different credential used an existing channel")
			w.WriteHeader(403)
			return
		}
		for _, m := range messages {
			switch m["action"] {
			case "subscribe":
				c.frames = append(c.frames, map[string]any{"id": m["id"], "response": "subscribe", "ok": "ok"})
			case "ack":
			case "delete":
				c.deleted = true
			case "poke":
				// The unmodified Theme Talon bridge may update its palette. No
				// messaging action or appearance publication is allowed here.
				body, ok := m["json"].(map[string]any)
				put, _ := body["put-entry"].(map[string]any)
				if !ok || m["app"] != "settings" || put["desk"] != "talon" || put["bucket-key"] != "ui-prefs" || (put["entry-key"] != "themes" && put["entry-key"] != "accent") {
					p.t.Error("unexpected application write", m["app"], m["mark"])
					w.WriteHeader(400)
					return
				}
				p.talon[put["entry-key"].(string)] = put["value"]
				c.frames = append(c.frames, map[string]any{"id": m["id"], "response": "poke", "ok": "ok"})
			default:
				p.t.Error("unexpected channel action", m["action"])
			}
		}
		w.WriteHeader(204)
		return
	}
	if r.Method != "GET" || c == nil || c.token != cookie.Value {
		w.WriteHeader(404)
		return
	}
	c.active++
	defer func() { c.active-- }()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Connection", "close")
	w.WriteHeader(200)
	for !c.deleted && p.sessions[cookie.Value] && !offline.Load() {
		var body strings.Builder
		for _, frame := range c.frames {
			p.sequence++
			raw, _ := json.Marshal(frame)
			fmt.Fprintf(&body, "id: %d\ndata: %s\n\n", p.sequence, raw)
		}
		c.frames = nil
		if body.Len() == 0 {
			body.WriteString(": heartbeat\n\n")
		}
		p.mu.Unlock()
		_, err := io.WriteString(w, body.String())
		if err == nil {
			err = http.NewResponseController(w).Flush()
		}
		if err == nil {
			select {
			case <-r.Context().Done():
				err = r.Context().Err()
			case <-time.After(50 * time.Millisecond):
			}
		}
		p.mu.Lock()
		if err != nil {
			return
		}
	}
}

func TestFleetRealReceiverReconnect(t *testing.T) {
	if os.Getenv("OMABIT_REAL_RECONNECT") != "private-v1" {
		t.Skip("requires scripts/qualification/fleet/real_reconnect.py private capsule")
	}
	if os.Getenv("HOME") != "/lab/home" || os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		t.Fatal("private capsule environment missing")
	}
	profile, err := os.ReadFile("/lab/profile.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, plugin := range []string{"thelifeandtimes.urbit-theme.reconnect-lab", "urbit.dms.reconnect-lab"} {
		t.Run(plugin, func(t *testing.T) {
			ctx := context.Background()
			p := &reconnectPeer{t: t, sessions: map[string]bool{}, origins: map[string]string{}, channels: map[string]*reconnectChannel{}, profile: profile, talon: map[string]any{}}
			// Two URLs reach the same ship/session authority, as with a relay.
			var oldOffline, newOffline atomic.Bool
			first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { p.serve(w, r, &oldOffline) }))
			defer first.Close()
			second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { p.serve(w, r, &newOffline) }))
			defer second.Close()
			binding := shiproute.Binding{RunnerID: "reconnect-runner", Target: apppolicy.Target{InstanceID: "alpha"}, Ship: "~zod", Continuity: strings.Repeat("c", 64)}
			selection := protocol.ShipRouteSelection{Target: binding.Target, ConfigRevision: 3, RuntimeID: strings.Repeat("a", 64)}
			record := shiproute.DefaultRecord(binding)
			record.Revision, record.Actor, record.UpdatedAt = 1, "private-fixture", time.Now().UTC()
			for i, url := range []string{first.URL, second.URL} {
				record.Settings.Routes = append(record.Settings.Routes, shiproute.Route{ID: fmt.Sprintf("route-%d", i), Origin: url, Network: "local", Scope: binding.RunnerID, Source: "manual", Enabled: true})
			}
			routes, _ := record.Settings.Effective(binding, 1, nil)
			var codes atomic.Int32
			socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/hello":
					reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: binding.RunnerID, Mode: "simulation", AccessRole: "operator", Features: []string{protocol.AppPolicyFeature, protocol.ShipRoutesFeature, protocol.ShipRouteCodeFeature}})
				case protocol.AppPolicyPath:
					reply(t, w, protocol.AppPolicyStatus{Version: 1, RunnerID: binding.RunnerID, Mode: "simulation", ReadAt: time.Now().UTC(), Policy: apppolicy.Default(), Targets: []protocol.AppPolicyTarget{{Target: binding.Target, Ship: binding.Ship, Ready: true}}})
				case "/v1/snapshot":
					reply(t, w, protocol.Snapshot{RunnerID: binding.RunnerID, Mode: "simulation", Instances: []protocol.Instance{{ID: "alpha", ConfigRevision: 3, ObservedState: "running", Resources: &protocol.ResourceMetrics{ContainerID: selection.RuntimeID}}}})
				case protocol.ShipRoutesPath:
					reply(t, w, protocol.ShipRoutes{Version: 1, RunnerID: binding.RunnerID, Mode: "simulation", Selection: selection, Record: record, Registry: routes, ReadAt: time.Now().UTC()})
				case protocol.ShipRouteCodePath:
					var q protocol.ShipRouteCodeRequest
					if json.NewDecoder(r.Body).Decode(&q) != nil || q.Binding != binding || q.Selection != selection || q.RegistryRevision != 1 || (q.Route != record.Settings.Routes[0] && q.Route != record.Settings.Routes[1]) {
						t.Error("incorrect route/code binding")
						w.WriteHeader(400)
						return
					}
					codes.Add(1)
					reply(t, w, protocol.ShipRouteCode{Version: 1, RunnerID: binding.RunnerID, Mode: "simulation", Request: q, Code: "lidlut-tabwed-pillex-ridrup"})
				default:
					t.Error("unexpected runner request", r.URL.Path)
					w.WriteHeader(404)
				}
			})
			dir := filepath.Join("/lab/evidence", plugin)
			registry, err := collection.Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = registry.Enroll(ctx, "fixture", socket); err != nil {
				t.Fatal(err)
			}
			invoke := func(want int, args ...string) string {
				t.Helper()
				var out, stderr bytes.Buffer
				code := run(ctx, append([]string{"--config-dir", dir, "--format", "json"}, args...), &out, &stderr)
				if strings.Contains(out.String()+stderr.String(), "reconnect-private-cookie") || strings.Contains(out.String()+stderr.String(), "lidlut-tabwed-pillex-ridrup") {
					t.Fatal("credential in public CLI output")
				}
				if want >= 0 && code != want {
					t.Fatalf("%v exit=%d want=%d out=%s err=%s", args, code, want, &out, &stderr)
				}
				return out.String()
			}
			var installed []applicationauth.InstalledPlugin
			if err := json.Unmarshal([]byte(invoke(0, "app-plugins")), &installed); err != nil {
				t.Fatal(err)
			}
			var target collection.ApplicationTarget
			for _, item := range installed {
				if item.Target.PluginID == plugin && item.Replacement {
					target = item.Target
				}
			}
			if target.PluginID == "" {
				t.Fatalf("staged capable plugin missing: %+v", installed)
			}
			allow := true
			grant, _ := json.Marshal(collection.ApplicationShipApprovalRequest{Version: 1, Binding: binding, Target: target, Allowed: &allow, Selection: &selection})
			pin := []string{"--runner", "fixture", "--expected-runner-id", binding.RunnerID}
			invoke(0, append(append([]string{}, pin...), "app-ship-approve", "--request-json", string(grant), "--allow-owner-session")...)
			request, _ := json.Marshal(collection.FleetApplicationRequest{Version: 1, Binding: binding, Target: target, GrantRevision: 1})
			var result applicationResult
			parse := func(raw string) *applicationView {
				t.Helper()
				result = applicationResult{}
				if err := json.Unmarshal([]byte(raw), &result); err != nil || result.Connection == nil {
					t.Fatalf("invalid result %s: %v", raw, err)
				}
				return result.Connection
			}
			current := parse(invoke(0, append(append([]string{}, pin...), "app-fleet-connect", "--request-json", string(request), "--request-id", collection.NewSessionRef())...))
			if current.Phase != "connected" || current.Origin != first.URL {
				t.Fatal(current)
			}
			consumer := exec.Command("/usr/bin/python3", "-B", "/lab/consumer.py", filepath.Join(os.Getenv("HOME"), ".config/omarchy/plugins", plugin))
			stdin, err := consumer.StdinPipe()
			if err != nil {
				t.Fatal(err)
			}
			stdout, err := consumer.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			stderr, err := os.Create(dir + "/consumer.stderr")
			if err != nil {
				t.Fatal(err)
			}
			defer stderr.Close()
			consumer.Stderr = stderr
			if err = consumer.Start(); err != nil {
				t.Fatal(err)
			}
			var messagesMu sync.Mutex
			var messages []json.RawMessage
			readerDone := make(chan struct{})
			go func() {
				defer close(readerDone)
				scanner := bufio.NewScanner(stdout)
				scanner.Buffer(make([]byte, 4096), 1<<20)
				for scanner.Scan() {
					raw := append(json.RawMessage(nil), scanner.Bytes()...)
					if !json.Valid(raw) || bytes.Contains(raw, []byte("reconnect-private-cookie")) {
						t.Error("invalid or secret consumer output")
						continue
					}
					messagesMu.Lock()
					messages = append(messages, raw)
					messagesMu.Unlock()
				}
			}()
			finished := make(chan error, 1)
			go func() { finished <- consumer.Wait() }()
			defer func() {
				oldOffline.Store(false)
				io.WriteString(stdin, "{\"action\":\"shutdown-transport\"}\n")
				select {
				case err := <-finished:
					if err != nil {
						t.Error("consumer shutdown", err)
					}
				case <-time.After(20 * time.Second):
					consumer.Process.Kill()
					<-finished
					t.Error("consumer required forced stop")
				}
				stdin.Close()
				<-readerDone
				messagesMu.Lock()
				defer messagesMu.Unlock()
				raw, _ := json.MarshalIndent(messages, "", "  ")
				os.WriteFile(dir+"/consumer-events.json", raw, 0600)
			}()
			wait := func(label string, predicate func() bool) {
				t.Helper()
				deadline := time.Now().Add(25 * time.Second)
				for !predicate() {
					if time.Now().After(deadline) {
						messagesMu.Lock()
						defer messagesMu.Unlock()
						t.Fatalf("%s timed out; consumer=%s", label, messages)
					}
					time.Sleep(50 * time.Millisecond)
				}
			}
			wait("initial active stream", func() bool { return p.active("reconnect-private-cookie-1") })
			maintain := func(want int, id string) string {
				t.Helper()
				a, archived, err := registry.Application(ctx, id)
				if err != nil || archived {
					t.Fatal("missing attempt", id, err)
				}
				return invoke(want, "app-fleet-maintain", id, "--expected-revision", strconv.FormatUint(a.Revision, 10))
			}
			cleanup := func(id string) {
				t.Helper()
				for i := 0; i < 4; i++ {
					_, archived, err := registry.Application(ctx, id)
					if err != nil {
						t.Fatal(err)
					}
					if archived {
						return
					}
					maintain(0, id)
				}
				t.Fatal("predecessor not archived")
			}
			for stage := 2; stage <= 4; stage++ {
				previous := current.RequestID
				p.mu.Lock()
				p.rejectToken = fmt.Sprintf("reconnect-private-cookie-%d", stage-1)
				p.mu.Unlock()
				if stage >= 3 {
					oldOffline.Store(true)
				}
				fault := filepath.Join("/lab/faults", plugin)
				want := 0
				if stage == 4 {
					if err := os.WriteFile(fault, nil, 0600); err != nil {
						t.Fatal(err)
					}
					want = 1
				}
				current = parse(maintain(want, previous))
				if current.Replacement.PreviousID != previous {
					t.Fatal("replacement lost lineage", current)
				}
				if stage == 4 {
					if current.Phase != "delivering" {
						t.Fatal("lost acknowledgement was not retained", current)
					}
					if err := os.Remove(fault); err != nil {
						t.Fatal(err)
					}
					current = parse(maintain(0, current.RequestID))
				}
				if current.Phase != "connected" {
					t.Fatal(current)
				}
				if stage >= 3 && current.Origin != second.URL {
					t.Fatal("route change not selected", current)
				}
				wait(fmt.Sprintf("successor %d active stream", stage), func() bool { return p.active(fmt.Sprintf("reconnect-private-cookie-%d", stage)) })
				wait("predecessor stream drained", func() bool { return !p.active(fmt.Sprintf("reconnect-private-cookie-%d", stage-1)) })
				if stage == 3 {
					maintain(1, previous)
					if !p.active("reconnect-private-cookie-3") {
						t.Fatal("pending old cleanup stopped new transport")
					}
					oldOffline.Store(false)
				}
				cleanup(previous)
				a, _, err := registry.Application(ctx, current.RequestID)
				if err != nil {
					t.Fatal(err)
				}
				parse(invoke(0, "app-fleet-health", a.ID, "--expected-revision", strconv.FormatUint(a.Revision, 10)))
				if result.Health != "healthy" || !p.active(fmt.Sprintf("reconnect-private-cookie-%d", stage)) {
					t.Fatal("old cleanup harmed successor", result.Health)
				}
				p.mu.Lock()
				logins, logouts := p.logins, p.logouts
				p.mu.Unlock()
				if logins != stage || logouts != stage-1 || codes.Load() != int32(stage) {
					t.Fatal("unexpected issuance/revocation count", logins, logouts, codes.Load())
				}
				t.Logf("stage %d: actual consumer resumed, predecessor archived, fresh credential count=%d", stage, logins)
			}
			calls, err := os.ReadFile(filepath.Join("/lab/evidence", "wire-"+plugin+".log"))
			if err != nil {
				t.Fatal(err)
			}
			for mode, want := range map[string]int{"initial\n": 1, "--prepare-replacement\n": 3, "--replace\n": 3, "--replacement-status\n": 2} {
				if strings.Count(string(calls), mode) != want {
					t.Fatal("helper call replay or missing observation", mode, string(calls))
				}
			}
		})
	}
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestManagedApplicationSelection(t *testing.T) {
	now := time.Now()
	s := managedApplicationSelection{Runner: "fixture", RunnerID: "runner-fixture", InstanceID: "alpha", Revision: 2, ContainerID: strings.Repeat("a", 64), Ship: "~zod"}
	valid := protocol.Instance{ID: "alpha", ConfigRevision: 2, Ship: "~zod", ObservedState: "running", Endpoint: "http://127.0.0.1:18080", Resources: &protocol.ResourceMetrics{ContainerID: s.ContainerID}, ObservedAt: &now}
	snapshot := protocol.Snapshot{RunnerID: s.RunnerID, Instances: []protocol.Instance{valid}}
	if err := s.validate(snapshot, now); err != nil {
		t.Fatal(err)
	}
	for _, fault := range []string{"runner", "identity", "revision", "container", "stopped", "busy", "recovery", "stale", "unknown"} {
		t.Run(fault, func(t *testing.T) {
			x := snapshot
			x.Instances = append([]protocol.Instance(nil), snapshot.Instances...)
			i := &x.Instances[0]
			switch fault {
			case "runner":
				x.RunnerID = "other"
			case "identity":
				i.Ship = "~nec"
			case "revision":
				i.ConfigRevision++
			case "container":
				i.Resources = &protocol.ResourceMetrics{ContainerID: strings.Repeat("b", 64)}
			case "stopped":
				i.ObservedState = "stopped"
			case "busy":
				i.ActiveOperationID = "work"
			case "recovery":
				i.RecoveryRequired = true
			case "stale":
				old := now.Add(-time.Minute)
				i.ObservedAt = &old
			case "unknown":
				i.IdentityPending = true
			}
			if err := s.validate(x, now); err == nil {
				t.Fatal("changed selection accepted")
			}
		})
	}
}

func TestManagedApplicationPrivateLoginAndReuse(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	var logins, codes atomic.Int32
	var authenticated atomic.Bool
	shipServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/~/host":
			fmt.Fprint(w, "~zod")
		case "/~/login":
			if r.Header.Get("Cookie") != "" {
				t.Error("parent login unexpectedly inherited cookies")
			}
			r.ParseForm()
			if r.Form.Get("password") != "lidlut-tabwed-pillex-ridrup" {
				t.Error("wrong private code")
			}
			logins.Add(1)
			authenticated.Store(true)
			http.SetCookie(w, &http.Cookie{Name: "urbauth-~zod", Value: "fixture-private-cookie", Path: "/", MaxAge: 3600})
		case "/~/name":
			c, e := r.Cookie("urbauth-~zod")
			if e != nil || c.Value != "fixture-private-cookie" || !authenticated.Load() {
				w.WriteHeader(403)
				return
			}
			fmt.Fprint(w, "~zod")
		case "/~/logout":
			w.WriteHeader(303)
		default:
			t.Error("unexpected HTTP path", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer shipServer.Close()
	s := managedApplicationSelection{Runner: "fixture", RunnerID: "runner-fixture", InstanceID: "alpha", Revision: 2, ContainerID: strings.Repeat("a", 64), Ship: "~zod", Origin: shipServer.URL}
	snapshot := protocol.Snapshot{RunnerID: s.RunnerID, Instances: []protocol.Instance{{ID: s.InstanceID, ConfigRevision: s.Revision, Ship: s.Ship, ObservedState: "running", Endpoint: s.Origin, Resources: &protocol.ResourceMetrics{ContainerID: s.ContainerID}, ObservedAt: &now}}}
	// A private test Unix socket, never an installed runner or Docker daemon.
	sockDir, err := os.MkdirTemp("", "omabit-auth-rpc-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(sockDir)
	socket := filepath.Join(sockDir, "runner.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	rpc := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(protocol.Header, "1")
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/hello":
			json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: protocol.Version, RunnerID: s.RunnerID, Mode: "development-managed", AccessRole: "operator", Features: []string{"login-code"}})
		case "/v1/snapshot":
			json.NewEncoder(w).Encode(snapshot)
		case "/v1/instances/alpha/login-code":
			var request protocol.LoginCodeRequest
			if json.NewDecoder(r.Body).Decode(&request) != nil || request.ExpectedRevision != s.Revision || request.ContainerID != s.ContainerID {
				t.Error("unbound code request")
			}
			codes.Add(1)
			json.NewEncoder(w).Encode(protocol.LoginCode{RunnerID: s.RunnerID, InstanceID: s.InstanceID, ConfigRevision: s.Revision, ContainerID: s.ContainerID, Code: "lidlut-tabwed-pillex-ridrup"})
		default:
			t.Error("unexpected runner RPC", r.URL.Path)
			w.WriteHeader(404)
		}
	})}
	go rpc.Serve(listener)
	defer rpc.Close()
	registry, err := collection.Open(filepath.Join(t.TempDir(), "collector"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = registry.Enroll(ctx, s.Runner, socket); err != nil {
		t.Fatal(err)
	}
	// Explicit file adapter for this headless fixture, never an automatic fallback.
	if _, err = registry.EnrollShip(ctx, "fixture-ship", s.Origin, s.Ship, "file"); err != nil {
		t.Fatal(err)
	}
	first, err := prepareManagedApplicationShip(ctx, registry, s)
	if err != nil || first.SessionRef == "" {
		t.Fatal("private managed login", err)
	}
	again, err := prepareManagedApplicationShip(ctx, registry, s)
	if err != nil || again.ID != first.ID || again.SessionRef != first.SessionRef || logins.Load() != 1 || codes.Load() != 1 {
		t.Fatal("parent session was not reused", err, logins.Load(), codes.Load())
	}
	stale := s
	stale.Revision++
	if _, err = prepareManagedApplicationShip(ctx, registry, stale); err == nil || codes.Load() != 1 {
		t.Fatal("stale fleet selection accessed code", err)
	}
	authenticated.Store(false)
	refreshed, err := prepareManagedApplicationShip(ctx, registry, s)
	if err != nil || refreshed.SessionRef == first.SessionRef || logins.Load() != 2 || codes.Load() != 2 {
		t.Fatal("expired managed session did not recover", err)
	}
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "~nec") }))
	defer bad.Close()
	wrong := s
	wrong.Origin = bad.URL
	if _, err = prepareManagedApplicationShip(ctx, registry, wrong); err == nil || codes.Load() != 2 {
		t.Fatal("wrong HTTP identity reached code retrieval", err)
	}
}

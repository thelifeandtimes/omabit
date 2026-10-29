package eyre

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

// This test is only reachable through the opt-in, fresh-pier namespace harness.
// It never accepts a user-supplied ship URL or a production credential.
func requireIsolatedAuthLab(t *testing.T) {
	t.Helper()
	if os.Getenv("OMABIT_AUTH_LAB") != "1" {
		t.Skip("run scripts/qualification/app-connections/qualify.py in its isolated lab")
	}
	var guard struct {
		HostNetworkNamespace string `json:"host_network_namespace"`
		HostPIDNamespace     string `json:"host_pid_namespace"`
		RunID                string `json:"run_id"`
	}
	data, err := os.ReadFile("/run/omabit-auth-lab.json")
	if err != nil || json.Unmarshal(data, &guard) != nil || !strings.HasPrefix(guard.RunID, "omabit-auth-") {
		t.Fatal("missing isolated qualification guard")
	}
	netns, err := os.Readlink("/proc/self/ns/net")
	if err != nil || netns == guard.HostNetworkNamespace || guard.HostNetworkNamespace == "" {
		t.Fatal("qualification must not use the host network namespace")
	}
	pidns, err := os.Readlink("/proc/self/ns/pid")
	if err != nil || pidns == guard.HostPIDNamespace || guard.HostPIDNamespace == "" {
		t.Fatal("qualification must not use the host process namespace")
	}
	for _, forbidden := range []string{"/home", "/run/user", "/var/run/docker.sock"} {
		if _, err := os.Stat(forbidden); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("host resource must not be mounted in the lab", forbidden)
		}
	}
}

func TestIndependentSessionsIsolatedLive(t *testing.T) {
	requireIsolatedAuthLab(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	owner, err := New("http://127.0.0.1:18081", "~zod")
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	if err := owner.Login(ctx, []byte(fakeCode)); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := owner.Logout(context.Background()); err != nil {
			t.Error("owner cleanup", err)
		}
	}()
	first, err := owner.NewIndependentSession(ctx)
	if err != nil {
		t.Fatal("first independent session", err)
	}
	defer first.Close()
	defer func() {
		if first.session.Value != "" {
			if err := first.Logout(context.Background()); err != nil {
				t.Error("first cleanup", err)
			}
		}
	}()
	second, err := owner.NewIndependentSession(ctx)
	if err != nil {
		t.Fatal("second independent session", err)
	}
	defer second.Close()
	defer func() {
		if err := second.Logout(context.Background()); err != nil {
			t.Error("second cleanup", err)
		}
	}()
	if first.session.Value == second.session.Value {
		t.Fatal("plugin sessions reused a cookie")
	}
	for _, client := range []*Client{first, second} {
		data, err := client.Scry(ctx, "hood", "/kiln/pikes")
		if err != nil {
			t.Fatal("direct desk inventory", err)
		}
		var desks map[string]json.RawMessage
		if json.Unmarshal(data, &desks) != nil || desks["base"] == nil {
			t.Fatal("missing base desk")
		}
	}
	copy, err := first.ExportSession()
	if err != nil {
		t.Fatal(err)
	}
	replay, err := New(first.origin, first.ship)
	if err != nil {
		t.Fatal(err)
	}
	defer replay.Close()
	if err := replay.RestoreSession(copy); err != nil {
		t.Fatal(err)
	}
	if err := first.Logout(ctx); err != nil {
		t.Fatal("plugin cleanup", err)
	}
	if err := replay.Check(ctx); !errors.Is(err, ErrAuth) {
		t.Fatal("revoked cookie still accepted", err)
	}
	for _, client := range []*Client{owner, second} {
		if err := client.Check(ctx); err != nil {
			t.Fatal("another session was revoked", err)
		}
	}
	t.Log("PASS: fresh isolated fake; two independent sessions; direct desk reads; remote revocation affects only the selected session; credentials omitted")
}

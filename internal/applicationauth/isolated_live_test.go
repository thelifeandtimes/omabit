package applicationauth

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/credentials"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
)

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

type lostLiveReply struct {
	*Artifact
	offered []byte
}

func (r *lostLiveReply) Deliver(ctx context.Context, data []byte) ([]byte, error) {
	var offer struct {
		Session json.RawMessage `json:"session"`
	}
	if json.Unmarshal(data, &offer) != nil {
		return nil, ErrDelivery
	}
	r.offered = append([]byte(nil), offer.Session...)
	reply, err := r.Artifact.Deliver(ctx, data)
	clear(reply)
	if err != nil {
		return nil, err
	}
	return nil, ErrDelivery // receiver committed, but issuer did not get its ack
}

func TestDurablePluginHandoffIsolatedLive(t *testing.T) {
	requireIsolatedAuthLab(t)
	if os.Getenv("OMABIT_PLUGIN_LAB") != "1" {
		t.Skip("requires both plugin artifacts")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	registryDir := "/lab/issuer-collector"
	r, err := collection.Open(registryDir)
	if err != nil {
		t.Fatal(err)
	}
	ship, err := r.EnrollShip(ctx, "lab", "http://127.0.0.1:18081", "~zod", "secret-service")
	if err != nil {
		t.Fatal(err)
	}
	owner, err := eyre.New(ship.Origin, ship.Ship)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	if err := owner.Login(ctx, []byte(fixtureCode)); err != nil {
		t.Fatal(err)
	}
	ownerData, err := owner.ExportSession()
	if err != nil {
		t.Fatal(err)
	}
	defer clear(ownerData)
	store := credentials.NewSecretService()
	ref := collection.NewSessionRef()
	ship, err = r.SaveShipSession(ctx, ship, ref, func(ctx context.Context) error {
		return store.Save(ctx, credentials.Scope(ship.ID, ship.Origin, ref), ownerData)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := eyre.RevokeStoredSession(context.Background(), ship.Origin, ship.Ship, append([]byte(nil), ownerData...)); err != nil {
			t.Error("owner cleanup failed", err)
		}
		if err := store.Delete(context.Background(), credentials.Scope(ship.ID, ship.Origin, ref)); err != nil {
			t.Error("owner local cleanup", err)
		}
	}()
	env := map[string]string{}
	for _, key := range []string{"HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_RUNTIME_DIR", "DBUS_SESSION_BUS_ADDRESS", "LANG"} {
		env[key] = os.Getenv(key)
	}
	env["XDG_STATE_HOME"] = "/lab/issuer-plugin-state"
	engine := New(r)
	artifacts := map[string]*Artifact{}
	connections := map[string]collection.ApplicationAttempt{}
	for _, plugin := range []string{"theme", "dms"} {
		root := "/artifacts/plugins/" + plugin
		var files []string
		if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.IsDir() {
				name, _ := filepath.Rel(root, path)
				if artifactPath(name) {
					files = append(files, name)
				}
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		entry := "application_connection.py"
		if plugin == "theme" {
			entry = "client/" + entry
		}
		artifact, err := OpenReceiver(root, entry, files, env)
		if err != nil {
			t.Fatal("open artifact", plugin, err)
		}
		artifact, err = artifact.Retain(ctx, "/lab/receiver-generations")
		if err != nil {
			t.Fatal("retain artifact", plugin, err)
		}
		// This is a fixture approval for the disposable ship and staged lab IDs.
		a, err := r.ApproveApplication(ctx, ship, collection.NewSessionRef(), artifact.Target())
		if err != nil {
			t.Fatal(err)
		}
		lost := &lostLiveReply{Artifact: artifact}
		got, err := engine.Connect(ctx, a.ID, lost)
		if !errors.Is(err, ErrDelivery) || got.Phase != "closed" || !got.ReceiverReconciliation {
			t.Fatal("lost-ack cleanup", plugin, got.Phase, err)
		}
		replay, _ := eyre.New(ship.Origin, ship.Ship)
		if err := replay.RestoreSession(lost.offered); err != nil {
			t.Fatal(err)
		}
		if err := replay.Check(ctx); !errors.Is(err, eyre.ErrAuth) {
			t.Fatal("receiver's session was not remotely revoked", err)
		}
		replay.Close()
		r, err = collection.Open(registryDir)
		if err != nil {
			t.Fatal(err)
		}
		engine = New(r)
		artifact, err = LoadReceiver("/lab/receiver-generations", artifact.Target().SHA256, env)
		if err != nil {
			t.Fatal(err)
		}
		got, err = engine.Reconcile(ctx, a.ID, artifact)
		if err != nil || got.ReceiverReconciliation || got.ReceiverOutcome != "removed" {
			t.Fatal("real receiver reconciliation", plugin, got.ReceiverOutcome, err)
		}
		if _, err := engine.Reconcile(ctx, a.ID, artifact); err != nil {
			t.Fatal("repeat reconciliation", err)
		}
		a, err = r.ApproveApplication(ctx, ship, collection.NewSessionRef(), artifact.Target())
		if err != nil {
			t.Fatal(err)
		}
		got, err = engine.Connect(ctx, a.ID, artifact)
		if err != nil || got.Phase != "connected" {
			t.Fatal("new approval after recovery", plugin, err)
		}
		artifacts[plugin], connections[plugin] = artifact, got
	}
	owner.Close()
	probe := func(plugin string) {
		cmd := exec.CommandContext(ctx, "/usr/bin/python3", "-B", "/artifacts/plugin_probe.py", plugin, "read")
		cmd.Env = artifacts[plugin].environment
		out, err := cmd.Output()
		defer clear(out)
		var result struct {
			OK bool `json:"ok"`
		}
		if err != nil || len(out) > 4096 || json.Unmarshal(out, &result) != nil || !result.OK {
			t.Fatal("direct receiver probe", plugin, err)
		}
	}
	probe("theme")
	probe("dms")
	for _, plugin := range []string{"theme", "dms"} {
		if _, err := engine.Disconnect(ctx, connections[plugin].ID); err != nil {
			t.Fatal("disconnect", plugin, err)
		}
		if _, err := engine.Reconcile(ctx, connections[plugin].ID, artifacts[plugin]); err != nil {
			t.Fatal("reconcile", plugin, err)
		}
		if plugin == "theme" {
			probe("dms")
		}
	}
	owner, _ = eyre.New(ship.Origin, ship.Ship)
	defer owner.Close()
	if err := owner.RestoreSession(append([]byte(nil), ownerData...)); err != nil {
		t.Fatal(err)
	}
	if err := owner.Check(ctx); err != nil {
		t.Fatal("parent login was affected", err)
	}
	t.Log("PASS: retained verified Python artifacts; real private keyring; lost-ack remote revocation and account reconciliation after issuer reopen; fresh approvals and direct plugin reads/SSE; independent disconnect")
}

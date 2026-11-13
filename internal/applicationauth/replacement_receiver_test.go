package applicationauth

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestReplacementCapabilityBelongsToTheExactApprovedGeneration(t *testing.T) {
	root, path, env := installedFixture(t)
	old, _, err := ResolveInstalled(root, "fixture.auth-lab", env)
	if err != nil || old.SupportsReplacement() {
		t.Fatal("legacy helper gained replacement", err)
	}
	for _, call := range []func(context.Context, []byte) ([]byte, error){old.PrepareReplacement, old.Replace, old.ReplacementStatus} {
		if _, err := call(context.Background(), []byte(`{}`)); !errors.Is(err, ErrArtifact) {
			t.Fatal("legacy helper was executed", err)
		}
	}
	cache := filepath.Join(t.TempDir(), "generations")
	retained, err := old.Retain(context.Background(), cache)
	if err != nil {
		t.Fatal(err)
	}
	declaration, _ := json.Marshal(Declaration{2, "python3", OwnerAuthority, "receiver.py", []string{"manifest.json", DeclarationName, "receiver.py"}})
	if err := os.WriteFile(filepath.Join(path, DeclarationName), declaration, 0600); err != nil {
		t.Fatal(err)
	}
	updated, _, err := ResolveInstalled(root, "fixture.auth-lab", env)
	if err != nil || !updated.SupportsReplacement() || updated.Target() == retained.Target() || retained.SupportsReplacement() {
		t.Fatal("installed update widened previous grant", err)
	}
	reloaded, err := LoadReceiver(cache, retained.Target().SHA256, env)
	if err != nil || reloaded.SupportsReplacement() {
		t.Fatal("cached legacy capability changed", err)
	}
	retainedNew, err := updated.Retain(context.Background(), cache)
	if err != nil || !retainedNew.SupportsReplacement() {
		t.Fatal("approved replacement capability was not retained", err)
	}
}

func TestReplacementPrivateModesUseBoundedPinnedLauncher(t *testing.T) {
	root, path, env := installedFixture(t)
	d := Declaration{2, "python3", OwnerAuthority, "receiver.py", []string{"manifest.json", DeclarationName, "receiver.py"}}
	data, _ := json.Marshal(d)
	if err := os.WriteFile(filepath.Join(path, DeclarationName), data, 0600); err != nil {
		t.Fatal(err)
	}
	code := "import json,sys; value=json.load(sys.stdin); print(json.dumps({'mode':sys.argv[1:],'request_id':value['request_id']}))"
	if err := os.WriteFile(filepath.Join(path, "receiver.py"), []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	a, _, err := ResolveInstalled(root, "fixture.auth-lab", env)
	if err != nil {
		t.Fatal(err)
	}
	for mode, call := range map[string]func(context.Context, []byte) ([]byte, error){
		"--prepare-replacement": a.PrepareReplacement, "--replace": a.Replace, "--replacement-status": a.ReplacementStatus,
	} {
		reply, err := call(context.Background(), []byte(`{"request_id":"synthetic-bound-id"}`))
		var value struct {
			Mode []string `json:"mode"`
			ID   string   `json:"request_id"`
		}
		if err != nil || json.Unmarshal(reply, &value) != nil || len(value.Mode) != 1 || value.Mode[0] != mode || value.ID != "synthetic-bound-id" {
			t.Fatal(mode, err)
		}
	}
	if _, err := a.run(context.Background(), nil, "--unapproved-command"); !errors.Is(err, ErrArtifact) {
		t.Fatal("arbitrary mode accepted", err)
	}
	if err := os.WriteFile(filepath.Join(path, "receiver.py"), []byte("raise Exception('changed')"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Replace(context.Background(), nil); !errors.Is(err, ErrArtifact) {
		t.Fatal("mutated artifact invoked", err)
	}
}

func TestUnknownReplacementDeclarationsAreNotCapabilities(t *testing.T) {
	for _, version := range []int{0, 3, -1} {
		root, path, env := installedFixture(t)
		d := Declaration{version, "python3", OwnerAuthority, "receiver.py", []string{"manifest.json", DeclarationName, "receiver.py"}}
		data, _ := json.Marshal(d)
		if err := os.WriteFile(filepath.Join(path, DeclarationName), data, 0600); err != nil {
			t.Fatal(err)
		}
		if _, _, err := ResolveInstalled(root, "fixture.auth-lab", env); !errors.Is(err, ErrArtifact) {
			t.Fatal("unknown capability accepted", version, err)
		}
	}
}

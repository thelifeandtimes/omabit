package main

import (
	"context"
	"path/filepath"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
)

func TestAppDevelopmentHelperOfflineProfileAndValidation(t *testing.T) {
	ctx := context.Background()
	config := filepath.Join(t.TempDir(), "collector")
	call := func(command string, args ...string) int {
		return run(ctx, append([]string{"--config-dir", config, command}, args...))
	}
	if call("app-development-status") != 0 {
		t.Fatal("fresh profile status failed")
	}
	for _, raw := range []string{`{}`, `{"update":{"scope":"global","gates":{"commit":"inherit","publish":"allow"}}}`, `{"unknown":true}`} {
		if call("app-development-save", "--request-json", raw) == 0 {
			t.Fatal("invalid request admitted", raw)
		}
	}
	request := `{"revision":0,"update":{"scope":"global","target":{},"gates":{"commit":"allow","publish":"block"}},"runners":[]}`
	if call("app-development-save", "--request-json", request) != 0 {
		t.Fatal("global authoring failed")
	}
	r, err := collection.Open(config)
	if err != nil {
		t.Fatal(err)
	}
	v, err := r.AppDevelopment(ctx)
	if err != nil || v.Global.Revision != 1 || v.Global.Gates.Commit != apppolicy.Allow {
		t.Fatal(v, err)
	}
	if call("app-development-save", "--request-json", request) == 0 {
		t.Fatal("stale local review accepted")
	}
	other := filepath.Join(t.TempDir(), "collector")
	r, err = collection.Open(other)
	if err != nil {
		t.Fatal(err)
	}
	v, err = r.AppDevelopment(ctx)
	if err != nil || v.Global.Gates != apppolicy.Blocked() {
		t.Fatal("policy crossed profiles", v, err)
	}
}

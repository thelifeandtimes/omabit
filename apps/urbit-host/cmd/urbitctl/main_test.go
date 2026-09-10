package main

import (
	"os"
	"path/filepath"
	"testing"
)

func cliEnv(t *testing.T) {
	t.Helper()
	r := t.TempDir()
	for _, v := range []string{"XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME", "XDG_RUNTIME_DIR"} {
		p := filepath.Join(r, v)
		os.MkdirAll(p, 0700)
		t.Setenv(v, p)
	}
}
func TestCLIConfiguration(t *testing.T) {
	cliEnv(t)
	if e := run([]string{"host", "init", "--name", "workstation"}); e != nil {
		t.Fatal(e)
	}
	if e := run([]string{"host", "init", "--name", "overwrite"}); e == nil {
		t.Fatal("overwrote host")
	}
	if e := run([]string{"host", "add", "--ssh", "workstation-tailnet", "--default", "workstation"}); e != nil {
		t.Fatal(e)
	}
	if e := run([]string{"host", "list"}); e != nil {
		t.Fatal(e)
	}
}
func TestCLIRejectsAmbiguousOrSecretInput(t *testing.T) {
	cliEnv(t)
	for _, a := range [][]string{{"ship", "code", "some-ship"}, {"ship", "boot", "--label", "x"}, {"ship", "boot", "--comet", "--key-stdin", "--label", "x"}, {"ship", "exec", "x"}, {"host", "add", "--ssh", "host;bad", "x"}, {"dev", "create", "label", "--group", "demo"}} {
		if e := run(a); e == nil {
			t.Fatalf("accepted %v", a)
		}
	}
}

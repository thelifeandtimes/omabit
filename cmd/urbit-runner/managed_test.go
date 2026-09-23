package main

import (
	"bytes"
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestManagedUsage(t *testing.T) {
	for _, tc := range []struct {
		args []string
		code int
	}{
		{[]string{"--help"}, 0}, {nil, 2}, {[]string{"--acknowledge-development", "extra"}, 2},
	} {
		var stdout, stderr bytes.Buffer
		if code := runManaged(context.Background(), tc.args, &stdout, &stderr); code != tc.code {
			t.Fatalf("%v: exit %d, %s", tc.args, code, &stderr)
		}
	}
}

func TestManagedStaleSocketHandling(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "runner.sock")
	if err := removeStaleManagedSocket(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := removeStaleManagedSocket(path); err == nil {
		t.Fatal("regular file removed")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	l.SetUnlinkOnClose(false)
	if err := os.Chmod(path, os.ModeSticky|0600); err != nil {
		t.Fatal(err)
	}
	if err := removeStaleManagedSocket(path); err == nil {
		t.Fatal("live listener replaced")
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if err := removeStaleManagedSocket(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatal("stale socket retained")
	}
}

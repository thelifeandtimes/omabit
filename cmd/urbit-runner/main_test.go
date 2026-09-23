package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestUsage(t *testing.T) {
	for _, tc := range []struct {
		args []string
		code int
	}{
		{nil, 2}, {[]string{"--help"}, 0}, {[]string{"demo", "--help"}, 0},
		{[]string{"start"}, 2}, {[]string{"demo"}, 2},
		{[]string{"demo", "--socket", "/tmp/unused", "--operation-delay", "0s"}, 2},
		{[]string{"demo", "--socket", "/tmp/unused", "extra"}, 2},
		{[]string{"demo", "--socket", "relative"}, 1},
	} {
		var output bytes.Buffer
		if code := run(context.Background(), tc.args, &output); code != tc.code {
			t.Fatalf("%v: exit %d, want %d; %s", tc.args, code, tc.code, &output)
		}
	}
}

func TestServeAndShutdown(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(dir, "runner.sock")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var output bytes.Buffer
	exit := make(chan int, 1)
	go func() { exit <- run(ctx, []string{"demo", "--socket", socket}, &output) }()
	c := client.New(socket)
	defer c.Close()
	deadline := time.Now().Add(3 * time.Second)
	for {
		hello, err := c.Hello(ctx)
		if err == nil {
			if hello.Mode != "simulation" {
				t.Fatalf("wrong mode %q", hello.Mode)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("runner did not start: %v", err)
		}
		time.Sleep(5 * time.Millisecond)
	}
	if _, err := c.Operate(ctx, "alpha", protocol.OperationRequest{
		RequestID: "shutdown-in-flight", Kind: "start", ExpectedRevision: 1,
	}); err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case code := <-exit:
		if code != 0 {
			t.Fatalf("exit %d: %s", code, &output)
		}
	case <-time.After(6 * time.Second):
		t.Fatal("runner did not shut down")
	}
	if _, err := os.Lstat(socket); !os.IsNotExist(err) {
		t.Fatalf("socket remains after shutdown: %v", err)
	}
}

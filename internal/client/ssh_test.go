package client

import (
	"context"
	"errors"
	"net"
	"os/exec"
	"slices"
	"testing"
	"time"
)

func TestCanceledSSHHandshakeKillsAndReapsChild(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan *sshConn, 1)
	c := newTransport(func(ctx context.Context) (net.Conn, error) {
		// A real child with open pipes that never answers the identity handshake.
		conn, err := startSSH(ctx, exec.Command("sleep", "60"))
		if err == nil {
			started <- conn.(*sshConn)
		}
		return conn, err
	}, &Identity{RunnerID: "runner-logs", Mode: "development-managed"})
	defer c.Close()
	done := make(chan error, 1)
	go func() { _, err := c.RuntimeLogs(ctx, "runner-logs", "alpha", 1); done <- err }()
	var child *sshConn
	select {
	case child = <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("child did not start")
	}
	defer child.Close()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("canceled request blocked on handshake")
	}
	c.Close() // A detached HTTP dial must be canceled as well.
	select {
	case <-child.done:
		if child.cmd.ProcessState == nil {
			t.Fatal("child was not reaped")
		}
	case <-time.After(time.Second):
		t.Fatal("pending SSH child survived closing the watcher")
	}
}

func TestSSHRejectsCommandInjectionAndUnsafeOverrides(t *testing.T) {
	for _, host := range []string{"", "-oProxyCommand=id", "user@host", "host;id", "host\n", "host name", "$(id)"} {
		if err := (SSHConfig{Host: host}).Validate(); err == nil {
			t.Fatalf("unsafe SSH target accepted: %q", host)
		}
	}
	args := sshArguments(SSHConfig{Host: "runner.example", ConfigFile: "/private/ssh.conf"})
	for _, option := range []string{"-oBatchMode=yes", "-oStrictHostKeyChecking=yes", "-oControlPath=none", "-oForwardAgent=no", "-oClearAllForwardings=yes", "-oPermitLocalCommand=no", "-oRemoteCommand=none"} {
		if !slices.Contains(args, option) {
			t.Fatalf("missing invariant %s", option)
		}
	}
	if args[len(args)-1] != SSHCommand || args[len(args)-2] != "runner.example" {
		t.Fatal("remote command is not fixed")
	}
}

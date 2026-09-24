package managed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/localrpc"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// This opt-in test runs the installed OpenSSH binaries with ephemeral keys and a
// loopback-only, unprivileged sshd. It changes no host SSH configuration or account.
func TestOpenSSHIntegration(t *testing.T) {
	if os.Getenv("OMABIT_TEST_OPENSSH") != "1" {
		t.Skip("set OMABIT_TEST_OPENSSH=1 and OMABIT_RUNNER_BINARY to a built runner")
	}
	binary := os.Getenv("OMABIT_RUNNER_BINARY")
	if !filepath.IsAbs(binary) {
		t.Fatal("OMABIT_RUNNER_BINARY must be absolute")
	}
	// OpenSSH StrictModes checks every ancestor of authorized_keys, including
	// /tmp. Keep this fixture beneath the owned build directory instead.
	dir, err := os.MkdirTemp(filepath.Dir(binary), ".ssh-fixture-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	write := func(path, data string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	generate := func(name string) string {
		t.Helper()
		path := filepath.Join(dir, name)
		if out, err := exec.Command("/usr/bin/ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", path).CombinedOutput(); err != nil {
			t.Fatalf("keygen: %s %v", out, err)
		}
		pub, err := os.ReadFile(path + ".pub")
		if err != nil {
			t.Fatal(err)
		}
		return string(pub)
	}
	hostPub := generate("host")
	keys, auth := filepath.Join(dir, "grants"), filepath.Join(dir, "authorized_keys")
	socket := filepath.Join(dir, "rpc.sock")
	var grants, authorized strings.Builder
	ids, lines := map[string]string{}, map[string]string{}
	for _, name := range []string{"a", "b", "observer", "untrusted"} {
		pub := generate(name)
		if name == "untrusted" {
			continue
		}
		role := "operator"
		if name == "observer" {
			role = "observer"
		}
		line, err := access.AuthorizedKey([]byte(pub), role)
		if err != nil {
			t.Fatal(err)
		}
		grant, err := access.ParseLine(line)
		if err != nil {
			t.Fatal(err)
		}
		ids[name], lines[name] = grant.Fingerprint, line
		fmt.Fprintln(&grants, line)
		fields := strings.Fields(pub)
		fmt.Fprintf(&authorized, "restrict,command=\"%s ssh-bridge --socket %s --authorized-keys %s --fingerprint %s\" %s %s\n", binary, socket, keys, grant.Fingerprint, fields[0], fields[1])
	}
	write(keys, grants.String())
	write(auth, authorized.String())
	r, err := newServer(dir, &fakeBackend{}, 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	r.keys = keys
	t.Cleanup(func() { r.Close() })
	ln, err := localrpc.Listen(socket)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: r, ConnContext: PeerContext, ReadHeaderTimeout: 5 * time.Second}
	served := make(chan error, 1)
	go func() { served <- srv.Serve(ln) }()
	t.Cleanup(func() { srv.Close(); <-served })
	portListener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := portListener.Addr().(*net.TCPAddr).Port
	portListener.Close()
	u, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(dir, "sshd_config")
	write(config, fmt.Sprintf("Port %d\nListenAddress 127.0.0.1\nHostKey %s/host\nPidFile %s/sshd.pid\nAuthorizedKeysFile %s\nUsePAM no\nPasswordAuthentication no\nKbdInteractiveAuthentication no\nAuthenticationMethods publickey\nPermitRootLogin no\nAllowUsers %s\nDisableForwarding yes\nPermitTTY no\nPermitUserEnvironment no\nLogLevel DEBUG1\n", port, dir, dir, auth, u.Username))
	if out, err := exec.Command("/usr/bin/sshd", "-t", "-f", config).CombinedOutput(); err != nil {
		t.Fatalf("sshd config: %s %v", out, err)
	}
	log, err := os.Create(filepath.Join(dir, "sshd.log"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { log.Close() })
	daemon := exec.Command("/usr/bin/sshd", "-D", "-e", "-f", config)
	daemon.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + u.HomeDir}
	daemon.Stderr = log
	if err := daemon.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { daemon.Process.Kill(); daemon.Wait() })
	for i := 0; i < 100; i++ {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:"+strconv.Itoa(port), 50*time.Millisecond)
		if err == nil {
			conn.Close()
			break
		}
		if i == 99 {
			data, _ := os.ReadFile(log.Name())
			t.Fatalf("sshd not ready: %s", data)
		}
		time.Sleep(20 * time.Millisecond)
	}
	known := filepath.Join(dir, "known_hosts")
	hostFields := strings.Fields(hostPub)
	knownData := fmt.Sprintf("[127.0.0.1]:%d %s %s\n", port, hostFields[0], hostFields[1])
	write(known, knownData)
	clientConfig := filepath.Join(dir, "ssh_config")
	var cfg strings.Builder
	for _, name := range []string{"a", "b", "observer", "untrusted"} {
		fmt.Fprintf(&cfg, "Host %s\n  HostName 127.0.0.1\n  Port %d\n  User %s\n  IdentityFile %s/%s\n  IdentitiesOnly yes\n  UserKnownHostsFile %s\n  GlobalKnownHostsFile /dev/null\n", name, port, u.Username, dir, name, known)
	}
	write(clientConfig, cfg.String())
	connect := func(name string, pin *client.Identity) *client.Client {
		c := client.NewSSH(client.SSHConfig{Host: name, ConfigFile: clientConfig}, pin)
		t.Cleanup(c.Close)
		return c
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	a := connect("a", nil)
	hello, err := a.Hello(ctx)
	if err != nil {
		data, _ := os.ReadFile(log.Name())
		probe := exec.CommandContext(ctx, "/usr/bin/ssh", "-vv", "-F", clientConfig, "-oBatchMode=yes", "-oStrictHostKeyChecking=yes", "a", client.SSHCommand)
		probe.Stdin = strings.NewReader("GET /hello HTTP/1.1\r\nHost: runner\r\nConnection: close\r\n\r\n")
		diagnostic, _ := probe.CombinedOutput()
		t.Fatalf("SSH hello: %v; sshd: %s; fixture SSH trace: %s", err, data, diagnostic)
	}
	if hello.Actor != "ssh-key:"+ids["a"] || hello.AccessRole != "operator" {
		t.Fatalf("incorrect authenticated actor: %+v", hello)
	}
	// Exercise the built collector's SSH enrollment and readable named selection.
	collector := filepath.Join(filepath.Dir(binary), "urbit-collector")
	profile := filepath.Join(dir, "collector")
	enrollment, err := exec.CommandContext(ctx, collector, "--config-dir", profile, "enroll", "remote", "--ssh", "a", "--ssh-config", clientConfig).CombinedOutput()
	if err != nil {
		t.Fatalf("collector enrollment: %s %v", enrollment, err)
	}
	var saved struct{ Transport, RunnerID string }
	var fields map[string]any
	if err := json.Unmarshal(enrollment, &fields); err != nil {
		t.Fatal(err)
	}
	saved.Transport, _ = fields["transport"].(string)
	saved.RunnerID, _ = fields["runner_id"].(string)
	if saved.Transport != "ssh" || saved.RunnerID != hello.RunnerID {
		t.Fatalf("bad enrollment: %s", enrollment)
	}
	readable, err := exec.CommandContext(ctx, collector, "--config-dir", profile, "--format", "text", "--runner", "remote", "hello").CombinedOutput()
	if err != nil || !strings.Contains(string(readable), "Actor: ssh-key:"+ids["a"]) {
		t.Fatalf("named SSH selection: %s %v", readable, err)
	}
	b, observer := connect("b", nil), connect("observer", nil)
	reg, err := a.CreateFake(ctx, registration("alpha", "shared-id", 18080))
	if err != nil || reg.Actor != hello.Actor {
		t.Fatalf("register A: %+v %v", reg, err)
	}
	second, err := b.CreateFake(ctx, registration("bravo", "shared-id", 18082))
	if err != nil || second.ID == reg.ID || second.Actor != "ssh-key:"+ids["b"] {
		t.Fatalf("actor-scoped requests: %+v %v", second, err)
	}
	if _, err := observer.Snapshot(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := observer.CreateFake(ctx, registration("blocked", "blocked", 18083)); err == nil {
		t.Fatal("observer mutation succeeded")
	}
	if _, err := connect("untrusted", nil).Hello(ctx); err == nil {
		t.Fatal("untrusted key authenticated")
	}
	conf := protocol.ConfigureRequest{ExpectedRevision: 1, Config: protocol.Configuration{HTTPPort: 18081, AmesPort: 34343, Loom: 31}}
	if _, err := a.Configure(ctx, "alpha", conf); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Configure(ctx, "alpha", conf); err == nil {
		t.Fatal("cross-key stale write accepted")
	}
	op, err := a.Operate(ctx, "alpha", protocol.OperationRequest{RequestID: "start-over-ssh", Kind: "start", ExpectedRevision: 2})
	if err != nil {
		t.Fatal(err)
	}
	a.Close()
	waitOperation(t, b, op.ID, "succeeded")
	a = connect("a", &client.Identity{RunnerID: hello.RunnerID, Mode: hello.Mode})
	if replay, err := a.Operate(ctx, "alpha", protocol.OperationRequest{RequestID: "start-over-ssh", Kind: "start", ExpectedRevision: 2}); err != nil || replay.ID != op.ID {
		t.Fatalf("reconnect/replay: %+v %v", replay, err)
	}
	if _, err := connect("b", &client.Identity{RunnerID: "wrong", Mode: hello.Mode}).Hello(ctx); !errors.Is(err, client.ErrIdentityMismatch) {
		t.Fatalf("identity pin: %v", err)
	}
	wrong := strings.Fields(generate("wrong-host"))
	write(known, fmt.Sprintf("[127.0.0.1]:%d %s %s\n", port, wrong[0], wrong[1]))
	if _, err := connect("b", nil).Hello(ctx); err == nil {
		t.Fatal("changed SSH host key accepted")
	}
	write(known, knownData)
	for _, extra := range [][]string{{"b", "id"}, {"-W", "127.0.0.1:" + strconv.Itoa(port), "b"}} {
		args := append([]string{"-F", clientConfig, "-oBatchMode=yes", "-oStrictHostKeyChecking=yes"}, extra...)
		if err := exec.CommandContext(ctx, "/usr/bin/ssh", args...).Run(); err == nil {
			t.Fatal("arbitrary command or forwarding allowed")
		}
	}
	// Removing a grant blocks existing sessions as well as new logins.
	write(keys, lines["b"]+"\n"+lines["observer"]+"\n")
	if _, err := a.Snapshot(ctx); err == nil {
		t.Fatal("revoked existing session still authorized")
	}
	if _, err := b.Snapshot(ctx); err != nil {
		t.Fatal("revocation affected independent operator")
	}
	t.Log("PASS: actual OpenSSH, two operator keys, observer restrictions, scoped replay, client disconnect, reconnect, host-key and runner-ID mismatch, forwarding/command refusal, live revocation")
}

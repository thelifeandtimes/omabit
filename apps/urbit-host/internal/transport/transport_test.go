package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"omarchy-urbit/internal/fleet"
)

func paths(t *testing.T) fleet.Paths {
	t.Helper()
	r := t.TempDir()
	p := fleet.Paths{Config: filepath.Join(r, "c"), Data: filepath.Join(r, "d"), State: filepath.Join(r, "s"), Runtime: filepath.Join(r, "r")}
	if e := p.Ensure(); e != nil {
		t.Fatal(e)
	}
	return p
}
func TestProtocolLimitsAndNoClientRole(t *testing.T) {
	for _, s := range []string{`{"schema":1,"method":"doctor","role":"owner"}` + "\n", `{"schema":1} {"schema":1}` + "\n", strings.Repeat("x", MaxRequest+1)} {
		if _, e := ReadRequest(strings.NewReader(s)); e == nil {
			t.Fatal("accepted invalid protocol")
		}
	}
	q, e := ReadRequest(strings.NewReader(`{"schema":1,"method":"doctor"}` + "\n"))
	if e != nil || q.Method != "doctor" {
		t.Fatal(q, e)
	}
}
func TestSSHCommandDoesNotInterpolatePayload(t *testing.T) {
	for _, s := range []string{"-oProxyCommand=bad", "host;touch /tmp/x", "$(id)", "a b", "user@host\n"} {
		if e := ValidateSSH(s); e == nil {
			t.Fatal(s)
		}
	}
	for _, s := range []string{"workstation", "me@100.90.0.1", "me@host.tailnet.ts.net"} {
		if e := ValidateSSH(s); e != nil {
			t.Fatal(e)
		}
	}
	a := SSHArgs("workstation", true)
	s := strings.Join(a, " ")
	if !strings.Contains(s, "ForwardAgent=no") || !strings.Contains(s, "rpc --agent") || strings.Contains(s, "StrictHostKeyChecking=no") {
		t.Fatal(s)
	}
}

type dummy struct{}

func (dummy) Run(context.Context, []string, []byte) ([]byte, error) {
	return []byte(`"test-version"`), nil
}
func serve(t *testing.T, p fleet.Paths) *fleet.Manager {
	t.Helper()
	h := fleet.HostConfig{Schema: 1, ID: fleet.ID(), Name: "workstation", DataRoot: p.Data}
	m, e := fleet.NewManager(&fleet.Engine{Exec: dummy{}, Paths: p, Host: h})
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, p, m) }()
	t.Cleanup(func() {
		cancel()
		select {
		case e := <-done:
			if e != nil {
				t.Error(e)
			}
		case <-time.After(5 * time.Second):
			t.Error("service did not stop")
		}
	})
	for n := 0; n < 100; n++ {
		if _, e := os.Stat(Socket(p, true)); e == nil {
			return m
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("no service sockets")
	return nil
}
func TestUnixRPCFixedRoles(t *testing.T) {
	p := paths(t)
	serve(t, p)
	ctx := context.Background()
	v, e := Local(ctx, p, false, fleet.Request{Schema: 1, Method: "doctor"})
	if e != nil || !v.OK {
		t.Fatal(v, e)
	}
	v, e = Local(ctx, p, true, fleet.Request{Schema: 1, Method: "notices.dismiss", Target: "x"})
	if e != nil || v.OK || v.Error.Code != "forbidden" {
		t.Fatal(v, e)
	}
	v, e = Local(ctx, p, true, fleet.Request{Schema: 1, Method: "group.create", Label: "demo"})
	if e != nil || !v.OK {
		t.Fatal(v, e)
	}
	st, _ := os.Stat(Socket(p, false))
	if st.Mode().Perm() != 0600 {
		t.Fatal(st.Mode())
	}
}
func TestGateway(t *testing.T) {
	p := paths(t)
	serve(t, p)
	var out strings.Builder
	e := Gateway(context.Background(), p, true, strings.NewReader(`{"schema":1,"method":"doctor"}`+"\n"), &out)
	if e != nil {
		t.Fatal(e)
	}
	var r fleet.Response
	if e = json.Unmarshal([]byte(out.String()), &r); e != nil || !r.OK {
		t.Fatal(out.String(), e)
	}
}
func TestInventoryRetainsStaleSnapshot(t *testing.T) {
	p := paths(t)
	snapshot := json.RawMessage(`{"host_id":"old","instances":[{"container_status":"running"}]}`)
	if e := fleet.AtomicJSON(filepath.Join(p.State, "inventory", "local.json"), snapshot); e != nil {
		t.Fatal(e)
	}
	a, e := Inventory(context.Background(), p, false)
	if e != nil || len(a) != 1 {
		t.Fatal(a, e)
	}
	if a[0].Reachable || !a[0].Stale || !strings.Contains(string(a[0].Snapshot), "running") {
		t.Fatal(a)
	}
}
func TestLocalProxyRawStreamsAndShutdown(t *testing.T) {
	p := paths(t)
	up, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer up.Close()
	go func() {
		for {
			c, e := up.Accept()
			if e != nil {
				return
			}
			go func() { defer c.Close(); b := make([]byte, 4); io.ReadFull(c, b); c.Write([]byte("pong")) }()
		}
	}()
	reserve, e := net.Listen("tcp", "127.0.0.2:0")
	if e != nil {
		t.Fatal(e)
	}
	addr := reserve.Addr().String()
	reserve.Close()
	ctl := filepath.Join(p.Runtime, "proxy.sock")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- Proxy(ctx, addr, up.Addr().String(), ctl) }()
	for n := 0; n < 100; n++ {
		if _, e := os.Stat(ctl); e == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	tunnel := Tunnel{Listen: addr, Control: ctl}
	if !tunnelCheck(ctx, tunnel) {
		t.Fatal("proxy control not healthy")
	}
	c, e := net.DialTimeout("tcp", addr, time.Second)
	if e != nil {
		t.Fatal(e)
	}
	c.SetDeadline(time.Now().Add(time.Second))
	fmt.Fprint(c, "ping")
	b := make([]byte, 4)
	_, e = io.ReadFull(c, b)
	c.Close()
	if e != nil || string(b) != "pong" {
		t.Fatal(string(b), e)
	}
	if e = CloseTunnel(ctx, tunnel); e != nil {
		t.Fatal(e)
	}
	select {
	case e := <-done:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("proxy did not stop")
	}
}
func TestProxyRejectsPublicBind(t *testing.T) {
	e := Proxy(context.Background(), "0.0.0.0:12345", "127.0.0.1:80", filepath.Join(t.TempDir(), "sock"))
	if e == nil {
		t.Fatal("public proxy")
	}
}
func TestDistinctInstanceTunnelKeys(t *testing.T) {
	p := paths(t)
	c := Client{Paths: p, Host: Host{Name: "workstation", SSH: "workstation"}}
	if c.tunnelPath("zod-one") == c.tunnelPath("zod-two") {
		t.Fatal("conflated instances")
	}
	other := c
	other.Host.Name = "laptop"
	if c.tunnelPath("zod-one") == other.tunnelPath("zod-one") {
		t.Fatal("conflated hosts")
	}
}

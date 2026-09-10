package transport

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"omarchy-urbit/internal/fleet"
)

type Tunnel struct {
	Instance string `json:"instance_id"`
	Host     string `json:"host"`
	SSH      string `json:"ssh,omitempty"`
	Listen   string `json:"listen"`
	Target   string `json:"target"`
	Control  string `json:"control"`
	URL      string `json:"url"`
	Record   string `json:"-"`
}

func (c Client) tunnelPath(id string) string {
	h := sha256.Sum256([]byte(c.Host.Name + "\x00" + id))
	return filepath.Join(c.Paths.Runtime, "tunnels", hex.EncodeToString(h[:8])+".json")
}
func (c Client) TunnelInfo(id string) (Tunnel, error) {
	var t Tunnel
	path := c.tunnelPath(id)
	e := fleet.ReadJSON(path, &t)
	t.Record = path
	if e == nil && (t.Instance != id || t.Host != c.Host.Name || t.SSH != c.Host.SSH) {
		e = fmt.Errorf("tunnel identity mismatch")
	}
	return t, e
}
func tunnelCheck(ctx context.Context, t Tunnel) bool {
	if t.SSH != "" {
		return exec.CommandContext(ctx, "ssh", "-S", t.Control, "-O", "check", t.SSH).Run() == nil
	}
	conn, e := net.DialTimeout("unix", t.Control, time.Second)
	if e != nil {
		return false
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(time.Second))
	io.WriteString(conn, "ping\n")
	b := make([]byte, 5)
	n, e := conn.Read(b)
	return e == nil && string(b[:n]) == "pong\n"
}
func (c Client) EnsureTunnel(ctx context.Context, o fleet.Observation) (Tunnel, error) {
	if c.Agent {
		return Tunnel{}, fmt.Errorf("interactive access uses an operator SSH identity; agent Hoon uses ship exec")
	}
	if o.Endpoint != "http-responsive" || o.HostPort < 1 || o.HostPort > 65535 {
		return Tunnel{}, fmt.Errorf("ship HTTP endpoint is not available")
	}
	dir := filepath.Join(c.Paths.Runtime, "tunnels")
	if e := os.MkdirAll(dir, 0700); e != nil {
		return Tunnel{}, e
	}
	lock, e := fleet.AcquireLock(filepath.Join(dir, "access.lock"))
	if e != nil {
		return Tunnel{}, e
	}
	defer lock.Close()
	t, e := c.TunnelInfo(o.Instance.ID)
	target := "127.0.0.1:" + strconv.Itoa(o.HostPort)
	if e == nil && t.Target == target && tunnelCheck(ctx, t) {
		return t, nil
	}
	if e == nil && tunnelCheck(ctx, t) {
		return t, fmt.Errorf("existing tunnel has a changed endpoint; close it first")
	}
	h := sha256.Sum256([]byte(c.Host.Name + "\x00" + o.Instance.ID))
	ip := fmt.Sprintf("127.%d.%d.%d", 1+int(h[0])%254, h[1], 1+int(h[2])%254)
	addr := net.JoinHostPort(ip, "8080")
	probe, e := net.Listen("tcp", addr)
	if e != nil {
		return t, fmt.Errorf("isolated browser address occupied; refusing to reuse an unknown listener")
	}
	probe.Close()
	path := c.tunnelPath(o.Instance.ID)
	t = Tunnel{Instance: o.Instance.ID, Host: c.Host.Name, SSH: c.Host.SSH, Listen: addr, Target: target, Control: strings.TrimSuffix(path, ".json") + ".sock", URL: "http://" + addr + "/", Record: path}
	if st, e := os.Lstat(t.Control); e == nil {
		if st.Mode()&os.ModeSocket == 0 {
			return t, fmt.Errorf("unexpected tunnel control file")
		}
		if e = os.Remove(t.Control); e != nil {
			return t, e
		}
	}
	if c.Host.SSH != "" {
		a := []string{"-M", "-S", t.Control, "-fNT", "-o", "BatchMode=yes", "-o", "ConnectTimeout=10", "-o", "ForwardAgent=no", "-o", "ExitOnForwardFailure=yes", "-o", "ServerAliveInterval=30", "-o", "ServerAliveCountMax=3", "-L", addr + ":" + target, c.Host.SSH}
		if e = exec.CommandContext(ctx, "ssh", a...).Run(); e != nil {
			return t, fmt.Errorf("could not establish private SSH tunnel (%v)", e)
		}
	} else {
		exe, e := os.Executable()
		if e != nil {
			return t, e
		}
		cmd := exec.Command(exe, "proxy", "--listen", addr, "--target", target, "--control", t.Control)
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		log, e := os.OpenFile(strings.TrimSuffix(path, ".json")+".log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if e != nil {
			return t, e
		}
		defer log.Close()
		cmd.Stdout = log
		cmd.Stderr = log
		if e = cmd.Start(); e != nil {
			return t, e
		}
		cmd.Process.Release()
		ready := false
		for n := 0; n < 40; n++ {
			if tunnelCheck(ctx, t) {
				ready = true
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		if !ready {
			return t, fmt.Errorf("local browser proxy did not start; inspect its runtime log")
		}
	}
	if e = fleet.AtomicJSON(path, t); e != nil {
		CloseTunnel(ctx, t)
		return t, e
	}
	return t, nil
}
func CloseTunnel(ctx context.Context, t Tunnel) error {
	if t.SSH != "" {
		if e := exec.CommandContext(ctx, "ssh", "-S", t.Control, "-O", "exit", t.SSH).Run(); e != nil {
			return fmt.Errorf("SSH tunnel exit failed; it may already be closed")
		}
	} else {
		c, e := net.DialTimeout("unix", t.Control, time.Second)
		if e != nil {
			return e
		}
		io.WriteString(c, "stop\n")
		c.Close()
	}
	if t.Record != "" {
		return os.Remove(t.Record)
	}
	return nil
}

// Proxy is a loopback-only byte stream proxy. WebSocket upgrades need no special
// handling. It is not an HTTP control API and never evaluates client input.
func Proxy(ctx context.Context, listen, target, control string) error {
	for _, a := range []string{listen, target} {
		host, port, e := net.SplitHostPort(a)
		ip := net.ParseIP(host)
		if e != nil || ip == nil || !ip.IsLoopback() {
			return fmt.Errorf("proxy addresses must be numeric loopback addresses")
		}
		n, e := strconv.Atoi(port)
		if e != nil || n < 1 || n > 65535 {
			return fmt.Errorf("invalid proxy port")
		}
	}
	listener, e := net.Listen("tcp", listen)
	if e != nil {
		return e
	}
	defer listener.Close()
	ctl, e := net.Listen("unix", control)
	if e != nil {
		return e
	}
	defer ctl.Close()
	defer os.Remove(control)
	os.Chmod(control, 0600)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() { <-ctx.Done(); listener.Close(); ctl.Close() }()
	go func() {
		for {
			c, e := ctl.Accept()
			if e != nil {
				return
			}
			c.SetDeadline(time.Now().Add(time.Second))
			b := make([]byte, 16)
			n, _ := c.Read(b)
			switch string(b[:n]) {
			case "ping\n":
				io.WriteString(c, "pong\n")
			case "stop\n":
				cancel()
			}
			c.Close()
		}
	}()
	var connections sync.WaitGroup
	for {
		client, e := listener.Accept()
		if e != nil {
			if ctx.Err() != nil {
				return nil
			}
			return e
		}
		connections.Add(1)
		go func() {
			defer connections.Done()
			defer client.Close()
			up, e := net.DialTimeout("tcp", target, 5*time.Second)
			if e != nil {
				return
			}
			defer up.Close()
			done := make(chan struct{})
			go func() {
				select {
				case <-ctx.Done():
					client.Close()
					up.Close()
				case <-done:
				}
			}()
			defer close(done)
			half := make(chan struct{})
			go func() {
				io.Copy(up, client)
				if tcp, ok := up.(*net.TCPConn); ok {
					tcp.CloseWrite()
				}
				close(half)
			}()
			io.Copy(client, up)
			client.Close()
			up.Close()
			<-half
		}()
	}
}

// Decorate remote inventory for UI clients. Runtime/endpoint observations stay
// separate from whether this particular client currently owns an access tunnel.
func (c Client) Decorate(v *fleet.Response) {
	var listing struct {
		HostID    string              `json:"host_id"`
		HostName  string              `json:"host_name"`
		Observed  string              `json:"observed_at"`
		Instances []fleet.Observation `json:"instances"`
	}
	if json.Unmarshal(v.Result, &listing) != nil || listing.Instances == nil {
		return
	}
	for n := range listing.Instances {
		o := &listing.Instances[n]
		if o.GUIEnabled {
			t, e := c.TunnelInfo(o.Instance.ID)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			live := e == nil && tunnelCheck(ctx, t)
			cancel()
			if !live {
				o.GUIEnabled = false
				o.GUIReason = "needs tunnel"
			}
		}
	}
	v.Result, _ = json.Marshal(listing)
}

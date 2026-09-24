package client

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const SSHCommand = "urbit-runner-rpc-v1"

type SSHConfig struct {
	Host       string `json:"host"`
	ConfigFile string `json:"config_file,omitempty"`
}

func (s SSHConfig) Validate() error {
	if len(s.Host) == 0 || len(s.Host) > 253 || s.Host[0] == '-' {
		return errors.New("SSH host must be a configured alias or hostname")
	}
	for _, c := range s.Host {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || strings.ContainsRune("._-", c)) {
			return errors.New("SSH host accepts only ASCII letters, digits, dot, underscore, and hyphen")
		}
	}
	if s.ConfigFile != "" && (!filepath.IsAbs(s.ConfigFile) || filepath.Clean(s.ConfigFile) != s.ConfigFile || strings.ContainsAny(s.ConfigFile, "\x00\r\n")) {
		return errors.New("SSH config path must be absolute and clean")
	}
	return nil
}

func NewSSH(config SSHConfig, identity *Identity) *Client {
	return newTransport(func(ctx context.Context) (net.Conn, error) {
		if err := config.Validate(); err != nil {
			return nil, err
		}
		return dialSSH(ctx, config)
	}, identity)
}

func sshArguments(c SSHConfig) []string {
	args := []string{"-T", "-oBatchMode=yes", "-oStrictHostKeyChecking=yes", "-oUpdateHostKeys=no", "-oClearAllForwardings=yes", "-oForwardAgent=no", "-oForwardX11=no", "-oPermitLocalCommand=no", "-oControlMaster=no", "-oControlPath=none", "-oControlPersist=no", "-oRemoteCommand=none", "-oRequestTTY=no", "-oSessionType=default", "-oForkAfterAuthentication=no", "-oConnectTimeout=5", "-oConnectionAttempts=1", "-oServerAliveInterval=5", "-oServerAliveCountMax=1", "-oLogLevel=ERROR", "-oEscapeChar=none"}
	if c.ConfigFile != "" {
		args = append(args, "-F", c.ConfigFile)
	}
	return append(args, "--", c.Host, SSHCommand)
}

type sshConn struct {
	read, write *os.File
	cmd         *exec.Cmd
	done        chan struct{}
	once        sync.Once
}

type sshAddr string

func (a sshAddr) Network() string                     { return "ssh" }
func (a sshAddr) String() string                      { return string(a) }
func (c *sshConn) LocalAddr() net.Addr                { return sshAddr("collector") }
func (c *sshConn) RemoteAddr() net.Addr               { return sshAddr("runner") }
func (c *sshConn) Read(p []byte) (int, error)         { return c.read.Read(p) }
func (c *sshConn) Write(p []byte) (int, error)        { return c.write.Write(p) }
func (c *sshConn) SetReadDeadline(t time.Time) error  { return c.read.SetReadDeadline(t) }
func (c *sshConn) SetWriteDeadline(t time.Time) error { return c.write.SetWriteDeadline(t) }
func (c *sshConn) SetDeadline(t time.Time) error {
	return errors.Join(c.SetReadDeadline(t), c.SetWriteDeadline(t))
}
func (c *sshConn) Close() error {
	c.once.Do(func() {
		c.read.Close()
		c.write.Close()
		_ = c.cmd.Process.Kill()
		<-c.done
	})
	return nil
}

func dialSSH(ctx context.Context, config SSHConfig) (net.Conn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	in, write, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	read, out, err := os.Pipe()
	if err != nil {
		in.Close()
		write.Close()
		return nil, err
	}
	cmd := exec.Command("/usr/bin/ssh", sshArguments(config)...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = in, out, io.Discard
	if err := cmd.Start(); err != nil {
		in.Close()
		out.Close()
		read.Close()
		write.Close()
		return nil, errors.New("cannot launch the OpenSSH client")
	}
	in.Close()
	out.Close()
	c := &sshConn{read: read, write: write, cmd: cmd, done: make(chan struct{})}
	go func() { _ = cmd.Wait(); close(c.done) }() // let the reader drain stdout before Close
	return c, nil
}

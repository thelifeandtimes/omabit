// Package transport carries versioned requests over owner-only Unix sockets and
// ordinary SSH. Roles are chosen by the listening socket, never by JSON input.
package transport

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"omarchy-urbit/internal/fleet"
)

const MaxRequest = 1 << 20
const MaxResponse = 8 << 20

func ReadRequest(r io.Reader) (fleet.Request, error) {
	var q fleet.Request
	b, e := readLine(r, MaxRequest)
	if e != nil {
		return q, e
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if e = d.Decode(&q); e != nil {
		return q, fmt.Errorf("invalid JSON request")
	}
	var trailing any
	if d.Decode(&trailing) != io.EOF {
		return q, fmt.Errorf("one JSON request per connection required")
	}
	return q, nil
}
func readLine(r io.Reader, max int) ([]byte, error) {
	br := bufio.NewReader(io.LimitReader(r, int64(max+1)))
	b, e := br.ReadBytes('\n')
	if len(b) > max {
		return nil, fmt.Errorf("message exceeds size limit")
	}
	if e != nil && e != io.EOF {
		return nil, e
	}
	if len(b) == 0 {
		return nil, io.ErrUnexpectedEOF
	}
	return b, nil
}
func Socket(p fleet.Paths, agent bool) string {
	n := "owner.sock"
	if agent {
		n = "agent.sock"
	}
	return filepath.Join(p.Runtime, n)
}
func Local(ctx context.Context, p fleet.Paths, agent bool, q fleet.Request) (fleet.Response, error) {
	d := net.Dialer{}
	c, e := d.DialContext(ctx, "unix", Socket(p, agent))
	if e != nil {
		return fleet.Response{}, fmt.Errorf("host service unavailable: %w", e)
	}
	defer c.Close()
	deadline := time.Now().Add(90 * time.Second)
	if t, ok := ctx.Deadline(); ok {
		deadline = t
	}
	c.SetDeadline(deadline)
	if e = json.NewEncoder(c).Encode(q); e != nil {
		return fleet.Response{}, e
	}
	b, e := readLine(c, MaxResponse)
	if e != nil {
		return fleet.Response{}, e
	}
	var v fleet.Response
	e = json.Unmarshal(b, &v)
	if e == nil && v.Schema != fleet.Schema {
		e = fmt.Errorf("unsupported response schema")
	}
	return v, e
}
func SSHArgs(target string, agent bool) []string {
	// Fixed remote command; request contents are stdin, not shell arguments.
	command := `exec "$HOME/.local/bin/urbitctl" rpc`
	if agent {
		command += " --agent"
	}
	return []string{"-T", "-o", "BatchMode=yes", "-o", "ConnectTimeout=10", "-o", "ForwardAgent=no", target, command}
}
func Remote(ctx context.Context, target string, agent bool, q fleet.Request) (fleet.Response, error) {
	if e := ValidateSSH(target); e != nil {
		return fleet.Response{}, e
	}
	b, _ := json.Marshal(q)
	b = append(b, '\n')
	cmd := exec.CommandContext(ctx, "ssh", SSHArgs(target, agent)...)
	cmd.Stdin = bytes.NewReader(b)
	out := &limitBuffer{Limit: MaxResponse}
	cmd.Stdout = out
	cmd.Stderr = io.Discard
	if e := cmd.Run(); e != nil {
		return fleet.Response{}, fmt.Errorf("SSH RPC failed; verify SSH alias, host key, installed CLI and user service (%v)", e)
	}
	var v fleet.Response
	if e := json.Unmarshal(out.Bytes(), &v); e != nil {
		return v, fmt.Errorf("remote response was not JSON; check login-shell output")
	}
	if v.Schema != fleet.Schema {
		return v, fmt.Errorf("unsupported response schema")
	}
	return v, nil
}

type limitBuffer struct {
	bytes.Buffer
	Limit int
}

func (b *limitBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > b.Limit {
		return 0, fmt.Errorf("output exceeds limit")
	}
	return b.Buffer.Write(p)
}
func Gateway(ctx context.Context, p fleet.Paths, agent bool, in io.Reader, out io.Writer) error {
	q, e := ReadRequest(in)
	if e != nil {
		return json.NewEncoder(out).Encode(fleet.Failure(e))
	}
	v, e := Local(ctx, p, agent, q)
	if e != nil {
		v = fleet.Failure(e)
	}
	return json.NewEncoder(out).Encode(v)
}
func Serve(ctx context.Context, p fleet.Paths, m *fleet.Manager) error {
	if e := p.Ensure(); e != nil {
		return e
	}
	lock, e := fleet.AcquireLock(filepath.Join(p.Runtime, "daemon.lock"))
	if e != nil {
		return e
	}
	defer lock.Close()
	var listeners []net.Listener
	defer func() {
		for _, l := range listeners {
			l.Close()
		}
	}()
	var workers sync.WaitGroup
	for _, agent := range []bool{false, true} {
		path := Socket(p, agent)
		if st, e := os.Lstat(path); e == nil {
			if st.Mode()&os.ModeSocket == 0 {
				return fmt.Errorf("refusing to replace non-socket at %s", path)
			}
			if e = os.Remove(path); e != nil {
				return e
			}
		} else if !os.IsNotExist(e) {
			return e
		}
		l, e := net.Listen("unix", path)
		if e != nil {
			return e
		}
		listeners = append(listeners, l)
		os.Chmod(path, 0600)
		role := "owner"
		if agent {
			role = "agent"
		}
		workers.Add(1)
		go func(l net.Listener, role string) {
			defer workers.Done()
			sem := make(chan struct{}, 16)
			for {
				c, e := l.Accept()
				if e != nil {
					return
				}
				select {
				case sem <- struct{}{}:
				case <-ctx.Done():
					c.Close()
					return
				}
				workers.Add(1)
				go func() {
					defer workers.Done()
					defer func() { <-sem }()
					defer c.Close()
					c.SetDeadline(time.Now().Add(90 * time.Second))
					q, e := ReadRequest(c)
					v := fleet.Response{}
					if e != nil {
						v = fleet.Failure(e)
					}
					if e == nil {
						requestCtx, cancel := context.WithTimeout(ctx, 80*time.Second)
						v = m.Handle(requestCtx, role, q)
						cancel()
					}
					json.NewEncoder(c).Encode(v)
				}()
			}
		}(l, role)
	}
	<-ctx.Done()
	for _, l := range listeners {
		l.Close()
	}
	workers.Wait()
	m.Close()
	return nil
}

// No URLs, shell metacharacters, whitespace or option-like target strings.
func ValidateSSH(v string) error {
	if v == "" || len(v) > 255 || strings.HasPrefix(v, "-") {
		return errors.New("invalid SSH target")
	}
	for _, c := range v {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("._-@:", c)) {
			return errors.New("use an SSH configuration alias or user@host")
		}
	}
	return nil
}

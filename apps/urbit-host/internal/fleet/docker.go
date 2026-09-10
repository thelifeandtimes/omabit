package fleet

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

//go:embed scripts/*.sh
var scripts embed.FS

type Executor interface {
	Run(context.Context, []string, []byte) ([]byte, error)
}
type DockerCLI struct{}
type boundedOutput struct {
	bytes.Buffer
	limit int
}

func (b *boundedOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > b.limit {
		return 0, fmt.Errorf("Docker output exceeds bounded capture")
	}
	return b.Buffer.Write(p)
}

// Docker arguments are never passed through a host shell. Deliberately do not
// attach command output to errors: control replies may contain login secrets.
func (DockerCLI) Run(ctx context.Context, args []string, in []byte) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdin = bytes.NewReader(in)
	out := boundedOutput{limit: 4 << 20}
	errout := boundedOutput{limit: 16 << 10}
	cmd.Stdout = &out
	cmd.Stderr = &errout
	if len(args) > 0 && args[0] == "logs" {
		cmd.Stderr = &out
	}
	e := cmd.Run()
	if e != nil {
		return nil, fmt.Errorf("docker %s failed (%v); inspect local Docker/service logs", args[0], e)
	}
	if out.Len() > 4<<20 {
		return nil, fmt.Errorf("Docker response exceeds 4 MiB")
	}
	return out.Bytes(), nil
}

type Engine struct {
	Exec  Executor
	Paths Paths
	Host  HostConfig
}

func (e *Engine) run(ctx context.Context, a ...string) ([]byte, error) {
	return e.Exec.Run(ctx, a, nil)
}
func (e *Engine) RuntimeDir() string { return filepath.Join(e.Paths.Data, "runtime", Version) }
func (e *Engine) InstallScripts() error {
	d := e.RuntimeDir()
	if err := os.MkdirAll(d, 0755); err != nil {
		return err
	}
	account, group := AccountFiles(os.Getuid(), os.Getgid())
	files := map[string][]byte{"passwd": []byte(account), "group": []byte(group)}
	for _, n := range []string{"init.sh", "boot.sh", "control.sh"} {
		b, err := scripts.ReadFile("scripts/" + n)
		if err != nil {
			return err
		}
		files[n] = b
	}
	for n, b := range files {
		p := filepath.Join(d, n)
		if existing, err := os.ReadFile(p); err == nil {
			if !bytes.Equal(existing, b) {
				return fmt.Errorf("runtime scripts differ at %s; use a new version, do not overwrite live scripts", d)
			}
			continue
		}
		mode := os.FileMode(0755)
		if n == "passwd" || n == "group" {
			mode = 0644
		}
		if err := os.WriteFile(p, b, mode); err != nil {
			return err
		}
	}
	return nil
}
func (e *Engine) ownerArgs() []string {
	return []string{"--user", fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()), "--cap-drop", "ALL", "--cap-add", "NET_BIND_SERVICE", "--security-opt", "no-new-privileges", "--log-driver", "local", "--log-opt", "max-size=10m", "--log-opt", "max-file=3"}
}
func (e *Engine) EnsureImage(ctx context.Context, ref string) error {
	if err := ValidateImage(ref); err != nil {
		return err
	}
	_, err := e.run(ctx, "image", "inspect", ref)
	if err == nil {
		return nil
	}
	_, err = e.run(ctx, "pull", ref)
	return err
}
func (e *Engine) inspect(ctx context.Context, name string) (map[string]any, error) {
	b, err := e.run(ctx, "container", "inspect", name)
	if err != nil {
		return nil, err
	}
	var a []map[string]any
	if err = json.Unmarshal(b, &a); err != nil {
		return nil, err
	}
	if len(a) != 1 {
		return nil, fmt.Errorf("unexpected inspect result")
	}
	return a[0], nil
}
func containerState(m map[string]any) string {
	s, _ := m["State"].(map[string]any)
	v, _ := s["Status"].(string)
	return v
}
func (e *Engine) checkOwner(m map[string]any, id string) error {
	c, _ := m["Config"].(map[string]any)
	l, _ := c["Labels"].(map[string]any)
	if l[LabelPrefix+".host"] != e.Host.ID || l[LabelPrefix+".instance"] != id {
		return fmt.Errorf("container name collision or ownership mismatch")
	}
	return nil
}
func (e *Engine) CheckContainer(ctx context.Context, i Instance) (string, error) {
	m, err := e.inspect(ctx, i.Container)
	if err != nil {
		return "unknown", err
	}
	if err = e.checkOwner(m, i.ID); err != nil {
		return "unknown", err
	}
	return containerState(m), nil
}
func (e *Engine) Exists(ctx context.Context, name string) (bool, error) {
	b, err := e.run(ctx, "container", "ls", "-a", "--filter", "name=^/"+name+"$", "--format", "{{.Names}}")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(b)) == name, nil
}
func (e *Engine) EnsureGroup(ctx context.Context, g Group, image string) error {
	exists, err := e.Exists(ctx, g.Namespace)
	if err != nil {
		return err
	}
	if exists {
		m, err := e.inspect(ctx, g.Namespace)
		if err != nil {
			return err
		}
		if err = e.checkOwner(m, "group-"+g.ID); err != nil {
			return err
		}
		if containerState(m) != "running" {
			_, err = e.run(ctx, "start", g.Namespace)
			return err
		}
		return nil
	}
	// The bridge distinguishes environments; a namespace keeper additionally shares
	// loopback among member ships. A bridge alone does not satisfy fake routing.
	b, err := e.run(ctx, "network", "ls", "--filter", "name=^"+g.Network+"$", "--format", "{{.Name}}")
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(b)) == "" {
		_, err = e.run(ctx, "network", "create", "--driver", "bridge", "--label", LabelPrefix+".host="+e.Host.ID, g.Network)
		if err != nil {
			return err
		}
	} else {
		b, err = e.run(ctx, "network", "inspect", "--format", `{{index .Labels "`+LabelPrefix+`.host"}}`, g.Network)
		if err != nil {
			return err
		}
		if strings.TrimSpace(string(b)) != e.Host.ID {
			return fmt.Errorf("network ownership mismatch")
		}
	}
	a := []string{"run", "-d", "--name", g.Namespace, "--label", LabelPrefix + ".host=" + e.Host.ID, "--label", LabelPrefix + ".instance=group-" + g.ID, "--network", g.Network, "--restart", "unless-stopped", "--publish", fmt.Sprintf("127.0.0.1:%d-%d:8080-8335/tcp", g.HTTPBase, g.HTTPBase+255)}
	a = append(a, e.ownerArgs()...)
	a = append(a, "--entrypoint", "/bin/sh", image, "-c", "trap 'exit 0' TERM INT; while :; do sleep 3600 & wait $!; done")
	_, err = e.run(ctx, a...)
	return err
}
func (e *Engine) CreateArgs(i Instance, g *Group, dir, secret string) []string {
	a := []string{"create", "--name", i.Container, "--label", LabelPrefix + ".host=" + e.Host.ID, "--label", LabelPrefix + ".instance=" + i.ID, "--label", LabelPrefix + ".kind=" + i.Kind, "--restart", "unless-stopped", "--stop-timeout", "180", "--mount", "type=bind,src=" + dir + ",dst=/urbit", "--mount", "type=bind,src=" + e.RuntimeDir() + ",dst=/opt/omarchy-urbit,readonly", "--tmpfs", "/tmp:rw,nosuid,nodev,mode=1777", "--workdir", "/urbit", "--env", "HOME=/tmp", "--env", "TMUX_TMPDIR=/tmp"}
	a = append(a, e.ownerArgs()...)
	a = append(a, "--mount", "type=bind,src="+filepath.Join(e.RuntimeDir(), "passwd")+",dst=/etc/passwd,readonly", "--mount", "type=bind,src="+filepath.Join(e.RuntimeDir(), "group")+",dst=/etc/group,readonly")
	if g != nil {
		a = append(a, "--network", "container:"+g.Namespace)
	} else {
		a = append(a, "--network", "bridge", "--publish", fmt.Sprintf("127.0.0.1:%d:%d/tcp", i.HostPort, i.HTTPPort), "--publish", fmt.Sprintf("%d:%d/udp", i.AmesPort, i.AmesPort))
	}
	if secret != "" {
		a = append(a, "--mount", "type=bind,src="+secret+",dst=/boot-secret,readonly")
	}
	return append(a, "--entrypoint", "/bin/bash", i.Image, "/opt/omarchy-urbit/init.sh", i.Kind, i.Identity, strconv.Itoa(i.HTTPPort), strconv.Itoa(i.AmesPort), strconv.Itoa(i.Loom), i.Pill)
}
func (e *Engine) Create(ctx context.Context, i Instance, g *Group, secret string) error {
	if err := e.EnsureImage(ctx, i.Image); err != nil {
		return err
	}
	if err := e.InstallScripts(); err != nil {
		return err
	}
	if g != nil {
		if err := e.EnsureGroup(ctx, *g, i.Image); err != nil {
			return err
		}
	}
	dir, err := CheckPath(e.Host.DataRoot, i.RelativeDir())
	if err != nil {
		return err
	}
	if err = os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	exists, err := e.Exists(ctx, i.Container)
	if err != nil {
		return err
	}
	if !exists {
		if _, err = e.run(ctx, e.CreateArgs(i, g, dir, secret)...); err != nil {
			return err
		}
	} else {
		m, err := e.inspect(ctx, i.Container)
		if err != nil {
			return err
		}
		if err = e.checkOwner(m, i.ID); err != nil {
			return err
		}
	}
	return e.Start(ctx, i)
}
func (e *Engine) Start(ctx context.Context, i Instance) error {
	if _, err := e.CheckContainer(ctx, i); err != nil {
		return err
	}
	if _, err := e.run(ctx, "update", "--restart=unless-stopped", i.Container); err != nil {
		return err
	}
	_, err := e.run(ctx, "start", i.Container)
	return err
}
func (e *Engine) Stop(ctx context.Context, i Instance) error {
	s, err := e.CheckContainer(ctx, i)
	if err != nil {
		return err
	}
	if s == "exited" || s == "created" {
		_, err := e.run(ctx, "update", "--restart=no", i.Container)
		return err
	}
	if s != "running" && s != "restarting" {
		return fmt.Errorf("cannot gracefully stop container in state %s", s)
	}
	// Disable automatic restart before a graceful signal (docker kill is not docker stop).
	if _, err = e.run(ctx, "update", "--restart=no", i.Container); err != nil {
		return err
	}
	// Never escalate automatically to SIGKILL. Failed shutdown remains visible.
	if _, err = e.run(ctx, "kill", "--signal=TERM", i.Container); err != nil {
		return err
	}
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("graceful stop did not finish; no force-kill was sent")
		case <-tick.C:
			s, err = e.CheckContainer(ctx, i)
			if err != nil {
				return err
			}
			if s == "exited" {
				return nil
			}
		}
	}
}
func (e *Engine) Remove(ctx context.Context, i Instance) error {
	exists, err := e.Exists(ctx, i.Container)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	s, err := e.CheckContainer(ctx, i)
	if err != nil {
		return err
	}
	if s != "exited" && s != "created" {
		return fmt.Errorf("refusing to remove a non-stopped container")
	}
	_, err = e.run(ctx, "rm", i.Container)
	return err
}
func (e *Engine) Observe(ctx context.Context, i Instance, g *Group) Observation {
	o := Observation{Instance: i, Container: "unknown", Runtime: "unknown", Endpoint: "unavailable", Observed: Now(), HostPort: i.HostPort, GUIReason: "unavailable"}
	if g != nil {
		idx, _ := Galaxy(i.Identity)
		o.HostPort = g.HTTPBase + idx
	}
	if i.Archived != "" || i.Lifecycle == "purged" {
		o.Container = "absent"
		o.Runtime = "stopped"
		return o
	}
	exists, err := e.Exists(ctx, i.Container)
	if err != nil {
		o.Error = err.Error()
		return o
	}
	if !exists {
		o.Container = "missing"
		return o
	}
	s, err := e.CheckContainer(ctx, i)
	if err != nil {
		o.Error = err.Error()
		return o
	}
	o.Container = s
	if s != "running" {
		o.Runtime = "stopped"
		return o
	}
	// This proves HTTP responsiveness, not an independent Hoon/Arvo event probe.
	o.Runtime = "starting-or-unresponsive"
	req, _ := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://127.0.0.1:%d/", o.HostPort), nil)
	client := &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{Proxy: nil, DisableKeepAlives: true}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := client.Do(req)
	if err == nil {
		io.Copy(io.Discard, io.LimitReader(res.Body, 4096))
		res.Body.Close()
		if res.StatusCode >= 200 && res.StatusCode < 500 {
			o.Endpoint = "http-responsive"
			o.Runtime = "http-responsive"
			o.GUIEnabled = true
			o.GUIReason = ""
		} else {
			o.Endpoint = "http-error"
		}
	}
	return o
}
func (e *Engine) Control(ctx context.Context, i Instance, expression string) (string, error) {
	s, err := e.CheckContainer(ctx, i)
	if err != nil {
		return "", err
	}
	if s != "running" {
		return "", fmt.Errorf("ship is not running")
	}
	in, _ := json.Marshal(map[string]any{"source": map[string]string{"dojo": expression}, "sink": map[string]any{"stdout": nil}})
	b, err := e.Exec.Run(ctx, []string{"exec", "-i", i.Container, "/bin/bash", "/opt/omarchy-urbit/control.sh"}, in)
	if err != nil {
		return "", err
	}
	var out string
	if err = json.Unmarshal(b, &out); err != nil {
		return "", fmt.Errorf("unexpected loopback API response; runtime adapter needs validation")
	}
	return strings.TrimSpace(out), nil
}
func (e *Engine) Stats(ctx context.Context, instances []Instance) ([]map[string]string, error) {
	a := []string{"stats", "--no-stream", "--format", "{{json .}}"}
	for _, i := range instances {
		if i.Archived == "" && i.Lifecycle != "purged" {
			exists, err := e.Exists(ctx, i.Container)
			if err != nil {
				return nil, err
			}
			if !exists {
				continue
			}
			state, err := e.CheckContainer(ctx, i)
			if err != nil {
				return nil, err
			}
			if state == "running" {
				a = append(a, i.Container)
			}
		}
	}
	if len(a) == 4 {
		return []map[string]string{}, nil
	}
	b, err := e.run(ctx, a...)
	if err != nil {
		return nil, err
	}
	out := []map[string]string{}
	for _, line := range bytes.Split(bytes.TrimSpace(b), []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var m map[string]string
		if err = json.Unmarshal(line, &m); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

// AccountFiles supplies a name for the numeric host UID in an otherwise unknown
// image. Only the ship container's account files are overlaid, never the host's.
// This avoids requiring the runtime image to pre-create every possible host UID.
func AccountFiles(uid, gid int) (string, string) {
	p := "root:x:0:0:root:/root:/bin/bash\n"
	if uid != 0 {
		p += fmt.Sprintf("runner:x:%d:%d:Urbit runner:/tmp:/bin/bash\n", uid, gid)
	}
	if uid != 65534 {
		p += "nobody:x:65534:65534:nobody:/:/bin/false\n"
	}
	g := "root:x:0:\n"
	if gid != 0 {
		g += fmt.Sprintf("runner:x:%d:\n", gid)
	}
	if gid != 65534 {
		g += "nogroup:x:65534:\n"
	}
	return p, g
}

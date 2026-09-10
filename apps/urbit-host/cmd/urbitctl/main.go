// urbitctl is a JSON-first operator/agent CLI and per-user execution-host daemon.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"omarchy-urbit/internal/fleet"
	"omarchy-urbit/internal/transport"
)

const usage = `urbitctl 0.1.0-pilot — Docker Urbit runner (Linux; disposable-ship pilot)

Global flags BEFORE command: --host ALIAS --agent --request-id ID --json
Command flags BEFORE positional arguments. JSON is the default output.

Local setup: host init --name workstation [--data-root /absolute/path]
             host add --ssh SSH_ALIAS [--default] NAME
             host list | daemon | doctor | version
Development: dev group create NAME | dev group list
             dev create --group NAME --fake zod --label demo-zod [--image REF]
Ships:       ship list [--all-hosts] | ship inspect ID_OR_LABEL
             ship boot --comet --label NAME
             ship boot --ship '~sampel-palnet' --key-stdin --label NAME
             ship start|stop|restart|archive [--confirm FULL_ID] ID_OR_LABEL
             ship purge --confirm FULL_ID ID_OR_LABEL
             ship logs ID_OR_LABEL
             ship code --clipboard|--show ID_OR_LABEL
             ship exec --hoon-stdin [--confirm FULL_ID] ID_OR_LABEL
             ship dojo|open|tunnel|close-tunnel ID_OR_LABEL
Operations:  operation list | operation inspect ID | operation wait ID
Observe:     metrics | updates check | notices list | notices dismiss ID

Boot options: --image REF --channel latest|edge|canary --pill HTTPS_URL --loom 31
Real-network creation is gated off by default. See docs/PILOT.md.
Interactive Dojo is operator-only in this pilot. Agent CLI uses ship exec.
No backups, runtime replacement, cross-host fake relay or QML UI yet.
`

func main() {
	if e := run(os.Args[1:]); e != nil {
		var reported *alreadyReported
		if !errors.As(e, &reported) {
			emit(fleet.Failure(e))
		}
		os.Exit(1)
	}
}
func emit(v any)     { enc := json.NewEncoder(os.Stdout); enc.SetIndent("", "  "); enc.Encode(v) }
func ok(v any) error { emit(fleet.Success(v)); return nil }
func fs(name string) *flag.FlagSet {
	f := flag.NewFlagSet(name, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	return f
}
func requireArgs(f *flag.FlagSet, n int) error {
	if f.NArg() != n {
		return fmt.Errorf("%s expects %d positional argument(s); put flags before arguments", f.Name(), n)
	}
	return nil
}
func readInput(max int) (string, error) {
	b, e := io.ReadAll(io.LimitReader(os.Stdin, int64(max+1)))
	if e != nil {
		return "", e
	}
	if len(b) > max {
		return "", fmt.Errorf("stdin exceeds limit")
	}
	return strings.TrimSpace(string(b)), nil
}
func run(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		fmt.Print(usage)
		return nil
	}
	global := fs("urbitctl")
	host := global.String("host", "", "host alias")
	agent := global.Bool("agent", false, "restricted gateway")
	rid := global.String("request-id", "", "idempotency key")
	global.Bool("json", true, "JSON output (default)")
	if e := global.Parse(args); e != nil {
		return e
	}
	args = global.Args()
	if len(args) == 0 {
		return errors.New("command required")
	}
	p, e := fleet.DefaultPaths()
	if e != nil {
		return e
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if args[0] == "version" {
		return ok(map[string]any{"version": fleet.Version, "schema": fleet.Schema})
	}
	if args[0] == "host" {
		return hostCommand(p, args[1:])
	}
	if args[0] == "rpc" {
		f := fs("rpc")
		a := f.Bool("agent", false, "")
		if e = f.Parse(args[1:]); e != nil {
			return e
		}
		if e = requireArgs(f, 0); e != nil {
			return e
		}
		return transport.Gateway(ctx, p, *a, os.Stdin, os.Stdout)
	}
	if args[0] == "daemon" {
		return daemon(p)
	}
	if args[0] == "proxy" {
		f := fs("proxy")
		listen := f.String("listen", "", "")
		target := f.String("target", "", "")
		ctl := f.String("control", "", "")
		if e = f.Parse(args[1:]); e != nil {
			return e
		}
		sig, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		return transport.Proxy(sig, *listen, *target, *ctl)
	}
	c, e := transport.NewClient(p, *host, *agent)
	if e != nil {
		return e
	}
	q := fleet.Request{Schema: fleet.Schema, ID: *rid}
	if q.ID == "" {
		q.ID = fleet.ID()
	}
	var clipboard, show, all, wait bool
	action := ""
	rest := args[1:]
	switch args[0] {
	case "doctor", "metrics":
		q.Method = args[0]
		if len(rest) != 0 {
			return errors.New("unexpected arguments")
		}
	case "dev":
		if len(rest) == 0 {
			return errors.New("dev command required")
		}
		if rest[0] == "group" {
			if len(rest) < 2 {
				return errors.New("group command required")
			}
			q.Method = "group." + rest[1]
			if rest[1] == "create" && len(rest) == 3 {
				q.Label = rest[2]
			} else if rest[1] != "list" || len(rest) != 2 {
				return errors.New("use dev group create NAME or dev group list")
			}
		} else if rest[0] == "create" {
			q.Method = "ship.create"
			q.Kind = "fake"
			f := bootFlags(&q)
			f.StringVar(&q.Group, "group", "", "")
			f.StringVar(&q.Identity, "fake", "zod", "")
			if e = f.Parse(rest[1:]); e != nil {
				return e
			}
			if e = requireArgs(f, 0); e != nil {
				return e
			}
		} else {
			return errors.New("unknown development command")
		}
	case "ship":
		if len(rest) == 0 {
			return errors.New("ship action required")
		}
		action = rest[0]
		q.Method = "ship." + action
		f := fs("ship " + action)
		switch action {
		case "boot":
			q.Method = "ship.create"
			f = bootFlags(&q)
			comet := f.Bool("comet", false, "")
			key := f.Bool("key-stdin", false, "")
			f.StringVar(&q.Identity, "ship", "", "")
			if e = f.Parse(rest[1:]); e != nil {
				return e
			}
			if e = requireArgs(f, 0); e != nil {
				return e
			}
			if *comet {
				q.Kind = "comet"
				if *key || q.Identity != "" {
					return errors.New("--comet cannot be combined with --ship or --key-stdin")
				}
			} else {
				q.Kind = "keyed"
				if !*key {
					return errors.New("use --comet or --ship with --key-stdin")
				}
				q.Key, e = readInput(16384)
				if e != nil {
					return e
				}
			}
		case "list":
			f.BoolVar(&all, "all-hosts", false, "")
			if e = f.Parse(rest[1:]); e != nil {
				return e
			}
			if e = requireArgs(f, 0); e != nil {
				return e
			}
		default:
			f.StringVar(&q.Confirm, "confirm", "", "")
			f.BoolVar(&clipboard, "clipboard", false, "")
			f.BoolVar(&show, "show", false, "")
			hoon := f.Bool("hoon-stdin", false, "")
			if e = f.Parse(rest[1:]); e != nil {
				return e
			}
			if e = requireArgs(f, 1); e != nil {
				return e
			}
			q.Target = f.Arg(0)
			if action == "exec" {
				if !*hoon {
					return errors.New("use --hoon-stdin")
				}
				q.Hoon, e = readInput(65536)
				if e != nil {
					return e
				}
			}
			if action == "code" && clipboard == show {
				return errors.New("choose exactly one of --clipboard or --show")
			}
			if action == "open" || action == "tunnel" || action == "close-tunnel" {
				q.Method = "ship.inspect"
			}
		}
	case "operation", "notices", "updates":
		if len(rest) == 0 {
			return errors.New("subcommand required")
		}
		sub := rest[0]
		q.Method = args[0] + "." + sub
		if args[0] == "operation" && sub == "wait" {
			q.Method = "operation.inspect"
			wait = true
		}
		if sub == "inspect" || sub == "wait" || sub == "dismiss" {
			if len(rest) != 2 {
				return errors.New("ID required")
			}
			q.Target = rest[1]
		} else if len(rest) != 1 {
			return errors.New("unexpected arguments")
		}
	default:
		return errors.New("unknown command; run urbitctl help")
	}
	if all {
		v, e := transport.Inventory(ctx, p, *agent)
		if e != nil {
			return e
		}
		return ok(v)
	}
	if wait {
		return waitOperation(c, q)
	}
	v, e := c.Call(ctx, q)
	q.Key = ""
	q.Hoon = ""
	if e != nil {
		return e
	}
	if !v.OK {
		emit(v)
		return &alreadyReported{}
	}
	if action == "list" {
		c.Decorate(&v)
	}
	switch action {
	case "code":
		if clipboard {
			var data struct {
				Code string `json:"code"`
			}
			if e = json.Unmarshal(v.Result, &data); e != nil {
				return e
			}
			cmd := exec.CommandContext(ctx, "wl-copy", "--type", "text/plain")
			cmd.Stdin = strings.NewReader(data.Code)
			if e = cmd.Run(); e != nil {
				return fmt.Errorf("local Wayland clipboard unavailable (%v)", e)
			}
			return ok(map[string]bool{"copied": true})
		}
	case "dojo":
		return dojo(c, q.Target, v)
	case "open", "tunnel", "close-tunnel":
		var o fleet.Observation
		if e = json.Unmarshal(v.Result, &o); e != nil {
			return e
		}
		if action == "close-tunnel" {
			t, e := c.TunnelInfo(o.Instance.ID)
			if e != nil {
				return e
			}
			if e = transport.CloseTunnel(ctx, t); e != nil {
				return e
			}
			return ok(map[string]bool{"closed": true})
		}
		t, e := c.EnsureTunnel(ctx, o)
		if e != nil {
			return e
		}
		if action == "open" {
			if e = exec.CommandContext(ctx, "xdg-open", t.URL).Run(); e != nil {
				return fmt.Errorf("tunnel available at %s; xdg-open failed (%v)", t.URL, e)
			}
		}
		return ok(t)
	}
	emit(v)
	return nil
}

type alreadyReported struct{}

func (*alreadyReported) Error() string { return "remote operation rejected" }
func bootFlags(q *fleet.Request) *flag.FlagSet {
	f := fs("boot/create")
	f.StringVar(&q.Label, "label", "", "")
	f.StringVar(&q.Channel, "channel", "", "")
	f.StringVar(&q.Image, "image", "", "")
	f.StringVar(&q.Pill, "pill", "", "")
	f.IntVar(&q.Loom, "loom", 31, "")
	return f
}
func hostCommand(p fleet.Paths, args []string) error {
	if len(args) == 0 {
		return errors.New("host command required")
	}
	f := fs("host " + args[0])
	switch args[0] {
	case "init":
		name := f.String("name", "", "")
		root := f.String("data-root", "", "")
		if e := f.Parse(args[1:]); e != nil {
			return e
		}
		if e := requireArgs(f, 0); e != nil {
			return e
		}
		c, e := fleet.InitHost(p, *name, *root)
		if e != nil {
			return e
		}
		return ok(c)
	case "add":
		ssh := f.String("ssh", "", "")
		def := f.Bool("default", false, "")
		if e := f.Parse(args[1:]); e != nil {
			return e
		}
		if e := requireArgs(f, 1); e != nil {
			return e
		}
		if e := p.Ensure(); e != nil {
			return e
		}
		c, e := transport.AddHost(p, f.Arg(0), *ssh, *def)
		if e != nil {
			return e
		}
		return ok(c)
	case "list":
		c, e := transport.LoadClient(p)
		if e != nil {
			return e
		}
		return ok(c)
	default:
		return errors.New("unknown host command")
	}
}
func daemon(p fleet.Paths) error {
	if e := p.Ensure(); e != nil {
		return e
	}
	lock, e := fleet.AcquireLock(filepath.Join(p.State, "host.lock"))
	if e != nil {
		return e
	}
	defer lock.Close()
	c, e := fleet.LoadHost(p)
	if e != nil {
		return fmt.Errorf("initialize this execution host first: %w", e)
	}
	engine := &fleet.Engine{Exec: fleet.DockerCLI{}, Paths: p, Host: c}
	m, e := fleet.NewManager(engine)
	if e != nil {
		return e
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	done := make(chan struct{})
	go func() { defer close(done); m.Housekeeping(ctx) }()
	err := transport.Serve(ctx, p, m)
	stop()
	<-done
	return err
}
func waitOperation(c transport.Client, q fleet.Request) error {
	ctx, cancel := context.WithTimeout(context.Background(), 22*time.Minute)
	defer cancel()
	for {
		v, e := c.Call(ctx, q)
		if e != nil {
			return e
		}
		if !v.OK {
			emit(v)
			return &alreadyReported{}
		}
		var op fleet.Operation
		if e = json.Unmarshal(v.Result, &op); e != nil {
			return e
		}
		switch op.State {
		case "succeeded":
			emit(v)
			return nil
		case "failed", "interrupted":
			emit(v)
			return &alreadyReported{}
		}
		select {
		case <-ctx.Done():
			return errors.New("wait timed out; host operation is not cancelled")
		case <-time.After(time.Second):
		}
	}
}
func dojo(c transport.Client, target string, v fleet.Response) error {
	if c.Agent {
		return errors.New("interactive Dojo requires operator credentials; use ship exec for development automation")
	}
	var data struct {
		Container string `json:"container"`
	}
	if e := json.Unmarshal(v.Result, &data); e != nil {
		return e
	}
	if !strings.HasPrefix(data.Container, "ou-ship-") || len(data.Container) != 40 {
		return errors.New("unexpected container identifier")
	}
	for _, r := range strings.TrimPrefix(data.Container, "ou-ship-") {
		if !strings.ContainsRune("0123456789abcdef", r) {
			return errors.New("invalid container identifier")
		}
	}
	var cmd *exec.Cmd
	if c.Host.SSH != "" {
		cmd = exec.Command("ssh", "-t", "-o", "ForwardAgent=no", c.Host.SSH, "docker exec -it "+data.Container+" tmux attach-session -t urbit")
	} else {
		cmd = exec.Command("docker", "exec", "-it", data.Container, "tmux", "attach-session", "-t", "urbit")
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

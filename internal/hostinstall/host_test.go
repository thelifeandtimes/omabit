package hostinstall

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func hostPublic(seed byte) []byte {
	data := append([]byte{0, 0, 0, 11}, []byte("ssh-ed25519")...)
	data = append(data, 0, 0, 0, 32)
	data = append(data, bytes.Repeat([]byte{seed}, 32)...)
	return []byte("ssh-ed25519 " + base64.StdEncoding.EncodeToString(data) + "\n")
}

func hostFixture(t *testing.T) (*fixture, hostProbe) {
	f := newFixture(t)
	for _, name := range hostHelpers {
		f.put("/usr/bin/"+name, "inert test helper", 0755)
	}
	run := f.i.run
	f.i.run = func(ctx context.Context, path string, args, env []string) ([]byte, error) {
		out, err := run(ctx, path, args, env)
		if err != nil {
			return out, err
		}
		if path == "/usr/bin/ssh-keygen" {
			if slices.Contains(args, "-y") {
				return hostPublic(2), nil
			}
			keyPath := args[len(args)-1]
			f.put(keyPath, "private fixture", 0600)
			f.put(keyPath+".pub", string(hostPublic(2)), 0644)
		}
		if path == "/usr/bin/usermod" {
			f.put("/etc/passwd", "root:x:0:0::/root:/bin/sh\nurbit-runner:x:942:942::/var/lib/urbit-runner:"+access.ShellPath+"\n", 0644)
		}
		if path == "/usr/bin/systemctl" && slices.Contains(args, "enable") && slices.Contains(args, "urbit-runner.service") {
			f.put("/run/user/942/urbit-runner/runner.sock", "inert socket fixture", 0600)
		}
		return out, nil
	}
	probe := func(context.Context, string) (protocol.Hello, bool, error) {
		return protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-" + strings.Repeat("a", 32), Mode: "development-managed"}, false, nil
	}
	configureActivationFixture(f)
	return f, probe
}

func TestHostSetupFreshThenRerunPreservesIdentityAndGrants(t *testing.T) {
	f, probe := hostFixture(t)
	result, err := f.i.installHost(context.Background(), "/bundle", hostPublic(1), []byte("runner-one"), probe, nil)
	if err != nil || !result.Ready || result.Partial || result.UID != testUID || result.HostPublicKey != strings.TrimSpace(string(hostPublic(2))) {
		t.Fatal("fresh host", result, err)
	}
	key, err := os.ReadFile(f.i.path(hostKeyPath))
	if err != nil {
		t.Fatal(err)
	}
	observer, err := access.AuthorizedKey(hostPublic(3), "observer")
	if err != nil {
		t.Fatal(err)
	}
	grants, err := os.ReadFile(f.i.path(access.DefaultKeys))
	if err != nil {
		t.Fatal(err)
	}
	f.put(access.DefaultKeys, string(grants)+observer+"\n", 0644)
	again, err := f.i.installHost(context.Background(), "/bundle", hostPublic(1), []byte("runner-two"), probe, nil)
	if err != nil || !again.Ready || again.RunnerID != result.RunnerID || again.HostPublicKey != result.HostPublicKey {
		t.Fatal("rerun", again, err)
	}
	current, err := os.ReadFile(f.i.path(hostKeyPath))
	if err != nil || !bytes.Equal(key, current) {
		t.Fatal("host key changed")
	}
	current, err = os.ReadFile(f.i.path(access.DefaultKeys))
	if err != nil || string(current) != string(grants)+observer+"\n" {
		t.Fatal("grants duplicated or lost")
	}
	current, err = os.ReadFile(f.i.path(access.ShellPath))
	if err != nil || string(current) != "runner-two" {
		t.Fatal("current runner was not updated")
	}
	creates, keys := 0, 0
	for _, c := range f.calls {
		if c.path == "/usr/bin/useradd" && !slices.Contains(c.args, "--help") {
			creates++
		}
		if c.path == "/usr/bin/ssh-keygen" && !slices.Contains(c.args, "-y") {
			keys++
		}
	}
	if creates != 1 || keys != 1 {
		t.Fatal("identity regenerated", creates, keys)
	}
}

func TestHostSetupResumesAfterPartialFirstInstall(t *testing.T) {
	f, probe := hostFixture(t)
	f.fail = func(c call) error {
		if c.path == "/usr/bin/ssh-keygen" {
			return errors.New("key generation interrupted")
		}
		return nil
	}
	r, err := f.i.installHost(context.Background(), "/bundle", hostPublic(1), []byte("runner"), probe, nil)
	if err == nil || !r.Partial || !f.created {
		t.Fatal("partial install evidence", r, err)
	}
	f.fail = nil
	r, err = f.i.installHost(context.Background(), "/bundle", hostPublic(1), []byte("runner"), probe, nil)
	if err != nil || !r.Ready {
		t.Fatal("partial installation cannot resume", r, err)
	}
}

func TestHostSetupPreservesModifiedConfigAndMissingHostIdentity(t *testing.T) {
	for _, fault := range []string{"sshd-config", "host-key", "busy", "identity", "unit"} {
		t.Run(fault, func(t *testing.T) {
			f, probe := hostFixture(t)
			if _, err := f.i.installHost(context.Background(), "/bundle", hostPublic(1), []byte("runner"), probe, nil); err != nil {
				t.Fatal(err)
			}
			switch fault {
			case "sshd-config":
				f.put(sshConfigPath, "custom config", 0644)
			case "host-key":
				if err := os.Remove(f.i.path(hostKeyPath)); err != nil {
					t.Fatal(err)
				}
			case "unit":
				f.put(managerUnit, "custom unit", 0644)
			case "busy":
				probe = func(context.Context, string) (protocol.Hello, bool, error) { return protocol.Hello{}, true, nil }
			case "identity":
				calls := 0
				probe = func(context.Context, string) (protocol.Hello, bool, error) {
					calls++
					id := "runner-before"
					if calls > 1 {
						id = "runner-after"
					}
					return protocol.Hello{RunnerID: id, Mode: "development-managed"}, false, nil
				}
			}
			if r, err := f.i.installHost(context.Background(), "/bundle", hostPublic(1), []byte("updated"), probe, nil); err == nil || r.Ready {
				t.Fatal("unsafe update accepted", fault, r, err)
			}
			if fault == "sshd-config" {
				data, _ := os.ReadFile(f.i.path(sshConfigPath))
				if string(data) != "custom config" {
					t.Fatal("custom config overwritten")
				}
			}
			if fault == "unit" {
				data, _ := os.ReadFile(f.i.path(managerUnit))
				if string(data) != "custom unit" {
					t.Fatal("custom unit overwritten")
				}
			}
			if fault == "host-key" {
				if _, err := os.Stat(f.i.path(hostKeyPath)); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("host identity regenerated")
				}
			}
		})
	}
}

func TestHostSetupRejectsInvalidInputsBeforeMutation(t *testing.T) {
	for _, fault := range []string{"root", "public", "bundle"} {
		f, probe := hostFixture(t)
		pub := hostPublic(1)
		switch fault {
		case "root":
			f.i.euid = 1000
		case "public":
			pub = []byte("not a public key")
		case "bundle":
			f.i.verified = func(string) (map[string][]byte, error) { return nil, errors.New("invalid bundle") }
		}
		if _, err := f.i.installHost(context.Background(), "/bundle", pub, []byte("runner"), probe, nil); err == nil || f.created || f.i.mutated {
			t.Fatal("invalid setup changed host", fault, err)
		}
	}
}

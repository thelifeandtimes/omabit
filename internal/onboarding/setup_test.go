package onboarding

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostinstall"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostsetup"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

func setupFixture(t *testing.T) (dependencies, string, string, string, *int) {
	t.Helper()
	base := t.TempDir()
	dir := filepath.Join(base, "setup")
	config := filepath.Join(base, "collector")
	runner := filepath.Join(base, "urbit-runner")
	if err := os.WriteFile(runner, []byte("inert fixture"), 0700); err != nil {
		t.Fatal(err)
	}
	d := defaults()
	calls := new(int)
	d.inspect = func() hostsetup.Report {
		return hostsetup.Report{Checks: []hostsetup.Check{{ID: "platform", Status: "pass"}, {ID: "docker-version", Status: "pass"}, {ID: "account", Status: "missing"}}}
	}
	d.tool = func(string) bool { return true }
	d.prepare = func(context.Context, string) (rootless.Manifest, error) { return rootless.Manifest{}, nil }
	d.elevate = func(ctx context.Context, exe, bundle, public string, emit func(Event)) (hostinstall.HostResult, error) {
		*calls++
		if exe != runner || !strings.HasPrefix(bundle, dir+string(os.PathSeparator)) {
			t.Fatal("unbound installer input")
		}
		pub, err := os.ReadFile(public)
		if err != nil {
			t.Fatal(err)
		}
		line, err := access.AuthorizedKey(pub, "operator")
		if err != nil {
			t.Fatal(err)
		}
		grant, err := access.ParseLine(line)
		if err != nil {
			t.Fatal(err)
		}
		return hostinstall.HostResult{Ready: true, UID: 942, RunnerID: "runner-" + strings.Repeat("a", 32), Mode: "development-managed", SSHPort: 22222, HostPublicKey: strings.TrimSpace(string(pub)), OperatorFingerprint: grant.Fingerprint}, nil
	}
	d.probe = func(context.Context, collection.Connection) error { return nil }
	d.connect = func(ctx context.Context, path string, ssh client.SSHConfig, host hostinstall.HostResult) (collection.Connection, error) {
		if path != config {
			t.Fatal("wrong registry")
		}
		if _, err := collection.Open(path); err != nil {
			t.Fatal(err)
		}
		c := collection.Connection{Name: "local", Transport: "ssh", SSH: &ssh, RunnerID: host.RunnerID, Mode: host.Mode, Revision: 1, EnrolledAt: time.Now().UTC()}
		data, err := json.Marshal(map[string]any{"version": 1, "connections": []collection.Connection{c}})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "runners.json"), data, 0600); err != nil {
			t.Fatal(err)
		}
		return c, nil
	}
	return d, dir, config, runner, calls
}

func TestSetupFreshFlowAndVerifiedNoOpRerun(t *testing.T) {
	d, dir, config, runner, calls := setupFixture(t)
	var events []Event
	s, err := setup(context.Background(), config, dir, runner, d, func(e Event) { events = append(events, e) })
	if err != nil || !s.Ready || s.Connection == nil || *calls != 1 {
		t.Fatal(s, err)
	}
	key, err := readPrivate(filepath.Join(dir, "id_ed25519"), 16384)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(config, "runners.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range events {
		data, _ := json.Marshal(e)
		if bytes.Contains(data, key) || bytes.Contains(data, []byte("PRIVATE KEY")) {
			t.Fatal("private key leaked into progress")
		}
	}
	s, err = setup(context.Background(), config, dir, runner, d, nil)
	if err != nil || !s.Ready || *calls != 1 {
		t.Fatal("ready setup requested authorization again", s, err)
	}
	after, err := os.ReadFile(filepath.Join(config, "runners.json"))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("rerun changed enrollment")
	}
	after, err = readPrivate(filepath.Join(dir, "id_ed25519"), 16384)
	if err != nil || !bytes.Equal(key, after) {
		t.Fatal("rerun changed private key")
	}
}

func TestSetupFailureDoesNotClaimReadinessOrReplaceIdentity(t *testing.T) {
	for _, fault := range []string{"prerequisite", "download", "authorization", "identity", "connect"} {
		t.Run(fault, func(t *testing.T) {
			d, dir, config, runner, calls := setupFixture(t)
			switch fault {
			case "prerequisite":
				d.inspect = func() hostsetup.Report {
					return hostsetup.Report{Checks: []hostsetup.Check{{ID: "docker-version", Status: "blocked"}}}
				}
			case "download":
				d.prepare = func(context.Context, string) (rootless.Manifest, error) {
					return rootless.Manifest{}, errors.New("network failed")
				}
			case "authorization":
				d.elevate = func(context.Context, string, string, string, func(Event)) (hostinstall.HostResult, error) {
					return hostinstall.HostResult{}, errors.New("cancelled")
				}
			case "identity":
				elevate := d.elevate
				d.elevate = func(ctx context.Context, a, b, c string, emit func(Event)) (hostinstall.HostResult, error) {
					r, e := elevate(ctx, a, b, c, emit)
					r.OperatorFingerprint = "wrong"
					return r, e
				}
			case "connect":
				d.connect = func(context.Context, string, client.SSHConfig, hostinstall.HostResult) (collection.Connection, error) {
					return collection.Connection{}, errors.New("SSH unavailable")
				}
			}
			if s, err := setup(context.Background(), config, dir, runner, d, nil); err == nil || s.Ready {
				t.Fatal("failed setup reported ready", s, err)
			}
			if _, err := os.Stat(filepath.Join(dir, "ready.json")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("failed setup published receipt")
			}
			if fault == "prerequisite" && *calls != 0 {
				t.Fatal("blocked host reached authorization")
			}
		})
	}
}

func TestSetupRetryAfterAuthorizationCancellationKeepsKey(t *testing.T) {
	d, dir, config, runner, _ := setupFixture(t)
	elevate := d.elevate
	d.elevate = func(context.Context, string, string, string, func(Event)) (hostinstall.HostResult, error) {
		return hostinstall.HostResult{}, errors.New("cancelled")
	}
	if _, err := setup(context.Background(), config, dir, runner, d, nil); err == nil {
		t.Fatal("cancellation ignored")
	}
	key, err := readPrivate(filepath.Join(dir, "id_ed25519"), 16384)
	if err != nil {
		t.Fatal(err)
	}
	d.elevate = elevate
	if s, err := setup(context.Background(), config, dir, runner, d, nil); err != nil || !s.Ready {
		t.Fatal(s, err)
	}
	again, err := readPrivate(filepath.Join(dir, "id_ed25519"), 16384)
	if err != nil || !bytes.Equal(key, again) {
		t.Fatal("retry rotated desktop identity")
	}
}

func TestSetupRefusesChangedHostIdentity(t *testing.T) {
	d, dir, config, runner, _ := setupFixture(t)
	if _, err := setup(context.Background(), config, dir, runner, d, nil); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(config, "runners.json"))
	if err != nil {
		t.Fatal(err)
	}
	d.probe = func(context.Context, collection.Connection) error { return errors.New("offline") }
	elevate := d.elevate
	d.elevate = func(ctx context.Context, a, b, c string, emit func(Event)) (hostinstall.HostResult, error) {
		r, e := elevate(ctx, a, b, c, emit)
		r.RunnerID = "runner-" + strings.Repeat("b", 32)
		return r, e
	}
	if _, err := setup(context.Background(), config, dir, runner, d, nil); err == nil || !strings.Contains(err.Error(), "identity changed") {
		t.Fatal("replaced pinned local identity", err)
	}
	after, err := os.ReadFile(filepath.Join(config, "runners.json"))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("existing enrollment changed")
	}
}

func TestPrivateSetupStorageAndSSHConfiguration(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "setup with space%")
	if err := privateDirectory(dir); err != nil {
		t.Fatal(err)
	}
	unlock, err := lockSetup(dir)
	if err != nil {
		t.Fatal(err)
	}
	if other, err := lockSetup(dir); err == nil {
		other()
		t.Fatal("concurrent setup admitted")
	}
	unlock()
	path := filepath.Join(dir, "keep")
	if err := os.WriteFile(path, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := writePrivate(path, []byte("replace")); err == nil {
		t.Fatal("unsafe file overwritten")
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := readPrivate(link, 100); err == nil {
		t.Fatal("symlink followed")
	}
	config, err := writeSSH(dir, "ssh-ed25519 fixture")
	if err != nil {
		t.Fatal(err)
	}
	data, err := readPrivate(config.ConfigFile, 16384)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"IdentityAgent none", "IdentitiesOnly yes", "StrictHostKeyChecking yes", "space%%", "GlobalKnownHostsFile /dev/null"} {
		if !strings.Contains(string(data), want) {
			t.Fatal("missing SSH boundary", want)
		}
	}
}

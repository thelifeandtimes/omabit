package hostinstall

import (
	"context"
	"errors"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostsetup"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
)

func nativePackageFixture(t *testing.T) (*fixture, ownerhost.Profile, hostProbe, *[][]string) {
	t.Helper()
	f, p, probe := ownerFixture(t)
	f.i.installPrerequisites = true
	f.put("/usr/bin/pacman", "inert package manager", 0755)
	for _, name := range []string{"rootlesskit", "slirp4netns"} {
		if err := os.Remove(f.i.path("/usr/bin/" + name)); err != nil {
			t.Fatal(err)
		}
	}
	inspect := f.i.inspectOwner
	f.i.inspectOwner = func(name string) hostsetup.Report {
		r := inspect(name)
		r.Compatibility = &hostcompat.Report{Distribution: hostcompat.Distribution{ID: "omarchy", Provider: "pacman"}, InstalledEngineVersion: "29.9.2"}
		for _, name := range []string{"rootlesskit", "slirp4netns"} {
			status := "pass"
			if _, err := os.Stat(f.i.path("/usr/bin/" + name)); errors.Is(err, os.ErrNotExist) {
				status = "missing"
			}
			r.Checks = append(r.Checks, hostsetup.Check{ID: "binary-" + name, Status: status, Detail: name})
		}
		return r
	}
	transactions := new([][]string)
	run := f.i.run
	f.i.run = func(ctx context.Context, path string, args, env []string) ([]byte, error) {
		if path != "/usr/bin/pacman" {
			return run(ctx, path, args, env)
		}
		if slices.Equal(args, []string{"-Qq"}) {
			installed := "docker\ncontainerd\nrunc\nshadow\nopenssh\n"
			for _, name := range []string{"rootlesskit", "slirp4netns"} {
				if _, err := os.Stat(f.i.path("/usr/bin/" + name)); err == nil {
					installed += name + "\n"
				}
			}
			return []byte(installed), nil
		}
		if len(args) < 5 || !slices.Equal(args[:4], []string{"-S", "--needed", "--noconfirm", "--"}) || strings.Contains(strings.Join(env, " "), "DOCKER_HOST=") {
			t.Fatal("unexpected package transaction", args, env)
		}
		*transactions = append(*transactions, slices.Clone(args[4:]))
		for _, name := range args[4:] {
			if !slices.Contains([]string{"rootlesskit", "slirp4netns"}, name) {
				t.Fatal("unexpected package", name)
			}
			f.put("/usr/bin/"+name, "inert installed executable", 0755)
		}
		return nil, nil
	}
	return f, p, probe, transactions
}

func TestAuthorizedPackagesThenVerificationThenOwnerSetup(t *testing.T) {
	f, p, probe, transactions := nativePackageFixture(t)
	compat := f.i.inspectCompatibility
	f.i.inspectCompatibility = func() hostcompat.Report {
		if len(*transactions) != 1 {
			t.Fatal("package authentication ran before prerequisites were installed")
		}
		return compat()
	}
	var steps []string
	r, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("runner"), p, probe, func(step string) { steps = append(steps, step) })
	if err != nil || !r.Ready || !reflect.DeepEqual(*transactions, [][]string{{"rootlesskit", "slirp4netns"}}) || !slices.Equal(steps[:2], []string{"packages", "package-verify"}) {
		t.Fatal(r, err, steps, *transactions)
	}
	r2, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("runner"), p, probe, nil)
	if err != nil || !r2.Ready || r2.RunnerID != r.RunnerID || r2.HostPublicKey != r.HostPublicKey || len(*transactions) != 1 {
		t.Fatal("retry installed packages again or replaced identity", r2, err, *transactions)
	}
}

func TestPackageFailurePreservesPartialWorkAndRetryReplans(t *testing.T) {
	f, p, probe, transactions := nativePackageFixture(t)
	run := f.i.run
	f.i.run = func(ctx context.Context, path string, args, env []string) ([]byte, error) {
		if path == "/usr/bin/pacman" && args[0] == "-S" {
			f.put("/usr/bin/rootlesskit", "inert installed executable", 0755)
			return nil, errors.New("download interrupted")
		}
		return run(ctx, path, args, env)
	}
	r, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("runner"), p, probe, nil)
	if err == nil || r.Ready || !r.Partial {
		t.Fatal(r, err)
	}
	if _, err := os.Stat(f.i.path(ownerhost.Path)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("failed package transaction reached host activation")
	}
	f.i.run = run
	r, err = f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("runner"), p, probe, nil)
	if err != nil || !r.Ready || !reflect.DeepEqual(*transactions, [][]string{{"slirp4netns"}}) {
		t.Fatal("retry did not preserve completed work", r, err, *transactions)
	}
}

func TestPackageSetupRefusalsDoNotActivateServices(t *testing.T) {
	for _, fault := range []string{"unapproved", "bundle", "owner", "legacy", "unsupported", "kernel", "damaged", "database", "lock", "false-success", "unsafe-manager", "provenance"} {
		t.Run(fault, func(t *testing.T) {
			f, p, probe, transactions := nativePackageFixture(t)
			run, inspect := f.i.run, f.i.inspectOwner
			switch fault {
			case "unapproved":
				f.i.installPrerequisites = false
			case "bundle":
				f.i.verified = func(string) (map[string][]byte, error) { return nil, errors.New("bad bundle") }
			case "owner":
				p.UID = 0
			case "legacy":
				f.mkdir("/var/lib/urbit-runner", 0700)
			case "unsupported", "kernel":
				f.i.inspectOwner = func(name string) hostsetup.Report {
					r := inspect(name)
					if fault == "unsupported" {
						r.Compatibility.Distribution.ID = "other"
					} else {
						r.Checks = append(r.Checks, hostsetup.Check{ID: "userns-clone", Status: "blocked", Detail: "disabled"})
					}
					return r
				}
			case "unsafe-manager":
				f.metadata["/usr/bin/pacman"] = fakeInfo{uid: 1000, mode: 0755}
			case "provenance":
				f.i.inspectCompatibility = func() hostcompat.Report { return hostcompat.Report{} }
				f.i.verifyCompatibilityWithOptions = func(context.Context, hostcompat.VerificationOptions) (hostcompat.Report, error) {
					return hostcompat.Report{}, errors.New("payload authentication failed")
				}
			default:
				f.i.run = func(ctx context.Context, path string, args, env []string) ([]byte, error) {
					if path == "/usr/bin/pacman" {
						if fault == "damaged" && args[0] == "-Qq" {
							return []byte("rootlesskit\n"), nil
						}
						if fault == "database" || fault == "lock" && args[0] == "-S" {
							return nil, errors.New("package manager unavailable or locked")
						}
						if fault == "false-success" && args[0] == "-S" {
							return nil, nil
						}
					}
					return run(ctx, path, args, env)
				}
			}
			r, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("runner"), p, probe, nil)
			if err == nil || r.Ready || fault != "provenance" && len(*transactions) != 0 {
				t.Fatal("refused host proceeded", r, err, *transactions)
			}
			for _, c := range f.calls {
				if c.path == "/usr/bin/systemctl" || c.path == "/usr/bin/usermod" {
					t.Fatal("failed package preparation reached service/account setup", c)
				}
			}
		})
	}
}

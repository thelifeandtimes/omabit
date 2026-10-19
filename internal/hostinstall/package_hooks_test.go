package hostinstall

import (
	"bytes"
	"context"
	"errors"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

func TestPackageHooksInstallOnlyAfterActivatedTrustedRunner(t *testing.T) {
	for _, provider := range []string{"pacman", "deb"} {
		t.Run(provider, func(t *testing.T) {
			f, p, probe := ownerFixture(t)
			f.i.inspectCompatibility = func() hostcompat.Report {
				return hostcompat.Report{InstalledEngineVersion: "29.9.1", Provenance: "verified", Distribution: hostcompat.Distribution{Provider: provider}}
			}
			// The old/current path must not be used to install hooks before the
			// ordinary quiescence-aware management activation publishes this build.
			before := tree(t, f.i.root)
			if err := f.i.installPackageHooks(context.Background(), []byte("new runner")); err == nil {
				t.Fatal("installed hooks without current runner")
			}
			if !reflect.DeepEqual(before, tree(t, f.i.root)) {
				t.Fatal("early installation changed files")
			}
			f.beforeCommand = func(c call) {
				if c.path != "/usr/bin/pacman-conf" && c.path != "/usr/bin/apt-config" {
					return
				}
				current, err := os.ReadFile(f.i.path(access.ShellPath))
				if err != nil || string(current) != "new runner" {
					t.Fatal("hooks before current generation activation")
				}
			}
			for range 2 {
				if _, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("new runner"), p, probe, nil); err != nil {
					t.Fatal(err)
				}
			}
			files, _ := rootless.PackageHookFiles(provider)
			for path, want := range files {
				got, err := f.i.readInstalled(context.Background(), path, maxOutput)
				if err != nil || !bytes.Equal(got, want) {
					t.Fatal(path, err)
				}
			}
			for _, c := range f.calls {
				if slices.Contains(c.args, "verify-host-packages") {
					t.Fatal("installer unnecessarily executed hook verifier")
				}
			}
		})
	}
}

func TestPackageHooksRefuseUnknownProvidersLocalEditsAndUnsafePaths(t *testing.T) {
	for _, fault := range []string{"unknown", "old-runner", "non-executable-runner", "edited-wrapper", "edited-hook", "symlink", "writable", "custom-directory", "custom-cache", "earlier-cleanup"} {
		t.Run(fault, func(t *testing.T) {
			f := newFixture(t)
			f.put(access.ShellPath, "new runner", 0755)
			switch fault {
			case "unknown":
				f.i.inspectCompatibility = func() hostcompat.Report {
					return hostcompat.Report{Distribution: hostcompat.Distribution{Provider: "unknown"}}
				}
			case "old-runner":
				f.put(access.ShellPath, "old runner without verification CLI", 0755)
			case "non-executable-runner":
				f.put(access.ShellPath, "new runner", 0644)
			case "edited-wrapper":
				f.put(rootless.PackageRefreshPath, "#!/bin/sh\n# administrator content\n", 0755)
			case "edited-hook":
				f.put(rootless.PacmanPackageHookPath, "# administrator content\n", 0644)
			case "symlink":
				f.mkdir(rootless.PackageHookDir, 0755)
				if err := os.Symlink(access.ShellPath, f.i.path(rootless.PackageRefreshPath)); err != nil {
					t.Fatal(err)
				}
			case "writable":
				f.mkdir(rootless.PackageHookDir, 0777)
			case "custom-directory":
				run := f.i.run
				f.i.run = func(ctx context.Context, path string, args, env []string) ([]byte, error) {
					if path == "/usr/bin/pacman-conf" {
						return []byte("/unreviewed/hooks\n"), nil
					}
					return run(ctx, path, args, env)
				}
			case "custom-cache":
				run := f.i.run
				f.i.run = func(ctx context.Context, path string, args, env []string) ([]byte, error) {
					if path == "/usr/bin/pacman-conf" && slices.Equal(args, []string{"CacheDir"}) {
						return []byte("/custom/cache\n"), nil
					}
					return run(ctx, path, args, env)
				}
			case "earlier-cleanup":
				f.put("/etc/pacman.d/hooks/00-cleanup.hook", "[Action]\nWhen = PostTransaction\nExec = /unrelated/admin/cleanup\n", 0644)
			}
			before := tree(t, f.i.root)
			if err := f.i.installPackageHooks(context.Background(), []byte("new runner")); err == nil {
				t.Fatal("accepted", fault)
			}
			if !reflect.DeepEqual(before, tree(t, f.i.root)) {
				t.Fatal("changed host before refusal", fault)
			}
		})
	}
}

func TestKnownPackageHookMigrationAndInterruptedPublication(t *testing.T) {
	f := newFixture(t)
	f.put(access.ShellPath, "new runner", 0755)
	files, _ := rootless.PackageHookFiles("pacman")
	// Test the exact-content migration seam without inventing a released older
	// hook version. Future actual versions populate the compiled-in allowlist.
	previous := []byte("known earlier generated fixture\n")
	f.put(rootless.PackageRefreshPath, string(previous), 0755)
	if err := f.i.reviewPackageHook(context.Background(), rootless.PackageRefreshPath, files[rootless.PackageRefreshPath], [][]byte{previous}); err != nil {
		t.Fatal(err)
	}
	if err := f.i.reviewPackageHook(context.Background(), rootless.PackageRefreshPath, files[rootless.PackageRefreshPath], nil); err == nil {
		t.Fatal("unknown previous bytes accepted")
	}
	if err := os.Remove(f.i.path(rootless.PackageRefreshPath)); err != nil {
		t.Fatal(err)
	}
	f.i.syncDirectory = func(path string) error {
		if path == f.i.path("/etc/pacman.d/hooks") {
			return errors.New("lost publication acknowledgement")
		}
		return nil
	}
	if err := f.i.installPackageHooks(context.Background(), []byte("new runner")); err == nil {
		t.Fatal("lost publication acknowledgement accepted")
	}
	f.i.syncDirectory = nil
	if err := f.i.installPackageHooks(context.Background(), []byte("new runner")); err != nil {
		t.Fatal("interrupted hook install did not resume", err)
	}
}

func TestAPTEffectiveConfigurationAndCleanupOrdering(t *testing.T) {
	base := "Dir \"/\";\nDir::Etc \"etc/apt\";\nDir::Etc::parts \"apt.conf.d\";\nDir::Cache \"var/cache/apt\";\nDir::Cache::archives \"archives/\";\n"
	ours := "DPkg::Post-Invoke:: \"" + rootless.PackageRefreshPath + " || true\";\n"
	cleanup := "DPkg::Post-Invoke:: \"/fixed/cache-cleanup\";\n"
	// This named-child shape/order is emitted by native APT2.6.1 apt-config
	// and executed by RunScripts, as the independent private-root probe showed.
	namedCleanup := "DPkg::Post-Invoke::cleanup \"/fixed/cache-cleanup\";\n"
	for _, tc := range []struct {
		name, data    string
		installed, ok bool
	}{
		{"before-install", base + cleanup, false, true},
		{"early-refresh", base + ours + cleanup, true, true},
		{"case-aliases", strings.ReplaceAll(strings.ReplaceAll(base+ours, "Dir::Cache", "dir::cache"), "archives \"", "Archives \""), true, true},
		{"absolute-alias", strings.Replace(strings.Replace(base, "etc/apt", "/etc/apt", 1), "apt.conf.d", "/etc/apt/apt.conf.d", 1) + ours, true, true},
		{"late-refresh", base + cleanup + ours, true, false},
		{"named-cleanup-before-refresh", base + "DPkg::Post-Invoke \"\";\n" + namedCleanup + ours, true, false},
		{"named-cleanup-after-refresh", base + ours + namedCleanup, true, true},
		{"named-refresh-first", base + strings.Replace(ours, "Post-Invoke:: ", "Post-Invoke::refresh ", 1) + namedCleanup, true, true},
		{"empty-named-child-not-executed", base + "DPkg::Post-Invoke::empty \"\";\n" + ours + namedCleanup, true, true},
		{"grandchild-not-executed", base + "DPkg::Post-Invoke::group \"\";\nDPkg::Post-Invoke::group::nested \"/not-executed\";\n" + ours, true, true},
		{"missing-refresh-named-only", base + namedCleanup, true, false},
		{"parent-value-not-a-hook", base + "DPkg::Post-Invoke \"" + rootless.PackageRefreshPath + " || true\";\n" + namedCleanup, true, false},
		{"cleared-hook", base + cleanup, true, false},
		{"other-directory", strings.Replace(base, "apt.conf.d", "other.d", 1), false, false},
		{"other-cache", strings.Replace(base, "var/cache/apt", "var/cache/custom", 1), false, false},
		{"alternate-root", base + "RootDir \"/target\";\n", false, false},
		{"metadata-command", strings.Replace(base, "apt.conf.d", "$(touch injected)", 1), false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateAPTConfiguration([]byte(tc.data), tc.installed); (err == nil) != tc.ok {
				t.Fatal(err)
			}
		})
	}
}

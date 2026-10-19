package hostinstall

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

func TestAuthorizedInstallerRefreshesNativeProofBeforeChanges(t *testing.T) {
	for _, status := range []string{"metadata-only", "unknown", "verified", "refresh-failure", "nonroot"} {
		t.Run(status, func(t *testing.T) {
			f := newFixture(t)
			refreshes := 0
			f.i.inspectCompatibility = func() hostcompat.Report {
				return hostcompat.Report{InstalledEngineVersion: "29.9.1", Provenance: status}
			}
			f.i.verifyCompatibility = func(context.Context) (hostcompat.Report, error) {
				refreshes++
				if f.i.mutated || len(f.calls) != 0 {
					t.Fatal("verification after host mutation or execution")
				}
				if status == "refresh-failure" {
					return hostcompat.Report{}, errors.New("unavailable authenticated archive")
				}
				return hostcompat.Report{InstalledEngineVersion: "29.9.1", Provenance: "verified"}, nil
			}
			if status == "nonroot" {
				f.i.euid = 1000
			}
			before := tree(t, f.i.root)
			_, err := f.i.install(context.Background(), "/bundle")
			if status == "refresh-failure" || status == "nonroot" {
				if err == nil || !reflect.DeepEqual(before, tree(t, f.i.root)) || len(f.calls) != 0 {
					t.Fatal("failed proof/auth changed host", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			want := 1
			if status == "nonroot" || status == "verified" {
				want = 0
			}
			if refreshes != want {
				t.Fatal("wrong refresh count", refreshes)
			}
		})
	}
}

func TestGuardRepairRegisteredOwnerRepeatedAndStoppedEngine(t *testing.T) {
	f, p, probe := ownerFixture(t)
	service, err := rootless.ServiceFiles()
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range service {
		f.files[name] = data
	}
	launcher, err := os.ReadFile("../rootless/assets/upstream/dockerd-rootless.sh")
	if err != nil {
		t.Fatal(err)
	}
	f.files["dockerd-rootless.sh"] = launcher
	if _, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("runner"), p, probe, nil); err != nil {
		t.Fatal(err)
	}
	previous, err := rootless.PreviousServiceFiles()
	if err != nil {
		t.Fatal(err)
	}
	f.put(rootless.InstallDir+"/check-engine.sh", string(ownerEngineCheck(previous["check-engine.sh"], p)), 0755)
	f.put(unitPath, string(ownerEngineUnit(previous[rootless.UnitName], p)), 0644)
	f.put(p.Home+"/.local/share/urbit-runner/state.db", "registered instances, history, held update fence", 0600)
	f.put(p.Home+"/.local/share/urbit-runner/docker/identity", "existing data-root", 0600)
	f.put(p.Piers+"/ship/data", "existing pier", 0600)
	before := tree(t, f.i.root)
	f.i.lookupUser = func(name string) (*user.User, error) {
		if name != p.Name {
			t.Fatal(name)
		}
		return &user.User{Username: p.Name, Uid: "942", Gid: "942", HomeDir: p.Home}, nil
	}
	f.calls = nil
	f.i.inspectCompatibility = func() hostcompat.Report {
		return hostcompat.Report{InstalledEngineVersion: "29.9.1", Provenance: "verified"}
	}
	for attempt := 0; attempt < 2; attempt++ {
		result, err := f.i.repairEngineGuard(context.Background())
		if err != nil || result.Partial || result.Updated != (attempt == 0) || result.Reloaded {
			t.Fatal(result, err)
		}
	}
	for _, call := range f.calls {
		if call.path != "/usr/bin/systemctl" || len(call.args) == 0 || call.args[0] != "show" {
			t.Fatal("repair ran engine or management command", call)
		}
	}
	after := tree(t, f.i.root)
	for _, path := range []string{rootless.InstallDir + "/check-engine.sh", unitPath} {
		delete(before, f.i.path(path))
		delete(after, f.i.path(path))
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("guard repair altered registered resources or state")
	}
	guard, _ := os.ReadFile(f.i.path(rootless.InstallDir + "/check-engine.sh"))
	if !bytes.Equal(guard, ownerEngineCheck(service["check-engine.sh"], p)) {
		t.Fatal("guard not migrated")
	}
}

func TestEngineGuardMigrationRefusesLocalEditsBeforeEitherReplacement(t *testing.T) {
	for _, changed := range []string{"check-engine.sh", rootless.UnitName} {
		t.Run(changed, func(t *testing.T) {
			f, p, _ := ownerFixture(t)
			old, _ := rootless.PreviousServiceFiles()
			current, _ := rootless.ServiceFiles()
			guard, unit := ownerEngineCheck(old["check-engine.sh"], p), ownerEngineUnit(old[rootless.UnitName], p)
			if changed == "check-engine.sh" {
				guard = append(guard, []byte("# local edit\n")...)
			} else {
				unit = append(unit, []byte("# local edit\n")...)
			}
			f.put(rootless.InstallDir+"/check-engine.sh", string(guard), 0755)
			f.put(unitPath, string(unit), 0644)
			before := tree(t, f.i.root)
			if _, err := f.i.updateEngineGuard(context.Background(), current, &p, false); err == nil {
				t.Fatal("accepted local edit")
			}
			if !reflect.DeepEqual(before, tree(t, f.i.root)) {
				t.Fatal("changed files before refusal")
			}
		})
	}
}

func TestPreviousOwnerGuardMigratesDuringRepeatedInstallWithoutEngineRestart(t *testing.T) {
	f, p, probe := ownerFixture(t)
	current, _ := rootless.ServiceFiles()
	for name, data := range current {
		f.files[name] = data
	}
	if _, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("runner"), p, probe, nil); err != nil {
		t.Fatal(err)
	}
	old, _ := rootless.PreviousServiceFiles()
	f.put(rootless.InstallDir+"/check-engine.sh", string(ownerEngineCheck(old["check-engine.sh"], p)), 0755)
	f.put(unitPath, string(ownerEngineUnit(old[rootless.UnitName], p)), 0644)
	f.calls = nil
	for range 2 {
		if _, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("runner"), p, probe, nil); err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range f.calls {
		if c.path == "/usr/bin/systemctl" && slices.Contains(c.args, rootless.UnitName) && (slices.Contains(c.args, "--now") || slices.Contains(c.args, "start") || slices.Contains(c.args, "restart") || slices.Contains(c.args, "stop")) {
			t.Fatal("engine activation during guard migration", c)
		}
	}
	guard, _ := os.ReadFile(f.i.path(rootless.InstallDir + "/check-engine.sh"))
	if !bytes.Equal(guard, ownerEngineCheck(current["check-engine.sh"], p)) {
		t.Fatal("guard not updated during registered install")
	}
}

func TestGuardMigrationResumesAfterUnitPublicationFailure(t *testing.T) {
	f, p, _ := ownerFixture(t)
	old, _ := rootless.PreviousServiceFiles()
	current, _ := rootless.ServiceFiles()
	f.put(rootless.InstallDir+"/check-engine.sh", string(ownerEngineCheck(old["check-engine.sh"], p)), 0755)
	f.put(unitPath, string(ownerEngineUnit(old[rootless.UnitName], p)), 0644)
	f.i.syncDirectory = func(path string) error {
		if path == f.i.path(filepath.Dir(unitPath)) {
			return errors.New("lost directory sync")
		}
		return nil
	}
	if _, err := f.i.updateEngineGuard(context.Background(), current, &p, false); err == nil {
		t.Fatal("lost publication acknowledgement accepted")
	}
	f.i.syncDirectory = nil
	if _, err := f.i.updateEngineGuard(context.Background(), current, &p, false); err != nil {
		t.Fatal("interrupted update did not resume", err)
	}
	if len(f.calls) != 0 {
		t.Fatal("migration invoked a service")
	}
}

func TestDedicatedPreviousGuardAndStopTimeoutMigrate(t *testing.T) {
	for _, stop := range []string{"360", "120"} {
		t.Run(stop, func(t *testing.T) {
			f := newFixture(t)
			old, _ := rootless.PreviousServiceFiles()
			current, _ := rootless.ServiceFiles()
			f.put(rootless.InstallDir+"/check-engine.sh", string(old["check-engine.sh"]), 0755)
			f.put(unitPath, strings.ReplaceAll(string(old[rootless.UnitName]), "TimeoutStopSec=360", "TimeoutStopSec="+stop), 0644)
			for attempt := 0; attempt < 2; attempt++ {
				updated, err := f.i.updateEngineGuard(context.Background(), current, nil, false)
				if err != nil || updated != (attempt == 0) {
					t.Fatal(updated, err)
				}
			}
			if len(f.calls) != 0 {
				t.Fatal("migration invoked services")
			}
		})
	}
}

func TestDebianNativeAliasesAndAlternatives(t *testing.T) {
	f, p, probe := ownerFixture(t)
	f.i.inspectCompatibility = func() hostcompat.Report {
		return hostcompat.Report{InstalledEngineVersion: rootless.EngineVersion, Provenance: "verified", Distribution: hostcompat.Distribution{Provider: "deb"}}
	}
	service, _ := rootless.ServiceFiles()
	for name, data := range service {
		f.files[name] = data
	}
	for _, name := range []string{"dockerd", "runuser", "useradd", "usermod", "nologin", "sshd", "iptables"} {
		f.mkdir("/usr/sbin", 0755)
		if err := os.Rename(f.i.path("/usr/bin/"+name), f.i.path("/usr/sbin/"+name)); err != nil {
			t.Fatal(err)
		}
	}
	f.mkdir("/etc/alternatives", 0755)
	f.put("/usr/sbin/xtables-nft-multi", "inert native iptables", 0755)
	if err := os.Remove(f.i.path("/usr/sbin/iptables")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/etc/alternatives/iptables", f.i.path("/usr/sbin/iptables")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/usr/sbin/xtables-nft-multi", f.i.path("/etc/alternatives/iptables")); err != nil {
		t.Fatal(err)
	}
	run := f.i.run
	f.i.run = func(ctx context.Context, path string, args, env []string) ([]byte, error) {
		return run(ctx, strings.Replace(path, "/usr/sbin/", "/usr/bin/", 1), args, env)
	}
	for range 2 {
		if _, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("runner"), p, probe, nil); err != nil {
			t.Fatal(err)
		}
	}
	unit, _ := os.ReadFile(f.i.path(unitPath))
	if !bytes.Contains(unit, []byte("Environment=DOCKERD=/usr/sbin/dockerd")) {
		t.Fatal("wrong native daemon path")
	}
	ssh, _ := os.ReadFile(f.i.path(sshUnit))
	if !bytes.Contains(ssh, []byte("/usr/sbin/sshd")) {
		t.Fatal("wrong native sshd path")
	}
	if err := os.Remove(f.i.path("/etc/alternatives/iptables")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/home/desktop/evil", f.i.path("/etc/alternatives/iptables")); err != nil {
		t.Fatal(err)
	}
	if err := f.i.helper(context.Background(), "/usr/bin/iptables"); err == nil {
		t.Fatal("arbitrary alternatives target admitted")
	}
	if _, err := f.i.resolveHelper(context.Background(), filepath.Join(p.Home, "dockerd")); err == nil {
		t.Fatal("arbitrary executable admitted")
	}
}

func TestCompatibleEngineAPIBoundsStillEnforced(t *testing.T) {
	for _, api := range []string{"1.46", "1.47", "1.54"} {
		f := newFixture(t)
		var info map[string]any
		json.Unmarshal(f.info, &info)
		info["ServerVersion"] = "29.9.1"
		f.info, _ = json.Marshal(info)
		run := f.i.run
		f.i.run = func(ctx context.Context, path string, args, env []string) ([]byte, error) {
			if path == "/usr/bin/runuser" && slices.Contains(args, "version") {
				return json.Marshal(map[string]string{"Version": "29.9.1", "APIVersion": api, "MinAPIVersion": "1.44"})
			}
			return run(ctx, path, args, env)
		}
		_, err := f.i.install(context.Background(), "/bundle")
		if (err == nil) != (api != "1.46") {
			t.Fatal(api, err)
		}
	}
}

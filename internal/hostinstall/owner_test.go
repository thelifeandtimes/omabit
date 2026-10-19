package hostinstall

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostsetup"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

func ownerFixture(t *testing.T) (*fixture, ownerhost.Profile, hostProbe) {
	f, probe := hostFixture(t)
	p := ownerhost.Profile{Version: 1, Name: "desktop", UID: testUID, GID: testUID, Home: "/home/desktop", Piers: ownerhost.DefaultPiers}
	f.mkdir(p.Home, 0755)
	f.metadata[p.Home] = fakeInfo{uid: p.UID, gid: p.GID, mode: os.ModeDir | 0755}
	f.put("/etc/passwd", "root:x:0:0::/root:/bin/bash\ndesktop:x:942:942::/home/desktop:/bin/bash\n", 0644)
	f.put("/etc/group", "root:x:0:\ndesktop:x:942:\nwheel:x:998:desktop\n", 0644)
	f.put("/etc/subuid", "desktop:100000:65536\n", 0644)
	f.put("/etc/subgid", "desktop:100000:65536\n", 0644)
	f.files[rootless.UnitName] = []byte("ConditionUser=urbit-runner\nEnvironment=HOME=/var/lib/urbit-runner\n")
	var info map[string]any
	json.Unmarshal(f.info, &info)
	info["DockerRootDir"] = p.Home + "/.local/share/urbit-runner/docker"
	f.info, _ = json.Marshal(info)
	f.i.inspectOwner = func(name string) hostsetup.Report {
		if name != p.Name {
			t.Fatal(name)
		}
		r := hostsetup.Report{}
		for _, id := range []string{"platform", "systemd", "cgroup-v2", "userns-limit", "owner-account", "docker-version"} {
			r.Checks = append(r.Checks, hostsetup.Check{ID: id, Status: "pass"})
		}
		return r
	}
	f.i.chownOwner = func(path string, uid, gid uint32) error {
		rel, err := filepath.Rel(f.i.root, path)
		if err != nil || strings.HasPrefix(rel, "..") {
			t.Fatal("outside fixture", path)
		}
		f.metadata["/"+rel] = fakeInfo{uid: uid, gid: gid, mode: os.ModeDir | 0755}
		return nil
	}
	run := f.i.run
	f.i.run = func(ctx context.Context, path string, args, env []string) ([]byte, error) {
		if path == "/usr/bin/usermod" {
			if len(args) != 3 || args[2] != p.Name || !slices.Contains([]string{"--add-subuids", "--add-subgids"}, args[0]) {
				t.Fatal("changed login account", args)
			}
			f.calls = append(f.calls, call{path, slices.Clone(args), slices.Clone(env)})
			file := "/etc/subuid"
			if args[0] == "--add-subgids" {
				file = "/etc/subgid"
			}
			old, _ := os.ReadFile(f.i.path(file))
			start, _, _ := strings.Cut(args[1], "-")
			f.put(file, string(old)+p.Name+":"+start+":65536\n", 0644)
			return nil, nil
		}
		return run(ctx, path, args, env)
	}
	return f, p, probe
}

func TestOwnerInstallAndRerunPreserveLoginAndGrants(t *testing.T) {
	f, p, probe := ownerFixture(t)
	passwd, _ := os.ReadFile(f.i.path("/etc/passwd"))
	groups, _ := os.ReadFile(f.i.path("/etc/group"))
	first, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("runner one"), p, probe, nil)
	if err != nil || !first.Ready || first.Owner != p.Name || first.UID != p.UID || first.PierRoot != ownerhost.DefaultPiers {
		t.Fatal(first, err)
	}
	observer, err := access.OwnerAuthorizedKey(hostPublic(3), "observer")
	if err != nil {
		t.Fatal(err)
	}
	keys, _ := os.ReadFile(f.i.path(access.DefaultKeys))
	f.put(access.DefaultKeys, string(keys)+observer+"\n", 0644)
	second, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("runner two"), p, probe, nil)
	if err != nil || !second.Ready || second.RunnerID != first.RunnerID || second.HostPublicKey != first.HostPublicKey {
		t.Fatal(second, err)
	}
	for path, want := range map[string][]byte{"/etc/passwd": passwd, "/etc/group": groups, access.DefaultKeys: append(keys, []byte(observer+"\n")...)} {
		got, _ := os.ReadFile(f.i.path(path))
		if !bytes.Equal(got, want) {
			t.Fatal("changed", path)
		}
	}
	for _, c := range f.calls {
		if c.path == "/usr/bin/useradd" || c.path == "/usr/bin/usermod" || slices.Contains(c.args, "/run/docker.sock") {
			t.Fatal("unexpected account/engine change", c)
		}
	}
	unit, _ := os.ReadFile(f.i.path(unitPath))
	if !bytes.Contains(unit, []byte("ConditionUser=desktop")) || !bytes.Contains(unit, []byte(`HOME=%h`)) {
		t.Fatal(string(unit))
	}
	ssh, _ := os.ReadFile(f.i.path(sshConfigPath))
	if !bytes.Contains(ssh, []byte("AllowUsers desktop\n")) {
		t.Fatal(string(ssh))
	}
	if _, err = os.Stat(f.i.path(rootless.HomeDir)); !os.IsNotExist(err) {
		t.Fatal("created legacy home")
	}
}

func TestOwnerInstallAllocatesOnlyMissingSubordinateRanges(t *testing.T) {
	f, p, probe := ownerFixture(t)
	f.put("/etc/subuid", "other:1048576:65536\n", 0644)
	f.put("/etc/subgid", "other:1048576:65536\n", 0644)
	if _, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("runner"), p, probe, nil); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"subuid", "subgid"} {
		data, _ := os.ReadFile(f.i.path("/etc/" + file))
		if !strings.Contains(string(data), "other:1048576:65536\n") || !strings.Contains(string(data), "desktop:1114112:65536\n") {
			t.Fatal(string(data))
		}
	}
}

func TestOwnerInstallRefusesLegacyOtherOwnerAndForeignPierRoot(t *testing.T) {
	for _, fault := range []string{"legacy", "profile", "root", "home", "override"} {
		t.Run(fault, func(t *testing.T) {
			f, p, probe := ownerFixture(t)
			switch fault {
			case "legacy":
				f.mkdir(rootless.HomeDir, 0700)
			case "profile":
				old := p
				old.Name = "another"
				raw, _ := json.Marshal(old)
				f.put(ownerhost.Path, string(raw), 0644)
			case "root":
				f.mkdir(p.Piers, 0755)
			case "home":
				f.metadata[p.Home] = fakeInfo{uid: p.UID, gid: p.GID, mode: os.ModeDir | 0777}
			case "override":
				f.put(p.Home+"/.config/systemd/user/urbit-docker.service", "operator unit", 0644)
			}
			if _, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("runner"), p, probe, nil); err == nil {
				t.Fatal("accepted", fault)
			}
			for _, c := range f.calls {
				if c.path == "/usr/bin/systemctl" || c.path == "/usr/bin/usermod" {
					t.Fatal("mutated refused host", c)
				}
			}
		})
	}
}

func TestSubordinateAllocationAvoidsRealUIDsAndExistingRanges(t *testing.T) {
	start, err := freeSubIDs("other:1048576:65536\n", []uint64{1114113})
	if err != nil || start != 1179648 {
		t.Fatal(start, err)
	}
	if _, err = freeSubIDs("other:"+strconv.FormatUint(1<<32-2, 10)+":5", nil); err == nil {
		t.Fatal("overflow accepted")
	}
}

func TestOwnerEngineCheckUsesRealGuardTemplateAndPreservesLauncherPin(t *testing.T) {
	files, err := rootless.ServiceFiles()
	if err != nil {
		t.Fatal(err)
	}
	p := ownerhost.Profile{Version: 1, Name: "desktop", UID: 1000, GID: 1000, Home: "/home/desktop", Piers: ownerhost.DefaultPiers}
	check := string(ownerEngineCheck(files["check-engine.sh"], p))
	for _, want := range []string{`[ "$(id -un)" = 'desktop' ]`, `[ "${HOME:-}" = '/home/desktop' ]`, `getent passwd 'desktop'`, `[ "$home" = '/home/desktop' ]`, rootless.InstallDir + "/dockerd-rootless.sh", rootless.LauncherSHA256} {
		if !strings.Contains(check, want) {
			t.Fatal("missing owner guard", want, check)
		}
	}
	if strings.Contains(check, "'urbit-runner'") || strings.Contains(check, "'/var/lib/urbit-runner'") {
		t.Fatal("retained legacy account check")
	}
}

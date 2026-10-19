package hostinstall

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostsetup"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

const systemdIdentityNSS = "passwd: files systemd\ngroup: files [SUCCESS=merge] systemd\nshadow: files\n"

func TestInstalledOwnerUpdatePreservesExistingUnknownNSSGrants(t *testing.T) {
	for _, nss := range []string{systemdIdentityNSS, "passwd: files sss\ngroup: files sss\nsubid: files\n"} {
		t.Run(strings.Fields(nss)[2], func(t *testing.T) {
			f, p, probe := ownerFixture(t)
			p.Piers = "/srv/custom-owner-piers"
			first, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("original manager"), p, probe, nil)
			if err != nil || !first.Ready {
				t.Fatal(first, err)
			}
			// Model a previously installed host, including pre-journal profiles.
			if err := os.Remove(f.i.path(ownerEngineActivationPath)); err != nil {
				t.Fatal(err)
			}
			f.put("/etc/nsswitch.conf", nss, 0644)
			grants := map[string]string{
				"/etc/subuid": "# existing delegated grants\nother:200000:65536\ndesktop:100000:32768\n942:132768:32768\n",
				"/etc/subgid": "# preserve order and unrelated owners\n942:300000:65536\nother:400000:65536\n",
			}
			objects := map[string]os.FileInfo{}
			for path, text := range grants {
				f.put(path, text, 0644)
				objects[path], err = os.Stat(f.i.path(path))
				if err != nil {
					t.Fatal(err)
				}
			}
			profileBefore, _ := os.ReadFile(f.i.path(ownerhost.Path))
			inspect := f.i.inspectOwner
			f.i.inspectOwner = func(name string) hostsetup.Report {
				r := inspect(name)
				r.Checks = append(r.Checks, hostsetup.Check{ID: "subuid", Status: "unknown", Detail: "External identity domains remain unknown."}, hostsetup.Check{ID: "subgid", Status: "unknown", Detail: "External identity domains remain unknown."})
				return r
			}
			f.calls = nil
			for range 2 {
				result, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("updated manager"), p, probe, nil)
				if err != nil || !result.Ready || result.Owner != p.Name || result.UID != p.UID || result.PierRoot != p.Piers || result.RunnerID != first.RunnerID || result.HostPublicKey != first.HostPublicKey {
					t.Fatal(result, err)
				}
			}
			for _, c := range f.calls {
				if c.path == "/usr/bin/usermod" || c.path == "/usr/bin/useradd" {
					t.Fatal("management update changed OS grants", c)
				}
				if slices.Contains(c.args, rootless.UnitName) && (slices.Contains(c.args, "--now") || slices.Contains(c.args, "start") || slices.Contains(c.args, "restart") || slices.Contains(c.args, "stop")) {
					t.Fatal("management update changed engine intent", c)
				}
			}
			for path, want := range grants {
				got, err := os.ReadFile(f.i.path(path))
				after, statErr := os.Stat(f.i.path(path))
				if err != nil || statErr != nil || string(got) != want || !os.SameFile(objects[path], after) || after.Mode() != objects[path].Mode() || !after.ModTime().Equal(objects[path].ModTime()) {
					t.Fatal("existing grant object/bytes changed", path, err, statErr)
				}
			}
			if after, _ := os.ReadFile(f.i.path(ownerhost.Path)); !bytes.Equal(after, profileBefore) {
				t.Fatal("installed owner profile changed")
			}
			if after, _ := os.ReadFile(f.i.path("/etc/nsswitch.conf")); string(after) != nss {
				t.Fatal("NSS policy changed")
			}
			if _, err := os.Stat(f.i.path(hostsetup.SubIDReservationsPath)); !os.IsNotExist(err) {
				t.Fatal("reservation authority was invented", err)
			}
			if status, _ := hostsetup.ValidateSubIDAuthority(nss, nil, p.Name, grants["/etc/subuid"], "942", "subuid", []uint64{0, 942}); status != "blocked" {
				t.Fatal("fresh-grant authority was weakened")
			}
		})
	}
}

func TestInstalledOwnerMappingsNeverAllocateMissingGrants(t *testing.T) {
	for _, nss := range []string{"passwd: files\ngroup: files\nsubid: files\n", systemdIdentityNSS} {
		for _, missing := range []string{"subuid", "subgid"} {
			t.Run(strings.Fields(nss)[2]+"/"+missing, func(t *testing.T) {
				f, p, probe := ownerFixture(t)
				if _, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("runner"), p, probe, nil); err != nil {
					t.Fatal(err)
				}
				f.put("/etc/nsswitch.conf", nss, 0644)
				f.put("/etc/"+missing, "other:200000:65536\n", 0644)
				f.put(hostsetup.SubIDReservationsPath, `{"version":1,"authority":"preexisting administrator assertion","uid":{"start":100000,"count":65536},"gid":{"start":100000,"count":65536}}`, 0644)
				before := tree(t, f.i.root)
				f.calls, f.i.mutated = nil, false
				result, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("updated"), p, probe, nil)
				if err == nil || result.Ready || len(f.calls) != 0 || !reflect.DeepEqual(before, tree(t, f.i.root)) {
					t.Fatal("installed profile enabled a replacement grant", result, err)
				}
			})
		}
	}
}

func TestFreshOwnerStillRequiresUnknownNSSReservations(t *testing.T) {
	for _, existingGrants := range []bool{false, true} {
		f, p, probe := ownerFixture(t)
		f.put("/etc/nsswitch.conf", systemdIdentityNSS, 0644)
		if !existingGrants {
			f.put("/etc/subuid", "", 0644)
			f.put("/etc/subgid", "", 0644)
		}
		beforeUID, _ := os.ReadFile(f.i.path("/etc/subuid"))
		beforeGID, _ := os.ReadFile(f.i.path("/etc/subgid"))
		result, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("runner"), p, probe, nil)
		if err == nil || result.Ready || !strings.Contains(err.Error(), "reserv") || len(f.calls) != 0 {
			t.Fatal("fresh unknown-NSS setup bypassed authority", result, err)
		}
		for path, want := range map[string][]byte{"/etc/subuid": beforeUID, "/etc/subgid": beforeGID} {
			if got, _ := os.ReadFile(f.i.path(path)); !bytes.Equal(got, want) {
				t.Fatal("fresh refused setup changed grants", path)
			}
		}
		for _, path := range []string{ownerhost.Path, hostsetup.SubIDReservationsPath} {
			if _, err := os.Stat(f.i.path(path)); !os.IsNotExist(err) {
				t.Fatal("fresh refusal invented authority", path, err)
			}
		}
	}
}

func TestPreservedMappingsRejectUnsafeOrContradictoryEvidence(t *testing.T) {
	for _, fault := range []string{"missing", "short", "malformed", "overlap", "local-uid", "primary-gid", "group-gid", "owner-uid", "owner-gid", "uid-binding", "gid-binding", "home-binding", "duplicate-owner", "profile-owner", "profile-uid", "profile-gid", "profile-home", "profile-root", "profile-version", "profile-field", "grant-owner", "grant-writable", "grant-symlink", "home-owner", "home-writable", "external-subid", "reservation-conflict", "reservation-malformed"} {
		t.Run(fault, func(t *testing.T) {
			f, p, _ := ownerFixture(t)
			profile, _ := json.Marshal(p)
			f.put(ownerhost.Path, string(profile)+"\n", 0644)
			f.put("/etc/nsswitch.conf", systemdIdentityNSS, 0644)
			switch fault {
			case "missing":
				f.put("/etc/subuid", "other:100000:65536\n", 0644)
			case "short":
				f.put("/etc/subuid", "desktop:100000:1\n", 0644)
			case "malformed":
				f.put("/etc/subuid", "desktop:100000:65536\nbad-record\n", 0644)
			case "overlap":
				f.put("/etc/subuid", "desktop:100000:65536\nother:100001:1\n", 0644)
			case "local-uid", "primary-gid":
				uid, gid := "100010", "700000"
				if fault == "primary-gid" {
					uid, gid = gid, uid
				}
				f.put("/etc/passwd", "root:x:0:0::/root:/bin/sh\ndesktop:x:942:942::/home/desktop:/bin/sh\nother:x:"+uid+":"+gid+"::/home/other:/bin/sh\n", 0644)
			case "group-gid":
				f.put("/etc/group", "root:x:0:\ndesktop:x:942:\nother:x:100010:\n", 0644)
			case "owner-uid", "owner-gid":
				f.put("/etc/passwd", "root:x:0:0::/root:/bin/sh\n", 0644)
				f.put("/etc/group", "root:x:0:\n", 0644)
				file := "/etc/subuid"
				if fault == "owner-gid" {
					file = "/etc/subgid"
				}
				f.put(file, "desktop:942:65536\n", 0644)
			case "uid-binding", "gid-binding", "home-binding", "duplicate-owner":
				line := "desktop:x:942:942::/home/desktop:/bin/sh\n"
				if fault == "uid-binding" {
					line = "desktop:x:943:942::/home/desktop:/bin/sh\n"
				}
				if fault == "gid-binding" {
					line = "desktop:x:942:943::/home/desktop:/bin/sh\n"
				}
				if fault == "home-binding" {
					line = "desktop:x:942:942::/home/other:/bin/sh\n"
				}
				if fault == "duplicate-owner" {
					line += line
				}
				f.put("/etc/passwd", "root:x:0:0::/root:/bin/sh\n"+line, 0644)
			case "profile-owner", "profile-uid", "profile-gid", "profile-home", "profile-root", "profile-version", "profile-field":
				changed := p
				switch fault {
				case "profile-owner":
					changed.Name = "other"
				case "profile-uid":
					changed.UID++
				case "profile-gid":
					changed.GID++
				case "profile-home":
					changed.Home = "/home/other"
				case "profile-root":
					changed.Piers = "/srv/other"
				case "profile-version":
					changed.Version++
				}
				data, _ := json.Marshal(changed)
				if fault == "profile-field" {
					data = append([]byte(`{"future":true,`), data[1:]...)
				}
				f.put(ownerhost.Path, string(data)+"\n", 0644)
			case "grant-owner":
				f.metadata["/etc/subuid"] = fakeInfo{uid: p.UID, mode: 0644}
			case "grant-writable":
				f.put("/etc/subuid", "desktop:100000:65536\n", 0666)
			case "grant-symlink":
				if err := os.Rename(f.i.path("/etc/subuid"), f.i.path("/etc/subuid.saved")); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("subuid.saved", f.i.path("/etc/subuid")); err != nil {
					t.Fatal(err)
				}
			case "home-owner":
				f.metadata[p.Home] = fakeInfo{uid: p.UID + 1, gid: p.GID, mode: os.ModeDir | 0755}
			case "home-writable":
				f.metadata[p.Home] = fakeInfo{uid: p.UID, gid: p.GID, mode: os.ModeDir | 0775}
			case "external-subid":
				f.put("/etc/nsswitch.conf", systemdIdentityNSS+"subid: sss\n", 0644)
			case "reservation-conflict":
				f.put(hostsetup.SubIDReservationsPath, `{"version":1,"authority":"existing narrower policy","uid":{"start":200000,"count":65536},"gid":{"start":200000,"count":65536}}`, 0644)
			case "reservation-malformed":
				f.put(hostsetup.SubIDReservationsPath, "{}", 0644)
			}
			before := tree(t, f.i.root)
			if err := f.i.preserveOwnerMappings(context.Background(), p); err == nil {
				t.Fatal("accepted", fault)
			}
			if len(f.calls) != 0 || f.i.mutated || !reflect.DeepEqual(before, tree(t, f.i.root)) {
				t.Fatal("preservation check mutated host", fault)
			}
		})
	}
}

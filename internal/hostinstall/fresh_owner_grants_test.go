package hostinstall

import (
	"bytes"
	"context"
	"os"
	"reflect"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostsetup"
)

func TestFreshOwnerGrantReuseRechecksIdentityAndRootFileEvidence(t *testing.T) {
	for _, fault := range []string{"uid", "gid", "home", "missing-owner", "duplicate-owner", "grant-owner", "grant-mode", "grant-link", "collision", "overlap", "short-gid", "malformed-gid", "external-subid", "reservation-conflict", "reservation-malformed"} {
		t.Run(fault, func(t *testing.T) {
			f, p, _ := ownerFixture(t)
			f.put("/etc/nsswitch.conf", systemdIdentityNSS, 0644)
			switch fault {
			case "uid":
				f.put("/etc/passwd", "root:x:0:0::/root:/bin/sh\ndesktop:x:943:942::/home/desktop:/bin/sh\n", 0644)
			case "gid":
				f.put("/etc/passwd", "root:x:0:0::/root:/bin/sh\ndesktop:x:942:943::/home/desktop:/bin/sh\n", 0644)
			case "home":
				f.put("/etc/passwd", "root:x:0:0::/root:/bin/sh\ndesktop:x:942:942::/home/other:/bin/sh\n", 0644)
			case "missing-owner":
				f.put("/etc/passwd", "root:x:0:0::/root:/bin/sh\n", 0644)
			case "duplicate-owner":
				f.put("/etc/passwd", "root:x:0:0::/root:/bin/sh\ndesktop:x:942:942::/home/desktop:/bin/sh\ndesktop:x:942:942::/home/desktop:/bin/sh\n", 0644)
			case "grant-owner":
				f.metadata["/etc/subuid"] = fakeInfo{uid: p.UID, mode: 0644}
			case "grant-mode":
				f.put("/etc/subuid", "desktop:100000:65536\n", 0666)
			case "grant-link":
				if err := os.Rename(f.i.path("/etc/subuid"), f.i.path("/etc/subuid.saved")); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("subuid.saved", f.i.path("/etc/subuid")); err != nil {
					t.Fatal(err)
				}
			case "collision":
				f.put("/etc/group", "root:x:0:\ndesktop:x:942:\nother:x:100010:\n", 0644)
			case "overlap":
				f.put("/etc/subuid", "desktop:100000:65536\nother:100001:1\n", 0644)
			case "short-gid":
				f.put("/etc/subgid", "desktop:100000:1\n", 0644)
			case "malformed-gid":
				f.put("/etc/subgid", "broken\n", 0644)
			case "external-subid":
				f.put("/etc/nsswitch.conf", systemdIdentityNSS+"subid: sss\n", 0644)
			case "reservation-conflict":
				f.put(hostsetup.SubIDReservationsPath, `{"version":1,"authority":"existing policy","uid":{"start":200000,"count":65536},"gid":{"start":200000,"count":65536}}`, 0644)
			case "reservation-malformed":
				f.put(hostsetup.SubIDReservationsPath, "{}", 0644)
			}
			before := tree(t, f.i.root)
			if err := f.i.ownerMappings(context.Background(), p); err == nil {
				t.Fatal("accepted", fault)
			}
			if len(f.calls) != 0 || f.i.mutated || !reflect.DeepEqual(before, tree(t, f.i.root)) {
				t.Fatal("invalid fresh mapping evidence changed host state", fault)
			}
		})
	}
}

func TestFreshMappingPlansValidateBothFilesBeforeAllocation(t *testing.T) {
	f, p, _ := ownerFixture(t)
	f.put("/etc/subuid", "", 0644)
	f.put("/etc/subgid", "malformed\n", 0644)
	before, _ := os.ReadFile(f.i.path("/etc/subuid"))
	if err := f.i.ownerMappings(context.Background(), p); err == nil {
		t.Fatal("invalid second mapping file was accepted")
	}
	after, _ := os.ReadFile(f.i.path("/etc/subuid"))
	if len(f.calls) != 0 || f.i.mutated || !bytes.Equal(before, after) {
		t.Fatal("UIDs were allocated before GID validation")
	}
}

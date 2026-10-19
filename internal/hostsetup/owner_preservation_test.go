package hostsetup

import (
	"os"
	"strings"
	"testing"
)

func TestInstalledOwnerSubIDUncertaintyIsAdvisoryWithoutInventingAuthority(t *testing.T) {
	nss := "passwd: files systemd\ngroup: files [SUCCESS=merge] systemd\n"
	grant := "desktop:100000:65536\n"
	status, detail := ownerSubIDStatus(nss, nil, os.ErrNotExist, true, "desktop", grant, "942", "subuid", []uint64{0, 942})
	if status != "unknown" || !strings.Contains(detail, "collision safety remains unknown") || !strings.Contains(detail, "without allocation") {
		t.Fatal(status, detail)
	}
	if status, _ := ownerSubIDStatus(nss, nil, os.ErrNotExist, false, "desktop", grant, "942", "subuid", []uint64{0, 942}); status != "blocked" {
		t.Fatal("fresh policy weakened", status)
	}
	for _, tc := range []struct {
		name, text, nss string
		ids             []uint64
		want            string
	}{
		{"missing", "other:100000:65536\n", nss, []uint64{0, 942}, "missing"},
		{"collision", grant, nss, []uint64{0, 942, 100001}, "blocked"},
		{"overlap", grant + "other:100001:2\n", nss, []uint64{0, 942}, "blocked"},
		{"malformed", "broken", nss, []uint64{0, 942}, "unknown"},
		{"inventory-unknown", grant, nss, nil, "unknown"},
		{"external-subid", grant, nss + "subid: sss\n", []uint64{0, 942}, "blocked"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, detail := ownerSubIDStatus(tc.nss, nil, os.ErrNotExist, true, "desktop", tc.text, "942", "subuid", tc.ids)
			if status != tc.want || strings.Contains(detail, "without allocation") {
				t.Fatal(status, detail)
			}
		})
	}
	if status, _ := ownerSubIDStatus(nss, []byte("{}"), nil, true, "desktop", grant, "942", "subuid", []uint64{0, 942}); status != "blocked" {
		t.Fatal("malformed explicit policy bypassed")
	}
	if status, _ := ownerSubIDStatus(nss, nil, os.ErrPermission, true, "desktop", grant, "942", "subuid", []uint64{0, 942}); status != "blocked" {
		t.Fatal("unreadable explicit policy bypassed")
	}
}

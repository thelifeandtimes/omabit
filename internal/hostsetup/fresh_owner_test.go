package hostsetup

import (
	"os"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
)

func TestFreshLocalOwnerConsumesNativeGrantsOnStockNSS(t *testing.T) {
	p := ownerhost.Profile{Version: 1, Name: "desktop", UID: 1000, GID: 1000, Home: "/home/desktop", Piers: ownerhost.DefaultPiers}
	passwd := "root:x:0:0::/root:/bin/bash\ndesktop:x:1000:1000::/home/desktop:/bin/bash\n"
	groups := "root:x:0:\ndesktop:x:1000:\n"
	uids, gids := LocalIDs(passwd, groups)
	if !LocalOwnerBinding(passwd, p) {
		t.Fatal("existing local owner was not identified")
	}
	for _, nss := range []string{
		"passwd: files systemd\ngroup: files [SUCCESS=merge] systemd\n",
		"passwd: files systemd\ngroup: files systemd\nsubid: files\n",
		"passwd: files sss\ngroup: files sss\nsubid: files\n",
	} {
		for kind, ids := range map[string][]uint64{"subuid": uids, "subgid": gids} {
			status, detail := ValidateOwnerSubIDAuthority(nss, nil, os.ErrNotExist, LocalOwnerBinding(passwd, p), p.Name, "desktop:100000:65536\n", "1000", kind, ids)
			if status != "pass" || !strings.Contains(detail, "without allocation") {
				t.Fatal(kind, status, detail)
			}
			// Reusing an existing grant must not imply authority to allocate new IDs.
			if LocalIdentityAuthority(nss) {
				t.Fatal("NSS completeness was invented")
			}
			if status, _ := ValidateOwnerSubIDAuthority(nss, nil, os.ErrNotExist, true, p.Name, "", "1000", kind, ids); status != "blocked" {
				t.Fatal("missing grants authorize allocation", status)
			}
		}
	}
}

func TestLocalOwnerBindingRejectsMissingChangedAndDuplicateAccounts(t *testing.T) {
	p := ownerhost.Profile{Version: 1, Name: "desktop", UID: 1000, GID: 1000, Home: "/home/desktop", Piers: ownerhost.DefaultPiers}
	line := "desktop:x:1000:1000::/home/desktop:/bin/bash\n"
	for _, bad := range []string{"", "broken", strings.Replace(line, "1000:1000", "1001:1000", 1), strings.Replace(line, "1000:1000", "1000:1001", 1), strings.Replace(line, "/home/desktop", "/home/other", 1), line + line} {
		if LocalOwnerBinding(bad, p) {
			t.Fatal("accepted unbound local identity", bad)
		}
	}
}

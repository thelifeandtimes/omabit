package access

import (
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"testing"
)

func TestPrivateChainInventoryObserverExactReadOnly(t *testing.T) {
	g := Grant{Role: "observer"}
	if !g.Allows("GET", protocol.PrivateChainProjectsPath) {
		t.Fatal("observer cannot read saved inventory")
	}
	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE", "HEAD"} {
		if g.Allows(method, protocol.PrivateChainProjectsPath) {
			t.Fatal(method)
		}
	}
	for _, path := range []string{protocol.PrivateChainProjectsPath + "/", protocol.PrivateChainProjectsPath + "/start", protocol.PrivateChainProjectsPath + "/project"} {
		if g.Allows("GET", path) {
			t.Fatal("broadened observer route", path)
		}
	}
	if (Grant{Role: "unknown"}).Allows("GET", protocol.PrivateChainProjectsPath) {
		t.Fatal("unknown role admitted")
	}
}

package access

import (
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"testing"
)

func TestProjectRecoveryCheckpointObserverReadOnly(t *testing.T) {
	g := Grant{Role: "observer"}
	path := protocol.ProjectRecoveryCheckpointPath("project", "operation")
	if !g.Allows("GET", path) {
		t.Fatal("observer denied")
	}
	for _, method := range []string{"POST", "PUT", "DELETE"} {
		if g.Allows(method, path) {
			t.Fatal(method)
		}
	}
	for _, bad := range []string{path + "/extra", "/v1/projects/p/recovery/", "/v1/projects/p/q/recovery/op"} {
		if g.Allows("GET", bad) {
			t.Fatal(bad)
		}
	}
}

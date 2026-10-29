package access

import (
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"testing"
)

func TestProjectRecoverySourceObserverReadOnly(t *testing.T) {
	g := Grant{Role: "observer"}
	path := protocol.ProjectRecoverySourcePath("project")
	if !g.Allows("GET", path) {
		t.Fatal("observer denied")
	}
	for _, method := range []string{"POST", "PUT", "DELETE"} {
		if g.Allows(method, path) {
			t.Fatal(method)
		}
	}
	for _, bad := range []string{path + "/extra", "/v1/projects/a/b/recovery-source", "/v1/projects//recovery-source"} {
		if g.Allows("GET", bad) {
			t.Fatal(bad)
		}
	}
}

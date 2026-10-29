package access

import "testing"

func TestObserverProjectInventoryIsExactReadOnly(t *testing.T) {
	g := Grant{Role: "observer"}
	if !g.Allows("GET", "/v1/projects") {
		t.Fatal("missing read")
	}
	for _, method := range []string{"POST", "PUT", "DELETE", "PATCH"} {
		if g.Allows(method, "/v1/projects") {
			t.Fatal("mutation allowed")
		}
	}
	for _, path := range []string{"/v1/projects/", "/v1/projects/new", "/v1/projects/prepare"} {
		if g.Allows("GET", path) {
			t.Fatal("unrelated path allowed")
		}
	}
	if (Grant{Role: "unknown"}).Allows("GET", "/v1/projects") {
		t.Fatal("unknown role")
	}
}

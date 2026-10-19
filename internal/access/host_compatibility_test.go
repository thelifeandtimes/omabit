package access

import "testing"

func TestObserverHostCompatibilityExactReadRoute(t *testing.T) {
	observer := Grant{Role: "observer"}
	if !observer.Allows("GET", "/v1/host-compatibility") {
		t.Fatal("observer lacks explicit host report permission")
	}
	for _, method := range []string{"POST", "PUT", "DELETE", "PATCH"} {
		if observer.Allows(method, "/v1/host-compatibility") {
			t.Fatal("observer can mutate host")
		}
	}
	for _, path := range []string{"/v1/host-compatibility/verify", "/v1/host-compatibility/", "/v1/verify-host-packages"} {
		if observer.Allows("GET", path) {
			t.Fatal("observer permission widened", path)
		}
	}
	if (Grant{Role: "unknown"}).Allows("GET", "/v1/host-compatibility") {
		t.Fatal("unknown role permitted")
	}
}

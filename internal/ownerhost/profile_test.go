package ownerhost

import "testing"

func TestOwnerProfileRequiresExplicitNonRootIdentityAndPaths(t *testing.T) {
	p := Profile{Version: 1, Name: "alice", UID: 1000, GID: 1000, Home: "/home/alice", Piers: DefaultPiers}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, fault := range []string{"uid", "gid", "root", "legacy", "name", "home", "piers", "escape"} {
		bad := p
		switch fault {
		case "uid":
			bad.UID = 0
		case "gid":
			bad.GID = 0
		case "root":
			bad.Name = "root"
		case "legacy":
			bad.Name = "urbit-runner"
		case "name":
			bad.Name = "alice\nUser=root"
		case "home":
			bad.Home = "relative"
		case "piers":
			bad.Piers = "/"
		case "escape":
			bad.Home = "/home/a%h"
		}
		if bad.Validate() == nil {
			t.Fatal("accepted", fault)
		}
	}
}

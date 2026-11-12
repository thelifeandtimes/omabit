package ownerhost

import "testing"

func TestDevelopmentPathsNeverMutateInstalledProfile(t *testing.T) {
	old := developmentRoot
	t.Cleanup(func() { developmentRoot = old })
	developmentRoot = ""
	p := Profile{Version: 1, Name: "alice", UID: 1000, GID: 1000, Home: "/home/alice", Piers: DefaultPiers}
	if selectedProfilePath() != Path || RuntimeDirectory() != "" || *runtimeProfile(&p) != p {
		t.Fatal("ordinary defaults changed")
	}
	developmentRoot = "/srv/dev/urbit"
	got := runtimeProfile(&p)
	if got.Home != "/srv/dev/urbit/state" || p.Home != "/home/alice" || selectedProfilePath() != "/srv/dev/urbit/owner.json" {
		t.Fatal("development state is not separate")
	}
	developmentRoot = ""
	for _, path := range []string{"", "/srv/urbit", "/home/alice", "/srv/dev/urbit/../urbit"} {
		if ConfigureDevelopment(path) == nil || Development() {
			t.Fatalf("accepted unsafe override %q", path)
		}
	}
}

func TestNamedDevelopmentRootsStayDistinct(t *testing.T) {
	for _, root := range []string{"/srv/dev/urbit", "/srv/dev/urbit-040", "/srv/dev/urbit-review-2"} {
		if !validDevelopmentRoot(root) {
			t.Fatal(root)
		}
	}
	for _, root := range []string{"/srv/urbit", "/srv/dev/urbit/040", "/srv/dev/urbit-", "/srv/dev/urbit-../urbit", "/srv/dev/urbit-040/", "/srv/dev/urbit-040\n", "/tmp/urbit-040"} {
		if validDevelopmentRoot(root) {
			t.Fatal("unsafe root", root)
		}
	}
	if developmentRuntimeName("/srv/dev/urbit") != "omabit-dev" || developmentRuntimeName("/srv/dev/urbit-040") != "omabit-dev-040" {
		t.Fatal("runtime paths collided")
	}
}

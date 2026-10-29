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

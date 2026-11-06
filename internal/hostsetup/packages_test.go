package hostsetup

import (
	"reflect"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
)

func TestNativePackagePlanDistinguishesMissingFromUnsupported(t *testing.T) {
	r := Report{Compatibility: &hostcompat.Report{Distribution: hostcompat.Distribution{ID: "omarchy", Provider: "pacman"}}, Checks: []Check{
		{ID: "binary-docker", Status: "missing"}, {ID: "binary-dockerd", Status: "missing"},
		{ID: "docker-version", Status: "unknown"}, {ID: "binary-rootlesskit", Status: "missing"},
		{ID: "binary-slirp4netns", Status: "missing"}, {ID: "binary-ssh", Status: "missing"},
		{ID: "binary-ssh-keygen", Status: "missing"}, {ID: "binary-sshd", Status: "missing"},
	}}
	if got := NativePackages(r); !reflect.DeepEqual(got, []string{"docker", "openssh", "rootlesskit", "slirp4netns"}) {
		t.Fatal(got)
	}
	for _, c := range r.Checks {
		if !InstallableCheck(r, c) {
			t.Fatal("fresh missing prerequisite blocked setup", c)
		}
	}
	for _, c := range []Check{{ID: "docker-version", Status: "blocked"}, {ID: "binary-rootlesskit", Status: "unknown"}, {ID: "binary-docker", Status: "blocked"}, {ID: "binary-pkexec", Status: "missing"}, {ID: "binary-untrusted", Status: "missing"}} {
		if InstallableCheck(r, c) {
			t.Fatal("non-installable check bypassed", c)
		}
	}
	r.Compatibility.InstalledEngineVersion = "30.0.0"
	if InstallableCheck(r, r.Checks[2]) {
		t.Fatal("installed engine could be silently replaced")
	}
	r.Compatibility.Distribution.ID = "unreviewed"
	if len(NativePackages(r)) != 0 {
		t.Fatal("unreviewed distribution got a pacman plan")
	}
	r.Compatibility = nil
	if len(NativePackages(r)) != 0 {
		t.Fatal("missing metadata got an install plan")
	}
}

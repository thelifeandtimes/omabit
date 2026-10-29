package hostcompat

import (
	"strings"
	"testing"
)

func TestPackageVerificationErrorNamesOnlyUnverifiedPackages(t *testing.T) {
	r := Report{Packages: []Package{{Name: "docker.io", Provenance: "verified"}, {Name: "uidmap", Provenance: "unverified"}, {Name: "uidmap", Provenance: "unverified"}, {Name: "slirp4netns", Provenance: "unverified"}, {Name: "invalid name\n", Provenance: "unverified"}}}
	got := packageVerificationError(r).Error()
	if !strings.Contains(got, "slirp4netns, uidmap;") || strings.Contains(got, "docker.io") || strings.Contains(got, "invalid name") {
		t.Fatal(got)
	}
	if !strings.Contains(packageVerificationError(Report{}).Error(), "inventory is unavailable") {
		t.Fatal("missing inventory guidance")
	}
}

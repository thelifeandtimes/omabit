//go:build linux

package hostcompat

import (
	"context"
	"strings"
	"testing"
)

func TestUbuntuSplitComponentsAndConflictingAuthority(t *testing.T) {
	const base = "deb http://us.archive.ubuntu.com/ubuntu jammy "
	for _, tc := range []struct {
		name, extra string
		valid       bool
	}{
		{"stock components", base + "universe\n" + base + "multiverse\n", true},
		{"duplicate", base + "main restricted\n", true},
		{"different key", "deb [signed-by=/etc/apt/other.gpg] http://us.archive.ubuntu.com/ubuntu jammy universe\n", false},
		{"different architecture", "deb [arch=arm64] http://us.archive.ubuntu.com/ubuntu jammy universe\n", false},
		{"different transport", "deb https://us.archive.ubuntu.com/ubuntu jammy universe\n", false},
		{"insecure", "deb [trusted=yes] http://us.archive.ubuntu.com/ubuntu jammy universe\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := debFixture()
			addFixture(m, "etc/apt/sources.list", []byte(base+"main restricted\n"+tc.extra), 0644)
			sources, _, err := aptSources(m)
			if (err == nil) != tc.valid {
				t.Fatalf("sources %v error %v", sources, err)
			}
			if tc.valid && (len(sources) != 1 || !sources[0].permitsIndex("main/binary-amd64/Packages", "amd64")) {
				t.Fatal("lost source scope", sources)
			}
			if tc.name == "stock components" && strings.Join(sources[0].Components, ",") != "main,multiverse,restricted,universe" {
				t.Fatal(sources)
			}
		})
	}
}

func TestSplitAPTComponentsRetainCryptographicVerification(t *testing.T) {
	m, run, key, _ := dockerAPTFixture(t, false)
	// Same Release, separate components, mixing supported list/deb822 syntax.
	addFixture(m, "etc/apt/sources.list", []byte("deb [arch=amd64 signed-by=/"+key+"] https://download.docker.com/linux/ubuntu noble extra\n"), 0644)
	v := verifier{fsys: m, run: run, store: fixtureStore(m)}
	snapshot := inspect(m, "linux", "amd64")
	p := snapshot.report.Packages[0]
	if _, err := v.verifyPackage(context.Background(), "deb", p, snapshot.owners[p.Role]); err != nil {
		t.Fatal("split source signature/payload rejected", err)
	}
	// The merged scope must not authenticate altered installed payloads.
	m["usr/sbin/dockerd"].Data = []byte("altered executable")
	v = verifier{fsys: m, run: run, store: fixtureStore(m)}
	if _, err := v.verifyPackage(context.Background(), "deb", p, snapshot.owners[p.Role]); err == nil {
		t.Fatal("altered executable accepted")
	}
}

//go:build linux

package hostcompat

import (
	"io/fs"
	"path"
	"strings"
	"syscall"
	"testing"
	"testing/fstest"
)

func fixtureFS(files map[string]string) fstest.MapFS {
	m := fstest.MapFS{".": &fstest.MapFile{Mode: fs.ModeDir | 0755, Sys: &syscall.Stat_t{Uid: 0}}}
	for name, data := range files {
		addFixture(m, name, []byte(data), 0644)
	}
	return m
}

func addFixture(m fstest.MapFS, name string, data []byte, mode fs.FileMode) {
	m[name] = &fstest.MapFile{Data: data, Mode: mode, Sys: &syscall.Stat_t{Uid: 0}}
	for dir := path.Dir(name); dir != "."; dir = path.Dir(dir) {
		if m[dir] == nil {
			m[dir] = &fstest.MapFile{Mode: fs.ModeDir | 0755, Sys: &syscall.Stat_t{Uid: 0}}
		}
	}
}

func pacmanFixture() fstest.MapFS {
	m := fixtureFS(map[string]string{
		"etc/os-release": "ID=arch\nID_LIKE='arch linux'\nPRETTY_NAME=ignored\n",
		"var/lib/pacman/local/docker-29.9.1-1/desc":  "%NAME%\ndocker\n\n%VERSION%\n1:29.9.1-1\n\n%ARCH%\nx86_64\n\n%VALIDATION%\npgp\n",
		"var/lib/pacman/local/docker-29.9.1-1/files": "%FILES%\nusr/bin/dockerd\nusr/bin/docker\n",
	})
	addFixture(m, "usr/bin/dockerd", []byte("authenticated dockerd"), 0755)
	addFixture(m, "usr/bin/docker", []byte("authenticated docker"), 0755)
	return m
}

func debFixture() fstest.MapFS {
	m := fixtureFS(map[string]string{
		"etc/os-release":                      "ID=ubuntu\nVERSION_ID=\"24.04\"\nID_LIKE=debian\n",
		"var/lib/dpkg/status":                 "Package: docker.io\nStatus: install ok installed\nVersion: 5:29.9.1-1~ubuntu.24.04~noble\nArchitecture: amd64\nDescription: native\n continued\n .\n more\n\nPackage: removed\nStatus: deinstall ok config-files\nVersion: 1\nArchitecture: amd64\n",
		"var/lib/dpkg/info/docker.io.list":    "/usr/sbin/dockerd\n/usr/bin/docker\n",
		"var/lib/dpkg/info/docker.io.md5sums": "not publisher authentication",
	})
	addFixture(m, "usr/sbin/dockerd", []byte("debian dockerd"), 0755)
	addFixture(m, "usr/bin/docker", []byte("debian docker"), 0755)
	return m
}

func TestNativeMetadataFullVersionsAndNoAuthenticationClaim(t *testing.T) {
	for _, tc := range []struct {
		m                      fstest.MapFS
		provider, version, exe string
	}{{pacmanFixture(), "pacman", "1:29.9.1-1", "/usr/bin/dockerd"}, {debFixture(), "deb", "5:29.9.1-1~ubuntu.24.04~noble", "/usr/sbin/dockerd"}} {
		r := Inspect(tc.m, "linux", "amd64")
		if r.Distribution.Provider != tc.provider || r.InstalledEngineVersion != "29.9.1" || r.Packages[0].Version != tc.version || r.Packages[0].Executables[0] != tc.exe {
			t.Fatalf("wrong native metadata: %+v", r)
		}
		if r.Packages[0].Provenance != "metadata-only" || r.Provenance == "verified" || r.CanMutate || r.Qualification != "unqualified" {
			t.Fatal("metadata was promoted to authentication or qualification")
		}
		if tc.provider == "pacman" && !strings.Contains(r.Packages[0].Detail, "package-manager assertion") {
			t.Fatal("Pacman assertion method not labelled")
		}
	}
}

func TestParserErrorsAndPrerelease(t *testing.T) {
	for _, s := range []string{"Package: docker\nPackage: other\n", " continuation\n", "bad line\n"} {
		if _, err := parseControl(s); err == nil {
			t.Fatalf("accepted malformed control %q", s)
		}
	}
	for _, s := range []string{"%NAME%\ndocker\n\n%NAME%\nother\n", "orphan\n"} {
		if _, err := parsePacman(s); err == nil {
			t.Fatalf("accepted malformed pacman %q", s)
		}
	}
	for _, s := range []string{"ID=arch\nID=ubuntu\n", "ID=$(secret)\n", "ID=\"not closed\n"} {
		if _, err := parseOSRelease(s); err == nil {
			t.Fatalf("accepted os-release %q", s)
		}
	}
	for _, v := range []string{"29.9.1~rc1-1", "29.9.1-rc1-1", "29.9.1beta", "garbage", "x:29.9.1-1"} {
		if upstreamVersion(v) != "" {
			t.Fatalf("accepted unstable package %s", v)
		}
	}
	if upstreamVersion("5:29.9.1-1~ubuntu.24.04~noble") != "29.9.1" {
		t.Fatal("epoch/revision handling")
	}
	for _, v := range []string{"29.9.1+dfsg1-1", "29.9.1+dev.build-1", "29.9.1+build-rc1-1", "29.9.1+build-meta"} {
		if upstreamVersion(v) != "29.9.1" {
			t.Fatalf("stable build metadata rejected: %s", v)
		}
	}
}

func TestUnsafeOwnershipMissingEvidenceAndDuplicateOwners(t *testing.T) {
	for _, mutate := range []func(fstest.MapFS){
		func(m fstest.MapFS) { m["usr/bin/dockerd"].Sys = &syscall.Stat_t{Uid: 1000} },
		func(m fstest.MapFS) { m["usr/bin"].Mode |= 0020 },
		func(m fstest.MapFS) { m["var/lib/pacman/local"].Sys = nil },
		func(m fstest.MapFS) { m["usr/bin/dockerd"].Mode = fs.ModeNamedPipe | 0600 },
		func(m fstest.MapFS) {
			addFixture(m, "var/lib/pacman/local/other/desc", []byte("%NAME%\nother\n\n%VERSION%\n1.0.0-1\n"), 0644)
			addFixture(m, "var/lib/pacman/local/other/files", []byte("%FILES%\nusr/bin/dockerd\n"), 0644)
		},
	} {
		m := pacmanFixture()
		mutate(m)
		r := Inspect(m, "linux", "amd64")
		if r.Packages[0].Provenance != "unknown" || r.InstalledEngineVersion != "" {
			t.Fatalf("unsafe metadata accepted: %+v", r.Packages[0])
		}
	}
}

func TestBoundedReadsAndSymlinks(t *testing.T) {
	m := pacmanFixture()
	if _, err := readTrusted(m, "usr/bin/dockerd", 2); err == nil {
		t.Fatal("oversized read accepted")
	}
	addFixture(m, "usr/bin/dockerd", []byte("../sbin/dockerd"), fs.ModeSymlink|0777)
	addFixture(m, "usr/sbin/dockerd", []byte("target"), 0755)
	resolved, _, err := resolve(m, "usr/bin/dockerd")
	if err != nil || resolved != "usr/sbin/dockerd" {
		t.Fatalf("safe fixed symlink: %q %v", resolved, err)
	}
	m["usr/sbin/dockerd"].Mode = fs.ModeSymlink | 0777
	m["usr/sbin/dockerd"].Data = []byte("../bin/dockerd")
	if _, _, err := resolve(m, "usr/bin/dockerd"); err == nil {
		t.Fatal("symlink cycle accepted")
	}
	m["usr/bin/dockerd"].Data = []byte("../../../../outside")
	if _, _, err := resolve(m, "usr/bin/dockerd"); err == nil {
		t.Fatal("root escape accepted")
	}
}

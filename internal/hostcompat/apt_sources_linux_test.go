//go:build linux

package hostcompat

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func dockerAPTFixture(t *testing.T, armor bool) (fstest.MapFS, commandRunner, string, string) {
	t.Helper()
	home := signingHome(t)
	m := debFixture()
	keyPath := "etc/apt/keyrings/docker.gpg"
	exportArgs := []string{"--no-options", "--homedir", home, "--batch", "--export"}
	if armor {
		keyPath = "etc/apt/keyrings/docker.asc"
		exportArgs = append(exportArgs, "--armor")
	}
	key := nativeTestCommand(t, "/usr/bin/gpg", exportArgs...)
	addFixture(m, keyPath, key, 0644)
	addFixture(m, "etc/apt/sources.list.d/docker.sources", []byte("Types: deb\nURIs: https://download.docker.com/linux/ubuntu\nSuites: noble\nComponents: stable\nArchitectures: amd64\nSigned-By: /"+keyPath+"\n"), 0644)
	s := inspect(m, "linux", "amd64")
	owner := s.owners["engine"]
	artifact := "var/cache/apt/archives/docker.io_29.9.1-1_amd64.deb"
	addFixture(m, artifact, []byte("authenticated fixture deb container"), 0644)
	h := sha256.Sum256(m[artifact].Data)
	index := fmt.Sprintf("Package: docker.io\nVersion: %s\nArchitecture: amd64\nSize: %d\nSHA256: %x\nFilename: dists/noble/pool/docker.io.deb\n\n", owner.version, len(m[artifact].Data), h)
	indexName := "var/lib/apt/lists/download.docker.com_linux_ubuntu_dists_noble_stable_binary-amd64_Packages.lz4"
	addFixture(m, indexName, []byte(index), 0644)
	ih := sha256.Sum256([]byte(index))
	plain := fmt.Sprintf("Origin: Docker\nSuite: noble\nCodename: noble\nDate: %s\nValid-Until: %s\nSHA256:\n %x %d stable/binary-amd64/Packages\n", time.Now().Add(-time.Hour).UTC().Format(time.RFC1123Z), time.Now().Add(time.Hour).UTC().Format(time.RFC1123Z), ih, len(index))
	name := filepath.Join(t.TempDir(), "Release")
	if err := os.WriteFile(name, []byte(plain), 0600); err != nil {
		t.Fatal(err)
	}
	signed := nativeTestCommand(t, "/usr/bin/gpg", "--no-options", "--homedir", home, "--batch", "--pinentry-mode", "loopback", "--passphrase", "", "--output", "-", "--clearsign", name)
	addFixture(m, "var/lib/apt/lists/download.docker.com_linux_ubuntu_dists_noble_InRelease", signed, 0644)
	for _, tool := range []string{"usr/bin/gpg", "usr/bin/gpgv", "usr/bin/dpkg-deb", "usr/lib/apt/apt-helper"} {
		addFixture(m, tool, []byte("native fixture tool identity"), 0755)
	}
	payload := makeTar(t, map[string][]byte{"./usr/sbin/dockerd": m["usr/sbin/dockerd"].Data, "./usr/bin/docker": m["usr/bin/docker"].Data})
	real := fixtureRunner(t, m, home)
	run := func(ctx context.Context, tool string, args []string, limit int64) ([]byte, error) {
		switch tool {
		case "/usr/lib/apt/apt-helper":
			f := m[strings.TrimPrefix(args[1], "/")]
			if f == nil {
				return nil, errEvidence
			}
			return f.Data, nil
		case "/usr/bin/dpkg-deb":
			if args[0] == "--field" {
				return []byte("Package: docker.io\nVersion: " + owner.version + "\nArchitecture: amd64\n"), nil
			}
			return payload, nil
		default:
			return real(ctx, tool, args, limit)
		}
	}
	return m, run, keyPath, home
}

func TestOfficialDockerSignedByTrueCryptoArmorAndDurableAPTProof(t *testing.T) {
	for _, armor := range []bool{false, true} {
		t.Run(fmt.Sprint(armor), func(t *testing.T) {
			m, run, _, _ := dockerAPTFixture(t, armor)
			v := verifier{fsys: m, run: run, store: fixtureStore(m)}
			set, _, err := v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), receiptSet{})
			if err != nil || len(set.Packages) != 2 {
				t.Fatalf("scoped APT authentication: %+v %v", set, err)
			}
			for _, rec := range set.Packages {
				if !rec.Authorized || rec.SourceID == "" || len(rec.Retained) != 3 {
					t.Fatal("source-scoped retained proof incomplete")
				}
			}
			publishFixtureSet(t, m, set)
			for name := range m {
				if strings.HasPrefix(name, "var/cache/apt/archives/") || strings.HasPrefix(name, "var/lib/apt/lists/") {
					delete(m, name)
				}
			}
			// apt update may rotate all list bytes without changing root authority.
			addFixture(m, "var/lib/apt/lists/unrelated_new_InRelease", []byte("new transient list"), 0644)
			m["var/lib/dpkg/status"].Data = append(m["var/lib/dpkg/status"].Data, []byte("\nPackage: unrelated\nStatus: deinstall ok config-files\nVersion: 2\nArchitecture: amd64\n")...)
			if r := Inspect(runtimeReadFS{m, t}, "linux", "amd64"); r.Packages[0].Provenance != "verified" {
				t.Fatalf("APT rotation invalidated unchanged bytes: %+v", r.Packages[0])
			}
			v = verifier{fsys: m, run: run, store: fixtureStore(m)}
			next, _, err := v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), set)
			if err != nil || len(next.Packages) != 2 || !next.Packages[0].Authorized || !next.Packages[1].Authorized {
				t.Fatalf("retained APT chain refresh: %+v %v", next, err)
			}
		})
	}
}

func TestAPTAuthorityIsPerSourceAndNoTrustedYesBypass(t *testing.T) {
	m, run, keyPath, _ := dockerAPTFixture(t, false)
	otherHome := signingHome(t)
	otherKey := nativeTestCommand(t, "/usr/bin/gpg", "--no-options", "--homedir", otherHome, "--batch", "--export")
	addFixture(m, "etc/apt/keyrings/other.gpg", otherKey, 0644)
	// Keep the correct signing key on disk, authorized for a different source.
	addFixture(m, "etc/apt/sources.list", []byte("deb [signed-by=/"+keyPath+"] https://different.example/linux/ubuntu noble stable\n"), 0644)
	m["etc/apt/sources.list.d/docker.sources"].Data = []byte("Types: deb\nURIs: https://download.docker.com/linux/ubuntu\nSuites: noble\nComponents: stable\nSigned-By: /etc/apt/keyrings/other.gpg\n")
	s := inspect(m, "linux", "amd64")
	v := verifier{fsys: m, run: run, store: fixtureStore(m)}
	if _, err := v.verifyPackage(context.Background(), "deb", s.report.Packages[0], s.owners["engine"]); err == nil {
		t.Fatal("key authorized for another source authenticated Docker repository")
	}
	for _, text := range []string{
		"deb [trusted=yes signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu noble stable\n",
		"deb [allow-insecure=yes] https://download.docker.com/linux/ubuntu noble stable\n",
		"deb https://download.docker.com/linux/ubuntu noble stable\n",
		"deb [signed-by=ABCDEF] https://download.docker.com/linux/ubuntu noble stable\n",
		"deb [signed-by=/etc/apt/keyrings/docker.kbx] https://download.docker.com/linux/ubuntu noble stable\n",
		"deb [signed-by=/etc/apt/keyrings/docker.gpg] https://user:secret@download.docker.com/linux/ubuntu noble stable\n",
	} {
		if _, err := parseListSources(text); err == nil {
			t.Fatalf("unsafe/unsupported source accepted: %q", text)
		}
	}
	for _, tail := range []string{"Trusted: yes\n", "Trusted: no\n", "trusted: yes\n", "Check-Valid-Until: no\n", "Signed-By: -----BEGIN PGP PUBLIC KEY BLOCK-----\n inline\n"} {
		if _, err := parseDeb822Sources("Types: deb\nURIs: https://download.docker.com/linux/ubuntu\nSuites: noble\nComponents: stable\n" + tail); err == nil {
			t.Fatalf("unsafe deb822 source accepted: %q", tail)
		}
	}
	if _, err := parseListSources("deb [arch=amd64 signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu noble stable\n"); err != nil {
		t.Fatal("normal official Docker .list rejected", err)
	}
}

func TestAPTKeyringFilenameRolloverRevalidatesSameRepositoryProof(t *testing.T) {
	m, run, keyPath, home := dockerAPTFixture(t, true)
	v := verifier{fsys: m, run: run, store: fixtureStore(m)}
	old, _, err := v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), receiptSet{})
	if err != nil || len(old.Packages) == 0 {
		t.Fatalf("fixture authentication: %v", err)
	}
	publishFixtureSet(t, m, old)
	for name := range m {
		if strings.HasPrefix(name, "var/cache/apt/archives/") || strings.HasPrefix(name, "var/lib/apt/lists/") {
			delete(m, name)
		}
	}
	newPath := "etc/apt/keyrings/docker-new.gpg"
	addFixture(m, newPath, nativeTestCommand(t, "/usr/bin/gpg", "--no-options", "--homedir", home, "--batch", "--export"), 0644)
	m["etc/apt/sources.list.d/docker.sources"].Data = []byte(strings.ReplaceAll(string(m["etc/apt/sources.list.d/docker.sources"].Data), keyPath, newPath))
	delete(m, keyPath)
	if Inspect(runtimeReadFS{m, t}, "linux", "amd64").Packages[0].Provenance == "verified" {
		t.Fatal("source change not noticed before root revalidation")
	}
	v = verifier{fsys: m, run: run, store: fixtureStore(m)}
	next, _, err := v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), old)
	if err != nil {
		t.Fatal(err)
	}
	for _, rec := range next.Packages {
		if !rec.Authorized {
			t.Fatal("same repository proof could not be validated by its current configured keyring")
		}
	}
	// Expanding source components/architectures does not strand the old exact
	// package, but removing its authenticated component must withdraw it.
	sourceFile := m["etc/apt/sources.list.d/docker.sources"]
	sourceFile.Data = []byte(strings.ReplaceAll(strings.ReplaceAll(string(sourceFile.Data), "Components: stable", "Components: stable test"), "Architectures: amd64", "Architectures: amd64 arm64"))
	v = verifier{fsys: m, run: run, store: fixtureStore(m)}
	expanded, _, err := v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), next)
	if err != nil {
		t.Fatal(err)
	}
	for _, rec := range expanded.Packages {
		if !rec.Authorized {
			t.Fatal("unchanged package stranded by source scope expansion")
		}
	}
	sourceFile.Data = []byte(strings.ReplaceAll(string(sourceFile.Data), "Components: stable test", "Components: test"))
	removed, _, err := v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), expanded)
	if err != nil {
		t.Fatal(err)
	}
	for _, rec := range removed.Packages {
		if rec.Authorized {
			t.Fatal("removed source component retained authorization")
		}
	}
}

func TestAPTChangedAuthorityAndRetainedProofTamperWithdrawAuthorization(t *testing.T) {
	m, run, keyPath, _ := dockerAPTFixture(t, true)
	v := verifier{fsys: m, run: run, store: fixtureStore(m)}
	set, _, err := v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), receiptSet{})
	if err != nil || len(set.Packages) == 0 {
		t.Fatalf("fixture authentication: %v", err)
	}
	publishFixtureSet(t, m, set)
	for name := range m {
		if strings.HasPrefix(name, "var/cache/apt/archives/") || strings.HasPrefix(name, "var/lib/apt/lists/") {
			delete(m, name)
		}
	}
	originalKey := m[keyPath].Data
	m[keyPath].Data = []byte("invalid changed authority")
	if Inspect(runtimeReadFS{m, t}, "linux", "amd64").Packages[0].Provenance == "verified" {
		t.Fatal("changed source authority remained verified")
	}
	v = verifier{fsys: m, run: run, store: fixtureStore(m)}
	next, _, err := v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), set)
	if err != nil {
		t.Fatal(err)
	}
	for _, rec := range next.Packages {
		if rec.Authorized {
			t.Fatal("changed authority renewed cached proof")
		}
	}
	m[keyPath].Data = originalKey
	m[set.Packages[0].Retained[0].Path].Data = []byte("tampered retained artifact")
	next, _, err = v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), set)
	if err != nil {
		t.Fatal(err)
	}
	for _, rec := range next.Packages {
		if rec.Authorized {
			t.Fatal("tampered retained APT artifact accepted")
		}
	}
}

//go:build linux

package hostcompat

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"testing/fstest"
	"time"
)

func nativeTestCommand(t *testing.T, tool string, args ...string) []byte {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, tool, args...)
	c.Env = []string{"PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C"}
	b, err := c.Output()
	if err != nil {
		t.Fatalf("fixture command %s failed: %v", tool, err)
	}
	return b
}

func signingHome(t *testing.T) string {
	t.Helper()
	for _, tool := range []string{"/usr/bin/gpg", "/usr/bin/gpgv", "/usr/bin/bsdtar"} {
		if _, err := os.Stat(tool); err != nil {
			t.Skip("native fixture validation tools unavailable")
		}
	}
	home := t.TempDir()
	if err := os.Chmod(home, 0700); err != nil {
		t.Fatal(err)
	}
	nativeTestCommand(t, "/usr/bin/gpg", "--no-options", "--homedir", home, "--batch", "--pinentry-mode", "loopback", "--passphrase", "", "--quick-generate-key", "hostcompat isolated test fixture", "ed25519", "sign", "1d")
	nativeTestCommand(t, "/usr/bin/gpg", "--no-options", "--homedir", home, "--batch", "--check-trustdb")
	// Stop only the isolated fixture's agent, never the host's agent.
	t.Cleanup(func() {
		if _, err := os.Stat("/usr/bin/gpgconf"); err == nil {
			exec.Command("/usr/bin/gpgconf", "--homedir", home, "--kill", "gpg-agent").Run()
		}
	})
	return home
}

func makeTar(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for name, data := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0755, Typeflag: tar.TypeReg, Size: int64(len(data))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// fixtureRunner executes real native cryptographic/archive tools against an
// isolated fixture root. It never points a command at live host package state.
func fixtureRunner(t *testing.T, m fstest.MapFS, home string) commandRunner {
	t.Helper()
	root := t.TempDir()
	return func(ctx context.Context, tool string, args []string, limit int64) ([]byte, error) {
		mapped := append([]string{}, args...)
		for i, arg := range mapped {
			archivePrefix := ""
			if strings.HasPrefix(arg, "@/") {
				archivePrefix = "@"
				arg = strings.TrimPrefix(arg, "@")
			}
			if arg == "/etc/pacman.d/gnupg" {
				mapped[i] = home
				continue
			}
			if strings.HasPrefix(arg, "/") {
				f := m[strings.TrimPrefix(arg, "/")]
				if f == nil || !f.Mode.IsRegular() {
					return nil, errEvidence
				}
				name := filepath.Join(root, arg)
				if err := os.MkdirAll(filepath.Dir(name), 0700); err != nil {
					return nil, err
				}
				if err := os.WriteFile(name, f.Data, 0600); err != nil {
					return nil, err
				}
				mapped[i] = archivePrefix + name
			}
		}
		cmd := exec.CommandContext(ctx, tool, mapped...)
		cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C"}
		b, err := cmd.Output()
		if int64(len(b)) > limit {
			return nil, errEvidence
		}
		return b, err
	}
}

func authenticatedPacmanFixture(t *testing.T) (fstest.MapFS, string, string) {
	t.Helper()
	home := signingHome(t)
	m := pacmanFixture()
	archive := makeTar(t, map[string][]byte{".PKGINFO": []byte("pkgname = docker\npkgver = 1:29.9.1-1\narch = x86_64\n"), "usr/bin/dockerd": m["usr/bin/dockerd"].Data, "usr/bin/docker": m["usr/bin/docker"].Data})
	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	gz.Write(archive)
	gz.Close()
	artifact := "var/cache/pacman/pkg/docker-29.9.1-1-x86_64.pkg.tar.gz"
	addFixture(m, artifact, compressed.Bytes(), 0644)
	file := filepath.Join(t.TempDir(), "package")
	if err := os.WriteFile(file, compressed.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	sig := nativeTestCommand(t, "/usr/bin/gpg", "--no-options", "--homedir", home, "--batch", "--pinentry-mode", "loopback", "--passphrase", "", "--output", "-", "--detach-sign", file)
	addFixture(m, artifact+".sig", sig, 0644)
	for _, name := range []string{"pubring.kbx", "trustdb.gpg"} {
		b, err := os.ReadFile(filepath.Join(home, name))
		if err != nil {
			t.Fatal(err)
		}
		addFixture(m, "etc/pacman.d/gnupg/"+name, b, 0644)
	}
	public := nativeTestCommand(t, "/usr/bin/gpg", "--no-options", "--homedir", home, "--batch", "--export")
	addFixture(m, "usr/share/pacman/keyrings/fixture.gpg", public, 0644)
	addFixture(m, "usr/share/pacman/keyrings/fixture-revoked", nil, 0644)
	for _, tool := range []string{"usr/bin/gpg", "usr/bin/bsdtar"} {
		addFixture(m, tool, []byte("trusted fixture tool identity"), 0755)
	}
	return m, home, artifact
}

func TestActualPacmanSignaturePayloadAndOfflineReceiptTamper(t *testing.T) {
	m, home, artifact := authenticatedPacmanFixture(t)
	s := inspect(m, "linux", "amd64")
	v := verifier{fsys: m, run: fixtureRunner(t, m, home), authenticated: make(map[string][]digest)}
	rec, err := v.verifyPackage(context.Background(), "pacman", s.report.Packages[0], s.owners["engine"])
	if err != nil {
		t.Fatalf("actual signature+installed-byte verification: %v", err)
	}
	set := fixtureReceiptSet(t, m, "pacman", []receipt{rec})
	b, _ := json.Marshal(set)
	addFixture(m, receiptPath, b, 0644)
	r := Inspect(m, "linux", "amd64")
	if r.Packages[0].Provenance != "verified" || !strings.Contains(r.Packages[0].Detail, pacmanMethod) || r.Packages[1].Provenance == "verified" {
		t.Fatalf("offline receipt result: %+v", r.Packages)
	}
	for _, name := range []string{"usr/bin/dockerd", "usr/share/pacman/keyrings/fixture-revoked"} {
		t.Run("tamper/"+name, func(t *testing.T) {
			old := m[name].Data
			m[name].Data = append(append([]byte{}, old...), byte('!'))
			defer func() { m[name].Data = old }()
			if Inspect(m, "linux", "amd64").Packages[0].Provenance == "verified" {
				t.Fatal("tampered evidence retained verification")
			}
		})
	}
	m[receiptPath].Sys = &syscall.Stat_t{Uid: 1000}
	if Inspect(m, "linux", "amd64").Packages[0].Provenance == "verified" {
		t.Fatal("user-written receipt accepted")
	}
	m[receiptPath].Sys = &syscall.Stat_t{Uid: 0}
	delete(m, artifact)
	delete(m, artifact+".sig")
	if Inspect(m, "linux", "amd64").Packages[0].Provenance != "verified" {
		t.Fatal("transient cache cleanup invalidated durable authorization")
	}
	addFixture(m, "usr/share/pacman/keyrings/new-keyring.gpg", []byte("changed public trust inventory"), 0644)
	if Inspect(m, "linux", "amd64").Packages[0].Provenance == "verified" {
		t.Fatal("trust inventory change accepted")
	}
}

func TestActualVerificationRejectsPayloadSignatureAndUntrustedKey(t *testing.T) {
	m, home, artifact := authenticatedPacmanFixture(t)
	s := inspect(m, "linux", "amd64")
	verify := func(home string) error {
		v := verifier{fsys: m, run: fixtureRunner(t, m, home), authenticated: make(map[string][]digest)}
		_, err := v.verifyPackage(context.Background(), "pacman", s.report.Packages[0], s.owners["engine"])
		return err
	}
	old := m["usr/bin/dockerd"].Data
	m["usr/bin/dockerd"].Data = []byte("tampered executable")
	if verify(home) == nil {
		t.Fatal("authenticated package hid installed-file tampering")
	}
	m["usr/bin/dockerd"].Data = old
	sig := m[artifact+".sig"].Data
	m[artifact+".sig"].Data = []byte("forged signature")
	if verify(home) == nil {
		t.Fatal("forged signature accepted")
	}
	m[artifact+".sig"].Data = sig
	untrusted := t.TempDir()
	os.Chmod(untrusted, 0700)
	if err := os.WriteFile(filepath.Join(untrusted, "pubring.kbx"), m["etc/pacman.d/gnupg/pubring.kbx"].Data, 0600); err != nil {
		t.Fatal(err)
	}
	if verify(untrusted) == nil {
		t.Fatal("cryptographically valid but untrusted signer accepted")
	}
}

func TestAPTSignedIndexHashChainAndTamper(t *testing.T) {
	home := signingHome(t)
	m := debFixture()
	s := inspect(m, "linux", "amd64")
	owner := s.owners["engine"]
	// Debian tools are absent on Pacman hosts. The injected dpkg/apt adapters
	// supply their bounded outputs; gpgv verifies a real signed Release, and
	// every index, artifact and installed-file SHA256 link is exercised below.
	payload := makeTar(t, map[string][]byte{"./usr/sbin/dockerd": m["usr/sbin/dockerd"].Data, "./usr/bin/docker": m["usr/bin/docker"].Data})
	artifact := "var/cache/apt/archives/docker.io_29.9.1-1_amd64.deb"
	addFixture(m, artifact, []byte("fixture deb container with authenticated payload"), 0644)
	h := sha256.Sum256(m[artifact].Data)
	index := fmt.Sprintf("Package: docker.io\nVersion: %s\nArchitecture: amd64\nFilename: pool/docker.io.deb\nSize: %d\nSHA256: %x\n\n", owner.version, len(m[artifact].Data), h)
	indexName := "var/lib/apt/lists/archive_dists_noble_main_binary-amd64_Packages.lz4"
	addFixture(m, indexName, []byte(index), 0644)
	ih := sha256.Sum256([]byte(index))
	plain := fmt.Sprintf("Origin: Ubuntu\nDate: %s\nValid-Until: %s\nSHA256:\n %x %d main/binary-amd64/Packages\n", time.Now().Add(-time.Hour).UTC().Format(time.RFC1123Z), time.Now().Add(time.Hour).UTC().Format(time.RFC1123Z), ih, len(index))
	plain = "Suite: noble\n" + plain
	file := filepath.Join(t.TempDir(), "Release")
	os.WriteFile(file, []byte(plain), 0600)
	signed := nativeTestCommand(t, "/usr/bin/gpg", "--no-options", "--homedir", home, "--batch", "--pinentry-mode", "loopback", "--passphrase", "", "--output", "-", "--clearsign", file)
	releaseName := "var/lib/apt/lists/archive_dists_noble_InRelease"
	addFixture(m, releaseName, signed, 0644)
	key := nativeTestCommand(t, "/usr/bin/gpg", "--no-options", "--homedir", home, "--batch", "--export")
	addFixture(m, "usr/share/keyrings/ubuntu-archive-keyring.gpg", key, 0644)
	addFixture(m, "etc/apt/sources.list", []byte("deb [signed-by=/usr/share/keyrings/ubuntu-archive-keyring.gpg] https://archive noble main\n"), 0644)
	for _, tool := range []string{"usr/bin/gpgv", "usr/bin/dpkg-deb", "usr/lib/apt/apt-helper"} {
		addFixture(m, tool, []byte("fixture tool identity"), 0755)
	}
	real := fixtureRunner(t, m, home)
	run := func(ctx context.Context, tool string, args []string, limit int64) ([]byte, error) {
		switch tool {
		case "/usr/lib/apt/apt-helper":
			return m[strings.TrimPrefix(args[1], "/")].Data, nil
		case "/usr/bin/dpkg-deb":
			if args[0] == "--field" {
				return []byte("Package: docker.io\nVersion: " + owner.version + "\nArchitecture: amd64\n"), nil
			}
			return payload, nil
		default:
			return real(ctx, tool, args, limit)
		}
	}
	verify := func() (receipt, error) {
		v := verifier{fsys: m, run: run, authenticated: make(map[string][]digest)}
		return v.verifyPackage(context.Background(), "deb", s.report.Packages[0], owner)
	}
	rec, err := verify()
	if err != nil {
		t.Fatalf("signed APT chain: %v", err)
	}
	if rec.Method != aptMethod || len(rec.Evidence) != 3 {
		t.Fatal("APT evidence incomplete")
	}
	for _, name := range []string{releaseName, indexName, artifact, "usr/sbin/dockerd", "usr/share/keyrings/ubuntu-archive-keyring.gpg"} {
		t.Run("tamper/"+name, func(t *testing.T) {
			old := m[name].Data
			m[name].Data = []byte("tampered")
			defer func() { m[name].Data = old }()
			if _, err := verify(); err == nil {
				t.Fatal("APT chain accepted tampering")
			}
		})
	}
	b, _ := json.Marshal(fixtureReceiptSet(t, m, "deb", []receipt{rec}))
	addFixture(m, receiptPath, b, 0644)
	if Inspect(m, "linux", "amd64").Packages[0].Provenance != "verified" {
		t.Fatal("offline APT receipt not reusable")
	}
}

func fixtureReceiptSet(t *testing.T, m fstest.MapFS, provider string, recs []receipt) receiptSet {
	t.Helper()
	trust, err := trustSnapshot(m, provider)
	if err != nil {
		t.Fatal(err)
	}
	public, policy, err := publicTrustSnapshot(m, provider)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	return receiptSet{SchemaVersion: 2, PolicyID: PolicyID, Provider: provider, Trust: trust, PublicTrust: public, SourcePolicy: policy, VerifiedAt: now, ValidUntil: now.Add(ReceiptValidity), Packages: recs}
}

func TestArchiveAndSignatureBoundaries(t *testing.T) {
	for _, s := range []string{"[GNUPG:] VALIDSIG abc\n", "[GNUPG:] VALIDSIG abc\n[GNUPG:] TRUST_UNDEFINED\n", "[GNUPG:] VALIDSIG abc\n[GNUPG:] TRUST_FULLY\n[GNUPG:] REVKEYSIG abc\n"} {
		if trustedSignature([]byte(s)) {
			t.Fatal("untrusted signature status accepted")
		}
	}
	valid := "[GNUPG:] VALIDSIG " + strings.Repeat("A", 40) + " 2026-10-18 1792365388 0 4 0 22 10 00\n"
	if !trustedSignature([]byte(valid + "[GNUPG:] TRUST_FULLY 0 pgp\n")) {
		t.Fatal("trusted status rejected")
	}
	if trustedSignature([]byte(strings.Replace(valid, "22 10 00", "22 2 00", 1) + "[GNUPG:] TRUST_FULLY 0 pgp\n")) {
		t.Fatal("weak signature digest accepted")
	}
	if signatureStatus([]byte(valid+"[GNUPG:] EXPKEYSIG expired\n"), false) {
		t.Fatal("expired APT signer accepted")
	}
	for _, name := range []string{"../usr/bin/dockerd", "/usr/bin/dockerd", "usr/../usr/bin/dockerd"} {
		if _, err := hashTar(makeTar(t, map[string][]byte{name: []byte("payload")}), []string{"usr/bin/dockerd"}); err == nil {
			t.Fatal("unsafe archive name accepted")
		}
	}
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	tw.WriteHeader(&tar.Header{Name: "usr/bin/dockerd", Typeflag: tar.TypeSymlink, Linkname: "/outside"})
	tw.Close()
	if _, err := hashTar(buf.Bytes(), []string{"usr/bin/dockerd"}); err == nil {
		t.Fatal("archive symlink authenticated a file")
	}
	old := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC1123Z)
	if freshRelease([]byte("Date: " + old + "\nValid-Until: " + old + "\n")) {
		t.Fatal("expired release accepted on refresh")
	}
	if _, err := releaseHashes([]byte("SHA256:\n bad 1 ../../outside\n")); err == nil {
		t.Fatal("malformed Release hash accepted")
	}
}

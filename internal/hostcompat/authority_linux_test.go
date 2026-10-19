//go:build linux

package hostcompat

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeAPTFragmentSelectionAndEffectiveConfig(t *testing.T) {
	for _, name := range []string{"docker.sources", "docker.list", "vendor-2_amd64.sources", "vendor:docker.sources", "vendor:docker.list"} {
		if !nativeAPTFragment(name, false) {
			t.Fatalf("native source name rejected: %s", name)
		}
	}
	for _, name := range []string{".docker.sources", "docker.sources.disabled", "docker sources.list", "docker+.sources", "docker.sources~", ".vendor:docker.sources", "vendor:docker.sources.disabled"} {
		if nativeAPTFragment(name, false) {
			t.Fatalf("native-ignored source name admitted: %s", name)
		}
	}
	for _, name := range []string{"00-default", "99-authority.conf", "99:authority.conf", "99:authority"} {
		if !nativeAPTFragment(name, true) {
			t.Fatal("configuration fragment ignored")
		}
	}
	for _, name := range []string{".99-authority.conf", "99-authority.disabled", "99+authority.conf", ".99:authority.conf", "99:authority.conf.disabled", "99:authority."} {
		if nativeAPTFragment(name, true) {
			t.Fatal("native-ignored configuration loaded")
		}
	}
	for _, text := range []string{
		`Dir::Etc::sourcelist "/etc/apt/active.list";`,
		`Dir { Etc { sourceparts "/etc/apt/active.d"; }; };`,
		`Binary::apt-get::Dir::Etc::sourceparts "/etc/apt/active.d";`,
		`Dir::Etc::parts "/etc/apt/other-config";`,
		`Dir::Ignore-Files-Silently:: "docker.sources";`,
		`RootDir "/other-root";`,
		`#include "/etc/apt/other.conf"`,
		`#include"/etc/apt/other.conf"`,
		`#clear Dir::Etc;`,
		`Acquire::IndexTargets::deb::Packages::DefaultEnabled "false";`,
	} {
		if validateAPTConfiguration(text) == nil {
			t.Fatalf("effective override silently ignored: %s", text)
		}
	}
	for _, text := range []string{
		"# ordinary native comment\nAPT::Periodic::Update-Package-Lists \"1\";\n",
		`DPkg::Post-Invoke { "test -e /etc/apt/apt.conf && true"; }; Dir::Cache::pkgcache "";`,
		`DPkg::Pre-Install-Pkgs { "/usr/sbin/dpkg-preconfigure --apt || true"; }; DPkg::Tools::Options::/usr/sbin/dpkg-preconfigure::Version "2";`,
		`APT { NeverAutoRemove { "^linux-image.*"; }; }; Debug::NoLocking true;`,
		`/* Dir::Etc::sourcelist "ignored"; */ Acquire::Languages "none";`,
	} {
		if err := validateAPTConfiguration(text); err != nil {
			t.Fatalf("normal non-authority config rejected: %s", text)
		}
	}
}

func TestAuthorityRevocationWithdrawsReadersAndRootRefresh(t *testing.T) {
	for _, fault := range []string{"hidden-source", "redirect", "colon-redirect", "colon-no-extension-redirect", "nested-redirect", "environment"} {
		t.Run(fault, func(t *testing.T) {
			m, run, _, _ := dockerAPTFixture(t, false)
			v := verifier{fsys: m, run: run, store: fixtureStore(m)}
			set, _, err := v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), receiptSet{})
			if err != nil || len(set.Packages) != 2 {
				t.Fatal("initial verification", err)
			}
			publishFixtureSet(t, m, set)
			var override string
			switch fault {
			case "hidden-source":
				m["etc/apt/sources.list.d/.docker.sources"] = m["etc/apt/sources.list.d/docker.sources"]
				delete(m, "etc/apt/sources.list.d/docker.sources")
			case "redirect":
				addFixture(m, "etc/apt/apt.conf.d/99-authority", []byte(`Dir::Etc::sourcelist "/etc/apt/empty.list"; Dir::Etc::sourceparts "/etc/apt/empty.d";`), 0644)
			case "colon-redirect", "colon-no-extension-redirect":
				name := "etc/apt/apt.conf.d/99:authority"
				if fault == "colon-redirect" {
					name += ".conf"
				}
				addFixture(m, name, []byte(`Dir::Etc::sourcelist "/etc/apt/empty.list"; Dir::Etc::sourceparts "/etc/apt/empty.d";`), 0644)
			case "nested-redirect":
				addFixture(m, "etc/apt/apt.conf", []byte(`Dir { Etc { sourcelist "/etc/apt/empty.list"; sourceparts "/etc/apt/empty.d"; }; };`), 0644)
			case "environment":
				override = "/etc/apt/alternate.conf"
			}
			fsys := aptEnvironmentFS{m, override}
			if r := Inspect(fsys, "linux", "amd64"); r.Packages[0].Provenance == "verified" {
				t.Fatal("removed effective authority retained authorization")
			}
			v = verifier{fsys: fsys, run: run, store: fixtureStore(m)}
			next, failed, err := v.refreshPackages(context.Background(), inspect(fsys, "linux", "amd64"), set)
			if err != nil || !failed {
				t.Fatal("revocation did not yield an unresolved report", err)
			}
			for _, rec := range next.Packages {
				if rec.Authorized {
					t.Fatal("root reissued deleted authority")
				}
			}
		})
	}
}

func TestAPTConfigurationAuditBindingDoesNotConfuseHooksWithAuthority(t *testing.T) {
	m, run, _, _ := dockerAPTFixture(t, false)
	addFixture(m, "etc/apt/apt.conf.d/00:default.conf", []byte(`Acquire::Languages "none";`), 0644)
	v := verifier{fsys: m, run: run, store: fixtureStore(m)}
	set, _, err := v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), receiptSet{})
	if err != nil {
		t.Fatal(err)
	}
	publishFixtureSet(t, m, set)
	found := false
	for _, input := range set.Trust {
		if input.Path == "etc/apt/apt.conf.d/00:default.conf" {
			found = true
		}
	}
	if !found || set.SourcePolicy == "" {
		t.Fatal("trusted configuration inputs/model not recorded")
	}
	addFixture(m, "etc/apt/apt.conf.d/90:local-hook.conf", []byte(`DPkg::Post-Invoke { "root-owned-hook"; };`), 0644)
	if r := Inspect(m, "linux", "amd64"); r.Packages[0].Provenance != "verified" {
		t.Fatal("unrelated hook installation revoked unchanged source authority")
	}
	v = verifier{fsys: m, run: run, store: fixtureStore(m)}
	next, _, err := v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), set)
	if err != nil || len(next.Packages) != 2 {
		t.Fatal("unrelated colon-named config blocked root refresh", err)
	}
	for _, rec := range next.Packages {
		if !rec.Authorized {
			t.Fatal("unrelated colon-named config withdrew valid package authorization")
		}
	}
}

func TestRealMultiSignedReleaseWithSelectedKeyOnly(t *testing.T) {
	m, run, keyPath, home := dockerAPTFixture(t, false)
	release := "var/lib/apt/lists/download.docker.com_linux_ubuntu_dists_noble_InRelease"
	sources, _, err := aptSources(m)
	if err != nil {
		t.Fatal(err)
	}
	v := verifier{fsys: m, run: run, store: fixtureStore(m)}
	plain, err := v.signedRelease(context.Background(), sources[0], release, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"second archive signer", "third archive signer"} {
		nativeTestCommand(t, "/usr/bin/gpg", "--no-options", "--homedir", home, "--batch", "--pinentry-mode", "loopback", "--passphrase", "", "--quick-generate-key", name, "ed25519", "sign", "1d")
	}
	keys := nativeTestCommand(t, "/usr/bin/gpg", "--no-options", "--homedir", home, "--batch", "--with-colons", "--list-keys")
	var fingerprints []string
	for _, line := range strings.Split(string(keys), "\n") {
		f := strings.Split(line, ":")
		if len(f) > 9 && f[0] == "fpr" {
			fingerprints = append(fingerprints, f[9])
		}
	}
	if len(fingerprints) != 3 {
		t.Fatal("isolated multisigner fixture incomplete")
	}
	file := filepath.Join(t.TempDir(), "Release")
	if err := os.WriteFile(file, plain, 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"--no-options", "--homedir", home, "--batch", "--pinentry-mode", "loopback", "--passphrase", ""}
	for _, fingerprint := range fingerprints {
		args = append(args, "--local-user", fingerprint)
	}
	args = append(args, "--output", "-", "--clearsign", file)
	m[release].Data = nativeTestCommand(t, "/usr/bin/gpg", args...)
	status, nativeErr := run(context.Background(), "/usr/bin/gpgv", []string{"--keyring", "/" + keyPath, "--status-fd", "1", "/" + release}, 64<<10)
	if nativeErr == nil || !signatureExitAllowed(nativeErr) || strings.Count(string(status), "[GNUPG:] VALIDSIG") != 1 || strings.Count(string(status), "[GNUPG:] NO_PUBKEY") != 2 {
		t.Fatalf("native fixture not one valid plus two unavailable signatures: %v\n%s", nativeErr, status)
	}
	if got, err := v.signedRelease(context.Background(), sources[0], release, false); err != nil || strings.TrimRight(string(got), "\n") != strings.TrimRight(string(plain), "\n") {
		t.Fatal("authorized stable-key signature rejected", err)
	}
	snapshot := inspect(m, "linux", "amd64")
	if _, err := v.verifyPackage(context.Background(), "deb", snapshot.report.Packages[0], snapshot.owners["engine"]); err != nil {
		t.Fatal("multi-signed native package chain rejected", err)
	}
	// Verify the production native runner preserves bounded status on exit 2,
	// while non-status failures and output truncation remain strict.
	keyFile := filepath.Join(t.TempDir(), "selected.gpg")
	signedFile := filepath.Join(t.TempDir(), "InRelease")
	os.WriteFile(keyFile, m[keyPath].Data, 0600)
	os.WriteFile(signedFile, m[release].Data, 0600)
	root, err := os.OpenRoot("/")
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	native := nativeRunner(fixtureRoot{root})
	output, err := native(context.Background(), "/usr/bin/gpgv", []string{"--keyring", keyFile, "--status-fd", "1", signedFile}, 64<<10)
	if err == nil || !signatureExitAllowed(err) || !aptSignatureStatus(output) {
		t.Fatal("production runner discarded partial-success status", err)
	}
	if output, err := native(context.Background(), "/usr/bin/gpgv", []string{"--keyring", keyFile, "--status-fd", "1", signedFile}, 8); err == nil || len(output) != 0 {
		t.Fatal("bounded/truncated status was accepted")
	}
	if output, err := native(context.Background(), "/usr/bin/gpgv", []string{"--keyring", keyFile, "--output", "-", signedFile}, 4<<20); err == nil || len(output) != 0 {
		t.Fatal("non-status command exit was relaxed")
	}
	// A bad body cannot be accepted just because additional keys are missing.
	original := m[release].Data
	m[release].Data = []byte(strings.Replace(string(original), "Origin: Docker", "Origin: Forged", 1))
	if _, err := v.signedRelease(context.Background(), sources[0], release, false); err == nil {
		t.Fatal("invalid-only signatures accepted")
	}
	m[release].Data = original
	// One selected signer revoked, another selected signer still good: refuse
	// known selected-key revocation rather than hiding it behind the good one.
	rev, err := os.ReadFile(filepath.Join(home, "openpgp-revocs.d", fingerprints[0]+".rev"))
	if err != nil {
		t.Fatal(err)
	}
	start := strings.Index(string(rev), ":-----BEGIN PGP PUBLIC KEY BLOCK-----")
	if start < 0 {
		t.Fatal("fixture revocation missing")
	}
	revFile := filepath.Join(t.TempDir(), "revoke.asc")
	os.WriteFile(revFile, rev[start+1:], 0600)
	nativeTestCommand(t, "/usr/bin/gpg", "--no-options", "--homedir", home, "--batch", "--import", revFile)
	m[keyPath].Data = nativeTestCommand(t, "/usr/bin/gpg", "--no-options", "--homedir", home, "--batch", "--export", fingerprints[0], fingerprints[1])
	if _, err := v.signedRelease(context.Background(), sources[0], release, false); err == nil {
		t.Fatal("revoked selected signer ignored")
	}
}

func TestAPTSignatureRecordsAndCleartextBoundaries(t *testing.T) {
	valid := "[GNUPG:] NEWSIG\n[GNUPG:] VALIDSIG " + strings.Repeat("A", 40) + " 2026-10-19 1792374975 0 4 0 22 10 01\n"
	missing := "[GNUPG:] NEWSIG\n[GNUPG:] ERRSIG " + strings.Repeat("B", 16) + " 22 10 01 1792374975 9\n[GNUPG:] NO_PUBKEY " + strings.Repeat("B", 16) + "\n"
	if !aptSignatureStatus([]byte(valid+missing)) || aptSignatureStatus([]byte(missing)) {
		t.Fatal("selected/unknown signature accounting")
	}
	for _, bad := range []string{strings.Replace(missing, " 9\n", " 4\n", 1), strings.Replace(missing, "NO_PUBKEY "+strings.Repeat("B", 16), "NO_PUBKEY "+strings.Repeat("C", 16), 1), "[GNUPG:] NEWSIG\n[GNUPG:] BADSIG " + strings.Repeat("B", 16) + "\n", "[GNUPG:] NEWSIG\n[GNUPG:] REVKEYSIG " + strings.Repeat("B", 16) + "\n"} {
		if aptSignatureStatus([]byte(valid + bad)) {
			t.Fatal("invalid selected/malformed signature ignored")
		}
	}
	for _, bad := range []string{"unsigned text", "-----BEGIN PGP SIGNED MESSAGE-----\nHash: SHA512\n\nSuite: noble\n", "-----BEGIN PGP SIGNED MESSAGE-----\nHash: SHA512\n\nSuite: noble\n-----BEGIN PGP SIGNATURE-----\nfixture\n-----END PGP SIGNATURE-----\nunsigned trailing text"} {
		if _, err := releaseCleartext([]byte(bad)); err == nil {
			t.Fatal("ambiguous/truncated envelope accepted")
		}
	}
}

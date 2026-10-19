//go:build linux

package hostcompat

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func fixtureStore(m fstest.MapFS) func(context.Context, []byte, string) (string, error) {
	return func(ctx context.Context, b []byte, suffix string) (string, error) {
		if ctx.Err() != nil {
			return "", errEvidence
		}
		if suffix == "" {
			suffix = ".evidence"
		}
		h := sha256.Sum256(b)
		name := strings.TrimPrefix(VerificationCacheDirectory, "/") + "/" + hex.EncodeToString(h[:]) + suffix
		if f := m[name]; f != nil {
			if actual := sha256.Sum256(f.Data); actual != h {
				return "", errEvidence
			}
		} else {
			addFixture(m, name, append([]byte{}, b...), 0600)
		}
		return name, nil
	}
}

func publishFixtureSet(t *testing.T, m fstest.MapFS, set receiptSet) {
	t.Helper()
	b, err := json.Marshal(set)
	if err != nil {
		t.Fatal(err)
	}
	addFixture(m, receiptPath, b, 0644)
}

// These paths are inaccessible to the ordinary runtime UID. Any attempt to read
// them is a test failure, independently of fstest.MapFS ignoring Unix modes.
type runtimeReadFS struct {
	fs.FS
	t *testing.T
}

func (r runtimeReadFS) check(name string) error {
	for _, prefix := range []string{"etc/pacman.d/gnupg", "var/cache/", "var/lib/apt/lists", strings.TrimPrefix(VerificationCacheDirectory, "/"), "var/lib/omabit/hostcompat/.fetch-"} {
		if strings.HasPrefix(name, prefix) {
			r.t.Errorf("runtime reader accessed private/transient evidence: %s", name)
			return fs.ErrPermission
		}
	}
	return nil
}
func (r runtimeReadFS) Open(name string) (fs.File, error) {
	if err := r.check(name); err != nil {
		return nil, err
	}
	return r.FS.Open(name)
}
func (r runtimeReadFS) Stat(name string) (fs.FileInfo, error) {
	if err := r.check(name); err != nil {
		return nil, err
	}
	return fs.Stat(r.FS, name)
}
func (r runtimeReadFS) Lstat(name string) (fs.FileInfo, error) {
	if err := r.check(name); err != nil {
		return nil, err
	}
	return fs.Lstat(r.FS, name)
}
func (r runtimeReadFS) ReadLink(name string) (string, error) {
	if err := r.check(name); err != nil {
		return "", err
	}
	return fs.ReadLink(r.FS, name)
}

func TestRootRefreshRetainsProofAndNonRootReadsSurviveCachePrune(t *testing.T) {
	m, home, artifact := authenticatedPacmanFixture(t)
	v := verifier{fsys: m, run: fixtureRunner(t, m, home), store: fixtureStore(m)}
	set, _, err := v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), receiptSet{})
	if err != nil || len(set.Packages) != 2 {
		t.Fatalf("first refresh: %+v %v", set, err)
	}
	for _, rec := range set.Packages {
		if !rec.Authorized || len(rec.Retained) != 2 {
			t.Fatal("signed proof was not retained")
		}
	}
	publishFixtureSet(t, m, set)
	delete(m, artifact)
	delete(m, artifact+".sig")
	m["etc/pacman.d/gnupg"].Mode = fs.ModeDir | 0700
	m["etc/pacman.d/gnupg/trustdb.gpg"].Mode = 0600
	r := Inspect(runtimeReadFS{m, t}, "linux", "amd64")
	if r.Packages[0].Provenance != "verified" || r.Packages[1].Provenance != "verified" {
		t.Fatalf("non-root boot after OS prune: %+v", r.Packages)
	}
	// An unrelated package database edit does not revoke unchanged role tuples.
	addFixture(m, "var/lib/pacman/local/unrelated-1/desc", []byte("%NAME%\nunrelated\n\n%VERSION%\n1-1\n\n%ARCH%\nx86_64\n"), 0644)
	addFixture(m, "var/lib/pacman/local/unrelated-1/files", []byte("%FILES%\nusr/share/unrelated\n"), 0644)
	v = verifier{fsys: m, run: fixtureRunner(t, m, home), store: fixtureStore(m)}
	next, _, err := v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), set)
	if err != nil || len(next.Packages) != 2 || !next.Packages[0].Authorized || !next.Packages[1].Authorized {
		t.Fatalf("post-update retained-proof reuse: %+v %v", next, err)
	}
	publishFixtureSet(t, m, next)
	if Inspect(runtimeReadFS{m, t}, "linux", "amd64").Packages[0].Provenance != "verified" {
		t.Fatal("unchanged roles lost authorization")
	}
	// The retained artifact is audit/revalidation evidence, not a runtime input.
	for _, rec := range next.Packages {
		for _, item := range rec.Retained {
			delete(m, item.Path)
		}
	}
	if Inspect(runtimeReadFS{m, t}, "linux", "amd64").Packages[0].Provenance != "verified" {
		t.Fatal("audit-cache pruning revoked currently authenticated bytes")
	}
	failed, _, err := v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), next)
	if err != nil {
		t.Fatal(err)
	}
	for _, rec := range failed.Packages {
		if rec.Authorized {
			t.Fatal("missing proof was silently renewed")
		}
	}
	publishFixtureSet(t, m, failed)
	if Inspect(runtimeReadFS{m, t}, "linux", "amd64").Packages[0].Provenance == "verified" {
		t.Fatal("withdrawn authorization still admitted")
	}
}

func TestRootRefreshRejectsRevokedSignerEvenWithUnchangedPrivateTrustDB(t *testing.T) {
	m, home, artifact := authenticatedPacmanFixture(t)
	v := verifier{fsys: m, run: fixtureRunner(t, m, home), store: fixtureStore(m)}
	set, _, err := v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), receiptSet{})
	if err != nil {
		t.Fatal(err)
	}
	publishFixtureSet(t, m, set)
	delete(m, artifact)
	delete(m, artifact+".sig")
	status, err := v.run(context.Background(), "/usr/bin/gpg", []string{"--no-options", "--homedir", "/etc/pacman.d/gnupg", "--batch", "--status-fd", "1", "--verify", "/" + set.Packages[0].Retained[1].Path, "/" + set.Packages[0].Retained[0].Path}, 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	var fingerprint string
	for _, line := range strings.Split(string(status), "\n") {
		f := strings.Fields(line)
		if len(f) > 2 && f[1] == "VALIDSIG" {
			fingerprint = f[2]
		}
	}
	if fingerprint == "" {
		t.Fatal("fixture signer missing")
	}
	m["usr/share/pacman/keyrings/fixture-revoked"].Data = []byte(fingerprint + "\n")
	if Inspect(runtimeReadFS{m, t}, "linux", "amd64").Packages[0].Provenance == "verified" {
		t.Fatal("public revocation did not invalidate receipt immediately")
	}
	v = verifier{fsys: m, run: fixtureRunner(t, m, home), store: fixtureStore(m)}
	next, _, err := v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), set)
	if err != nil {
		t.Fatal(err)
	}
	for _, rec := range next.Packages {
		if rec.Authorized {
			t.Fatal("revoked signer renewed from stale private keyring")
		}
	}
}

func TestNinetyDaysOfflineAndInstalledPayloadTampering(t *testing.T) {
	m, home, _ := authenticatedPacmanFixture(t)
	v := verifier{fsys: m, run: fixtureRunner(t, m, home), store: fixtureStore(m)}
	set, _, err := v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), receiptSet{})
	if err != nil {
		t.Fatal(err)
	}
	set.VerifiedAt = time.Now().Add(-90 * 24 * time.Hour)
	set.ValidUntil = set.VerifiedAt.Add(ReceiptValidity)
	publishFixtureSet(t, m, set)
	if Inspect(runtimeReadFS{m, t}, "linux", "amd64").Packages[0].Provenance != "verified" {
		t.Fatal("unchanged authenticated package expired during an offline boot")
	}
	set, _, err = v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), set)
	if err != nil {
		t.Fatal(err)
	}
	publishFixtureSet(t, m, set)
	if Inspect(runtimeReadFS{m, t}, "linux", "amd64").Packages[0].Provenance != "verified" {
		t.Fatal("local retained proof did not renew offline")
	}
	m["usr/bin/dockerd"].Data = []byte("changed installed executable")
	if Inspect(runtimeReadFS{m, t}, "linux", "amd64").Packages[0].Provenance == "verified" {
		t.Fatal("installed-byte tampering accepted")
	}
	set, _, err = v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), set)
	if err != nil {
		t.Fatal(err)
	}
	for _, rec := range set.Packages {
		if rec.Role == "engine" && rec.Authorized {
			t.Fatal("tampered payload renewed")
		}
		if rec.Role == "cli" && !rec.Authorized {
			t.Fatal("unchanged authenticated role lost authorization")
		}
	}
}

func TestMutationDuringAuthenticationAndRetentionRefusesPublication(t *testing.T) {
	t.Run("artifact-after-signature", func(t *testing.T) {
		m, home, artifact := authenticatedPacmanFixture(t)
		s := inspect(m, "linux", "amd64")
		real := fixtureRunner(t, m, home)
		v := verifier{fsys: m, store: fixtureStore(m), run: func(ctx context.Context, tool string, args []string, limit int64) ([]byte, error) {
			b, err := real(ctx, tool, args, limit)
			if tool == "/usr/bin/gpg" {
				m[artifact].Data = append(append([]byte{}, m[artifact].Data...), byte('!'))
			}
			return b, err
		}}
		if _, err := v.verifyPackage(context.Background(), "pacman", s.report.Packages[0], s.owners["engine"]); err == nil {
			t.Fatal("artifact mutation raced past authentication")
		}
	})
	t.Run("cache-after-copy", func(t *testing.T) {
		m, home, _ := authenticatedPacmanFixture(t)
		s := inspect(m, "linux", "amd64")
		store := fixtureStore(m)
		v := verifier{fsys: m, run: fixtureRunner(t, m, home), store: func(ctx context.Context, b []byte, suffix string) (string, error) {
			name, err := store(ctx, b, suffix)
			if err == nil {
				m[name].Data = []byte("tampered retained proof")
			}
			return name, err
		}}
		rec, err := v.verifyPackage(context.Background(), "pacman", s.report.Packages[0], s.owners["engine"])
		if err != nil {
			t.Fatal(err)
		}
		if _, err := v.retainReceipt(context.Background(), rec); err == nil {
			t.Fatal("tampered retained proof published")
		}
	})
	t.Run("metadata-during-payload", func(t *testing.T) {
		m, home, _ := authenticatedPacmanFixture(t)
		s := inspect(m, "linux", "amd64")
		real := fixtureRunner(t, m, home)
		v := verifier{fsys: m, store: fixtureStore(m), run: func(ctx context.Context, tool string, args []string, limit int64) ([]byte, error) {
			b, err := real(ctx, tool, args, limit)
			if tool == "/usr/bin/bsdtar" && args[0] == "-cf" {
				m["var/lib/pacman/local/docker-29.9.1-1/desc"].Data = bytesReplaceVersion(m["var/lib/pacman/local/docker-29.9.1-1/desc"].Data)
			}
			return b, err
		}}
		if _, err := v.verifyPackage(context.Background(), "pacman", s.report.Packages[0], s.owners["engine"]); err == nil {
			t.Fatal("native identity changed during verification")
		}
	})
}

func bytesReplaceVersion(b []byte) []byte {
	return []byte(strings.ReplaceAll(string(b), "29.9.1", "29.9.2"))
}

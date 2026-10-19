//go:build linux

package hostcompat

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"testing/fstest"
)

func installSignedFixturePackage(t *testing.T, m fstest.MapFS, home, name, version string, payload map[string][]byte) {
	t.Helper()
	for file := range m {
		if strings.HasPrefix(file, "var/lib/pacman/local/"+name+"-") {
			delete(m, file)
		}
	}
	metadata := "var/lib/pacman/local/" + name + "-" + version
	addFixture(m, metadata+"/desc", []byte("%NAME%\n"+name+"\n\n%VERSION%\n"+version+"\n\n%ARCH%\nx86_64\n\n%VALIDATION%\npgp\n"), 0644)
	files := "%FILES%\n"
	archiveFiles := map[string][]byte{".PKGINFO": []byte("pkgname = " + name + "\npkgver = " + version + "\narch = x86_64\n")}
	for file, data := range payload {
		files += file + "\n"
		archiveFiles[file] = data
		addFixture(m, file, data, 0755)
	}
	addFixture(m, metadata+"/files", []byte(files), 0644)
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.Write(makeTar(t, archiveFiles))
	gz.Close()
	cacheVersion := version
	if _, after, ok := strings.Cut(version, ":"); ok {
		cacheVersion = after
	}
	artifact := "var/cache/pacman/pkg/" + name + "-" + cacheVersion + "-x86_64.pkg.tar.gz"
	addFixture(m, artifact, buf.Bytes(), 0644)
	file := filepath.Join(t.TempDir(), "package")
	if err := os.WriteFile(file, buf.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	sig := nativeTestCommand(t, "/usr/bin/gpg", "--no-options", "--homedir", home, "--batch", "--pinentry-mode", "loopback", "--passphrase", "", "--output", "-", "--detach-sign", file)
	addFixture(m, artifact+".sig", sig, 0644)
}

func TestPostUpdateRefreshCopiesChangedPackageAndKeepsUnchangedRoles(t *testing.T) {
	m, home, _ := authenticatedPacmanFixture(t)
	installSignedFixturePackage(t, m, home, "rootlesskit", "2.3.6-1", map[string][]byte{"usr/bin/rootlesskit": []byte("authenticated unchanged rootlesskit")})
	v := verifier{fsys: m, run: fixtureRunner(t, m, home), store: fixtureStore(m)}
	old, _, err := v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), receiptSet{})
	if err != nil || len(old.Packages) != 3 {
		t.Fatalf("initial refresh: %+v %v", old, err)
	}
	publishFixtureSet(t, m, old)
	for name := range m {
		if strings.HasPrefix(name, "var/cache/pacman/pkg/") {
			delete(m, name)
		}
	}
	installSignedFixturePackage(t, m, home, "docker", "1:29.9.2-1", map[string][]byte{"usr/bin/dockerd": []byte("new authenticated dockerd"), "usr/bin/docker": []byte("new authenticated docker CLI")})
	v = verifier{fsys: m, run: fixtureRunner(t, m, home), store: fixtureStore(m)}
	updated, _, err := v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), old)
	if err != nil || len(updated.Packages) != 3 {
		t.Fatalf("update refresh: %+v %v", updated, err)
	}
	for _, rec := range updated.Packages {
		if !rec.Authorized {
			t.Fatalf("role lost authorization: %+v", rec)
		}
		if rec.Role == "engine" && rec.Version != "1:29.9.2-1" {
			t.Fatal("old version relabeled")
		}
	}
	publishFixtureSet(t, m, updated)
	for name := range m {
		if strings.HasPrefix(name, "var/cache/pacman/pkg/") {
			delete(m, name)
		}
	}
	r := Inspect(runtimeReadFS{m, t}, "linux", "amd64")
	if r.InstalledEngineVersion != "29.9.2" || r.Packages[0].Provenance != "verified" {
		t.Fatalf("new copied evidence not readable after cleanup: %+v", r)
	}
	v = verifier{fsys: m, run: fixtureRunner(t, m, home), store: fixtureStore(m)}
	reused, _, err := v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), updated)
	if err != nil {
		t.Fatal(err)
	}
	for _, rec := range reused.Packages {
		if !rec.Authorized {
			t.Fatal("updated package could not refresh from retained proof")
		}
	}
}

func revokeFixtureSigner(t *testing.T, home string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(home, "openpgp-revocs.d"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatal("isolated signer revocation certificate missing")
	}
	b, err := os.ReadFile(filepath.Join(home, "openpgp-revocs.d", entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	start := bytes.Index(b, []byte(":-----BEGIN PGP PUBLIC KEY BLOCK-----"))
	if start < 0 {
		t.Fatal("fixture revocation armor missing")
	}
	name := filepath.Join(t.TempDir(), "revoke.asc")
	if err := os.WriteFile(name, b[start+1:], 0600); err != nil {
		t.Fatal(err)
	}
	// This import is solely into the isolated temporary test keyring.
	nativeTestCommand(t, "/usr/bin/gpg", "--no-options", "--homedir", home, "--batch", "--import", name)
}

func TestRealAPTRevocationRejectsPreviouslyAuthenticatedRetainedRelease(t *testing.T) {
	m, run, keyPath, home := dockerAPTFixture(t, true)
	v := verifier{fsys: m, run: run, store: fixtureStore(m)}
	old, _, err := v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), receiptSet{})
	if err != nil || len(old.Packages) == 0 {
		t.Fatalf("initial crypto validation: %v", err)
	}
	publishFixtureSet(t, m, old)
	for name := range m {
		if strings.HasPrefix(name, "var/cache/apt/archives/") || strings.HasPrefix(name, "var/lib/apt/lists/") {
			delete(m, name)
		}
	}
	revokeFixtureSigner(t, home)
	m[keyPath].Data = nativeTestCommand(t, "/usr/bin/gpg", "--no-options", "--homedir", home, "--batch", "--armor", "--export")
	if Inspect(runtimeReadFS{m, t}, "linux", "amd64").Packages[0].Provenance == "verified" {
		t.Fatal("revoked source authority not noticed by runtime reader")
	}
	v = verifier{fsys: m, run: run, store: fixtureStore(m)}
	set, _, err := v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), old)
	if err != nil {
		t.Fatal(err)
	}
	for _, rec := range set.Packages {
		if rec.Authorized {
			t.Fatal("real revoked signer renewed retained Release")
		}
	}
}

// Map real temporary-file ownership to the injected root while preserving inode,
// ctime, mode and size. This exercises a real same-size replacement during Read.
type rootInfo struct{ fs.FileInfo }

func (i rootInfo) Sys() any { s := *i.FileInfo.Sys().(*syscall.Stat_t); s.Uid = 0; return &s }

type racingFile struct {
	fs.File
	mutate func()
	done   bool
}

func (f *racingFile) Stat() (fs.FileInfo, error) {
	i, e := f.File.Stat()
	if e != nil {
		return nil, e
	}
	return rootInfo{i}, nil
}
func (f *racingFile) Read(b []byte) (int, error) {
	n, e := f.File.Read(b)
	if n > 0 && f.mutate != nil && !f.done {
		f.done = true
		f.mutate()
	}
	return n, e
}

func (f *racingFile) ReadDir(n int) ([]fs.DirEntry, error) {
	d, ok := f.File.(fs.ReadDirFile)
	if !ok {
		return nil, errEvidence
	}
	return d.ReadDir(n)
}

type racingFS struct {
	fs.FS
	mutate func()
}

func (r racingFS) Stat(name string) (fs.FileInfo, error) {
	i, e := fs.Stat(r.FS, name)
	if e != nil {
		return nil, e
	}
	return rootInfo{i}, nil
}
func (r racingFS) Lstat(name string) (fs.FileInfo, error) {
	i, e := fs.Lstat(r.FS, name)
	if e != nil {
		return nil, e
	}
	return rootInfo{i}, nil
}
func (r racingFS) ReadLink(name string) (string, error) { return fs.ReadLink(r.FS, name) }
func (r racingFS) Open(name string) (fs.File, error) {
	f, e := r.FS.Open(name)
	if e != nil {
		return nil, e
	}
	return &racingFile{File: f, mutate: r.mutate}, nil
}

func TestStrongHashRejectsSameSizeReplacementDuringRead(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "binary")
	if err := os.WriteFile(name, []byte("trusted"), 0644); err != nil {
		t.Fatal(err)
	}
	r := racingFS{FS: os.DirFS(dir), mutate: func() {
		tmp := filepath.Join(dir, "replacement")
		if err := os.WriteFile(tmp, []byte("changed"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(tmp, name); err != nil {
			t.Fatal(err)
		}
	}}
	if _, err := hashTrusted(r, "binary", maxBinary); err == nil {
		t.Fatal("hash of replaced filesystem object accepted")
	}
}

func TestInspectionDigestMemoRechecksObjectIdentity(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "binary")
	if err := os.WriteFile(name, []byte("trusted"), 0644); err != nil {
		t.Fatal(err)
	}
	reads := 0
	r := racingFS{FS: os.DirFS(dir), mutate: func() { reads++ }}
	checker := newDigestChecker(r)
	h := sha256.Sum256([]byte("trusted"))
	proof := []digest{{"binary", hex.EncodeToString(h[:])}}
	if !checker.matches(proof, 64) || !checker.matches(proof, 64) || reads != 1 {
		t.Fatal("unchanged critical file was not safely deduplicated")
	}
	tmp := filepath.Join(dir, "replacement")
	if err := os.WriteFile(tmp, []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmp, name); err != nil {
		t.Fatal(err)
	}
	if checker.matches(proof, 64) || reads != 2 {
		t.Fatal("memoized hash ignored changed filesystem identity")
	}
}

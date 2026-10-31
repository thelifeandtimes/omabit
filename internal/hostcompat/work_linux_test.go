//go:build linux

package hostcompat

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func materializeFixture(t *testing.T, m fs.FS) string {
	t.Helper()
	dir := t.TempDir()
	if err := fs.WalkDir(m, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name == "." {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		target := filepath.Join(dir, name)
		if entry.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		b, err := fs.ReadFile(m, name)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, info.Mode().Perm())
	}); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestInspectCurrentDigestCacheDetectsOutOfPackageTamperWithRestoredMtime(t *testing.T) {
	m, home, _ := authenticatedPacmanFixture(t)
	payload := map[string][]byte{}
	for _, role := range requiredRoles {
		payload["usr/bin/"+role.executable] = []byte("signed " + role.executable)
	}
	installSignedFixturePackage(t, m, home, "docker", "1:29.9.1-1", payload)
	v := verifier{fsys: m, run: fixtureRunner(t, m, home), store: fixtureStore(m)}
	set, failed, err := v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), receiptSet{})
	if err != nil || failed {
		t.Fatal("fixture verification failed", err)
	}
	publishFixtureSet(t, m, set)
	dir := materializeFixture(t, m)
	root := racingFS{FS: os.DirFS(dir)}
	cache := &fileDigestCache{}
	check := func() (Report, *evidenceWork) {
		f, w := withWork(root, inspectionLimits, cache)
		return inspectCurrentFS(f, "linux", "amd64", cache), w
	}
	r, cold := check()
	if r.Provenance != "verified" || cold.hashes == 0 {
		t.Fatalf("cold evidence: %+v", r)
	}
	r, warm := check()
	if r.Provenance != "verified" || warm.hashes != 0 {
		t.Fatalf("unchanged RPC/admission rehashed files: %d %+v", warm.hashes, r)
	}
	file := filepath.Join(dir, "usr/bin/dockerd")
	before, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	b[0] ^= 1
	if err := os.WriteFile(file, b, before.Mode().Perm()); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(file, before.ModTime(), before.ModTime()); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	a, bstat := before.Sys().(*syscall.Stat_t), after.Sys().(*syscall.Stat_t)
	if a.Ino != bstat.Ino || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) || a.Ctim == bstat.Ctim {
		t.Fatal("fixture did not preserve inode/size/mtime while changing ctime")
	}
	r, changed := check()
	if r.Packages[0].Provenance == "verified" || changed.hashes == 0 {
		t.Fatal("restored mtime bypassed actual-identity cache invalidation")
	}
	e := goodEngine("29.9.1")
	r.Engine = &e
	r = Evaluate(r, e.DataRoot)
	if r.CanMutate || !r.CanStop {
		t.Fatal("out-of-package mutation bypassed admission or disabled safe Stop")
	}
}

func TestAggregateInspectionBudgets(t *testing.T) {
	m := pacmanFixture()
	limits := inspectionLimits
	limits.hashes = 6
	f, w := withWork(m, limits, nil)
	addFixture(m, "usr/bin/one", []byte("12345"), 0755)
	addFixture(m, "usr/bin/two", []byte("12345"), 0755)
	if _, err := hashTrusted(f, "usr/bin/one", maxBinary); err != nil {
		t.Fatal(err)
	}
	if _, err := hashTrusted(f, "usr/bin/two", maxBinary); err == nil || !w.exhausted || w.hashes > 6 {
		t.Fatal("aggregate hash budget ignored")
	}
	limits = inspectionLimits
	limits.metadata = 1
	limits.reads = 1
	f, w = withWork(m, limits, nil)
	r := Inspect(f, "linux", "amd64")
	if !w.exhausted || r.Provenance == "verified" {
		t.Fatal("metadata budget did not fail closed")
	}
	found := false
	for _, c := range r.Checks {
		if c.ID == "packages.budget" {
			found = true
		}
	}
	if !found {
		t.Fatal("budget exhaustion was not reported")
	}
	limits = inspectionLimits
	limits.entries = 1
	f, w = withWork(m, limits, nil)
	if _, err := readDir(f, "usr/bin"); err == nil || !w.exhausted {
		t.Fatal("aggregate directory entry budget ignored")
	}
}

type repeatedReader struct {
	piece             []byte
	remaining, offset int
}

func (r *repeatedReader) Read(b []byte) (int, error) {
	if r.remaining == 0 {
		return 0, io.EOF
	}
	n := copy(b, r.piece[r.offset:])
	r.offset += n
	if r.offset == len(r.piece) {
		r.offset = 0
		r.remaining--
	}
	return n, nil
}

func TestTargetedStreamingUbuntuSizedPackageIndexAndSharedLookup(t *testing.T) {
	piece := []byte("Package: unrelated\nVersion: 1\nArchitecture: amd64\nDescription: " + strings.Repeat("d", 1024) + "\n\n")
	target := []byte("Package: target\nVersion: 29.9.1-1\nArchitecture: amd64\nSize: 1\nSHA256: " + strings.Repeat("a", 64) + "\nFilename: pool/target.deb\n\n")
	reader := func() io.Reader {
		unrelated := strings.NewReader("Package: large-provides\nVersion: 1\nArchitecture: amd64\nProvides: " + strings.Repeat("p", 70841) + "\n\n")
		return io.MultiReader(&repeatedReader{piece: piece, remaining: 40000}, unrelated, bytes.NewReader(target))
	}
	h := sha256.New()
	size, err := io.Copy(h, reader())
	if err != nil || size <= maxMetadata {
		t.Fatal("large-index fixture is not larger than metadata limit")
	}
	m := fixtureFS(map[string]string{"var/lib/apt/lists/fixture_Packages.lz4": "compressed fixture"})
	f, w := withWork(m, verificationLimits, nil)
	calls := 0
	v := verifier{fsys: f, aptTargets: map[string]bool{packageTuple("target", "29.9.1-1", "amd64"): true, packageTuple("also-needed", "1", "amd64"): true}, stream: boundStreams(func(_ context.Context, _ string, _ []string, limit int64, consume func(io.Reader) error) error {
		calls++
		if limit < size {
			t.Fatal("normal large index does not fit bounds")
		}
		return consume(reader())
	}, w)}
	index := releaseEntry{hash: hex.EncodeToString(h.Sum(nil)), name: "universe/binary-amd64/Packages", size: size}
	packages, _, err := v.aptIndex(context.Background(), "var/lib/apt/lists/fixture_Packages.lz4", index, nativePackage{name: "target", version: "29.9.1-1", arch: "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	if len(packages) != 1 || packages[0]["Package"] != "target" || w.records != 40002 {
		t.Fatal("unrelated repository paragraphs retained or records unbounded")
	}
	if _, _, err := v.aptIndex(context.Background(), "var/lib/apt/lists/fixture_Packages.lz4", index, nativePackage{name: "also-needed", version: "1", arch: "amd64"}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || w.commands != size {
		t.Fatal("shared repository index was decoded again for another role")
	}
}

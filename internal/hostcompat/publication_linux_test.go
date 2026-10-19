//go:build linux

package hostcompat

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestRestrictiveUmaskPublicAndPrivatePublication(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "var/lib"), 0755); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	r := fixtureRoot{root}
	old := syscall.Umask(0077)
	defer syscall.Umask(old)
	if err := prepareVerificationDirectories(r); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(receiptSet{SchemaVersion: 2, PolicyID: PolicyID, Provider: "pacman"})
	if err := atomicRootFile(r, receiptPath, b, 0644); err != nil {
		t.Fatal(err)
	}
	private, err := cacheStore(r)(context.Background(), []byte("private proof"), ".sig")
	if err != nil {
		t.Fatal(err)
	}
	for name, mode := range map[string]fs.FileMode{"var/lib/omabit": 0755, "var/lib/omabit/hostcompat": 0755, "var/lib/omabit/hostcompat/evidence-v2": 0700, receiptPath: 0644, private: 0600} {
		info, err := root.Stat(name)
		if err != nil || info.Mode().Perm() != mode {
			t.Fatalf("%s: mode=%v err=%v; want %o", name, info, err, mode)
		}
	}
	// Recognize and repair only the former root-owned generated layout.
	for _, name := range []string{"var/lib/omabit", "var/lib/omabit/hostcompat"} {
		if err := root.Chmod(name, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := prepareVerificationDirectories(r); err != nil {
		t.Fatal("known prior generation did not recover", err)
	}
	for _, name := range []string{"var/lib/omabit", "var/lib/omabit/hostcompat"} {
		info, _ := root.Stat(name)
		if info.Mode().Perm() != 0755 {
			t.Fatal("prior hook permissions not repaired")
		}
	}
}

func TestPublicDirectoryPreparationPreservesAdministratorTree(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "var/lib/omabit"), 0700); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := root.WriteFile("var/lib/omabit/administrator.secret", []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := prepareVerificationDirectories(fixtureRoot{root}); err == nil {
		t.Fatal("unrecognized private administrator tree widened")
	}
	info, _ := root.Stat("var/lib/omabit")
	if info.Mode().Perm() != 0700 {
		t.Fatal("foreign parent permissions changed")
	}
	if b, err := root.ReadFile("var/lib/omabit/administrator.secret"); err != nil || string(b) != "preserve" {
		t.Fatal("administrator file changed")
	}
}

// Run this test as root in the native guest or a disposable mapped user
// namespace. It modifies only its temp root and starts a real UID-1000 reader.
func TestRootHookPublicationReadableByUID1000(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("requires root in a disposable guest/user namespace")
	}
	m, home, _ := authenticatedPacmanFixture(t)
	payload := map[string][]byte{}
	for _, role := range requiredRoles {
		payload["usr/bin/"+role.executable] = []byte("signed fixture " + role.executable)
	}
	installSignedFixturePackage(t, m, home, "docker", "1:29.7.2-1", payload)
	v := verifier{fsys: m, run: fixtureRunner(t, m, home), store: fixtureStore(m)}
	set, failed, err := v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), receiptSet{})
	if err != nil || failed {
		t.Fatalf("fixture crypto verification: %v", err)
	}
	for name := range m {
		if strings.HasPrefix(name, "var/lib/omabit") {
			delete(m, name)
		}
	}
	dir := materializeFixture(t, m)
	for _, name := range []string{filepath.Dir(dir), dir} {
		if err := os.Chmod(name, 0755); err != nil {
			t.Fatal(err)
		}
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	func() {
		old := syscall.Umask(0077)
		defer syscall.Umask(old)
		if err := prepareVerificationDirectory(root); err != nil {
			t.Fatal(err)
		}
		if err := writeReceipts(root, set); err != nil {
			t.Fatal(err)
		}
		if _, err := cacheStore(root)(context.Background(), []byte("private retained proof"), ".sig"); err != nil {
			t.Fatal(err)
		}
	}()
	if report := Inspect(root.FS(), "linux", "amd64"); report.Provenance != "verified" {
		t.Fatalf("root report not verified: %+v", report)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	helper := filepath.Join(dir, "reader-test")
	if err := os.WriteFile(helper, b, 0755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(helper, "-test.run=^TestUID1000ReceiptReaderHelper$", "-test.v")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "HOSTCOMPAT_READER_ROOT="+dir)
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: 1000, Gid: 1000, Groups: []uint32{}}}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("real UID-1000 receipt reader failed: %v\n%s", err, out)
	}
}

func TestUID1000ReceiptReaderHelper(t *testing.T) {
	dir := os.Getenv("HOSTCOMPAT_READER_ROOT")
	if dir == "" {
		t.Skip("subprocess helper")
	}
	if os.Geteuid() != 1000 {
		t.Fatal("reader did not run as UID 1000")
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	r := Inspect(root.FS(), "linux", "amd64")
	if r.Provenance != "verified" {
		t.Fatalf("non-root report: %+v", r)
	}
	_, err = root.Open(strings.TrimPrefix(VerificationCacheDirectory, "/"))
	if !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("private evidence accessible to UID 1000: %v", err)
	}
}

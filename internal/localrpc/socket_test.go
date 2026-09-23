package localrpc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListenPermissionsAndCleanup(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(dir, "runner.sock")
	listener, err := Listen(socket)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(socket)
	if err != nil || info.Mode().Perm() != 0600 || info.Mode()&os.ModeSocket == 0 {
		t.Fatalf("socket permissions: %v, %v", info, err)
	}
	if other, err := Listen(socket); err == nil {
		other.Close()
		t.Fatal("replaced a live socket")
	}
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(socket); !os.IsNotExist(err) {
		t.Fatalf("socket not cleaned up: %v", err)
	}
}

func TestListenRefusesUnsafeParentsAndExistingFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	public := filepath.Join(dir, "public")
	if err := os.Mkdir(public, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(public, 0755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(public, "child")
	if err := os.Mkdir(child, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(child, filepath.Join(dir, "escape")); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "existing")
	if err := os.WriteFile(file, []byte("untouched"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"relative.sock", filepath.Join(public, "runner.sock"), filepath.Join(link, "runner.sock"), filepath.Join(dir, "absent", "runner.sock"), dir + "/escape/../runner.sock", file} {
		if listener, err := Listen(path); err == nil {
			listener.Close()
			t.Fatalf("accepted unsafe path %s", path)
		}
	}
	if _, err := os.Lstat(filepath.Join(public, "runner.sock")); !os.IsNotExist(err) {
		t.Fatalf("socket was created outside checked directory: %v", err)
	}
	data, err := os.ReadFile(file)
	if err != nil || string(data) != "untouched" {
		t.Fatalf("existing file was changed: %q, %v", data, err)
	}
	info, err := os.Stat(public)
	if err != nil || info.Mode().Perm() != 0755 {
		t.Fatal("existing directory permissions were changed")
	}
}

//go:build linux

package hostcompat

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fixtureRoot struct{ *os.Root }

func (r fixtureRoot) FS() fs.FS { return racingFS{FS: r.Root.FS()} }
func (r fixtureRoot) Lstat(name string) (os.FileInfo, error) {
	i, e := r.Root.Lstat(name)
	if e != nil {
		return nil, e
	}
	return rootInfo{i}, nil
}

func TestRealAtomicEvidencePublicationPruneAndTamper(t *testing.T) {
	dir := t.TempDir()
	cache := strings.TrimPrefix(VerificationCacheDirectory, "/")
	if err := os.MkdirAll(filepath.Join(dir, cache), 0700); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	r := fixtureRoot{root}
	store := cacheStore(r)
	retained, err := store(context.Background(), []byte("authenticated archive"), ".deb")
	if err != nil {
		t.Fatal(err)
	}
	again, err := store(context.Background(), []byte("authenticated archive"), ".deb")
	if err != nil || again != retained {
		t.Fatal("content-addressed evidence did not reuse the durable blob")
	}
	unused, err := store(context.Background(), []byte("old unreferenced proof"), ".sig")
	if err != nil {
		t.Fatal(err)
	}
	if err := root.WriteFile(cache+"/administrator-note", []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := atomicRootFile(r, receiptPath, []byte("old receipt"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := atomicRootFile(r, receiptPath, []byte("new receipt"), 0644); err != nil {
		t.Fatal(err)
	}
	b, err := root.ReadFile(receiptPath)
	if err != nil || string(b) != "new receipt" {
		t.Fatal("atomic receipt publication failed")
	}
	pruneVerificationCache(r, receiptSet{Packages: []receipt{{Authorized: false, Retained: []digest{{Path: retained}}}}})
	if _, err := root.Stat(retained); err != nil {
		t.Fatal("unauthorized recovery proof was pruned")
	}
	if _, err := root.Stat(unused); !os.IsNotExist(err) {
		t.Fatal("unreferenced owned blob not pruned")
	}
	if b, err := root.ReadFile(cache + "/administrator-note"); err != nil || string(b) != "preserve" {
		t.Fatal("unrelated cache file changed")
	}
	if err := root.WriteFile(retained, []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := store(context.Background(), []byte("authenticated archive"), ".deb"); err == nil {
		t.Fatal("tampered content-addressed blob silently reused or overwritten")
	}
	if err := root.Remove(receiptPath); err != nil {
		t.Fatal(err)
	}
	if err := root.WriteFile("untouched", []byte("preserve target"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := root.Symlink("../../../../untouched", receiptPath); err != nil {
		t.Fatal(err)
	}
	if err := atomicRootFile(r, receiptPath, []byte("replacement"), 0644); err == nil {
		t.Fatal("receipt symlink was replaced/followed")
	}
	if b, err := root.ReadFile("untouched"); err != nil || string(b) != "preserve target" {
		t.Fatal("alias target changed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := store(ctx, []byte("canceled proof"), ".sig"); err == nil {
		t.Fatal("canceled proof publication succeeded")
	}
	entries, err := readDir(r.FS(), cache)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".publish-") {
			t.Fatal("publication left an uncommitted staging file")
		}
	}
}

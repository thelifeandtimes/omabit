package deskwork

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func fixture(t *testing.T, values map[string]string) Manifest {
	t.Helper()
	files := []File{}
	for n, v := range values {
		s := sha256.Sum256([]byte(v))
		files = append(files, File{n, hex.EncodeToString(s[:]), int64(len(v))})
	}
	m, e := NewManifest(files)
	if e != nil {
		t.Fatal(e)
	}
	return m
}
func TestPathsAndManifest(t *testing.T) {
	for _, name := range []string{"app/example.hoon", "sys.kelvin", "desk.docket-0", "web/assets/foo.min.js", "web/Example_1.svg"} {
		if !ValidPath(name) {
			t.Fatal(name)
		}
	}
	for _, name := range []string{"../app.hoon", "/app.hoon", ".urb/log/data.mdb", "web/.env", "app//a.hoon", "web/../a.hoon", "a\\b.hoon", "app/a.hoon/", "filename", "app/a.Hoon", "a\x00.hoon", strings.Repeat("a/", 17) + "file.txt"} {
		if ValidPath(name) {
			t.Fatal(name)
		}
	}
	m := fixture(t, map[string]string{"a.txt": "one", "b.txt": "two"})
	m.Files[0].Bytes++
	if m.Validate() == nil {
		t.Fatal("tampering accepted")
	}
	m = fixture(t, map[string]string{"a.txt": "one"})
	m.Files = append(m.Files, m.Files[0])
	m.Digest = treeDigest(m.Files)
	if m.Validate() == nil {
		t.Fatal("duplicate accepted")
	}
	m = fixture(t, map[string]string{"a.txt": "one"})
	m.Files[0].Bytes = MaxFileBytes + 1
	m.Digest = treeDigest(m.Files)
	if m.Validate() == nil {
		t.Fatal("size accepted")
	}
	f := fixture(t, map[string]string{"a.txt": "a"}).Files[0]
	g := f
	g.Path = "a.txt/nested.txt"
	if _, err := NewManifest([]File{f, g}); err == nil {
		t.Fatal("file/directory collision accepted")
	}
}
func TestSnapshotBoundsAndLinks(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "app"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "app/foo.hoon"), []byte("source\n"), 0600); err != nil {
		t.Fatal(err)
	}
	m, err := Snapshot(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m, fixture(t, map[string]string{"app/foo.hoon": "source\n"})) {
		t.Fatal(m)
	}
	for _, target := range []string{"app/foo.hoon", "/etc/passwd"} {
		link := filepath.Join(dir, "link.txt")
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
		if _, err := Snapshot(dir); err == nil {
			t.Fatal("source link accepted")
		}
		os.Remove(link)
	}
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := Snapshot(dir); err == nil {
		t.Fatal("hidden directory accepted")
	}
	os.Remove(filepath.Join(dir, ".git"))
	f, err := os.Create(filepath.Join(dir, "large.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if err = f.Truncate(MaxFileBytes + 1); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if _, err := Snapshot(dir); err == nil {
		t.Fatal("large file accepted")
	}
}
func TestTransferPreservesGeneratedAndUntracked(t *testing.T) {
	prior := fixture(t, map[string]string{"app/a.hoon": "old", "gone.txt": "old"})
	current := fixture(t, map[string]string{"app/a.hoon": "old", "gone.txt": "old", "sys.kelvin": "generated", "notes.txt": "human", "shared.txt": "same"})
	next := fixture(t, map[string]string{"app/a.hoon": "new", "added.txt": "new", "shared.txt": "same"})
	p, err := PlanTransfer(prior, current, next)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Conflicts) != 0 || len(p.Changes) != 3 || !reflect.DeepEqual(p.Preserved, []string{"shared.txt"}) {
		t.Fatal(p)
	}
	kinds := map[string]string{}
	for _, c := range p.Changes {
		kinds[c.Path] = c.Kind
	}
	if !reflect.DeepEqual(kinds, map[string]string{"app/a.hoon": "replace", "gone.txt": "delete", "added.txt": "add"}) {
		t.Fatal(kinds)
	}
	if !reflect.DeepEqual(p.Managed, fixture(t, map[string]string{"app/a.hoon": "new", "added.txt": "new"})) {
		t.Fatal("adopted untracked file", p.Managed)
	}
}
func TestTransferRefusesConcurrentEditsAndCollisions(t *testing.T) {
	prior := fixture(t, map[string]string{"a.txt": "old", "gone.txt": "old", "deleted.txt": "old"})
	current := fixture(t, map[string]string{"a.txt": "editor", "gone.txt": "editor", "untracked.txt": "human", "folder.txt/child.txt": "keep"})
	next := fixture(t, map[string]string{"a.txt": "new", "untracked.txt": "new", "folder.txt": "file", "deleted.txt": "new"})
	p, err := PlanTransfer(prior, current, next)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Changes) != 0 || !reflect.DeepEqual(p.Conflicts, []string{"a.txt", "deleted.txt", "folder.txt", "gone.txt", "untracked.txt"}) {
		t.Fatal(p)
	}
}
func TestComparisonDetectsDirectWritesDeletionsAndDivergence(t *testing.T) {
	base := fixture(t, map[string]string{"a.txt": "base", "b.txt": "base"})
	unix := fixture(t, map[string]string{"a.txt": "unix", "new.txt": "unix"})
	c, e := Compare(base, unix, base)
	if e != nil || c.State != "unix-changes" || !reflect.DeepEqual(c.UnixChanges, []string{"a.txt", "b.txt", "new.txt"}) {
		t.Fatal(c, e)
	}
	c, e = Compare(base, base, unix)
	if e != nil || c.State != "clay-changes" {
		t.Fatal(c, e)
	}
	clay := fixture(t, map[string]string{"a.txt": "clay", "b.txt": "base"})
	c, e = Compare(base, unix, clay)
	if e != nil || c.State != "diverged" || !reflect.DeepEqual(c.Conflicts, []string{"a.txt"}) {
		t.Fatal(c, e)
	}
	c, e = Compare(base, unix, unix)
	if e != nil || c.State != "synchronized" || len(c.Conflicts) != 0 {
		t.Fatal(c, e)
	}
	empty := fixture(t, map[string]string{})
	c, e = Compare(base, empty, base)
	if e != nil || c.State != "unix-changes" || len(c.UnixChanges) != 2 {
		t.Fatal(c, e)
	}
}

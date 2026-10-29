//go:build linux

package privatechain

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

func filesystemWorkspace(t *testing.T) (string, pierstore.DirectoryIdentity, Plan, Workspace, *WorkspaceFilesystem) {
	t.Helper()
	p, w := workspaceFixture(t)
	directory := filepath.Join(t.TempDir(), "private-projects")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	id, err := workspaceDirectoryID(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	w.Parent = id
	d, err := OpenWorkspaceFilesystem(context.Background(), directory, id, p, w)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	return directory, id, p, w, d
}

func workspaceFiles(t *testing.T, directory string) map[string]os.FileMode {
	t.Helper()
	out := map[string]os.FileMode{}
	err := filepath.Walk(directory, func(p string, i os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		rel, e := filepath.Rel(directory, p)
		if e != nil {
			return e
		}
		out[rel] = i.Mode()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestWorkspaceFilesystemCreatesOnlyEmptyPrivateParents(t *testing.T) {
	directory, _, p, w, d := filesystemWorkspace(t)
	got, err := ExecuteWorkspace(context.Background(), p, w, d, func(string, Workspace) error { return nil }, time.Now)
	if err != nil || got.Phase != "observed" || !got.Observation.Synced {
		t.Fatal(got, err)
	}
	want := map[string]os.FileMode{".": os.ModeDir | 0700, w.Binding.ProjectID: os.ModeDir | 0700, w.Binding.ProjectID + "/piers": os.ModeDir | 0700}
	for _, m := range w.Mounts {
		want[strings.TrimPrefix(m.Path, "private-projects/")] = os.ModeDir | 0700
	}
	if got := workspaceFiles(t, directory); !reflect.DeepEqual(got, want) {
		t.Fatal(got)
	}
	// The future ship-named pier children and all keys/runtime inputs are absent.
	for _, m := range w.Mounts {
		if _, err := os.Lstat(filepath.Join(directory, w.Binding.ProjectID, "piers", m.InstanceID, strings.TrimPrefix(m.Ship, "~"))); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("created a pier", err)
		}
	}
}

func TestWorkspaceFilesystemRefusesExistingProjectWithoutAdoption(t *testing.T) {
	for _, kind := range []string{"directory", "symlink", "file"} {
		t.Run(kind, func(t *testing.T) {
			directory, _, p, w, d := filesystemWorkspace(t)
			target := filepath.Join(directory, w.Binding.ProjectID)
			sentinel := []byte("synthetic-private-value")
			switch kind {
			case "directory":
				mustInput(t, os.Mkdir(target, 0700))
				mustInput(t, os.WriteFile(filepath.Join(target, "keep"), sentinel, 0600))
			case "file":
				mustInput(t, os.WriteFile(target, sentinel, 0600))
			case "symlink":
				outside := filepath.Join(t.TempDir(), "outside")
				mustInput(t, os.Mkdir(outside, 0700))
				mustInput(t, os.Symlink(outside, target))
			}
			before := workspaceFiles(t, directory)
			got, err := ExecuteWorkspace(context.Background(), p, w, d, func(string, Workspace) error { return nil }, time.Now)
			if err == nil || got.Phase != "root-dispatched" || got.Root != (pierstore.DirectoryIdentity{}) {
				t.Fatal(got, err)
			}
			if after := workspaceFiles(t, directory); !reflect.DeepEqual(before, after) {
				t.Fatal("changed preexisting object")
			}
		})
	}
}

func TestWorkspaceFilesystemParentReplacementAndAliasesRefuse(t *testing.T) {
	for _, kind := range []string{"replaced", "symlink", "shared", "wrong-identity", "closed", "wrong-project"} {
		t.Run(kind, func(t *testing.T) {
			directory, id, p, w, d := filesystemWorkspace(t)
			switch kind {
			case "replaced":
				mustInput(t, os.Rename(directory, directory+"-original"))
				mustInput(t, os.Mkdir(directory, 0700))
			case "symlink":
				mustInput(t, os.Rename(directory, directory+"-original"))
				mustInput(t, os.Symlink(directory+"-original", directory))
			case "shared":
				mustInput(t, os.Chmod(directory, 0755))
			case "wrong-identity":
				id.Inode++
				if other, err := OpenWorkspaceFilesystem(context.Background(), directory, id, p, w); err == nil {
					other.Close()
					t.Fatal("accepted wrong parent")
				}
				return
			case "closed":
				mustInput(t, d.Close())
			case "wrong-project":
				w.Binding.ProjectID = "private-project-other"
				var err error
				w, err = NewWorkspace(p, w.Binding, 1, w.OperationID, w.Parent)
				if err != nil {
					t.Fatal(err)
				}
			}
			got, err := ExecuteWorkspace(context.Background(), p, w, d, func(string, Workspace) error { return nil }, time.Now)
			if err == nil || got.Phase != "root-dispatched" {
				t.Fatal(got, err)
			}
			if _, err = os.Lstat(filepath.Join(directory, w.Binding.ProjectID)); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("unexpected project", err)
			}
		})
	}
}

func TestWorkspaceFilesystemSyncFailureRetainsOriginalRoot(t *testing.T) {
	directory, _, p, w, d := filesystemWorkspace(t)
	d.syncFile = func(*os.File) error { return errors.New("synthetic sync failure") }
	got, err := ExecuteWorkspace(context.Background(), p, w, d, func(string, Workspace) error { return nil }, time.Now)
	if err == nil || got.Phase != "root-created" || got.Root == (pierstore.DirectoryIdentity{}) {
		t.Fatal(got, err)
	}
	f, err := os.Open(filepath.Join(directory, w.Binding.ProjectID))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	id, err := workspaceDirectoryID(f)
	if err != nil || !id.SameObject(got.Root) {
		t.Fatal(id, err)
	}
	if _, err = ExecuteWorkspace(context.Background(), p, got, d, func(string, Workspace) error { t.Fatal("replayed root"); return nil }, time.Now); err == nil {
		t.Fatal("replayed root creation")
	}
}

func completedWorkspace(t *testing.T) (string, pierstore.DirectoryIdentity, Plan, Workspace) {
	t.Helper()
	directory, id, p, w, d := filesystemWorkspace(t)
	// Simulate an acknowledgement/journal loss after mkdir and sync completed.
	got, err := ExecuteWorkspace(context.Background(), p, w, d, func(_ string, next Workspace) error {
		if next.Phase == "layout-created" {
			return errors.New("lost commit")
		}
		return nil
	}, time.Now)
	if err == nil || got.Phase != "layout-dispatched" {
		t.Fatal(got, err)
	}
	mustInput(t, d.Close())
	return directory, id, p, got
}

func TestWorkspaceFilesystemReopenRecoversLostLayoutAcknowledgement(t *testing.T) {
	directory, id, p, w := completedWorkspace(t)
	before := workspaceFiles(t, directory)
	d, err := OpenWorkspaceFilesystem(context.Background(), directory, id, p, w)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if err = d.CreateLayout(context.Background(), w); err == nil {
		t.Fatal("reopened driver repeated mkdir")
	}
	got, err := ObserveWorkspace(context.Background(), p, w, d, func(expected string, next Workspace) error {
		if expected != "layout-dispatched" || next.Root != w.Root {
			t.Fatal(next)
		}
		return nil
	}, time.Now)
	if err != nil || got.Phase != "observed" || !got.Observation.Synced {
		t.Fatal(got, err)
	}
	if !reflect.DeepEqual(before, workspaceFiles(t, directory)) {
		t.Fatal("recovery changed layout")
	}
}

func TestWorkspaceFilesystemRecoveryPreservesPartialAndUnexpectedTrees(t *testing.T) {
	for _, kind := range []string{"missing", "extra-root", "extra-leaf", "leaf-symlink", "piers-symlink", "leaf-shared", "root-replaced", "sync-failed"} {
		t.Run(kind, func(t *testing.T) {
			directory, id, p, w := completedWorkspace(t)
			root := filepath.Join(directory, w.Binding.ProjectID)
			leaf := filepath.Join(directory, w.Binding.ProjectID, "piers", w.Mounts[0].InstanceID)
			switch kind {
			case "missing":
				mustInput(t, os.Remove(leaf))
			case "extra-root":
				mustInput(t, os.WriteFile(filepath.Join(root, "keep"), []byte("private sentinel"), 0600))
			case "extra-leaf":
				mustInput(t, os.WriteFile(filepath.Join(leaf, "keep"), []byte("private sentinel"), 0600))
			case "leaf-symlink":
				mustInput(t, os.Remove(leaf))
				mustInput(t, os.Symlink(t.TempDir(), leaf))
			case "piers-symlink":
				piers := filepath.Join(root, "piers")
				mustInput(t, os.Rename(piers, piers+"-original"))
				mustInput(t, os.Symlink(piers+"-original", piers))
			case "leaf-shared":
				mustInput(t, os.Chmod(leaf, 0755))
			case "root-replaced":
				mustInput(t, os.Rename(root, root+"-original"))
				mustInput(t, os.Mkdir(root, 0700))
			}
			before := workspaceFiles(t, directory)
			d, err := OpenWorkspaceFilesystem(context.Background(), directory, id, p, w)
			if err != nil {
				t.Fatal(err)
			}
			defer d.Close()
			if kind == "sync-failed" {
				d.syncFile = func(*os.File) error { return errors.New("sync failed") }
			}
			got, err := ObserveWorkspace(context.Background(), p, w, d, func(string, Workspace) error { t.Fatal("unexpected successful journal"); return nil }, time.Now)
			if err == nil || !reflect.DeepEqual(got, w) || strings.Contains(err.Error(), "sentinel") {
				t.Fatal(got, err)
			}
			if !reflect.DeepEqual(before, workspaceFiles(t, directory)) {
				t.Fatal("recovery changed rejected layout")
			}
		})
	}
}

func TestWorkspaceFilesystemCancellationAndRepeatedDispatch(t *testing.T) {
	_, _, p, w, d := filesystemWorkspace(t)
	var err error
	w, err = w.Advance(p, "planned", "root-dispatched", pierstore.DirectoryIdentity{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if id, err := d.CreateRoot(ctx, w); !errors.Is(err, context.Canceled) || id != (pierstore.DirectoryIdentity{}) {
		t.Fatal(id, err)
	}
	id, err := d.CreateRoot(context.Background(), w)
	if err != nil {
		t.Fatal(err)
	}
	if other, err := d.CreateRoot(context.Background(), w); err == nil || other != (pierstore.DirectoryIdentity{}) {
		t.Fatal("repeated root dispatch", other, err)
	}
	w.Phase = "layout-dispatched"
	w.Root = id
	if err = d.CreateLayout(context.Background(), w); err != nil {
		t.Fatal(err)
	}
	if err = d.CreateLayout(context.Background(), w); err == nil {
		t.Fatal("repeated layout dispatch")
	}
}

func TestWorkspaceFilesystemInterruptedPartialLayoutIsNotFilledOnRecovery(t *testing.T) {
	directory, id, p, w, d := filesystemWorkspace(t)
	calls := 0
	d.syncFile = func(f *os.File) error {
		calls++
		if calls == 3 {
			return errors.New("first leaf sync lost")
		}
		return f.Sync()
	}
	got, err := ExecuteWorkspace(context.Background(), p, w, d, func(string, Workspace) error { return nil }, time.Now)
	if err == nil || got.Phase != "layout-dispatched" {
		t.Fatal(got, err)
	}
	before := workspaceFiles(t, directory)
	if len(before) != 4 {
		t.Fatal("expected only the first member mount", before)
	}
	mustInput(t, d.Close())
	reopened, err := OpenWorkspaceFilesystem(context.Background(), directory, id, p, got)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if _, err = ObserveWorkspace(context.Background(), p, got, reopened, func(string, Workspace) error { t.Fatal("committed partial layout"); return nil }, time.Now); err == nil {
		t.Fatal("accepted missing member")
	}
	if !reflect.DeepEqual(before, workspaceFiles(t, directory)) {
		t.Fatal("filled or changed partial layout")
	}
}

func TestWorkspaceFilesystemChangedPiersParentStopsRemainingCreation(t *testing.T) {
	directory, _, p, w, d := filesystemWorkspace(t)
	piers := filepath.Join(directory, w.Binding.ProjectID, "piers")
	calls := 0
	d.syncFile = func(f *os.File) error {
		calls++
		if calls == 3 {
			if err := os.Rename(piers, piers+"-original"); err != nil {
				return err
			}
			if err := os.Mkdir(piers, 0700); err != nil {
				return err
			}
		}
		return f.Sync()
	}
	got, err := ExecuteWorkspace(context.Background(), p, w, d, func(string, Workspace) error { return nil }, time.Now)
	if err == nil || got.Phase != "layout-dispatched" {
		t.Fatal(got, err)
	}
	for _, parent := range []string{piers, piers + "-original"} {
		if _, err = os.Lstat(filepath.Join(parent, w.Mounts[1].InstanceID)); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("continued creation under replaced parent", err)
		}
	}
}

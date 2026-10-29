//go:build linux

package privatechain

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
)

func readyWorkspaceFixture(t *testing.T) (string, Plan, Workspace) {
	t.Helper()
	directory, _, p, w, d := filesystemWorkspace(t)
	ready, err := ExecuteWorkspace(context.Background(), p, w, d, func(string, Workspace) error { return nil }, time.Now)
	if err != nil || ready.Phase != "observed" {
		t.Fatal(ready, err)
	}
	if err = d.Close(); err != nil {
		t.Fatal(err)
	}
	return directory, p, ready
}

func TestWorkspaceVerifierReadsOnlyRetainedReadyObjects(t *testing.T) {
	directory, p, w := readyWorkspaceFixture(t)
	before := workspaceFiles(t, directory)
	v, err := OpenWorkspaceVerifier(context.Background(), directory, p, w)
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()
	if _, ok := any(v).(WorkspaceDriver); ok {
		t.Fatal("verifier exposes creation authority")
	}
	v.filesystem.syncFile = func(*os.File) error {
		t.Fatal("read-only verification synced storage")
		return errors.New("sync forbidden")
	}
	// Caller edits after opening must not rebind held verification authority.
	p.Members[0].InstanceID = "other"
	p.Foundation.Contracts[0].CodeSHA256 = "other"
	w.Mounts[0].Path = "other"
	w.Observation.Mounts[0].Inode++
	for range 2 {
		if err = v.VerifyEmpty(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(before, workspaceFiles(t, directory)) {
		t.Fatal("verification changed the tree")
	}
	if err = v.Close(); err != nil {
		t.Fatal(err)
	}
	if err = v.Close(); err != nil {
		t.Fatal("second close", err)
	}
	if err = v.VerifyEmpty(context.Background()); err == nil {
		t.Fatal("closed verifier retained access")
	}
}

func TestWorkspaceVerifierRejectsReplacementWithIdenticalLayout(t *testing.T) {
	for _, kind := range []string{"parent", "root", "piers", "member"} {
		t.Run(kind, func(t *testing.T) {
			directory, p, w := readyWorkspaceFixture(t)
			v, err := OpenWorkspaceVerifier(context.Background(), directory, p, w)
			if err != nil {
				t.Fatal(err)
			}
			defer v.Close()
			root := filepath.Join(directory, w.Binding.ProjectID)
			piers := filepath.Join(root, "piers")
			target := map[string]string{"parent": directory, "root": root, "piers": piers, "member": filepath.Join(piers, w.Mounts[0].InstanceID)}[kind]
			retained := filepath.Join(filepath.Dir(directory), "retained-"+kind)
			if err = os.Rename(target, retained); err != nil {
				t.Fatal(err)
			}
			if err = os.Mkdir(target, 0700); err != nil {
				t.Fatal(err)
			}
			if kind == "parent" {
				if err = os.Mkdir(root, 0700); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "parent" || kind == "root" {
				if err = os.Mkdir(piers, 0700); err != nil {
					t.Fatal(err)
				}
			}
			if kind != "member" {
				for _, m := range w.Mounts {
					if err = os.Mkdir(filepath.Join(piers, m.InstanceID), 0700); err != nil {
						t.Fatal(err)
					}
				}
			}
			before := workspaceFiles(t, directory)
			if err = v.VerifyEmpty(context.Background()); err == nil {
				t.Fatal("same-shaped replacement adopted")
			}
			if !reflect.DeepEqual(before, workspaceFiles(t, directory)) {
				t.Fatal("refusal repaired replacement")
			}
			if reopened, err := OpenWorkspaceVerifier(context.Background(), directory, p, w); err == nil {
				reopened.Close()
				t.Fatal("factory adopted replacement")
			}
			if _, err = os.Stat(retained); err != nil {
				t.Fatal("original object was lost", err)
			}
		})
	}
}

func TestWorkspaceVerifierRejectsUnexpectedEntriesAndPermissions(t *testing.T) {
	for _, kind := range []string{"root-file", "member-file", "member-directory", "member-link", "piers-link", "parent-mode", "root-mode", "member-mode"} {
		t.Run(kind, func(t *testing.T) {
			directory, p, w := readyWorkspaceFixture(t)
			v, err := OpenWorkspaceVerifier(context.Background(), directory, p, w)
			if err != nil {
				t.Fatal(err)
			}
			defer v.Close()
			root := filepath.Join(directory, w.Binding.ProjectID)
			piers := filepath.Join(root, "piers")
			leaf := filepath.Join(piers, w.Mounts[0].InstanceID)
			switch kind {
			case "root-file":
				err = os.WriteFile(filepath.Join(root, "unexpected"), []byte("synthetic"), 0600)
			case "member-file":
				err = os.WriteFile(filepath.Join(leaf, "unexpected"), []byte("synthetic"), 0600)
			case "member-directory":
				err = os.Mkdir(filepath.Join(leaf, "unexpected"), 0700)
			case "member-link", "piers-link":
				target := leaf
				if kind == "piers-link" {
					target = piers
				}
				retained := filepath.Join(filepath.Dir(directory), "retained")
				if err = os.Rename(target, retained); err == nil {
					err = os.Symlink(retained, target)
				}
			case "parent-mode":
				err = os.Chmod(directory, 0755)
			case "root-mode":
				err = os.Chmod(root, 0755)
			case "member-mode":
				err = os.Chmod(leaf, 0755)
			}
			if err != nil {
				t.Fatal(err)
			}
			before := workspaceFiles(t, directory)
			if err = v.VerifyEmpty(context.Background()); err == nil {
				t.Fatal("changed workspace admitted")
			}
			if !reflect.DeepEqual(before, workspaceFiles(t, directory)) {
				t.Fatal("verification changed rejected tree")
			}
		})
	}
}

func TestWorkspaceVerifierRequiresCompletedBoundSelection(t *testing.T) {
	for _, kind := range []string{"planned", "interrupted", "wrong-plan", "wrong-root", "wrong-member", "wrong-parent", "missing-parent", "alias", "relative", "cancel"} {
		t.Run(kind, func(t *testing.T) {
			directory, p, w := readyWorkspaceFixture(t)
			selected := directory
			ctx := context.Background()
			switch kind {
			case "planned":
				w.Phase = "planned"
			case "interrupted":
				w.Phase = "layout-dispatched"
				w.Observation = nil
			case "wrong-plan":
				p.Members[0].HTTPPort++
			case "wrong-root":
				w.Root.Inode++
			case "wrong-member":
				w.Observation.Mounts[0].Inode += 1000
			case "wrong-parent":
				w.Parent.Inode++
			case "missing-parent":
				selected = filepath.Join(filepath.Dir(directory), "absent", "private-projects")
			case "alias":
				alias := filepath.Join(filepath.Dir(directory), "alias")
				if err := os.Symlink(directory, alias); err != nil {
					t.Fatal(err)
				}
				selected = alias
			case "relative":
				selected = "private-projects"
			case "cancel":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			before := workspaceFiles(t, directory)
			if v, err := OpenWorkspaceVerifier(ctx, selected, p, w); err == nil {
				v.Close()
				t.Fatal("invalid ready workspace admitted")
			}
			if !reflect.DeepEqual(before, workspaceFiles(t, directory)) {
				t.Fatal("failed open changed the tree")
			}
			if kind == "missing-parent" {
				if _, err := os.Stat(selected); !os.IsNotExist(err) {
					t.Fatal("missing parent was created", err)
				}
			}
		})
	}
}

func TestWorkspaceVerifierCancellationAndConcurrentClose(t *testing.T) {
	directory, p, w := readyWorkspaceFixture(t)
	v, err := OpenWorkspaceVerifier(context.Background(), directory, p, w)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err = v.VerifyEmpty(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation lost", err)
	}
	var group sync.WaitGroup
	for range 8 {
		group.Go(func() { _ = v.VerifyEmpty(context.Background()) })
	}
	group.Go(func() { _ = v.Close() })
	group.Wait()
	if err = v.VerifyEmpty(context.Background()); err == nil {
		t.Fatal("closed verifier retained access")
	}
	if err = v.Close(); err != nil {
		t.Fatal(err)
	}
}

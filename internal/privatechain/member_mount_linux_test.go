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
)

func TestMemberMountReadOnlyBindingAndEmptyBoundary(t *testing.T) {
	directory, p, w := readyWorkspaceFixture(t)
	member, peer := w.Mounts[0].InstanceID, w.Mounts[1].InstanceID
	root := filepath.Join(directory, w.Binding.ProjectID, "piers")
	peerFile := filepath.Join(root, peer, "private-peer-payload")
	mustInput(t, os.WriteFile(peerFile, []byte("synthetic peer contents"), 0600))
	before := workspaceFiles(t, directory)
	v, err := OpenMemberMountVerifier(context.Background(), directory, p, w, member)
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()
	p.Members[0].InstanceID = "changed"
	w.Mounts[0].InstanceID = "changed"
	w.Observation.Mounts[0].Inode++
	for range 2 {
		if err = v.VerifyEmpty(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(before, workspaceFiles(t, directory)) {
		t.Fatal("reader changed storage")
	}
	partial := filepath.Join(root, member, "partial")
	mustInput(t, os.WriteFile(partial, []byte("retained partial boot"), 0600))
	before = workspaceFiles(t, directory)
	if v.VerifyEmpty(context.Background()) == nil {
		t.Fatal("partial boot accepted as empty")
	}
	if err = v.Verify(context.Background()); err != nil {
		t.Fatal("mount identity conflated with contents", err)
	}
	if !reflect.DeepEqual(before, workspaceFiles(t, directory)) {
		t.Fatal("refusal changed storage")
	}
	if _, ok := any(v).(WorkspaceDriver); ok {
		t.Fatal("reader exposes creation")
	}
	mustInput(t, v.Close())
	mustInput(t, v.Close())
	if v.Verify(context.Background()) == nil || v.VerifyEmpty(context.Background()) == nil {
		t.Fatal("closed reader retained authority")
	}
}

func TestMemberMountRefusesReplacementAndKeepsOriginal(t *testing.T) {
	for _, kind := range []string{"parent", "root", "piers", "member"} {
		t.Run(kind, func(t *testing.T) {
			directory, p, w := readyWorkspaceFixture(t)
			instance := w.Mounts[0].InstanceID
			v, err := OpenMemberMountVerifier(context.Background(), directory, p, w, instance)
			if err != nil {
				t.Fatal(err)
			}
			defer v.Close()
			root := filepath.Join(directory, w.Binding.ProjectID)
			piers := filepath.Join(root, "piers")
			target := map[string]string{"parent": directory, "root": root, "piers": piers, "member": filepath.Join(piers, instance)}[kind]
			retained := filepath.Join(filepath.Dir(directory), "retained-"+kind)
			mustInput(t, os.Rename(target, retained))
			mustInput(t, os.Mkdir(target, 0700))
			entries, err := os.ReadDir(retained)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				mustInput(t, os.Rename(filepath.Join(retained, entry.Name()), filepath.Join(target, entry.Name())))
			}
			before := workspaceFiles(t, directory)
			if v.Verify(context.Background()) == nil || v.VerifyEmpty(context.Background()) == nil {
				t.Fatal("replacement accepted")
			}
			if other, err := OpenMemberMountVerifier(context.Background(), directory, p, w, instance); err == nil {
				other.Close()
				t.Fatal("replacement adopted on reopen")
			}
			if !reflect.DeepEqual(before, workspaceFiles(t, directory)) {
				t.Fatal("replacement repaired")
			}
			if _, err = os.Stat(retained); err != nil {
				t.Fatal("retained original removed", err)
			}
		})
	}
}

func TestMemberMountRejectsUnselectedOrUnsafeLayout(t *testing.T) {
	for _, kind := range []string{"wrong-member", "wrong-phase", "wrong-root", "wrong-mount", "symlink-parent", "symlink-member", "public-mount", "extra-member", "missing-peer"} {
		t.Run(kind, func(t *testing.T) {
			directory, p, w := readyWorkspaceFixture(t)
			instance := w.Mounts[0].InstanceID
			piers := filepath.Join(directory, w.Binding.ProjectID, "piers")
			mount := filepath.Join(piers, instance)
			switch kind {
			case "wrong-member":
				instance = "other"
			case "wrong-phase":
				w.Phase = "planned"
			case "wrong-root":
				w.Root.Inode++
			case "wrong-mount":
				w.Observation.Mounts[0].Inode++
			case "symlink-parent":
				alias := filepath.Join(filepath.Dir(directory), "alias")
				mustInput(t, os.Mkdir(alias, 0700))
				link := filepath.Join(alias, "private-projects")
				mustInput(t, os.Symlink(directory, link))
				directory = link
			case "symlink-member":
				kept := filepath.Join(filepath.Dir(directory), "retained-member")
				mustInput(t, os.Rename(mount, kept))
				mustInput(t, os.Symlink(kept, mount))
			case "public-mount":
				mustInput(t, os.Chmod(mount, 0755))
			case "extra-member":
				mustInput(t, os.Mkdir(filepath.Join(piers, "unselected"), 0700))
			case "missing-peer":
				mustInput(t, os.Rename(filepath.Join(piers, w.Mounts[1].InstanceID), filepath.Join(filepath.Dir(directory), "retained-peer")))
			}
			if v, err := OpenMemberMountVerifier(context.Background(), directory, p, w, instance); err == nil {
				v.Close()
				t.Fatal("unsafe selection accepted")
			}
		})
	}
}

func TestMemberMountCancellationAndConcurrentClose(t *testing.T) {
	directory, p, w := readyWorkspaceFixture(t)
	instance := w.Mounts[0].InstanceID
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := OpenMemberMountVerifier(ctx, directory, p, w, instance); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	v, err := OpenMemberMountVerifier(context.Background(), directory, p, w, instance)
	if err != nil {
		t.Fatal(err)
	}
	if err = v.VerifyEmpty(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	for range 8 {
		group.Add(1)
		go func() {
			defer group.Done()
			_ = v.Verify(context.Background())
			_ = v.VerifyEmpty(context.Background())
			_ = v.Close()
		}()
	}
	group.Wait()
	if v.Verify(context.Background()) == nil {
		t.Fatal("closed reader accepted")
	}
}

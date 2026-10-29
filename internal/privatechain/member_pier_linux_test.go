//go:build linux

package privatechain

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"golang.org/x/sys/unix"
)

func memberPierFixture(t *testing.T) (string, Plan, Workspace, string, string, pierstore.Identity) {
	t.Helper()
	directory, p, w := readyWorkspaceFixture(t)
	m := w.Mounts[0]
	pier := filepath.Join(directory, w.Binding.ProjectID, "piers", m.InstanceID, strings.TrimPrefix(m.Ship, "~"))
	for _, sub := range []string{".urb/log", ".urb/chk"} {
		mustInput(t, os.MkdirAll(filepath.Join(pier, sub), 0755))
	}
	mustInput(t, os.WriteFile(filepath.Join(pier, ".urb/log/synthetic-event"), []byte("inert payload"), 0600))
	mustInput(t, os.WriteFile(filepath.Join(pier, ".urb/chk/synthetic-checkpoint"), []byte("inert checkpoint"), 0600))
	id, err := pierstore.Identify(pier)
	mustInput(t, err)
	return directory, p, w, m.InstanceID, pier, id
}
func TestMemberPierHeldReadOnlyState(t *testing.T) {
	d, p, w, instance, pier, id := memberPierFixture(t)
	peer := filepath.Join(d, w.Binding.ProjectID, "piers", w.Mounts[1].InstanceID, "unread-peer")
	mustInput(t, unix.Mkfifo(peer, 0600))
	before := workspaceFiles(t, d)
	v, err := OpenMemberPierVerifier(context.Background(), d, p, w, instance, id)
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()
	p.Members[0].Ship = "~bud"
	w.Observation.Mounts[0].Inode++
	id.Inode++
	for range 2 {
		mustInput(t, v.Verify(context.Background()))
		mustInput(t, v.VerifyStoppedState(context.Background()))
	}
	if !reflect.DeepEqual(before, workspaceFiles(t, d)) {
		t.Fatal("metadata verifier changed tree")
	}
	raw, err := os.ReadFile(filepath.Join(pier, ".urb/log/synthetic-event"))
	mustInput(t, err)
	if string(raw) != "inert payload" {
		t.Fatal("payload changed")
	}
	// Metadata validation never opens regular state files for reading.
	mustInput(t, os.Chmod(filepath.Join(pier, ".urb/log/synthetic-event"), 0000))
	mustInput(t, v.VerifyStoppedState(context.Background()))
	mustInput(t, v.Close())
	mustInput(t, v.Close())
	if v.Verify(context.Background()) == nil || v.VerifyStoppedState(context.Background()) == nil {
		t.Fatal("closed verifier accepted")
	}
}
func TestMemberPierLockPresenceNeverMeansRepair(t *testing.T) {
	for _, kind := range []string{"root", "urb-vere", "urb-lock", "symlink", "fifo"} {
		t.Run(kind, func(t *testing.T) {
			d, p, w, instance, pier, id := memberPierFixture(t)
			v, err := OpenMemberPierVerifier(context.Background(), d, p, w, instance, id)
			mustInput(t, err)
			defer v.Close()
			lock := filepath.Join(pier, ".vere.lock")
			if kind == "urb-vere" {
				lock = filepath.Join(pier, ".urb/vere.lock")
			}
			if kind == "urb-lock" {
				lock = filepath.Join(pier, ".urb/lock")
			}
			switch kind {
			case "symlink":
				mustInput(t, os.Symlink("absent-target", lock))
			case "fifo":
				mustInput(t, unix.Mkfifo(lock, 0600))
			default:
				mustInput(t, os.WriteFile(lock, []byte("synthetic lock"), 0600))
			}
			mustInput(t, v.Verify(context.Background()))
			if v.VerifyStoppedState(context.Background()) == nil {
				t.Fatal("lock accepted as stopped")
			}
			if _, err = os.Lstat(lock); err != nil {
				t.Fatal("lock removed")
			}
		})
	}
}
func TestMemberPierRefusesUnsafeStateWithoutPayloadReads(t *testing.T) {
	for _, kind := range []string{"missing-log", "missing-chk", "symlink-log", "symlink-file", "hardlink-file", "fifo", "group-write-file", "world-write-dir", "special-mode", "extra-pier", "missing-urb", "linked-urb"} {
		t.Run(kind, func(t *testing.T) {
			d, p, w, instance, pier, id := memberPierFixture(t)
			v, err := OpenMemberPierVerifier(context.Background(), d, p, w, instance, id)
			mustInput(t, err)
			defer v.Close()
			state := filepath.Join(pier, ".urb")
			payload := filepath.Join(state, "log/synthetic-event")
			kept := filepath.Join(t.TempDir(), "retained")
			switch kind {
			case "missing-log":
				mustInput(t, os.Rename(filepath.Join(state, "log"), kept))
			case "missing-chk":
				mustInput(t, os.Rename(filepath.Join(state, "chk"), kept))
			case "symlink-log":
				mustInput(t, os.Rename(filepath.Join(state, "log"), kept))
				mustInput(t, os.Symlink(kept, filepath.Join(state, "log")))
			case "symlink-file":
				mustInput(t, os.Rename(payload, kept))
				mustInput(t, os.Symlink(kept, payload))
			case "hardlink-file":
				mustInput(t, os.Link(payload, kept))
			case "fifo":
				mustInput(t, unix.Mkfifo(filepath.Join(state, "log/special"), 0600))
			case "group-write-file":
				mustInput(t, os.Chmod(payload, 0660))
			case "world-write-dir":
				mustInput(t, os.Chmod(filepath.Join(state, "chk"), 0777))
			case "special-mode":
				mustInput(t, os.Chmod(filepath.Join(state, "chk"), 0755|os.ModeSetgid))
			case "extra-pier":
				mustInput(t, os.Mkdir(filepath.Join(filepath.Dir(pier), "other"), 0700))
			case "missing-urb":
				mustInput(t, os.Rename(state, kept))
			case "linked-urb":
				mustInput(t, os.Rename(state, kept))
				mustInput(t, os.Symlink(kept, state))
			}
			before := workspaceFiles(t, d)
			err = v.VerifyStoppedState(context.Background())
			if err == nil {
				t.Fatal("unsafe state accepted")
			}
			if strings.Contains(err.Error(), pier) {
				t.Fatal("path exposed")
			}
			if !reflect.DeepEqual(before, workspaceFiles(t, d)) {
				t.Fatal("refusal changed tree")
			}
		})
	}
}
func TestMemberPierRefusesReplacedObjects(t *testing.T) {
	for _, kind := range []string{"mount", "pier", "urb"} {
		t.Run(kind, func(t *testing.T) {
			d, p, w, instance, pier, id := memberPierFixture(t)
			v, err := OpenMemberPierVerifier(context.Background(), d, p, w, instance, id)
			mustInput(t, err)
			defer v.Close()
			target := pier
			if kind == "mount" {
				target = filepath.Dir(pier)
			}
			if kind == "urb" {
				target = filepath.Join(pier, ".urb")
			}
			kept := filepath.Join(t.TempDir(), "retained")
			mustInput(t, os.Rename(target, kept))
			mustInput(t, os.Mkdir(target, 0700))
			entries, err := os.ReadDir(kept)
			mustInput(t, err)
			for _, entry := range entries {
				mustInput(t, os.Rename(filepath.Join(kept, entry.Name()), filepath.Join(target, entry.Name())))
			}
			if v.Verify(context.Background()) == nil || v.VerifyStoppedState(context.Background()) == nil {
				t.Fatal("replacement accepted")
			}
			if other, e := OpenMemberPierVerifier(context.Background(), d, p, w, instance, id); e == nil {
				other.Close()
				t.Fatal("replacement adopted")
			}
			if _, err = os.Stat(kept); err != nil {
				t.Fatal("original removed")
			}
		})
	}
}
func TestMemberPierRejectsUnboundIdentity(t *testing.T) {
	for _, kind := range []string{"zero", "pier", "urb", "birth", "same-object", "mount-object", "other-member"} {
		t.Run(kind, func(t *testing.T) {
			d, p, w, instance, _, id := memberPierFixture(t)
			switch kind {
			case "zero":
				id = pierstore.Identity{}
			case "pier":
				id.Inode++
			case "urb":
				id.UrbInode++
			case "birth":
				id.Birth = "invalid"
			case "same-object":
				id.UrbDevice = id.Device
				id.UrbInode = id.Inode
				id.UrbBirth = id.Birth
			case "mount-object":
				id.Device = w.Root.Device
				id.Inode = w.Root.Inode
				id.Birth = w.Root.Birth
			case "other-member":
				instance = w.Mounts[1].InstanceID
			}
			if v, err := OpenMemberPierVerifier(context.Background(), d, p, w, instance, id); err == nil {
				v.Close()
				t.Fatal("unbound identity accepted")
			}
		})
	}
}
func TestMemberPierBoundedMetadataWalk(t *testing.T) {
	t.Run("entries", func(t *testing.T) {
		d, p, w, instance, pier, id := memberPierFixture(t)
		v, err := OpenMemberPierVerifier(context.Background(), d, p, w, instance, id)
		mustInput(t, err)
		defer v.Close()
		log := filepath.Join(pier, ".urb/log")
		// Two root directories and two existing files leave8188 accepted entries.
		for i := 0; i < 8188; i++ {
			mustInput(t, os.WriteFile(filepath.Join(log, fmt.Sprintf("entry-%04d", i)), nil, 0600))
		}
		mustInput(t, v.VerifyStoppedState(context.Background()))
		mustInput(t, os.WriteFile(filepath.Join(log, "over-limit"), nil, 0600))
		if v.VerifyStoppedState(context.Background()) == nil {
			t.Fatal("entry bound ignored")
		}
	})
	t.Run("depth", func(t *testing.T) {
		d, p, w, instance, pier, id := memberPierFixture(t)
		v, err := OpenMemberPierVerifier(context.Background(), d, p, w, instance, id)
		mustInput(t, err)
		defer v.Close()
		deepest := filepath.Join(pier, ".urb/log")
		for range 128 {
			deepest = filepath.Join(deepest, "d")
			mustInput(t, os.Mkdir(deepest, 0700))
		}
		mustInput(t, v.VerifyStoppedState(context.Background()))
		mustInput(t, os.Mkdir(filepath.Join(deepest, "d"), 0700))
		if v.VerifyStoppedState(context.Background()) == nil {
			t.Fatal("depth bound ignored")
		}
	})
}
func TestMemberPierCancellationAndConcurrentClose(t *testing.T) {
	d, p, w, instance, _, id := memberPierFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := OpenMemberPierVerifier(ctx, d, p, w, instance, id); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	v, err := OpenMemberPierVerifier(context.Background(), d, p, w, instance, id)
	mustInput(t, err)
	if err = v.VerifyStoppedState(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	for range 8 {
		group.Add(1)
		go func() {
			defer group.Done()
			_ = v.Verify(context.Background())
			_ = v.VerifyStoppedState(context.Background())
			_ = v.Close()
		}()
	}
	group.Wait()
	if v.VerifyStoppedState(context.Background()) == nil {
		t.Fatal("closed verifier accepted")
	}
}

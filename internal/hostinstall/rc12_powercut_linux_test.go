//go:build linux && amd64 && powercut

package hostinstall

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/piercopy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"golang.org/x/sys/unix"
)

// This entry point is compiled only into the private VM test binary. It invokes
// the original production filesystem primitives, with no service/RPC/runtime.
func TestRC12PowercutGuest(t *testing.T) {
	phase := os.Getenv("OMABIT_POWERCUT_PHASE")
	if phase == "" {
		t.Skip("run through scripts/qualify-powercut.py")
	}
	marker, err := os.ReadFile("/powercut-guest")
	if err != nil || string(marker) != "isolated-rc12-filesystem-guest-v1\n" || os.Getppid() != 1 || os.Geteuid() != 0 {
		t.Fatal("private VM guard failed")
	}
	var source, destination syscall.Stat_t
	pcMust(t, syscall.Stat("/source", &source))
	pcMust(t, syscall.Stat("/destination", &destination))
	if source.Dev == destination.Dev {
		t.Fatal("fixture requires two independent filesystems")
	}
	switch os.Getenv("OMABIT_POWERCUT_SUITE") {
	case "storage":
		pcStorage(t, phase)
	case "update":
		pcUpdate(t, phase)
	default:
		t.Fatal("unknown suite")
	}
}

func pcMust(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func pcEvent(value map[string]any) {
	b, _ := json.Marshal(value)
	fmt.Printf("POWERCUT %s\n", b)
}

func pcPayload(name string, size int) []byte {
	seed := sha256.Sum256([]byte("isolated-powercut-fixture-v1/" + name))
	b := make([]byte, size)
	for i := range b {
		b[i] = seed[i%len(seed)] ^ byte(i>>13) ^ byte(i>>5)
	}
	return b
}

func pcFiles() map[string][]byte {
	files := map[string][]byte{
		".urb/chk/image.bin":  pcPayload("image", 4<<20),
		".urb/log/events.bin": pcPayload("events", 2<<20),
		".urb/log/empty":      {},
		"notes.txt":           pcPayload("notes", 271),
	}
	for n := 0; n < 8; n++ {
		name := fmt.Sprintf(".urb/log/nested/%02d.bin", n)
		files[name] = pcPayload(name, 128<<10)
	}
	return files
}

// Independent oracle: verify exact paths, bytes and private modes, not only the
// production Measure fingerprint. Extra/missing files or directories fail.
func pcComplete(root string) (bool, string) {
	want := map[string]string{".": "directory"}
	for name, data := range pcFiles() {
		want[name] = fmt.Sprintf("%x", sha256.Sum256(data))
		for dir := filepath.Dir(name); dir != "."; dir = filepath.Dir(dir) {
			want[dir] = "directory"
		}
	}
	got := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() && info.Mode().Perm() == 0700 {
			got[rel] = "directory"
		} else if info.Mode().IsRegular() && info.Mode().Perm() == 0600 {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			got[rel] = fmt.Sprintf("%x", sha256.Sum256(data))
		} else {
			return fmt.Errorf("unexpected type/mode: %s %v", rel, info.Mode())
		}
		return nil
	})
	if err != nil {
		return false, err.Error()
	}
	if !reflect.DeepEqual(want, got) {
		return false, "layout/content mismatch"
	}
	return true, "exact layout, SHA-256 contents and modes match"
}

func pcStorage(t *testing.T, phase string) {
	ctx := context.Background()
	const original, held, home = "/source/original/pier", "/source/held/pier", "/destination/home"
	location := pierstore.Location{Path: "fakeships/fixture/pier"}
	dest := filepath.Join(home, "piers", location.Path)
	switch phase {
	case "seed":
		for name, data := range pcFiles() {
			p := filepath.Join(original, name)
			pcMust(t, os.MkdirAll(filepath.Dir(p), 0700))
			pcMust(t, os.WriteFile(p, data, 0600))
		}
		pcMust(t, os.Mkdir(home, 0700))
		ok, reason := pcComplete(original)
		if !ok {
			t.Fatal(reason)
		}
		pcEvent(map[string]any{"event": "seed", "fixture": "12 files / 7340303 bytes; synthetic, no ship identity"})
	case "crash":
		pcMust(t, os.Mkdir(filepath.Dir(held), 0700))
		pcMust(t, pierstore.RenameDirectory(original, held))
		_, err := pierstore.PrepareMount(home, &location)
		pcMust(t, err)
		pcMust(t, pierstore.SyncDestinationParents(home, location, nil))
		if err := pierstore.RenameDirectory(held, dest); err != unix.EXDEV {
			t.Fatalf("real cross-device rename must fail EXDEV: %v", err)
		}
		pcMust(t, os.Mkdir(dest, 0700))
		source, err := os.OpenRoot(held)
		pcMust(t, err)
		defer source.Close()
		target, err := os.OpenRoot(dest)
		pcMust(t, err)
		defer target.Close()
		removal, err := pierstore.ReviewRemoval(ctx, held)
		pcMust(t, err)
		// This marker lives only in initramfs RAM and performs no disk barrier.
		pcMust(t, os.WriteFile("/powercut-armed", []byte("storage"), 0600))
		copied, err := piercopy.Copy(ctx, source, target, func(int, int64) error { return nil })
		pcMust(t, err)
		pcMust(t, pierstore.SyncDestination(home, location, nil))
		pcMust(t, removal.Check(ctx, held))
		for _, r := range []*os.Root{source, target} {
			actual, err := piercopy.Measure(ctx, r)
			pcMust(t, err)
			if actual != copied {
				t.Fatal("production copy verification mismatch")
			}
		}
		_, err = pierstore.CheckMount(home, location)
		pcMust(t, err)
		pcMust(t, removal.Remove(ctx, held, func() error {
			if pierstore.Busy(held) {
				return fmt.Errorf("unexpected synthetic lock")
			}
			return nil
		}, nil))
		t.Fatal("requested boundary not intercepted")
	case "verify":
		originalOK, originalReason := pcComplete(original)
		heldOK, heldReason := pcComplete(held)
		destOK, destReason := pcComplete(dest)
		cut := os.Getenv("OMABIT_POWERCUT_CASE")
		requireDestination := strings.HasPrefix(cut, "source-") || cut == "destination-after-fsync"
		pcEvent(map[string]any{"event": "storage-readback", "original_complete": originalOK,
			"held_complete": heldOK, "destination_complete": destOK, "destination_required": requireDestination,
			"original_reason": originalReason, "held_reason": heldReason, "destination_reason": destReason})
		if !originalOK && !heldOK && !destOK || requireDestination && !destOK {
			t.Fatal("power loss violated complete source OR durable destination invariant")
		}
		if !strings.HasPrefix(cut, "source-") && !heldOK {
			t.Fatal("held source must remain complete before source removal")
		}
		if cut == "source-removed" {
			_, err := os.Lstat(held)
			if !os.IsNotExist(err) {
				t.Fatal("source removal did not survive its completed parent fsync")
			}
			pcEvent(map[string]any{"event": "source-removal-durable", "absent": true})
		}
	default:
		t.Fatal("unknown phase")
	}
}

func pcUpdate(t *testing.T, phase string) {
	ctx := context.Background()
	i := newInstaller()
	i.root = "/destination"
	old, target := pcPayload("old-executable", 2<<20), pcPayload("target-executable", 2<<20)
	previous := executableGeneration{SHA256: digestBytes(old), MinSchema: 11, Schema: 20}
	next := executableGeneration{SHA256: digestBytes(target), MinSchema: 11, Schema: 20}
	j := &activationJournal{Version: 1, Owner: "fixture", UID: 1000, RunnerID: "runner-" + strings.Repeat("1", 32),
		Token: strings.Repeat("2", 64), Stage: "quiesced", Previous: &previous, Target: next}
	switch phase {
	case "seed":
		pcMust(t, i.retainGeneration(ctx, old, previous))
		pcMust(t, i.retainGeneration(ctx, target, next))
		pcMust(t, i.installFile(ctx, access.ShellPath, old, 0755, false))
		pcMust(t, i.saveActivation(ctx, j, "quiesced"))
	case "crash":
		pcMust(t, os.WriteFile("/powercut-armed", []byte("update"), 0600))
		pcMust(t, i.saveActivation(ctx, j, "stopping"))
		data, err := i.retainedBytes(ctx, next)
		pcMust(t, err)
		pcMust(t, i.installFile(ctx, access.ShellPath, data, 0755, true))
		pcMust(t, i.saveActivation(ctx, j, "starting"))
		t.Fatal("requested boundary not intercepted")
	case "verify":
		for _, g := range []executableGeneration{previous, next} {
			_, err := i.retainedBytes(ctx, g)
			pcMust(t, err)
		}
		data, err := i.readInstalled(ctx, activationJournalPath, 16384)
		pcMust(t, err)
		var actual activationJournal
		pcMust(t, json.Unmarshal(data, &actual))
		stage := actual.Stage
		actual.Stage = "quiesced"
		if !reflect.DeepEqual(actual, *j) {
			t.Fatal("journal identity/contract corruption")
		}
		binary, err := i.readInstalled(ctx, access.ShellPath, 128<<20)
		pcMust(t, err)
		oldOK, newOK := bytes.Equal(binary, old), bytes.Equal(binary, target)
		if !oldOK && !newOK || stage != "quiesced" && stage != "stopping" && stage != "starting" || stage == "quiesced" && !oldOK || stage == "starting" && !newOK {
			t.Fatalf("invalid journal/executable pair: %s old=%t target=%t", stage, oldOK, newOK)
		}
		cut := os.Getenv("OMABIT_POWERCUT_CASE")
		if strings.HasPrefix(cut, "journal-") && !oldOK || strings.HasPrefix(cut, "executable-") && stage != "stopping" {
			t.Fatal("publication ordering violated: stopping journal must precede executable replacement")
		}
		if cut == "journal-after-dir-fsync" && stage != "stopping" || cut == "executable-after-dir-fsync" && !newOK || cut == "starting-after-dir-fsync" && (stage != "starting" || !newOK) {
			t.Fatal("successful directory fsync failed to persist publication")
		}
		for path, mode := range map[string]os.FileMode{activationJournalPath: 0600, access.ShellPath: 0755, generationPath(previous): 0755, generationPath(next): 0755} {
			info, err := os.Stat(i.path(path))
			pcMust(t, err)
			if info.Mode().Perm() != mode {
				t.Fatalf("publication mode changed: %s %v", path, info.Mode())
			}
		}
		pcEvent(map[string]any{"event": "update-readback", "stage": stage, "old_complete": oldOK,
			"target_complete": newOK, "retained_generations_verified": true, "journal_sha256": digestBytes(data)})
		// Replay only the real filesystem writers; this is not service recovery.
		for n := 0; n < 2; n++ {
			pcMust(t, i.installFile(ctx, access.ShellPath, target, 0755, true))
			pcMust(t, i.saveActivation(ctx, j, "starting"))
		}
		replayed, err := i.readInstalled(ctx, access.ShellPath, 128<<20)
		pcMust(t, err)
		if !bytes.Equal(replayed, target) {
			t.Fatal("filesystem replay failed")
		}
		replayedJournal, err := i.readInstalled(ctx, activationJournalPath, 16384)
		pcMust(t, err)
		pcMust(t, json.Unmarshal(replayedJournal, &actual))
		if !reflect.DeepEqual(actual, *j) || actual.Stage != "starting" {
			t.Fatal("journal filesystem replay failed")
		}
		pcEvent(map[string]any{"event": "update-filesystem-replay", "repetitions": 2, "passed": true})
	default:
		t.Fatal("unknown phase")
	}
}

//go:build linux

package deskwork

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
)

func stageFixture(t *testing.T) (string, StageIntent, []byte) {
	t.Helper()
	state := t.TempDir()
	if err := os.Chmod(state, 0700); err != nil {
		t.Fatal(err)
	}
	root, err := PrepareStageCache(state)
	if err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	if err = os.WriteFile(filepath.Join(source, "hello.txt"), []byte("hello\n"), 0600); err != nil {
		t.Fatal(err)
	}
	m, err := Snapshot(source)
	if err != nil {
		t.Fatal(err)
	}
	intent := StageIntent{Version: 1, ID: strings.Repeat("a", 64), Target: apppolicy.Target{InstanceID: "alpha", Desk: "example"}, SourceDigest: m.Digest, Files: len(m.Files), Bytes: 6}
	var bundle bytes.Buffer
	if err = WriteBundle(context.Background(), source, m, &bundle); err != nil {
		t.Fatal(err)
	}
	return root, intent, bundle.Bytes()
}
func TestSourceStageCacheVerifiesFrozenContentAndReplay(t *testing.T) {
	root, intent, bundle := stageFixture(t)
	s, err := CacheStage(context.Background(), root, intent, "operator", bytes.NewReader(bundle))
	if err != nil {
		t.Fatal(err)
	}
	got, path, err := ReadStage(context.Background(), root, intent.ID, "operator")
	if err != nil || !reflect.DeepEqual(got, s) || path != filepath.Join(root, intent.ID, "source") {
		t.Fatal(got, path, err)
	}
	// Reconciliation does not read a new upload or create another directory.
	again, err := CacheStage(context.Background(), root, intent, "operator", bytes.NewReader(nil))
	if err != nil || !reflect.DeepEqual(s, again) {
		t.Fatal(again, err)
	}
	if _, _, err = ReadStage(context.Background(), root, intent.ID, "other-operator"); err == nil {
		t.Fatal("stage crossed operator")
	}
	other := intent
	other.Target.Desk = "other"
	if _, err = CacheStage(context.Background(), root, other, "operator", bytes.NewReader(bundle)); err == nil {
		t.Fatal("request ID rebound")
	}
	if err = os.WriteFile(filepath.Join(path, "hello.txt"), []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err = ReadStage(context.Background(), root, intent.ID, "operator"); err == nil {
		t.Fatal("changed stage accepted")
	}
	if _, err = CacheStage(context.Background(), root, intent, "operator", bytes.NewReader(bundle)); err == nil {
		t.Fatal("corrupt stage silently overwritten")
	}
}
func TestSourceStageFailedUploadLeavesNoUsableOrPartialTree(t *testing.T) {
	for _, kind := range []string{"truncated", "trailing", "hash", "intent-size", "cancelled"} {
		t.Run(kind, func(t *testing.T) {
			root, intent, bundle := stageFixture(t)
			ctx := context.Background()
			switch kind {
			case "truncated":
				bundle = bundle[:len(bundle)-1]
			case "trailing":
				bundle = append(bundle, 1)
			case "hash":
				bundle[len(bundle)-1] ^= 1
			case "intent-size":
				intent.Bytes++
			case "cancelled":
				c, cancel := context.WithCancel(ctx)
				cancel()
				ctx = c
			}
			if _, err := CacheStage(ctx, root, intent, "operator", bytes.NewReader(bundle)); err == nil {
				t.Fatal("bad upload accepted")
			}
			entries, err := os.ReadDir(root)
			if err != nil || len(entries) != 0 {
				t.Fatal("failed upload retained a tree", entries, err)
			}
		})
	}
}
func TestSourceStageRejectsLinkedAndMalformedCacheMetadata(t *testing.T) {
	for _, kind := range []string{"linked-root", "linked-entry", "linked-source", "linked-receipt", "malformed", "missing-source", "missing-receipt"} {
		t.Run(kind, func(t *testing.T) {
			root, intent, bundle := stageFixture(t)
			if _, err := CacheStage(context.Background(), root, intent, "operator", bytes.NewReader(bundle)); err != nil {
				t.Fatal(err)
			}
			entry := filepath.Join(root, intent.ID)
			source := filepath.Join(entry, "source")
			receipt := filepath.Join(entry, "receipt.json")
			switch kind {
			case "linked-root", "linked-entry", "linked-source", "linked-receipt":
				path := map[string]string{"linked-root": root, "linked-entry": entry, "linked-source": source, "linked-receipt": receipt}[kind]
				if err := os.Rename(path, path+"-original"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(path+"-original", path); err != nil {
					t.Fatal(err)
				}
			case "malformed":
				if err := os.WriteFile(receipt, []byte(`{"allow_commit":true}`), 0600); err != nil {
					t.Fatal(err)
				}
			case "missing-source":
				if err := os.Rename(source, source+"-retained"); err != nil {
					t.Fatal(err)
				}
			case "missing-receipt":
				if err := os.Remove(receipt); err != nil {
					t.Fatal(err)
				}
			}
			if _, _, err := ReadStage(context.Background(), root, intent.ID, "operator"); err == nil {
				t.Fatal("unsafe stage accepted")
			}
			if _, err := CacheStage(context.Background(), root, intent, "operator", bytes.NewReader(bundle)); err == nil {
				t.Fatal("unsafe stage repaired implicitly")
			}
		})
	}
}
func TestSourceStageCapacityAndUnknownEntriesRefused(t *testing.T) {
	root, intent, bundle := stageFixture(t)
	if err := os.Mkdir(filepath.Join(root, ".pending-unknown"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := CacheStage(context.Background(), root, intent, "operator", bytes.NewReader(bundle)); err == nil {
		t.Fatal("unknown cache entry ignored")
	}
	if err := os.Remove(filepath.Join(root, ".pending-unknown")); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < MaxCachedStages; i++ {
		id := strings.Repeat(string(rune('a'+i%6)), 63) + string("0123456789abcdef"[i])
		intent.ID = id
		if _, err := CacheStage(context.Background(), root, intent, "operator", bytes.NewReader(bundle)); err != nil {
			t.Fatal(i, err)
		}
	}
	intent.ID = strings.Repeat("f", 64)
	if _, err := CacheStage(context.Background(), root, intent, "operator", bytes.NewReader(bundle)); err == nil {
		t.Fatal("cache exceeded count bound")
	}
	// Metadata byte accounting is checked without allocating a 1GiB fixture.
	root2, intent2, bundle2 := stageFixture(t)
	_, err := CacheStage(context.Background(), root2, intent2, "operator", bytes.NewReader(bundle2))
	if err != nil {
		t.Fatal(err)
	}
	if err = stageCapacity(root2, MaxCachedSourceBytes); err == nil {
		t.Fatal("byte bound ignored")
	}
}

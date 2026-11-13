//go:build linux

package deskwork

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourceStageDiscardRetainsOtherSourceAndSupportsRecovery(t *testing.T) {
	for _, phase := range []string{"complete", "partial-upload", "partial-discard", "before-intent", "after-intent-removal", "during-source-removal", "changed-source"} {
		t.Run(phase, func(t *testing.T) {
			root, intent, bundle := stageFixture(t)
			_, err := CacheStage(context.Background(), root, intent, "operator", bytes.NewReader(bundle))
			if err != nil {
				t.Fatal(err)
			}
			selected := filepath.Join(root, intent.ID)
			switch phase {
			case "partial-upload":
				target := filepath.Join(root, ".pending-"+intent.ID)
				if err = os.Rename(selected, target); err != nil {
					t.Fatal(err)
				}
				if err = os.Remove(filepath.Join(target, "receipt.json")); err != nil {
					t.Fatal(err)
				}
			case "partial-discard":
				if err = os.Rename(selected, filepath.Join(root, ".discard-"+intent.ID)); err != nil {
					t.Fatal(err)
				}
			case "during-source-removal":
				target := filepath.Join(root, ".discard-"+intent.ID)
				if err = os.Rename(selected, target); err != nil {
					t.Fatal(err)
				}
				if err = os.Remove(filepath.Join(target, "source", "hello.txt")); err != nil {
					t.Fatal(err)
				}
			case "before-intent", "after-intent-removal":
				if err = os.RemoveAll(selected); err != nil {
					t.Fatal(err)
				}
				prefix := ".pending-"
				if phase == "after-intent-removal" {
					prefix = ".discard-"
				}
				if err = os.Mkdir(filepath.Join(root, prefix+intent.ID), 0700); err != nil {
					t.Fatal(err)
				}
			case "changed-source":
				if err = os.WriteFile(filepath.Join(selected, "source", "hello.txt"), []byte("modified"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			sentinel := filepath.Join(root, "unrelated")
			if err = os.WriteFile(sentinel, []byte("preserve"), 0600); err != nil {
				t.Fatal(err)
			}
			called := false
			guard := func(r StageReceipt) error {
				called = true
				if r.Intent != intent {
					t.Fatal("wrong guard binding")
				}
				return nil
			}
			if err = DiscardStage(root, intent.ID, "operator", intent.SourceDigest, guard); err != nil {
				t.Fatal(err)
			}
			if !called && phase != "before-intent" && phase != "after-intent-removal" {
				t.Fatal("discard skipped guard")
			}
			entries, err := os.ReadDir(root)
			if err != nil || len(entries) != 1 || entries[0].Name() != "unrelated" {
				t.Fatal(entries, err)
			}
			if err = DiscardStage(root, intent.ID, "operator", intent.SourceDigest, guard); err != nil {
				t.Fatal("absent stage not idempotent", err)
			}
			if err = os.Remove(sentinel); err != nil {
				t.Fatal(err)
			}
			if _, err = CacheStage(context.Background(), root, intent, "operator", bytes.NewReader(bundle)); err != nil {
				t.Fatal("cache slot not reusable", err)
			}
		})
	}
}
func TestSourceStageDiscardRefusesWrongBindingAndUnknownContent(t *testing.T) {
	for _, scenario := range []string{"actor", "digest", "guard", "missing-intent", "malformed-intent", "unknown-partial", "linked-partial", "identity", "conflicting-intent"} {
		t.Run(scenario, func(t *testing.T) {
			root, intent, bundle := stageFixture(t)
			_, err := CacheStage(context.Background(), root, intent, "operator", bytes.NewReader(bundle))
			if err != nil {
				t.Fatal(err)
			}
			actor, digest := "operator", intent.SourceDigest
			selected := filepath.Join(root, intent.ID)
			guard := func(StageReceipt) error { return nil }
			switch scenario {
			case "actor":
				actor = "another"
			case "digest":
				digest = strings.Repeat("f", 64)
			case "guard":
				guard = func(StageReceipt) error { return errors.New("unresolved effect") }
			case "missing-intent":
				err = os.Remove(filepath.Join(selected, "intent.json"))
			case "malformed-intent":
				err = os.WriteFile(filepath.Join(selected, "intent.json"), []byte("{}"), 0600)
			case "unknown-partial":
				next := filepath.Join(root, ".pending-"+intent.ID)
				err = os.Rename(selected, next)
				if err == nil {
					err = os.Remove(filepath.Join(next, "intent.json"))
				}
			case "linked-partial":
				next := selected + "-retained"
				err = os.Rename(selected, next)
				if err == nil {
					err = os.Symlink(next, filepath.Join(root, ".pending-"+intent.ID))
				}
			case "conflicting-intent":
				r, e := pendingStage(selected)
				if e != nil {
					t.Fatal(e)
				}
				r.Intent.Target.Desk = "another"
				data, _ := json.Marshal(r)
				err = os.WriteFile(filepath.Join(selected, "intent.json"), data, 0600)
			case "identity":
				next := filepath.Join(root, strings.Repeat("e", 64))
				err = os.Rename(selected, next)
				intent.ID = strings.Repeat("e", 64)
			}
			if err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadDir(root)
			if err != nil {
				t.Fatal(err)
			}
			if err = DiscardStage(root, intent.ID, actor, digest, guard); err == nil {
				t.Fatal("unsafe discard accepted")
			}
			after, err := os.ReadDir(root)
			if err != nil || len(after) != len(before) {
				t.Fatal("refused discard changed cache", err)
			}
		})
	}
}

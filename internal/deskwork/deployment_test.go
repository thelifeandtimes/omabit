package deskwork

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func deployManifest(t *testing.T, files map[string]string, canonical bool) Manifest {
	t.Helper()
	out := []File{}
	for name, body := range files {
		sum := sha256.Sum256([]byte(body))
		size := int64(len(body))
		if canonical {
			size = 0
		}
		out = append(out, File{Path: name, SHA256: hex.EncodeToString(sum[:]), Bytes: size})
	}
	m, err := NewManifest(out)
	if err != nil {
		t.Fatal(err)
	}
	return m
}
func TestDeploymentInitialSourcePreservesGeneratedAndSharedFiles(t *testing.T) {
	current := deployManifest(t, map[string]string{"desk.docket": "generated", "shared.hoon": "canonical shared"}, true)
	source := deployManifest(t, map[string]string{"app/demo.hoon": "app source", "shared.hoon": "shared bytes", "data.custom": "new mark"}, false)
	known := deployManifest(t, map[string]string{"shared.hoon": "canonical shared"}, true)
	plan, err := PlanDeployment(EmptyOwnership(), current, source, known)
	if err != nil || plan.Validate() != nil || len(plan.Conflicts) != 0 || len(plan.Changes) != 2 || !reflect.DeepEqual(plan.Preserved, []string{"desk.docket", "shared.hoon"}) || !reflect.DeepEqual(plan.Managed, []string{"app/demo.hoon", "data.custom"}) {
		t.Fatal(plan, err)
	}
	normalized := deployManifest(t, map[string]string{"app/demo.hoon": "canonical app", "shared.hoon": "canonical shared", "data.custom": "canonical new mark"}, true)
	result := deployManifest(t, map[string]string{"desk.docket": "generated", "shared.hoon": "canonical shared", "app/demo.hoon": "canonical app", "data.custom": "canonical new mark"}, true)
	owned, err := VerifyDeployment(plan, result, normalized)
	if err != nil || len(owned.Raw.Files) != 2 || owned.Validate() != nil {
		t.Fatal(owned, err)
	}
	if _, ok := owned.Raw.Index()["shared.hoon"]; ok {
		t.Fatal("identical shared file adopted")
	}
	for _, scenario := range []string{"generated-edit", "extra-file", "wrong-source", "missing-path", "incomplete-normalization"} {
		t.Run(scenario, func(t *testing.T) {
			files := map[string]string{"desk.docket": "generated", "shared.hoon": "canonical shared", "app/demo.hoon": "canonical app", "data.custom": "canonical new mark"}
			norm := normalized
			switch scenario {
			case "generated-edit":
				files["desk.docket"] = "modified"
			case "extra-file":
				files["unexpected.txt"] = "new"
			case "wrong-source":
				files["app/demo.hoon"] = "wrong"
			case "missing-path":
				delete(files, "data.custom")
			case "incomplete-normalization":
				norm = known
			}
			if _, err := VerifyDeployment(plan, deployManifest(t, files, true), norm); err == nil {
				t.Fatal("unverified result accepted")
			}
		})
	}
}
func TestDeploymentUpdatesOwnedFilesAndReportsConflicts(t *testing.T) {
	raw := deployManifest(t, map[string]string{"app/demo.hoon": "old source", "old.txt": "old", "same.txt": "same"}, false)
	prior := deployManifest(t, map[string]string{"app/demo.hoon": "old canonical", "old.txt": "old canonical", "same.txt": "same canonical"}, true)
	ownership := Ownership{1, raw, prior}
	now := deployManifest(t, map[string]string{"app/demo.hoon": "old canonical", "old.txt": "old canonical", "same.txt": "same canonical", "generated.txt": "keep"}, true)
	next := deployManifest(t, map[string]string{"app/demo.hoon": "new source", "new.txt": "new", "same.txt": "same"}, false)
	empty := EmptyOwnership().Clay
	p, err := PlanDeployment(ownership, now, next, empty)
	if err != nil || p.Validate() != nil || len(p.Conflicts) != 0 || len(p.Changes) != 3 {
		t.Fatal(p, err)
	}
	kinds := map[string]string{}
	for _, c := range p.Changes {
		kinds[c.Path] = c.Kind
	}
	if !reflect.DeepEqual(kinds, map[string]string{"app/demo.hoon": "replace", "new.txt": "add", "old.txt": "delete"}) {
		t.Fatal(kinds)
	}
	if !reflect.DeepEqual(p.Managed, []string{"app/demo.hoon", "new.txt", "same.txt"}) {
		t.Fatal(p.Managed)
	}
	for _, scenario := range []string{"external-edit", "external-delete", "unowned-collision", "directory-collision"} {
		t.Run(scenario, func(t *testing.T) {
			current := map[string]string{"app/demo.hoon": "old canonical", "old.txt": "old canonical", "same.txt": "same canonical", "generated.txt": "keep"}
			source := map[string]string{"app/demo.hoon": "new source", "new.txt": "new", "same.txt": "same"}
			want := "app/demo.hoon"
			switch scenario {
			case "external-edit":
				current[want] = "other"
			case "external-delete":
				delete(current, want)
			case "unowned-collision":
				source["generated.txt"] = "replace"
				want = "generated.txt"
			case "directory-collision":
				source["generated.txt/child.txt"] = "new"
				want = "generated.txt/child.txt"
			}
			got, e := PlanDeployment(ownership, deployManifest(t, current, true), deployManifest(t, source, false), empty)
			if e != nil || !reflect.DeepEqual(got.Conflicts, []string{want}) {
				t.Fatal(got.Conflicts, e)
			}
			if _, e = VerifyDeployment(got, now, empty); e == nil {
				t.Fatal("conflicted plan verified")
			}
		})
	}
}
func TestDeploymentCanonicalNoopAndPlanIntegrity(t *testing.T) {
	raw := deployManifest(t, map[string]string{"hello.txt": "hello\n"}, false)
	clay := deployManifest(t, map[string]string{"hello.txt": "canonical hello"}, true)
	source := deployManifest(t, map[string]string{"hello.txt": "hello"}, false)
	p, err := PlanDeployment(Ownership{1, raw, clay}, clay, source, clay)
	if err != nil || len(p.Changes) != 0 || len(p.Conflicts) != 0 {
		t.Fatal(p, err)
	}
	owned, err := VerifyDeployment(p, clay, clay)
	if err != nil || owned.Raw.Digest != source.Digest {
		t.Fatal(owned, err)
	}
	for _, kind := range []string{"digest", "changes", "managed", "preserved", "conflicts"} {
		t.Run(kind, func(t *testing.T) {
			data, _ := json.Marshal(p)
			var q DeploymentPlan
			json.Unmarshal(data, &q)
			switch kind {
			case "digest":
				q.Digest = strings.Repeat("f", 64)
			case "changes":
				q.Changes = append(q.Changes, Change{Path: "other.txt", Kind: "delete"})
			case "managed":
				q.Managed = []string{}
			case "preserved":
				q.Preserved = []string{"other.txt"}
			case "conflicts":
				q.Conflicts = []string{"hello.txt"}
			}
			if q.Validate() == nil {
				t.Fatal("tampered plan accepted")
			}
		})
	}
	bad := deployManifest(t, map[string]string{"outside.txt": "not selected"}, true)
	if _, err = PlanDeployment(EmptyOwnership(), clay, source, bad); err == nil {
		t.Fatal("normalizer selected another file")
	}
}
func TestDeploymentReadsExactBytesAndRefusesChangedStage(t *testing.T) {
	for _, kind := range []string{"ok", "changed", "link", "cancelled"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "hello.txt")
			if err := os.WriteFile(path, []byte("hello"), 0600); err != nil {
				t.Fatal(err)
			}
			source, err := Snapshot(dir)
			if err != nil {
				t.Fatal(err)
			}
			p, err := PlanDeployment(EmptyOwnership(), EmptyOwnership().Clay, source, EmptyOwnership().Clay)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch kind {
			case "changed":
				os.WriteFile(path, []byte("edited"), 0600)
			case "link":
				os.Remove(path)
				os.Symlink("/unavailable", path)
			case "cancelled":
				cancel()
			}
			delta, err := DeploymentFiles(ctx, dir, p)
			if kind == "ok" {
				if err != nil || len(delta) != 1 || delta[0].Path != "hello.txt" || string(delta[0].Bytes) != "hello" {
					t.Fatal(delta, err)
				}
			} else if err == nil {
				t.Fatal("changed stage accepted")
			}
		})
	}
}
func TestDeploymentRejectsOversizedAtomicDelta(t *testing.T) {
	large, _ := NewManifest([]File{{Path: "one.txt", SHA256: strings.Repeat("a", 64), Bytes: MaxFileBytes}, {Path: "two.txt", SHA256: strings.Repeat("b", 64), Bytes: 1}})
	if _, err := PlanDeployment(EmptyOwnership(), EmptyOwnership().Clay, large, EmptyOwnership().Clay); err == nil {
		t.Fatal("oversized delta accepted")
	}
}

func TestDeploymentRenormalizesUnchangedBytesWhenConverterChanged(t *testing.T) {
	raw := deployManifest(t, map[string]string{"hello.txt": "same bytes"}, false)
	oldClay := deployManifest(t, map[string]string{"hello.txt": "old noun"}, true)
	known := deployManifest(t, map[string]string{"hello.txt": "new noun"}, true)
	p, err := PlanDeployment(Ownership{1, raw, oldClay}, oldClay, raw, known)
	if err != nil || len(p.Changes) != 1 || p.Changes[0].Kind != "replace" {
		t.Fatal("changed mark conversion mistaken for no-op", p, err)
	}
}

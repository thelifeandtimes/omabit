package grubbery_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

func plan(t *testing.T) (grubbery.Plan, []deskwork.SourceFile) {
	t.Helper()
	m, files := source(t, `{"state":"/state"}`, deskwork.SourceFile{Path: "assets/ui/index.html", Bytes: []byte("<!doctype html><p>exact client</p>\n")}, deskwork.SourceFile{Path: "assets/source.hoon", Bytes: []byte{0, 255}})
	p, err := grubbery.NewPlan(binding(), strings.Repeat("d", 64), strings.Repeat("1", 64), m, files, grubbery.Before{SnapshotDigest: strings.Repeat("e", 64), SourceState: "absent", DeskState: "absent"})
	if err != nil {
		t.Fatal(err)
	}
	return p, files
}

func TestAppPlanStagesImmutableCodeBeforeVersionAndActivation(t *testing.T) {
	p, files := plan(t)
	if p.Validate(files) != nil {
		t.Fatal("original plan invalid")
	}
	n, _ := p.Binding.Namespace()
	release, _ := p.Binding.ReleaseRoot(p.RequestID)
	seen := map[string]bool{}
	codeAt, versionAt, pokeAt := -1, -1, -1
	for i, step := range p.Steps {
		if seen[step.ID] || len(step.ID) != 64 {
			t.Fatal("ambiguous native phase ID")
		}
		seen[step.ID] = true
		request, err := p.Request(step.ID, files)
		if err != nil {
			t.Fatal(err)
		}
		query, err := url.ParseQuery(request.Query)
		if err != nil {
			t.Fatal(err)
		}
		if query.Has("force") || strings.Contains(request.Path, "/sand/") || strings.Contains(request.Path, "/cull/") || strings.Contains(request.Path, "/permits/") || request.Method == "DELETE" {
			t.Fatal("deployment authored permission or destructive effect", request)
		}
		if step.SourcePath != "" {
			var original []byte
			for _, f := range files {
				if f.Path == step.SourcePath {
					original = f.Bytes
					break
				}
			}
			sum := sha256.Sum256(request.Body)
			if !bytes.Equal(request.Body, original) || step.PayloadSHA256 != hex.EncodeToString(sum[:]) || request.Path != "/grubbery/api/make"+release+"/"+step.SourcePath {
				t.Fatal("source left its exact immutable release", step)
			}
			if strings.HasPrefix(step.SourcePath, "nex/") {
				if query.Get("blot") != "/hoon" || request.ContentType != "text/x-hoon" {
					t.Fatal("source Hoon mark omitted")
				}
			}
			if step.SourcePath == "assets/source.hoon" && query.Has("blot") {
				t.Fatal("binary asset became executable Hoon source")
			}
			if step.SourcePath == "assets/ui/index.html" && request.ContentType != "text/html" {
				t.Fatal("served HTML would have an incorrect MIME")
			}
			if len(request.Body) > 0 {
				request.Body[0] ^= 255
				if bytes.Equal(request.Body, original) {
					t.Fatal("request body aliases cached source")
				}
			}
		}
		if step.Kind == "observe-code" {
			codeAt = i
			if request.Method != "GET" || request.Path != "/grubbery/api/build"+release || query.Get("all") != "1" {
				t.Fatal("compile observation escaped staging namespace")
			}
		}
		if step.Path == release+"/version.json" {
			versionAt = i
		}
		if step.Kind == "poke-source" {
			pokeAt = i
			if request.Method != "POST" || query.Get("blot") != "/json" || request.Path != "/grubbery/api/poke"+n.DeskRoot+"/source.json" {
				t.Fatal("source fiber not poked as native JSON")
			}
			var body struct {
				Code string `json:"code"`
			}
			if json.Unmarshal(request.Body, &body) != nil || body.Code != release {
				t.Fatal("source pointer differs from immutable release")
			}
		}
	}
	if codeAt < 0 || versionAt <= codeAt || pokeAt <= versionAt || p.Steps[len(p.Steps)-1].Kind != "observe-followed-source" {
		t.Fatal("source published before compilation/version proof")
	}
	if _, err := p.Request(strings.Repeat("0", 64), files); err == nil {
		t.Fatal("unknown native phase accepted")
	}
}

func TestAppPlanRefusesMutationAndDoesNotClaimAnExistingUnownedRoot(t *testing.T) {
	p, files := plan(t)
	for _, change := range []string{"path", "kind", "source-path", "control", "step-id", "binding", "epoch", "order", "digest"} {
		t.Run(change, func(t *testing.T) {
			raw, _ := json.Marshal(p)
			var changed grubbery.Plan
			json.Unmarshal(raw, &changed)
			switch change {
			case "path":
				changed.Steps[0].Path = "/apps/shell"
			case "kind":
				changed.Steps[0].Kind = "sand"
			case "source-path":
				changed.Steps[0].SourcePath = "../../secret.txt"
			case "control":
				changed.Steps[0].ControlJSON = `{"neck":"/shell"}`
			case "step-id":
				changed.Steps[0].ID = strings.Repeat("f", 64)
			case "binding":
				changed.Binding.Actor = "other"
			case "epoch":
				changed.Epoch++
			case "order":
				changed.Steps[0], changed.Steps[1] = changed.Steps[1], changed.Steps[0]
			case "digest":
				changed.Digest = strings.Repeat("a", 64)
			}
			if changed.Validate(files) == nil {
				t.Fatal("changed plan accepted")
			}
			if _, err := changed.Request(changed.Steps[0].ID, files); err == nil {
				t.Fatal("changed plan produced a request")
			}
		})
	}
	for _, state := range []string{"unowned", "conflict", "unknown"} {
		before := p.Before
		before.SourceState = state
		if got, err := grubbery.NewPlan(p.Binding, p.StageID, p.RequestID, p.Source.Manifest, files, before); err == nil || got.Steps != nil {
			t.Fatal("unowned namespace returned effects")
		}
	}
	files[0].Bytes = []byte(`{"other":"/state"}`)
	if _, err := p.Request(p.Steps[0].ID, files); err == nil {
		t.Fatal("changed cache produced native request")
	}
}

func TestOwnedAppUpdateLeavesSavedStateAndPriorCodeUntouched(t *testing.T) {
	p, files := plan(t)
	n, _ := p.Binding.Namespace()
	prior, _ := p.Binding.ReleaseRoot(p.RequestID)
	ownerHash := ""
	for _, s := range p.Steps {
		if s.Path == n.SourceRoot+"/omabit-owner.json" {
			ownerHash = s.PayloadSHA256
		}
	}
	before := grubbery.Before{SnapshotDigest: strings.Repeat("f", 64), SourceState: "owned", SourceOwnerSHA256: ownerHash, DeskState: "owned", DeskOwnerSHA256: ownerHash, Epoch: 1, ReleaseRoot: prior}
	update, err := grubbery.NewPlan(p.Binding, p.StageID, strings.Repeat("2", 64), p.Source.Manifest, files, before)
	if err != nil {
		t.Fatal(err)
	}
	if update.Epoch != 2 {
		t.Fatal("update version did not advance")
	}
	for _, step := range update.Steps {
		if step.Kind == "make-directory" && step.Path == n.DeskRoot || step.Kind == "make-file" && strings.HasPrefix(step.Path, n.DataRoot+"/") || strings.HasPrefix(step.Path, prior+"/") {
			t.Fatal("update modified saved state or prior immutable code", step)
		}
	}
	before.SourceOwnerSHA256 = strings.Repeat("0", 64)
	if _, err := grubbery.NewPlan(p.Binding, p.StageID, strings.Repeat("2", 64), p.Source.Manifest, files, before); err == nil {
		t.Fatal("changed ownership marker accepted")
	}
	before.SourceOwnerSHA256 = ownerHash
	before.ReleaseRoot = "/other/" + strings.Repeat("1", 64) + "/code"
	if _, err := grubbery.NewPlan(p.Binding, p.StageID, strings.Repeat("2", 64), p.Source.Manifest, files, before); err == nil {
		t.Fatal("foreign current release accepted")
	}
}

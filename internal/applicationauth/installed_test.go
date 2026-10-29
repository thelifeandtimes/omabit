package applicationauth

import (
	"context"
	"encoding/json"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func installedFixture(t *testing.T) (string, string, map[string]string) {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "fixture.auth-lab")
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{"manifest.json": `{"id":"fixture.auth-lab","name":"Fixture (Auth Lab)"}`, "application-profile.json": `{"version":1,"profile":"auth-lab"}`, "receiver.py": "raise Exception('discovery executed helper')"}
	d, _ := json.Marshal(Declaration{1, "python3", OwnerAuthority, "receiver.py", []string{"manifest.json", DeclarationName, "receiver.py"}})
	files[DeclarationName] = string(d)
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(path, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root, path, map[string]string{"HOME": t.TempDir()}
}

func TestInstalledDeclarationBindsProfileInventoryAndRejectsChangedApproval(t *testing.T) {
	root, path, env := installedFixture(t)
	list, err := DiscoverInstalled(root, env)
	if err != nil || len(list) != 1 || list[0].Status != "available" || list[0].Target.Profile != "auth-lab" {
		t.Fatal(list, err)
	}
	a, _, err := ResolveInstalled(root, "fixture.auth-lab", env)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(a.names, ","), "application-profile.json") {
		t.Fatal("profile omitted from digest")
	}
	if err := os.WriteFile(filepath.Join(path, "receiver.py"), []byte("print('updated')"), 0600); err != nil {
		t.Fatal(err)
	}
	changed, _, err := ResolveInstalled(root, "fixture.auth-lab", env)
	if err != nil || changed.Target() == a.Target() || !errors.Is(a.Verify(context.Background()), ErrArtifact) {
		t.Fatal("changed source retained approval", err)
	}
	if _, _, err := ResolveInstalled(root, "../fixture.auth-lab", env); err == nil {
		t.Fatal("path traversal accepted")
	}
}

func TestInstalledInvalidDeclarationsRemainUnavailable(t *testing.T) {
	for _, kind := range []string{"duplicate", "unknown", "runtime", "authority", "omitted-self", "link", "foreign-manifest", "traversal", "missing-helper"} {
		t.Run(kind, func(t *testing.T) {
			root, path, env := installedFixture(t)
			p := filepath.Join(path, DeclarationName)
			data, _ := os.ReadFile(p)
			var d Declaration
			if err := json.Unmarshal(data, &d); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "duplicate":
				data = []byte(strings.Replace(string(data), `"version":1`, `"version":0,"version":1`, 1))
			case "unknown":
				data = append([]byte(`{"extra":true,`), data[1:]...)
			case "runtime":
				d.Runtime = "shell"
			case "authority":
				d.Authority = "desk-restricted"
			case "omitted-self":
				d.Files = d.Files[:1]
			case "foreign-manifest":
				if err := os.WriteFile(filepath.Join(path, "manifest.json"), []byte(`{"id":"another"}`), 0600); err != nil {
					t.Fatal(err)
				}
			case "traversal":
				d.Entry = "../receiver.py"
				d.Files = append(d.Files, d.Entry)
			case "missing-helper":
				d.Files = append(d.Files, "absent.py")
			case "link":
				if err := os.Rename(p, p+".real"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(p+".real", p); err != nil {
					t.Fatal(err)
				}
			}
			if kind != "duplicate" && kind != "unknown" {
				data, _ = json.Marshal(d)
			}
			if kind != "link" {
				if err := os.WriteFile(p, data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			list, err := DiscoverInstalled(root, env)
			if err != nil || len(list) != 1 || list[0].Status != "invalid_declaration" {
				t.Fatal(list, err)
			}
			if _, _, err := ResolveInstalled(root, "fixture.auth-lab", env); !errors.Is(err, ErrArtifact) {
				t.Fatal(err)
			}
		})
	}
}

func TestExpectedRevisionCheckedInsideApplicationTransaction(t *testing.T) {
	f := setup(t)
	target := collection.ApplicationTarget{PluginID: "fixture.auth-lab", Profile: "auth-lab", SHA256: strings.Repeat("a", 64)}
	// Keep setup independent of receiver behavior: an approved attempt can be
	// closed without issuing a token, but a stale UI revision cannot close it.
	a, err := f.r.ApproveApplication(context.Background(), f.ship, strings.Repeat("d", 32), target)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.e.AtRevision(a.Revision+1).Disconnect(context.Background(), a.ID); !errors.Is(err, collection.ErrApplicationConflict) {
		t.Fatal("stale disconnect", err)
	}
	current, err := f.e.AtRevision(a.Revision).Recover(context.Background(), a.ID)
	if err != nil || current.Phase != "closed" || current.Outcome != "not-issued" {
		t.Fatal(current, err)
	}
}

func TestExpiredParentClosesWithoutIssuingOrBlockingNewApproval(t *testing.T) {
	f := setup(t)
	receiver := newReceiver()
	approved := f.approve(t, receiver)
	f.mu.Lock()
	clear(f.sessions)
	f.mu.Unlock()
	result, err := f.e.Connect(context.Background(), approved.ID, receiver)
	if !errors.Is(err, ErrNotIssued) || result.Phase != "closed" || result.Outcome != "not-issued" || result.ReceiverReconciliation || receiver.calls != 0 {
		t.Fatal(result, err)
	}
	f.mu.Lock()
	if f.logins != 1 {
		t.Error("expired parent attempted another login", f.logins)
	}
	f.mu.Unlock()
	if _, err := f.r.ApproveApplication(context.Background(), f.ship, collection.NewSessionRef(), receiver.target); err != nil {
		t.Fatal("known non-issuance blocked a new explicit approval", err)
	}
}

//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

func TestProjectMemberMountConflicts(t *testing.T) {
	for _, mode := range []string{"empty", "unrelated", "same", "ancestor", "alias", "ancestor-alias", "missing-source", "relative-source", "volume", "unknown-type", "own-created", "own-reordered", "own-running", "own-missing", "own-mount-changed", "inventory-changed", "ids-changed", "duplicate-id", "malformed", "missing-mounts", "extra-row", "source-replaced", "target-replaced", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			f, _, _ := managedFixture(t)
			resource, _, anchor := resourceContractFixture(t)
			f.engine.ID = resource.Preparation.EngineID
			originalContract := projectMemberContractFixture(t)
			if err := pierstore.Init(f.d.home); err != nil {
				t.Fatal(err)
			}
			loc := pierstore.Location{Path: originalContract.selection.Storage.Path}
			path, err := pierstore.PrepareMount(f.d.home, &loc)
			if err != nil {
				t.Fatal(err)
			}
			selection := originalContract.selection
			selection.Storage = loc
			c, err := newProjectMemberContract(selection, f.d.home, path, candidate.ManifestDigest)
			if err != nil {
				t.Fatal(err)
			}
			member := projectMemberContainerFixture(t, c)
			dir := filepath.Join(f.d.home, ".local/share/urbit-runner/managed/project-resources-"+c.selection.Project.ProjectID)
			if err = os.MkdirAll(filepath.Join(dir, "docker-config"), 0700); err != nil {
				t.Fatal(err)
			}
			source := filepath.Join(t.TempDir(), "unrelated")
			if err = os.Mkdir(source, 0700); err != nil {
				t.Fatal(err)
			}
			mountType := "bind"
			switch mode {
			case "same":
				source = path
			case "ancestor":
				source = filepath.Dir(path)
			case "alias", "ancestor-alias":
				target := path
				if mode == "ancestor-alias" {
					target = filepath.Dir(path)
				}
				source = filepath.Join(t.TempDir(), "alias")
				if err = os.Symlink(target, source); err != nil {
					t.Fatal(err)
				}
			case "missing-source":
				source += "-missing"
			case "relative-source":
				source = "relative/source"
			case "volume":
				mountType = "volume"
			case "unknown-type":
				mountType = "other"
			}
			records := []projectContainerMounts{{ID: anchor.ID, Mounts: anchor.Mounts}}
			if mode != "empty" && !strings.HasPrefix(mode, "own-") {
				records = append(records, projectContainerMounts{ID: strings.Repeat("d", 64), Mounts: []mount{{Type: mountType, Source: source, Destination: "/data", RW: true}}})
			}
			ownID := ""
			if strings.HasPrefix(mode, "own-") {
				ownID = member.ID
				if mode != "own-missing" {
					records = append(records, projectContainerMounts{ID: ownID, Mounts: slices.Clone(member.Mounts)})
				}
				if mode == "own-reordered" {
					slices.Reverse(records[1].Mounts)
				}
				if mode == "own-running" {
					*member.State.Running = true
					member.State.Status = "running"
					member.State.Pid = 100
					member.State.StartedAt = "2026-10-23T12:00:00Z"
				}
				if mode == "own-mount-changed" {
					records[1].Mounts[0].Source = source
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			lists, reads := 0, 0
			original := f.d.execute
			f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
				if len(args) < 6 || args[0] != "--host" || args[2] != "--config" || args[3] != filepath.Join(dir, "docker-config") {
					t.Fatal("unbound call", args)
				}
				if _, ok := ctx.Deadline(); !ok {
					t.Fatal("unbounded call")
				}
				a := args[4:]
				switch {
				case reflect.DeepEqual(a, []string{"image", "inspect", candidate.Image}):
					return original(ctx, args, env)
				case reflect.DeepEqual(a, []string{"container", "ls", "--all", "--quiet", "--no-trunc"}):
					lists++
					var ids []string
					for _, v := range records {
						ids = append(ids, v.ID)
					}
					if mode == "ids-changed" && lists > 1 {
						ids = append(ids, strings.Repeat("f", 64))
					}
					return strings.Join(ids, "\n"), nil
				case len(a) > 4 && reflect.DeepEqual(a[:4], []string{"container", "inspect", "--format", projectMountFormat}):
					reads++
					var ids []string
					for _, v := range records {
						ids = append(ids, v.ID)
					}
					slices.Sort(ids)
					if !reflect.DeepEqual(ids, a[4:]) {
						t.Fatal("mount query not bound to listed IDs")
					}
					if mode == "malformed" {
						return "[", nil
					}
					if reads == 2 {
						switch mode {
						case "source-replaced":
							if err = os.Rename(source, source+"-retained"); err != nil {
								t.Fatal(err)
							}
							if err = os.Mkdir(source, 0700); err != nil {
								t.Fatal(err)
							}
						case "target-replaced":
							if err = os.Rename(path, path+"-retained"); err != nil {
								t.Fatal(err)
							}
							if err = os.Mkdir(path, 0700); err != nil {
								t.Fatal(err)
							}
						case "cancelled":
							cancel()
						}
					}
					var out strings.Builder
					for i, v := range records {
						if mode == "duplicate-id" {
							v.ID = anchor.ID
						}
						if mode == "inventory-changed" && reads > 1 && i == 1 {
							v.Mounts = slices.Clone(v.Mounts)
							v.Mounts[0].RW = false
						}
						b, _ := json.Marshal(v)
						if mode == "missing-mounts" {
							b, _ = json.Marshal(map[string]string{"id": v.ID})
						}
						out.Write(b)
						out.WriteByte('\n')
					}
					if mode == "extra-row" {
						out.WriteString("{}\n")
					}
					return out.String(), nil
				case reflect.DeepEqual(a, []string{"container", "inspect", ownID}):
					b, _ := json.Marshal([]container{member})
					return string(b), nil
				default:
					t.Fatal("unexpected mutation or command", a)
					return "", errors.New("unexpected")
				}
			}
			raw, err := openProjectAnchorStart(ctx, resource.Selection, c.selection.Anchor.Prepared, f.d)
			if err != nil {
				t.Fatal(err)
			}
			engine := raw.(*projectAnchorEngine)
			err = engine.rejectConflictingProjectMounts(ctx, c, ownID)
			success := mode == "empty" || mode == "unrelated" || mode == "own-created" || mode == "own-reordered"
			if (err == nil) != success {
				t.Fatalf("%s: %v", mode, err)
			}
			if mode == "source-replaced" {
				if _, err = os.Stat(source + "-retained"); err != nil {
					t.Fatal("test lost original source", err)
				}
			}
			if mode == "target-replaced" {
				if _, err = os.Stat(path + "-retained"); err != nil {
					t.Fatal("test lost original parent", err)
				}
			}
		})
	}
}

func TestProjectMountObjectsDistinguishDescendantsAndSiblings(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a")
	child := filepath.Join(a, "child")
	sibling := filepath.Join(root, "ab")
	for _, p := range []string{child, sibling} {
		if err := os.MkdirAll(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	objects := func(p string) []projectPathObject {
		v, err := projectPathObjects(context.Background(), p)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	parent := objects(a)
	if !projectPathsOverlap(parent, objects(child)) || !projectPathsOverlap(objects(child), parent) || projectPathsOverlap(parent, objects(sibling)) {
		t.Fatal("incorrect overlap")
	}
	before := objects(a)
	if err := os.Rename(a, a+"-retained"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(a, 0700); err != nil {
		t.Fatal(err)
	}
	if sameProjectPathObjects(before, objects(a)) {
		t.Fatal("replacement retained object evidence")
	}
}

func TestConsumedKeyMountExceptionIsNarrow(t *testing.T) {
	base := t.TempDir()
	if err := os.Mkdir(filepath.Join(base, "keys"), 0700); err != nil {
		t.Fatal(err)
	}
	q := &qualifier{base: base, d: defaults(), uid: os.Geteuid(), gid: os.Getegid()}
	m := mount{Type: "bind", Source: filepath.Join(base, "keys", strings.Repeat("a", 32)+".key"), Destination: "/artifacts/boot.key", Propagation: "rprivate"}
	if !q.consumedBootKeyMount(m) {
		t.Fatal("consumed read-only key refused")
	}
	for _, change := range []func(*mount){func(v *mount) { v.RW = true }, func(v *mount) { v.Destination = "/urbit" }, func(v *mount) { v.Source = filepath.Join(base, "missing") }, func(v *mount) { v.Type = "volume" }} {
		v := m
		change(&v)
		if q.consumedBootKeyMount(v) {
			t.Fatal("unresolved mount exempted", v)
		}
	}
	if err := os.WriteFile(m.Source, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	if q.consumedBootKeyMount(m) {
		t.Fatal("existing source skipped")
	}
}

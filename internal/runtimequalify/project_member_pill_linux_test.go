//go:build linux && amd64

package runtimequalify

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

func TestProjectMemberVerifiedPillDispatchSnapshot(t *testing.T) {
	for _, mode := range []string{"unchanged", "read-atime", "never-verified", "changed-content-restored-mtime", "same-content-replacement", "writable", "hardlink", "symlink", "extra-artifact", "directory-replaced", "cancelled", "bad-hash", "changed-during-hash"} {
		t.Run(mode, func(t *testing.T) {
			f, _, _ := managedFixture(t)
			base := filepath.Join(f.d.home, ".local/share/urbit-runner/managed")
			workspace := filepath.Join(base, "private-workspace")
			pill := filepath.Join(workspace, candidate.PillName)
			if err := os.Mkdir(workspace, 0700); err != nil {
				t.Fatal(err)
			}
			data := []byte("tiny disposable test pill\n")
			if err := os.WriteFile(pill, data, 0400); err != nil {
				t.Fatal(err)
			}
			id, err := pierstore.IdentifyMount(workspace)
			if err != nil {
				t.Fatal(err)
			}
			q := &qualifier{d: f.d, base: base, uid: f.i.uid, gid: f.i.gid}
			e := &projectMemberLaunchEngine{anchor: &projectAnchorEngine{base: &projectDockerEngine{q: q}}, contract: projectMemberContract{pillPath: pill}, workspace: id}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "never-verified" {
				if e.checkVerifiedWorkspace(ctx) == nil {
					t.Fatal("unverified artifact accepted")
				}
				return
			}
			if mode == "bad-hash" {
				e.anchor.base.q.d.pillSHA = strings.Repeat("0", 64)
			}
			if mode == "changed-during-hash" {
				original := q.d.stat
				reads := 0
				q.d.stat = func(path string) (node, error) {
					if path == pill {
						reads++
						if reads == 2 {
							if err := os.Chmod(pill, 0600); err != nil {
								t.Fatal(err)
							}
							if err := os.WriteFile(pill, []byte(strings.Repeat("x", len(data))), 0400); err != nil {
								t.Fatal(err)
							}
							if err := os.Chmod(pill, 0400); err != nil {
								t.Fatal(err)
							}
						}
					}
					return original(path)
				}
			}
			err = e.checkWorkspace(ctx)
			if mode == "bad-hash" || mode == "changed-during-hash" {
				if err == nil || e.verifiedPill != (projectPillSnapshot{}) || e.checkVerifiedWorkspace(ctx) == nil {
					t.Fatal("unverified snapshot escaped", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			must := func(err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
			}
			switch mode {
			case "read-atime":
				_, err = os.ReadFile(pill)
				must(err)
			case "changed-content-restored-mtime":
				info, err := os.Stat(pill)
				must(err)
				must(os.Chmod(pill, 0600))
				must(os.WriteFile(pill, []byte(strings.Repeat("x", len(data))), 0400))
				must(os.Chmod(pill, 0400))
				must(os.Chtimes(pill, info.ModTime(), info.ModTime()))
			case "same-content-replacement":
				must(os.Rename(pill, filepath.Join(base, "original-pill")))
				must(os.WriteFile(pill, data, 0400))
			case "writable":
				must(os.Chmod(pill, 0600))
			case "hardlink":
				must(os.Link(pill, filepath.Join(base, "second-link")))
			case "symlink":
				must(os.Rename(pill, filepath.Join(base, "original-pill")))
				must(os.Symlink(filepath.Join(base, "original-pill"), pill))
			case "extra-artifact":
				must(os.WriteFile(filepath.Join(workspace, "other"), []byte("keep"), 0600))
			case "directory-replaced":
				must(os.Rename(workspace, workspace+"-original"))
				must(os.Mkdir(workspace, 0700))
				must(os.WriteFile(pill, data, 0400))
			case "cancelled":
				cancel()
			}
			err = e.checkVerifiedWorkspace(ctx)
			if (err == nil) != (mode == "unchanged" || mode == "read-atime") {
				t.Fatal("snapshot result", mode, err)
			}
		})
	}
}

//go:build linux && amd64

package managed

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type provisioningWorkspaceBackend struct {
	*queuedWorkspaceBackend
	provisions atomic.Int32
}

func (b *provisioningWorkspaceBackend) ProvisionPrivateWorkspaceParent(ctx context.Context, path string) (pierstore.DirectoryIdentity, error) {
	b.provisions.Add(1)
	return (realBackend{}).ProvisionPrivateWorkspaceParent(ctx, path)
}

func TestPrivateWorkspaceProvisionFirstUseThenActorReplay(t *testing.T) {
	r, b, p, req := privateWorkspaceQueueFixture(t)
	g02Check(t, os.Remove(b.directory))
	backend := &provisioningWorkspaceBackend{queuedWorkspaceBackend: b}
	r.b = backend
	op, hit, err := r.queuePrivateWorkspace("owner", p.Binding.ProjectID, req)
	g02Check(t, err)
	if hit || op.Status != "queued" || backend.provisions.Load() != 1 || b.inspected.Load() != 0 {
		t.Fatal("first-use setup differs")
	}
	w := workspaceOpened(t, b)
	if w.Parent.Inode == 0 {
		t.Fatal("new parent identity not retained")
	}
	// Exact replay is independent of the filesystem even before the worker runs.
	g02Check(t, os.Rename(b.directory, b.directory+"-retained"))
	for _, replayOnly := range []bool{false, true} {
		req.ReplayOnly = replayOnly
		replay, hit, err := r.queuePrivateWorkspace("owner", p.Binding.ProjectID, req)
		g02Check(t, err)
		if !hit || replay.ID != op.ID || backend.provisions.Load() != 1 || b.inspected.Load() != 0 {
			t.Fatal("replay probed or recreated parent")
		}
	}
	close(b.release)
	waitProjectWorker(t, r)
	actual, err := loadOperation(r.s.db, op.ID)
	g02Check(t, err)
	if actual.Status != "interrupted" {
		t.Fatal("missing bound parent not preserved as uncertainty", actual.Status)
	}
	if _, err = os.Stat(b.directory); !os.IsNotExist(err) {
		t.Fatal("worker recreated missing parent")
	}
}

func TestPrivateWorkspaceProvisionCompletesRealEmptyLayout(t *testing.T) {
	r, b, p, req := privateWorkspaceQueueFixture(t)
	g02Check(t, os.Remove(b.directory))
	backend := &provisioningWorkspaceBackend{queuedWorkspaceBackend: b}
	r.b = backend
	close(b.release)
	op, _, err := r.queuePrivateWorkspace("owner", p.Binding.ProjectID, req)
	g02Check(t, err)
	waitProjectWorker(t, r)
	actual, err := loadOperation(r.s.db, op.ID)
	g02Check(t, err)
	w, err := loadPrivateWorkspace(r.s.db, p.Binding.ProjectID)
	g02Check(t, err)
	if actual.Status != "succeeded" || w.Phase != "observed" || backend.provisions.Load() != 1 {
		t.Fatal("first-use layout failed")
	}
	for _, m := range w.Mounts {
		entries, err := os.ReadDir(filepath.Join(r.s.directory, m.Path))
		if err != nil || len(entries) != 0 {
			t.Fatal("created a pier/key", err)
		}
	}
}

func TestPrivateWorkspaceProvisionRejectsFreshInvalidIntentBeforeFilesystem(t *testing.T) {
	for _, mode := range []string{"revision", "review", "project", "request", "actor", "replay-only", "busy", "fence", "closed", "canceled"} {
		t.Run(mode, func(t *testing.T) {
			r, b, p, req := privateWorkspaceQueueFixture(t)
			g02Check(t, os.Remove(b.directory))
			backend := &provisioningWorkspaceBackend{queuedWorkspaceBackend: b}
			r.b = backend
			actor := "owner"
			switch mode {
			case "revision":
				req.ExpectedRevision++
			case "review":
				req.ReviewDigest = strings.Repeat("f", 64)
			case "project":
				p.Binding.ProjectID = "private-project-missing"
			case "request":
				req.RequestID = "../invalid"
			case "actor":
				actor = ""
			case "replay-only":
				req.ReplayOnly = true
			case "busy":
				r.workGate <- struct{}{}
				defer func() { <-r.workGate }()
			case "fence":
				r.updateFence.Held = true
			case "closed":
				r.closing = true
			case "canceled":
				r.cancel()
			}
			_, hit, err := r.queuePrivateWorkspace(actor, p.Binding.ProjectID, req)
			if err == nil || hit || backend.provisions.Load() != 0 || b.inspected.Load() != 0 {
				t.Fatal("invalid request touched storage", err)
			}
			if _, err = os.Lstat(b.directory); !os.IsNotExist(err) {
				t.Fatal("invalid request created parent")
			}
			var count int
			g02Check(t, r.s.db.QueryRow("SELECT count(*) FROM private_project_workspaces").Scan(&count))
			if count != 0 {
				t.Fatal("invalid request recorded workspace")
			}
		})
	}
}

func TestPrivateWorkspaceProvisionNeverRecreatesOrRebindsClaimedParent(t *testing.T) {
	for _, mode := range []string{"missing", "replaced", "permissions"} {
		t.Run(mode, func(t *testing.T) {
			r, b, p, req := privateWorkspaceQueueFixture(t)
			backend := &provisioningWorkspaceBackend{queuedWorkspaceBackend: b}
			r.b = backend
			close(b.release)
			first, _, err := r.queuePrivateWorkspace("owner", p.Binding.ProjectID, req)
			g02Check(t, err)
			waitProjectWorker(t, r)
			old, err := loadPrivateWorkspace(r.s.db, p.Binding.ProjectID)
			g02Check(t, err)
			registered, _, err := r.s.registerPrivateProject("owner", privateProjectRequest(t, "second-registration", "second", 19180))
			g02Check(t, err)
			second, _ := readPrivateProject(t, r.s, registered.ProjectID)
			digest, err := second.Plan.Digest()
			g02Check(t, err)
			request := privateWorkspaceRequest{RequestID: "second-workspace", ExpectedRevision: 1, ReviewDigest: digest}
			if mode == "permissions" {
				g02Check(t, os.Chmod(b.directory, 0755))
			} else {
				g02Check(t, os.Rename(b.directory, b.directory+"-retained"))
				if mode == "replaced" {
					g02Check(t, os.Mkdir(b.directory, 0700))
				}
			}
			_, hit, err := r.queuePrivateWorkspace("owner", second.Binding.ProjectID, request)
			requireFault(t, err, "private_storage_recovery_required")
			if hit || backend.provisions.Load() != 1 {
				t.Fatal("claimed parent was provisioned again")
			}
			saved, err := loadPrivateWorkspace(r.s.db, p.Binding.ProjectID)
			g02Check(t, err)
			if saved.Parent != old.Parent || saved.OperationID != first.ID {
				t.Fatal("original workspace changed")
			}
			if mode == "missing" {
				if _, err = os.Lstat(b.directory); !os.IsNotExist(err) {
					t.Fatal("missing claimed parent recreated")
				}
			}
		})
	}
}

func TestPrivateWorkspaceProvisionPreservesOrphanDataBeforeAnyReceipt(t *testing.T) {
	r, b, p, req := privateWorkspaceQueueFixture(t)
	backend := &provisioningWorkspaceBackend{queuedWorkspaceBackend: b}
	r.b = backend
	path := filepath.Join(b.directory, "unknown-project")
	g02Check(t, os.Mkdir(path, 0700))
	file := filepath.Join(path, "retained")
	g02Check(t, os.WriteFile(file, []byte("synthetic retained content"), 0600))
	if _, _, err := r.queuePrivateWorkspace("owner", p.Binding.ProjectID, req); err == nil {
		t.Fatal("unknown data adopted")
	}
	raw, err := os.ReadFile(file)
	g02Check(t, err)
	if string(raw) != "synthetic retained content" {
		t.Fatal("orphan data changed")
	}
	var count int
	g02Check(t, r.s.db.QueryRow("SELECT count(*) FROM private_project_workspaces").Scan(&count))
	if count != 0 {
		t.Fatal("orphan parent accepted")
	}
}

func TestPrivateWorkspaceProvisionRefusesRetainedMaintenanceBeforeFilesystem(t *testing.T) {
	r, b, p, req := privateWorkspaceQueueFixture(t)
	g02Check(t, os.Remove(b.directory))
	backend := &provisioningWorkspaceBackend{queuedWorkspaceBackend: b}
	r.b = backend
	_, _, err := r.s.register("owner", registration("legacy", "legacy", 18580))
	g02Check(t, err)
	v, err := loadInstance(r.s.db, "legacy")
	g02Check(t, err)
	v.Runtime.Runtime.Containers = append(v.Runtime.Runtime.Containers, runtimequalify.ContainerRecord{Kind: "pack", CreateUncertain: true})
	tx, err := r.s.db.Begin()
	g02Check(t, err)
	g02Check(t, saveInstance(tx, v))
	g02Check(t, tx.Commit())
	_, hit, err := r.queuePrivateWorkspace("owner", p.Binding.ProjectID, req)
	requireFault(t, err, "host_recovery_required")
	if hit || backend.provisions.Load() != 0 || b.inspected.Load() != 0 {
		t.Fatal("retained maintenance reached filesystem")
	}
	if _, err = os.Lstat(b.directory); !os.IsNotExist(err) {
		t.Fatal("retained maintenance created parent")
	}
	if len(r.workGate) != 0 {
		t.Fatal("refusal leaked work gate")
	}
	var count int
	g02Check(t, r.s.db.QueryRow("SELECT count(*) FROM private_project_workspaces").Scan(&count))
	if count != 0 {
		t.Fatal("refusal recorded workspace")
	}
}

func TestPrivateWorkspaceProvisionRetainsEmptyParentAfterReceiptRollback(t *testing.T) {
	for _, target := range []string{"journal", "mirror"} {
		t.Run(target, func(t *testing.T) {
			r, b, p, req := privateWorkspaceQueueFixture(t)
			g02Check(t, os.Remove(b.directory))
			backend := &provisioningWorkspaceBackend{queuedWorkspaceBackend: b}
			r.b = backend
			stmt := `CREATE TRIGGER fail_workspace BEFORE INSERT ON private_project_workspaces BEGIN SELECT RAISE(ABORT,'synthetic journal failure'); END;`
			if target == "mirror" {
				stmt = `CREATE TRIGGER fail_workspace BEFORE UPDATE OF runtime ON operations BEGIN SELECT RAISE(ABORT,'synthetic mirror failure'); END;`
			}
			_, err := r.s.db.Exec(stmt)
			g02Check(t, err)
			if _, hit, err := r.queuePrivateWorkspace("owner", p.Binding.ProjectID, req); err == nil || hit {
				t.Fatal("failed receipt accepted")
			}
			parent, err := privatechain.InspectWorkspaceParent(context.Background(), b.directory)
			g02Check(t, err)
			entries, err := os.ReadDir(b.directory)
			g02Check(t, err)
			if len(entries) != 0 || backend.provisions.Load() != 1 || len(r.workGate) != 0 {
				t.Fatal("failed receipt altered metadata or retained gate")
			}
			var count int
			g02Check(t, r.s.db.QueryRow("SELECT count(*) FROM private_project_workspaces").Scan(&count))
			if count != 0 {
				t.Fatal("failed receipt left workspace")
			}
			g02Check(t, r.s.db.QueryRow("SELECT count(*) FROM operations WHERE request_id=?", req.RequestID).Scan(&count))
			if count != 0 {
				t.Fatal("failed receipt left operation")
			}
			select {
			case <-b.opened:
				t.Fatal("worker ran without receipt")
			default:
			}
			_, err = r.s.db.Exec("DROP TRIGGER fail_workspace")
			g02Check(t, err)
			close(b.release)
			op, hit, err := r.queuePrivateWorkspace("owner", p.Binding.ProjectID, req)
			g02Check(t, err)
			waitProjectWorker(t, r)
			actual, err := loadOperation(r.s.db, op.ID)
			g02Check(t, err)
			w, err := loadPrivateWorkspace(r.s.db, p.Binding.ProjectID)
			g02Check(t, err)
			if hit || actual.Status != "succeeded" || !w.Parent.SameObject(parent) || backend.provisions.Load() != 2 {
				t.Fatal("initial metadata retry replaced parent or failed")
			}
		})
	}
}

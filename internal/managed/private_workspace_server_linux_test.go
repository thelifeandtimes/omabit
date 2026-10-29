//go:build linux && amd64

package managed

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

type queuedWorkspaceBackend struct {
	fakeBackend
	directory string
	inspected atomic.Int32
	closed    atomic.Int32
	opened    chan privatechain.Workspace
	release   chan struct{}
	fail      string
}

func (b *queuedWorkspaceBackend) InspectPrivateWorkspaceParent(ctx context.Context, directory string) (pierstore.DirectoryIdentity, error) {
	b.inspected.Add(1)
	if directory != b.directory {
		return pierstore.DirectoryIdentity{}, errors.New("backend path was not selected by store")
	}
	return privatechain.InspectWorkspaceParent(ctx, directory)
}
func (b *queuedWorkspaceBackend) OpenPrivateWorkspace(ctx context.Context, directory string, p privatechain.Plan, w privatechain.Workspace) (privateWorkspaceDriver, error) {
	if directory != b.directory {
		return nil, errors.New("worker path differs")
	}
	b.opened <- w.Clone()
	select {
	case <-b.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if b.fail == "setup" {
		return nil, errors.New("synthetic setup failure")
	}
	if b.fail == "nil" {
		return nil, nil
	}
	if b.fail == "typed-nil" || b.fail == "typed-nil-error" {
		var driver *countedWorkspaceDriver
		if b.fail == "typed-nil-error" {
			return driver, errors.New("synthetic missing driver")
		}
		return driver, nil
	}
	fs, err := privatechain.OpenWorkspaceFilesystem(ctx, directory, w.Parent, p, w)
	if err != nil {
		return nil, err
	}
	driver := &countedWorkspaceDriver{WorkspaceFilesystem: fs, backend: b}
	if b.fail == "setup-with-driver" {
		return driver, errors.New("synthetic partial setup")
	}
	return driver, nil
}

type countedWorkspaceDriver struct {
	*privatechain.WorkspaceFilesystem
	backend *queuedWorkspaceBackend
}

func (d *countedWorkspaceDriver) Close() error {
	d.backend.closed.Add(1)
	err := d.WorkspaceFilesystem.Close()
	if d.backend.fail == "close" {
		return errors.Join(err, errors.New("synthetic close error"))
	}
	return err
}
func privateWorkspaceQueueFixture(t *testing.T) (*Server, *queuedWorkspaceBackend, storedPrivateProject, privateWorkspaceRequest) {
	t.Helper()
	s := testStore(t)
	p, req := privateWorkspaceFixture(t, s)
	directory := filepath.Join(s.directory, "private-projects")
	g02Check(t, os.Mkdir(directory, 0700))
	b := &queuedWorkspaceBackend{directory: directory, opened: make(chan privatechain.Workspace, 2), release: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	r := &Server{s: s, b: b, ctx: ctx, cancel: cancel, workGate: make(chan struct{}, 1), executableSHA256: strings.Repeat("f", 64)}
	t.Cleanup(func() { cancel(); r.wg.Wait() })
	return r, b, p, req
}
func workspaceOpened(t *testing.T, b *queuedWorkspaceBackend) privatechain.Workspace {
	t.Helper()
	select {
	case w := <-b.opened:
		return w
	case <-time.After(2 * time.Second):
		t.Fatal("workspace driver was not opened")
		return privatechain.Workspace{}
	}
}
func TestPrivateWorkspaceQueueUsesHostGateAndRealTemporaryFilesystem(t *testing.T) {
	r, b, p, req := privateWorkspaceQueueFixture(t)
	r.workGate <- struct{}{}
	op, hit, err := r.queuePrivateWorkspace("owner", p.Binding.ProjectID, req)
	g02Check(t, err)
	if hit || op.Status != "queued" || b.inspected.Load() != 1 {
		t.Fatal(op, hit, b.inspected.Load())
	}
	replay, hit, err := r.queuePrivateWorkspace("owner", p.Binding.ProjectID, req)
	g02Check(t, err)
	if !hit || replay.ID != op.ID || b.inspected.Load() != 1 {
		t.Fatal("replay read parent or changed receipt")
	}
	select {
	case <-b.opened:
		t.Fatal("workspace bypassed host gate")
	default:
	}
	if entries, err := os.ReadDir(b.directory); err != nil || len(entries) != 0 {
		t.Fatal("directory mutated before gate", err)
	}
	if response := rc11Fence(r, "acquire", strings.Repeat("a", 64)); response.Code != 409 {
		t.Fatal("management fence ignored queued worker", response.Code)
	}
	<-r.workGate
	w := workspaceOpened(t, b)
	if w.Binding != p.Binding || w.OperationID != op.ID || w.Revision != p.Revision || w.Phase != "planned" {
		t.Fatal("worker selection differs", w)
	}
	if response := rc11Fence(r, "acquire", strings.Repeat("a", 64)); response.Code != 409 {
		t.Fatal("management fence ignored active worker", response.Code)
	}
	close(b.release)
	waitProjectWorker(t, r)
	actual, err := loadOperation(r.s.db, op.ID)
	g02Check(t, err)
	saved, err := loadPrivateWorkspace(r.s.db, p.Binding.ProjectID)
	g02Check(t, err)
	if actual.Status != "succeeded" || saved.Phase != "observed" || b.closed.Load() != 1 {
		t.Fatal(actual, saved, b.closed.Load())
	}
	for _, mount := range saved.Mounts {
		entries, err := os.ReadDir(filepath.Join(r.s.directory, mount.Path))
		g02Check(t, err)
		if len(entries) != 0 {
			t.Fatal("workspace created a pier or key")
		}
	}
	g02Check(t, validatePrivateWorkspaces(r.s.db))
	if response := rc11Fence(r, "acquire", strings.Repeat("a", 64)); response.Code != 200 {
		t.Fatal("completed worker retained management fence", response.Code)
	}
	req.RequestID = "another-request"
	_, _, err = r.queuePrivateWorkspace("owner", p.Binding.ProjectID, req)
	requireFault(t, err, "management_quiesced")
}
func TestPrivateWorkspaceQueueReplayDoesNotInspectMissingOrReplacedParent(t *testing.T) {
	r, b, p, req := privateWorkspaceQueueFixture(t)
	r.workGate <- struct{}{}
	op, _, err := r.queuePrivateWorkspace("owner", p.Binding.ProjectID, req)
	g02Check(t, err)
	original := b.directory + "-retained"
	g02Check(t, os.Rename(b.directory, original))
	// Missing-parent replay succeeds without filesystem reads or enqueueing.
	replay, hit, err := r.queuePrivateWorkspace("owner", p.Binding.ProjectID, req)
	g02Check(t, err)
	if !hit || replay.ID != op.ID || b.inspected.Load() != 1 {
		t.Fatal("replay touched missing parent")
	}
	g02Check(t, os.Mkdir(b.directory, 0700))
	close(b.release)
	<-r.workGate
	waitProjectWorker(t, r)
	actual, err := loadOperation(r.s.db, op.ID)
	g02Check(t, err)
	if actual.Status != "interrupted" || b.closed.Load() != 0 {
		t.Fatal("replacement parent adopted", actual)
	}
	for _, directory := range []string{original, b.directory} {
		entries, err := os.ReadDir(directory)
		g02Check(t, err)
		if len(entries) != 0 {
			t.Fatal("replacement or original directory modified")
		}
	}
	replay, hit, err = r.queuePrivateWorkspace("owner", p.Binding.ProjectID, req)
	g02Check(t, err)
	if !hit || replay.Status != "interrupted" || b.inspected.Load() != 1 {
		t.Fatal("uncertain worker retried", replay)
	}
}
func TestPrivateWorkspaceQueueCancellationAndSetupFailuresRetainIntent(t *testing.T) {
	for _, kind := range []string{"waiting", "setup", "nil", "typed-nil", "typed-nil-error", "setup-with-driver", "cancel-open", "close"} {
		t.Run(kind, func(t *testing.T) {
			r, b, p, req := privateWorkspaceQueueFixture(t)
			b.fail = kind
			if kind == "waiting" {
				r.workGate <- struct{}{}
			}
			op, _, err := r.queuePrivateWorkspace("owner", p.Binding.ProjectID, req)
			g02Check(t, err)
			if kind == "waiting" {
				r.cancel()
			} else {
				workspaceOpened(t, b)
				if kind == "cancel-open" {
					r.cancel()
				} else {
					close(b.release)
				}
			}
			waitProjectWorker(t, r)
			actual, err := loadOperation(r.s.db, op.ID)
			g02Check(t, err)
			want := "interrupted"
			if kind == "close" {
				want = "succeeded"
			}
			if actual.Status != want || r.failed.Load() {
				t.Fatal("lost durable outcome", actual, r.failed.Load())
			}
			expectedCloses := int32(0)
			if kind == "close" || kind == "setup-with-driver" {
				expectedCloses = 1
			}
			if b.closed.Load() != expectedCloses {
				t.Fatal("driver authority not closed", b.closed.Load())
			}
			req.ReplayOnly = true
			replay, hit, err := r.s.acceptPrivateWorkspace("owner", p.Binding.ProjectID, req, pierstore.DirectoryIdentity{})
			g02Check(t, err)
			if !hit || replay.Status != want || replay.ID != op.ID {
				t.Fatal(replay, hit)
			}
			g02Check(t, validatePrivateWorkspaces(r.s.db))
		})
	}
}
func TestPrivateWorkspaceQueueRefusesBeforeFilesystemOrCreatesNothing(t *testing.T) {
	for _, kind := range []string{"closed", "failed", "canceled", "fenced", "no-backend", "lookup-miss", "bad-request", "missing-parent"} {
		t.Run(kind, func(t *testing.T) {
			r, b, p, req := privateWorkspaceQueueFixture(t)
			switch kind {
			case "closed":
				r.closing = true
			case "failed":
				r.failed.Store(true)
			case "canceled":
				r.cancel()
			case "fenced":
				r.updateFence.Held = true
			case "no-backend":
				r.b = &fakeBackend{}
			case "lookup-miss":
				req.ReplayOnly = true
			case "bad-request":
				req.RequestID = "/invalid"
			case "missing-parent":
				g02Check(t, os.Remove(b.directory))
			}
			if _, _, err := r.queuePrivateWorkspace("owner", p.Binding.ProjectID, req); err == nil {
				t.Fatal("invalid queue admitted")
			}
			count := int32(0)
			if kind == "missing-parent" {
				count = 1
			}
			if b.inspected.Load() != count {
				t.Fatal("filesystem inspected before admission")
			}
			var workspaces int
			g02Check(t, r.s.db.QueryRow("SELECT count(*) FROM private_project_workspaces").Scan(&workspaces))
			if workspaces != 0 {
				t.Fatal("refusal created intent")
			}
			select {
			case <-b.opened:
				t.Fatal("refusal opened worker")
			default:
			}
		})
	}
}

func TestPrivateWorkspaceRealBackendOpenFailureReturnsNoDriver(t *testing.T) {
	r, _, p, req := privateWorkspaceQueueFixture(t)
	op, _, err := r.s.acceptPrivateWorkspace("owner", p.Binding.ProjectID, req, privateWorkspaceParent())
	g02Check(t, err)
	w, err := loadPrivateWorkspace(r.s.db, op.ProjectID)
	g02Check(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, ctx := range []context.Context{ctx, context.Background()} {
		driver, err := (realBackend{}).OpenPrivateWorkspace(ctx, filepath.Join(r.s.directory, "private-projects"), p.Plan, w)
		if err == nil || driver != nil {
			t.Fatal("failed factory returned typed-nil driver authority", driver, err)
		}
	}
}

// Retained receipts remain readable after this runner loses runtime support.
func TestPrivateWorkspaceQueueReplaysWithoutRuntimeBackend(t *testing.T) {
	r, b, p, req := privateWorkspaceQueueFixture(t)
	close(b.release)
	op, _, err := r.queuePrivateWorkspace("owner", p.Binding.ProjectID, req)
	g02Check(t, err)
	waitProjectWorker(t, r)
	original, err := loadOperation(r.s.db, op.ID)
	g02Check(t, err)
	if original.Status != "succeeded" {
		t.Fatal("setup did not finish")
	}
	r.b = &fakeBackend{}
	for _, replayOnly := range []bool{false, true} {
		req.ReplayOnly = replayOnly
		replay, hit, err := r.queuePrivateWorkspace("owner", p.Binding.ProjectID, req)
		g02Check(t, err)
		if !hit || replay.ID != original.ID || replay.Status != original.Status || b.inspected.Load() != 1 || b.closed.Load() != 1 {
			t.Fatal("replay depended on runtime or changed intent")
		}
	}
	req.ReplayOnly = false
	req.RequestID = "fresh-without-runtime"
	if _, hit, err := r.queuePrivateWorkspace("owner", p.Binding.ProjectID, req); err == nil || hit {
		t.Fatal("missing runtime admitted a new mutation")
	}
	if b.inspected.Load() != 1 || b.closed.Load() != 1 {
		t.Fatal("replay accessed removed runtime")
	}
}

func TestPrivateWorkspaceWorkerRejectsReboundQueuedReceipt(t *testing.T) {
	for _, field := range []string{"actor", "request", "revision", "project", "operation"} {
		t.Run(field, func(t *testing.T) {
			r, b, p, req := privateWorkspaceQueueFixture(t)
			r.workGate <- struct{}{}
			op, _, err := r.queuePrivateWorkspace("owner", p.Binding.ProjectID, req)
			g02Check(t, err)
			forged := op
			switch field {
			case "actor":
				forged.Actor = "another-owner"
			case "request":
				forged.RequestID = "another-request"
			case "revision":
				forged.ConfigRevision++
			case "project":
				forged.ProjectID = "private-project-" + strings.Repeat("d", 32)
			case "operation":
				forged.ID = "another-operation"
			}
			if r.executePrivateWorkspace(b, forged) == nil {
				t.Fatal("rebound receipt opened workspace")
			}
			select {
			case <-b.opened:
				t.Fatal("driver opened for rebound receipt")
			default:
			}
			if entries, err := os.ReadDir(b.directory); err != nil || len(entries) != 0 {
				t.Fatal("rebound receipt changed tree", err)
			}
			r.cancel()
			waitProjectWorker(t, r)
			original, err := loadOperation(r.s.db, op.ID)
			g02Check(t, err)
			if original.Status != "interrupted" || b.closed.Load() != 0 {
				t.Fatal("original queued intent changed")
			}
		})
	}
}

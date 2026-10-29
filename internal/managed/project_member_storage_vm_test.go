//go:build linux && amd64

package managed

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

// The shared guard admits only the fresh no-NIC disposable VM, never the host.
func TestProjectMemberStorageDisposableVM(t *testing.T) {
	exerciseProjectDisposableVM(t, true, true, false, false, false, false, false, false)
}

type vmMemberStorageBackend struct {
	fakeBackend
	store   *store
	wrapped *vmMemberStorageDriver
}

func (b *vmMemberStorageBackend) OpenProjectMemberStorage(ctx context.Context, p projectnetwork.Plan, s projectnetwork.MemberStorage) (projectnetwork.MemberStorageDriver, error) {
	d, err := runtimequalify.OpenProjectMemberStorage(ctx, p, s)
	if err != nil {
		return nil, err
	}
	b.wrapped = &vmMemberStorageDriver{store: b.store, driver: d}
	return b.wrapped, nil
}

type vmMemberStorageDriver struct {
	store  *store
	driver projectnetwork.MemberStorageDriver
	calls  []string
}

func (d *vmMemberStorageDriver) check(s projectnetwork.MemberStorage, phase string) error {
	actual, err := loadProjectMemberStorage(d.store.db, s.InstanceID)
	if err != nil {
		return err
	}
	if actual.Phase != phase || !reflect.DeepEqual(actual, s) {
		return errors.New("filesystem action preceded its durable member journal")
	}
	d.calls = append(d.calls, phase)
	return nil
}
func (d *vmMemberStorageDriver) CreateMount(ctx context.Context, s projectnetwork.MemberStorage) (pierstore.DirectoryIdentity, error) {
	if err := d.check(s, "dispatched"); err != nil {
		return pierstore.DirectoryIdentity{}, err
	}
	return d.driver.CreateMount(ctx, s)
}
func (d *vmMemberStorageDriver) InspectMount(ctx context.Context, s projectnetwork.MemberStorage) error {
	if err := d.check(s, "created"); err != nil {
		return err
	}
	return d.driver.InspectMount(ctx, s)
}
func exerciseVMMemberStorage(t *testing.T, ctx context.Context, s *store, p storedProject, publicMode ...bool) []map[string]any {
	t.Helper()
	results := []map[string]any{}
	b := &vmMemberStorageBackend{store: s}
	child, cancel := context.WithCancel(ctx)
	r := &Server{s: s, b: b, ctx: child, cancel: cancel, workGate: make(chan struct{}, 1), uid: uint32(os.Geteuid())}
	defer func() { cancel(); r.wg.Wait() }()
	var cli *vmPublicRecoveryCLI
	if len(publicMode) > 0 && publicMode[0] {
		var openErr error
		cli, openErr = vmOpenPublicProjectCLI(ctx, r, p.ID, os.Getenv("OMABIT_TEST_PROJECT_VM"), "storage")
		if openErr != nil {
			t.Fatal(openErr)
		}
		defer cli.close()
	}

	digest, err := p.Plan.Digest()
	if err != nil {
		t.Fatal(err)
	}
	for _, member := range p.Members {
		result := map[string]any{"ok": false, "instance_id": member.InstanceID, "stage": "queue"}
		results = append(results, result)
		req := projectMemberStorageRequest{RequestID: "prepare-storage-" + member.InstanceID, ExpectedRevision: p.Revision, ReviewDigest: digest}

		queue := func(q projectMemberStorageRequest, replay bool) (protocol.Operation, bool, error) {
			if cli != nil {
				op, e := cli.operate("prepare-storage", member.InstanceID, q.RequestID, q.ExpectedRevision, q.ReviewDigest, replay)
				return op, replay, e
			}
			return r.queueProjectMemberStorage("qualification", p.ID, member.InstanceID, q)
		}
		if cli != nil {
			result["public_actions"] = cli.proof
		}
		op, hit, err := queue(req, false)
		if err != nil || hit {
			t.Error(op, hit, err)
			return results
		}
		result["operation_id"] = op.ID
		done := make(chan struct{})
		go func() { r.wg.Wait(); close(done) }()
		select {
		case <-done:
		case <-ctx.Done():
			t.Error("member storage worker deadline")
			return results
		}
		state, err := loadProjectMemberStorage(s.db, member.InstanceID)
		if err != nil {
			t.Error(err)
			return results
		}
		result["journal"] = state
		receipt, err := loadOperation(s.db, op.ID)
		if err != nil {
			t.Error(err)
			return results
		}
		result["operation"] = receipt
		if b.wrapped != nil {
			result["calls"] = b.wrapped.calls
		}
		if receipt.Status != "succeeded" || state.Phase != "observed" || state.RuntimeID != member.RuntimeID || b.wrapped == nil || !reflect.DeepEqual(b.wrapped.calls, []string{"dispatched", "created"}) {
			t.Error("incomplete member storage")
			return results
		}
		mount, err := pierstore.CheckMount("/home/lab", state.Location)
		if err != nil {
			t.Error(err)
			return results
		}
		result["mount_path"] = mount
		entries, err := os.ReadDir(mount)
		if err != nil || len(entries) != 0 {
			t.Error("member parent is not empty", err)
			return results
		}
		result["empty_parent"] = true
		info, err := os.Stat(mount)
		if err != nil || info.Mode().Perm() != 0700 {
			t.Error("wrong member parent mode", err)
			return results
		}
		result["mode"] = uint32(info.Mode().Perm())
		repeated, hit, err := queue(req, true)
		if err != nil || !hit || !reflect.DeepEqual(repeated, receipt) {
			t.Error("member storage replay changed receipt", err)
			return results
		}
		req.RequestID = "second-storage-" + member.InstanceID
		if _, _, err = queue(req, false); err == nil {
			t.Error("new request redispatched member storage")
			return results
		}
		if r.failed.Load() {
			t.Error("member worker failed runner")
			return results
		}
		result["repeat_dispatch_refused"] = true
		result["stage"] = "complete"
		result["ok"] = true
	}
	if cli != nil {
		cli.proof["ok"] = true
	}
	return results
}

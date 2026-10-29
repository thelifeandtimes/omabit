//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type queuedPrivateLifecycleBackend struct {
	fakeBackend
	t                 *testing.T
	s                 *store
	inputs            privatechain.RuntimeInputInspection
	inspected, closed atomic.Int32
	opened            chan string
	release           chan struct{}
	fail              string
}

func (b *queuedPrivateLifecycleBackend) InspectPrivateGatewayInputs(_ context.Context, directory string, p privatechain.Plan, w privatechain.Workspace, digest string) (string, privatechain.RuntimeInputInspection, error) {
	b.inspected.Add(1)
	if directory != b.s.directory || w.Validate(p) != nil || digest != b.inputs.ManifestSHA256 {
		return "", privatechain.RuntimeInputInspection{}, errors.New("input selection differs")
	}
	if b.fail == "inputs" {
		return "", privatechain.RuntimeInputInspection{}, errors.New("synthetic input refusal")
	}
	return "private-engine", b.inputs, nil
}
func (b *queuedPrivateLifecycleBackend) open(ctx context.Context, kind, directory, operation string) error {
	b.t.Helper()
	want := b.s.directory
	if kind == "workspace-observation" {
		want = filepath.Join(want, "private-projects")
	}
	if directory != want {
		b.t.Fatal("factory received an unowned path")
	}
	op, err := loadOperation(b.s.db, operation)
	g02Check(b.t, err)
	if op.Status != "queued" {
		b.t.Fatal("factory opened without a fresh queued receipt")
	}
	b.opened <- kind
	select {
	case <-b.release:
	case <-ctx.Done():
		return ctx.Err()
	}
	if b.fail == "setup" {
		return errors.New("synthetic factory refusal")
	}
	return nil
}
func (b *queuedPrivateLifecycleBackend) close() error {
	b.closed.Add(1)
	if b.fail == "close" {
		return errors.New("synthetic close failure")
	}
	return nil
}

type queuedGatewayDriver struct {
	*privateGatewayStoreDriver
	backend *queuedPrivateLifecycleBackend
}

func (d *queuedGatewayDriver) Close() error { return d.backend.close() }

type queuedWorkspaceObserver struct {
	*workspaceObservationOnly
	backend *queuedPrivateLifecycleBackend
}

func (d *queuedWorkspaceObserver) Close() error { return d.backend.close() }

type queuedGatewayObserver struct {
	*gatewayObservationOnly
	backend *queuedPrivateLifecycleBackend
}

func (d *queuedGatewayObserver) Close() error { return d.backend.close() }
func (b *queuedPrivateLifecycleBackend) OpenPrivateGateway(ctx context.Context, directory string, _ privatechain.Plan, _ privatechain.Workspace, g privatechain.Gateway) (privateGatewayDriver, error) {
	if err := b.open(ctx, "gateway", directory, g.Selection.OperationID); err != nil {
		return nil, err
	}
	if b.fail == "nil" {
		return nil, nil
	}
	if b.fail == "typed-nil" {
		var d *queuedGatewayDriver
		return d, nil
	}
	d := &queuedGatewayDriver{privateGatewayStoreDriver: &privateGatewayStoreDriver{t: b.t, s: b.s}, backend: b}
	if b.fail == "driver" {
		d.fail = "start-gateway"
	}
	if b.fail == "setup-with-driver" {
		return d, errors.New("synthetic partial setup")
	}
	return d, nil
}
func (b *queuedPrivateLifecycleBackend) OpenPrivateWorkspaceObserver(ctx context.Context, directory string, _ privatechain.Plan, _ privatechain.Workspace, operation string) (privateWorkspaceObserver, error) {
	if err := b.open(ctx, "workspace-observation", directory, operation); err != nil {
		return nil, err
	}
	if b.fail == "nil" {
		return nil, nil
	}
	if b.fail == "typed-nil" {
		var d *queuedWorkspaceObserver
		return d, nil
	}
	d := &queuedWorkspaceObserver{workspaceObservationOnly: &workspaceObservationOnly{t: b.t, s: b.s, operation: operation, fail: b.fail == "driver"}, backend: b}
	if b.fail == "setup-with-driver" {
		return d, errors.New("synthetic partial setup")
	}
	return d, nil
}
func (b *queuedPrivateLifecycleBackend) OpenPrivateGatewayObserver(ctx context.Context, directory string, _ privatechain.Plan, _ privatechain.Workspace, _ privatechain.Gateway, operation string) (privateGatewayObserver, error) {
	if err := b.open(ctx, "gateway-observation", directory, operation); err != nil {
		return nil, err
	}
	if b.fail == "nil" {
		return nil, nil
	}
	if b.fail == "typed-nil" {
		var d *queuedGatewayObserver
		return d, nil
	}
	d := &queuedGatewayObserver{gatewayObservationOnly: &gatewayObservationOnly{t: b.t, s: b.s, operation: operation, fail: b.fail == "driver"}, backend: b}
	if b.fail == "setup-with-driver" {
		return d, errors.New("synthetic partial setup")
	}
	return d, nil
}

type privateLifecycleAdmit func(actor, request string, replay bool) (protocol.Operation, bool, error)

func privateLifecycleQueueFixture(t *testing.T, kind string) (*Server, *queuedPrivateLifecycleBackend, privateLifecycleAdmit) {
	t.Helper()
	var s *store
	var p storedPrivateProject
	var inputs privatechain.RuntimeInputInspection
	var gatewayReq privateGatewayRequest
	var workspaceReq privateWorkspaceObservationRequest
	var observationReq privateGatewayObservationRequest
	var memberReq privateMemberLaunchRequest
	var instance string
	switch kind {
	case "member-launch":
		s, p, instance, memberReq = privateMemberStoreFixture(t)
	case "gateway":
		s, p, gatewayReq, inputs = privateGatewayStoreFixture(t)
	case "workspace-observation":
		s, p, workspaceReq = privateWorkspaceObservationFixture(t)
	case "gateway-observation":
		s, p, observationReq = privateGatewayObservationFixture(t)
	default:
		t.Fatal(kind)
	}
	b := &queuedPrivateLifecycleBackend{t: t, s: s, inputs: inputs, opened: make(chan string, 2), release: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	r := &Server{s: s, b: b, ctx: ctx, cancel: cancel, workGate: make(chan struct{}, 1), executableSHA256: strings.Repeat("f", 64)}
	t.Cleanup(func() { cancel(); r.wg.Wait() })
	admit := func(actor, request string, replay bool) (protocol.Operation, bool, error) {
		switch kind {
		case "member-launch":
			req := memberReq
			req.RequestID = request
			req.ReplayOnly = replay
			return r.queuePrivateMemberLaunch(actor, p.Binding.ProjectID, instance, req)
		case "gateway":
			req := gatewayReq
			req.RequestID = request
			req.ReplayOnly = replay
			return r.queuePrivateGateway(actor, p.Binding.ProjectID, req)
		case "workspace-observation":
			req := workspaceReq
			req.RequestID = request
			req.ReplayOnly = replay
			return r.queuePrivateWorkspaceObservation(actor, p.Binding.ProjectID, req)
		default:
			req := observationReq
			req.RequestID = request
			req.ReplayOnly = replay
			return r.queuePrivateGatewayObservation(actor, p.Binding.ProjectID, req)
		}
	}
	return r, b, admit
}
func lifecycleOpened(t *testing.T, b *queuedPrivateLifecycleBackend, kind string) {
	t.Helper()
	select {
	case got := <-b.opened:
		if got != kind {
			t.Fatal(got, kind)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("private lifecycle worker did not open")
	}
}

func TestPrivateLifecycleQueueSharesGateAndReplaysWithoutBackend(t *testing.T) {
	for _, kind := range []string{"gateway", "workspace-observation", "gateway-observation", "member-launch"} {
		t.Run(kind, func(t *testing.T) {
			r, b, admit := privateLifecycleQueueFixture(t, kind)
			r.workGate <- struct{}{}
			op, hit, err := admit("queue-actor", "queue-request", false)
			g02Check(t, err)
			if hit || op.Status != "queued" {
				t.Fatal(op, hit)
			}
			before := b.inspected.Load()
			replay, hit, err := admit("queue-actor", "queue-request", false)
			g02Check(t, err)
			if !hit || replay.ID != op.ID || b.inspected.Load() != before {
				t.Fatal("replay repeated input inspection")
			}
			select {
			case <-b.opened:
				t.Fatal("worker bypassed shared gate")
			default:
			}
			if response := rc11Fence(r, "acquire", strings.Repeat("a", 64)); response.Code != 409 {
				t.Fatal("queued worker did not fence management")
			}
			<-r.workGate
			lifecycleOpened(t, b, kind)
			if response := rc11Fence(r, "acquire", strings.Repeat("a", 64)); response.Code != 409 {
				t.Fatal("active worker did not fence management")
			}
			close(b.release)
			waitProjectWorker(t, r)
			actual, err := loadOperation(r.s.db, op.ID)
			g02Check(t, err)
			if actual.Status != "succeeded" || b.closed.Load() != 1 || r.failed.Load() {
				t.Fatal("worker outcome", actual.Status, b.closed.Load(), r.failed.Load())
			}
			r.b = &fakeBackend{}
			replay, hit, err = admit("queue-actor", "queue-request", true)
			g02Check(t, err)
			if !hit || replay.ID != op.ID || replay.Status != "succeeded" || b.inspected.Load() != before {
				t.Fatal("receipt required unavailable backend")
			}
			_, _, err = admit("different-actor", "queue-request", true)
			requireFault(t, err, "replay_not_found")
			if response := rc11Fence(r, "acquire", strings.Repeat("a", 64)); response.Code != 200 {
				t.Fatal("worker count leaked")
			}
		})
	}
}
func TestPrivateLifecycleQueueFailureAndCancellationPreserveIntent(t *testing.T) {
	for _, kind := range []string{"gateway", "workspace-observation", "gateway-observation", "member-launch"} {
		for _, failure := range []string{"waiting", "setup", "nil", "typed-nil", "setup-with-driver", "cancel-open", "driver", "close"} {
			t.Run(kind+"/"+failure, func(t *testing.T) {
				r, b, admit := privateLifecycleQueueFixture(t, kind)
				b.fail = failure
				if failure == "waiting" {
					r.workGate <- struct{}{}
				}
				op, _, err := admit("queue-actor", "queue-request", false)
				g02Check(t, err)
				if failure == "waiting" {
					r.cancel()
				} else {
					lifecycleOpened(t, b, kind)
					if failure == "cancel-open" {
						r.cancel()
					} else {
						close(b.release)
					}
				}
				waitProjectWorker(t, r)
				actual, err := loadOperation(r.s.db, op.ID)
				g02Check(t, err)
				want := "interrupted"
				if failure == "close" {
					want = "succeeded"
				}
				closes := int32(0)
				if failure == "driver" || failure == "close" || failure == "setup-with-driver" {
					closes = 1
				}
				if actual.Status != want || b.closed.Load() != closes || r.failed.Load() {
					t.Fatal("lost outcome/handle", actual.Status, b.closed.Load(), r.failed.Load())
				}
				_, err = projectReservations(r.s.db)
				g02Check(t, err)
			})
		}
	}
}
func TestPrivateLifecycleQueueRefusesBeforeBackendOrNewIntent(t *testing.T) {
	for _, kind := range []string{"gateway", "workspace-observation", "gateway-observation", "member-launch"} {
		for _, failure := range []string{"closed", "failed", "canceled", "fenced", "no-backend", "lookup-miss", "bad-request", "bad-actor"} {
			t.Run(kind+"/"+failure, func(t *testing.T) {
				r, b, admit := privateLifecycleQueueFixture(t, kind)
				var before int
				g02Check(t, r.s.db.QueryRow("SELECT count(*) FROM operations").Scan(&before))
				actor, request, replay := "queue-actor", "queue-request", false
				switch failure {
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
					replay = true
				case "bad-request":
					request = "/invalid"
				case "bad-actor":
					actor = ""
				}
				if _, _, err := admit(actor, request, replay); err == nil {
					t.Fatal("invalid queue admitted")
				}
				var after int
				g02Check(t, r.s.db.QueryRow("SELECT count(*) FROM operations").Scan(&after))
				if after != before || b.inspected.Load() != 0 {
					t.Fatal("refusal read backend or created intent")
				}
				select {
				case <-b.opened:
					t.Fatal("refusal opened worker")
				default:
				}
			})
		}
	}
}
func TestPrivateLifecycleQueueRejectsChangedReceiptBeforeOpening(t *testing.T) {
	for _, kind := range []string{"gateway", "workspace-observation", "gateway-observation", "member-launch"} {
		t.Run(kind, func(t *testing.T) {
			r, b, admit := privateLifecycleQueueFixture(t, kind)
			r.workGate <- struct{}{}
			op, _, err := admit("queue-actor", "queue-request", false)
			g02Check(t, err)
			// Corruption must stop before an adapter is acquired; it also prevents
			// saving an interruption receipt, so the runner fails further admission.
			_, err = r.s.db.Exec("UPDATE operations SET runtime='{}' WHERE id=?", op.ID)
			g02Check(t, err)
			close(b.release)
			<-r.workGate
			waitProjectWorker(t, r)
			select {
			case <-b.opened:
				t.Fatal("corrupt receipt reached factory")
			default:
			}
			if !r.failed.Load() || b.closed.Load() != 0 {
				t.Fatal("corrupt state was not fenced")
			}
		})
	}
}
func TestPrivateLifecycleWorkspaceObserverHidesCreationAndRecoversLostLayout(t *testing.T) {
	s := testStore(t)
	p, req := privateWorkspaceFixture(t, s)
	directory := filepath.Join(s.directory, "private-projects")
	g02Check(t, os.Mkdir(directory, 0700))
	parent, err := privatechain.InspectWorkspaceParent(context.Background(), directory)
	g02Check(t, err)
	prepared, _, err := s.acceptPrivateWorkspace("owner", p.Binding.ProjectID, req, parent)
	g02Check(t, err)
	w, err := loadPrivateWorkspace(s.db, p.Binding.ProjectID)
	g02Check(t, err)
	fs, err := privatechain.OpenWorkspaceFilesystem(context.Background(), directory, parent, p.Plan, w)
	g02Check(t, err)
	if err = s.runPrivateWorkspace(context.Background(), p.Binding.ProjectID, prepared.ID, lostLayoutAcknowledgement{fs}); err == nil {
		t.Fatal("missing simulated lost layout reply")
	}
	g02Check(t, fs.Close())
	w, err = loadPrivateWorkspace(s.db, p.Binding.ProjectID)
	g02Check(t, err)
	before, err := json.Marshal(w)
	g02Check(t, err)
	source, err := loadOperation(s.db, prepared.ID)
	g02Check(t, err)
	originalOp, err := json.Marshal(source)
	g02Check(t, err)
	observer, err := (realBackend{}).OpenPrivateWorkspaceObserver(context.Background(), directory, p.Plan, w, "observation-receipt")
	g02Check(t, err)
	if _, ok := observer.(privatechain.WorkspaceDriver); ok {
		t.Fatal("observation factory exposed directory creation")
	}
	observed, err := observer.ObserveLayout(context.Background(), w)
	g02Check(t, err)
	if !observed.Complete || !observed.Root.SameObject(w.Root) {
		t.Fatal("read-only factory lost directory ownership")
	}
	g02Check(t, observer.Close())
	if _, err = observer.ObserveLayout(context.Background(), w); err == nil {
		t.Fatal("closed observer retained filesystem authority")
	}
	ctx, cancel := context.WithCancel(context.Background())
	r := &Server{s: s, b: realBackend{}, ctx: ctx, cancel: cancel, workGate: make(chan struct{}, 1)}
	t.Cleanup(func() { cancel(); r.wg.Wait() })
	recovery := privateWorkspaceObservationRequest{RequestID: "recover-real-layout", ExpectedRevision: 1, ReviewDigest: w.PlanDigest, SourceOperationID: w.OperationID, SourceDigest: privateWorkspaceSourceDigest(w)}
	op, hit, err := r.queuePrivateWorkspaceObservation("observer", p.Binding.ProjectID, recovery)
	g02Check(t, err)
	if hit {
		t.Fatal("new real recovery replayed")
	}
	waitProjectWorker(t, r)
	ready, evidence, err := readyPrivateWorkspaceEvidence(s.db, p.Binding.ProjectID)
	g02Check(t, err)
	if ready.Phase != "observed" || evidence.ID != op.ID || evidence.Status != "succeeded" {
		t.Fatal("real queue recovery failed")
	}
	afterWorkspace, err := loadPrivateWorkspace(s.db, p.Binding.ProjectID)
	g02Check(t, err)
	after, err := json.Marshal(afterWorkspace)
	g02Check(t, err)
	actual, err := loadOperation(s.db, prepared.ID)
	g02Check(t, err)
	actualOp, err := json.Marshal(actual)
	g02Check(t, err)
	if !bytes.Equal(before, after) || !bytes.Equal(originalOp, actualOp) {
		t.Fatal("queued observation rewrote original preparation")
	}
	for _, mount := range ready.Mounts {
		entries, err := os.ReadDir(filepath.Join(s.directory, mount.Path))
		g02Check(t, err)
		if len(entries) != 0 {
			t.Fatal("observer created pier or runtime state")
		}
	}
}

func TestPrivateLifecycleQueueRefusesRetainedMaintenanceBeforeFactory(t *testing.T) {
	for _, kind := range []string{"gateway", "workspace-observation", "gateway-observation", "member-launch"} {
		t.Run(kind, func(t *testing.T) {
			r, b, admit := privateLifecycleQueueFixture(t, kind)
			_, _, err := r.s.register("actor", registration("legacy", "legacy", 18580))
			g02Check(t, err)
			v, err := loadInstance(r.s.db, "legacy")
			g02Check(t, err)
			v.Runtime.Runtime.Containers = append(v.Runtime.Runtime.Containers, runtimequalify.ContainerRecord{Kind: "pack", CreateUncertain: true})
			tx, err := r.s.db.Begin()
			g02Check(t, err)
			g02Check(t, saveInstance(tx, v))
			g02Check(t, tx.Commit())
			close(b.release)
			op, _, err := admit("queue-actor", "queue-request", false)
			g02Check(t, err)
			waitProjectWorker(t, r)
			actual, err := loadOperation(r.s.db, op.ID)
			g02Check(t, err)
			if actual.Status != "interrupted" || r.failed.Load() {
				t.Fatal("maintenance did not preserve interrupted intent")
			}
			select {
			case <-b.opened:
				t.Fatal("retained maintenance reached factory")
			default:
			}
			replay, hit, err := admit("queue-actor", "queue-request", false)
			g02Check(t, err)
			if !hit || replay.ID != op.ID || replay.Status != "interrupted" {
				t.Fatal("uncertain worker restarted")
			}
		})
	}
}

type queuedMemberDriver struct {
	*capture46StoreDriver
	backend *queuedPrivateLifecycleBackend
}

func (d *queuedMemberDriver) Close() error { return d.backend.close() }
func (b *queuedPrivateLifecycleBackend) PrivateMemberLaunchContract(directory string, p privatechain.Plan, w privatechain.Workspace, g privatechain.Gateway, instance, runtime, operation string) (string, error) {
	b.inspected.Add(1)
	if directory != b.s.directory {
		b.t.Fatal("contract got foreign directory")
	}
	return privateMemberStoreContract(p, w, g, instance, runtime, operation)
}
func (b *queuedPrivateLifecycleBackend) OpenPrivateMemberLaunch(ctx context.Context, directory string, p privatechain.Plan, w privatechain.Workspace, s privatechain.MemberLaunch) (privateMemberLaunchDriver, error) {
	if err := b.open(ctx, "member-launch", directory, s.OperationID); err != nil {
		return nil, err
	}
	if s.Validate(p, w) != nil || s.Version != 3 {
		b.t.Fatal("factory lacks valid launch")
	}
	if b.fail == "nil" {
		return nil, nil
	}
	if b.fail == "typed-nil" {
		var d *queuedMemberDriver
		return d, nil
	}
	d := &queuedMemberDriver{capture46StoreDriver: &capture46StoreDriver{&privateMemberStoreDriver{t: b.t, s: b.s}}, backend: b}
	if b.fail == "driver" {
		d.fail = "start"
	}
	if b.fail == "capture-observe" {
		d.fail = "observe"
	}
	if b.fail == "setup-with-driver" {
		return d, errors.New("synthetic partial setup")
	}
	return d, nil
}

//go:build linux && amd64

package managed

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type queuedRecoveryBackend struct {
	*queuedProjectBackend
	id, fault          string
	readinessOpens     int
	transientReadiness bool
	reference          projectAnchorRecoveryJob
	reviewOpened       chan struct{}
	calls              []string
	finished, started  map[string]time.Time
}

func (b *queuedRecoveryBackend) source(s runtimequalify.ProjectAnchorRecoveryReviewSelection) error {
	if !reflect.DeepEqual(s, recoveryRuntimeSource(b.s, b.reference)) {
		return errors.New("queue changed recovery source")
	}
	return nil
}
func (b *queuedRecoveryBackend) OpenProjectAnchorRecoveryReview(ctx context.Context, s runtimequalify.ProjectAnchorRecoveryReviewSelection) (projectnetwork.AnchorRecoveryObserver, error) {
	if err := b.source(s); err != nil {
		return nil, err
	}
	b.calls = append(b.calls, "review")
	b.reviewOpened <- struct{}{}
	select {
	case <-b.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if b.fault == "review" {
		return nil, errors.New("synthetic review failure")
	}
	return &queuedRecoveryDriver{b: b}, nil
}
func (b *queuedRecoveryBackend) OpenProjectAnchorDrain(_ context.Context, s runtimequalify.ProjectAnchorDrainSelection) (projectnetwork.AnchorDrainDriver, error) {
	if err := b.source(s.Source); err != nil {
		return nil, err
	}
	for i, m := range s.Drain.Members {
		if m.Phase != "exited" {
			return &queuedRecoveryDriver{b: b, index: i}, nil
		}
	}
	return nil, errors.New("no drain target")
}
func (b *queuedRecoveryBackend) OpenProjectAnchorNamespace(_ context.Context, s runtimequalify.ProjectAnchorNamespaceSelection) (projectnetwork.AnchorNamespaceDriver, error) {
	return &queuedRecoveryDriver{b: b}, b.source(s.Source)
}
func (b *queuedRecoveryBackend) OpenProjectAnchorRetirement(_ context.Context, s runtimequalify.ProjectAnchorRetirementSelection) (projectnetwork.AnchorRetirementDriver, error) {
	if err := b.source(s.Source); err != nil {
		return nil, err
	}
	for i, m := range s.Retirement.Members {
		if m.Phase != "absent" {
			return &queuedRecoveryDriver{b: b, index: i}, nil
		}
	}
	return nil, errors.New("no retirement target")
}
func (b *queuedRecoveryBackend) PlanProjectAnchorReplacement(_ context.Context, s runtimequalify.ProjectAnchorReplacementPlanSelection) (projectnetwork.AnchorReplacement, error) {
	if err := b.source(s.Source); err != nil {
		return projectnetwork.AnchorReplacement{}, err
	}
	var members []projectnetwork.RecoveredMember
	for i, id := range s.Members {
		c, err := projectnetwork.SelectRecoveryMemberCreation(b.reference.Source, s.Retirement, id.InstanceID, id.OperationID)
		if err != nil {
			return projectnetwork.AnchorReplacement{}, err
		}
		members = append(members, projectnetwork.RecoveredMember{Creation: c, Container: projectnetwork.ResourceIdentity{ContractDigest: b.reference.Journal.Readiness.Replacement.Members[i].Container.ContractDigest}, Phase: "planned"})
	}
	return projectnetwork.NewAnchorReplacement(b.reference.Source, s.Retirement, s.OperationID, members)
}
func (b *queuedRecoveryBackend) OpenProjectAnchorReplacement(_ context.Context, s runtimequalify.ProjectAnchorReplacementSelection) (projectnetwork.AnchorReplacementDriver, error) {
	if err := b.source(s.Source); err != nil {
		return nil, err
	}
	for i, m := range s.Replacement.Members {
		if m.Phase != "running" {
			return &queuedRecoveryDriver{b: b, index: i}, nil
		}
	}
	return nil, errors.New("no replacement target")
}
func (b *queuedRecoveryBackend) OpenProjectAnchorRecoveryReadiness(_ context.Context, s runtimequalify.ProjectAnchorRecoveryReadinessSelection) (projectnetwork.AnchorRecoveryReadinessDriver, error) {
	b.readinessOpens++
	if b.fault == "readiness" {
		return nil, context.Canceled
	}
	if b.transientReadiness && b.readinessOpens == 1 {
		return nil, errors.New("synthetic listener not ready")
	}
	return &queuedRecoveryDriver{b: b}, b.source(s.Source)
}

type queuedRecoveryDriver struct {
	b     *queuedRecoveryBackend
	index int
}

func (d *queuedRecoveryDriver) VerifyEngine(ctx context.Context, id string) error {
	if id != d.b.reference.Source.Launches[0].Anchor.Prepared.EngineID {
		return errors.New("wrong engine")
	}
	return ctx.Err()
}
func (d *queuedRecoveryDriver) InspectAnchorRecovery(_ context.Context, _ projectnetwork.AnchorRecoveryContext) (projectnetwork.AnchorRecoveryObservation, error) {
	o := recoveryTestClone(d.b.t, d.b.reference.Journal.Readiness.Replacement.Retirement.Namespace.Drain.Review.After)
	o.CapturedAt = time.Now().UTC()
	o.Runtime.ObservedAt = o.CapturedAt
	return o, nil
}
func (d *queuedRecoveryDriver) drainObservation(o projectnetwork.AnchorDrainObservation) projectnetwork.AnchorDrainObservation {
	o = recoveryTestClone(d.b.t, o)
	o.CapturedAt = time.Now().UTC()
	o.Runtime.ObservedAt = o.CapturedAt
	for i := range o.Processes {
		if v := d.b.finished[o.Processes[i].InstanceID]; !v.IsZero() {
			o.Processes[i].Process.FinishedAt = v
		}
	}
	return o
}
func (d *queuedRecoveryDriver) InspectDrain(_ context.Context, _ projectnetwork.AnchorDrain) (projectnetwork.AnchorDrainObservation, error) {
	r := d.b.reference.Journal.Readiness.Replacement.Retirement.Namespace.Drain
	return d.drainObservation(*r.Members[d.index].Before), nil
}
func (d *queuedRecoveryDriver) SignalDrainMember(ctx context.Context, s projectnetwork.AnchorDrain, i int) error {
	j, err := loadProjectAnchorRecovery(d.b.s.db, d.b.id)
	if err != nil {
		return err
	}
	if j.Journal.Drain == nil || !reflect.DeepEqual(*j.Journal.Drain, s) || i != d.index || s.Members[i].Phase != "stop-dispatched" {
		return errors.New("signal before durable dispatch")
	}
	d.b.calls = append(d.b.calls, "signal")
	d.b.finished[s.Members[i].InstanceID] = time.Now().UTC()
	if d.b.fault == "signal" {
		return errors.New("synthetic lost signal reply")
	}
	return ctx.Err()
}
func (d *queuedRecoveryDriver) AwaitDrainExit(_ context.Context, _ projectnetwork.AnchorDrain, i int) (projectnetwork.AnchorDrainObservation, error) {
	r := d.b.reference.Journal.Readiness.Replacement.Retirement.Namespace.Drain
	return d.drainObservation(*r.Members[i].After), nil
}
func (d *queuedRecoveryDriver) InspectNamespace(_ context.Context, s projectnetwork.AnchorNamespace) (projectnetwork.AnchorDrainObservation, error) {
	r := d.b.reference.Journal.Readiness.Replacement.Retirement.Namespace
	if s.Phase == "planned" {
		return d.drainObservation(*r.Before), nil
	}
	return d.drainObservation(*r.After), nil
}
func (d *queuedRecoveryDriver) StartRecoveryAnchor(context.Context, projectnetwork.AnchorNamespace) error {
	return errors.New("restarted anchor must not receive another Start")
}
func (d *queuedRecoveryDriver) retirementObservation(o projectnetwork.AnchorRetirementObservation, replaced bool) projectnetwork.AnchorRetirementObservation {
	o = recoveryTestClone(d.b.t, o)
	o.CapturedAt = time.Now().UTC()
	o.Runtime.ObservedAt = o.CapturedAt
	for i := range o.Processes {
		p := &o.Processes[i]
		if replaced {
			if v := d.b.started[p.InstanceID]; !v.IsZero() {
				p.Process.StartedAt = v
			}
		} else if v := d.b.finished[p.InstanceID]; !v.IsZero() {
			p.Process.FinishedAt = v
		}
	}
	return o
}
func (d *queuedRecoveryDriver) InspectRetirement(_ context.Context, s projectnetwork.AnchorRetirement) (projectnetwork.AnchorRetirementObservation, error) {
	r := d.b.reference.Journal.Readiness.Replacement.Retirement.Members[d.index]
	if s.Members[d.index].Phase == "planned" {
		return d.retirementObservation(*r.Before, false), nil
	}
	return d.retirementObservation(*r.After, false), nil
}
func (d *queuedRecoveryDriver) RemoveRetiredMember(ctx context.Context, s projectnetwork.AnchorRetirement, i int) error {
	j, err := loadProjectAnchorRecovery(d.b.s.db, d.b.id)
	if err != nil {
		return err
	}
	if j.Journal.Retirement == nil || !reflect.DeepEqual(*j.Journal.Retirement, s) || i != d.index || s.Members[i].Phase != "remove-dispatched" {
		return errors.New("removal before durable dispatch")
	}
	d.b.calls = append(d.b.calls, "remove")
	if d.b.fault == "remove" {
		return errors.New("synthetic lost removal reply")
	}
	return ctx.Err()
}
func (d *queuedRecoveryDriver) InspectReplacement(_ context.Context, _ projectnetwork.AnchorReplacement, i int, stage string) (projectnetwork.AnchorRetirementObservation, error) {
	m := d.b.reference.Journal.Readiness.Replacement.Members[i]
	var o *projectnetwork.AnchorRetirementObservation
	switch stage {
	case "absent":
		o = m.Before
	case "created":
		o = m.Created
	case "running":
		o = m.After
	default:
		return projectnetwork.AnchorRetirementObservation{}, errors.New("unexpected replacement stage")
	}
	return d.retirementObservation(*o, true), nil
}
func (d *queuedRecoveryDriver) CreateReplacement(ctx context.Context, s projectnetwork.AnchorReplacement, i int) (string, error) {
	j, err := loadProjectAnchorRecovery(d.b.s.db, d.b.id)
	if err != nil {
		return "", err
	}
	if j.Journal.Replacement == nil || !reflect.DeepEqual(*j.Journal.Replacement, s) || i != d.index || s.Members[i].Phase != "create-dispatched" {
		return "", errors.New("Create before durable dispatch")
	}
	d.b.calls = append(d.b.calls, "create")
	if d.b.fault == "create-unknown" {
		return "", errors.New("synthetic lost Create reply without ID")
	}
	id := d.b.reference.Journal.Readiness.Replacement.Members[i].Container.ID
	if d.b.fault == "create" {
		return id, errors.New("synthetic error with returned ID")
	}
	return id, ctx.Err()
}
func (d *queuedRecoveryDriver) StartReplacement(ctx context.Context, s projectnetwork.AnchorReplacement, i int) error {
	j, err := loadProjectAnchorRecovery(d.b.s.db, d.b.id)
	if err != nil {
		return err
	}
	if j.Journal.Replacement == nil || !reflect.DeepEqual(*j.Journal.Replacement, s) || i != d.index || s.Members[i].Phase != "start-dispatched" {
		return errors.New("Start before durable dispatch")
	}
	d.b.calls = append(d.b.calls, "start")
	d.b.started[s.Members[i].Creation.InstanceID] = time.Now().UTC()
	if d.b.fault == "start" {
		return errors.New("synthetic lost Start reply")
	}
	return ctx.Err()
}
func (d *queuedRecoveryDriver) InspectRecoveryReadiness(context.Context, projectnetwork.AnchorReplacement) (projectnetwork.AnchorRetirementObservation, error) {
	if d.b.fault == "readiness" {
		return projectnetwork.AnchorRetirementObservation{}, errors.New("synthetic readiness failure")
	}
	return d.retirementObservation(d.b.reference.Journal.Readiness.After, true), nil
}
func (d *queuedRecoveryDriver) ReadHostIdentity(ctx context.Context, endpoint string) (string, error) {
	return (&recoveryStoreReadiness{h: d.b.reference.Source}).ReadHostIdentity(ctx, endpoint)
}

func recoveryQueueFixture(t *testing.T) (*Server, *queuedRecoveryBackend, storedProject, projectAnchorRecoveryRequest) {
	t.Helper()
	r, base, _, _ := projectQueueFixture(t)
	p, req := recoveryStoreProject(t, r.s)
	b := &queuedRecoveryBackend{queuedProjectBackend: base, reviewOpened: make(chan struct{}, 2), finished: map[string]time.Time{}, started: map[string]time.Time{}}
	r.b = b
	return r, b, p, req
}
func prepareQueuedRecoveryReference(t *testing.T, r *Server, b *queuedRecoveryBackend, id string) {
	t.Helper()
	b.id = id
	dir := privateDir(t)
	path := filepath.Join(dir, "runner.sqlite")
	_, err := r.s.db.Exec("VACUUM INTO ?", path)
	g02Check(t, err)
	g02Check(t, os.Chmod(path, 0600))
	reference, err := testOpenStore(dir)
	g02Check(t, err)
	defer reference.close()
	j, err := loadProjectAnchorRecovery(reference.db, id)
	g02Check(t, err)
	b.reference = completeRecoveryStoreFixture(t, reference, j)
}
func waitRecoveryQueue(t *testing.T, r *Server) {
	t.Helper()
	done := make(chan struct{})
	go func() { r.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("recovery queue did not finish")
	}
}
func TestAnchorRecoveryQueueSharesAdmissionAndCompletes(t *testing.T) {
	r, b, p, req := recoveryQueueFixture(t)
	r.workGate <- struct{}{}
	op, hit, err := r.queueProjectAnchorRecovery("actor", p.ID, req)
	g02Check(t, err)
	if hit {
		t.Fatal("unexpected replay")
	}
	prepareQueuedRecoveryReference(t, r, b, op.ID)
	again, hit, err := r.queueProjectAnchorRecovery("actor", p.ID, req)
	g02Check(t, err)
	if !hit || again.ID != op.ID {
		t.Fatal("queue replay changed identity")
	}
	select {
	case <-b.reviewOpened:
		t.Fatal("factory crossed work gate")
	default:
	}
	if w := rc11Fence(r, "acquire", strings.Repeat("a", 64)); w.Code != 409 {
		t.Fatal("update crossed worker admission", w.Code)
	}
	<-r.workGate
	select {
	case <-b.reviewOpened:
	case <-time.After(5 * time.Second):
		t.Fatal("recovery review not reached")
	}
	close(b.release)
	waitRecoveryQueue(t, r)
	actual, err := loadOperation(r.s.db, op.ID)
	g02Check(t, err)
	if actual.Status != "succeeded" {
		t.Fatal("recovery did not complete", actual.Status, actual.Message, b.calls)
	}
	g02Check(t, validateProjectAnchorRecoveries(r.s.db))
	expected := []string{"review", "signal", "signal", "remove", "remove", "create", "start", "create", "start"}
	if !reflect.DeepEqual(expected, b.calls) {
		t.Fatal("unexpected recovery effects", b.calls)
	}
	replay, hit, err := r.queueProjectAnchorRecovery("actor", p.ID, req)
	g02Check(t, err)
	if !hit || replay.Status != "succeeded" {
		t.Fatal("completed replay scheduled work")
	}
	waitRecoveryQueue(t, r)
	if !reflect.DeepEqual(expected, b.calls) {
		t.Fatal("effects replayed")
	}
	if w := rc11Fence(r, "acquire", strings.Repeat("a", 64)); w.Code != 200 {
		t.Fatal("completed worker retained admission", w.Code)
	}
	req.RequestID = "while-quiesced"
	_, _, err = r.queueProjectAnchorRecovery("actor", p.ID, req)
	requireFault(t, err, "management_quiesced")
}
func TestAnchorRecoveryQueueKeepsUncertainEffects(t *testing.T) {
	for _, fault := range []string{"review", "signal", "remove", "create", "start", "readiness"} {
		t.Run(fault, func(t *testing.T) {
			r, b, p, req := recoveryQueueFixture(t)
			r.workGate <- struct{}{}
			b.fault = fault
			op, _, err := r.queueProjectAnchorRecovery("actor", p.ID, req)
			g02Check(t, err)
			prepareQueuedRecoveryReference(t, r, b, op.ID)
			close(b.release)
			<-r.workGate
			waitRecoveryQueue(t, r)
			j, err := loadProjectAnchorRecovery(r.s.db, op.ID)
			g02Check(t, err)
			actual, err := loadOperation(r.s.db, op.ID)
			g02Check(t, err)
			want := "interrupted"
			if fault == "review" {
				want = "failed"
			}
			if actual.Status != want {
				t.Fatal("wrong failure status", actual.Status, b.calls)
			}
			if fault != "review" && projectRetirementClear(r.s.db, p.ID, "") == nil {
				t.Fatal("uncertainty released project fence")
			}
			if fault == "create" && (j.Journal.Replacement == nil || j.Journal.Replacement.Members[0].Container.ID == "") {
				t.Fatal("returned Create ID discarded")
			}
			calls := append([]string(nil), b.calls...)
			g02Check(t, r.s.reconcileInterrupted())
			replay, hit, err := r.queueProjectAnchorRecovery("actor", p.ID, req)
			g02Check(t, err)
			if !hit || replay.Status != want {
				t.Fatal("failure replay changed")
			}
			waitRecoveryQueue(t, r)
			if !reflect.DeepEqual(calls, b.calls) {
				t.Fatal("uncertain effect replayed")
			}
			saved, err := loadProjectAnchorRecovery(r.s.db, op.ID)
			g02Check(t, err)
			if !reflect.DeepEqual(saved, j) {
				t.Fatal("reconciliation changed evidence")
			}
			g02Check(t, validateProjectAnchorRecoveries(r.s.db))
		})
	}
}

func TestAnchorRecoveryQueueReadinessWaitDoesNotRepeatEffects(t *testing.T) {
	r, b, p, req := recoveryQueueFixture(t)
	r.workGate <- struct{}{}
	op, _, err := r.queueProjectAnchorRecovery("actor", p.ID, req)
	if err != nil {
		t.Fatal(err)
	}
	prepareQueuedRecoveryReference(t, r, b, op.ID)
	b.transientReadiness = true
	close(b.release)
	<-r.workGate
	waitRecoveryQueue(t, r)
	got, err := loadOperation(r.s.db, op.ID)
	if err != nil || got.Status != "succeeded" {
		t.Fatal(got, err)
	}
	if b.readinessOpens != 2 {
		t.Fatal("readiness did not use a fresh adapter", b.readinessOpens)
	}
	counts := map[string]int{}
	for _, call := range b.calls {
		counts[call]++
	}
	for _, call := range []string{"signal", "remove", "create", "start"} {
		if counts[call] != len(p.Members) {
			t.Fatal("readiness repeated runtime effects", b.calls)
		}
	}
}

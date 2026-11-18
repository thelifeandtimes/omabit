package grubbery_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

func hashValue(v any) string {
	raw, _ := json.Marshal(v)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func copyRecord(r grubbery.Record) grubbery.Record {
	raw, _ := json.Marshal(r)
	var out grubbery.Record
	json.Unmarshal(raw, &out)
	return out
}

// This models transaction/CAS semantics, not persistent/fsync qualification.
type memoryJournal struct {
	mu        sync.Mutex
	records   map[string]grubbery.Record
	leases    map[string]string
	loseReply string
}

func newJournal() *memoryJournal {
	return &memoryJournal{records: map[string]grubbery.Record{}, leases: map[string]string{}}
}
func (m *memoryJournal) Load(_ context.Context, id string) (grubbery.Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.records[id]
	if !ok {
		return grubbery.Record{}, grubbery.ErrMissing
	}
	return copyRecord(r), nil
}
func (m *memoryJournal) Create(_ context.Context, r grubbery.Record) (grubbery.Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r.Validate() != nil {
		return grubbery.Record{}, grubbery.ErrHistory
	}
	id := r.Plan.RequestID
	if _, ok := m.records[id]; ok {
		return grubbery.Record{}, grubbery.ErrConflict
	}
	n, _ := r.Plan.Binding.Namespace()
	if m.leases[n.DeskRoot] != "" {
		return grubbery.Record{}, grubbery.ErrNamespaceBusy
	}
	m.records[id] = copyRecord(r)
	m.leases[n.DeskRoot] = id
	return copyRecord(r), nil
}
func (m *memoryJournal) Append(_ context.Context, id string, revision uint64, event grubbery.Event) (grubbery.Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.records[id]
	if !ok {
		return grubbery.Record{}, grubbery.ErrMissing
	}
	if r.Revision != revision {
		return grubbery.Record{}, grubbery.ErrCAS
	}
	next := copyRecord(r)
	next.Events = append(next.Events, event)
	next.Revision++
	if next.Validate() != nil {
		return grubbery.Record{}, grubbery.ErrHistory
	}
	m.records[id] = copyRecord(next)
	if done, _ := next.Terminal(); done {
		n, _ := r.Plan.Binding.Namespace()
		if m.leases[n.DeskRoot] == id {
			delete(m.leases, n.DeskRoot)
		}
	}
	if m.loseReply == event.State {
		m.loseReply = ""
		return grubbery.Record{}, errors.New("injected committed journal reply loss")
	}
	return copyRecord(next), nil
}

type phaseBackend struct {
	journal                              *memoryJournal
	files                                []deskwork.SourceFile
	mu                                   sync.Mutex
	applied                              map[string]bool
	admissions, dispatches, observations atomic.Int32
	deny, lost, hide, wrong              atomic.Bool
	before                               func()
}

func (b *phaseBackend) Admit(_ context.Context, p grubbery.Plan, s grubbery.Step) (grubbery.Review, error) {
	b.admissions.Add(1)
	if b.before != nil {
		b.before()
	}
	if b.deny.Load() {
		return grubbery.Review{}, errors.New("fixture policy blocks")
	}
	return grubbery.Review{Version: 1, PlanDigest: p.Digest, StepID: s.ID, BindingDigest: hashValue(p.Binding), PolicyRevision: 3, NativeEvidenceDigest: hashValue("before:" + s.ID)}, nil
}
func (b *phaseBackend) Dispatch(ctx context.Context, p grubbery.Plan, s grubbery.Step, q grubbery.Request) (string, error) {
	r, err := b.journal.Load(ctx, p.RequestID)
	if err != nil {
		return "", err
	}
	state, err := r.State(s.ID)
	if err != nil || state != "dispatched" {
		return "", errors.New("native call happened before durable dispatch")
	}
	expected, err := p.Request(s.ID, b.files)
	if err != nil || !reflect.DeepEqual(q, expected) {
		return "", errors.New("native call changed source/target/phase")
	}
	b.dispatches.Add(1)
	b.mu.Lock()
	b.applied[s.ID] = true
	b.mu.Unlock()
	if b.lost.Load() {
		return "", errors.New("native accepted; reply deliberately lost")
	}
	return hashValue("accepted:" + s.ID), nil
}
func (b *phaseBackend) ObserveAfter(_ context.Context, p grubbery.Plan, s grubbery.Step) (*grubbery.Proof, error) {
	b.observations.Add(1)
	b.mu.Lock()
	applied := b.applied[s.ID]
	b.mu.Unlock()
	if !applied || b.hide.Load() {
		return nil, nil
	}
	proof := &grubbery.Proof{Version: 1, PlanDigest: p.Digest, StepID: s.ID, BindingDigest: hashValue(p.Binding), NativeEvidenceDigest: hashValue("after:" + s.ID)}
	if b.wrong.Load() {
		proof.BindingDigest = strings.Repeat("0", 64)
	}
	return proof, nil
}

func execution(t *testing.T) (grubbery.Execution, grubbery.Call, grubbery.Plan, *memoryJournal, *phaseBackend, *atomic.Int32) {
	t.Helper()
	p, files := plan(t)
	j := newJournal()
	b := &phaseBackend{journal: j, files: files, applied: map[string]bool{}}
	loads := new(atomic.Int32)
	e := grubbery.Execution{Store: j, Backend: b, LoadSource: func(context.Context, string) ([]deskwork.SourceFile, error) { loads.Add(1); return files, nil }, Now: func() time.Time { return time.Date(2026, 11, 18, 1, 0, 0, 0, time.UTC) }}
	c := grubbery.Call{RunnerID: p.Binding.RunnerID, Mode: p.Binding.Mode, Actor: p.Binding.Actor, Target: p.Binding.Target, AppID: p.Binding.AppID, RequestID: p.RequestID, PlanDigest: p.Digest, StepID: p.Steps[0].ID}
	if _, err := e.Begin(context.Background(), c, p); err != nil {
		t.Fatal(err)
	}
	return e, c, p, j, b, loads
}

func TestExactCompletedPhaseReplayAfterReopenNeverStartsNext(t *testing.T) {
	e, c, p, j, b, loads := execution(t)
	ctx := context.Background()
	r, err := e.Advance(ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	if state, _ := r.State(c.StepID); state != "observed" {
		t.Fatal("first phase did not complete")
	}
	before := copyRecord(r)
	calls, reads, sourceReads := b.dispatches.Load(), b.observations.Load(), loads.Load()
	// A new coordinator has only the saved journal. Runtime/source authority is
	// deliberately absent. Replaying the completed original step must work.
	reopened := grubbery.Execution{Store: j}
	for i := 0; i < 3; i++ {
		got, err := reopened.Advance(ctx, c)
		if err != nil || !reflect.DeepEqual(got, before) {
			t.Fatal("completed replay changed progress", err)
		}
	}
	if _, err := reopened.Begin(ctx, c, p); err != nil {
		t.Fatal("matched begin replay needed source/runtime", err)
	}
	if b.dispatches.Load() != calls || b.observations.Load() != reads || loads.Load() != sourceReads {
		t.Fatal("historical replay acquired native/source authority")
	}
	current, done, err := r.Current()
	if err != nil || done || current.ID != p.Steps[1].ID {
		t.Fatal("next saved selection changed")
	}
	c.StepID = current.ID
	if _, err = e.Advance(ctx, c); err != nil {
		t.Fatal(err)
	}
	if b.dispatches.Load() != 2 {
		t.Fatal("explicit next phase did not dispatch exactly once")
	}
}

func TestLostNativeReplyRetainsOriginalAndReconcilesWithoutSourceOrRedispatch(t *testing.T) {
	e, c, _, j, b, loads := execution(t)
	ctx := context.Background()
	b.lost.Store(true)
	r, err := e.Advance(ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	if state, _ := r.State(c.StepID); state != "uncertain" || b.dispatches.Load() != 1 {
		t.Fatal("lost native reply was not retained")
	}
	if len(r.Events) != 2 || r.Events[0].State != "dispatched" || r.Events[1].State != "uncertain" {
		t.Fatal("original uncertainty lost")
	}
	before := copyRecord(r)
	sourceReads := loads.Load()
	admissions := b.admissions.Load()
	observations := b.observations.Load()
	reopened := grubbery.Execution{Store: j, Backend: b}
	if got, err := reopened.Advance(ctx, c); err != nil || !reflect.DeepEqual(got, before) {
		t.Fatal("uncertain advance resubmitted")
	}
	if b.dispatches.Load() != 1 || b.admissions.Load() != admissions || b.observations.Load() != observations {
		t.Fatal("uncertain replay probed or dispatched")
	}
	b.hide.Store(true)
	for i := 0; i < 5; i++ {
		got, err := reopened.Reconcile(ctx, c)
		if err != nil || !reflect.DeepEqual(got, before) {
			t.Fatal("unproved polling grew persistent history")
		}
	}
	b.hide.Store(false)
	b.wrong.Store(true)
	if got, err := reopened.Reconcile(ctx, c); err != nil || !reflect.DeepEqual(got, before) {
		t.Fatal("foreign proof changed progress")
	}
	b.wrong.Store(false)
	resolved, err := reopened.Reconcile(ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	if state, _ := resolved.State(c.StepID); state != "observed" || len(resolved.Events) != 3 || !reflect.DeepEqual(resolved.Events[:2], before.Events) {
		t.Fatal("resolution replaced original dispatch/uncertainty")
	}
	if loads.Load() != sourceReads || b.dispatches.Load() != 1 {
		t.Fatal("read-only resolution needed source or repeated native write")
	}
}

func TestConcurrentPhaseCallsOnlyDispatchTheCASWinner(t *testing.T) {
	e, c, _, _, b, _ := execution(t)
	arrived := new(atomic.Int32)
	barrier := make(chan struct{})
	b.before = func() {
		if arrived.Add(1) == 2 {
			close(barrier)
		}
		<-barrier
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := e.Advance(context.Background(), c); results <- err }()
	}
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	if b.dispatches.Load() != 1 {
		t.Fatal("losing dispatch CAS repeated native effect", b.dispatches.Load())
	}
}

func TestJournalCommitReplyLossAndRefusedPhasesNeverReachNative(t *testing.T) {
	for _, scenario := range []string{"future", "unknown", "actor", "target", "mode", "digest", "policy", "source", "journal"} {
		t.Run(scenario, func(t *testing.T) {
			e, c, p, j, b, _ := execution(t)
			switch scenario {
			case "future":
				c.StepID = p.Steps[2].ID
			case "unknown":
				c.StepID = strings.Repeat("0", 64)
			case "actor":
				c.Actor = "other"
			case "target":
				c.Target.InstanceID = "other"
			case "mode":
				c.Mode = "other"
			case "digest":
				c.PlanDigest = strings.Repeat("0", 64)
			case "policy":
				b.deny.Store(true)
			case "source":
				e.LoadSource = func(context.Context, string) ([]deskwork.SourceFile, error) { return nil, errors.New("cache gone") }
			case "journal":
				j.loseReply = "dispatched"
			}
			if _, err := e.Advance(context.Background(), c); err == nil {
				t.Fatal("unsafe phase accepted", scenario)
			}
			if b.dispatches.Load() != 0 {
				t.Fatal("refused/unconfirmed journal phase called native")
			}
			if scenario == "journal" {
				original := grubbery.Call{RunnerID: p.Binding.RunnerID, Mode: p.Binding.Mode, Actor: p.Binding.Actor, Target: p.Binding.Target, AppID: p.Binding.AppID, RequestID: p.RequestID, PlanDigest: p.Digest, StepID: p.Steps[0].ID}
				r, err := e.Advance(context.Background(), original)
				if err != nil {
					t.Fatal(err)
				}
				if state, _ := r.State(original.StepID); state != "dispatched" || b.dispatches.Load() != 0 {
					t.Fatal("unconfirmed saved dispatch was replayed")
				}
			}
		})
	}
}

func TestAppNamespaceClaimAndBeginIntentCannotBeSilentlyReplaced(t *testing.T) {
	e, c, p, _, b, _ := execution(t)
	ctx := context.Background()
	m, files := source(t, `{"state":"/state"}`)
	other, err := grubbery.NewPlan(p.Binding, p.StageID, strings.Repeat("2", 64), m, files, p.Before)
	if err != nil {
		t.Fatal(err)
	}
	c2 := c
	c2.RequestID = other.RequestID
	c2.PlanDigest = other.Digest
	c2.StepID = other.Steps[0].ID
	e.LoadSource = func(context.Context, string) ([]deskwork.SourceFile, error) { return files, nil }
	if _, err := e.Begin(ctx, c2, other); !errors.Is(err, grubbery.ErrNamespaceBusy) {
		t.Fatal("another operation acquired active app namespace", err)
	}
	if _, err := e.Begin(ctx, c, other); !errors.Is(err, grubbery.ErrConflict) {
		t.Fatal("changed Begin body was silently treated as old request", err)
	}
	if b.dispatches.Load() != 0 {
		t.Fatal("reservation invoked native")
	}
}

func TestNamespaceLeaseReleasesOnlyAfterEveryOriginalPhaseIsObserved(t *testing.T) {
	e, c, p, _, b, _ := execution(t)
	ctx := context.Background()
	for _, step := range p.Steps {
		c.StepID = step.ID
		r, err := e.Advance(ctx, c)
		if err != nil {
			t.Fatal(err)
		}
		if state, _ := r.State(step.ID); state != "observed" {
			t.Fatal("phase incomplete")
		}
	}
	if b.dispatches.Load() != int32(len(p.Steps)) {
		t.Fatal("one native call per phase was not retained")
	}
	r, err := e.Lookup(ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	if _, done, err := r.Current(); err != nil || !done {
		t.Fatal("saved source-following sequence incomplete")
	}
	marker, err := p.Binding.OwnerMarker()
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(marker)
	owner := hex.EncodeToString(sum[:])
	release, _ := p.Binding.ReleaseRoot(p.RequestID)
	before := grubbery.Before{SnapshotDigest: strings.Repeat("f", 64), SourceState: "owned", SourceOwnerSHA256: owner, DeskState: "owned", DeskOwnerSHA256: owner, Epoch: 1, ReleaseRoot: release}
	next, err := grubbery.NewPlan(p.Binding, p.StageID, strings.Repeat("2", 64), p.Source.Manifest, b.files, before)
	if err != nil {
		t.Fatal(err)
	}
	c.RequestID = next.RequestID
	c.PlanDigest = next.Digest
	c.StepID = next.Steps[0].ID
	if _, err = e.Begin(ctx, c, next); err != nil {
		t.Fatal("completed operation retained namespace lease", err)
	}
}

func TestSavedHistoryRejectsReorderingRebindingAndBrokenHashChain(t *testing.T) {
	e, c, _, _, _, _ := execution(t)
	r, err := e.Advance(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"sequence", "previous", "digest", "target", "future-step", "proof", "time", "created", "nil-events", "revision"} {
		t.Run(change, func(t *testing.T) {
			changed := copyRecord(r)
			switch change {
			case "sequence":
				changed.Events[0].Sequence++
			case "previous":
				changed.Events[0].PreviousDigest = strings.Repeat("0", 64)
			case "digest":
				changed.Events[1].Digest = strings.Repeat("0", 64)
			case "target":
				changed.Plan.Binding.Target.InstanceID = "other"
			case "future-step":
				changed.Events[0].StepID = changed.Plan.Steps[2].ID
			case "proof":
				changed.Events[len(changed.Events)-1].Proof.NativeEvidenceDigest = "not-proof"
			case "time":
				changed.Events[0].At = time.Time{}
			case "created":
				changed.CreatedAt = changed.CreatedAt.Add(time.Second)
			case "nil-events":
				changed.Events = nil
			case "revision":
				changed.Revision++
			}
			if changed.Validate() == nil {
				t.Fatal("changed history accepted")
			}
		})
	}
	// A wall-clock correction cannot reverse the causal sequence/CAS chain.
	base, err := grubbery.NewRecord(r.Plan, time.Date(2026, 11, 18, 2, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	first := r.Events[0]
	first.At = time.Date(2026, 11, 18, 1, 0, 0, 0, time.UTC)
	base, err = base.WithEvent(first)
	if err != nil || base.Validate() != nil {
		t.Fatal("clock correction erased a valid causal event", err)
	}
}

func TestMetadataReplaySurvivesMissingCacheButNeverAuthorizesChangedBytes(t *testing.T) {
	p, files := plan(t)
	if p.ValidateMetadata() != nil {
		t.Fatal("attested plan metadata unavailable")
	}
	if p.Validate(nil) == nil {
		t.Fatal("metadata became upload authority")
	}
	clone := copyRecord(grubbery.Record{Plan: p}).Plan
	if clone.ValidateMetadata() != nil || clone.Validate(files) != nil {
		t.Fatal("metadata round trip changed exact plan")
	}
	files[0].Bytes = []byte(`{"other":"/state"}`)
	if clone.ValidateMetadata() != nil || clone.Validate(files) == nil {
		t.Fatal("changed source was authorized by a metadata-only check")
	}
}

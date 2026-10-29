package projectnetwork

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func anchorStartFixture(t *testing.T) (Plan, AnchorStart, Observation) {
	t.Helper()
	plan, err := Build(Request{Label: "Private", ConfirmSharedAdministration: true, Members: []MemberRequest{{InstanceID: "zod", Ship: "~zod", HTTPPort: 18081}, {InstanceID: "nec", Ship: "~nec", HTTPPort: 18082}}}, Reservations{})
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := plan.Digest()
	p := Preparation{ProjectID: "project-example", ProjectRevision: 1, Generation: 1, OperationID: "op-prepare", EngineID: "engine-test", PlanDigest: digest, Sequence: 6, Bridge: Creation{Name: "bridge", ID: strings.Repeat("a", 64), ContractDigest: strings.Repeat("1", 64), Phase: "observed"}, Anchor: Creation{Name: "anchor", ID: strings.Repeat("b", 64), ContractDigest: strings.Repeat("2", 64), Phase: "observed"}}
	s, err := NewAnchorStart(plan, p, "op-start")
	if err != nil {
		t.Fatal(err)
	}
	o := Observation{ProjectID: p.ProjectID, Revision: 1, Generation: 1, EngineID: p.EngineID, ObservedAt: time.Unix(1000, 0), Complete: true, Bridge: ResourceObservation{Resource: ResourceIdentity{p.Bridge.ID, p.Bridge.ContractDigest}, Present: true}, Anchor: AnchorObservation{ResourceObservation: ResourceObservation{Resource: ResourceIdentity{p.Anchor.ID, p.Anchor.ContractDigest}, Present: true}, BridgeID: p.Bridge.ID}}
	for _, m := range plan.Members {
		o.Anchor.Requested = append(o.Anchor.Requested, Publication{m.InternalHTTPPort, plan.BindAddress, m.HTTPPort, "tcp"})
	}
	return plan, s, o
}

type startEngineFixture struct {
	t             *testing.T
	retained      *AnchorStart
	before, after Observation
	calls         []string
	startError    bool
	cancel        context.CancelFunc
}

func (e *startEngineFixture) VerifyEngine(context.Context, string) error {
	e.calls = append(e.calls, "engine")
	return nil
}
func (e *startEngineFixture) InspectAnchor(_ context.Context, p Preparation) (Observation, error) {
	if p != e.retained.Prepared {
		e.t.Fatal("different creation receipt")
	}
	e.calls = append(e.calls, "inspect")
	if e.retained.Phase == "planned" {
		return e.before, nil
	}
	if e.retained.Phase != "started" {
		e.t.Fatal("runtime inspected before acknowledgement commit")
	}
	return e.after, nil
}
func (e *startEngineFixture) StartAnchor(_ context.Context, p Preparation) error {
	e.calls = append(e.calls, "start")
	if e.retained.Phase != "dispatched" || p != e.retained.Prepared {
		e.t.Fatal("start preceded intent or changed ownership")
	}
	if e.cancel != nil {
		e.cancel()
	}
	if e.startError {
		return errors.New("synthetic provider detail must not escape")
	}
	return nil
}
func startEngine(t *testing.T, s *AnchorStart, o Observation) *startEngineFixture {
	after := o
	after.Anchor.Running = true
	after.Anchor.Namespace = NamespaceIdentity{BootID: "01234567-89ab-cdef-0123-456789abcdef", Device: 4, Inode: 55}
	after.BridgeAttachments = []string{s.Prepared.Anchor.ID}
	after.Anchor.Active = append([]Publication(nil), o.Anchor.Requested...)
	return &startEngineFixture{t: t, retained: s, before: o, after: after}
}
func TestAnchorStartWriteAheadBoundariesAndLostReplies(t *testing.T) {
	for _, mode := range []string{"complete", "commit-dispatched", "commit-started", "commit-observed", "lost-start-reply", "cancel-before-start", "cancel-after-ack"} {
		t.Run(mode, func(t *testing.T) {
			plan, initial, o := anchorStartFixture(t)
			retained := initial
			e := startEngine(t, &retained, o)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			e.startError = mode == "lost-start-reply"
			if mode == "cancel-after-ack" {
				e.cancel = cancel
			}
			commit := func(expected, phase string, ns *NamespaceIdentity) error {
				if mode == "commit-"+phase {
					return errors.New("synthetic commit failure")
				}
				next, err := retained.Advance(plan, expected, phase, ns)
				if err != nil {
					return err
				}
				retained = next
				if mode == "cancel-before-start" && phase == "dispatched" {
					cancel()
				}
				return nil
			}
			got, err := ExecuteAnchorStart(ctx, plan, initial, e, commit, func() time.Time { return o.ObservedAt })
			if (err == nil) != (mode == "complete") || !reflect.DeepEqual(got, retained) {
				t.Fatal(got, retained, err)
			}
			want := map[string]string{"complete": "observed", "commit-dispatched": "planned", "commit-started": "dispatched", "commit-observed": "started", "lost-start-reply": "dispatched", "cancel-before-start": "dispatched", "cancel-after-ack": "started"}[mode]
			if retained.Phase != want {
				t.Fatal(retained.Phase, want)
			}
			calls := strings.Join(e.calls, ",")
			count := strings.Count(calls, "start")
			expectedStarts := 1
			if mode == "commit-dispatched" || mode == "cancel-before-start" {
				expectedStarts = 0
			}
			if count != expectedStarts {
				t.Fatal(e.calls)
			}
			if err != nil && strings.Contains(err.Error(), "provider detail") {
				t.Fatal("provider data escaped")
			}
			if retained.Phase != "planned" {
				n := len(e.calls)
				_, err = ExecuteAnchorStart(context.Background(), plan, retained, e, commit, func() time.Time { return o.ObservedAt })
				if err == nil || len(e.calls) != n {
					t.Fatal("replayed retained start")
				}
			}
		})
	}
}
func TestAnchorStartRefusesIncompleteStaleAndForeignObservations(t *testing.T) {
	cases := map[string]func(*Observation){
		"incomplete": func(o *Observation) { o.Complete = false },
		"stale":      func(o *Observation) { o.ObservedAt = o.ObservedAt.Add(-31 * time.Second) },
		"future":     func(o *Observation) { o.ObservedAt = o.ObservedAt.Add(6 * time.Second) },
		"engine":     func(o *Observation) { o.EngineID = "other" },
		"generation": func(o *Observation) { o.Generation++ },
		"resource":   func(o *Observation) { o.Anchor.Resource.ID = strings.Repeat("e", 64) },
		"contract":   func(o *Observation) { o.Bridge.Resource.ContractDigest = strings.Repeat("e", 64) },
		"missing":    func(o *Observation) { o.Anchor.Present = false },
		"bridge":     func(o *Observation) { o.Anchor.BridgeID = strings.Repeat("e", 64) },
		"attachment": func(o *Observation) { o.BridgeAttachments = []string{strings.Repeat("e", 64)} },
		"member":     func(o *Observation) { o.Members = []MemberObservation{{InstanceID: "unexpected"}} },
		"requested":  func(o *Observation) { o.Anchor.Requested = nil },
	}
	for phase := 0; phase < 2; phase++ {
		for name, mutate := range cases {
			t.Run(name+[]string{"-before", "-after"}[phase], func(t *testing.T) {
				plan, initial, o := anchorStartFixture(t)
				retained := initial
				e := startEngine(t, &retained, o)
				if phase == 0 {
					mutate(&e.before)
				} else {
					mutate(&e.after)
				}
				commit := func(expected, next string, ns *NamespaceIdentity) error {
					v, err := retained.Advance(plan, expected, next, ns)
					if err == nil {
						retained = v
					}
					return err
				}
				_, err := ExecuteAnchorStart(context.Background(), plan, initial, e, commit, func() time.Time { return o.ObservedAt })
				if err == nil || retained.Namespace != nil {
					t.Fatal("untrusted observation established readiness")
				}
				want := "planned"
				if phase == 1 {
					want = "started"
				}
				if retained.Phase != want {
					t.Fatal(retained)
				}
			})
		}
	}
}
func TestAnchorStartPhaseAndNamespaceBinding(t *testing.T) {
	plan, s, o := anchorStartFixture(t)
	if _, err := NewAnchorStart(plan, s.Prepared, s.Prepared.OperationID); err == nil {
		t.Fatal("creation receipt reused as start")
	}
	bad := s.Prepared
	bad.Sequence = 5
	bad.Anchor.Phase = "created"
	if _, err := NewAnchorStart(plan, bad, "op-start"); err == nil {
		t.Fatal("unverified resources started")
	}
	if _, err := s.Advance(plan, "planned", "observed", nil); err == nil {
		t.Fatal("phase skipped")
	}
	e := startEngine(t, &s, o)
	for _, running := range []bool{false, true} {
		actual := e.before
		if running {
			actual = e.after
		}
		if err := s.inspect(plan, actual, o.ObservedAt, running); err != nil {
			t.Fatal(err)
		}
		actual.Anchor.Active = []Publication{{InternalPort: 12321, HostAddress: "127.0.0.1", HostPort: 19000, Protocol: "tcp"}}
		if err := s.inspect(plan, actual, o.ObservedAt, running); err == nil {
			t.Fatal("wrong active publications accepted")
		}
	}
}

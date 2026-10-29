package privatechain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"
)

type memberHTTPFixture struct {
	t                         *testing.T
	p                         Plan
	w                         Workspace
	s                         MemberLaunch
	calls                     int
	failAt, cancelAt          int
	cancel                    context.CancelFunc
	wrongIdentity, mutate     bool
	changeBefore, changeAfter func(*MemberHTTPObservation)
	observations              int
}

func newMemberHTTPFixture(t *testing.T) *memberHTTPFixture {
	h, s := newMemberLaunchHarness(t, false)
	running, err := h.execute(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	return &memberHTTPFixture{t: t, p: h.p, w: h.w, s: running}
}
func (f *memberHTTPFixture) step() error {
	f.calls++
	if f.cancelAt == f.calls {
		f.cancel()
	}
	if f.failAt == f.calls {
		return errors.New("PRIVATE detail")
	}
	return nil
}
func (f *memberHTTPFixture) selected(s MemberLaunch) {
	f.t.Helper()
	if !reflect.DeepEqual(s, f.s) {
		f.t.Fatal("driver got different launch")
	}
	if f.mutate {
		s.Gateway.Observation.PublicKeys[0].Auth = "driver changed copy"
		s.Running.Birth.Pier.Inode++
	}
}
func (f *memberHTTPFixture) VerifyHeld(_ context.Context, s MemberLaunch) error {
	f.selected(s)
	return f.step()
}
func (f *memberHTTPFixture) ObserveRuntime(_ context.Context, s MemberLaunch) (MemberHTTPObservation, error) {
	f.selected(s)
	f.observations++
	o := MemberHTTPObservation{Gateway: f.s.Running.Gateway.Clone(), Process: f.s.Running.Birth.After, Pier: f.s.Running.Birth.Pier}
	o.Gateway.ObservedAt = workspaceNow.Add(time.Duration(4+f.observations) * time.Second)
	o.Process.ObservedAt = o.Gateway.ObservedAt
	if f.observations == 1 && f.changeBefore != nil {
		f.changeBefore(&o)
	}
	if f.observations == 2 && f.changeAfter != nil {
		f.changeAfter(&o)
	}
	return o, f.step()
}
func (f *memberHTTPFixture) ReadHostIdentity(_ context.Context, endpoint string) (string, error) {
	m, want, err := memberHTTPSelection(f.p, f.w, f.s)
	if err != nil || endpoint != want {
		f.t.Fatal("unbound endpoint")
	}
	if f.wrongIdentity {
		return "~not-a-member", f.step()
	}
	return m.Ship, f.step()
}
func (f *memberHTTPFixture) collect(ctx context.Context) (MemberHTTPReadiness, error) {
	return CollectMemberHTTPReadiness(ctx, f.p, f.w, f.s, f, func() time.Time { return workspaceNow.Add(7 * time.Second) })
}

func TestMemberHTTPReadinessRetainsLaunchAndClones(t *testing.T) {
	f := newMemberHTTPFixture(t)
	f.mutate = true
	before, _ := json.Marshal(f.s)
	r, err := f.collect(context.Background())
	if err != nil || f.calls != 7 || r.Validate(f.p, f.w, f.s) != nil {
		t.Fatal("complete observation refused", err, f.calls)
	}
	after, _ := json.Marshal(f.s)
	if string(before) != string(after) {
		t.Fatal("driver mutated source launch")
	}
	hash, err := r.Digest(f.p, f.w, f.s)
	if err != nil || len(hash) != 64 {
		t.Fatal("readiness digest missing")
	}
	copy := r.Clone()
	copy.Before.Gateway.PublicKeys[0].Auth = "changed"
	if reflect.DeepEqual(copy, r) {
		t.Fatal("clone aliases")
	}
	raw, _ := json.Marshal(r)
	var decoded MemberHTTPReadiness
	if json.Unmarshal(raw, &decoded) != nil || !reflect.DeepEqual(r, decoded) || decoded.Validate(f.p, f.w, f.s) != nil {
		t.Fatal("historical evidence changed on serialization")
	}
}
func TestMemberHTTPReadinessFailuresAndCancellationReturnNoEvidence(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		for at := 1; at <= 7; at++ {
			t.Run(fmt.Sprintf("cancel-%v-step-%d", cancelled, at), func(t *testing.T) {
				f := newMemberHTTPFixture(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				f.cancel = cancel
				if cancelled {
					f.cancelAt = at
				} else {
					f.failAt = at
				}
				r, err := f.collect(ctx)
				if !errors.Is(err, ErrMemberHTTPReadiness) || !reflect.DeepEqual(r, MemberHTTPReadiness{}) || f.calls != at {
					t.Fatal("failed observation advanced or leaked partial evidence", f.calls, at, err)
				}
			})
		}
	}
}
func TestMemberHTTPReadinessRejectsWrongEndpointIdentity(t *testing.T) {
	f := newMemberHTTPFixture(t)
	f.wrongIdentity = true
	r, err := f.collect(context.Background())
	if err == nil || f.calls != 4 || !reflect.DeepEqual(r, MemberHTTPReadiness{}) {
		t.Fatal("wrong identity admitted")
	}
}
func TestMemberHTTPReadinessRejectsChangedObservations(t *testing.T) {
	changes := map[string]func(*MemberHTTPObservation){
		"pier":            func(o *MemberHTTPObservation) { o.Pier.Inode++ },
		"urb":             func(o *MemberHTTPObservation) { o.Pier.UrbBirth = "different" },
		"container":       func(o *MemberHTTPObservation) { o.Process.ContainerID = "other" },
		"contract":        func(o *MemberHTTPObservation) { o.Process.ContractDigest = "other" },
		"restart":         func(o *MemberHTTPObservation) { o.Process.StartedAt = o.Process.StartedAt.Add(time.Second) },
		"namespace":       func(o *MemberHTTPObservation) { o.Process.Namespace.Inode++ },
		"incomplete":      func(o *MemberHTTPObservation) { o.Process.CompleteContract = false },
		"stopped":         func(o *MemberHTTPObservation) { o.Process.Running = false },
		"process-stale":   func(o *MemberHTTPObservation) { o.Process.ObservedAt = workspaceNow.Add(-time.Minute) },
		"process-future":  func(o *MemberHTTPObservation) { o.Process.ObservedAt = workspaceNow.Add(time.Minute) },
		"gateway-stale":   func(o *MemberHTTPObservation) { o.Gateway.ObservedAt = workspaceNow.Add(-time.Minute) },
		"gateway-changed": func(o *MemberHTTPObservation) { o.Gateway.Head++ },
	}
	for name, change := range changes {
		for _, before := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s-before-%v", name, before), func(t *testing.T) {
				f := newMemberHTTPFixture(t)
				if before {
					f.changeBefore = change
				} else {
					f.changeAfter = change
				}
				r, err := f.collect(context.Background())
				if err == nil || !reflect.DeepEqual(r, MemberHTTPReadiness{}) {
					t.Fatal("changed observation admitted")
				}
				if before && f.calls != 2 {
					t.Fatal("untrusted runtime reached HTTP")
				}
			})
		}
	}
}
func TestMemberHTTPReadinessRejectsReceiptTamperingAndLegacy(t *testing.T) {
	f := newMemberHTTPFixture(t)
	good, err := f.collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*MemberHTTPReadiness){
		"instance": func(r *MemberHTTPReadiness) { r.InstanceID = "other" }, "operation": func(r *MemberHTTPReadiness) { r.LaunchOperationID = "other" }, "digest": func(r *MemberHTTPReadiness) { r.LaunchDigest = "other" }, "ship": func(r *MemberHTTPReadiness) { r.Ship = "~other" }, "endpoint": func(r *MemberHTTPReadiness) { r.Endpoint = "http://remote.invalid/~/host" }, "old-capture": func(r *MemberHTTPReadiness) { r.CapturedAt = r.Before.Process.ObservedAt }, "backwards": func(r *MemberHTTPReadiness) {
			r.After.Process.ObservedAt = r.Before.Process.ObservedAt.Add(-time.Second)
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := good.Clone()
			change(&r)
			if r.Validate(f.p, f.w, f.s) == nil {
				t.Fatal("tampered receipt admitted")
			}
		})
	}
	for _, kind := range []string{"legacy", "no-birth", "not-running", "peer"} {
		t.Run(kind, func(t *testing.T) {
			f := newMemberHTTPFixture(t)
			switch kind {
			case "legacy":
				f.s.Version = 1
				f.s.Running.Birth = nil
			case "no-birth":
				f.s.Running.Birth = nil
			case "not-running":
				f.s.Phase = "started"
				f.s.Running = nil
			case "peer":
				h, s := newMemberLaunchHarness(t, true)
				v, err := h.execute(context.Background(), s)
				if err != nil {
					t.Fatal(err)
				}
				f.p, f.w, f.s = h.p, h.w, v
			}
			r, err := f.collect(context.Background())
			if err == nil || f.calls != 0 || !reflect.DeepEqual(r, MemberHTTPReadiness{}) {
				t.Fatal("unsupported source reached driver")
			}
		})
	}
}

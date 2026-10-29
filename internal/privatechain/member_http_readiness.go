package privatechain

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

var ErrMemberHTTPReadiness = errors.New("private member HTTP identity could not be established from its retained launch")

// A concrete observer must bracket its complete gateway/process observations
// with continuously held pier identities under the common management fence.
// This value cannot capture or adopt a previously unbound pier.
type MemberHTTPObservation struct {
	Gateway GatewayObservation `json:"gateway"`
	Process MemberBirthProcess `json:"process"`
	Pier    pierstore.Identity `json:"pier"`
}

// Historical HTTP identity evidence for the first captured member. It is not
// current mutation authority, Gall readiness, an Eyre session, or a successful
// Stop. No persistence or second-member admission is supplied by this model.
type MemberHTTPReadiness struct {
	Version           int                   `json:"version"`
	InstanceID        string                `json:"instance_id"`
	LaunchOperationID string                `json:"launch_operation_id"`
	LaunchDigest      string                `json:"launch_digest"`
	Ship              string                `json:"ship"`
	Endpoint          string                `json:"endpoint"`
	Before            MemberHTTPObservation `json:"before"`
	After             MemberHTTPObservation `json:"after"`
	CapturedAt        time.Time             `json:"captured_at"`
}

func (o MemberHTTPObservation) Clone() MemberHTTPObservation { o.Gateway = o.Gateway.Clone(); return o }
func (r MemberHTTPReadiness) Clone() MemberHTTPReadiness {
	r.Before = r.Before.Clone()
	r.After = r.After.Clone()
	return r
}

func memberHTTPSelection(p Plan, w Workspace, s MemberLaunch) (Member, string, error) {
	if s.Validate(p, w) != nil || (s.Version != 2 && s.Version != 3) || s.Phase != "running" || s.Running == nil || s.Running.Birth == nil || len(s.Peers) != 0 {
		return Member{}, "", ErrMemberHTTPReadiness
	}
	for _, m := range p.Members {
		if m.InstanceID == s.InstanceID {
			return m, fmt.Sprintf("http://127.0.0.1:%d/~/host", m.HTTPPort), nil
		}
	}
	return Member{}, "", ErrMemberHTTPReadiness
}

func memberHTTPObservationValid(s MemberLaunch, o MemberHTTPObservation, now time.Time) bool {
	if now.IsZero() || o.Pier != s.Running.Birth.Pier || !validMemberBirthProcess(s, o.Process, s.Running.Birth.EmptyObservedAt, now) || !o.Process.StartedAt.Equal(s.Running.StartedAt) || o.Process.ObservedAt.Before(s.Running.ObservedAt) || o.Gateway.ObservedAt.Before(s.Running.ObservedAt) || !memberFresh(o.Gateway.ObservedAt, now) {
		return false
	}
	want, got := s.Running.Gateway.Clone(), o.Gateway.Clone()
	want.ObservedAt = time.Time{}
	got.ObservedAt = time.Time{}
	return reflect.DeepEqual(want, got)
}

func (r MemberHTTPReadiness) Validate(p Plan, w Workspace, s MemberLaunch) error {
	m, endpoint, err := memberHTTPSelection(p, w, s)
	if err != nil || r.Version != 1 || r.InstanceID != s.InstanceID || r.LaunchOperationID != s.OperationID || r.LaunchDigest != digest("omabit-private-member-http-launch-v1", s) || r.Ship != m.Ship || r.Endpoint != endpoint || r.CapturedAt.IsZero() {
		return ErrMemberHTTPReadiness
	}
	for _, o := range []MemberHTTPObservation{r.Before, r.After} {
		if !memberHTTPObservationValid(s, o, r.CapturedAt) || o.Process.ObservedAt.After(r.CapturedAt) || o.Gateway.ObservedAt.After(r.CapturedAt) {
			return ErrMemberHTTPReadiness
		}
	}
	if r.After.Process.ObservedAt.Before(r.Before.Process.ObservedAt) || r.After.Gateway.ObservedAt.Before(r.Before.Gateway.ObservedAt) {
		return ErrMemberHTTPReadiness
	}
	before, after := r.Before.Process, r.After.Process
	before.ObservedAt = time.Time{}
	after.ObservedAt = time.Time{}
	if before != after {
		return ErrMemberHTTPReadiness
	}
	return nil
}
func (r MemberHTTPReadiness) Digest(p Plan, w Workspace, s MemberLaunch) (string, error) {
	if err := r.Validate(p, w, s); err != nil {
		return "", err
	}
	return digest("omabit-private-member-http-readiness-v1", r), nil
}

type MemberHTTPReadinessDriver interface {
	VerifyHeld(context.Context, MemberLaunch) error
	ObserveRuntime(context.Context, MemberLaunch) (MemberHTTPObservation, error)
	ReadHostIdentity(context.Context, string) (string, error)
}

// One read-only attempt. A bounded higher-level operation may repeat an HTTP
// observation; this helper never repeats a call, launches a ship, opens a login,
// rewrites launch history, or returns partial evidence after failure.
func CollectMemberHTTPReadiness(ctx context.Context, p Plan, w Workspace, s MemberLaunch, d MemberHTTPReadinessDriver, now func() time.Time) (MemberHTTPReadiness, error) {
	refuse := func() (MemberHTTPReadiness, error) { return MemberHTTPReadiness{}, ErrMemberHTTPReadiness }
	p, w = gatewayInputs(p, w)
	s = s.Clone()
	m, endpoint, err := memberHTTPSelection(p, w, s)
	if err != nil || d == nil || now == nil || ctx.Err() != nil {
		return refuse()
	}
	verify := func() bool { return ctx.Err() == nil && d.VerifyHeld(ctx, s.Clone()) == nil && ctx.Err() == nil }
	if !verify() {
		return refuse()
	}
	before, err := d.ObserveRuntime(ctx, s.Clone())
	before = before.Clone()
	if err != nil || !memberHTTPObservationValid(s, before, now()) || !verify() {
		return refuse()
	}
	identity, err := d.ReadHostIdentity(ctx, endpoint)
	if err != nil || identity != m.Ship || !verify() {
		return refuse()
	}
	after, err := d.ObserveRuntime(ctx, s.Clone())
	after = after.Clone()
	if err != nil || !verify() {
		return refuse()
	}
	r := MemberHTTPReadiness{Version: 1, InstanceID: s.InstanceID, LaunchOperationID: s.OperationID, LaunchDigest: digest("omabit-private-member-http-launch-v1", s), Ship: identity, Endpoint: endpoint, Before: before, After: after, CapturedAt: now()}
	if ctx.Err() != nil || r.Validate(p, w, s) != nil {
		return refuse()
	}
	return r, nil
}

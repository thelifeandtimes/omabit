package privatechain

import (
	"context"
	"errors"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

// All effects are exclusive and tied to the exact durable dispatch. Preparation
// creates only the separately owned pinned-pill artifact directory, never a pier
// or key. The complete gateway, peer contracts and held empty target mount must
// be verified before each effect; bootstrap key proof and the held key/pill must
// be rechecked at Create/Start. No method may adopt, remove, repair or retry.
type MemberLaunchDriver interface {
	VerifyPreparation(context.Context, MemberLaunch) error
	PrepareArtifact(context.Context, MemberLaunch) (pierstore.DirectoryIdentity, error)
	VerifyKey(context.Context, MemberLaunch) (MemberKeyAcceptance, error)
	CreateMember(context.Context, MemberLaunch) (string, error)
	InspectCreated(context.Context, MemberLaunch) error
	StartMember(context.Context, MemberLaunch) error
	ObserveRunning(context.Context, MemberLaunch) (MemberRunningObservation, error)
}

// Capture is a single read-only attempt using the original pre-Start handle.
// It neither reopens an existing pier nor authorizes another Start.
type CapturedMemberLaunchDriver interface {
	MemberLaunchDriver
	CaptureBirth(context.Context, MemberLaunch) (MemberBirthEvidence, error)
}
type MemberLaunchCommit func(expected string, next MemberLaunch) error

// Fresh intent only. A known artifact/container identity is retained even when
// an effect returns an error. A lost commit reply or Start acknowledgement never
// authorizes another effect. The caller reloads its durable receipt and offers
// separately attributed observation/recovery; this function never resumes it.
func ExecuteMemberLaunch(ctx context.Context, p Plan, w Workspace, s MemberLaunch, d MemberLaunchDriver, commit MemberLaunchCommit, now func() time.Time) (MemberLaunch, error) {
	return executeMemberLaunch(ctx, p, w, s, d, commit, now, false)
}

// Version3 records capture before full running observation. It accepts only
// fresh intent; an interrupted journal needs separately attributed observation.
func ExecuteCapturedMemberLaunch(ctx context.Context, p Plan, w Workspace, s MemberLaunch, d CapturedMemberLaunchDriver, commit MemberLaunchCommit, now func() time.Time) (MemberLaunch, error) {
	return executeMemberLaunch(ctx, p, w, s, d, commit, now, true)
}

func executeMemberLaunch(ctx context.Context, p Plan, w Workspace, s MemberLaunch, d MemberLaunchDriver, commit MemberLaunchCommit, now func() time.Time, capture bool) (MemberLaunch, error) {
	version := 2
	if capture {
		version = 3
	}
	p, w = gatewayInputs(p, w)
	s = s.Clone()
	if s.Validate(p, w) != nil || s.Version != version || s.Phase != "planned" || d == nil || commit == nil || now == nil {
		return s, ErrMemberLaunch
	}
	fail := func() error { return errors.Join(ErrMemberLaunchUnresolved, ctx.Err()) }
	step := func(phase string, result MemberLaunchResult) error {
		next, err := s.Advance(p, w, s.Phase, phase, result)
		if err != nil {
			return fail()
		}
		if commit(s.Phase, next.Clone()) != nil {
			return fail()
		}
		s = next
		return nil
	}
	check := func(f func(context.Context, MemberLaunch) error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if f(ctx, s.Clone()) != nil {
			return fail()
		}
		return ctx.Err()
	}
	if err := check(d.VerifyPreparation); err != nil {
		return s, err
	}
	if err := step("artifact-dispatched", MemberLaunchResult{}); err != nil {
		return s, err
	}
	if ctx.Err() != nil {
		return s, ctx.Err()
	}
	artifact, effectErr := d.PrepareArtifact(ctx, s.Clone())
	if artifact != (pierstore.DirectoryIdentity{}) {
		if err := step("artifact-created", MemberLaunchResult{Artifact: artifact}); err != nil {
			return s, err
		}
	}
	if effectErr != nil || s.Phase != "artifact-created" {
		return s, fail()
	}
	if err := check(d.VerifyPreparation); err != nil {
		return s, err
	}
	key, err := d.VerifyKey(ctx, s.Clone())
	if err != nil || ctx.Err() != nil || s.checkKey(p, w, key, now()) != nil {
		return s, fail()
	}
	if err = step("key-verified", MemberLaunchResult{Key: &key}); err != nil {
		return s, err
	}
	if err = check(d.VerifyPreparation); err != nil {
		return s, err
	}
	if err = step("create-dispatched", MemberLaunchResult{}); err != nil {
		return s, err
	}
	if ctx.Err() != nil {
		return s, ctx.Err()
	}
	id, effectErr := d.CreateMember(ctx, s.Clone())
	if id != "" {
		if err = step("created", MemberLaunchResult{ContainerID: id}); err != nil {
			return s, err
		}
	}
	if effectErr != nil || s.Phase != "created" {
		return s, fail()
	}
	if err = check(d.InspectCreated); err != nil {
		return s, err
	}
	if err = check(d.VerifyPreparation); err != nil {
		return s, err
	}
	if err = step("start-dispatched", MemberLaunchResult{}); err != nil {
		return s, err
	}
	if err = check(d.StartMember); err != nil {
		return s, err
	}
	if err = step("started", MemberLaunchResult{}); err != nil {
		return s, err
	}
	if ctx.Err() != nil {
		return s, ctx.Err()
	}
	if capture {
		driver, ok := d.(CapturedMemberLaunchDriver)
		if !ok {
			return s, fail()
		}
		birth, err := driver.CaptureBirth(ctx, s.Clone())
		if err != nil || ctx.Err() != nil || birth.Validate(p, w, s) != nil || !memberFresh(birth.CapturedAt, now()) {
			return s, fail()
		}
		if err = step("captured", MemberLaunchResult{Capture: &birth}); err != nil {
			return s, err
		}
		if ctx.Err() != nil {
			return s, ctx.Err()
		}
	}
	running, err := d.ObserveRunning(ctx, s.Clone())
	if err != nil || ctx.Err() != nil || s.checkRunning(p, w, running, now()) != nil {
		return s, fail()
	}
	if err = step("running", MemberLaunchResult{Running: &running}); err != nil {
		return s, err
	}
	return s, nil
}

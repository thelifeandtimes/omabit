//go:build linux

package privatechain

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

// This handle must be opened before first boot while the caller holds the
// common runner fence. It continuously holds the originally empty mount. Arm
// rechecks that same empty object after durable Start dispatch, immediately
// before the single Start effect. Capture cannot be opened after boot or reused
// for recovery. There are no filesystem writes, locks removed or payload reads.
type MemberBirthCapture struct {
	mu                                       sync.Mutex
	plan                                     Plan
	workspace                                Workspace
	initial                                  MemberLaunch
	armed                                    *MemberLaunch
	emptyAt                                  time.Time
	mount                                    *MemberMountVerifier
	pier                                     *MemberPierVerifier
	armAttempted, captureAttempted, verified bool
	now                                      func() time.Time
}

func OpenMemberBirthCapture(ctx context.Context, directory string, p Plan, w Workspace, s MemberLaunch) (*MemberBirthCapture, error) {
	return openMemberBirthCapture(ctx, directory, p, w, s, time.Now)
}
func openMemberBirthCapture(ctx context.Context, directory string, p Plan, w Workspace, s MemberLaunch, now func() time.Time) (*MemberBirthCapture, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if s.Validate(p, w) != nil || s.Phase != "planned" || now == nil {
		return nil, ErrMemberBirth
	}
	mount, err := OpenMemberMountVerifier(ctx, directory, p, w, s.InstanceID)
	if err != nil {
		return nil, ErrMemberBirth
	}
	if mount.VerifyEmpty(ctx) != nil {
		mount.Close()
		return nil, ErrMemberBirth
	}
	p, w = gatewayInputs(p, w)
	return &MemberBirthCapture{plan: p, workspace: w, initial: s.Clone(), mount: mount, now: now}, nil
}

func (v *MemberBirthCapture) Arm(ctx context.Context, s MemberLaunch) error {
	s = s.Clone()
	if v == nil {
		return ErrMemberBirth
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.armAttempted || v.mount == nil {
		return ErrMemberBirth
	}
	v.armAttempted = true
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if s.Validate(v.plan, v.workspace) != nil || s.Phase != "start-dispatched" {
		return ErrMemberBirth
	}
	base := s.Clone()
	base.Phase = "planned"
	base.Artifact = pierstore.DirectoryIdentity{}
	base.Key = nil
	base.ContainerID = ""
	base.Running = nil
	if !reflect.DeepEqual(base, v.initial) || v.mount.VerifyEmpty(ctx) != nil {
		return ErrMemberBirth
	}
	now := v.now().UTC()
	if now.IsZero() || now.Before(s.Key.CheckedAt) || ctx.Err() != nil {
		return ErrMemberBirth
	}
	selected := s.Clone()
	v.armed = &selected
	v.emptyAt = now
	return nil
}

type MemberBirthObserver func(context.Context, MemberLaunch) (MemberBirthProcess, error)

// One read-only attempt after acknowledged Start. Both process observations and
// the held parent/pier/.urb checks must agree. No partial evidence is returned;
// errors retain all on-disk state, and cannot authorize another first boot.
func (v *MemberBirthCapture) Capture(ctx context.Context, s MemberLaunch, observe MemberBirthObserver) (MemberBirthEvidence, error) {
	s = s.Clone()
	if v == nil {
		return MemberBirthEvidence{}, ErrMemberBirth
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	refuse := func() (MemberBirthEvidence, error) {
		return MemberBirthEvidence{}, errors.Join(ErrMemberBirth, ctx.Err())
	}
	if v.captureAttempted || v.armed == nil || v.mount == nil || observe == nil {
		return refuse()
	}
	v.captureAttempted = true
	want := v.armed.Clone()
	want.Phase = "started"
	if !reflect.DeepEqual(want, s) || s.Validate(v.plan, v.workspace) != nil || ctx.Err() != nil {
		return refuse()
	}
	before, err := observe(ctx, s.Clone())
	if err != nil || ctx.Err() != nil || !validMemberBirthProcess(s, before, v.emptyAt, v.now()) || v.mount.Verify(ctx) != nil {
		return refuse()
	}
	ship := ""
	for _, m := range v.workspace.Mounts {
		if m.InstanceID == s.InstanceID {
			ship = strings.TrimPrefix(m.Ship, "~")
		}
	}
	if ship == "" || workspaceContents(v.mount.held[3], []string{ship}) != nil {
		return refuse()
	}
	var files []*os.File
	defer func() {
		if files != nil {
			_ = closeMemberMountFiles(files)
		}
	}()
	var expected [2]pierstore.DirectoryIdentity
	parent := v.mount.held[3]
	for i, name := range []string{ship, ".urb"} {
		f, e := memberPierChild(parent, name, true)
		if e != nil {
			return refuse()
		}
		files = append(files, f)
		st, e := memberPierMetadata(f)
		if e != nil {
			return refuse()
		}
		expected[i] = st.identity()
		parent = f
	}
	id := pierstore.Identity{Device: expected[0].Device, Inode: expected[0].Inode, Birth: expected[0].Birth, UrbDevice: expected[1].Device, UrbInode: expected[1].Inode, UrbBirth: expected[1].Birth}
	v.pier = &MemberPierVerifier{mount: v.mount, ship: ship, expected: expected, held: files}
	files = nil
	if v.pier.Verify(ctx) != nil {
		return refuse()
	}
	after, err := observe(ctx, s.Clone())
	if err != nil || ctx.Err() != nil || v.pier.Verify(ctx) != nil {
		return refuse()
	}
	e := MemberBirthEvidence{InstanceID: s.InstanceID, OperationID: s.OperationID, RuntimeID: s.RuntimeID, ContainerID: s.ContainerID, ContractDigest: s.ContractDigest, EmptyObservedAt: v.emptyAt, CapturedAt: v.now().UTC(), Pier: id, Before: before, After: after}
	if e.Validate(v.plan, v.workspace, s) != nil || ctx.Err() != nil {
		return refuse()
	}
	v.verified = true
	return e, nil
}

// Verify keeps the captured objects pinned for the caller's subsequent full
// gateway/HTTP readiness checks. It does not persist them or infer clean Stop.
func (v *MemberBirthCapture) Verify(ctx context.Context) error {
	if v == nil {
		return ErrMemberBirth
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.verified || v.pier == nil || v.mount == nil {
		return ErrMemberBirth
	}
	return v.pier.Verify(ctx)
}
func (v *MemberBirthCapture) Close() error {
	if v == nil {
		return nil
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	var err error
	if v.pier != nil {
		err = v.pier.Close()
	} else if v.mount != nil {
		err = v.mount.Close()
	}
	v.mount = nil
	v.pier = nil
	v.armed = nil
	v.verified = false
	return err
}

//go:build linux

package privatechain

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"golang.org/x/sys/unix"
)

type memberBirthFixture struct {
	directory, root, pier        string
	p                            Plan
	w                            Workspace
	initial, dispatched, started MemberLaunch
	now                          time.Time
	v                            *MemberBirthCapture
	calls                        int
}

func newMemberBirthFixture(t *testing.T) *memberBirthFixture {
	t.Helper()
	d, p, w, g, root, _ := memberKeyFixture(t)
	initial, err := NewMemberLaunch(p, w, g, p.Members[0].InstanceID, strings.Repeat("d", 32), "birth-member", strings.Repeat("6", 64), nil)
	mustInput(t, err)
	f := &memberBirthFixture{directory: d, root: root, p: p, w: w, initial: initial, now: workspaceNow.Add(3 * time.Second)}
	f.pier = filepath.Join(root, "piers", initial.InstanceID, strings.TrimPrefix(p.Members[0].Ship, "~"))
	f.v, err = openMemberBirthCapture(context.Background(), d, p, w, initial, func() time.Time { return f.now })
	mustInput(t, err)
	t.Cleanup(func() { f.v.Close() })
	artifact, err := pierstore.IdentifyMount(t.TempDir())
	mustInput(t, err)
	keyGateway := g.Observation.Clone()
	keyGateway.ObservedAt = workspaceNow.Add(2 * time.Second)
	key := MemberKeyAcceptance{CheckedAt: keyGateway.ObservedAt, ContractDigest: initial.ContractDigest, KeySHA256: strings.Repeat("a", 64), Gateway: keyGateway, Verified: true}
	j := initial
	for _, phase := range []string{"artifact-dispatched", "artifact-created", "key-verified", "create-dispatched", "created", "start-dispatched"} {
		result := MemberLaunchResult{}
		switch phase {
		case "artifact-created":
			result.Artifact = artifact
		case "key-verified":
			result.Key = &key
		case "created":
			result.ContainerID = strings.Repeat("7", 64)
		}
		j, err = j.Advance(p, w, j.Phase, phase, result)
		mustInput(t, err)
	}
	f.dispatched = j
	f.started = j.Clone()
	f.started.Phase = "started"
	return f
}
func (f *memberBirthFixture) boot(t *testing.T) {
	t.Helper()
	mustInput(t, f.v.Arm(context.Background(), f.dispatched))
	mustInput(t, os.MkdirAll(filepath.Join(f.pier, ".urb/log"), 0755))
	f.now = workspaceNow.Add(5 * time.Second)
}
func (f *memberBirthFixture) observe(_ context.Context, s MemberLaunch) (MemberBirthProcess, error) {
	f.calls++
	f.now = f.now.Add(time.Millisecond)
	return MemberBirthProcess{ObservedAt: f.now, StartedAt: workspaceNow.Add(4 * time.Second), ContainerID: s.ContainerID, ContractDigest: s.ContractDigest, Namespace: s.Gateway.Observation.Namespace, CompleteContract: true, Running: true}, nil
}

func TestMemberBirthContinuousEmptyMountAndReadOnlyCapture(t *testing.T) {
	f := newMemberBirthFixture(t)
	f.boot(t)
	// Birth capture deliberately never traverses payload, peers, chain or keys.
	mustInput(t, unix.Mkfifo(filepath.Join(f.pier, ".urb/log/opaque"), 0600))
	mustInput(t, unix.Mkfifo(filepath.Join(f.root, "piers", f.p.Members[1].InstanceID, "unread-peer"), 0600))
	mustInput(t, os.WriteFile(filepath.Join(f.pier, ".vere.lock"), []byte("synthetic running lock"), 0600))
	before := workspaceFiles(t, f.directory)
	e, err := f.v.Capture(context.Background(), f.started, f.observe)
	mustInput(t, err)
	mustInput(t, e.Validate(f.p, f.w, f.started))
	mustInput(t, f.v.Verify(context.Background()))
	id, err := pierstore.Identify(f.pier)
	mustInput(t, err)
	if e.Pier != id || f.calls != 2 || !reflect.DeepEqual(before, workspaceFiles(t, f.directory)) {
		t.Fatal("capture changed/read unexpected state")
	}
	if _, err = f.v.Capture(context.Background(), f.started, f.observe); err == nil || f.calls != 2 {
		t.Fatal("capture repeated")
	}
	if v, err := OpenMemberBirthCapture(context.Background(), f.directory, f.p, f.w, f.initial); err == nil {
		v.Close()
		t.Fatal("capture opened after boot")
	}
	mustInput(t, f.v.Close())
	mustInput(t, f.v.Close())
	if f.v.Verify(context.Background()) == nil {
		t.Fatal("closed capture verified")
	}
}

func TestMemberBirthArmRefusesChangedOrNonemptyDispatch(t *testing.T) {
	for _, kind := range []string{"phase", "operation", "runtime", "gateway", "nonempty", "cancelled", "closed", "pre-key-clock"} {
		t.Run(kind, func(t *testing.T) {
			f := newMemberBirthFixture(t)
			s := f.dispatched.Clone()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch kind {
			case "phase":
				s.Phase = "created"
			case "operation":
				s.OperationID = "another"
			case "runtime":
				s.RuntimeID = strings.Repeat("e", 32)
			case "gateway":
				s.Gateway.ContainerID = strings.Repeat("9", 64)
			case "nonempty":
				mustInput(t, os.Mkdir(f.pier, 0755))
			case "cancelled":
				cancel()
			case "closed":
				mustInput(t, f.v.Close())
			case "pre-key-clock":
				f.now = workspaceNow
			}
			before := workspaceFiles(t, f.directory)
			if f.v.Arm(ctx, s) == nil {
				t.Fatal("invalid arm accepted")
			}
			if f.v.Arm(context.Background(), f.dispatched) == nil {
				t.Fatal("failed arm repeated")
			}
			if e, err := f.v.Capture(context.Background(), f.started, f.observe); err == nil || !reflect.DeepEqual(e, MemberBirthEvidence{}) || f.calls != 0 {
				t.Fatal("unarmed capture accepted")
			}
			if !reflect.DeepEqual(before, workspaceFiles(t, f.directory)) {
				t.Fatal("arm refusal changed files")
			}
		})
	}
}

func TestMemberBirthRefusesMissingOrReplacedObjects(t *testing.T) {
	for _, kind := range []string{"missing-pier", "missing-urb", "extra-child", "linked-pier", "linked-urb", "fifo-urb", "public-pier", "mount-replaced", "pier-during-observation", "urb-during-observation"} {
		t.Run(kind, func(t *testing.T) {
			f := newMemberBirthFixture(t)
			f.boot(t)
			kept := filepath.Join(t.TempDir(), "retained")
			replace := func(target string) { mustInput(t, os.Rename(target, kept)); mustInput(t, os.Mkdir(target, 0755)) }
			switch kind {
			case "missing-pier":
				mustInput(t, os.Rename(f.pier, kept))
			case "missing-urb":
				mustInput(t, os.Rename(filepath.Join(f.pier, ".urb"), kept))
			case "extra-child":
				mustInput(t, os.Mkdir(filepath.Join(filepath.Dir(f.pier), "extra"), 0700))
			case "linked-pier":
				mustInput(t, os.Rename(f.pier, kept))
				mustInput(t, os.Symlink(kept, f.pier))
			case "linked-urb":
				target := filepath.Join(f.pier, ".urb")
				mustInput(t, os.Rename(target, kept))
				mustInput(t, os.Symlink(kept, target))
			case "fifo-urb":
				target := filepath.Join(f.pier, ".urb")
				mustInput(t, os.Rename(target, kept))
				mustInput(t, unix.Mkfifo(target, 0600))
			case "public-pier":
				mustInput(t, os.Chmod(f.pier, 0777))
			case "mount-replaced":
				replace(filepath.Dir(f.pier))
			}
			observe := func(ctx context.Context, s MemberLaunch) (MemberBirthProcess, error) {
				o, e := f.observe(ctx, s)
				if f.calls == 2 {
					switch kind {
					case "pier-during-observation":
						replace(f.pier)
					case "urb-during-observation":
						replace(filepath.Join(f.pier, ".urb"))
					}
				}
				return o, e
			}
			e, err := f.v.Capture(context.Background(), f.started, observe)
			if err == nil || !reflect.DeepEqual(e, MemberBirthEvidence{}) {
				t.Fatal("changed objects accepted")
			}
			if f.v.Verify(context.Background()) == nil {
				t.Fatal("failed capture exposed accepted verifier")
			}
			count := f.calls
			if _, err = f.v.Capture(context.Background(), f.started, observe); err == nil || f.calls != count {
				t.Fatal("failed capture repeated")
			}
			if kind != "extra-child" && kind != "public-pier" {
				if _, err = os.Lstat(kept); err != nil {
					t.Fatal("original removed")
				}
			}
		})
	}
}

func TestMemberBirthRequiresFreshUnchangedExactProcess(t *testing.T) {
	for _, kind := range []string{"container", "contract", "namespace", "incomplete", "stopped", "old-start", "old-observation", "future", "before-start", "second-start", "second-namespace", "time-reversed", "driver-error", "cancelled", "selection"} {
		t.Run(kind, func(t *testing.T) {
			f := newMemberBirthFixture(t)
			f.boot(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			s := f.started.Clone()
			if kind == "selection" {
				s.OperationID = "changed"
			}
			observe := func(ctx context.Context, s MemberLaunch) (MemberBirthProcess, error) {
				o, err := f.observe(ctx, s)
				switch kind {
				case "container":
					o.ContainerID = strings.Repeat("8", 64)
				case "contract":
					o.ContractDigest = strings.Repeat("9", 64)
				case "namespace":
					o.Namespace.Inode++
				case "incomplete":
					o.CompleteContract = false
				case "stopped":
					o.Running = false
				case "old-start":
					o.StartedAt = workspaceNow
				case "old-observation":
					o.ObservedAt = f.now.Add(-31 * time.Second)
				case "future":
					o.ObservedAt = f.now.Add(6 * time.Second)
				case "before-start":
					o.ObservedAt = workspaceNow
				case "second-start":
					if f.calls == 2 {
						o.StartedAt = o.StartedAt.Add(time.Millisecond)
					}
				case "second-namespace":
					if f.calls == 2 {
						o.Namespace.Inode++
					}
				case "time-reversed":
					if f.calls == 2 {
						o.ObservedAt = o.ObservedAt.Add(-time.Second)
					}
				case "driver-error":
					err = errors.New("PRIVATE diagnostics")
				case "cancelled":
					cancel()
				}
				return o, err
			}
			before := workspaceFiles(t, f.directory)
			e, err := f.v.Capture(ctx, s, observe)
			if err == nil || strings.Contains(err.Error(), "PRIVATE") || !reflect.DeepEqual(e, MemberBirthEvidence{}) {
				t.Fatal("bad process accepted or diagnostic exposed", err)
			}
			if f.v.Verify(context.Background()) == nil || !reflect.DeepEqual(before, workspaceFiles(t, f.directory)) {
				t.Fatal("failed capture granted authority or changed files")
			}
		})
	}
}

func TestMemberBirthHistoricalEvidenceCannotChangeSelection(t *testing.T) {
	f := newMemberBirthFixture(t)
	f.boot(t)
	e, err := f.v.Capture(context.Background(), f.started, f.observe)
	mustInput(t, err)
	for _, kind := range []string{"instance", "operation", "runtime", "container", "contract", "zero-pier", "alias", "foreign-device", "empty-time", "capture-time"} {
		t.Run(kind, func(t *testing.T) {
			bad := e
			switch kind {
			case "instance":
				bad.InstanceID = "other"
			case "operation":
				bad.OperationID = "other"
			case "runtime":
				bad.RuntimeID = strings.Repeat("e", 32)
			case "container":
				bad.ContainerID = strings.Repeat("8", 64)
			case "contract":
				bad.ContractDigest = strings.Repeat("9", 64)
			case "zero-pier":
				bad.Pier = pierstore.Identity{}
			case "alias":
				bad.Pier.UrbDevice = bad.Pier.Device
				bad.Pier.UrbInode = bad.Pier.Inode
				bad.Pier.UrbBirth = bad.Pier.Birth
			case "foreign-device":
				bad.Pier.Device++
			case "empty-time":
				bad.EmptyObservedAt = workspaceNow
			case "capture-time":
				bad.CapturedAt = workspaceNow
			}
			if bad.Validate(f.p, f.w, f.started) == nil {
				t.Fatal("altered evidence accepted")
			}
		})
	}
	// No wall-clock expiry of historical evidence.
	f.now = f.now.Add(24 * time.Hour)
	mustInput(t, e.Validate(f.p, f.w, f.started))
}

func TestMemberBirthObserverCannotMutateAcceptedInputs(t *testing.T) {
	f := newMemberBirthFixture(t)
	f.boot(t)
	expected := f.started.Clone()
	observe := func(ctx context.Context, s MemberLaunch) (MemberBirthProcess, error) {
		o, err := f.observe(ctx, s)
		s.Gateway.Observation.PublicKeys[0].Auth = "changed observer input"
		f.started.Gateway.Observation.PublicKeys[0].Auth = "changed caller input"
		return o, err
	}
	e, err := f.v.Capture(context.Background(), f.started, observe)
	mustInput(t, err)
	mustInput(t, e.Validate(f.p, f.w, expected))
}

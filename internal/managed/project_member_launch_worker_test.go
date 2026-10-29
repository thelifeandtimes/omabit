package managed

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

type storedMemberLaunchDriver struct {
	t               *testing.T
	s               *store
	instance, fault string
	cancel          context.CancelFunc
	effects         []string
}

func (d *storedMemberLaunchDriver) retained(state projectnetwork.MemberLaunch, phase string) {
	d.t.Helper()
	j, err := loadProjectMemberLaunch(d.s.db, d.instance)
	if err != nil || j.Launch == nil || j.Launch.Phase != phase || !reflect.DeepEqual(*j.Launch, state) {
		d.t.Fatal("effect before committed journal", phase, err)
	}
	if err = validateProjectMemberLaunches(d.s.db); err != nil {
		d.t.Fatal("effect before committed mirror", err)
	}
}
func (d *storedMemberLaunchDriver) VerifyEngine(context.Context, string) error { return nil }
func (d *storedMemberLaunchDriver) PrepareWorkspace(_ context.Context, state projectnetwork.MemberLaunch) (pierstore.DirectoryIdentity, error) {
	d.retained(state, "workspace-dispatched")
	d.effects = append(d.effects, "workspace")
	return pierstore.DirectoryIdentity{Device: 7, Inode: 192, Birth: "1000:2"}, nil
}
func (d *storedMemberLaunchDriver) Preflight(_ context.Context, state projectnetwork.MemberLaunch) error {
	d.retained(state, state.Phase)
	if state.Phase != "workspace-created" && state.Phase != "created" {
		d.t.Fatal("incorrect preflight phase")
	}
	return nil
}
func (d *storedMemberLaunchDriver) Create(_ context.Context, state projectnetwork.MemberLaunch) (string, error) {
	d.retained(state, "create-dispatched")
	d.effects = append(d.effects, "create")
	if d.fault == "late-create" {
		return strings.Repeat("c", 64), errors.New("synthetic late error")
	}
	return strings.Repeat("c", 64), nil
}
func (d *storedMemberLaunchDriver) Start(_ context.Context, state projectnetwork.MemberLaunch) error {
	d.retained(state, "start-dispatched")
	d.effects = append(d.effects, "start")
	if d.fault == "lost-start" {
		return errors.New("synthetic lost response")
	}
	if d.fault == "cancel-after-start" {
		d.cancel()
	}
	return nil
}
func (d *storedMemberLaunchDriver) Inspect(_ context.Context, state projectnetwork.MemberLaunch) (projectnetwork.Observation, error) {
	d.retained(state, state.Phase)
	p, err := loadProject(d.s.db, state.Storage.ProjectID)
	if err != nil {
		d.t.Fatal(err)
	}
	r := state.Anchor.Prepared
	o := projectnetwork.Observation{ProjectID: r.ProjectID, Revision: r.ProjectRevision, Generation: r.Generation, EngineID: r.EngineID, ObservedAt: time.Now().UTC(), Complete: true, Bridge: projectnetwork.ResourceObservation{Resource: projectnetwork.ResourceIdentity{ID: r.Bridge.ID, ContractDigest: r.Bridge.ContractDigest}, Present: true}, BridgeAttachments: []string{r.Anchor.ID}, Anchor: projectnetwork.AnchorObservation{ResourceObservation: projectnetwork.ResourceObservation{Resource: projectnetwork.ResourceIdentity{ID: r.Anchor.ID, ContractDigest: r.Anchor.ContractDigest}, Present: true}, Running: true, Namespace: *state.Anchor.Namespace, BridgeID: r.Bridge.ID}}
	for _, member := range p.Plan.Members {
		o.Anchor.Requested = append(o.Anchor.Requested, projectnetwork.Publication{InternalPort: member.InternalHTTPPort, HostAddress: p.Plan.BindAddress, HostPort: member.HTTPPort, Protocol: "tcp"})
	}
	o.Anchor.Active = append([]projectnetwork.Publication(nil), o.Anchor.Requested...)
	for _, peer := range state.Peers {
		o.Members = append(o.Members, projectnetwork.MemberObservation{InstanceID: peer.InstanceID, ResourceObservation: projectnetwork.ResourceObservation{Resource: peer.Container, Present: true}, Running: true, Namespace: *state.Anchor.Namespace, AnchorID: r.Anchor.ID})
	}
	if state.Container.ID != "" {
		m := projectnetwork.MemberObservation{InstanceID: d.instance, ResourceObservation: projectnetwork.ResourceObservation{Resource: state.Container, Present: true}, Running: state.Phase == "started", AnchorID: r.Anchor.ID}
		if m.Running {
			m.Namespace = *state.Anchor.Namespace
		}
		o.Members = append(o.Members, m)
	}
	return o, nil
}
func TestProjectMemberLaunchWorkerUsesCommittedSQLiteEvidence(t *testing.T) {
	for fault, phase := range map[string]string{"success": "running", "late-create": "created", "lost-start": "start-dispatched", "cancel-after-start": "started"} {
		t.Run(fault, func(t *testing.T) {
			s := testStore(t)
			p, req := preparedMemberLaunchProject(t, s)
			instance := p.Members[0].InstanceID
			op, _, err := s.acceptProjectMemberLaunch("actor", p.ID, instance, req)
			if err != nil {
				t.Fatal(err)
			}
			initial := memberLaunchSelection(t, s, op)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			driver := &storedMemberLaunchDriver{t: t, s: s, instance: instance, fault: fault, cancel: cancel}
			err = s.runProjectMemberLaunch(ctx, instance, op.ID, initial, driver)
			if (err == nil) != (fault == "success") {
				t.Fatal(err)
			}
			j, err := loadProjectMemberLaunch(s.db, instance)
			if err != nil || j.Launch == nil || j.Launch.Phase != phase || j.Launch.Container.ID != strings.Repeat("c", 64) {
				t.Fatal(j, err)
			}
			actual, err := loadOperation(s.db, op.ID)
			if err != nil {
				t.Fatal(err)
			}
			want := "interrupted"
			if fault == "success" {
				want = "succeeded"
			}
			if actual.Status != want {
				t.Fatal(actual)
			}
			expected := []string{"workspace", "create"}
			if fault != "late-create" {
				expected = append(expected, "start")
			}
			if !reflect.DeepEqual(driver.effects, expected) {
				t.Fatal(driver.effects)
			}
			if err = s.runProjectMemberLaunch(context.Background(), instance, op.ID, initial, driver); err == nil || !reflect.DeepEqual(driver.effects, expected) {
				t.Fatal("worker replayed effects", err)
			}
		})
	}
}

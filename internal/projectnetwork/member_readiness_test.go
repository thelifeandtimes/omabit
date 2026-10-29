package projectnetwork

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

func initialReadinessFixture(t *testing.T) (Plan, []MemberLaunch, Observation, pierstore.Location) {
	t.Helper()
	plan, zod, o := memberLaunchFixture(t)
	peer := zod.Peers[0]
	storage, err := NewMemberStorage(plan, zod.Storage.ProjectID, 1, "nec", peer.RuntimeID, "storage-nec")
	if err != nil {
		t.Fatal(err)
	}
	storage.Phase = "observed"
	storage.Location.MountIdentity = pierstore.DirectoryIdentity{Device: 8, Inode: 90, Birth: "1000:3"}
	nec, err := NewMemberLaunch(plan, zod.Anchor, storage, "launch-nec", peer.Container.ContractDigest, nil)
	if err != nil {
		t.Fatal(err)
	}
	nec.Container = peer.Container
	nec.Phase = "running"
	nec.Namespace = zod.Anchor.Namespace
	nec.Workspace = pierstore.DirectoryIdentity{Device: 8, Inode: 111, Birth: "1000:4"}
	zod.Container.ID = strings.Repeat("c", 64)
	zod.Phase = "running"
	zod.Namespace = zod.Anchor.Namespace
	zod.Workspace = pierstore.DirectoryIdentity{Device: 8, Inode: 112, Birth: "1000:5"}
	o.Members = append(o.Members, MemberObservation{InstanceID: "zod", ResourceObservation: ResourceObservation{Resource: zod.Container, Present: true}, Running: true, AnchorID: zod.Anchor.Prepared.Anchor.ID, Namespace: *zod.Namespace})
	location := zod.Storage.Location
	location.Identity = pierstore.Identity{Device: 8, Inode: 113, UrbDevice: 8, UrbInode: 114, Birth: "1000:6", UrbBirth: "1000:7"}
	return plan, []MemberLaunch{zod, nec}, o, location
}

type initialReadinessFixtureDriver struct {
	before, after                       Observation
	first, second                       pierstore.Location
	ship, endpoint, fault               string
	reads, inspections, piers, verifies int
	cancel                              context.CancelFunc
}

func (d *initialReadinessFixtureDriver) VerifyEngine(context.Context, string) error {
	d.verifies++
	if d.fault == "engine" || d.fault == "late-engine" && d.verifies == 6 {
		return errors.New("private fixture detail")
	}
	return nil
}
func (d *initialReadinessFixtureDriver) InspectProject(context.Context) (Observation, error) {
	d.inspections++
	if d.fault == "inspect" {
		return Observation{}, errors.New("private fixture detail")
	}
	if d.inspections == 1 {
		return d.before, nil
	}
	return d.after, nil
}
func (d *initialReadinessFixtureDriver) InspectPier(_ context.Context, s MemberLaunch) (pierstore.Location, error) {
	d.piers++
	// Mutable driver arguments must not rewrite the original launch selection.
	s.Anchor.Namespace.Inode++
	s.Namespace.Inode++
	s.Peers[0].InstanceID = "edited"
	if d.fault == "pier" {
		return pierstore.Location{}, errors.New("private fixture detail")
	}
	if d.piers == 1 {
		return d.first, nil
	}
	return d.second, nil
}
func (d *initialReadinessFixtureDriver) ReadHostIdentity(_ context.Context, endpoint string) (string, error) {
	d.reads++
	d.endpoint = endpoint
	if d.fault == "cancel" {
		d.cancel()
	}
	if d.fault == "http" {
		return "", errors.New("secret HTTP response")
	}
	return d.ship, nil
}
func TestInitialMemberReadinessBracketsIdentityAndStorage(t *testing.T) {
	cases := []string{"pass", "engine", "late-engine", "inspect", "pier", "http", "cancel", "wrong-ship", "pier-replaced", "urb-replaced", "parent-replaced", "uncertain-storage", "missing-urb", "different-path", "incomplete-before", "stopped-peer", "unknown-join", "changed-anchor", "changed-member", "stale", "reverse-time", "future", "late-collection"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			plan, launches, o, location := initialReadinessFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			d := &initialReadinessFixtureDriver{before: cloneReadinessObservation(o), after: cloneReadinessObservation(o), first: location, second: location, ship: "~zod", fault: name, cancel: cancel}
			now := time.Unix(1001, 0)
			switch name {
			case "wrong-ship":
				d.ship = "~nec"
			case "pier-replaced":
				d.second.Identity.Inode++
			case "urb-replaced":
				d.second.Identity.UrbInode++
			case "parent-replaced":
				d.first.MountIdentity.Inode++
			case "uncertain-storage":
				d.first.Uncertain = true
			case "missing-urb":
				d.first.Identity.UrbBirth = ""
			case "different-path":
				d.first.Path = "fakeships/nec/nec"
			case "incomplete-before":
				d.before.Complete = false
			case "stopped-peer":
				d.before.Members[0].Running = false
				d.before.Members[0].Namespace = NamespaceIdentity{}
			case "unknown-join":
				d.before.BridgeAttachments = append(d.before.BridgeAttachments, strings.Repeat("f", 64))
			case "changed-anchor":
				d.after.Anchor.Namespace.Inode++
			case "changed-member":
				d.after.Members[0].Namespace.Inode++
			case "stale":
				d.before.ObservedAt = now.Add(-31 * time.Second)
			case "reverse-time":
				d.after.ObservedAt = o.ObservedAt.Add(-time.Second)
			case "future":
				d.after.ObservedAt = now.Add(6 * time.Second)
			case "late-collection":
				now = o.ObservedAt.Add(31 * time.Second)
			}
			beforeNamespace := *launches[0].Namespace
			got, err := CollectInitialMemberReadiness(ctx, plan, launches, "zod", d, func() time.Time { return now })
			if *launches[0].Namespace != beforeNamespace || launches[0].Peers[0].InstanceID != "nec" {
				t.Fatal("driver mutated retained launch")
			}
			if name == "pass" {
				if err != nil || got.Validate(plan, launches) != nil || got.Storage != location || d.reads != 1 || d.piers != 2 || d.inspections != 2 || d.endpoint != "http://127.0.0.1:18081/~/host" {
					t.Fatal(got, err, d)
				}
				// Retained observations cannot alias mutable buffers owned by the driver.
				d.before.Members[0].Namespace.Inode++
				d.after.Anchor.Active[0].HostPort++
				if got.Validate(plan, launches) != nil {
					t.Fatal("driver owns receipt buffers")
				}
				got.CapturedAt = got.CapturedAt.Add(31 * time.Second)
				if got.Validate(plan, launches) == nil {
					t.Fatal("stale proof accepted")
				}
			} else if err == nil || !reflect.DeepEqual(got, InitialMemberReadiness{}) || strings.Contains(err.Error(), "private") || strings.Contains(err.Error(), "secret") {
				t.Fatal("invalid/partial proof escaped", got, err)
			}
			if name == "incomplete-before" || name == "stopped-peer" || name == "unknown-join" || name == "stale" || name == "late-collection" {
				if d.reads != 0 || d.piers != 0 {
					t.Fatal("untrusted runtime was probed")
				}
			}
		})
	}
}
func TestInitialRuntimeBindingRequiresCompleteSerialLaunches(t *testing.T) {
	for _, name := range []string{"pass", "partial", "not-running", "different-anchor", "missing-peer", "wrong-peer", "duplicate-member", "different-plan", "duplicate-runtime", "duplicate-operation"} {
		t.Run(name, func(t *testing.T) {
			plan, launches, _, _ := initialReadinessFixture(t)
			switch name {
			case "partial":
				launches = launches[:1]
			case "not-running":
				launches[0].Phase = "started"
				launches[0].Namespace = nil
			case "different-anchor":
				copy := *launches[0].Anchor.Namespace
				copy.Inode++
				launches[0].Anchor.Namespace = &copy
				launches[0].Namespace = &copy
			case "missing-peer":
				launches[0].Peers = nil
			case "wrong-peer":
				launches[0].Peers[0].Container.ContractDigest = strings.Repeat("5", 64)
			case "duplicate-member":
				launches[1] = launches[0]
			case "different-plan":
				plan.Members[0].HTTPPort++
			case "duplicate-operation":
				launches[1].OperationID = launches[0].OperationID
			case "duplicate-runtime":
				launches[1].Storage.RuntimeID = launches[0].Storage.RuntimeID
			}
			b, err := InitialRuntimeBinding(plan, launches)
			if name == "pass" {
				if err != nil || b.Validate(plan) != nil || b.Members[0].InstanceID != "nec" {
					t.Fatal(b, err)
				}
			} else if err == nil || !reflect.DeepEqual(b, RuntimeBinding{}) {
				t.Fatal("invalid launch set accepted", b, err)
			}
		})
	}
}

func TestInitialMemberReadinessReceiptRejectsAlteredEvidence(t *testing.T) {
	mutations := map[string]func(*InitialMemberReadiness){
		"instance":        func(r *InitialMemberReadiness) { r.InstanceID = "nec" },
		"operation":       func(r *InitialMemberReadiness) { r.LaunchOperationID = "different" },
		"endpoint":        func(r *InitialMemberReadiness) { r.Endpoint = "http://127.0.0.1:18082/~/host" },
		"ship":            func(r *InitialMemberReadiness) { r.Ship = "~nec" },
		"binding":         func(r *InitialMemberReadiness) { r.Binding.EngineID = "different" },
		"parent":          func(r *InitialMemberReadiness) { r.Storage.MountIdentity.Inode++ },
		"pier-is-parent":  func(r *InitialMemberReadiness) { r.Storage.Identity.Inode = r.Storage.MountIdentity.Inode },
		"urb-is-pier":     func(r *InitialMemberReadiness) { r.Storage.Identity.UrbInode = r.Storage.Identity.Inode },
		"no-birth":        func(r *InitialMemberReadiness) { r.Storage.Identity.Birth = "" },
		"no-observation":  func(r *InitialMemberReadiness) { r.After = Observation{} },
		"no-capture-time": func(r *InitialMemberReadiness) { r.CapturedAt = time.Time{} },
		"uncertain":       func(r *InitialMemberReadiness) { r.Storage.Uncertain = true },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			plan, launches, o, location := initialReadinessFixture(t)
			d := &initialReadinessFixtureDriver{before: cloneReadinessObservation(o), after: cloneReadinessObservation(o), first: location, second: location, ship: "~zod"}
			got, err := CollectInitialMemberReadiness(context.Background(), plan, launches, "zod", d, func() time.Time { return time.Unix(1001, 0) })
			if err != nil {
				t.Fatal(err)
			}
			mutate(&got)
			if got.Validate(plan, launches) == nil {
				t.Fatal("altered readiness accepted")
			}
		})
	}
}

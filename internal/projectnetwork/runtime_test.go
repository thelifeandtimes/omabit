package projectnetwork

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func runtimeFixture(t *testing.T) (Plan, RuntimeBinding, Observation, time.Time) {
	t.Helper()
	plan, err := Build(pair(), Reservations{})
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := plan.Digest()
	resource := func(id string) ResourceIdentity {
		return ResourceIdentity{strings.Repeat(id, 64), strings.Repeat("f", 64)}
	}
	ns := NamespaceIdentity{"01234567-0123-0123-0123-0123456789ab", 4, 1000}
	b := RuntimeBinding{ProjectID: "project-a", Revision: 1, Generation: 3, PlanDigest: digest, EngineID: "private-engine", Bridge: resource("a"), Anchor: resource("b"), Namespace: ns, Members: []RuntimeMember{
		{"receiver", strings.Repeat("1", 32), resource("c")}, {"sender", strings.Repeat("2", 32), resource("d")},
	}}
	now := time.Date(2026, 10, 23, 10, 0, 0, 0, time.UTC)
	o := Observation{ProjectID: b.ProjectID, Revision: b.Revision, Generation: b.Generation, EngineID: b.EngineID, ObservedAt: now, Complete: true, Bridge: ResourceObservation{b.Bridge, true}, BridgeAttachments: []string{b.Anchor.ID}, Anchor: AnchorObservation{ResourceObservation: ResourceObservation{b.Anchor, true}, Running: true, Namespace: ns, BridgeID: b.Bridge.ID}}
	for _, m := range plan.Members {
		o.Anchor.Requested = append(o.Anchor.Requested, Publication{m.InternalHTTPPort, plan.BindAddress, m.HTTPPort, "tcp"})
	}
	o.Anchor.Active = append([]Publication(nil), o.Anchor.Requested...)
	for _, m := range b.Members {
		o.Members = append(o.Members, MemberObservation{InstanceID: m.InstanceID, ResourceObservation: ResourceObservation{m.Container, true}, Running: true, AnchorID: b.Anchor.ID, Namespace: ns})
	}
	return plan, b, o, now
}

func cloneRuntime[T any](v T) T {
	data, _ := json.Marshal(v)
	var result T
	_ = json.Unmarshal(data, &result)
	return result
}

func TestProjectRuntimeReadinessDoesNotMutateEvidence(t *testing.T) {
	p, b, o, now := runtimeFixture(t)
	before := cloneRuntime(o)
	got, err := AssessRuntime(p, b, o, now)
	if err != nil || got.State != "ready" || got.RecoveryRequired || len(got.Running) != 2 || len(got.Stopped) != 0 {
		t.Fatal(got, err)
	}
	if !reflect.DeepEqual(before, o) {
		t.Fatal("assessment mutated observations")
	}
	// Inspection order is immaterial; the stored canonical plan remains ordered.
	o.Members[0], o.Members[1] = o.Members[1], o.Members[0]
	o.Anchor.Active[0], o.Anchor.Active[1] = o.Anchor.Active[1], o.Anchor.Active[0]
	if next, err := AssessRuntime(p, b, o, now); err != nil || !reflect.DeepEqual(next, got) {
		t.Fatal(next, err)
	}
}

func TestProjectAnchorNamespaceReplacementNeverMakesOldMembersReady(t *testing.T) {
	p, b, o, now := runtimeFixture(t)
	// Same exact anchor ID, publications and all immutable settings; its netns changed.
	// This is the behavior measured in the private Docker restart experiments.
	o.Anchor.Namespace.Inode++
	got, err := AssessRuntime(p, b, o, now)
	if err != nil || got.State != "namespace-changed" || !got.RecoveryRequired || len(got.Running) != 2 || got.RecordedNamespace == got.ObservedNamespace {
		t.Fatal(got, err)
	}
	// Even members already in the new netns cannot silently update the saved generation.
	for i := range o.Members {
		o.Members[i].Namespace = o.Anchor.Namespace
	}
	if got, err = AssessRuntime(p, b, o, now); err != nil || got.State != "namespace-changed" || !got.RecoveryRequired {
		t.Fatal(got, err)
	}
	// A later, separately persisted replacement generation makes the old snapshot stale.
	b.Generation++
	if _, err = AssessRuntime(p, b, o, now); !errors.Is(err, ErrRuntimeObservation) {
		t.Fatal("old generation accepted", err)
	}
}

func TestProjectRuntimeDegradedStatesPreserveOwnership(t *testing.T) {
	cases := map[string]func(*Observation){
		"member-stopped": func(o *Observation) { o.Members[0].Running = false; o.Members[0].Namespace = NamespaceIdentity{} },
		"anchor-unavailable": func(o *Observation) {
			o.Anchor.Running = false
			o.Anchor.Namespace = NamespaceIdentity{}
			o.Anchor.Active = nil
			o.BridgeAttachments = nil
		},
		"namespace-changed":        func(o *Observation) { o.Anchor.Namespace.Inode++ },
		"member-namespace-changed": func(o *Observation) { o.Members[0].Namespace.Inode++ },
		"publication-unavailable":  func(o *Observation) { o.Anchor.Active = nil },
		"member-unavailable": func(o *Observation) {
			m := &o.Members[0]
			m.Present = false
			m.Resource.ContractDigest = ""
			m.Running = false
			m.Namespace = NamespaceIdentity{}
			m.AnchorID = ""
		},
		"bridge-unavailable": func(o *Observation) {
			o.Bridge.Present = false
			o.Bridge.Resource.ContractDigest = ""
			o.BridgeAttachments = nil
			o.Anchor.Present = false
			o.Anchor.Resource.ContractDigest = ""
			o.Anchor.Running = false
			o.Anchor.Namespace = NamespaceIdentity{}
			o.Anchor.BridgeID = ""
			o.Anchor.Requested = nil
			o.Anchor.Active = nil
		},
	}
	for state, change := range cases {
		t.Run(state, func(t *testing.T) {
			p, b, o, now := runtimeFixture(t)
			original := cloneRuntime(b)
			change(&o)
			got, err := AssessRuntime(p, b, o, now)
			if err != nil || got.State != state || got.RecoveryRequired != (state != "member-stopped") {
				t.Fatal(got, err)
			}
			if !reflect.DeepEqual(original, b) {
				t.Fatal("assessment adopted or discarded resources")
			}
			if state == "member-unavailable" && len(got.Stopped) != 0 {
				t.Fatal("missing resource inferred stopped")
			}
		})
	}
}

func TestProjectRuntimeRefusesIncompleteReplacedOrForeignEvidence(t *testing.T) {
	cases := map[string]func(*Observation){
		"partial":                     func(o *Observation) { o.Complete = false },
		"old-time":                    func(o *Observation) { o.ObservedAt = o.ObservedAt.Add(-31 * time.Second) },
		"future-time":                 func(o *Observation) { o.ObservedAt = o.ObservedAt.Add(6 * time.Second) },
		"engine":                      func(o *Observation) { o.EngineID = "different-engine" },
		"revision":                    func(o *Observation) { o.Revision++ },
		"generation":                  func(o *Observation) { o.Generation++ },
		"project":                     func(o *Observation) { o.ProjectID = "project-b" },
		"bridge":                      func(o *Observation) { o.Bridge.Resource.ID = strings.Repeat("e", 64) },
		"anchor":                      func(o *Observation) { o.Anchor.Resource.ID = strings.Repeat("e", 64) },
		"bridge-options":              func(o *Observation) { o.Bridge.Resource.ContractDigest = strings.Repeat("e", 64) },
		"anchor-options":              func(o *Observation) { o.Anchor.Resource.ContractDigest = strings.Repeat("e", 64) },
		"member-options":              func(o *Observation) { o.Members[0].Resource.ContractDigest = strings.Repeat("e", 64) },
		"member-container":            func(o *Observation) { o.Members[0].Resource.ID = strings.Repeat("e", 64) },
		"unknown-attachment":          func(o *Observation) { o.BridgeAttachments = append(o.BridgeAttachments, strings.Repeat("e", 64)) },
		"duplicate-attachment":        func(o *Observation) { o.BridgeAttachments = append(o.BridgeAttachments, o.BridgeAttachments[0]) },
		"missing-attachment":          func(o *Observation) { o.BridgeAttachments = nil },
		"missing-member":              func(o *Observation) { o.Members = o.Members[:1] },
		"duplicate-member":            func(o *Observation) { o.Members[1] = o.Members[0] },
		"foreign-member":              func(o *Observation) { o.Members[0].InstanceID = "foreign" },
		"foreign-anchor":              func(o *Observation) { o.Members[0].AnchorID = strings.Repeat("e", 64) },
		"foreign-bridge":              func(o *Observation) { o.Anchor.BridgeID = strings.Repeat("e", 64) },
		"missing-namespace":           func(o *Observation) { o.Members[0].Namespace = NamespaceIdentity{} },
		"missing-anchor-namespace":    func(o *Observation) { o.Anchor.Namespace = NamespaceIdentity{} },
		"active-absent-member":        func(o *Observation) { o.Members[0].Present = false; o.Members[0].Resource.ContractDigest = "" },
		"active-absent-anchor":        func(o *Observation) { o.Anchor.Present = false; o.Anchor.Resource.ContractDigest = "" },
		"stopped-with-namespace":      func(o *Observation) { o.Members[0].Running = false },
		"absent-bridge-active-anchor": func(o *Observation) { o.Bridge.Present = false; o.Bridge.Resource.ContractDigest = "" },
		"admin-port":                  func(o *Observation) { o.Anchor.Requested[0].InternalPort = 12321 },
		"wildcard":                    func(o *Observation) { o.Anchor.Requested[0].HostAddress = "0.0.0.0" },
		"ipv6":                        func(o *Observation) { o.Anchor.Requested[0].HostAddress = "::1" },
		"udp":                         func(o *Observation) { o.Anchor.Requested[0].Protocol = "udp" },
		"duplicate-publication":       func(o *Observation) { o.Anchor.Requested[1] = o.Anchor.Requested[0] },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			p, b, o, now := runtimeFixture(t)
			change(&o)
			got, err := AssessRuntime(p, b, o, now)
			if !errors.Is(err, ErrRuntimeObservation) || !reflect.DeepEqual(got, Assessment{}) {
				t.Fatal("untrusted evidence returned assessment", got, err)
			}
		})
	}
}

func TestProjectBindingMustComeFromCompleteReviewedOwnership(t *testing.T) {
	changes := []func(*RuntimeBinding){
		func(b *RuntimeBinding) { b.PlanDigest = strings.Repeat("e", 64) },
		func(b *RuntimeBinding) { b.Revision = 0 }, func(b *RuntimeBinding) { b.Generation = 0 },
		func(b *RuntimeBinding) { b.EngineID = "" }, func(b *RuntimeBinding) { b.ProjectID = "../other" },
		func(b *RuntimeBinding) { b.Namespace.BootID = "" }, func(b *RuntimeBinding) { b.Namespace.Device = 0 },
		func(b *RuntimeBinding) { b.Anchor.ID = b.Bridge.ID }, func(b *RuntimeBinding) { b.Members = b.Members[:1] },
		func(b *RuntimeBinding) { b.Members[0].RuntimeID = b.Members[1].RuntimeID },
		func(b *RuntimeBinding) { b.Members[0].Container = b.Members[1].Container },
		func(b *RuntimeBinding) { b.Members[0].Container = b.Anchor },
		func(b *RuntimeBinding) { b.Members[0], b.Members[1] = b.Members[1], b.Members[0] },
	}
	for _, change := range changes {
		p, b, o, now := runtimeFixture(t)
		change(&b)
		if got, err := AssessRuntime(p, b, o, now); !errors.Is(err, ErrRuntimeObservation) || !reflect.DeepEqual(got, Assessment{}) {
			t.Fatal(got, err)
		}
	}
}

func TestProjectRuntimeKeepsUnpublishedAndUnexpectedPortsUnavailable(t *testing.T) {
	for _, change := range []func(*Observation){
		func(o *Observation) { o.Anchor.Active = nil },
		func(o *Observation) { o.Anchor.Active[0].HostAddress = "0.0.0.0" },
		func(o *Observation) { o.Anchor.Active[0].HostAddress = "::1" },
		func(o *Observation) {
			o.Anchor.Active = append(o.Anchor.Active, Publication{12321, "127.0.0.1", 18321, "tcp"})
		},
		func(o *Observation) { o.Anchor.Active[1] = o.Anchor.Active[0] },
	} {
		p, b, o, now := runtimeFixture(t)
		change(&o)
		got, err := AssessRuntime(p, b, o, now)
		if err != nil || got.State != "publication-unavailable" || !got.RecoveryRequired {
			t.Fatal(got, err)
		}
	}
}

func TestProjectRuntimeNewBootAndMissingAnchorRemainRecovery(t *testing.T) {
	p, b, o, now := runtimeFixture(t)
	o.Anchor.Namespace.BootID = "11234567-0123-0123-0123-0123456789ab"
	// An identical inode number after a host boot cannot identify the prior netns.
	if got, err := AssessRuntime(p, b, o, now); err != nil || got.State != "namespace-changed" || !got.RecoveryRequired {
		t.Fatal(got, err)
	}
	o.Anchor = AnchorObservation{ResourceObservation: ResourceObservation{Resource: ResourceIdentity{ID: b.Anchor.ID}, Present: false}}
	o.BridgeAttachments = nil
	got, err := AssessRuntime(p, b, o, now)
	if err != nil || got.State != "anchor-unavailable" || !got.RecoveryRequired || len(got.Running) != 2 {
		t.Fatal(got, err)
	}
	if len(got.Stopped) != 0 {
		t.Fatal("missing anchor erased live members")
	}
}

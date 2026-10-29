package privatechain

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestMemberBirthRecordNewLaunchRequiresCapture(t *testing.T) {
	h, s := newMemberLaunchHarness(t, false)
	got, err := ExecuteMemberLaunch(context.Background(), h.p, h.w, s, h, h.commit, h.now)
	if err != nil || got.Version != 2 || got.Running == nil || got.Running.Birth == nil {
		t.Fatal("new launch lacks durable birth", err)
	}
	if got.Validate(h.p, h.w) != nil {
		t.Fatal("valid record refused")
	}
	for _, kind := range []string{"missing", "instance", "operation", "container", "contract", "empty-time", "capture-time", "process", "namespace", "pier", "alias", "legacy-with-birth"} {
		t.Run(kind, func(t *testing.T) {
			x := got.Clone()
			b := x.Running.Birth
			switch kind {
			case "missing":
				x.Running.Birth = nil
			case "instance":
				b.InstanceID = "other"
			case "operation":
				b.OperationID = "other"
			case "container":
				b.ContainerID = strings.Repeat("f", 64)
			case "contract":
				b.ContractDigest = strings.Repeat("f", 64)
			case "empty-time":
				b.EmptyObservedAt = x.Key.CheckedAt.Add(-time.Second)
			case "capture-time":
				b.CapturedAt = x.Running.Gateway.ObservedAt.Add(time.Second)
			case "process":
				b.After.StartedAt = b.After.StartedAt.Add(time.Second)
			case "namespace":
				b.After.Namespace.Inode++
			case "pier":
				b.Pier.Inode = 0
			case "alias":
				b.Pier.Inode = x.Artifact.Inode
				b.Pier.Birth = x.Artifact.Birth
			case "legacy-with-birth":
				x.Version = 1
			}
			if x.Validate(h.p, h.w) == nil {
				t.Fatal("changed capture accepted")
			}
		})
	}
	copy := got.Clone()
	copy.Running.Birth.Pier.Inode++
	if got.Running.Birth.Pier.Inode == copy.Running.Birth.Pier.Inode {
		t.Fatal("capture clone aliases")
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var decoded MemberLaunch
	if json.Unmarshal(raw, &decoded) != nil || !reflect.DeepEqual(got, decoded) || decoded.Validate(h.p, h.w) != nil {
		t.Fatal("capture serialization changed")
	}
}

func TestMemberBirthRecordHistoricalLaunchRemainsObservationOnly(t *testing.T) {
	h, s := newMemberLaunchHarness(t, false)
	got, err := ExecuteMemberLaunch(context.Background(), h.p, h.w, s, h, h.commit, h.now)
	if err != nil {
		t.Fatal(err)
	}
	got.Version = 1
	got.Running.Birth = nil
	if got.Validate(h.p, h.w) != nil {
		t.Fatal("original44 history invalidated")
	}
	raw, err := json.Marshal(got)
	if err != nil || strings.Contains(string(raw), "\"birth\":{") {
		t.Fatal("old history gained capture")
	}
	s.Version = 1
	h2, _ := newMemberLaunchHarness(t, false)
	if _, err = ExecuteMemberLaunch(context.Background(), h2.p, h2.w, s, h2, h2.commit, h2.now); err == nil || len(h2.events) != 0 {
		t.Fatal("old intent executed under new contract")
	}
}

package projectnetwork

import (
	"context"
	"reflect"
	"testing"
)

func TestRecoveryMemberCreationPreservesOriginalAuthority(t *testing.T) {
	for _, stopped := range []bool{false, true} {
		f := newAnchorRetirementFixture(t, stopped)
		retired := f.saved.Clone()
		for !retired.Complete() {
			var err error
			retired, err = ExecuteNextAnchorRetirement(context.Background(), f.history, retired, f, f.commit, f.clock)
			if err != nil {
				t.Fatal(err)
			}
		}
		old := retired.Clone()
		history := f.history.Clone()
		for _, m := range retired.Namespace.Drain.Review.Context.Members {
			c, err := SelectRecoveryMemberCreation(f.history, retired, m.Context.Member.InstanceID, "replacement-"+m.Context.Member.InstanceID)
			if err != nil || c.Validate(f.history, retired) != nil || c.Generation != m.Context.Member.Generation+1 || c.Storage != m.Context.Storage || c.Namespace == *c.OriginalAnchor.Namespace || c.Namespace != retired.Namespace.After.Runtime.Anchor.Namespace {
				t.Fatal("wrong successor contract", err)
			}
			for _, mutate := range []func(*RecoveryMemberCreation){
				func(c *RecoveryMemberCreation) { c.Generation++ }, func(c *RecoveryMemberCreation) { c.PreviousGeneration++ }, func(c *RecoveryMemberCreation) { c.PreviousOperationID = "foreign" },
				func(c *RecoveryMemberCreation) { c.Storage.Identity.Inode++ }, func(c *RecoveryMemberCreation) { c.RuntimeID = "foreign" }, func(c *RecoveryMemberCreation) { c.NamespaceOperationID = "foreign" },
				func(c *RecoveryMemberCreation) { c.RetirementOperationID = "foreign" }, func(c *RecoveryMemberCreation) { c.Namespace = *c.OriginalAnchor.Namespace }, func(c *RecoveryMemberCreation) { *c.OriginalAnchor.Namespace = c.Namespace },
			} {
				bad := c.Clone()
				mutate(&bad)
				if bad.Validate(f.history, retired) == nil {
					t.Fatal("forged replacement contract accepted")
				}
			}
			*c.OriginalAnchor.Namespace = c.Namespace
			if !reflect.DeepEqual(history, f.history) || !reflect.DeepEqual(old, retired) {
				t.Fatal("contract aliased original authority")
			}
		}
	}
}
func TestRecoveryMemberCreationRefusesPartialOrReusedIntent(t *testing.T) {
	f := newAnchorRetirementFixture(t, false)
	s := f.saved.Clone()
	instance := s.Members[0].InstanceID
	if _, err := SelectRecoveryMemberCreation(f.history, s, instance, "new-member"); err == nil {
		t.Fatal("unretired member accepted")
	}
	for !s.Complete() {
		var err error
		s, err = ExecuteNextAnchorRetirement(context.Background(), f.history, s, f, f.commit, f.clock)
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, op := range []string{"", s.OperationID, s.Namespace.OperationID, s.Namespace.Drain.OperationID, f.history.Launches[0].OperationID, f.history.Launches[0].Anchor.OperationID} {
		if _, err := SelectRecoveryMemberCreation(f.history, s, instance, op); err == nil {
			t.Fatal("reused operation accepted", op)
		}
	}
	if _, err := SelectRecoveryMemberCreation(f.history, s, "foreign-member", "new-member"); err == nil {
		t.Fatal("unknown member accepted")
	}
	missing := s.Clone()
	missing.Members[len(s.Members)-1].After.Piers = nil
	if _, err := SelectRecoveryMemberCreation(f.history, missing, instance, "new-member"); err == nil {
		t.Fatal("missing pier evidence accepted")
	}
}

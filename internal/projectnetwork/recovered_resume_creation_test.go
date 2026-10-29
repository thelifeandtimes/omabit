package projectnetwork

import (
	"reflect"
	"testing"
	"time"
)

func TestRecoveredResumeCreationPreservesExplicitNamespace(t *testing.T) {
	h, r, tail := recoveredHistoryFixture(t)
	history, err := AppendAnchorRecoveryEpoch(h, r)
	if err != nil {
		t.Fatal(err)
	}
	for i, next := range tail {
		history.Resumes = tail[:i]
		creation, err := SelectRecoveredResumeCreation(history, next.Retirement, next.OperationID)
		if err != nil || creation.Validate(history, next.Retirement) != nil {
			t.Fatal("post-recovery creation refused", err)
		}
		if creation.PreviousGeneration != next.PreviousGeneration || creation.Generation != next.PreviousGeneration+1 || creation.RetirementOperationID != next.Retirement.OperationID || creation.RecoveryOperationID != r.Replacement.OperationID || creation.NamespaceOperationID != r.Replacement.Retirement.Namespace.OperationID || creation.Namespace != r.Binding.Namespace || !reflect.DeepEqual(creation.OriginalAnchor, h.Launches[0].Anchor) {
			t.Fatal("history relabeled")
		}
		before := history.Clone()
		for _, mutate := range []func(*RecoveredResumeCreation){
			func(c *RecoveredResumeCreation) { c.Generation++ }, func(c *RecoveredResumeCreation) { c.PreviousGeneration++ }, func(c *RecoveredResumeCreation) { c.PreviousOperationID = "foreign" },
			func(c *RecoveredResumeCreation) { c.Storage.Identity.Inode++ }, func(c *RecoveredResumeCreation) { c.NamespaceOperationID = "foreign" }, func(c *RecoveredResumeCreation) { c.RecoveryOperationID = "foreign" },
			func(c *RecoveredResumeCreation) { c.RetirementOperationID = "foreign" }, func(c *RecoveredResumeCreation) { c.Namespace = *c.OriginalAnchor.Namespace }, func(c *RecoveredResumeCreation) { *c.OriginalAnchor.Namespace = c.Namespace },
		} {
			bad := creation.Clone()
			mutate(&bad)
			if bad.Validate(history, next.Retirement) == nil {
				t.Fatal("forged creation accepted")
			}
		}
		*creation.OriginalAnchor.Namespace = creation.Namespace
		if !reflect.DeepEqual(history, before) {
			t.Fatal("contract aliases historical authority")
		}
	}
}
func TestRecoveredResumeCreationRefusesChangedAuthority(t *testing.T) {
	h, r, tail := recoveredHistoryFixture(t)
	history, err := AppendAnchorRecoveryEpoch(h, r)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"no-recovery", "incomplete-retirement", "wrong-creator", "changed-process", "old-pier", "changed-binding", "early-stop", "old-operation", "pending-retirement-operation", "wrong-tail"} {
		t.Run(mode, func(t *testing.T) {
			selected := history.Clone()
			retired := cloneMemberRetirement(tail[0].Retirement)
			operation := tail[0].OperationID
			switch mode {
			case "no-recovery":
				selected = h.Clone()
			case "incomplete-retirement":
				retired.Phase = "removed"
				retired.After = nil
			case "wrong-creator":
				retired.Exit.Stop.LaunchOperationID = "foreign"
			case "changed-process":
				started := retired.Exit.Stop.Before.StartedAt.Add(time.Nanosecond)
				retired.Exit.Stop.Before.StartedAt = started
				retired.Exit.Observation.StartedAt = started
				retired.Before.StartedAt = started
			case "old-pier":
				retired.Exit.Stop.Storage.Identity.Inode++
			case "changed-binding":
				retired.Exit.Stop.Binding.Namespace.Inode++
			case "early-stop":
				retired.Exit.Stop.Before.CapturedAt = r.CapturedAt.Add(-time.Nanosecond)
				retired.Exit.Stop.Before.Runtime.ObservedAt = retired.Exit.Stop.Before.CapturedAt
			case "old-operation":
				operation = r.Replacement.Retirement.Namespace.OperationID
			case "pending-retirement-operation":
				operation = retired.OperationID
			case "wrong-tail":
				selected.Resumes = tail[:1]
			}
			if _, err := SelectRecoveredResumeCreation(selected, retired, operation); err == nil {
				t.Fatal("invalid authority accepted")
			}
		})
	}
}

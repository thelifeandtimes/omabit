package projectnetwork

import (
	"reflect"
	"testing"
)

func TestAnchorRecoveryContextValidatesWholeHistoryAndKeepsSnapshotsPrivate(t *testing.T) {
	p, launches, resumes, originals, ready, want, _ := anchorRecoveryReviewFixture(t, false)
	h := AnchorRecoveryHistory{Plan: p, Launches: launches, Resumes: resumes, Originals: originals, Current: ready}
	for _, fault := range []string{"unfinished-tail", "corrupt-tail", "foreign-binding", "changed-original"} {
		t.Run(fault, func(t *testing.T) {
			v := h.Clone()
			switch fault {
			case "unfinished-tail":
				v.Resumes[len(v.Resumes)-1].Phase = "start-dispatched"
				v.Resumes[len(v.Resumes)-1].After = nil
			case "corrupt-tail":
				v.Resumes[len(v.Resumes)-1].Retirement.Exit.Stop.Storage.Identity.Inode++
			case "foreign-binding":
				v.Current[0].Binding.Members[0].Container.ID = "foreign"
			case "changed-original":
				v.Originals[0].Storage.Identity.Inode++
			}
			if _, err := SelectAnchorRecoveryHistoryContext(v); err == nil {
				t.Fatal("accepted changed history after an earlier valid prefix")
			}
		})
	}
	first, err := SelectAnchorRecoveryHistoryContext(h)
	if err != nil {
		t.Fatal(err)
	}
	first.Binding.Members[0].InstanceID = "caller-change"
	first.Members[0].Context.Binding.Members[0].InstanceID = "caller-change"
	again, err := SelectAnchorRecoveryHistoryContext(h)
	if err != nil || !reflect.DeepEqual(again, want) {
		t.Fatal("returned context changed retained input", err)
	}
}

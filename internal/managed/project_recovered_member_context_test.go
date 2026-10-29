package managed

import (
	"fmt"
	"reflect"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

func recoveredContextObservation(t *testing.T, s *store, p storedProject, instance string) string {
	t.Helper()
	h, originals, _, err := projectRecoveryHistory(s.db, p.ID, false)
	g02Check(t, err)
	original := ""
	for i, r := range h.Originals {
		if r.InstanceID == instance {
			original = originals[i]
		}
	}
	c, err := projectnetwork.SelectRecoveryMemberContext(h, instance)
	g02Check(t, err)
	digest, err := p.Plan.Digest()
	g02Check(t, err)
	req := projectMemberContextReadinessRequest{OriginalReadinessOperationID: original, CreatorOperationID: c.Member.OperationID, RequestID: fmt.Sprintf("recovered-ready-%s-%d", instance, c.Member.Generation), ExpectedRevision: p.Revision, ReviewDigest: digest}
	op, _, err := s.acceptProjectMemberContextReadiness("actor", p.ID, instance, req)
	g02Check(t, err)
	_, err = s.beginProjectMemberContextReadiness(op.ID)
	g02Check(t, err)
	g02Check(t, s.completeProjectMemberContextReadiness(op.ID, contextReadinessProof(t, s, op.ID)))
	return op.ID
}
func TestAnchorRecoveryStoreOrdinaryLifecycleBetweenRecoveries(t *testing.T) {
	dir := privateDir(t)
	s, err := testOpenStore(dir)
	g02Check(t, err)
	p, req := recoveryStoreProject(t, s)
	op, _, err := s.acceptProjectAnchorRecovery("actor", p.ID, req)
	g02Check(t, err)
	j, err := loadProjectAnchorRecovery(s.db, op.ID)
	g02Check(t, err)
	completeRecoveryStoreFixture(t, s, j)
	old := currentStopHistory(t, s)
	instance := p.Members[0].InstanceID
	ready := recoveredContextObservation(t, s, p, instance)
	stopReq := projectMemberStopRequest{ReadinessOperationID: ready, RequestID: "stop-after-recovery", ExpectedRevision: p.Revision, ReviewDigest: req.ReviewDigest}
	stop, _, err := s.acceptProjectMemberStop("actor", p.ID, instance, stopReq)
	g02Check(t, err)
	_, err = s.beginProjectMemberStop(stop.ID)
	g02Check(t, err)
	before, after := contextStopObservations(t, s, ready)
	advanceStopFixture(t, s, p, stop.ID, "stop-dispatched", &before)
	// Keep the Stop uncertain; the separate observation records its clean exit.
	g02Check(t, s.interruptMemberStop(stop.ID))
	retained, err := loadProjectMemberStop(s.db, stop.ID)
	g02Check(t, err)
	if retained.Stop.Before == nil || after.StartedAt != before.StartedAt {
		t.Fatal("fixture lost current process")
	}
	resumedReq := retireStoppedGenerationFixture(t, s, retained)
	resumed, _, err := s.acceptProjectMemberResume("actor", p.ID, instance, resumedReq)
	g02Check(t, err)
	resumeThroughFixture(t, s, resumed.ID, "running")
	g02Check(t, s.finishProjectMemberResume(resumed.ID))
	g02Check(t, validateProjectMemberContextReadiness(s.db))
	g02Check(t, validateProjectAnchorRecoveries(s.db))
	// Historical readiness remains valid but cannot authorize another Stop now.
	oldReady, err := loadProjectMemberContextReadiness(s.db, ready)
	g02Check(t, err)
	_, _, err = contextReadinessJobSource(s.db, oldReady)
	g02Check(t, err)
	stopReq.RequestID = "stale-recovered-stop"
	if _, _, err = s.acceptProjectMemberStop("actor", p.ID, instance, stopReq); err == nil {
		t.Fatal("old recovered readiness authorized current Stop")
	}
	recoveredContextObservation(t, s, p, instance)
	h, _, _, err := projectRecoveryHistory(s.db, p.ID, true)
	g02Check(t, err)
	if len(h.Epochs) != 1 || len(h.Resumes) != 1 {
		t.Fatal("ordinary replacement lost recovery ancestry")
	}
	req.RequestID = "recover-after-ordinary"
	req.HistoryDigest = recoveryHistoryDigest(h)
	op, _, err = s.acceptProjectAnchorRecovery("actor", p.ID, req)
	g02Check(t, err)
	j, err = loadProjectAnchorRecovery(s.db, op.ID)
	g02Check(t, err)
	completeRecoveryStoreFixture(t, s, j)
	g02Check(t, validateProjectMemberContextReadiness(s.db))
	g02Check(t, validateProjectAnchorRecoveries(s.db))
	assertContextHistory(t, s, old)
	h, _, _, err = projectRecoveryHistory(s.db, p.ID, true)
	g02Check(t, err)
	if len(h.Epochs) != 2 || len(h.Epochs[1].Resumes) != 1 {
		t.Fatal("second recovery lost intervening ordinary creator")
	}
	g02Check(t, s.close())
	s, err = testOpenStore(dir)
	g02Check(t, err)
	defer s.close()
	g02Check(t, s.reconcileInterrupted())
	again, _, _, err := projectRecoveryHistory(s.db, p.ID, true)
	g02Check(t, err)
	if !reflect.DeepEqual(h, again) {
		t.Fatal("mixed history changed on reopen")
	}
	g02Check(t, validateProjectMemberContextReadiness(s.db))
}

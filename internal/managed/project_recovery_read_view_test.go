package managed

import "testing"

func TestRecoveryReadViewIsPrivateAndDiscardedBeforeNextRead(t *testing.T) {
	s := testStore(t)
	p, _ := recoveryStoreProject(t, s)
	view := projectRecoveryReadOnly(s.db)
	h, originals, current, err := projectRecoveryHistory(view, p.ID, true)
	g02Check(t, err)
	originalPath := h.Originals[0].Storage.Path
	originalID := originals[0]
	currentID := current[0]
	h.Originals[0].Storage.Path = "caller-change"
	originals[0] = "caller-change"
	current[0] = "caller-change"
	again, ids, currents, err := projectRecoveryHistory(view, p.ID, true)
	g02Check(t, err)
	if again.Originals[0].Storage.Path != originalPath || ids[0] != originalID || currents[0] != currentID {
		t.Fatal("caller changed retained read view")
	}
	// The earlier read-only pass is over. A changed receipt must be rejected by
	// every new pass; nothing on store or Server retains the old view.
	_, err = s.db.Exec("UPDATE operations SET runtime='{}' WHERE id=?", originalID)
	g02Check(t, err)
	if _, _, _, err = projectRecoveryHistory(projectRecoveryReadOnly(s.db), p.ID, true); err == nil {
		t.Fatal("fresh validation reused old evidence")
	}
	if _, _, _, err = projectRecoveryHistory(s.db, p.ID, true); err == nil {
		t.Fatal("ordinary uncached read reused old evidence")
	}
}

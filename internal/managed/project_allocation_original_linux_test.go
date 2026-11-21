//go:build linux && amd64

package managed

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectusage"
)

func originalAllocationAPIFixture(t *testing.T) (*Server, *allocationFixtureBackend, storedProject, projectusage.Request, map[string]string) {
	t.Helper()
	r, old, _, _ := projectQueueFixture(t)
	p, req, ids := readyStopProject(t, r.s)
	b := &allocationFixtureBackend{stopRecoveryFixtureBackend: &stopRecoveryFixtureBackend{expansionFixtureBackend: &expansionFixtureBackend{queuedProjectBackend: old}}}
	r.b = b
	return r, b, p, projectusage.Request{RunnerID: r.s.runnerID, ProjectID: p.ID, ExpectedRevision: p.Revision, ReviewDigest: req.ReviewDigest, IncludePrivateData: true}, ids
}

func TestProjectAllocationOriginalReadyHistoryDoesNotCreateExpansionOrRewriteRows(t *testing.T) {
	r, b, p, in, _ := originalAllocationAPIFixture(t)
	before := appAllRows(t, r.s.db)
	state, binding, err := r.s.projectAllocationSelection(t.Context(), in)
	g02Check(t, err)
	if binding.LedgerKind != projectusage.OriginalHistory || len(state.Members) != len(p.Plan.Members) {
		t.Fatal("original source not distinguished", binding)
	}
	for _, m := range state.Members {
		if m.Creator.Generation != 1 || m.Creator.Mode != "boot" || m.Initialized.Identity.Inode == 0 || m.Initialized.Path != m.Creator.Storage.Path {
			t.Fatal("initialized original identity missing")
		}
	}
	w := allocationCall(t, r, in, "operator")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var out projectusage.Observation
	g02Check(t, json.Unmarshal(w.Body.Bytes(), &out))
	if out.Validate() != nil || out.Binding != binding || b.reads.Load() != 1 || out.FullProjectBudgetKnown {
		t.Fatal("original observation changed", out)
	}
	if _, err = loadProjectExpansion(r.s.db, p.ID); err != sql.ErrNoRows {
		t.Fatal("read persisted an expansion", err)
	}
	if !reflect.DeepEqual(before, appAllRows(t, r.s.db)) {
		t.Fatal("original source read wrote history")
	}
}

func TestProjectAllocationOriginalRawProofAndReceiptChangesWithholdCensus(t *testing.T) {
	r, b, p, in, ids := originalAllocationAPIFixture(t)
	before := appAllRows(t, r.s.db)
	for _, variant := range []string{"raw-readiness", "receipt-fingerprint"} {
		t.Run(variant, func(t *testing.T) {
			id := ids[p.Members[0].InstanceID]
			var original []byte
			query, restore := "", ""
			if variant == "raw-readiness" {
				query = "SELECT body FROM project_member_readiness WHERE operation_id=?"
				restore = "UPDATE project_member_readiness SET body=? WHERE operation_id=?"
			} else {
				query = "SELECT fingerprint FROM operations WHERE id=?"
				restore = "UPDATE operations SET fingerprint=? WHERE id=?"
			}
			g02Check(t, r.s.db.QueryRow(query, id).Scan(&original))
			var originalValue any = original
			var changedValue any = append(append([]byte(nil), original...), byte(' '))
			if variant == "receipt-fingerprint" {
				originalValue = string(original)
				changedValue = string(changedValue.([]byte))
			}
			b.onRead = func() {
				_, err := r.s.db.Exec(restore, changedValue, id)
				g02Check(t, err)
			}
			w := allocationCall(t, r, in, "operator")
			_, err := r.s.db.Exec(restore, originalValue, id)
			g02Check(t, err)
			b.onRead = nil
			if w.Code != 409 || !bytes.Contains(w.Body.Bytes(), []byte("project_allocation_selection_changed")) || bytes.Contains(w.Body.Bytes(), []byte("/fixture")) || len(r.projectAllocations) != 0 {
				t.Fatal("changed original history escaped", w.Code, w.Body.String())
			}
			if !reflect.DeepEqual(before, appAllRows(t, r.s.db)) {
				t.Fatal("original history not restored")
			}
		})
	}
}

func TestProjectAllocationOriginalNeverAdoptsRetiredOrRecoveringMembers(t *testing.T) {
	for _, variant := range []string{"retired", "recovering"} {
		t.Run(variant, func(t *testing.T) {
			r, old, _, _ := projectQueueFixture(t)
			var p storedProject
			if variant == "retired" {
				p, _ = readyResumeProject(t, r.s)
			} else {
				var req projectAnchorRecoveryRequest
				p, req = recoveryStoreProject(t, r.s)
				_, _, err := r.s.acceptProjectAnchorRecovery("actor", p.ID, req)
				g02Check(t, err)
			}
			b := &allocationFixtureBackend{stopRecoveryFixtureBackend: &stopRecoveryFixtureBackend{expansionFixtureBackend: &expansionFixtureBackend{queuedProjectBackend: old}}}
			r.b = b
			digest, err := p.Plan.Digest()
			g02Check(t, err)
			in := projectusage.Request{RunnerID: r.s.runnerID, ProjectID: p.ID, ExpectedRevision: p.Revision, ReviewDigest: digest, IncludePrivateData: true}
			before := appAllRows(t, r.s.db)
			w := allocationCall(t, r, in, "operator")
			if w.Code != 409 || !bytes.Contains(w.Body.Bytes(), []byte("project_allocation_unsupported_phase")) || b.reads.Load() != 0 || !reflect.DeepEqual(before, appAllRows(t, r.s.db)) {
				t.Fatal("historical creator adopted", w.Code, w.Body.String())
			}
		})
	}
}

func TestProjectAllocationOriginalHistoryLimitsPrecedeBodyDecode(t *testing.T) {
	r, _, p, _, ids := originalAllocationAPIFixture(t)
	id := ids[p.Members[0].InstanceID]
	var original []byte
	g02Check(t, r.s.db.QueryRow("SELECT body FROM project_member_readiness WHERE operation_id=?", id).Scan(&original))
	_, err := r.s.db.Exec("UPDATE project_member_readiness SET body=? WHERE operation_id=?", strings.Repeat("x", maxAllocationJournalBytes+1), id)
	g02Check(t, err)
	tx, err := r.s.db.BeginTx(t.Context(), &sql.TxOptions{ReadOnly: true})
	g02Check(t, err)
	state, body, err := projectOriginalAllocationSelection(t.Context(), tx, p, r.s.runnerID)
	tx.Rollback()
	_, restoreErr := r.s.db.Exec("UPDATE project_member_readiness SET body=? WHERE operation_id=?", original, id)
	g02Check(t, restoreErr)
	if err == nil || state.Version != 0 || body != nil {
		t.Fatal("oversized invalid body decoded into storage", err)
	}
	f, ok := err.(*fault)
	if !ok || f.problem.Code != "project_allocation_history_limit" {
		t.Fatal("wrong bounded history refusal", err)
	}
}

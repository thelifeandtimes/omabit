package managed

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestGrubberySavedReceiptIndependentOfLifecycleSourceAndActor(t *testing.T) {
	s := testStore(t)
	record, _ := grubberyFixture(t, s)
	j := grubberyJournal{s}
	record, err := j.Create(t.Context(), record)
	g02Check(t, err)
	record = appAppendFixture(t, j, record, record.Plan.Steps[0], "dispatched")
	record = appAppendFixture(t, j, record, record.Plan.Steps[0], "uncertain")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := &Server{s: s, ctx: ctx, uid: uint32(os.Geteuid())}
	// A different current operator can read the original creator's receipt.
	// There is deliberately no runtime backend or source cache in this fixture.
	r.mu.Lock()
	done := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		done <- deskHTTPRequest(t, r, "GET", protocol.GrubberyAppOperationsPath+"/"+record.Plan.RequestID, nil)
	}()
	select {
	case got := <-done:
		r.mu.Unlock()
		var out protocol.GrubberyAppOperation
		if got.Code != 200 || json.Unmarshal(got.Body.Bytes(), &out) != nil || out.Validate() != nil || !reflect.DeepEqual(out.Record, record) {
			t.Fatal("saved lookup changed original uncertainty", got.Code, got.Body.String())
		}
	case <-time.After(2 * time.Second):
		r.mu.Unlock()
		t.Fatal("saved lookup waited for lifecycle mutex")
	}
	for _, tc := range []struct {
		method, id, role string
		status           int
	}{
		{"GET", record.Plan.RequestID, "observer", 403},
		{"POST", record.Plan.RequestID, "operator", 405},
		{"GET", strings.Repeat("f", 64), "operator", 404},
		{"GET", "bad", "operator", 400},
		{"GET", record.Plan.RequestID + "/advance", "operator", 400},
	} {
		w := httptest.NewRecorder()
		r.serveGrubberyAppReceipt(w, httptest.NewRequest(tc.method, protocol.GrubberyAppOperationsPath+"/"+tc.id, nil), tc.role)
		if w.Code != tc.status {
			t.Fatalf("%+v: status %d", tc, w.Code)
		}
	}
	again, err := j.Load(t.Context(), record.Plan.RequestID)
	g02Check(t, err)
	if !reflect.DeepEqual(again, record) {
		t.Fatal("receipt reads changed original history")
	}
}

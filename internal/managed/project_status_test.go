package managed

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestProjectStatusReadsQueuedAndInterruptedHistoryWithoutEffects(t *testing.T) {
	r, c, _ := testEndpoint(t, privateDir(t), &fakeBackend{})
	op, _ := acceptProjectFixture(t, r.s)
	r.s.portAvailable = func(string, uint16) bool { t.Fatal("status probed host port"); return false }
	for _, want := range []string{"queued", "interrupted"} {
		if want == "interrupted" {
			if err := r.s.reconcileInterrupted(); err != nil {
				t.Fatal(err)
			}
		}
		var before, after int
		r.s.db.QueryRow("SELECT total_changes()").Scan(&before)
		result, err := c.ProjectStatus(context.Background(), client.Identity{RunnerID: r.s.runnerID, Mode: r.s.mode}, op.ProjectID)
		if err != nil || result.Basis != "stored-operation-history" || result.RuntimeObservation != "not-performed" || len(result.Operations) != 2 {
			t.Fatal(result, err)
		}
		if result.Operations[0].ID != op.ID || result.Operations[0].Status != want || result.Operations[1].Kind != "register-project" {
			t.Fatal(result.Operations)
		}
		r.s.db.QueryRow("SELECT total_changes()").Scan(&after)
		if before != after {
			t.Fatal("read changed store")
		}
		raw, _ := json.Marshal(result)
		for _, forbidden := range []string{"runtime_id", "contract_digest", "cookie", `"message"`} {
			if strings.Contains(string(raw), forbidden) {
				t.Fatal("private body exposed", forbidden)
			}
		}
	}
	if _, err := r.s.db.Exec("UPDATE operations SET actor='foreign' WHERE id=?", op.ID); err != nil {
		t.Fatal(err)
	}
	if partial, err := r.s.projectStatus(context.Background(), op.ProjectID); err == nil || partial.Operations != nil {
		t.Fatal("changed actor accepted", partial, err)
	}
}

func TestProjectStatusObserverRouteAndUnknownProject(t *testing.T) {
	r, err := newServer(privateDir(t), &fakeBackend{}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	op, _ := acceptProjectFixture(t, r.s)
	line, key := grantLine(t, "observer")
	r.keys = filepath.Join(privateDir(t), "keys")
	if err = os.WriteFile(r.keys, []byte(line+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{"GET", protocol.ProjectStatusPath(op.ProjectID), 200},
		{"GET", protocol.ProjectStatusPath("missing-project"), 409},
		{"POST", protocol.ProjectStatusPath(op.ProjectID), 403},
		{"GET", protocol.ProjectStatusPath(op.ProjectID) + "/extra", 403},
		{"GET", "/v1/projects/a/b/status", 403},
	} {
		q := httptest.NewRequest(tc.method, tc.path, nil)
		q = q.WithContext(context.WithValue(q.Context(), peerKey{}, uint32(os.Geteuid())))
		q.Header.Set(access.Header, key)
		q.Header.Set(protocol.Header, "1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, q)
		if w.Code != tc.status {
			t.Fatal(tc, w.Code, w.Body.String())
		}
	}
}

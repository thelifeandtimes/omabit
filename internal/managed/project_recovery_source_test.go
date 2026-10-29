//go:build linux && amd64

package managed

import (
	"context"
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestProjectRecoverySourceMatchesReviewedHistoryWithoutEffects(t *testing.T) {
	r, b, p, req := recoveryQueueFixture(t)
	before := append([]string(nil), b.calls...)
	var changes int
	r.s.db.QueryRow("SELECT total_changes()").Scan(&changes)
	v, err := r.s.projectRecoverySource(context.Background(), p.ID)
	g02Check(t, err)
	h, originals, current, err := projectRecoveryHistory(r.s.db, p.ID, true)
	g02Check(t, err)
	if v.HistoryDigest != req.HistoryDigest || v.HistoryDigest != recoveryHistoryDigest(h) || v.Project.ReviewDigest != req.ReviewDigest || len(v.Members) != len(p.Members) || v.ActionAuthority != "not-granted" || v.RuntimeObservation != "not-performed" {
		t.Fatal(v)
	}
	for _, member := range v.Members {
		found := false
		for _, id := range originals {
			if id == member.OriginalReadinessOperationID {
				op, e := loadOperation(r.s.db, id)
				g02Check(t, e)
				if op.InstanceID != member.InstanceID {
					t.Fatal(member)
				}
				found = true
			}
		}
		if !found {
			t.Fatal("original missing", member)
		}
		if member.CurrentReadinessOperationID != "" {
			found = false
			for _, id := range current {
				if id == member.CurrentReadinessOperationID {
					found = true
				}
			}
			if !found {
				t.Fatal("current missing", member)
			}
		}
		op, e := loadOperation(r.s.db, member.CreatorOperationID)
		g02Check(t, e)
		if op.InstanceID != member.InstanceID {
			t.Fatal(member)
		}
	}
	var after int
	r.s.db.QueryRow("SELECT total_changes()").Scan(&after)
	if changes != after || !reflect.DeepEqual(before, b.calls) || r.updateWorkers != 0 {
		t.Fatal("read changed history or backend")
	}
	raw, _ := json.Marshal(v)
	for _, private := range []string{"container_id", "runtime_id", "cookie", "/home/", "/srv/", "contract_digest"} {
		if strings.Contains(string(raw), private) {
			t.Fatal(private)
		}
	}
}
func TestProjectRecoverySourceObserverPrerequisitesAndCorruption(t *testing.T) {
	for _, scenario := range []string{"ready", "not-launched", "not-observed", "unknown", "corrupt"} {
		t.Run(scenario, func(t *testing.T) {
			var r *Server
			var project string
			want := 200
			switch scenario {
			case "not-launched":
				server, _, id, _ := projectQueueFixture(t)
				r, project, want = server, id, 409
			case "not-observed":
				server, _, p, _ := readinessQueueFixture(t)
				r, project, want = server, p.ID, 409
			default:
				server, _, p, _ := recoveryQueueFixture(t)
				r, project = server, p.ID
			}
			if scenario == "unknown" {
				project = "missing-project"
				want = 409
			}
			if scenario == "corrupt" {
				_, err := r.s.db.Exec("UPDATE project_member_readiness SET body=? WHERE project_id=?", []byte(`{}`), project)
				g02Check(t, err)
				want = 500
			}
			r.uid = uint32(os.Geteuid())
			line, key := grantLine(t, "observer")
			r.keys = filepath.Join(privateDir(t), "keys")
			g02Check(t, os.WriteFile(r.keys, []byte(line+"\n"), 0600))
			q := httptest.NewRequest("GET", protocol.ProjectRecoverySourcePath(project), nil)
			q = q.WithContext(context.WithValue(q.Context(), peerKey{}, uint32(os.Geteuid())))
			q.Header.Set(access.Header, key)
			q.Header.Set(protocol.Header, "1")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, q)
			if w.Code != want || r.failed.Load() != (scenario == "corrupt") || r.updateWorkers != 0 {
				t.Fatal(scenario, w.Code, w.Body.String(), r.failed.Load())
			}
		})
	}
}

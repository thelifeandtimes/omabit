package managed

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestProjectInventoryConsistentReadPreservesClaims(t *testing.T) {
	s := testStore(t)
	for i, prefix := range []string{"a", "b"} {
		if _, _, err := s.registerProject("actor", projectRequest(t, "request-"+prefix, prefix, uint16(18080+10*i))); err != nil {
			t.Fatal(err)
		}
	}
	s.portAvailable = func(string, uint16) bool { t.Fatal("inventory probed a host port"); return false }
	var before, after int
	s.db.QueryRow("SELECT total_changes()").Scan(&before)
	r, err := s.projectInventory(context.Background())
	if err != nil || len(r.Projects) != 2 || r.RuntimeObservation != "not-performed" || r.Basis != "stored-reservations" {
		t.Fatal(r, err)
	}
	s.db.QueryRow("SELECT total_changes()").Scan(&after)
	if before != after {
		t.Fatal("read changed store")
	}
	if r.Projects[0].ID >= r.Projects[1].ID || len(r.Projects[0].Members) != 2 {
		t.Fatal("unordered/incomplete inventory")
	}
	if _, err = s.db.Exec("DELETE FROM project_members WHERE instance_id=?", "a-zod"); err != nil {
		t.Fatal(err)
	}
	if partial, err := s.projectInventory(context.Background()); err == nil || partial.Projects != nil {
		t.Fatal("corrupt claims returned", partial, err)
	}
}

func TestProjectInventoryAuthenticatedReadRoute(t *testing.T) {
	r, err := newServer(privateDir(t), &fakeBackend{}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	observer, id := grantLine(t, "observer")
	r.keys = filepath.Join(privateDir(t), "keys")
	if err := os.WriteFile(r.keys, []byte(observer+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, method, path, origin, version string
		peer                                bool
		code                                int
	}{
		{"observer", "GET", protocol.ProjectInventoryPath, "", "1", true, 200},
		{"anonymous", "GET", protocol.ProjectInventoryPath, "", "1", false, 403},
		{"origin", "GET", protocol.ProjectInventoryPath, "https://invalid.example", "1", true, 403},
		{"version", "GET", protocol.ProjectInventoryPath, "", "", true, 426},
		{"mutation", "POST", protocol.ProjectInventoryPath, "", "1", true, 403},
		{"suffix", "GET", protocol.ProjectInventoryPath + "/unreviewed", "", "1", true, 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.peer {
				q = q.WithContext(context.WithValue(q.Context(), peerKey{}, uint32(os.Geteuid())))
			}
			q.Header.Set(access.Header, id)
			q.Header.Set(protocol.Header, tc.version)
			q.Header.Set("Origin", tc.origin)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, q)
			if w.Code != tc.code {
				t.Fatal(w.Code, w.Body.String())
			}
			if tc.code == 200 {
				var out protocol.ProjectInventory
				if json.Unmarshal(w.Body.Bytes(), &out) != nil || out.Projects == nil || len(out.Projects) != 0 || out.RunnerID != r.s.runnerID {
					t.Fatal(out)
				}
			}
		})
	}
}

func TestProjectInventoryUnixClientRoundTrip(t *testing.T) {
	r, c, _ := testEndpoint(t, privateDir(t), &fakeBackend{})
	op, _, err := r.s.registerProject("operator", projectRequest(t, "one", "a", 18080))
	if err != nil {
		t.Fatal(err)
	}
	h, err := c.Hello(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.Projects(context.Background(), client.Identity{RunnerID: h.RunnerID, Mode: h.Mode})
	if err != nil || len(out.Projects) != 1 || out.Projects[0].ID != op.ProjectID {
		t.Fatal(out, err)
	}
}

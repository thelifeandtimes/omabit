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
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestPrivateProjectInventoryObserverNoRuntimeOrPrivateTokens(t *testing.T) {
	r, err := newServer(privateDir(t), &fakeBackend{}, time.Hour)
	g02Check(t, err)
	defer r.Close()
	r.s.portAvailable = func(string, uint16) bool { return true }
	op, _, err := r.s.registerPrivateProject("actor", privateProjectRequest(t, "one", "a", 18381))
	g02Check(t, err)
	p, _ := readPrivateProject(t, r.s, op.ProjectID)
	r.s.portAvailable = func(string, uint16) bool { t.Error("inventory probed host"); return false }
	line, key := grantLine(t, "observer")
	r.keys = filepath.Join(privateDir(t), "keys")
	g02Check(t, os.WriteFile(r.keys, []byte(line+"\n"), 0600))
	request := func() *httptest.ResponseRecorder {
		q := httptest.NewRequest("GET", protocol.PrivateChainProjectsPath, nil)
		q = q.WithContext(context.WithValue(q.Context(), peerKey{}, uint32(os.Geteuid())))
		q.Header.Set(access.Header, key)
		q.Header.Set(protocol.Header, "1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, q)
		return w
	}
	var before, after int
	g02Check(t, r.s.db.QueryRow("SELECT total_changes()").Scan(&before))
	w := request()
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var out privatechain.Inventory
	g02Check(t, json.Unmarshal(w.Body.Bytes(), &out))
	if len(out.Projects) != 1 || out.Projects[0].Binding != p.Binding || out.RuntimeAuthorized || out.RuntimeObservation != "not-performed" {
		t.Fatal("wrong inventory")
	}
	g02Check(t, r.s.db.QueryRow("SELECT total_changes()").Scan(&after))
	if before != after {
		t.Fatal("inventory changed saved state")
	}
	for _, secret := range []string{"runtime_id", "private_key", "cookie", p.Members[0].RuntimeID, p.Members[1].RuntimeID} {
		if strings.Contains(w.Body.String(), secret) {
			t.Fatal("private token escaped", secret)
		}
	}
	_, err = r.s.db.Exec("DELETE FROM private_project_members WHERE instance_id='a-zod'")
	g02Check(t, err)
	if w = request(); w.Code == 200 {
		t.Fatal("corrupt claims returned inventory")
	}
}

package managed

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type logFixture struct {
	fakeBackend
	calls   int
	fail    bool
	replace func()
}

func (b *logFixture) Logs(context.Context, runtimequalify.ManagedState) (protocol.RuntimeLogs, error) {
	b.calls++
	if b.fail {
		return protocol.RuntimeLogs{}, errors.New("PRIVATE-KEY")
	}
	if b.replace != nil {
		b.replace()
	}
	return protocol.RuntimeLogs{ContainerID: strings.Repeat("a", 64), Source: "docker", ObservedAt: time.Now().UTC(), Lines: []string{"boot: fixture output"}}, nil
}

func TestLogsAreOperatorOnlyRevisionBoundAndNotInSnapshots(t *testing.T) {
	s := testStore(t)
	if _, _, err := s.register("actor", registration("alpha", "register", 18080)); err != nil {
		t.Fatal(err)
	}
	v, err := loadInstance(s.db, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	v.Runtime.Runtime.Containers = []runtimequalify.ContainerRecord{{ID: strings.Repeat("a", 64), Kind: "http-boot"}}
	v.Public.RecoveryRequired = true // Failed/working containers must remain readable.
	save := func() {
		tx, err := s.db.Begin()
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		if err = saveInstance(tx, v); err != nil {
			t.Fatal(err)
		}
		if err = tx.Commit(); err != nil {
			t.Fatal(err)
		}
	}
	save()
	b := &logFixture{}
	r := &Server{s: s, b: b, uid: uint32(os.Geteuid())}
	call := func(rev uint64) *httptest.ResponseRecorder {
		body, _ := json.Marshal(protocol.LogsRequest{ExpectedRevision: rev})
		req := httptest.NewRequest("POST", "/v1/instances/alpha/logs", strings.NewReader(string(body)))
		req.Header.Set(protocol.Header, "1")
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(context.WithValue(req.Context(), peerKey{}, r.uid))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	if w := call(2); w.Code != 409 || b.calls != 0 {
		t.Fatal("stale read admitted")
	}
	if w := call(1); w.Code != 200 || !strings.Contains(w.Body.String(), "fixture output") {
		t.Fatal(w.Code, w.Body.String())
	}
	w := httptest.NewRecorder()
	r.serveLogs(w, httptest.NewRequest("POST", "/", nil), "observer", "alpha")
	if w.Code != 403 || b.calls != 1 {
		t.Fatal("observer read admitted")
	}
	b.fail = true
	if w := call(1); w.Code != 502 || strings.Contains(w.Body.String(), "PRIVATE") {
		t.Fatal("backend error exposed")
	}
	b.fail = false
	snap, err := s.snapshot()
	if err != nil {
		t.Fatal(err)
	}
	wire, _ := json.Marshal(snap)
	d := snap.Instances[0].RuntimeStatus
	if d.DefaultSnapshotMinutes != 60 || d.SnapshotMinutes == nil || *d.SnapshotMinutes != 2 {
		t.Fatal("snapshot confused desired and recorded runtime intervals")
	}
	if strings.Contains(string(wire), "fixture output") {
		t.Fatal("logs in fleet snapshot")
	}
	b.replace = func() {
		v.Runtime.Runtime.Containers[0].ID = strings.Repeat("b", 64)
		save()
	}
	if w := call(1); w.Code != 409 {
		t.Fatal("replacement accepted", w.Code)
	}
}

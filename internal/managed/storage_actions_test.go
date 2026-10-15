package managed

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func TestDeletionJournalReplayNameReuseAndLateObservation(t *testing.T) {
	s := testStore(t)
	if _, _, err := s.register("operator", registration("delete-me", "register", 18080)); err != nil {
		t.Fatal(err)
	}
	before, err := loadInstance(s.db, "delete-me")
	if err != nil {
		t.Fatal(err)
	}
	req := protocol.OperationRequest{Kind: "delete-pier", RequestID: "delete", ExpectedRevision: 1}
	_, _, err = s.accept("operator", "delete-me", req)
	requireFault(t, err, "invalid_request")
	req.ConfirmDelete = true
	op, _, err := s.accept("operator", "delete-me", req)
	if err != nil {
		t.Fatal(err)
	}
	op, v, err := s.beginOperation(op.ID)
	if err != nil {
		t.Fatal(err)
	}
	v.Runtime.Deletion = &runtimequalify.ManagedDeletion{Stage: "complete"}
	if err = s.runtimeUpdate(v.Public.ID, op.ID, v.Runtime); err != nil {
		t.Fatal(err)
	}
	if err = s.finish(op, v.Runtime, runtimequalify.ManagedObservation{State: "deleted", Health: "unknown"}, nil, false); err != nil {
		t.Fatal(err)
	}
	snapshot, err := s.snapshot()
	if err != nil || len(snapshot.Instances) != 0 {
		t.Fatal(snapshot, err)
	}
	got, replayed, err := s.accept("operator", "delete-me", req)
	if err != nil || !replayed || got.ID != op.ID || got.Status != "succeeded" {
		t.Fatal(got, err)
	}
	if err = s.observe(before, before.Runtime, runtimequalify.ManagedObservation{State: "uninitialized"}); err != nil {
		t.Fatal("late poll broke store", err)
	}
	replaced, _, err := s.register("operator", registration("delete-me", "replace", 18080))
	if err != nil || replaced.ConfigRevision != 2 {
		t.Fatal("name/port reuse did not advance revision", replaced, err)
	}
	if err = s.observe(before, before.Runtime, runtimequalify.ManagedObservation{State: "running"}); err != nil {
		t.Fatal("late old-generation poll broke store", err)
	}
	current, err := loadInstance(s.db, "delete-me")
	if err != nil || current.Runtime.ID == before.Runtime.ID || current.Public.ObservedState != "uninitialized" {
		t.Fatal("old observation affected replacement", err)
	}
	for _, kind := range []string{"start", "stop", "delete-pier"} {
		_, _, err = s.accept("operator", "delete-me", protocol.OperationRequest{Kind: kind, RequestID: "stale-" + kind, ExpectedRevision: 1, ConfirmDelete: kind == "delete-pier"})
		requireFault(t, err, "revision_conflict")
	}
	if _, replayed, err := s.accept("operator", "delete-me", req); err != nil || !replayed {
		t.Fatal("historical deletion replay lost", err)
	}
	if _, err = loadInstance(s.db, "delete-me"); err != nil {
		t.Fatal("historical replay deleted replacement", err)
	}
	req.ExpectedRevision = 2
	_, _, err = s.accept("operator", "delete-me", req)
	requireFault(t, err, "request_conflict")
}

func TestDeletionRejectsRunningBusyAndHandoffFencedInstances(t *testing.T) {
	for _, fault := range []string{"running", "busy", "handoff"} {
		t.Run(fault, func(t *testing.T) {
			s := testStore(t)
			_, _, err := s.register("actor", registration("one", "register", 18080))
			if err != nil {
				t.Fatal(err)
			}
			tx, err := s.db.Begin()
			if err != nil {
				t.Fatal(err)
			}
			v, err := loadInstance(tx, "one")
			if err != nil {
				t.Fatal(err)
			}
			switch fault {
			case "running":
				v.Public.DesiredState = "running"
				v.Public.ObservedState = "running"
			case "busy":
				v.Public.ActiveOperationID = "active"
			case "handoff":
				v.Runtime.Handoff = &protocol.Handoff{Direction: "out", State: "moved"}
			}
			if err = saveInstance(tx, v); err != nil {
				t.Fatal(err)
			}
			if err = tx.Commit(); err != nil {
				t.Fatal(err)
			}
			_, _, err = s.accept("actor", "one", protocol.OperationRequest{Kind: "delete-pier", RequestID: "delete", ExpectedRevision: 1, ConfirmDelete: true})
			if err == nil {
				t.Fatal("unsafe deletion accepted")
			}
		})
	}
}

type storageActionBackend struct {
	fakeBackend
	home string
}

func (b *storageActionBackend) StorageRoots() ([]pierstore.StorageRoot, error) {
	return pierstore.Roots(b.home)
}
func (b *storageActionBackend) RegisterStorageRoot(id, path string) (pierstore.StorageRoot, error) {
	return pierstore.RegisterRoot(b.home, id, path)
}
func (b *storageActionBackend) InspectExternalPier(ctx context.Context, path string) (pierstore.ExternalSource, error) {
	return pierstore.InspectExternal(ctx, b.home, path)
}
func (b *storageActionBackend) PierSources(ctx context.Context) ([]pierstore.Candidate, error) {
	return pierstore.Scan(ctx, b.home)
}

func TestStorageRootRPCReplayAndImportSourceConflict(t *testing.T) {
	b := &storageActionBackend{home: privateDir(t)}
	if err := pierstore.Init(b.home); err != nil {
		t.Fatal(err)
	}
	r, c, _ := testEndpoint(t, privateDir(t), b)
	ctx := context.Background()
	req := protocol.StorageRootRequest{ID: "disk", Path: privateDir(t), RequestID: "add-disk"}
	first, err := c.AddStorageRoot(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	bad := registration("unknown-location", "unknown-location", 18088)
	bad.PierPath = "@missing/fakeships/unknown/pier"
	if _, err = c.CreateFake(ctx, bad); err == nil {
		t.Fatal("unknown storage location registered")
	}
	again, err := c.AddStorageRoot(ctx, req)
	if err != nil || !reflect.DeepEqual(first, again) {
		t.Fatal("root add replay differs", err)
	}
	req.Path = privateDir(t)
	if _, err = c.AddStorageRoot(ctx, req); err == nil {
		t.Fatal("request identity reused for different storage")
	}
	recorder := httptest.NewRecorder()
	r.serveStorageActions(recorder, httptest.NewRequest(http.MethodPost, "/v1/storage-roots", strings.NewReader(`{}`)), "observer", "observer")
	if recorder.Code != http.StatusForbidden {
		t.Fatal("observer storage mutation", recorder.Code)
	}
	source := filepath.Join(privateDir(t), "source")
	if err = os.MkdirAll(filepath.Join(source, ".urb/chk"), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(filepath.Join(source, ".urb/log"), 0700); err != nil {
		t.Fatal(err)
	}
	preview, err := c.PreviewPierImport(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	request := protocol.ImportPierRequest{Source: source, EnrollPierRequest: protocol.EnrollPierRequest{ID: "reserved", Path: "fakeships/reserved/pier", Ship: "~zod", HTTPPort: 18080, Fingerprint: preview.Fingerprint, RequestID: "import-reviewed", ConfirmStopped: true}}
	accepted, err := c.ImportPier(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	waitOperation(t, c, accepted.ID, "succeeded")
	replayed, err := c.ImportPier(ctx, request)
	if err != nil || replayed.ID != accepted.ID {
		t.Fatal("import replay lost its journal identity", err)
	}
	if _, err = c.PreviewPierImport(ctx, source); err == nil {
		t.Fatal("claimed external source was offered twice")
	}
}

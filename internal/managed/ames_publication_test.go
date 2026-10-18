package managed

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func seedLegacyAmes(t *testing.T, s *store, id string, port uint16) storedInstance {
	t.Helper()
	s.mode = DevelopmentMode
	ship, kind, tier := "~sampel-palnet", "planet", "planets"
	// Use comet-shaped unique identities for additional reservations, since the
	// product's enrollment paths support moons, planets and comets.
	if id != "legacy" {
		ship, kind, tier = "~doznec-salbud-salwex-sogref--tasweb-navfun-batwet-sigfet", "comet", "comets"
	}
	location := pierstore.Location{Path: "livenet/" + tier + "/" + id + "/pier", Identity: pierstore.Identity{Device: 1, Inode: uint64(port), UrbInode: uint64(port) + 1}, MountIdentity: pierstore.DirectoryIdentity{Device: 1, Inode: uint64(port) + 2}}
	req := protocol.EnrollPierRequest{ID: id, Path: location.Path, Ship: ship, Fingerprint: pierstore.Fingerprint(location.Identity), HTTPPort: port, RequestID: "enroll-" + id, ConfirmStopped: true}
	op, err := s.enrollPier("actor", req, pierstore.Candidate{Path: location.Path, Kind: kind, Identity: location.Identity, MountIdentity: location.MountIdentity}, "legacy-enrollment-"+id)
	if err != nil {
		t.Fatal(err)
	}
	op, v, err := s.beginOperation(op.ID)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	v.Runtime.Prepared, v.Runtime.Initialized = true, true
	v.Runtime.Boot, v.Runtime.AmesPort = nil, 0 // historical unpublished enrollment
	v.Runtime.Runtime.RunID = v.Runtime.ID
	v.Runtime.Runtime.PillURL, v.Runtime.Runtime.PillSHA256, v.Runtime.Runtime.PillBytes = "historical-pill", strings.Repeat("e", 64), 123
	v.Runtime.Adoption.Uncertain, v.Runtime.Adoption.Verified, v.Runtime.Adoption.FinishedAt = false, true, &now
	v.Runtime.Comet.StartedAt, v.Runtime.Comet.FinishedAt = &now, &now
	v.Runtime.ExternalImport = &runtimequalify.ManagedExternalImport{Stage: "complete", Source: pierstore.ExternalSource{Path: "/historical-source/" + id, Identity: location.Identity, Fingerprint: strings.Repeat("d", 64)}}
	v.Runtime.Loom, v.Runtime.NativeControl = 29, true
	if err := s.runtimeUpdate(id, op.ID, v.Runtime); err != nil {
		t.Fatal(err)
	}
	if err := s.finish(op, v.Runtime, runtimequalify.ManagedObservation{State: "stopped", Health: "unknown"}, nil, false); err != nil {
		t.Fatal(err)
	}
	v, err = loadInstance(s.db, id)
	if err != nil {
		t.Fatal(err)
	}
	v.Public.Config.Loom, v.Public.Config.NativeControl = 29, true
	v.Public.AppliedRevision = 1
	saveRecoveryFixture(t, s, v)
	return v
}

func amesRequest() protocol.PublishAmesRequest {
	return protocol.PublishAmesRequest{RequestID: "publish", ExpectedRevision: 1, AmesPort: 34567, ConfirmNetwork: true}
}

func TestAmesPublicationAtomicMigrationReplayAndPreservation(t *testing.T) {
	s := testStore(t)
	before := seedLegacyAmes(t, s, "legacy", 18080)
	var originalReceipt []byte
	if err := s.db.QueryRow("SELECT body FROM operations WHERE request_id='enroll-legacy'").Scan(&originalReceipt); err != nil {
		t.Fatal(err)
	}
	s = rc11ReopenStore(t, s)
	unchanged, err := loadInstance(s.db, "legacy")
	if err != nil || !reflect.DeepEqual(unchanged, before) {
		t.Fatal("opening legacy state changed its contract", err)
	}
	checks := 0
	validate := func(_ context.Context, state runtimequalify.ManagedState) error {
		checks++
		if !reflect.DeepEqual(state, before.Runtime) {
			t.Fatal("validation did not use the original runtime contract")
		}
		return nil
	}
	op, replayed, err := s.publishAmes(context.Background(), "actor", "legacy", amesRequest(), validate)
	if err != nil || replayed || op.Status != "succeeded" || op.Kind != "publish-ames" || op.ConfigRevision != 2 || checks != 1 {
		t.Fatal(op, err)
	}
	after, err := loadInstance(s.db, "legacy")
	if err != nil {
		t.Fatal(err)
	}
	want := before
	want.Runtime, err = runtimequalify.AmesPublicationState(before.Runtime, 34567)
	if err != nil {
		t.Fatal(err)
	}
	want.RuntimeRevision++
	want.Public.Config.AmesPort, want.Public.AmesPublished, want.Public.ConfigRevision = 34567, true, 2
	want.Public.ObservedAt = after.Public.ObservedAt
	if !reflect.DeepEqual(after, want) || after.Public.DesiredState != "stopped" || after.Public.ObservedState != "stopped" || after.Public.AppliedRevision != 1 {
		t.Fatal("migration changed unrelated identity, settings, intent, storage or provenance")
	}
	var archived []byte
	if err := s.db.QueryRow("SELECT runtime FROM operations WHERE id=?", op.ID).Scan(&archived); err != nil {
		t.Fatal(err)
	}
	var oldRuntime runtimequalify.ManagedState
	if err := json.Unmarshal(archived, &oldRuntime); err != nil || !reflect.DeepEqual(oldRuntime, before.Runtime) {
		t.Fatal("old launch contract was not retained", err)
	}
	var retainedReceipt []byte
	if err := s.db.QueryRow("SELECT body FROM operations WHERE request_id='enroll-legacy'").Scan(&retainedReceipt); err != nil || string(retainedReceipt) != string(originalReceipt) {
		t.Fatal("enrollment history changed", err)
	}
	s = rc11ReopenStore(t, s)
	s.portAvailable = func(string, uint16) bool { return false }
	again, replayed, err := s.publishAmes(context.Background(), "actor", "legacy", amesRequest(), func(context.Context, runtimequalify.ManagedState) error {
		t.Fatal("replay performed live validation")
		return nil
	})
	if err != nil || !replayed || !reflect.DeepEqual(again, op) {
		t.Fatal("replay lost to stale revision or live port checks", again, err)
	}
	_, _, err = s.publishAmes(context.Background(), "other-actor", "legacy", amesRequest(), validate)
	requireFault(t, err, "revision_conflict")
	changed := amesRequest()
	changed.AmesPort++
	_, _, err = s.publishAmes(context.Background(), "actor", "legacy", changed, validate)
	requireFault(t, err, "request_conflict")
	s.portAvailable = func(string, uint16) bool { return true }
	seedLegacyAmes(t, s, "second", 18081)
	other, hit, err := s.publishAmes(context.Background(), "other-actor", "second", changed, func(context.Context, runtimequalify.ManagedState) error { return nil })
	if err != nil || hit || other.ID == op.ID || other.Actor != "other-actor" {
		t.Fatal("actor-scoped request ID was not independent", other, err)
	}
}

func TestAmesPublicationRefusalsLeaveStateAndReceiptsUntouched(t *testing.T) {
	for _, fault := range []string{"running", "desired-running", "active-operation", "recovery", "missing-provenance", "foreign-ship", "already-published", "retained-container", "uncertain", "validation", "host-port", "late-host-port", "configured-reservation", "runtime-reservation", "canceled", "no-confirmation", "stale-revision"} {
		t.Run(fault, func(t *testing.T) {
			s := testStore(t)
			v := seedLegacyAmes(t, s, "legacy", 18080)
			req := amesRequest()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			validate := func(context.Context, runtimequalify.ManagedState) error { return nil }
			switch fault {
			case "running":
				v.Public.ObservedState = "running"
			case "desired-running":
				v.Public.DesiredState = "running"
			case "active-operation":
				v.Public.ActiveOperationID = "other-operation"
			case "recovery":
				v.Public.RecoveryRequired = true
			case "missing-provenance":
				v.Runtime.Adoption = nil
			case "foreign-ship":
				v.Public.Ship = "~marzod"
			case "already-published":
				v.Runtime.Boot = &runtimequalify.ManagedBoot{}
			case "retained-container":
				v.Runtime.Runtime.Containers = []runtimequalify.ContainerRecord{{Kind: "comet-resume"}}
			case "uncertain":
				v.Runtime.MaintenanceUncertain = true
			case "validation":
				validate = func(context.Context, runtimequalify.ManagedState) error { return errors.New("foreign runtime") }
			case "host-port":
				s.portAvailable = func(kind string, port uint16) bool { return kind != "udp" || port != req.AmesPort }
			case "late-host-port":
				validate = func(context.Context, runtimequalify.ManagedState) error {
					s.portAvailable = func(kind string, port uint16) bool { return kind != "udp" || port != req.AmesPort }
					return nil
				}
			case "configured-reservation", "runtime-reservation":
				other := seedLegacyAmes(t, s, "second", 18081)
				other.Runtime.Boot = &runtimequalify.ManagedBoot{}
				if fault == "configured-reservation" {
					other.Public.Config.AmesPort = req.AmesPort
				} else {
					other.Runtime.AmesPort = req.AmesPort
				}
				saveRecoveryFixture(t, s, other)
			case "canceled":
				validate = func(context.Context, runtimequalify.ManagedState) error { cancel(); return nil }
			case "no-confirmation":
				req.ConfirmNetwork = false
			case "stale-revision":
				req.ExpectedRevision = 2
			}
			saveRecoveryFixture(t, s, v)
			if _, _, err := s.publishAmes(ctx, "actor", "legacy", req, validate); err == nil {
				t.Fatal("unsafe publication accepted")
			}
			after, err := loadInstance(s.db, "legacy")
			if err != nil || !reflect.DeepEqual(after, v) {
				t.Fatal("refusal changed the registration", err)
			}
			var count int
			if err := s.db.QueryRow("SELECT count(*) FROM operations WHERE request_id=?", req.RequestID).Scan(&count); err != nil || count != 0 {
				t.Fatal("refusal published a receipt", err)
			}
		})
	}
}

func TestAmesPublicationReceiptFailureRollsBackContractAndAllowsRetry(t *testing.T) {
	s := testStore(t)
	before := seedLegacyAmes(t, s, "legacy", 18080)
	if _, err := s.db.Exec(`CREATE TEMP TRIGGER fail_ames_receipt BEFORE INSERT ON operations WHEN NEW.request_id='publish' BEGIN SELECT RAISE(ABORT, 'injected receipt failure'); END`); err != nil {
		t.Fatal(err)
	}
	validate := func(context.Context, runtimequalify.ManagedState) error { return nil }
	if _, _, err := s.publishAmes(context.Background(), "actor", "legacy", amesRequest(), validate); err == nil {
		t.Fatal("receipt failure was ignored")
	}
	after, err := loadInstance(s.db, "legacy")
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("publication escaped the failed receipt transaction", err)
	}
	if _, err := s.db.Exec("DROP TRIGGER fail_ames_receipt"); err != nil {
		t.Fatal(err)
	}
	if op, hit, err := s.publishAmes(context.Background(), "actor", "legacy", amesRequest(), validate); err != nil || hit || op.ConfigRevision != 2 {
		t.Fatal("rolled-back request was not retryable", op, err)
	}
}

type amesValidationBackend struct {
	fakeBackend
	checks int
}

func (b *amesValidationBackend) ValidateAmesPublication(context.Context, runtimequalify.ManagedState) error {
	b.checks++
	return nil
}

func TestAmesPublicationEndpointCapabilityAuthorizationAndReplay(t *testing.T) {
	s := testStore(t)
	seedLegacyAmes(t, s, "legacy", 18080)
	b := &amesValidationBackend{}
	r := &Server{s: s, b: b, uid: uint32(os.Geteuid())}
	line, observer := grantLine(t, "observer")
	r.keys = filepath.Join(privateDir(t), "keys")
	if err := os.WriteFile(r.keys, []byte(line+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	call := func(method, path, body, grant string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req = req.WithContext(context.WithValue(req.Context(), peerKey{}, r.uid))
		req.Header.Set(protocol.Header, "1")
		req.Header.Set("Content-Type", "application/json")
		if grant != "" {
			req.Header.Set(access.Header, grant)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	var hello protocol.Hello
	if err := json.Unmarshal(call("GET", "/hello", "", "").Body.Bytes(), &hello); err != nil || !slices.Contains(hello.Features, protocol.AmesPublicationFeature) {
		t.Fatal("capability unavailable", err)
	}
	body, _ := json.Marshal(amesRequest())
	path := "/v1/instances/legacy/publish-ames"
	if w := call("POST", path, string(body), observer); w.Code != 403 || b.checks != 0 {
		t.Fatal("observer publication accepted", w.Code)
	}
	first := call("POST", path, string(body), "")
	second := call("POST", path, string(body), "")
	if first.Code != 201 || second.Code != 200 || first.Body.String() != second.Body.String() || b.checks != 1 || b.apply != 0 {
		t.Fatal("endpoint did not publish/replay synchronously", first.Code, first.Body, second.Code, second.Body)
	}
	var snapshot protocol.Snapshot
	if err := json.Unmarshal(call("GET", "/v1/snapshot", "", observer).Body.Bytes(), &snapshot); err != nil || !snapshot.Instances[0].AmesPublished || snapshot.Instances[0].DesiredState != "stopped" {
		t.Fatal("existing observer snapshot cannot see publication", err)
	}
	r.b = &fakeBackend{}
	if err := json.Unmarshal(call("GET", "/hello", "", "").Body.Bytes(), &hello); err != nil || slices.Contains(hello.Features, protocol.AmesPublicationFeature) {
		t.Fatal("unavailable validation advertised publication", err)
	}
}

package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimeprofile"
)

type runtimeProfilePreflightBackend struct {
	fakeBackend
	calls []string
	err   error
}

func (b *runtimeProfilePreflightBackend) VerifyRuntimeProfile(_ context.Context, id string) error {
	b.calls = append(b.calls, id)
	return b.err
}
func (*runtimeProfilePreflightBackend) InspectExternalPierWithProfile(context.Context, string, string) (pierstore.ExternalSource, error) {
	panic("runtime preflight must not inspect a pier")
}
func (*runtimeProfilePreflightBackend) InspectPierIdentityWithProfile(context.Context, string, string) (protocol.PierIdentity, error) {
	panic("runtime preflight must not inspect a pier")
}

func TestRuntimeProfilePreflightAuthorizesBeforeArtifactAccess(t *testing.T) {
	s := testStore(t)
	for _, tc := range []struct {
		name, role, method, id string
		missing                bool
		status, calls          int
	}{
		{"operator", "operator", "GET", runtimeprofile.JSONFixProfile, false, 200, 1},
		{"read-only", "monitor", "GET", runtimeprofile.JSONFixProfile, false, 403, 0},
		{"wrong-method", "operator", "POST", runtimeprofile.JSONFixProfile, false, 400, 0},
		{"unknown", "operator", "GET", "latest", false, 400, 0},
		{"empty", "operator", "GET", "", false, 400, 0},
		{"missing-artifact", "operator", "GET", runtimeprofile.JSONFixProfile, true, 409, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := &runtimeProfilePreflightBackend{}
			if tc.missing {
				b.err = errors.New("reviewed archive unavailable")
			}
			r := &Server{s: s, b: b}
			w := httptest.NewRecorder()
			r.serveRuntimeProfile(w, httptest.NewRequest(tc.method, "/v1/runtime-profiles/"+tc.id, nil), tc.role)
			if w.Code != tc.status || len(b.calls) != tc.calls {
				t.Fatal("preflight crossed authorization/availability boundary", w.Code, b.calls, w.Body.String())
			}
			if tc.calls != 0 && b.calls[0] != tc.id {
				t.Fatal("preflight selected a different artifact")
			}
			if w.Code == 200 {
				var got protocol.RuntimeProfilePreflight
				want, _ := runtimeprofile.Resolve(tc.id)
				if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || !got.Available || got.RunnerID != s.runnerID || got.Profile != want {
					t.Fatal("preflight omitted its runner/artifact binding", err, got)
				}
			}
		})
	}
}

func TestRuntimeProfilesPersistIndependentlyAndOldConfigurePreserves(t *testing.T) {
	dir := privateDir(t)
	s, err := testOpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { s.close() }()
	for n, id := range []string{"official", "patched"} {
		req := registration(id, "register-"+id, uint16(18080+n))
		if n == 1 {
			req.Config.RuntimeProfile = runtimeprofile.JSONFixProfile
		}
		if _, _, err := s.register("actor", req); err != nil {
			t.Fatal(err)
		}
	}
	var before []byte
	if err := s.db.QueryRow("SELECT body FROM instances WHERE id='official'").Scan(&before); err != nil {
		t.Fatal(err)
	}
	if err = s.close(); err != nil {
		t.Fatal(err)
	}
	s, err = testOpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	v, err := loadInstance(s.db, "patched")
	if err != nil || v.Public.Config.RuntimeProfile != runtimeprofile.JSONFixProfile || v.Runtime.RuntimeProfile != runtimeprofile.JSONFixProfile {
		t.Fatal("reopen lost pin", err)
	}
	oldRequest := v.Public.Config
	oldRequest.RuntimeProfile = ""
	got, err := s.configure("actor", "patched", protocol.ConfigureRequest{ExpectedRevision: v.Public.ConfigRevision, Config: oldRequest})
	if err != nil || got.Config.RuntimeProfile != runtimeprofile.JSONFixProfile {
		t.Fatal("old client reset pin", err)
	}
	for _, id := range []string{runtimeprofile.OfficialProfile, "latest"} {
		req := got.Config
		req.RuntimeProfile = id
		if _, err := s.configure("actor", "patched", protocol.ConfigureRequest{ExpectedRevision: got.ConfigRevision, Config: req}); err == nil {
			t.Fatal("unsupported downgrade accepted")
		}
	}
	var after []byte
	if err := s.db.QueryRow("SELECT body FROM instances WHERE id='official'").Scan(&after); err != nil || !bytes.Equal(before, after) {
		t.Fatal("peer was rewritten", err)
	}
	op, _, err := s.accept("actor", "patched", protocol.OperationRequest{Kind: "start", ExpectedRevision: got.ConfigRevision, RequestID: "start-pinned"})
	if err != nil {
		t.Fatal(err)
	}
	_, launch, err := s.beginOperation(op.ID)
	if err != nil || launch.Runtime.RuntimeProfile != runtimeprofile.JSONFixProfile {
		t.Fatal("operation lost binding", err)
	}
	status, err := s.snapshot()
	if err != nil {
		t.Fatal(err)
	}
	for _, i := range status.Instances {
		if i.ID == "patched" && (i.RuntimeStatus.RequestedRuntime == nil || i.RuntimeStatus.RequestedRuntime.Requirement != runtimeprofile.JSONFixRequirement || i.RuntimeStatus.LaunchRuntime != nil || i.RuntimeStatus.ObservedImageID != "") {
			t.Fatal("requested runtime was presented as observed", i.RuntimeStatus)
		}
	}
}

func TestRuntimeProfilesSchema65MigrationPreservesOriginalBytes(t *testing.T) {
	s := testStore(t)
	if _, _, err := s.register("actor", registration("legacy", "old", 18080)); err != nil {
		t.Fatal(err)
	}
	var before, after []byte
	if err := s.db.QueryRow("SELECT body FROM instances WHERE id='legacy'").Scan(&before); err != nil {
		t.Fatal(err)
	}
	// Synthetic previous schema marker; actual published-reader rehearsal is a
	// separate artifact qualification. No native process or pier is involved.
	if _, err := s.db.Exec("PRAGMA user_version=65"); err != nil {
		t.Fatal(err)
	}
	if err := s.initialize(); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow("SELECT body FROM instances WHERE id='legacy'").Scan(&after); err != nil || !bytes.Equal(before, after) {
		t.Fatal("migration rewrote original", err)
	}
	var version int
	if err := s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 66 {
		t.Fatal("missing older-reader fence", version, err)
	}
	var v storedInstance
	if err := json.Unmarshal(after, &v); err != nil || v.Runtime.RuntimeProfile != "" || v.Public.Config.RuntimeProfile != "" {
		t.Fatal("legacy encoding relabeled", err)
	}
	if err := s.initialize(); err != nil {
		t.Fatal("nonrepeatable migration", err)
	}
}

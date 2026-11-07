//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type sourceBackendFixture struct {
	deskBackendFixture
	unix, clay  string
	statusCalls int
	baseline    *deskwork.Manifest
	badStatus   bool
}

func (b *sourceBackendFixture) OpenManagedDesk(context.Context, runtimequalify.ManagedState, string) (deskRuntime, error) {
	return sourceRuntimeFixture{deskRuntimeFixture{&b.deskBackendFixture}, b}, nil
}

type sourceRuntimeFixture struct {
	deskRuntimeFixture
	b *sourceBackendFixture
}

func (d sourceRuntimeFixture) InspectSource(_ context.Context, desk, source string, base *deskwork.Manifest) (deskwork.SourceStatus, error) {
	b := d.b
	b.statusCalls++
	b.baseline = base
	if b.scenario == "read-fails" {
		return deskwork.SourceStatus{}, errors.New("unavailable")
	}
	if desk != "example" || source != "example" && source != "source-two" {
		return deskwork.SourceStatus{}, errors.New("wrong source selection")
	}
	unix, _ := deskwork.NewManifest([]deskwork.File{{Path: "hello.txt", SHA256: strings.Repeat(b.unix, 64)}})
	clay, _ := deskwork.NewManifest([]deskwork.File{{Path: "hello.txt", SHA256: strings.Repeat(b.clay, 64)}})
	baseline := clay
	if base != nil {
		baseline = *base
	}
	comparison, _ := deskwork.Compare(baseline, unix, clay)
	delta, _ := deskwork.Compare(clay, unix, clay)
	if base == nil && comparison.State != "synchronized" {
		comparison.State = "different"
	}
	if b.badStatus {
		comparison.State = "synchronized"
	}
	return deskwork.SourceStatus{Version: 1, Observation: deskwork.Observation{Exists: true, Revision: 2, ClayHash: "0v123", Publication: "private"}, ReadAt: time.Now().UTC(), Unix: unix, CanonicalUnix: unix, Clay: clay, Comparison: comparison, Differences: delta.UnixChanges, Autocommit: "unknown"}, nil
}
func TestDeskSourcePersistsOnlyConfirmedEqualityAndPreservesGates(t *testing.T) {
	b := &sourceBackendFixture{unix: "a", clay: "a"}
	r, intent := deskServerFixture(t, &b.deskBackendFixture)
	r.b = b
	q := protocol.DeskSourceRequest{Selection: deskSelection(intent)}
	policy, err := loadAppPolicy(r.s.db)
	g02Check(t, err)
	read := func() protocol.DeskSourceStatus {
		t.Helper()
		resp := deskHTTPRequest(t, r, "POST", protocol.DeskSourcePath, q)
		var out protocol.DeskSourceStatus
		if resp.Code != 200 || json.Unmarshal(resp.Body.Bytes(), &out) != nil || out.Validate() != nil {
			t.Fatal(resp.Code, resp.Body.String(), out.Validate())
		}
		return out
	}
	first := read()
	if first.State != "synchronized" || first.BaselineAt != nil || first.Autocommit != "unknown" {
		t.Fatal(first)
	}
	saved, err := r.s.sourceBaseline(q.Binding("example"))
	g02Check(t, err)
	if saved == nil || saved.Manifest.Digest != first.ClayDigest {
		t.Fatal(saved)
	}
	b.unix = "b"
	edited := read()
	if edited.State != "unix-changes" || edited.UnixChangeCount != 1 || edited.BaselineAt == nil {
		t.Fatal(edited)
	}
	b.unix = "a"
	b.clay = "c"
	if got := read(); got.State != "clay-changes" {
		t.Fatal(got)
	}
	b.unix = "b"
	if got := read(); got.State != "diverged" || got.ConflictCount != 1 {
		t.Fatal(got)
	}
	b.scenario = "read-fails"
	if resp := deskHTTPRequest(t, r, "POST", protocol.DeskSourcePath, q); resp.Code != 502 {
		t.Fatal("read failure reported as clean", resp.Code)
	}
	b.scenario = ""
	b.badStatus = true
	if resp := deskHTTPRequest(t, r, "POST", protocol.DeskSourcePath, q); resp.Code != 502 {
		t.Fatal("false equality accepted", resp.Code)
	}
	after, err := r.s.sourceBaseline(q.Binding("example"))
	g02Check(t, err)
	if !reflect.DeepEqual(saved, after) {
		t.Fatal("differences/failure overwrote shared baseline")
	}
	afterPolicy, err := loadAppPolicy(r.s.db)
	g02Check(t, err)
	if !reflect.DeepEqual(policy, afterPolicy) || b.calls.Load() != 0 {
		t.Fatal("source read changed permission or dispatched an effect")
	}
}
func TestDeskSourceRejectsChangedWorkflowAndRuntime(t *testing.T) {
	b := &sourceBackendFixture{unix: "b", clay: "a"}
	r, intent := deskServerFixture(t, &b.deskBackendFixture)
	r.b = b
	q := protocol.DeskSourceRequest{Selection: deskSelection(intent)}
	setting := deskwork.DefaultSetting(q.Selection.Target)
	setting.Desk.SourceName = "source-two"
	_, err := r.s.putDeskSetting("setup", deskwork.SettingUpdate{Target: q.Selection.Target, Desk: setting.Desk})
	g02Check(t, err)
	if resp := deskHTTPRequest(t, r, "POST", protocol.DeskSourcePath, q); resp.Code != 409 || b.statusCalls != 0 {
		t.Fatal("stale workflow accepted", resp.Code)
	}
	q.SettingsRevision = 1
	resp := deskHTTPRequest(t, r, "POST", protocol.DeskSourcePath, q)
	var out protocol.DeskSourceStatus
	if resp.Code != 200 || json.Unmarshal(resp.Body.Bytes(), &out) != nil || out.State != "different" || out.SourceName != "source-two" || b.baseline != nil {
		t.Fatal(resp.Code, resp.Body.String())
	}
	old := q.Selection.RuntimeID
	q.Selection.RuntimeID = strings.Repeat("c", 64)
	if resp := deskHTTPRequest(t, r, "POST", protocol.DeskSourcePath, q); resp.Code != 409 || b.statusCalls != 1 {
		t.Fatal("changed runtime accepted", resp.Code)
	}
	q.Selection.RuntimeID = old
	setting.Desk.Workflow = "urgit"
	setting.Desk.SourceName = ""
	_, err = r.s.putDeskSetting("setup", deskwork.SettingUpdate{Target: q.Selection.Target, ExpectedRevision: 1, Desk: setting.Desk})
	g02Check(t, err)
	q.SettingsRevision = 2
	if resp := deskHTTPRequest(t, r, "POST", protocol.DeskSourcePath, q); resp.Code != 400 || b.statusCalls != 1 {
		t.Fatal("native Urgit binding assumed", resp.Code)
	}
}

func TestDeskSourceRejectsObserverAndInjectedSourceBeforeRuntime(t *testing.T) {
	b := &sourceBackendFixture{unix: "a", clay: "a"}
	r, intent := deskServerFixture(t, &b.deskBackendFixture)
	r.b = b
	q := protocol.DeskSourceRequest{Selection: deskSelection(intent)}
	body, _ := json.Marshal(q)
	denied := httptest.NewRecorder()
	r.serveDeskSource(denied, httptest.NewRequest("POST", protocol.DeskSourcePath, bytes.NewReader(body)), "observer")
	if denied.Code != 403 {
		t.Fatal("observer reached source transport", denied.Code)
	}
	body = append(body[:len(body)-1], []byte(`,"source_name":"../../private"}`)...)
	out := httptest.NewRecorder()
	req := httptest.NewRequest("POST", protocol.DeskSourcePath, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.serveDeskSource(out, req, "operator")
	if out.Code != 400 || b.statusCalls != 0 {
		t.Fatal("unconfigured source read accepted", out.Code)
	}
}

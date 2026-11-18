//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

// This is a fresh guarded native HTTP adapter for the private test fixture. It
// deliberately does not claim qualification of the product's Docker/curl path.
// Each call closes its own actual owner session before evidence is returned.
func (f *nativeDeskFixture) appOwnerSession(ctx context.Context, run func(*eyre.Client) (grubbery.NativeResponse, error)) (out grubbery.NativeResponse, err error) {
	out.SessionCleanup = "not-created"
	if err = f.check(); err != nil {
		return out, err
	}
	port := f.HTTPPort
	if port == 0 {
		port = 18081
	}
	owner, err := eyre.New(fmt.Sprintf("http://127.0.0.1:%d", port), f.ship())
	if err != nil {
		return out, err
	}
	defer owner.Close()
	code, err := f.call(ctx, deskwork.FrontendOwnerCode(), nil)
	if err != nil {
		return out, err
	}
	raw := []byte(strings.TrimPrefix(strings.TrimSpace(code), "~"))
	code = ""
	defer clear(raw)
	out.SessionCleanup = "unconfirmed"
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		if owner.Logout(cleanup) == nil {
			out.SessionCleanup = "confirmed"
		}
	}()
	if err = owner.Login(ctx, raw); err != nil {
		return out, err
	}
	if err = f.check(); err != nil {
		return out, err
	}
	out, err = run(owner)
	out.SessionCleanup = "unconfirmed"
	if check := f.check(); check != nil {
		return out, check
	}
	return out, err
}
func (f *nativeDeskFixture) GrubberyRead(ctx context.Context, r grubbery.ReadRequest) (grubbery.NativeResponse, error) {
	if _, err := r.Request(); err != nil {
		return grubbery.NativeResponse{}, err
	}
	response, err := f.appOwnerSession(ctx, func(c *eyre.Client) (grubbery.NativeResponse, error) { return c.GrubberyRead(ctx, r) })
	// Retain one bounded private last-read body for decoder/contract failures.
	// No cookie, login code or app source is implicitly exported as a report.
	descriptor, _ := r.Descriptor()
	metadata, _ := json.Marshal(struct {
		Read     grubbery.ReadDescriptor
		Response grubbery.NativeResponse
	}{descriptor, response})
	if len(response.Body) <= 1<<20 {
		if e := os.WriteFile("/lab/managed-app-last-native-read.json", metadata, 0600); e != nil {
			return response, e
		}
		if e := os.WriteFile("/lab/managed-app-last-native-body", response.Body, 0600); e != nil {
			return response, e
		}
	}
	return response, err
}
func (f *nativeDeskFixture) GrubberyStep(ctx context.Context, p grubbery.Plan, id string, files []deskwork.SourceFile) (grubbery.NativeResponse, error) {
	q, err := p.Request(id, files)
	if err != nil {
		return grubbery.NativeResponse{}, err
	}
	clear(q.Body)
	return f.appOwnerSession(ctx, func(c *eyre.Client) (grubbery.NativeResponse, error) { return c.GrubberyStep(ctx, p, id, files) })
}

func nativeManagedGrubbery(t *testing.T, ctx context.Context, r *Server, f *nativeDeskFixture, frameworkID string, save func(string, any)) {
	t.Helper()
	continuity, err := f.SourceIdentity(ctx)
	g02Check(t, err)
	framework, err := loadDeskOperation(r.s.db, frameworkID)
	g02Check(t, err)
	last := framework.Steps[len(framework.Steps)-1]
	if last.Observation == nil {
		t.Fatal("missing exact framework source baseline")
	}
	actor := fmt.Sprintf("local-uid:%d", os.Geteuid())
	binding := grubbery.Binding{RunnerID: r.s.runnerID, Mode: r.s.mode, Actor: actor, Target: framework.Intent.Target, RuntimeID: framework.Intent.RuntimeID, ConfigRevision: framework.Intent.ConfigRevision, Continuity: continuity, FrameworkReceiptID: frameworkID, FrameworkBaseline: *last.Observation, AppID: "native-state"}
	// A maintained native JSON-state nexus, copied into this immutable bounded
	// candidate. No source directory is mounted into the running ship.
	files := []deskwork.SourceFile{{Path: "bill.json", Bytes: []byte(`{"state":"/state"}`)}, {Path: "nex/state.hoon", Bytes: nativeGrubberyStateNexus("managed-v1")}}
	dir := "/lab/managed-app-source"
	g02Check(t, os.Mkdir(dir, 0700))
	entries := []deskwork.File{}
	var size int64
	for _, file := range files {
		filename := filepath.Join(dir, file.Path)
		g02Check(t, os.MkdirAll(filepath.Dir(filename), 0700))
		g02Check(t, os.WriteFile(filename, file.Bytes, 0600))
		entries = append(entries, deskwork.File{Path: file.Path, Bytes: int64(len(file.Bytes)), SHA256: stopEvidenceDigest(file.Bytes)})
		size += int64(len(file.Bytes))
	}
	manifest, err := deskwork.NewManifest(entries)
	g02Check(t, err)
	stageID, requestID := stopEvidenceDigest([]byte("native-managed-stage-v1")), stopEvidenceDigest([]byte("native-managed-request-v1"))
	root := filepath.Join(r.s.directory, "desk-stages-v1")
	var bundle bytes.Buffer
	g02Check(t, deskwork.WriteBundle(ctx, dir, manifest, &bundle))
	setting, err := r.s.deskSetting(binding.Target)
	g02Check(t, err)
	_, err = deskwork.CacheStage(ctx, root, deskwork.StageIntent{Version: 1, ID: stageID, Target: binding.Target, SettingsRevision: setting.Revision, SourceDigest: manifest.Digest, Files: len(files), Bytes: size}, actor, &bundle)
	g02Check(t, err)
	policy, err := loadAppPolicy(r.s.db)
	g02Check(t, err)
	binding.PolicyRevision = policy.Revision
	r.mu.Lock()
	backend, err := r.nativeGrubberyBackend(ctx, binding, actor, "operator")
	if err != nil {
		r.mu.Unlock()
		t.Fatal(err)
	}
	before, err := backend.Inspect(ctx, binding)
	r.mu.Unlock()
	g02Check(t, err)
	plan, err := grubbery.NewPlan(binding, stageID, requestID, manifest, files, before)
	g02Check(t, err)
	save("managed-app-plan", plan)
	execution := backend.Execution(grubberyJournal{r.s})
	call := grubbery.Call{RunnerID: binding.RunnerID, Mode: binding.Mode, Actor: actor, Target: binding.Target, AppID: binding.AppID, RequestID: requestID, PlanDigest: plan.Digest}
	var record grubbery.Record
	// This private fixture has no concurrent lifecycle caller; retain the same
	// product guard across each complete journal/effect/after-observation call.
	r.mu.Lock()
	record, err = execution.Begin(ctx, call, plan)
	r.mu.Unlock()
	g02Check(t, err)
	for i, step := range plan.Steps {
		call.StepID = step.ID
		r.mu.Lock()
		record, err = execution.Advance(ctx, call)
		r.mu.Unlock()
		save(fmt.Sprintf("managed-app-phase-%02d", i), record)
		g02Check(t, err)
		for deadline := time.Now().Add(20 * time.Second); ; {
			state, stateErr := record.State(step.ID)
			g02Check(t, stateErr)
			if state == "observed" {
				break
			}
			if state == "refused" || time.Now().After(deadline) {
				t.Fatal("native managed phase remains unresolved", step.Kind, step.Path, state)
			}
			// Observe only. The original uncertain effect is never resubmitted.
			time.Sleep(250 * time.Millisecond)
			r.mu.Lock()
			record, err = execution.Reconcile(ctx, call)
			r.mu.Unlock()
			g02Check(t, err)
			save(fmt.Sprintf("managed-app-phase-%02d", i), record)
		}
	}
	if _, done, err := record.Current(); err != nil || !done {
		t.Fatal("native managed source not followed", err)
	}
	if record.Plan.Digest != plan.Digest {
		t.Fatal("native coordinator replaced original plan")
	}
	save("managed-app-source-followed", true)
	save("managed-app-boundary", "actual pinned kernel, typed native HTTP, real policy/SQLite/cache/phase evidence; synthetic lifecycle; no public app API, Docker transport, permits/routes/served UI/restart/peers")
}

var _ grubbery.NativeRuntime = (*nativeDeskFixture)(nil)

// A separate fixed kernel namespace, not a repair of any managed pending phase.
// Record the actual custom-mark directory behavior before claiming it is safe
// to skip a create or derive materialized directories from virtual source.
func nativeGrubberyMarkLayout(t *testing.T, ctx context.Context, f *nativeDeskFixture, owner *eyre.Client, save func(string, any)) {
	t.Helper()
	encoded, err := owner.ExportSession()
	g02Check(t, err)
	defer clear(encoded)
	var cookie struct {
		Name  string `json:"cookie_name"`
		Value string `json:"cookie_value"`
	}
	g02Check(t, json.Unmarshal(encoded, &cookie))
	defer func() { cookie.Value = "" }()
	log, err := os.OpenFile("/lab/managed-mar-layout-calls.jsonl", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	g02Check(t, err)
	defer log.Close()
	seq := 0
	appendEntry := func(v any) {
		t.Helper()
		raw, err := json.Marshal(v)
		g02Check(t, err)
		_, err = log.Write(append(raw, '\n'))
		g02Check(t, err)
		g02Check(t, log.Sync())
	}
	call := func(name, method, path, content string, body []byte) (int, []byte) {
		t.Helper()
		g02Check(t, f.check())
		seq++
		appendEntry(map[string]any{"sequence": seq, "name": name, "state": "dispatched", "method": method, "path": path, "bytes": len(body), "sha256": stopEvidenceDigest(body)})
		req, err := http.NewRequestWithContext(ctx, method, "http://127.0.0.1:18081"+path, bytes.NewReader(body))
		g02Check(t, err)
		req.GetBody = nil
		if content != "" {
			req.Header.Set("Content-Type", content)
		}
		req.AddCookie(&http.Cookie{Name: cookie.Name, Value: cookie.Value})
		response, err := f.client.Do(req)
		g02Check(t, err)
		raw, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
		response.Body.Close()
		g02Check(t, err)
		if len(raw) > 1<<20 {
			t.Fatal("mark layout response bound")
		}
		g02Check(t, f.check())
		appendEntry(map[string]any{"sequence": seq, "name": name, "state": "returned", "status": response.StatusCode, "bytes": len(raw), "sha256": stopEvidenceDigest(raw)})
		return response.StatusCode, raw
	}
	expect := func(name, method, path, content string, body []byte, status int) []byte {
		t.Helper()
		got, raw := call(name, method, path, content, body)
		if got != status {
			save("custom-mark-refusal", map[string]any{"name": name, "status": got, "response_sha256": stopEvidenceDigest(raw)})
			t.Fatal("custom mark layout refused", name, got)
		}
		return raw
	}
	nonce, err := randomID()
	g02Check(t, err)
	root := "/omabit-mar-contract-" + nonce
	code := root + "/code"
	expect("absent", "GET", "/grubbery/api/peek"+root+"/", "", nil, 404)
	expect("marker", "PUT", "/grubbery/api/make"+root+"/omabit-owner.json", "application/json", []byte(`{"purpose":"fresh bounded native mark-layout contract"}`), 201)
	expect("code", "PUT", "/grubbery/api/make"+code+"/", "application/json", []byte(`{"neck":"/code"}`), 201)
	beforeStatus, before := call("mar-before", "GET", "/grubbery/api/peek"+code+"/mar/?as=tree", "", nil)
	if beforeStatus != 200 && beforeStatus != 404 {
		t.Fatal("mark directory read unavailable", beforeStatus)
	}
	created, _ := call("create-mar-once", "PUT", "/grubbery/api/make"+code+"/mar/", "application/json", []byte(`{}`))
	if created != 201 && created != 409 {
		t.Fatal("mark directory create unavailable", created)
	}
	marAfter := expect("mar-after-create", "GET", "/grubbery/api/peek"+code+"/mar/?as=tree", "", nil, 200)
	raw, err := os.ReadFile("/artifacts/framework-source/gub/mar/hoon.hoon")
	g02Check(t, err)
	if stopEvidenceDigest(raw) != "fcec4e0a96c28b75dd8284b4b7e824c911442a706424b2ccf0b05cca7e857afe" {
		t.Fatal("custom mark fixture source changed")
	}
	expect("custom-source", "PUT", "/grubbery/api/make"+code+"/mar/custom.hoon?blot=%2Fhoon", "text/x-hoon", raw, 201)
	actual := expect("custom-source-bytes", "GET", "/grubbery/api/peek"+code+"/mar/custom.hoon", "", nil, 200)
	if !bytes.Equal(actual, raw) {
		t.Fatal("custom mark source bytes changed")
	}
	codeAfter := expect("code-after-custom", "GET", "/grubbery/api/peek"+code+"/?as=tree", "", nil, 200)
	marFinal := expect("mar-after-custom", "GET", "/grubbery/api/peek"+code+"/mar/?as=tree", "", nil, 200)
	build := expect("custom-build", "GET", "/grubbery/api/build"+code+"?all=1", "", nil, 200)
	report, err := grubbery.ParseBuildReport(build, code)
	g02Check(t, err)
	if report.Failed != 0 || report.Total != 5 {
		t.Fatal("custom mark compilation not complete/clean")
	}
	beforeValue := any(map[string]any{"status": beforeStatus, "sha256": stopEvidenceDigest(before)})
	if beforeStatus == 200 {
		beforeValue = json.RawMessage(before)
	}
	save("custom-mark-layout", map[string]any{"before_status": beforeStatus, "before": beforeValue, "create_status": created, "after_create": json.RawMessage(marAfter), "code_after_custom": json.RawMessage(codeAfter), "mar_after_custom": json.RawMessage(marFinal), "build": report, "custom_source_sha256": stopEvidenceDigest(raw), "calls": seq, "scope": "kernel diagnostic only; separate namespace, no managed receipt/ownership assertion"})
}

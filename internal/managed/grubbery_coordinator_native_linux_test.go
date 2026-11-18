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
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/appdev"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
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
	nativeManagedGrubberyRound(t, ctx, r, f, frameworkID, "managed-v1", "managed-app", false, save)
}

func TestMaintainedGrubberySamplesMatchFrozenNativeContract(t *testing.T) {
	for _, version := range []string{"1", "2"} {
		sample, err := appdev.ReadSample("grubbery-state-v" + version)
		g02Check(t, err)
		seen := false
		for _, file := range sample.Files {
			if file.Path == "nex/state.hoon" {
				seen = true
				if !bytes.Equal([]byte(file.Content), nativeGrubberyStateNexus("public-v"+version)) {
					t.Fatal("native fixture and maintained sample changed independently")
				}
			}
		}
		if !seen {
			t.Fatal("sample nexus absent")
		}
	}
}
func nativeManagedGrubberyRound(t *testing.T, ctx context.Context, r *Server, f *nativeDeskFixture, frameworkID, label, prefix string, public bool, save func(string, any)) grubbery.Record {
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
	files := []deskwork.SourceFile{{Path: "bill.json", Bytes: []byte(`{"state":"/state"}`)}, {Path: "nex/state.hoon", Bytes: nativeGrubberyStateNexus(label)}}
	dir := "/lab/managed-app-source-" + label
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
	stageID, requestID := stopEvidenceDigest([]byte("native-managed-stage-"+label)), stopEvidenceDigest([]byte("native-managed-request-"+label))
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

	var plan grubbery.Plan
	var execution grubbery.Execution
	var registry *collection.Registry
	if public {
		registry, err = collection.Open("/lab/collector")
		g02Check(t, err)
		review, e := registry.ReviewGrubberyAppSource(ctx, "native", r.s.runnerID, protocol.GrubberyAppReviewRequest{Selection: protocol.DeskSelection{Target: binding.Target, RuntimeID: binding.RuntimeID, ConfigRevision: binding.ConfigRevision}, FrameworkReceiptID: frameworkID, AppID: binding.AppID, StageID: stageID, RequestID: requestID})
		g02Check(t, e)
		plan = review.Plan
		binding = plan.Binding
	} else {
		r.mu.Lock()
		backend, e := r.nativeGrubberyBackend(ctx, binding, actor, "operator")
		if e != nil {
			r.mu.Unlock()
			t.Fatal(e)
		}
		before, e := backend.Inspect(ctx, binding)
		r.mu.Unlock()
		g02Check(t, e)
		plan, err = grubbery.NewPlan(binding, stageID, requestID, manifest, files, before)
		g02Check(t, err)
		execution = backend.Execution(grubberyJournal{r.s})
	}
	save(prefix+"-plan", plan)
	call := grubbery.Call{RunnerID: binding.RunnerID, Mode: binding.Mode, Actor: actor, Target: binding.Target, AppID: binding.AppID, RequestID: requestID, PlanDigest: plan.Digest}
	var record grubbery.Record
	if public {
		reply, e := registry.BeginGrubberyAppSource(ctx, "native", r.s.runnerID, protocol.GrubberyAppBeginRequest{Plan: plan})
		g02Check(t, e)
		record = reply.Record
	} else {
		r.mu.Lock()
		record, err = execution.Begin(ctx, call, plan)
		r.mu.Unlock()
		g02Check(t, err)
	}
	phaseCall := func(action string) {
		t.Helper()
		if public {
			reply, e := registry.GrubberyAppSourcePhase(ctx, "native", r.s.runnerID, requestID, action, protocol.GrubberyAppPhaseRequest{PlanDigest: plan.Digest, StepID: call.StepID})
			g02Check(t, e)
			record = reply.Record
		} else {
			r.mu.Lock()
			if action == "advance" {
				record, err = execution.Advance(ctx, call)
			} else {
				record, err = execution.Reconcile(ctx, call)
			}
			r.mu.Unlock()
			g02Check(t, err)
		}
	}
	for i, step := range plan.Steps {
		call.StepID = step.ID
		phaseCall("advance")
		save(fmt.Sprintf("%s-phase-%02d", prefix, i), record)
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
			phaseCall("reconcile")
			g02Check(t, err)
			save(fmt.Sprintf("%s-phase-%02d", prefix, i), record)
		}
	}
	if _, done, err := record.Current(); err != nil || !done {
		t.Fatal("native managed source not followed", err)
	}
	if record.Plan.Digest != plan.Digest {
		t.Fatal("native coordinator replaced original plan")
	}
	save(prefix+"-source-followed", true)
	boundary := "actual pinned kernel, typed native HTTP, real policy/SQLite/cache/phase evidence; synthetic lifecycle; no Docker transport, permits/routes/served UI/peers"
	if public {
		boundary += "; public runner API and pinned registry/client"
	} else {
		boundary += "; private coordinator only"
	}
	save(prefix+"-boundary", boundary)

	return record
}

func nativePublicGrubbery(t *testing.T, ctx context.Context, r *Server, f *nativeDeskFixture, frameworkID string, save func(string, any)) {
	t.Helper()
	first := nativeManagedGrubberyRound(t, ctx, r, f, frameworkID, "public-v1", "public-app-v1", true, save)
	n, err := first.Plan.Binding.Namespace()
	g02Check(t, err)
	// This fixed test-only state probe is deliberately separate from the product
	// source API. It has no configurable native route or command input.
	state := func(write bool, label bool, value string) string {
		t.Helper()
		response, err := f.appOwnerSession(ctx, func(owner *eyre.Client) (grubbery.NativeResponse, error) {
			encoded, e := owner.ExportSession()
			if e != nil {
				return grubbery.NativeResponse{}, e
			}
			defer clear(encoded)
			var cookie struct {
				Name  string `json:"cookie_name"`
				Value string `json:"cookie_value"`
			}
			if e = json.Unmarshal(encoded, &cookie); e != nil {
				return grubbery.NativeResponse{}, e
			}
			defer func() { cookie.Value = "" }()
			method, path, body := "GET", "/grubbery/api/peek"+n.DataRoot+"/state/state.json", []byte{}
			if label {
				path = "/grubbery/api/peek" + n.DataRoot + "/state/label.json"
			}
			if write {
				method = "POST"
				path = "/grubbery/api/poke" + n.DataRoot + "/state/state.json?blot=%2Fjson"
				body, _ = json.Marshal(value)
			}
			req, e := http.NewRequestWithContext(ctx, method, "http://127.0.0.1:18081"+path, bytes.NewReader(body))
			if e != nil {
				return grubbery.NativeResponse{}, e
			}
			req.GetBody = nil
			req.AddCookie(&http.Cookie{Name: cookie.Name, Value: cookie.Value})
			if write {
				req.Header.Set("Content-Type", "application/json")
			}
			client := *f.client
			client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
			resp, e := client.Do(req)
			if e != nil {
				return grubbery.NativeResponse{}, e
			}
			defer resp.Body.Close()
			raw, e := io.ReadAll(io.LimitReader(resp.Body, 4097))
			if e != nil || len(raw) > 4096 {
				return grubbery.NativeResponse{}, fmt.Errorf("fixed state response bound")
			}
			if resp.StatusCode != 200 {
				return grubbery.NativeResponse{}, fmt.Errorf("fixed state response status %d", resp.StatusCode)
			}
			return nativeManagedResponse(200, raw), nil
		})
		g02Check(t, err)
		if response.SessionCleanup != "confirmed" {
			t.Fatal("fixed state owner cleanup unconfirmed")
		}
		if write {
			return ""
		}
		var got string
		g02Check(t, json.Unmarshal(response.Body, &got))
		return got
	}
	if state(false, false, "") != "initial" {
		t.Fatal("initial managed state differs")
	}
	state(true, false, "managed-saved-entry")
	if state(false, false, "") != "managed-saved-entry" {
		t.Fatal("managed state poke unavailable")
	}
	save("public-app-v1-state-saved", true)
	second := nativeManagedGrubberyRound(t, ctx, r, f, frameworkID, "public-v2", "public-app-v2", true, save)
	if second.Plan.Epoch <= first.Plan.Epoch || second.Plan.Binding != first.Plan.Binding {
		t.Fatal("managed source successor changed namespace identity")
	}
	if state(false, false, "") != "managed-saved-entry" || state(false, true, "") != "public-v2" {
		t.Fatal("source update reset state or did not change code")
	}
	save("public-app-v2-state-and-code", true)
	restart := nativeGallRestart(t, ctx, f)
	save("public-app-same-pier-restart", restart)
	if state(false, false, "") != "managed-saved-entry" || state(false, true, "") != "public-v2" {
		t.Fatal("same-pier restart lost state or code")
	}
	state(true, false, "managed-entry-after-restart")
	if state(false, false, "") != "managed-entry-after-restart" {
		t.Fatal("state write after restart unavailable")
	}
	// The fixture has no background lifecycle observer. Reopen only this exact
	// task-owned store after all native source phases completed; opening must not
	// change either original record or issue any native request.
	r.wg.Wait()
	r.mu.Lock()
	directory := r.s.directory
	if err = r.s.close(); err != nil {
		r.mu.Unlock()
		t.Fatal(err)
	}
	r.s, err = testOpenStore(directory)
	r.mu.Unlock()
	g02Check(t, err)
	registry, err := collection.Open("/lab/collector")
	g02Check(t, err)
	for _, original := range []grubbery.Record{first, second} {
		reply, e := registry.GrubberyAppOperation(ctx, "native", r.s.runnerID, original.Plan.RequestID)
		g02Check(t, e)
		if !reflect.DeepEqual(reply.Record, original) {
			t.Fatal("reopen changed original app receipt")
		}
		replay, e := registry.GrubberyAppSourcePhase(ctx, "native", r.s.runnerID, original.Plan.RequestID, "advance", protocol.GrubberyAppPhaseRequest{PlanDigest: original.Plan.Digest, StepID: original.Plan.Steps[0].ID})
		g02Check(t, e)
		if !reflect.DeepEqual(replay.Record, original) {
			t.Fatal("old phase replay changed completed source")
		}
	}
	save("public-app-workflow", map[string]any{"source_versions": 2, "state_preserved_on_source_update": true, "new_code_label_observed": true, "same_pier_clean_restart": true, "state_preserved_on_restart": true, "write_after_restart": true, "original_receipts_after_store_reopen": true, "old_phase_replay_unchanged": true, "boundary": "actual public runner API/registry/client and pinned native HTTP; synthetic lifecycle and fixed kernel state probes; product Docker transport, pending-operation rebinding, CLI/native combination, permits/routes/assets/peers and installed upgrade unqualified"})
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

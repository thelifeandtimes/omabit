//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

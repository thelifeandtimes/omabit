//go:build linux && amd64

package managed

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/appdev"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// Native bootstrap only: no app-specific Grubbery writes, permit authoring or
// publication. Lifecycle is synthetic. The outer fresh-kernel supervisor owns
// process shutdown; a lost install reply only permits bounded reconciliation.
func TestNativeGrubberyBootstrap(t *testing.T) {
	f := nativeDeskGuard(t)
	var guard struct {
		WorkflowCase string `json:"workflow_case"`
		Inputs       struct {
			Commit string `json:"commit"`
			Digest string `json:"source_digest"`
		} `json:"framework_inputs"`
	}
	data, err := os.ReadFile("/guard.json")
	g02Check(t, err)
	g02Check(t, json.Unmarshal(data, &guard))
	framework, err := appdev.ReadFramework(appdev.GrubberyFramework)
	g02Check(t, err)
	if guard.WorkflowCase != "grubbery-bootstrap-v1" || guard.Inputs.Commit != framework.Commit || guard.Inputs.Digest != framework.Source.Digest {
		t.Fatal("explicit fresh pinned Grubbery bootstrap capsule required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	proof := map[string]any{"scope": "fresh kernel/bootstrap only; synthetic lifecycle; no app deployment/permits/publication"}
	save := func(name string, value any) {
		t.Helper()
		proof[name] = value
		data, err := json.MarshalIndent(proof, "", "  ")
		g02Check(t, err)
		g02Check(t, os.WriteFile("/lab/grubbery-bootstrap-proof.json", data, 0600))
		t.Log(name)
	}
	var mounts atomic.Int32
	f.auditEffect = func(body []byte) error {
		if strings.Contains(string(body), "+hood/mount") || strings.Contains(string(body), "%kiln-mount") {
			mounts.Add(1)
			return errors.New("Grubbery capsule refuses Unix mounts")
		}
		digest := sha256.Sum256(body)
		entry, err := json.Marshal(map[string]any{"sha256": hex.EncodeToString(digest[:]), "bytes": len(body), "unix_mount": false})
		if err != nil {
			return err
		}
		log, err := os.OpenFile("/lab/grubbery-native-effect-inputs.jsonl", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		defer log.Close()
		if _, err = log.Write(append(entry, '\n')); err != nil {
			return err
		}
		return log.Sync()
	}
	root, err := os.OpenRoot("/artifacts/framework-source")
	g02Check(t, err)
	defer root.Close()
	manifest, err := deskwork.SnapshotRoot(root)
	g02Check(t, err)
	g02Check(t, framework.VerifySource(manifest))
	save("framework-source", manifest)
	// A source directory outside Unix's mounts, confined to the fresh pier's
	// configured source alias. The supervisor never exposes a host checkout.
	source := filepath.Join(f.pier(), f.Desk)
	g02Check(t, os.Mkdir(source, 0700))
	for _, file := range manifest.Files {
		content, err := deskwork.ReadSourceFile(root, file)
		g02Check(t, err)
		path := filepath.Join(source, file.Path)
		g02Check(t, os.MkdirAll(filepath.Dir(path), 0700))
		g02Check(t, os.WriteFile(path, content, 0600))
	}
	g02Check(t, os.Mkdir("/lab/runner", 0700))
	s, err := testOpenStore("/lab/runner")
	g02Check(t, err)
	defer s.close()
	_, _, err = s.register("fixture", registration("alpha", "native-grubbery", 18081))
	g02Check(t, err)
	instance, err := loadInstance(s.db, "alpha")
	g02Check(t, err)
	instance.Public.ObservedState = "running"
	instance.Public.Resources = &protocol.ResourceMetrics{ContainerID: strings.Repeat("b", 64)}
	saveRecoveryFixture(t, s, instance)
	r := &Server{s: s, b: &nativeDeskBackend{&deskBackendFixture{}, f}, ctx: ctx, cancel: cancel, uid: uint32(os.Geteuid())}
	defer r.Close()
	listener, err := net.Listen("unix", "/lab/runner.sock")
	g02Check(t, err)
	g02Check(t, os.Chmod("/lab/runner.sock", 0600))
	server := &http.Server{Handler: r, ConnContext: func(ctx context.Context, _ net.Conn) context.Context {
		return context.WithValue(ctx, peerKey{}, uint32(os.Geteuid()))
	}}
	go server.Serve(listener)
	defer server.Close()
	registry, err := collection.Open("/lab/collector")
	g02Check(t, err)
	connection, err := registry.Enroll(ctx, "native", "/lab/runner.sock")
	g02Check(t, err)
	target := apppolicy.Target{InstanceID: "alpha", Desk: f.Desk}
	policy, err := s.putAppPolicy("fixture-owner", apppolicy.Update{Scope: "ship", Target: target.Ship(), Gates: apppolicy.Gates{CreateDesk: apppolicy.Allow, Commit: apppolicy.Inherit, Publish: apppolicy.Inherit}})
	g02Check(t, err)
	_, err = s.putAppPolicy("fixture-owner", apppolicy.Update{ExpectedRevision: policy.Revision, Scope: "desk", Target: target, Gates: apppolicy.Gates{CreateDesk: apppolicy.Inherit, Commit: apppolicy.Allow, Publish: apppolicy.Block}})
	g02Check(t, err)
	wait := func(id string) {
		t.Helper()
		for deadline := time.Now().Add(5 * time.Minute); ; {
			r.wg.Wait()
			op, err := registry.DeskOperation(ctx, "native", connection.RunnerID, id)
			g02Check(t, err)
			save("operation-"+id, op)
			state := op.Receipt.Steps[len(op.Receipt.Steps)-1].State
			if state == "observed" {
				return
			}
			if state != "uncertain" && state != "interrupted" || time.Now().After(deadline) {
				t.Fatal("bootstrap phase did not complete; preserve original receipt", state)
			}
			_, err = registry.ReconcileDeskOperation(ctx, "native", connection.RunnerID, id)
			g02Check(t, err)
			time.Sleep(time.Second)
		}
	}
	create, err := registry.PlanDeskCreation(ctx, "native", connection.RunnerID, "alpha", f.Desk)
	g02Check(t, err)
	_, err = registry.CreateDesk(ctx, "native", connection.RunnerID, create)
	g02Check(t, err)
	wait(create.Intent.ID)
	stage, err := registry.PlanDeskStage(ctx, "native", connection.RunnerID, "alpha", f.Desk, source)
	g02Check(t, err)
	_, err = registry.StageDesk(ctx, "native", connection.RunnerID, stage, source)
	g02Check(t, err)
	commit, err := registry.PlanDeskCommit(ctx, "native", connection.RunnerID, stage.Intent.ID)
	g02Check(t, err)
	save("commit-plan", commit)
	_, err = registry.CommitDesk(ctx, "native", connection.RunnerID, commit)
	g02Check(t, err)
	wait(commit.Intent.Intent.ID)
	autocommit, err := f.AutocommitActive(ctx)
	g02Check(t, err)
	if autocommit || mounts.Load() != 0 {
		t.Fatal("unexpected live Unix mount")
	}
	save("no-unix-mount", true)
	release, err := registry.PlanDeskRelease(ctx, "native", connection.RunnerID, "alpha", f.Desk, "activate-local", []deskwork.ReleaseCheck{})
	g02Check(t, err)
	save("activation-plan", release)
	_, err = registry.ReleaseDesk(ctx, "native", connection.RunnerID, release)
	g02Check(t, err)
	wait(release.Intent.Intent.ID)
	app, err := registry.InspectDeskApplication(ctx, "native", connection.RunnerID, "alpha", f.Desk)
	g02Check(t, err)
	save("application", app)
	live := map[string]bool{}
	for _, agent := range app.Observation.Application.Agents {
		live[agent.Name] = agent.Live
	}
	if !live["grubbery"] || !live["grub-client"] {
		t.Fatal("framework declared agents not both live")
	}
	owner := nativeGallOwner(t, ctx, f)
	defer owner.Close()
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 8*time.Second)
		defer stop()
		if owner.Logout(cleanup) != nil {
			t.Error("framework observation owner-session cleanup unconfirmed")
		}
	}()
	encoded, err := owner.ExportSession()
	g02Check(t, err)
	defer clear(encoded)
	var cookie struct {
		Name  string `json:"cookie_name"`
		Value string `json:"cookie_value"`
	}
	g02Check(t, json.Unmarshal(encoded, &cookie))
	// A fixed kernel-owned diagnostic read, independent of a working shell.
	// No generic HTTP operation is exposed in CLI/MCP or the product runtime.
	q, err := http.NewRequestWithContext(ctx, "GET", "http://127.0.0.1:18081/grubbery/api/build/code?all=1", nil)
	g02Check(t, err)
	q.AddCookie(&http.Cookie{Name: cookie.Name, Value: cookie.Value})
	cookie.Value = ""
	response, err := f.client.Do(q)
	g02Check(t, err)
	body, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	response.Body.Close()
	g02Check(t, err)
	if response.StatusCode != 200 || len(body) > 1<<20 || !json.Valid(body) {
		save("build-response-refusal", map[string]any{"status": response.StatusCode, "body": string(body)})
		t.Fatal("pinned kernel build observation unavailable")
	}
	var build struct {
		Namespace string `json:"namespace"`
		Total     int    `json:"total"`
		Failed    int    `json:"failed"`
		Files     []struct {
			Source string `json:"source"`
			Status string `json:"status"`
		} `json:"files"`
	}
	g02Check(t, json.Unmarshal(body, &build))
	save("kernel-build-report", json.RawMessage(body))
	if build.Namespace != "/code" || build.Total == 0 || len(build.Files) != build.Total {
		t.Fatal("kernel returned incomplete code build report")
	}
	save("bootstrap", fmt.Sprintf("pinned Gall agents and kernel diagnostics available; %d unrelated/required build failures still require explicit classification", build.Failed))
	// Keep this narrower than framework/app qualification, even with zero build
	// failures. App instances, permits, routes, state and peers are untested here.
	save("passed-bootstrap-only", true)
}

//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type managedAppNativeFixture struct {
	*deskBackendFixture
	server        *Server
	binding       grubbery.Binding
	app           deskwork.AppState
	observation   deskwork.Observation
	continuity    string
	marker        []byte
	reads, writes int
	lost          bool
}
type managedAppNativeRuntime struct{ f *managedAppNativeFixture }

func (f *managedAppNativeFixture) OpenManagedDesk(context.Context, runtimequalify.ManagedState, string) (deskRuntime, error) {
	return managedAppNativeRuntime{f}, nil
}
func (d managedAppNativeRuntime) ListDesks(context.Context) ([]string, error) {
	return []string{"grubbery"}, nil
}
func (d managedAppNativeRuntime) Create(context.Context, string) error {
	return errors.New("fixture refuses unrelated effect")
}
func (d managedAppNativeRuntime) Observe(context.Context, string) (deskwork.Observation, error) {
	return d.f.observation, nil
}
func (d managedAppNativeRuntime) SourceIdentity(context.Context) (string, error) {
	return d.f.continuity, nil
}
func (d managedAppNativeRuntime) Application(context.Context, string) (deskwork.AppState, error) {
	return d.f.app, nil
}
func nativeManagedResponse(status int, body []byte) grubbery.NativeResponse {
	return grubbery.NativeResponse{Attempted: true, SessionCleanup: "confirmed", Status: status, Body: append([]byte{}, body...), BodySHA256: stopEvidenceDigest(body)}
}
func (d managedAppNativeRuntime) GrubberyRead(_ context.Context, read grubbery.ReadRequest) (grubbery.NativeResponse, error) {
	d.f.reads++
	q, err := read.Request()
	if err != nil {
		return grubbery.NativeResponse{}, err
	}
	n, _ := d.f.binding.Namespace()
	if d.f.marker != nil && q.Path == "/grubbery/api/peek"+n.SourceRoot+"/" {
		raw, _ := json.Marshal(map[string]any{"neck": nil, "files": map[string]string{"omabit-owner.json": "mime"}, "dirs": map[string]any{}})
		return nativeManagedResponse(200, raw), nil
	}
	if d.f.marker != nil && q.Path == "/grubbery/api/peek"+n.SourceRoot+"/omabit-owner.json" {
		return nativeManagedResponse(200, d.f.marker), nil
	}
	return nativeManagedResponse(404, []byte("Not found")), nil
}
func (d managedAppNativeRuntime) GrubberyStep(ctx context.Context, p grubbery.Plan, id string, files []deskwork.SourceFile) (grubbery.NativeResponse, error) {
	r, err := (grubberyJournal{d.f.server.s}).Load(ctx, p.RequestID)
	if err != nil {
		return grubbery.NativeResponse{}, err
	}
	state, _ := r.State(id)
	if state != "dispatched" {
		return grubbery.NativeResponse{}, errors.New("effect preceded durable dispatch")
	}
	q, err := p.Request(id, files)
	if err != nil {
		return grubbery.NativeResponse{}, err
	}
	n, _ := p.Binding.Namespace()
	if q.Method != "PUT" || q.Path != "/grubbery/api/make"+n.SourceRoot+"/omabit-owner.json" {
		return grubbery.NativeResponse{}, errors.New("fixture only exercises first marker effect")
	}
	d.f.writes++
	d.f.marker = append([]byte{}, q.Body...)
	if d.f.lost {
		d.f.lost = false
		return grubbery.NativeResponse{}, errors.New("injected native response loss")
	}
	return nativeManagedResponse(201, []byte("Created")), nil
}

func managedGrubberyFixture(t *testing.T) (*Server, *managedAppNativeFixture, grubbery.Plan, []deskwork.SourceFile) {
	t.Helper()
	base := &deskBackendFixture{}
	r, _ := deskServerFixture(t, base)
	record, files := grubberyFixture(t, r.s)
	b := record.Plan.Binding
	b.Actor = fmt.Sprintf("local-uid:%d", os.Geteuid())
	f := &managedAppNativeFixture{deskBackendFixture: base, server: r, binding: b, observation: b.FrameworkBaseline, continuity: b.Continuity}
	f.app = deskwork.AppState{Zest: "live", SourceRevision: b.FrameworkBaseline.Revision, ExpectedAgents: []string{"grub-client", "grubbery"}, Agents: []deskwork.AppAgent{{Name: "grub-client", Live: true}, {Name: "grubbery", Live: true}}, FrontendMode: "none", FrontendState: "unavailable"}
	r.b = f
	_, err := r.s.putAppPolicy("setup", apppolicy.Update{Scope: "desk", Target: b.Target, Gates: apppolicy.Gates{Commit: apppolicy.Allow, Publish: apppolicy.Block, CreateDesk: apppolicy.Inherit}})
	g02Check(t, err)
	// A synthetic supported activation receipt, separate from framework commit.
	activation := releaseRecordFixture(t)
	activation.Actor = "fixture"
	activation.Intent.Intent = deskwork.Intent{Version: 1, ID: strings.Repeat("4", 64), Target: b.Target, RuntimeID: b.RuntimeID, ConfigRevision: b.ConfigRevision, Action: "activate-local", Baseline: b.FrameworkBaseline}
	activation.Review = deskwork.ReleaseReview{Version: 1, Action: "activate-local", Observation: b.FrameworkBaseline, Application: f.app, DeskSetting: deskwork.DefaultSetting(b.Target), ShipSetting: deskwork.DefaultSetting(b.Target.Ship()), Continuity: b.Continuity}
	activation.Review.Application.Zest = "dead"
	activation.Review.Application.Agents = []deskwork.AppAgent{{Name: "grub-client"}, {Name: "grubbery"}}
	g02Check(t, activation.Review.Seal())
	activation.Intent.Continuity = b.Continuity
	activation.Intent.ReviewDigest = activation.Review.Digest
	activation.Intent.ChecksDigest = deskwork.ReleaseChecksDigest(activation.Checks)
	receipt, _, err := r.s.createDeskRelease(activation)
	g02Check(t, err)
	receipt, err = r.s.appendDeskStep(receipt, deskStep(receipt, "dispatched"))
	g02Check(t, err)
	receipt, err = r.s.appendDeskStep(receipt, deskStep(receipt, "acknowledged"))
	g02Check(t, err)
	_, err = r.s.appendDeskStep(receipt, deskwork.Step{Kind: "activate-local", State: "observed", At: time.Now().UTC(), Observation: &b.FrameworkBaseline, Application: &f.app})
	g02Check(t, err)
	// A private, access-bound stage created through the same bounded reader and
	// bundle format as the public staging handler. No live Unix mount exists.
	source := privateDir(t)
	var size int64
	for _, file := range files {
		g02Check(t, os.MkdirAll(filepath.Dir(filepath.Join(source, file.Path)), 0700))
		g02Check(t, os.WriteFile(filepath.Join(source, file.Path), file.Bytes, 0600))
		size += int64(len(file.Bytes))
	}
	var bundle bytes.Buffer
	g02Check(t, deskwork.WriteBundle(t.Context(), source, record.Plan.Source.Manifest, &bundle))
	root := filepath.Join(r.s.directory, "desk-stages-v1")
	g02Check(t, os.Mkdir(root, 0700))
	_, err = deskwork.CacheStage(t.Context(), root, deskwork.StageIntent{Version: 1, ID: record.Plan.StageID, Target: b.Target, SourceDigest: record.Plan.Source.Manifest.Digest, Files: len(files), Bytes: size}, b.Actor, &bundle)
	g02Check(t, err)
	backend, err := r.nativeGrubberyBackend(t.Context(), b, b.Actor, "operator")
	g02Check(t, err)
	before, err := backend.Inspect(t.Context(), b)
	g02Check(t, err)
	p, err := grubbery.NewPlan(b, record.Plan.StageID, record.Plan.RequestID, record.Plan.Source.Manifest, files, before)
	g02Check(t, err)
	return r, f, p, files
}
func managedAppCall(p grubbery.Plan) grubbery.Call {
	return grubbery.Call{RunnerID: p.Binding.RunnerID, Mode: p.Binding.Mode, Actor: p.Binding.Actor, Target: p.Binding.Target, AppID: p.Binding.AppID, RequestID: p.RequestID, PlanDigest: p.Digest, StepID: p.Steps[0].ID}
}

func TestManagedGrubberyNativeJournalRecoverySurvivesReopenAndMissingCache(t *testing.T) {
	r, f, p, _ := managedGrubberyFixture(t)
	b, err := r.nativeGrubberyBackend(t.Context(), p.Binding, p.Binding.Actor, "operator")
	g02Check(t, err)
	e := b.Execution(grubberyJournal{r.s})
	call := managedAppCall(p)
	_, err = e.Begin(t.Context(), call, p)
	g02Check(t, err)
	f.lost = true
	original, err := e.Advance(t.Context(), call)
	g02Check(t, err)
	state, _ := original.State(call.StepID)
	if state != "uncertain" || f.writes != 1 {
		t.Fatal("native dispatch not fenced", state)
	}
	// Retain the original SQLite history, reopen through the normal reader and
	// make the exact source cache unavailable. Recovery must remain read-only.
	dir := r.s.directory
	g02Check(t, r.s.close())
	r.s, err = testOpenStore(dir)
	g02Check(t, err)
	cache := filepath.Join(dir, "desk-stages-v1", p.StageID)
	g02Check(t, os.Rename(cache, cache+"-unavailable"))
	policy, err := loadAppPolicy(r.s.db)
	g02Check(t, err)
	_, err = r.s.putAppPolicy("setup", apppolicy.Update{ExpectedRevision: policy.Revision, Scope: "desk", Target: p.Binding.Target, Gates: apppolicy.Gates{Commit: apppolicy.Block, Publish: apppolicy.Block, CreateDesk: apppolicy.Inherit}})
	g02Check(t, err)
	b, err = r.nativeGrubberyBackend(t.Context(), p.Binding, p.Binding.Actor, "operator")
	g02Check(t, err)
	e = b.Execution(grubberyJournal{r.s})
	reads := f.reads
	replay, err := e.Advance(t.Context(), call)
	g02Check(t, err)
	if !reflect.DeepEqual(replay, original) || f.reads != reads {
		t.Fatal("saved replay contacted native runtime")
	}
	reconciled, err := e.Reconcile(t.Context(), call)
	g02Check(t, err)
	state, _ = reconciled.State(call.StepID)
	if state != "observed" || f.writes != 1 || !reflect.DeepEqual(original.Events, reconciled.Events[:len(original.Events)]) {
		t.Fatal("exact native recovery changed original effect", state)
	}
	history, err := (grubberyJournal{r.s}).History(t.Context(), p.Binding)
	g02Check(t, err)
	if len(history) != 1 || !reflect.DeepEqual(history[0], reconciled) {
		t.Fatal("trusted history lost original record")
	}
}

func TestManagedGrubberyGuardRefusesDriftAndRevocationBeforeAppIO(t *testing.T) {
	for _, kind := range []string{"actor", "reader", "runner", "mode", "config", "container", "policy", "pier", "framework", "not-live", "missing-agent", "missing-activation", "unresolved-framework", "management-fence"} {
		t.Run(kind, func(t *testing.T) {
			r, f, p, _ := managedGrubberyFixture(t)
			actor, role, b := p.Binding.Actor, "operator", p.Binding
			switch kind {
			case "actor":
				actor = "local-uid:99999"
			case "reader":
				role = "reader"
			case "runner":
				b.RunnerID = "other"
			case "mode":
				b.Mode = "other"
			case "config":
				b.ConfigRevision++
			case "container":
				b.RuntimeID = strings.Repeat("9", 64)
			case "policy":
				policy, err := loadAppPolicy(r.s.db)
				g02Check(t, err)
				_, err = r.s.putAppPolicy("setup", apppolicy.Update{ExpectedRevision: policy.Revision, Scope: "desk", Target: b.Target, Gates: apppolicy.Gates{Commit: apppolicy.Block, Publish: apppolicy.Block, CreateDesk: apppolicy.Inherit}})
				g02Check(t, err)
			case "pier":
				f.continuity = strings.Repeat("9", 64)
			case "framework":
				f.observation.ClayHash = "0v999"
			case "not-live":
				f.app.Zest = "held"
			case "missing-agent":
				f.app.Agents[0].Live = false
			case "missing-activation":
				_, err := r.s.db.Exec("DELETE FROM desk_releases WHERE id=?", strings.Repeat("4", 64))
				g02Check(t, err)
			case "unresolved-framework":
				i := deskIntentFixture()
				i.ID = strings.Repeat("6", 64)
				i.Target = b.Target
				i.Action = "commit"
				i.SourceDigest = p.Source.Manifest.Digest
				i.Baseline = b.FrameworkBaseline
				_, _, err := r.s.createDeskOperation("fixture", i)
				g02Check(t, err)
			case "management-fence":
				r.updateFence.Held = true
			}
			reads := f.reads
			backend, err := r.nativeGrubberyBackend(t.Context(), b, actor, role)
			if kind == "policy" && err == nil {
				_, err = backend.Inspect(t.Context(), b)
			}
			if err == nil || f.writes != 0 || f.reads != reads {
				t.Fatal("changed authority reached app I/O", err)
			}
		})
	}
}

func TestManagedGrubberyUnfinishedStageCannotBeDiscarded(t *testing.T) {
	r, f, p, _ := managedGrubberyFixture(t)
	b, err := r.nativeGrubberyBackend(t.Context(), p.Binding, p.Binding.Actor, "operator")
	g02Check(t, err)
	e := b.Execution(grubberyJournal{r.s})
	call := managedAppCall(p)
	_, err = e.Begin(t.Context(), call, p)
	g02Check(t, err)
	q := protocol.DeskStageDiscardRequest{SourceDigest: p.Source.Manifest.Digest}
	reads := f.reads
	denied := deskHTTPRequest(t, r, "DELETE", protocol.DeskStagesPath+"/"+p.StageID, q)
	if denied.Code != 409 || f.reads != reads || f.writes != 0 {
		t.Fatal("discard lost pending source or contacted app", denied.Code)
	}
	cache := filepath.Join(r.s.directory, "desk-stages-v1", p.StageID)
	if _, err = os.Stat(cache); err != nil {
		t.Fatal("pending stage disappeared", err)
	}
	abandoned, err := e.Abandon(t.Context(), call)
	g02Check(t, err)
	allowed := deskHTTPRequest(t, r, "DELETE", protocol.DeskStagesPath+"/"+p.StageID, q)
	if allowed.Code != 200 {
		t.Fatal("resolved cache could not be discarded", allowed.Code, allowed.Body.String())
	}
	replay, err := e.Abandon(t.Context(), call)
	g02Check(t, err)
	if !reflect.DeepEqual(replay, abandoned) || f.reads != reads || f.writes != 0 {
		t.Fatal("abandonment replay needed discarded cache or native work")
	}
}

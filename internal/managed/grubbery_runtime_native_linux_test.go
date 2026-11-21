//go:build linux && amd64

package managed

import (
	"context"
	"errors"
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

// Synthetic control-plane/runtime metadata and inert native marker transport.
// Exact process observation and transport fencing have separate adapter tests.
type appRuntimeRecoveryFixture struct {
	*managedAppNativeFixture
	process     grubbery.RuntimeProcess
	unavailable bool
}
type appRuntimeRecoveryModel struct {
	managedAppNativeRuntime
	f *appRuntimeRecoveryFixture
}
type approvedAppRuntimeModel struct {
	appRuntimeRecoveryModel
	expected grubbery.RuntimeProcess
	readOnly bool
}

func (f *appRuntimeRecoveryFixture) OpenManagedDesk(_ context.Context, _ runtimequalify.ManagedState, id string) (deskRuntime, error) {
	if f.unavailable || id != f.process.ContainerID {
		return nil, grubbery.ErrAdmission
	}
	return appRuntimeRecoveryModel{managedAppNativeRuntime{f.managedAppNativeFixture}, f}, nil
}
func (d appRuntimeRecoveryModel) GrubberyProcess(context.Context) (grubbery.RuntimeProcess, error) {
	if d.f.unavailable {
		return grubbery.RuntimeProcess{}, grubbery.ErrAdmission
	}
	return d.f.process, nil
}
func (d appRuntimeRecoveryModel) GrubberyRecoveryView(_ context.Context, b grubbery.Binding, p grubbery.RuntimeProcess) (grubbery.NativeRuntime, error) {
	if b != d.f.binding || p != d.f.process {
		return nil, grubbery.ErrAdmission
	}
	return approvedAppRuntimeModel{d, p, true}, nil
}
func (d appRuntimeRecoveryModel) GrubberyRecovered(_ context.Context, r grubbery.Record, a grubbery.RuntimeRecoveryReceipt) (grubbery.NativeRuntime, error) {
	if a.ValidateAncestry(r) != nil || r.Plan.Binding != d.f.binding || a.Review.Process != d.f.process {
		return nil, grubbery.ErrAdmission
	}
	return approvedAppRuntimeModel{d, d.f.process, false}, nil
}
func (d approvedAppRuntimeModel) GrubberyRead(ctx context.Context, q grubbery.ReadRequest) (grubbery.NativeResponse, error) {
	if d.f.unavailable || d.expected != d.f.process {
		return grubbery.NativeResponse{}, grubbery.ErrAdmission
	}
	return d.managedAppNativeRuntime.GrubberyRead(ctx, q)
}
func (d approvedAppRuntimeModel) GrubberyStep(ctx context.Context, p grubbery.Plan, id string, files []deskwork.SourceFile) (grubbery.NativeResponse, error) {
	if d.readOnly || d.f.unavailable || d.expected != d.f.process {
		return grubbery.NativeResponse{}, grubbery.ErrAdmission
	}
	return d.managedAppNativeRuntime.GrubberyStep(ctx, p, id, files)
}

func managedRuntimeRecoveryFixture(t *testing.T) (*Server, *appRuntimeRecoveryFixture, grubbery.Record, protocol.DeskSelection) {
	t.Helper()
	r, native, p, _ := managedGrubberyFixture(t)
	b, err := r.nativeGrubberyBackend(t.Context(), p.Binding, p.Binding.Actor, "operator")
	g02Check(t, err)
	e := b.Execution(grubberyJournal{r.s})
	_, err = e.Begin(t.Context(), managedAppCall(p), p)
	g02Check(t, err)
	native.lost = true
	original, err := e.Advance(t.Context(), managedAppCall(p))
	g02Check(t, err)
	if state, _ := original.State(p.Steps[0].ID); state != "uncertain" || native.writes != 1 {
		t.Fatal("unknown original effect fixture required", state, native.writes)
	}
	f := &appRuntimeRecoveryFixture{managedAppNativeFixture: native, process: grubbery.RuntimeProcess{ContainerID: strings.Repeat("9", 64), PID: 4242, StartedAt: time.Now().UTC().Add(-time.Minute), HostBootID: "11111111-2222-3333-4444-555555555555", EngineID: "fixture-recovery-daemon", ImageManifestDigest: "sha256:" + strings.Repeat("8", 64), RuntimeVersion: "4.6-5a7614f", ContextSHA256: strings.Repeat("7", 64)}}
	// This models the already completed supported replacement selection only.
	// It is not native Stop/Start evidence or a user database repair procedure.
	v, err := loadInstance(r.s.db, p.Binding.Target.InstanceID)
	g02Check(t, err)
	v.Public.Resources.ContainerID = f.process.ContainerID
	saveRecoveryFixture(t, r.s, v)
	r.b = f
	selection := protocol.DeskSelection{Target: p.Binding.Target, RuntimeID: f.process.ContainerID, ConfigRevision: p.Binding.ConfigRevision}
	return r, f, original, selection
}

func TestManagedRuntimeRecoveryFreshApplyPreservesUnknownAndReconcilesWithoutSourceOrRedispatch(t *testing.T) {
	r, f, original, selection := managedRuntimeRecoveryFixture(t)
	actor := original.Plan.Binding.Actor
	g02Check(t, os.RemoveAll(filepath.Join(r.s.directory, "desk-stages-v1")))
	policy, err := loadAppPolicy(r.s.db)
	g02Check(t, err)
	_, err = r.s.putAppPolicy("setup", apppolicy.Update{ExpectedRevision: policy.Revision, Scope: "desk", Target: selection.Target, Gates: apppolicy.Gates{Commit: apppolicy.Block, Publish: apppolicy.Block, CreateDesk: apppolicy.Inherit}})
	g02Check(t, err)
	before := rowsWithoutRuntimeRecovery(appAllRows(t, r.s.db))
	review, err := r.reviewGrubberyRuntime(t.Context(), original.Plan.RequestID, selection, actor, "operator")
	g02Check(t, err)
	if review.EvidenceBasis != "native-phase-observed" || review.AppliedPhaseProof == nil || review.Process != f.process {
		t.Fatal("fresh native/process evidence missing", review)
	}
	receipt, err := r.applyGrubberyRuntime(t.Context(), original.Plan.RequestID, strings.Repeat("5", 64), review, actor, "operator")
	g02Check(t, err)
	if receipt.ValidateAncestry(original) != nil || f.writes != 1 || !reflect.DeepEqual(before, rowsWithoutRuntimeRecovery(appAllRows(t, r.s.db))) {
		t.Fatal("rebind rewrote or resolved original operation")
	}
	dir := r.s.directory
	g02Check(t, r.s.close())
	r.s, err = testOpenStore(dir)
	g02Check(t, err)
	head, err := (grubberyRuntimeJournal{r.s}).Latest(t.Context(), original.Plan.RequestID)
	g02Check(t, err)
	if head == nil || !reflect.DeepEqual(*head, receipt) {
		t.Fatal("store reopen lost approved exact audit head")
	}
	backend, err := r.nativeGrubberyBackendFor(t.Context(), original, actor, "operator")
	g02Check(t, err)
	got, err := backend.Execution(grubberyJournal{r.s}).Reconcile(t.Context(), managedAppCall(original.Plan))
	g02Check(t, err)
	if state, _ := got.State(original.Plan.Steps[0].ID); state != "observed" || f.writes != 1 || got.Revision != original.Revision+1 {
		t.Fatal("exact rebound reconciliation retried effect", state, f.writes)
	}
	for i, event := range original.Events {
		if !reflect.DeepEqual(got.Events[i], event) {
			t.Fatal("reconciliation changed original prefix")
		}
	}
	// Later pending work still needs current Commit and its original cache.
	next, _, err := got.Current()
	g02Check(t, err)
	call := managedAppCall(original.Plan)
	call.StepID = next.ID
	backend, err = r.nativeGrubberyBackendFor(t.Context(), got, actor, "operator")
	g02Check(t, err)
	if _, err = backend.Execution(grubberyJournal{r.s}).Advance(t.Context(), call); err == nil || f.writes != 1 {
		t.Fatal("metadata recovery granted a later write", err)
	}
	reads := f.reads
	f.unavailable = true
	r.updateFence.Held = true
	replay, err := r.applyGrubberyRuntime(t.Context(), original.Plan.RequestID, receipt.RequestID, review, actor, "operator")
	g02Check(t, err)
	if !reflect.DeepEqual(replay, receipt) || f.reads != reads || f.writes != 1 {
		t.Fatal("saved replay probed runtime or changed audit")
	}
}

func TestManagedRuntimeRecoveryApplyReobservesSelfResealedCallerClaims(t *testing.T) {
	for _, scenario := range []string{"pid", "namespace", "native", "actor", "reader", "fence", "config", "stale-head", "expired"} {
		t.Run(scenario, func(t *testing.T) {
			r, f, original, selection := managedRuntimeRecoveryFixture(t)
			actor, role := original.Plan.Binding.Actor, "operator"
			review, err := r.reviewGrubberyRuntime(t.Context(), original.Plan.RequestID, selection, actor, role)
			g02Check(t, err)
			switch scenario {
			case "pid":
				review.Process.PID++ // Valid self digest is not observed authority.
			case "namespace":
				f.process.ContextSHA256 = strings.Repeat("6", 64)
			case "native":
				f.marker = nil
			case "actor":
				actor = "local-uid:99999"
			case "reader":
				role = "reader"
			case "fence":
				r.updateFence.Held = true
			case "config":
				v, err := loadInstance(r.s.db, selection.Target.InstanceID)
				g02Check(t, err)
				v.Public.ConfigRevision++
				saveRecoveryFixture(t, r.s, v)
			case "stale-head":
				_, err = r.applyGrubberyRuntime(t.Context(), original.Plan.RequestID, strings.Repeat("4", 64), review, actor, role)
				g02Check(t, err)
			case "expired":
				review.ObservedAt = review.ObservedAt.Add(-10 * time.Minute)
				review.Process.StartedAt = review.Process.StartedAt.Add(-time.Hour)
			}
			review.Digest = review.ContentDigest()
			before := appAllRows(t, r.s.db)
			if _, err = r.applyGrubberyRuntime(t.Context(), original.Plan.RequestID, strings.Repeat("5", 64), review, actor, role); err == nil {
				t.Fatal("changed/resealed claim admitted", scenario)
			}
			if !reflect.DeepEqual(before, appAllRows(t, r.s.db)) || f.writes != 1 {
				t.Fatal("refused review changed original/audit state")
			}
			if scenario == "stale-head" && !errors.Is(err, grubbery.ErrCAS) {
				t.Fatal("changed head did not refuse exact CAS", err)
			}
		})
	}
}

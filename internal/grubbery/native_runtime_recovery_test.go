package grubbery_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

func TestNativeRuntimeRecoveryVerifiesPendingPrefixWithoutSourceOrEffect(t *testing.T) {
	for _, scenario := range []string{"empty", "compiled-prefix", "versioned-prefix", "compiler-failure", "cleanup-failure", "unowned-drift", "stale-prefix", "guard"} {
		t.Run(scenario, func(t *testing.T) {
			prior, files := plan(t)
			var scoped *scopedNativeModel
			h := nativeHarnessFor(t, prior, files, func(model *nativeModel) grubbery.NativeRuntime {
				scoped = &scopedNativeModel{nativeModel: model, cleanup: "confirmed"}
				return scoped
			})
			p := nativePlan(t, h, prior, prior.RequestID)
			release, _ := p.Binding.ReleaseRoot(p.RequestID)
			original, err := h.execution().Begin(t.Context(), callFor(p, ""), p)
			if err != nil {
				t.Fatal(err)
			}
			if scenario != "empty" && scenario != "unowned-drift" {
				for _, step := range p.Steps {
					original = nativeAdvance(t, h, p, step)
					if scenario == "versioned-prefix" {
						if step.Kind == "make-file" && step.Path == release+"/version.json" {
							break
						}
					} else if step.Kind == "observe-code" {
						break
					}
				}
			}
			step, done, err := original.Current()
			if err != nil || done {
				t.Fatal("unfinished pending fixture required", err)
			}
			if state, _ := original.State(step.ID); state != "pending" {
				t.Fatal(state)
			}
			calls, compilerReads := h.sourceCalls, scoped.compilerReads
			writes := map[string]int{}
			for id, n := range h.model.writes {
				writes[id] = n
			}
			switch scenario {
			case "compiler-failure":
				scoped.compilerFailure = true
			case "cleanup-failure":
				scoped.cleanup = "unconfirmed"
			case "unowned-drift":
				n, _ := p.Binding.Namespace()
				h.model.mkdir(n.SourceRoot, "")
			case "stale-prefix":
				nativeAdvance(t, h, p, step)
			case "guard":
				h.deny = true
			}
			// The stale-prefix scenario deliberately performs one earlier fixture
			// action; review itself may never perform another one.
			if scenario == "stale-prefix" {
				writes = map[string]int{}
				for id, n := range h.model.writes {
					writes[id] = n
				}
				calls = h.sourceCalls
			}
			h.sourceMissing = true
			evidence, err := h.backend.ObserveRuntimeRecovery(t.Context(), original)
			success := scenario == "empty" || scenario == "compiled-prefix" || scenario == "versioned-prefix"
			if success {
				if err != nil || evidence.EvidenceBasis != "verified-pending-prefix" || evidence.AppliedPhaseProof != nil || len(evidence.NativeEvidenceDigest) != 64 {
					t.Fatal("prefix review failed", evidence, err)
				}
				if scenario != "empty" && scoped.compilerReads != compilerReads+1 {
					t.Fatal("observed compiler prefix was not checked within capture")
				}
			} else if err == nil {
				t.Fatal("invalid recovery evidence accepted", scenario, evidence)
			}
			if scenario == "stale-prefix" && !errors.Is(err, grubbery.ErrCAS) {
				t.Fatal("stale review did not refuse its exact changed prefix", err)
			}
			if h.sourceCalls != calls || !reflect.DeepEqual(writes, h.model.writes) {
				t.Fatal("review loaded source or submitted an effect")
			}
			got, _ := h.journal.Load(t.Context(), p.RequestID)
			if scenario != "stale-prefix" && !reflect.DeepEqual(got, original) {
				t.Fatal("review appended or resolved original phase")
			}
		})
	}
}

type recoveryPrefixRaceRuntime struct {
	*scopedNativeModel
	after func()
}

func (r *recoveryPrefixRaceRuntime) WithGrubberyReads(ctx context.Context, binding grubbery.Binding, run func(grubbery.NativeReader) error) (string, error) {
	cleanup, err := r.scopedNativeModel.WithGrubberyReads(ctx, binding, run)
	if r.after != nil {
		after := r.after
		r.after = nil
		after()
	}
	return cleanup, err
}

func TestNativeRuntimeRecoveryRefusesPrefixProgressDuringCapture(t *testing.T) {
	prior, files := plan(t)
	var runtime *recoveryPrefixRaceRuntime
	h := nativeHarnessFor(t, prior, files, func(model *nativeModel) grubbery.NativeRuntime {
		runtime = &recoveryPrefixRaceRuntime{scopedNativeModel: &scopedNativeModel{nativeModel: model, cleanup: "confirmed"}}
		return runtime
	})
	p := nativePlan(t, h, prior, prior.RequestID)
	original, err := h.execution().Begin(t.Context(), callFor(p, ""), p)
	if err != nil {
		t.Fatal(err)
	}
	step := p.Steps[0]
	runtime.after = func() {
		next, err := original.WithEvent(grubbery.Event{StepID: step.ID, State: "dispatched", At: time.Now().UTC(), Review: &grubbery.Review{Version: 1, PlanDigest: p.Digest, StepID: step.ID, BindingDigest: hashValue(p.Binding), PolicyRevision: p.Binding.PolicyRevision, NativeEvidenceDigest: strings.Repeat("7", 64)}})
		if err != nil {
			t.Fatal(err)
		}
		h.journal.mu.Lock()
		h.journal.records[p.RequestID] = copyRecord(next)
		h.journal.mu.Unlock()
	}
	calls := h.sourceCalls
	evidence, err := h.backend.ObserveRuntimeRecovery(t.Context(), original)
	if !errors.Is(err, grubbery.ErrCAS) || evidence.NativeEvidenceDigest != "" || len(h.model.writes) != 0 || h.sourceCalls != calls {
		t.Fatal("changed prefix after capture acquired binding evidence", evidence, err)
	}
}

func TestNativeRuntimeRecoveryUnknownVersionRequiresObservedCompilerAndExactVersion(t *testing.T) {
	prior, files := plan(t)
	var scoped *scopedNativeModel
	h := nativeHarnessFor(t, prior, files, func(model *nativeModel) grubbery.NativeRuntime {
		scoped = &scopedNativeModel{nativeModel: model, cleanup: "confirmed"}
		return scoped
	})
	p := nativePlan(t, h, prior, prior.RequestID)
	if _, err := h.execution().Begin(t.Context(), callFor(p, ""), p); err != nil {
		t.Fatal(err)
	}
	release, _ := p.Binding.ReleaseRoot(p.RequestID)
	var original grubbery.Record
	var step grubbery.Step
	for _, phase := range p.Steps {
		if phase.Kind == "make-file" && phase.Path == release+"/version.json" {
			step = phase
			break
		}
		nativeAdvance(t, h, p, phase)
	}
	if step.ID == "" {
		t.Fatal("missing original native version phase")
	}
	h.model.lost = true
	original, err := h.execution().Advance(t.Context(), callFor(p, step.ID))
	if err != nil {
		t.Fatal(err)
	}
	h.sourceMissing = true
	calls, compiles := h.sourceCalls, scoped.compilerReads
	evidence, err := h.backend.ObserveRuntimeRecovery(t.Context(), original)
	if err != nil || evidence.AppliedPhaseProof == nil || evidence.AppliedPhaseProof.StepID != step.ID || scoped.compilerReads != compiles+1 || h.model.writes[step.ID] != 1 || h.sourceCalls != calls {
		t.Fatal("unknown version/compiled prefix was not independently checked", evidence, err)
	}
	h.model.files[release+"/version.json"] = []byte(`{"epoch":999}`)
	if _, err = h.backend.ObserveRuntimeRecovery(t.Context(), original); !errors.Is(err, grubbery.ErrObservation) {
		t.Fatal("changed native version became binding authority", err)
	}
}

func TestNativeRuntimeRecoveryRequiresExactAlreadyAppliedUnknownEffect(t *testing.T) {
	prior, files := plan(t)
	h := nativeHarnessFor(t, prior, files)
	p := nativePlan(t, h, prior, prior.RequestID)
	if _, err := h.execution().Begin(t.Context(), callFor(p, ""), p); err != nil {
		t.Fatal(err)
	}
	step := p.Steps[0]
	h.model.lost = true
	original, err := h.execution().Advance(t.Context(), callFor(p, step.ID))
	if err != nil {
		t.Fatal(err)
	}
	h.sourceMissing = true
	calls := h.sourceCalls
	evidence, err := h.backend.ObserveRuntimeRecovery(t.Context(), original)
	if err != nil || evidence.EvidenceBasis != "native-phase-observed" || evidence.AppliedPhaseProof == nil || evidence.AppliedPhaseProof.StepID != step.ID || evidence.AppliedPhaseProof.NativeEvidenceDigest != evidence.NativeEvidenceDigest {
		t.Fatal("exact unknown effect was not observed", evidence, err)
	}
	if h.model.writes[step.ID] != 1 || h.sourceCalls != calls {
		t.Fatal("unknown effect was redispatched or source cache loaded")
	}
	n, _ := p.Binding.Namespace()
	delete(h.model.files, n.SourceRoot+"/omabit-owner.json")
	if _, err = h.backend.ObserveRuntimeRecovery(t.Context(), original); !errors.Is(err, grubbery.ErrObservation) {
		t.Fatal("missing original effect became binding permission", err)
	}
	got, _ := h.journal.Load(t.Context(), p.RequestID)
	if !reflect.DeepEqual(got, original) || h.model.writes[step.ID] != 1 {
		t.Fatal("review resolved or retried original uncertainty")
	}
}

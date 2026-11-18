package grubbery_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

type scopedNativeModel struct {
	*nativeModel
	cleanup, scenario string
	scopes            int
	compilerReads     int
	compilerFailure   bool
}

type provisionalModelReader struct {
	model  *nativeModel
	closed bool
	scope  *scopedNativeModel
}

func (r *provisionalModelReader) GrubberyRead(ctx context.Context, q grubbery.ReadRequest) (grubbery.NativeResponse, error) {
	if r.closed {
		return grubbery.NativeResponse{}, errors.New("read scope already closed")
	}
	response, err := r.model.GrubberyRead(ctx, q)

	request, _ := q.Request()
	if r.scope != nil && strings.HasPrefix(request.Path, "/grubbery/api/build") {
		r.scope.compilerReads++
		if r.scope.compilerFailure {
			response = r.model.reply(200, []byte(`{"namespace":"/foreign","total":0,"failed":0,"files":[]}`))
		}
	}
	response.SessionCleanup = grubbery.NativeReadScopeActive
	return response, err
}

func (s *scopedNativeModel) GrubberyRead(context.Context, grubbery.ReadRequest) (grubbery.NativeResponse, error) {
	return grubbery.NativeResponse{}, errors.New("scoped capture fell back to individual authenticated reads")
}

func (s *scopedNativeModel) WithGrubberyReads(ctx context.Context, _ grubbery.Binding, run func(grubbery.NativeReader) error) (string, error) {
	s.scopes++
	reader := &provisionalModelReader{model: s.nativeModel, scope: s}
	defer func() { reader.closed = true }()
	switch s.scenario {
	case "never-called":
		return "confirmed", nil
	case "nil-reader":
		_ = run(nil)
		return "confirmed", nil
	case "twice":
		_ = run(reader)
		_ = run(reader)
		return "confirmed", nil
	case "ignore-failure":
		_ = run(reader)
		return "confirmed", nil
	}
	return s.cleanup, run(reader)
}

func TestNativeReadScopeProducesTheSameReviewedEvidenceAndSingleWrites(t *testing.T) {
	prior, files := plan(t)
	ordinary := nativeHarnessFor(t, prior, files)
	var scoped *scopedNativeModel
	withScope := nativeHarnessFor(t, prior, files, func(model *nativeModel) grubbery.NativeRuntime {
		scoped = &scopedNativeModel{nativeModel: model, cleanup: "confirmed"}
		return scoped
	})
	first := nativePlan(t, ordinary, prior, prior.RequestID)
	second := nativePlan(t, withScope, prior, prior.RequestID)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("scope changed reviewed native evidence or canonical plan")
	}
	for _, harness := range []*nativeHarness{ordinary, withScope} {
		if _, err := harness.execution().Begin(t.Context(), callFor(first, ""), first); err != nil {
			t.Fatal(err)
		}
		for _, step := range first.Steps {
			nativeAdvance(t, harness, first, step)
		}
	}
	if scoped.scopes == 0 || ordinary.model.reads != withScope.model.reads || !reflect.DeepEqual(ordinary.model.writes, withScope.model.writes) || !reflect.DeepEqual(ordinary.model.files, withScope.model.files) {
		t.Fatal("scope skipped/changed original reads or native writes")
	}
	for _, count := range scoped.writes {
		if count != 1 {
			t.Fatal("scoped authentication redispatched an effect")
		}
	}
}

func TestNativeReadScopeCannotAcceptEvidenceBeforeConfirmedCleanup(t *testing.T) {
	for _, cleanup := range []string{"", "unconfirmed", grubbery.NativeReadScopeActive, "not-created"} {
		t.Run(cleanup, func(t *testing.T) {
			prior, files := plan(t)
			h := nativeHarnessFor(t, prior, files, func(model *nativeModel) grubbery.NativeRuntime {
				return &scopedNativeModel{nativeModel: model, cleanup: cleanup}
			})
			if _, err := h.backend.Inspect(t.Context(), prior.Binding); !errors.Is(err, grubbery.ErrObservation) || len(h.model.writes) != 0 || len(h.journal.records) != 0 {
				t.Fatal("provisional/unclean scope became admitted evidence", err)
			}
		})
	}
}

func TestNativeReadScopeDoesNotIgnoreConditionalReadFailureOrInvalidCallback(t *testing.T) {
	for _, scenario := range []string{"never-called", "nil-reader", "twice", "ignore-failure"} {
		t.Run(scenario, func(t *testing.T) {
			prior, files := plan(t)
			h := nativeHarnessFor(t, prior, files, func(model *nativeModel) grubbery.NativeRuntime {
				return &scopedNativeModel{nativeModel: model, cleanup: "confirmed", scenario: scenario}
			})
			if scenario == "ignore-failure" {
				// A foreign existing root must stop capture before any deeper read.
				namespace, err := prior.Binding.Namespace()
				if err != nil {
					t.Fatal(err)
				}
				h.model.mkdir(namespace.SourceRoot, "")
			}
			if _, err := h.backend.Inspect(t.Context(), prior.Binding); !errors.Is(err, grubbery.ErrObservation) || len(h.model.writes) != 0 {
				t.Fatal("invalid callback became native evidence", err)
			}
			if scenario == "ignore-failure" && h.model.reads != 1 {
				t.Fatal("scope bypassed conditional root checks and read deeper content")
			}
		})
	}
}

func TestScopedCompilerChecksCannotOutliveCaptureOrIgnoreFailure(t *testing.T) {
	for _, scenario := range []string{"compiler-failed", "runtime-ignored-error", "logout-unconfirmed"} {
		t.Run(scenario, func(t *testing.T) {
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
			var compiler grubbery.Step
			for _, step := range p.Steps {
				if step.Kind == "observe-code" {
					compiler = step
					break
				}
				nativeAdvance(t, h, p, step)
			}
			if compiler.ID == "" {
				t.Fatal("fixture omitted compiler phase")
			}
			scoped.compilerFailure = true
			if scenario == "runtime-ignored-error" {
				scoped.scenario = "ignore-failure"
			}
			// Let dispatch's admission finish normally; its after-read then sees the
			// invalid compiler report from the independently supplied scoped reader.
			record, err := h.execution().Advance(t.Context(), callFor(p, compiler.ID))
			state, stateErr := record.State(compiler.ID)
			if err != nil || stateErr != nil || state != "uncertain" || scoped.compilerReads == 0 {
				t.Fatal("invalid compiled evidence became observed", state, err, stateErr)
			}
			for _, event := range record.Events {
				if event.StepID == compiler.ID && event.Proof != nil {
					t.Fatal("invalid compiler produced a saved proof")
				}
			}
			writes := map[string]int{}
			for key, count := range h.model.writes {
				writes[key] = count
			}
			// Valid bytes are still provisional until this particular capture's logout.
			if scenario == "logout-unconfirmed" {
				scoped.compilerFailure = false
				scoped.cleanup = "unconfirmed"
			}
			repeated, err := h.execution().Reconcile(t.Context(), callFor(p, compiler.ID))
			if err != nil || !reflect.DeepEqual(repeated, record) || !reflect.DeepEqual(writes, h.model.writes) {
				t.Fatal("failed compiler/cleanup reconciliation changed history or redispatched", err)
			}
			scoped.compilerFailure = false
			scoped.cleanup = "confirmed"
			scoped.scenario = ""
			observed, err := h.execution().Reconcile(t.Context(), callFor(p, compiler.ID))
			state, stateErr = observed.State(compiler.ID)
			if err != nil || stateErr != nil || state != "observed" || !reflect.DeepEqual(writes, h.model.writes) {
				t.Fatal("confirmed in-scope compiler read did not resolve without another effect", state, err)
			}
		})
	}
}

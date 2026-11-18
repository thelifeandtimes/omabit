package grubbery_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

type scopedNativeModel struct {
	*nativeModel
	cleanup, scenario string
	scopes            int
}

type provisionalModelReader struct{ model *nativeModel }

func (r provisionalModelReader) GrubberyRead(ctx context.Context, q grubbery.ReadRequest) (grubbery.NativeResponse, error) {
	response, err := r.model.GrubberyRead(ctx, q)
	response.SessionCleanup = grubbery.NativeReadScopeActive
	return response, err
}

func (s *scopedNativeModel) GrubberyRead(context.Context, grubbery.ReadRequest) (grubbery.NativeResponse, error) {
	return grubbery.NativeResponse{}, errors.New("scoped capture fell back to individual authenticated reads")
}

func (s *scopedNativeModel) WithGrubberyReads(ctx context.Context, _ grubbery.Binding, run func(grubbery.NativeReader) error) (string, error) {
	s.scopes++
	reader := provisionalModelReader{s.nativeModel}
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

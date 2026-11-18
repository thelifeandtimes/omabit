package service

import (
	"context"
	"errors"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type AppSourceReviewInput struct {
	Target
	Request protocol.GrubberyAppReviewRequest `json:"request"`
}

func (in AppSourceReviewInput) Validate() error {
	if in.Target.Validate() != nil || in.Request.Validate() != nil {
		return invalid()
	}
	return nil
}

type AppSourceBeginInput struct {
	Target
	Request protocol.GrubberyAppBeginRequest `json:"request"`
}

func (in AppSourceBeginInput) Validate() error {
	if in.Target.Validate() != nil || in.Request.Plan.ValidateMetadata() != nil || in.RunnerID != in.Request.Plan.Binding.RunnerID {
		return invalid()
	}
	return nil
}

type AppSourcePhaseInput struct {
	Target
	OperationID string                           `json:"operation_id"`
	Request     protocol.GrubberyAppPhaseRequest `json:"request"`
}

func (in AppSourcePhaseInput) Validate() error {
	if in.Target.Validate() != nil || !protocol.ValidDeskOperationID(in.OperationID) || in.Request.Validate() != nil {
		return invalid()
	}
	return nil
}

func (s *Service) registerGrubbery() error {
	bindings := "selected runner/mode, authenticated original actor, exact native target/config/pier/framework, access-bound stage and original request/plan/phase"
	describe := func(id, about string, effects []string) Descriptor {
		return description(id, about, nil, effects, []string{protocol.GrubberyAppSourceFeature}, "operator", bindings, "save original request ID and plan; repeat only the exact selected phase; uncertainty uses read-only reconciliation")
	}
	var errs []error
	errs = append(errs, selected(s, describe("app.review-source.v1", "Prepare an exact Grubbery source plan from private staging and fresh native ownership; no app effect or journal write.", []string{"local-metadata-read", "live-probe"}), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in AppSourceReviewInput) (protocol.GrubberyAppReview, error) {
		return r.ReviewGrubberyAppSource(ctx, in.Name, in.RunnerID, in.Request)
	}))
	errs = append(errs, selected(s, describe("app.begin-source.v1", "Retain an unchanged reviewed plan and claim its namespace lease; no native app effect.", []string{"local-metadata-read", "live-probe", "journal-write"}), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in AppSourceBeginInput) (protocol.GrubberyAppOperation, error) {
		return r.BeginGrubberyAppSource(ctx, in.Name, in.RunnerID, in.Request)
	}))
	for _, action := range []string{"advance", "reconcile", "abandon"} {
		effects := []string{"local-metadata-read", "live-probe", "journal-write"}
		about := "Read after one exact original app phase without redispatch or source cache; retain original uncertainty."
		if action == "advance" {
			effects = append(effects, "runtime-mutation")
			about = "Admit and dispatch one exact original app source phase once; saved replay never selects the next phase."
		}
		if action == "abandon" {
			about = "Abandon only proven preactivation app work; retain source/live state and audit, releasing its exact lease."
		}
		errs = append(errs, selected(s, describe("app."+action+"-source.v1", about, effects), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in AppSourcePhaseInput) (protocol.GrubberyAppOperation, error) {
			return r.GrubberyAppSourcePhase(ctx, in.Name, in.RunnerID, in.OperationID, action, in.Request)
		}))
	}
	return errors.Join(errs...)
}

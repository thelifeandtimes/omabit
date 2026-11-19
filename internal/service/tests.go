package service

import (
	"context"
	"errors"
	"sort"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/checkplan"
)

type TestPlanInput struct {
	ID string `json:"id"`
}

func (in TestPlanInput) Validate() error { _, err := checkplan.Read(in.ID); return err }

type PrivateTestReportsInput struct {
	IncludePrivateData bool `json:"include_private_data"`
}

func (in PrivateTestReportsInput) Validate() error {
	if !in.IncludePrivateData {
		return invalid()
	}
	return nil
}

func (s *Service) registerTests() error {
	offline := []string{"offline-read"}
	private := []string{"local-metadata-read", "private-runtime-read"}
	return errors.Join(
		bind(s, description("tests.plans.v1", "List finite compiled disposable checkpoint plans. Metadata only; never launches a test or reads a registry.", nil, offline, nil, "local-user", "compiled plan names; no command, repository, pier or environment argument", "read only; a plan is not execution evidence"), func(context.Context, *Service, Empty) ([]checkplan.Plan, error) { return checkplan.List() }),
		bind(s, description("tests.plan.v1", "Read one named runtime/HTTP checkpoint plan, its pinned inputs, limits, steps and digest. Existing acknowledged qualifiers remain local-owner only; no shared executor is exposed.", []string{"urbit-runner runtime-plan", "urbit-runner http-plan"}, offline, nil, "local-user", "exact compiled plan ID; no user command or runtime selection", "same constructor as legacy plan output; neither reading nor possessing a plan authorizes execution"), func(_ context.Context, _ *Service, in TestPlanInput) (checkplan.Plan, error) {
			return checkplan.Read(in.ID)
		}),
		bind(s, description("tests.reports.v1", "List only report snapshots explicitly selected at process startup, after include_private_data=true. No filesystem paths, raw output or automatic report discovery.", nil, private, nil, "local-user", "startup-selected private report snapshots only; distinct from source roots and runner enrollment", "immutable startup snapshots; no fresh filesystem/runtime read or test execution"), func(_ context.Context, s *Service, _ PrivateTestReportsInput) ([]checkplan.ReportBinding, error) {
			out := make([]checkplan.ReportBinding, 0, len(s.options.TestReports))
			for _, report := range s.options.TestReports {
				out = append(out, checkplan.ReportBinding{Name: report.Name(), PlanID: report.PlanID(), SHA256: report.SHA256()})
			}
			sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
			return out, nil
		}),
		bind(s, description("tests.report.v1", "Read a bounded summary of an exact startup-selected historical runtime/HTTP report. Digest verification authenticates selected bytes only. Reported pass/cleanup and compiled-pin comparison are not attested execution or repository-source proof. Raw errors/logs/paths are omitted.", nil, private, nil, "local-user", "exact allowed name, named plan, SHA256 and include_private_data=true; no model-supplied file path", "immutable historical snapshot; this service ran no test and claims no production/runtime qualification"), func(_ context.Context, s *Service, in checkplan.ReportSelection) (checkplan.Summary, error) {
			for _, report := range s.options.TestReports {
				if report.Name() == in.Name && report.PlanID() == in.PlanID && report.SHA256() == in.SHA256 {
					return report.Summary(), nil
				}
			}
			return checkplan.Summary{}, &refusal{"test_report_not_allowed", "The exact historical report was not selected at process startup.", "Use tests.reports.v1 with explicit private-data consent; reports cannot be loaded from tool-supplied paths."}
		}),
	)
}

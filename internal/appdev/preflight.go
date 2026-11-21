// Package appdev defines read-only application workflow compatibility checks.
// A preflight is guidance, never an authorization token or a runtime readiness
// proof. Desk effects must still review current Clay and publication evidence.
package appdev

import (
	"errors"
	"slices"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type PreflightRequest struct {
	Name       string   `json:"name"`
	RunnerID   string   `json:"runner_id"`
	Workflow   string   `json:"workflow"`
	InstanceID string   `json:"instance_id,omitempty"`
	Desk       string   `json:"desk,omitempty"`
	Effects    []string `json:"effects"`
}

func (r PreflightRequest) Validate() error {
	if !protocol.ValidID(r.Name) || !protocol.ValidID(r.RunnerID) || (r.Workflow != "gall" && r.Workflow != "grubbery") {
		return errors.New("select an enrolled runner, its expected identity and gall or grubbery workflow")
	}
	if (r.InstanceID == "") != (r.Desk == "") || (apppolicy.Target{InstanceID: r.InstanceID, Desk: r.Desk}).Validate() != nil {
		return errors.New("select both instance and desk, or omit both for environment-only preflight")
	}
	if len(r.Effects) == 0 || len(r.Effects) > 3 {
		return errors.New("select create-desk, commit and/or publish effects")
	}
	seen := map[string]bool{}
	for _, effect := range r.Effects {
		if (effect != "create-desk" && effect != "commit" && effect != "publish") || seen[effect] {
			return errors.New("invalid or duplicate application effect")
		}
		seen[effect] = true
	}
	return nil
}

type PolicyBlock struct {
	Effect string `json:"effect"`
	Scope  string `json:"scope"`
}
type PolicyCheck struct {
	Revision  uint64              `json:"revision"`
	ReadAt    time.Time           `json:"read_at"`
	Target    apppolicy.Target    `json:"target"`
	Effective apppolicy.Effective `json:"effective"`
	Blocked   []PolicyBlock       `json:"blocked"`
}
type Preflight struct {
	Version                  int                       `json:"version"`
	Request                  PreflightRequest          `json:"request"`
	Hello                    protocol.Hello            `json:"hello"`
	ClientBuild              *protocol.ExecutableBuild `json:"client_build,omitempty"`
	ObservedAt               time.Time                 `json:"observed_at"`
	Status                   string                    `json:"status"`
	Assurance                string                    `json:"assurance"`
	EnvironmentCompatible    bool                      `json:"environment_compatible"`
	Operator                 bool                      `json:"operator"`
	MissingAPIs              []string                  `json:"missing_apis"`
	MissingOperationAdapters []string                  `json:"missing_operation_adapters"`
	WorkflowAdapter          string                    `json:"workflow_adapter"`
	WorkflowScope            string                    `json:"workflow_scope"`
	WorkflowAdapterAvailable bool                      `json:"workflow_adapter_available"`
	Policy                   *PolicyCheck              `json:"policy,omitempty"`
	NextSteps                []string                  `json:"next_steps"`
}

func (p Preflight) Compatible() bool {
	return p.Status == "environment-compatible" || p.Status == "saved-policy-compatible"
}

// Assess binds policy to the exact target returned by the runner, including a
// project's creator/generation. It deliberately does not guess future ship grants
// when called before creating an application-specific project.
func Assess(req PreflightRequest, h protocol.Hello, policy *protocol.AppPolicyStatus) (Preflight, error) {
	p := Preflight{Version: 1, Request: req, Hello: h, ObservedAt: time.Now().UTC(), Assurance: "capability-and-saved-policy-only", MissingAPIs: []string{}, MissingOperationAdapters: []string{}, NextSteps: []string{}}
	if err := req.Validate(); err != nil {
		return p, err
	}
	if h.RunnerID != req.RunnerID {
		return p, errors.New("preflight runner identity changed")
	}
	p.Operator = h.AccessRole == "operator"
	apis := []string{protocol.AppPolicyFeature, protocol.DeskSettingsFeature}
	adapters := []string{protocol.DeskDiscoveryFeature}
	for _, effect := range req.Effects {
		switch effect {
		case "create-desk":
			adapters = append(adapters, protocol.DeskCreateFeature)
		case "commit":
			apis = append(apis, protocol.DeskStageFeature)
			adapters = append(adapters, protocol.DeskCommitFeature, protocol.DeskApplicationFeature)
		case "publish":
			adapters = append(adapters, protocol.DeskReleaseFeature)
		}
	}
	p.WorkflowAdapter = "gall-desk-v1"
	p.WorkflowScope = "selected-desk-effects"
	p.WorkflowAdapterAvailable = true
	if req.Workflow == "grubbery" {
		// This is the implemented source/restart workflow, not general app
		// readiness or support for peer/permit/route/asset management. Require
		// both native adapters and independent saved-receipt access before boot.
		p.WorkflowAdapter = grubbery.HTTPContract
		p.WorkflowScope = "source-and-restart"
		apis = append(apis, protocol.GrubberyAppReceiptsFeature, protocol.GrubberyRuntimeReceiptsFeature)
		adapters = append(adapters, protocol.GrubberyAppSourceFeature, protocol.GrubberyRuntimeRecoveryFeature)
		p.NextSteps = append(p.NextSteps, "This preflight covers versioned Grubbery source and restart recovery only; inspect the pinned framework and each app effect separately. Peer discovery, permits, routes and served assets require their own supported workflow and evidence.")
	}
	for _, feature := range apis {
		if !slices.Contains(h.Features, feature) {
			p.MissingAPIs = append(p.MissingAPIs, feature)
		}
	}
	for _, feature := range adapters {
		if !slices.Contains(h.Features, feature) {
			p.MissingOperationAdapters = append(p.MissingOperationAdapters, feature)
		}
	}
	p.EnvironmentCompatible = p.Operator && len(p.MissingAPIs) == 0 && len(p.MissingOperationAdapters) == 0 && p.WorkflowAdapterAvailable
	p.Status = "environment-compatible"
	if !p.EnvironmentCompatible {
		switch {
		case !p.Operator:
			p.Status = "operator-role-required"
			p.NextSteps = append(p.NextSteps, "Pair an operator key through Setup; do not change the existing key role implicitly.")
		case len(p.MissingAPIs) > 0:
			p.Status = "missing-runner-api"
		case len(p.MissingOperationAdapters) > 0:
			p.Status = "missing-operation-adapter"
		default:
			p.Status = "missing-workflow-adapter"
		}
		if len(p.MissingAPIs) > 0 || len(p.MissingOperationAdapters) > 0 {
			p.NextSteps = append(p.NextSteps, "Qualify and install a matching runner/collector in the selected DEV namespace, then repeat preflight before booting an app project.")
		}
		if !p.WorkflowAdapterAvailable {
			p.NextSteps = append(p.NextSteps, "The selected framework adapter is unavailable in this client; do not substitute raw shell or Dojo commands.")
		}
	}
	if req.InstanceID == "" {
		p.NextSteps = append(p.NextSteps, "After selecting a ship and desk, repeat preflight for its exact generation and review narrow permissions in Setup.")
		return p, nil
	}
	if policy == nil {
		if p.EnvironmentCompatible {
			return p, errors.New("supported runner returned no application policy")
		}
		return p, nil
	}
	if policy.RunnerID != h.RunnerID || policy.Mode != h.Mode || policy.Policy.Validate() != nil {
		return p, errors.New("application policy identity or format changed")
	}
	var target apppolicy.Target
	for _, t := range policy.Targets {
		if t.Ready && t.Target.InstanceID == req.InstanceID {
			target = t.Target
			target.Desk = req.Desk
			break
		}
	}
	if target.InstanceID == "" {
		if p.EnvironmentCompatible {
			p.Status = "target-unresolved"
		}
		p.NextSteps = append(p.NextSteps, "Resolve the selected ship's current generation before reviewing app effects.")
		return p, nil
	}
	check := &PolicyCheck{Revision: policy.Policy.Revision, ReadAt: policy.ReadAt, Target: target, Effective: policy.Policy.Effective(target), Blocked: []PolicyBlock{}}
	for _, effect := range req.Effects {
		decision := check.Effective.CreateDesk
		if effect == "commit" {
			decision = check.Effective.Commit
		}
		if effect == "publish" {
			decision = check.Effective.Publish
		}
		if decision.Value != apppolicy.Allow {
			check.Blocked = append(check.Blocked, PolicyBlock{Effect: effect, Scope: decision.Source})
		}
	}
	p.Policy = check
	if p.EnvironmentCompatible {
		p.Status = "saved-policy-compatible"
		if len(check.Blocked) > 0 {
			p.Status = "policy-denied"
		}
	}
	if len(check.Blocked) > 0 {
		p.NextSteps = append(p.NextSteps, "Review the named policy scopes in Setup and grant only the required ship/desk effects.")
	}
	p.NextSteps = append(p.NextSteps, "Review fresh desk, publication and application evidence before each effect; committing a published desk also requires Publish permission.")
	return p, nil
}

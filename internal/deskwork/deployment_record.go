package deskwork

import (
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"time"
)

// CommitIntent adds the frozen staged source and review to the ordinary durable
// desk effect intent. It carries no grant, expression, URL or caller host path.
type CommitIntent struct {
	Intent           Intent `json:"intent"`
	StageID          string `json:"stage_id"`
	SettingsRevision uint64 `json:"settings_revision"`
	SourceName       string `json:"source_name"`
	Continuity       string `json:"continuity"`
	PlanDigest       string `json:"plan_digest"`
}

func (i CommitIntent) Validate() error {
	if i.Intent.Validate() != nil || i.Intent.Action != "commit" || !digestPattern.MatchString(i.StageID) || i.SettingsRevision > apppolicy.MaxRevision || !ValidSourceName(i.SourceName) || !digestPattern.MatchString(i.Continuity) || !digestPattern.MatchString(i.PlanDigest) {
		return errors.New("invalid staged commit intent")
	}
	return nil
}

type DeploymentRecord struct {
	Version   int            `json:"version"`
	Intent    CommitIntent   `json:"intent"`
	Actor     string         `json:"actor"`
	Plan      DeploymentPlan `json:"plan"`
	CreatedAt time.Time      `json:"created_at"`
}

func (r DeploymentRecord) Validate() error {
	if r.Version != 1 || r.Intent.Validate() != nil || r.Actor == "" || !settingText(r.Actor, 128, false) || r.Plan.Validate() != nil || len(r.Plan.Conflicts) != 0 || len(r.Plan.Changes) == 0 || r.Plan.Digest != r.Intent.PlanDigest || r.Plan.Source.Digest != r.Intent.Intent.SourceDigest || r.CreatedAt.IsZero() {
		return errors.New("invalid durable deployment record")
	}
	return nil
}

// OwnedSource survives a runtime restart only when the same pier object, target
// generation and named source binding are verified. Moving to a different pier
// or source starts with no owned paths; it never silently adopts existing files.
type OwnedSource struct {
	Version     int              `json:"version"`
	Target      apppolicy.Target `json:"target"`
	SourceName  string           `json:"source_name"`
	Continuity  string           `json:"continuity"`
	OperationID string           `json:"operation_id"`
	Ownership   Ownership        `json:"ownership"`
	Observation Observation      `json:"observation"`
	ObservedAt  time.Time        `json:"observed_at"`
}

func (o OwnedSource) Validate() error {
	if o.Version != 1 || o.Target.Validate() != nil || o.Target.InstanceID == "" || o.Target.Desk == "" || o.Target.ProjectID != "" && o.Target.Generation == 0 || !ValidSourceName(o.SourceName) || !digestPattern.MatchString(o.Continuity) || !digestPattern.MatchString(o.OperationID) || o.Ownership.Validate() != nil || o.Observation.Validate() != nil || !o.Observation.Exists || o.ObservedAt.IsZero() {
		return errors.New("invalid committed source ownership")
	}
	return nil
}
func (o OwnedSource) Check(r DeploymentRecord, receipt Receipt) error {
	if o.Validate() != nil || r.Validate() != nil || receipt.Validate() != nil {
		return errors.New("invalid ownership evidence")
	}
	final := receipt.Steps[len(receipt.Steps)-1]
	if receipt.Intent != r.Intent.Intent || receipt.Actor != r.Actor || final.State != "observed" || final.Observation == nil || *final.Observation != o.Observation || final.VerifiedSourceDigest != r.Plan.Source.Digest || !final.At.Equal(o.ObservedAt) || o.Target != receipt.Intent.Target || o.OperationID != receipt.Intent.ID || o.SourceName != r.Intent.SourceName || o.Continuity != r.Intent.Continuity || len(o.Ownership.Raw.Files) != len(r.Plan.Managed) {
		return errors.New("ownership does not match observed deployment")
	}
	raw := r.Plan.Source.Index()
	for i, name := range r.Plan.Managed {
		if o.Ownership.Raw.Files[i] != raw[name] {
			return errors.New("ownership changed managed source")
		}
	}
	return nil
}

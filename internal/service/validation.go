package service

import (
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/appdev"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (i ProjectInput) Validate() error {
	if i.Target.Validate() != nil || !protocol.ValidID(i.ProjectID) {
		return invalid()
	}
	return nil
}
func (i CheckpointInput) Validate() error {
	if i.ProjectInput.Validate() != nil || !protocol.ValidID(i.OperationID) {
		return invalid()
	}
	return nil
}
func (i DiagnosisInput) Validate() error {
	if i.CheckpointInput.Validate() != nil || !protocol.ValidID(i.InstanceID) {
		return invalid()
	}
	return nil
}
func (i StopBundleInput) Validate() error {
	if i.CheckpointInput.Validate() != nil || !protocol.ValidID(i.InstanceID) {
		return invalid()
	}
	return nil
}
func (i StopReconcileInput) Validate() error {
	q := i.Request
	if i.ProjectInput.Validate() != nil || !protocol.ValidID(q.RequestID) || !q.ConfirmAction || q.Review.Validate() != nil || q.Review.ProjectID != i.ProjectID || q.Review.RunnerID != i.RunnerID {
		return invalid()
	}
	return nil
}
func (i ProjectActionInput) Validate() error {
	if i.ProjectInput.Validate() != nil || !protocol.ValidProjectActionRequest(i.Request) {
		return invalid()
	}
	return nil
}
func (i ProjectRecoverInput) Validate() error {
	if i.ProjectInput.Validate() != nil || !protocol.ValidProjectRecoveryAction(i.Kind, i.ParentID, i.Request) {
		return invalid()
	}
	return nil
}
func (i ProjectRegisterInput) Validate() error {
	q := i.Request
	if i.Target.Validate() != nil || !protocol.ValidID(q.RequestID) || !q.ConfirmReservation {
		return invalid()
	}
	req := projectnetwork.Request{Label: q.Label, ConfirmSharedAdministration: q.ConfirmSharedAdministration}
	for _, m := range q.Members {
		req.Members = append(req.Members, projectnetwork.MemberRequest{InstanceID: m.InstanceID, Ship: m.Ship, HTTPPort: m.HTTPPort})
	}
	p, err := projectnetwork.Build(req, projectnetwork.Reservations{})
	if err != nil {
		return invalid()
	}
	digest, err := p.Digest()
	if err != nil || digest != q.ReviewDigest {
		return invalid()
	}
	return nil
}
func (i PreflightInput) Validate() error {
	return (appdev.PreflightRequest{Name: i.Name, RunnerID: i.RunnerID, Workflow: i.Workflow, InstanceID: i.InstanceID, Desk: i.Desk, Effects: i.Effects}).Validate()
}
func (i DeskInput) Validate() error {
	if i.Target.Validate() != nil || i.InstanceID == "" || i.Desk == "" || (apppolicy.Target{InstanceID: i.InstanceID, Desk: i.Desk}).Validate() != nil {
		return invalid()
	}
	return nil
}
func (i DeskListInput) Validate() error {
	if i.Target.Validate() != nil || !protocol.ValidID(i.InstanceID) {
		return invalid()
	}
	return nil
}
func (i DeskSettingsInput) Validate() error {
	if i.Target.Validate() != nil || i.InstanceID == "" || (apppolicy.Target{InstanceID: i.InstanceID, Desk: i.Desk}).Validate() != nil {
		return invalid()
	}
	return nil
}
func (i StageInput) Validate() error {
	if i.Target.Validate() != nil || !protocol.ValidDeskOperationID(i.StageID) {
		return invalid()
	}
	return nil
}
func (i DeskOperationInput) Validate() error {
	if i.Target.Validate() != nil || !protocol.ValidDeskOperationID(i.OperationID) {
		return invalid()
	}
	return nil
}
func (i DeskCreateInput) Validate() error {
	if i.Target.Validate() != nil || i.Plan.Validate() != nil || i.Plan.RunnerID != i.RunnerID {
		return invalid()
	}
	return nil
}
func (i DeskCommitInput) Validate() error {
	if i.Target.Validate() != nil || i.Plan.Validate() != nil || i.Plan.RunnerID != i.RunnerID {
		return invalid()
	}
	return nil
}
func (i DeskAdoptInput) Validate() error {
	if i.Target.Validate() != nil || i.Plan.Validate() != nil || i.Plan.RunnerID != i.RunnerID {
		return invalid()
	}
	return nil
}
func (i DeskReleaseInput) Validate() error {
	if i.Target.Validate() != nil || i.Plan.Validate() != nil || i.Plan.RunnerID != i.RunnerID {
		return invalid()
	}
	return nil
}
func (i DeskFrontendInput) Validate() error {
	if i.Target.Validate() != nil || i.Plan.Validate() != nil || i.Plan.RunnerID != i.RunnerID {
		return invalid()
	}
	return nil
}
func (i DeskReceiverInput) Validate() error {
	if i.Target.Validate() != nil || i.Plan.Validate() != nil || i.Plan.RunnerID != i.RunnerID {
		return invalid()
	}
	return nil
}
func (i ReleaseReviewInput) Validate() error {
	if i.DeskInput.Validate() != nil || !deskwork.IsReleaseAction(i.Action) {
		return invalid()
	}
	return nil
}

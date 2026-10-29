//go:build linux && amd64

package managed

import (
	"context"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"log"
)

// Runtime adapter diagnostics contain fixed failure descriptions; the pure
// controller deliberately returns public generic errors. Keep this detail in
// bounded runner logs, never in application state or authentication responses.
type memberLaunchDiagnostic struct {
	projectnetwork.MemberLaunchDriver
	project, instance, operation string
}

func (d *memberLaunchDiagnostic) report(stage string, err error) {
	if err == nil {
		return
	}
	message := err.Error()
	if len(message) > 512 {
		message = message[:512]
	}
	log.Printf("project-member-launch project=%s instance=%s operation=%s stage=%s failed=%q", d.project, d.instance, d.operation, stage, message)
}
func (d *memberLaunchDiagnostic) VerifyEngine(ctx context.Context, id string) (err error) {
	defer func() { d.report("engine", err) }()
	return d.MemberLaunchDriver.VerifyEngine(ctx, id)
}
func (d *memberLaunchDiagnostic) PrepareWorkspace(ctx context.Context, s projectnetwork.MemberLaunch) (v pierstore.DirectoryIdentity, err error) {
	defer func() { d.report("workspace", err) }()
	return d.MemberLaunchDriver.PrepareWorkspace(ctx, s)
}
func (d *memberLaunchDiagnostic) Preflight(ctx context.Context, s projectnetwork.MemberLaunch) (err error) {
	defer func() { d.report("preflight", err) }()
	return d.MemberLaunchDriver.Preflight(ctx, s)
}
func (d *memberLaunchDiagnostic) Inspect(ctx context.Context, s projectnetwork.MemberLaunch) (v projectnetwork.Observation, err error) {
	defer func() { d.report("inspect", err) }()
	return d.MemberLaunchDriver.Inspect(ctx, s)
}
func (d *memberLaunchDiagnostic) Create(ctx context.Context, s projectnetwork.MemberLaunch) (id string, err error) {
	defer func() { d.report("create", err) }()
	return d.MemberLaunchDriver.Create(ctx, s)
}
func (d *memberLaunchDiagnostic) Start(ctx context.Context, s projectnetwork.MemberLaunch) (err error) {
	defer func() { d.report("start", err) }()
	return d.MemberLaunchDriver.Start(ctx, s)
}

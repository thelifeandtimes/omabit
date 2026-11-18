//go:build linux && amd64

package managed

import (
	"context"
	"path/filepath"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type grubberyRuntime interface {
	deskRuntime
	deskApplicationRuntime
	grubbery.NativeRuntime
	SourceIdentity(context.Context) (string, error)
}

// Caller holds r.mu throughout admission, durable dispatch, one effect and
// observation. actor and role come from authenticated transport, never JSON.
// This private factory does not register an API or advertise a capability.
func (r *Server) nativeGrubberyBackend(ctx context.Context, b grubbery.Binding, actor, role string) (*grubbery.NativeBackend, error) {
	// Construction grants no write permission. Exact reconciliation remains
	// available after Commit is revoked; Inspect/Admit/Dispatch recheck it.
	runtime, _, err := r.grubberyGuard(ctx, b, actor, role, false)
	if err != nil {
		return nil, err
	}
	guard := func(ctx context.Context, binding grubbery.Binding, write bool) (uint64, error) {
		if binding != b {
			return 0, grubbery.ErrConflict
		}
		_, revision, err := r.grubberyGuard(ctx, binding, actor, role, write)
		return revision, err
	}
	load := func(ctx context.Context, id string) ([]deskwork.SourceFile, error) {
		_, files, err := r.grubberyStageSource(ctx, b, id, actor)
		if err != nil {
			return nil, err
		}
		return files, nil
	}
	return grubbery.NewNativeBackend(runtime, (grubberyJournal{r.s}).History, guard, load)
}

func (r *Server) grubberyStageSource(ctx context.Context, b grubbery.Binding, id, actor string) (deskwork.Manifest, []deskwork.SourceFile, error) {
	stage, directory, err := deskwork.ReadStage(ctx, filepath.Join(r.s.directory, "desk-stages-v1"), id, actor)
	if err != nil || stage.Receipt.Intent.Target != b.Target {
		return deskwork.Manifest{}, nil, grubbery.ErrSource
	}
	setting, err := r.s.deskSetting(b.Target)
	if err != nil || setting.Revision != stage.Receipt.Intent.SettingsRevision || setting.Desk == nil || setting.Desk.Workflow != "mounted" {
		return deskwork.Manifest{}, nil, grubbery.ErrSource
	}
	files, err := deskwork.ReadSourceFiles(ctx, directory, stage.Manifest)
	if err != nil {
		return deskwork.Manifest{}, nil, grubbery.ErrSource
	}
	return stage.Manifest, files, nil
}

func (r *Server) grubberyGuard(ctx context.Context, b grubbery.Binding, actor, role string, write bool) (grubberyRuntime, uint64, error) {
	if b.Validate() != nil || actor != b.Actor || role != "operator" || b.RunnerID != r.s.runnerID || b.Mode != r.s.mode || r.closing || r.failed.Load() || r.updateFence.Held || !r.routineAuthorized(actor) || ctx.Err() != nil {
		return nil, 0, grubbery.ErrAdmission
	}
	if err := checkGrubberyFramework(r.s.db, b); err != nil {
		return nil, 0, err
	}
	if err := grubberyFrameworkQuiet(r.s.db, b); err != nil {
		return nil, 0, err
	}
	selection := protocol.DeskSelection{Target: b.Target, RuntimeID: b.RuntimeID, ConfigRevision: b.ConfigRevision}
	base, err := r.openDesk(ctx, selection)
	if err != nil {
		return nil, 0, grubbery.ErrAdmission
	}
	runtime, ok := base.(grubberyRuntime)
	if !ok {
		return nil, 0, grubbery.ErrAdmission
	}
	continuity, err := runtime.SourceIdentity(ctx)
	if err != nil || continuity != b.Continuity {
		return nil, 0, grubbery.ErrAdmission
	}
	before, err := runtime.Observe(ctx, b.Target.Desk)
	if err != nil || before != b.FrameworkBaseline {
		return nil, 0, grubbery.ErrAdmission
	}
	app, err := runtime.Application(ctx, b.Target.Desk)
	if err != nil || !grubberyFrameworkLive(app, b.FrameworkBaseline.Revision) {
		return nil, 0, grubbery.ErrAdmission
	}
	after, err := runtime.Observe(ctx, b.Target.Desk)
	if err != nil || after != before {
		return nil, 0, grubbery.ErrAdmission
	}
	current, err := runtime.SourceIdentity(ctx)
	if err != nil || current != continuity {
		return nil, 0, grubbery.ErrAdmission
	}
	policy, err := loadAppPolicy(r.s.db)
	if err != nil {
		return nil, 0, err
	}
	if write && policy.Check(b.Target, policy.Revision, "commit", after.Publication) != nil {
		return nil, 0, grubbery.ErrAdmission
	}
	return runtime, policy.Revision, nil
}

func grubberyFrameworkLive(app deskwork.AppState, revision uint64) bool {
	if app.Validate() != nil || app.Zest != "live" || app.SourceRevision != revision {
		return false
	}
	for _, name := range []string{"grubbery", "grub-client"} {
		expected, live := false, false
		for _, agent := range app.ExpectedAgents {
			if agent == name {
				expected = true
			}
		}
		for _, agent := range app.Agents {
			if agent.Name == name && agent.Live {
				live = true
			}
		}
		if !expected || !live {
			return false
		}
	}
	return true
}

// Source presence is not activation evidence. Require a separately observed
// supported activation of this exact framework baseline, and refuse another
// unresolved framework operation even if its current Clay hash still matches.
func grubberyFrameworkQuiet(q querier, b grubbery.Binding) error {
	ids, err := deskOperationIDs(q)
	if err != nil {
		return err
	}
	activated := false
	for _, id := range ids {
		op, err := loadDeskOperation(q, id)
		if err != nil {
			return err
		}
		if op.Intent.Target != b.Target {
			continue
		}
		last := op.Steps[len(op.Steps)-1]
		if last.State != "observed" && last.State != "blocked" {
			return grubbery.ErrAdmission
		}
		if op.Intent.Action != "activate-local" || last.State != "observed" || op.Intent.Baseline != b.FrameworkBaseline {
			continue
		}
		release, err := loadDeskRelease(q, id)
		if err != nil {
			return err
		}
		if release.Intent.Continuity != b.Continuity || last.Observation == nil || *last.Observation != b.FrameworkBaseline || last.Application == nil || !grubberyFrameworkLive(*last.Application, b.FrameworkBaseline.Revision) {
			continue
		}
		activated = true
	}
	if !activated {
		return grubbery.ErrAdmission
	}
	return nil
}

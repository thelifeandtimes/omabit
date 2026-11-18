//go:build linux && amd64

package managed

import (
	"context"
	"errors"
	"reflect"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func recoveryHeadDigest(receipt *grubbery.RuntimeRecoveryReceipt) string {
	if receipt == nil {
		return ""
	}
	return receipt.Digest
}

// Caller holds r.mu across review, apply or the original Execution call. A
// replacement selection never changes original binding/payload identity.
func (r *Server) grubberyRuntimeGuard(ctx context.Context, b grubbery.Binding, selection protocol.DeskSelection, process grubbery.RuntimeProcess, actor, role string, write bool) (grubberyRecoveryRuntime, uint64, error) {
	base, revision, err := r.grubberyGuardSelection(ctx, b, selection, actor, role, write)
	if err != nil {
		return nil, 0, err
	}
	runtime, ok := base.(grubberyRecoveryRuntime)
	if !ok {
		return nil, 0, grubbery.ErrAdmission
	}
	fresh, err := runtime.GrubberyProcess(ctx)
	if err != nil || fresh != process {
		return nil, 0, grubbery.ErrAdmission
	}
	return runtime, revision, nil
}

func (r *Server) reviewGrubberyRuntime(ctx context.Context, originalID string, selection protocol.DeskSelection, actor, role string) (grubbery.RuntimeRecoveryReview, error) {
	var out grubbery.RuntimeRecoveryReview
	if role != "operator" || !r.routineAuthorized(actor) {
		return out, grubbery.ErrAdmission
	}
	original, err := (grubberyJournal{r.s}).Load(ctx, originalID)
	if err != nil {
		return out, err
	}
	b := original.Plan.Binding
	if original.Version != 2 || original.Plan.Version != 2 {
		return out, grubbery.ErrLegacyPlan
	}
	previous, err := (grubberyRuntimeJournal{r.s}).Latest(ctx, originalID)
	if err != nil {
		return out, err
	}
	base, _, err := r.grubberyGuardSelection(ctx, b, selection, actor, role, false)
	if err != nil {
		return out, err
	}
	runtime, ok := base.(grubberyRecoveryRuntime)
	if !ok {
		return out, grubbery.ErrAdmission
	}
	process, err := runtime.GrubberyProcess(ctx)
	if err != nil || process.Validate() != nil {
		return out, grubbery.ErrAdmission
	}
	readonly, err := runtime.GrubberyRecoveryView(ctx, b, process)
	if err != nil {
		return out, grubbery.ErrAdmission
	}
	guard := func(ctx context.Context, binding grubbery.Binding, write bool) (uint64, error) {
		if binding != b || write {
			return 0, grubbery.ErrAdmission
		}
		_, revision, err := r.grubberyRuntimeGuard(ctx, b, selection, process, actor, role, false)
		return revision, err
	}
	backend, err := grubbery.NewNativeBackend(readonly, (grubberyJournal{r.s}).History, guard, func(context.Context, string) ([]deskwork.SourceFile, error) { return nil, grubbery.ErrSource })
	if err != nil {
		return out, err
	}
	evidence, err := backend.ObserveRuntimeRecovery(ctx, original)
	if err != nil {
		return out, err
	}
	if _, _, err = r.grubberyRuntimeGuard(ctx, b, selection, process, actor, role, false); err != nil {
		return out, err
	}
	fresh, err := (grubberyJournal{r.s}).Load(ctx, originalID)
	if err != nil {
		return out, err
	}
	head, err := (grubberyRuntimeJournal{r.s}).Latest(ctx, originalID)
	if err != nil {
		return out, err
	}
	if !reflect.DeepEqual(fresh, original) || recoveryHeadDigest(head) != recoveryHeadDigest(previous) {
		return out, grubbery.ErrCAS
	}
	step, done, err := original.Current()
	if err != nil || done {
		return out, grubbery.ErrPhase
	}
	state, err := original.State(step.ID)
	if err != nil {
		return out, err
	}
	eventDigest := original.HeaderDigest
	if len(original.Events) != 0 {
		eventDigest = original.Events[len(original.Events)-1].Digest
	}
	previousContainer := b.RuntimeID
	if previous != nil {
		previousContainer = previous.Review.Process.ContainerID
	}
	out = grubbery.RuntimeRecoveryReview{Version: 1, OriginalRequestID: originalID, PlanDigest: original.Plan.Digest, HeaderDigest: original.HeaderDigest, OriginalRevision: original.Revision, OriginalEventDigest: eventDigest, StepID: step.ID, State: state, OriginalBinding: b, PreviousRecoveryDigest: recoveryHeadDigest(previous), PreviousContainerID: previousContainer, Process: process, EvidenceBasis: evidence.EvidenceBasis, NativeEvidenceDigest: evidence.NativeEvidenceDigest, AppliedPhaseProof: evidence.AppliedPhaseProof, ObservedAt: time.Now().UTC()}
	out.Digest = out.ContentDigest()
	return out, out.ValidatePredecessor(original, previous)
}

func (r *Server) applyGrubberyRuntime(ctx context.Context, originalID, id string, review grubbery.RuntimeRecoveryReview, actor, role string) (grubbery.RuntimeRecoveryReceipt, error) {
	var out grubbery.RuntimeRecoveryReceipt
	if role != "operator" || !r.routineAuthorized(actor) || !runtimeRecoveryID(id) || id == originalID || review.Digest != review.ContentDigest() || review.OriginalRequestID != originalID || review.OriginalBinding.Actor != actor || review.OriginalBinding.RunnerID != r.s.runnerID || review.OriginalBinding.Mode != r.s.mode {
		return out, grubbery.ErrConflict
	}
	j := grubberyRuntimeJournal{r.s}
	if saved, err := j.Load(ctx, originalID, id); err == nil {
		if saved.Review.Digest != review.Digest {
			return out, grubbery.ErrConflict
		}
		return saved, nil
	} else if !errors.Is(err, grubbery.ErrMissing) {
		return out, err
	}
	original, err := (grubberyJournal{r.s}).Load(ctx, originalID)
	if err != nil {
		return out, err
	}
	head, err := j.Latest(ctx, originalID)
	if err != nil {
		return out, err
	}
	if err = review.MatchesCurrent(original, recoveryHeadDigest(head), time.Now().UTC()); err != nil {
		return out, err
	}
	b, err := review.EffectiveBinding(original)
	if err != nil {
		return out, err
	}
	selection := protocol.DeskSelection{Target: b.Target, RuntimeID: b.RuntimeID, ConfigRevision: b.ConfigRevision}
	fresh, err := r.reviewGrubberyRuntime(ctx, originalID, selection, actor, role)
	if err != nil {
		return out, err
	}
	now := time.Now().UTC()
	if err = review.MatchesFreshReview(original, recoveryHeadDigest(head), fresh, now); err != nil {
		return out, err
	}
	return j.Apply(ctx, originalID, id, review, now)
}

func (r *Server) nativeGrubberyBackendFor(ctx context.Context, original grubbery.Record, actor, role string) (*grubbery.NativeBackend, error) {
	head, err := (grubberyRuntimeJournal{r.s}).Latest(ctx, original.Plan.RequestID)
	if err != nil {
		return nil, err
	}
	if head == nil {
		return r.nativeGrubberyBackend(ctx, original.Plan.Binding, actor, role)
	}
	b, err := head.Review.EffectiveBinding(original)
	if err != nil {
		return nil, err
	}
	selection := protocol.DeskSelection{Target: b.Target, RuntimeID: b.RuntimeID, ConfigRevision: b.ConfigRevision}
	runtime, _, err := r.grubberyRuntimeGuard(ctx, original.Plan.Binding, selection, head.Review.Process, actor, role, false)
	if err != nil {
		return nil, err
	}
	adapter, err := runtime.GrubberyRecovered(ctx, original, *head)
	if err != nil {
		return nil, grubbery.ErrAdmission
	}
	guard := func(ctx context.Context, binding grubbery.Binding, write bool) (uint64, error) {
		if binding != original.Plan.Binding {
			return 0, grubbery.ErrConflict
		}
		fresh, err := (grubberyRuntimeJournal{r.s}).Latest(ctx, original.Plan.RequestID)
		if err != nil || recoveryHeadDigest(fresh) != head.Digest {
			return 0, grubbery.ErrCAS
		}
		_, revision, err := r.grubberyRuntimeGuard(ctx, binding, selection, head.Review.Process, actor, role, write)
		return revision, err
	}
	return r.grubberyBackend(adapter, original.Plan.Binding, guard)
}

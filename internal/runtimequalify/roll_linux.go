//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"fmt"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/epochs"
)

func isRollKind(kind string) bool {
	return kind == "roll-1" || kind == "roll-2" || kind == "roll-3" || kind == "roll-4"
}
func isEpochProfile(profile string) bool {
	return profile == ProfileEpochs || profile == ProfilePrune || prefixProfile(profile) || cometEpochProfile(profile)
}
func epochRollCount(profile string) int {
	if prefixProfile(profile) {
		return 4
	}
	return 3
}
func isEpochRestartKind(kind string) bool {
	return kind == "epoch-restart-1" || kind == "epoch-restart-2" || kind == "epoch-restart-3" || kind == "epoch-restart-4"
}

func (q *qualifier) epochAdmission() (uint64, uint64, error) {
	mem, disk, err := q.d.admission(q.r.Directory)
	if err != nil || mem < 5<<30 || disk < 8<<30 {
		return 0, 0, errors.New("epoch work admission requires MemAvailable >= 5 GiB and free disk >= 8 GiB")
	}
	return mem, disk, nil
}

func (q *qualifier) epochBoot(ctx context.Context) error {
	if q.d.managed || !isEpochProfile(q.d.profile) || cometProfile(q.d.profile) {
		return errors.New("epoch boot requires disposable qualification")
	}
	if _, _, err := q.epochAdmission(); err != nil {
		return err
	}
	return q.boot(ctx, "boot")
}

func (q *qualifier) epochBaseline(ctx context.Context) error {
	if q.d.managed || !isEpochProfile(q.d.profile) || !q.r.LimitsObserved || q.r.EpochLifecycle != nil {
		return errors.New("epoch baseline requires a fresh disposable epoch qualification")
	}
	if cometEpochProfile(q.d.profile) && (q.r.Comet == nil || !q.r.Comet.BootCompleted || !q.r.Comet.Uncertain) {
		return errors.New("comet epoch baseline requires its fresh completed boot")
	}
	for _, c := range q.r.Containers {
		if !c.Removed || c.CreateUncertain {
			return errors.New("epoch baseline requires resolved stopped containers")
		}
	}
	inv, err := q.inspectEpochs(ctx)
	if err != nil {
		return err
	}
	if err := validEpochSample(inv, q.r.RunID, 1); err != nil {
		return err
	}
	if inv.Entries[0].Name != "0i0" {
		return errors.New("fresh qualification must begin with epoch 0i0")
	}
	q.r.EpochLifecycle = &EpochLifecycleRecord{Initial: inv, Rolls: []EpochRollRecord{}}
	return q.save()
}

func (q *qualifier) roll(ctx context.Context, kind string) error {
	e := q.r.EpochLifecycle
	if q.d.managed || !isEpochProfile(q.d.profile) || !isRollKind(kind) || !q.r.LimitsObserved || e == nil || len(e.Rolls) >= epochRollCount(q.d.profile) || kind != fmt.Sprintf("roll-%d", len(e.Rolls)+1) {
		return errors.New("offline roll requires the next disposable qualification step")
	}
	if len(e.Rolls) > 0 {
		last := e.Rolls[len(e.Rolls)-1]
		if last.Uncertain || last.FinishedAt == nil || last.PostRestart == nil {
			return errors.New("previous roll and restart must be resolved before another roll")
		}
	}
	for _, c := range q.r.Containers {
		if !c.Removed || c.CreateUncertain {
			return errors.New("offline roll requires all preceding containers removed")
		}
	}
	before, err := q.inspectEpochs(ctx)
	if err != nil {
		return err
	}
	previous := e.Initial
	if len(e.Rolls) > 0 {
		previous = e.Rolls[len(e.Rolls)-1].PostRestart
	}
	if validEpochSample(before, q.r.RunID, len(e.Rolls)+1) != nil || !sameEpochNames(previous, before) {
		return errors.New("epoch set changed before the next roll")
	}
	mem, disk, err := q.epochAdmission()
	if err != nil {
		return err
	}
	e.Rolls = append(e.Rolls, EpochRollRecord{Kind: kind, StartedAt: time.Now().UTC(), Before: before, MemAvailableBytes: mem, DiskAvailableBytes: disk, Uncertain: true})
	r := &e.Rolls[len(e.Rolls)-1]
	if err := q.save(); err != nil {
		return err
	}
	if err := q.offlineUtility(ctx, epochContainerKind(q.d.profile, kind)); err != nil {
		return err
	}
	after, err := q.inspectEpochs(ctx)
	if err != nil {
		return err
	}
	r.After = after // retain observed evidence even when validation leaves uncertainty
	if err := validEpochSample(after, q.r.RunID, len(before.Entries)+1); err != nil {
		return fmt.Errorf("roll inventory/snapshot validation failed: observed %d epochs, expected %d complete epochs", len(after.Entries), len(before.Entries)+1)
	}
	if !epochPrefix(before, after) {
		return errors.New("roll changed or removed a preceding epoch")
	}
	now := time.Now().UTC()
	r.After, r.FinishedAt, r.Uncertain = after, &now, false
	if err := q.save(); err != nil {
		r.After, r.FinishedAt, r.Uncertain = nil, nil, true
		return err
	}
	return nil
}

func (q *qualifier) epochRestart(ctx context.Context, roll, kind string) error {
	e := q.r.EpochLifecycle
	if q.d.managed || !isEpochProfile(q.d.profile) || e == nil || len(e.Rolls) == 0 || !isEpochRestartKind(kind) || kind != fmt.Sprintf("epoch-restart-%d", len(e.Rolls)) {
		return errors.New("invalid post-roll restart")
	}
	r := &e.Rolls[len(e.Rolls)-1]
	if r.Kind != roll || r.Uncertain || r.After == nil || r.FinishedAt == nil || r.PostRestart != nil {
		return errors.New("post-roll restart requires a resolved utility outcome")
	}
	if _, _, err := q.epochAdmission(); err != nil {
		return err
	}
	boot := q.boot
	if cometEpochProfile(q.d.profile) {
		boot = q.cometRuntime
	}
	if err := boot(ctx, epochContainerKind(q.d.profile, kind)); err != nil {
		return err
	}
	inv, err := q.inspectEpochs(ctx)
	if err != nil {
		return err
	}
	if validEpochSample(inv, q.r.RunID, len(r.After.Entries)) != nil || !sameEpochNames(r.After, inv) {
		return errors.New("epoch set changed across post-roll restart")
	}
	r.PostRestart = inv
	if err := q.save(); err != nil {
		r.PostRestart = nil
		return err
	}
	return nil
}

func (q *qualifier) epochPreview(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	e := q.r.EpochLifecycle
	rolls := epochRollCount(q.d.profile)
	if q.d.managed || !isEpochProfile(q.d.profile) || e == nil || len(e.Rolls) != rolls {
		return errors.New("prefix preview requires all planned roll/restart steps")
	}
	last := e.Rolls[rolls-1]
	if last.Uncertain || last.PostRestart == nil {
		return errors.New("final roll/restart outcome is unresolved")
	}
	inv, err := q.inspectEpochs(ctx)
	if err != nil {
		return err
	}
	if validEpochSample(inv, q.r.RunID, rolls+1) != nil || !sameEpochNames(last.PostRestart, inv) {
		return errors.New("final epoch inventory changed")
	}
	selected := rolls - 2
	p, err := epochs.Plan(inv, inv.Entries[selected-1].Name, inv.Digest, time.Now().UTC())
	if err != nil {
		return err
	}
	if len(p.Selected) != selected || len(p.Retained) != 3 || p.DeletionSupported {
		return errors.New("prefix preview did not preserve the chosen three-epoch suffix")
	}
	e.Final, e.Preview = inv, &p
	return q.save()
}

func sameEpochNames(a, b *epochs.Inventory) bool {
	return a != nil && b != nil && len(a.Entries) == len(b.Entries) && epochPrefix(a, b)
}
func epochPrefix(a, b *epochs.Inventory) bool {
	if a == nil || b == nil || len(a.Entries) > len(b.Entries) {
		return false
	}
	for i, e := range a.Entries {
		if e.Name != b.Entries[i].Name {
			return false
		}
	}
	return true
}

// Used for persisted qualification evidence as well as newly inspected results.
func validEpochSample(inv *epochs.Inventory, id string, count int) error {
	if inv == nil || inv.Source != epochs.Source || inv.PierID != id || inv.ObservedAt.IsZero() || !epochs.ValidDigest(inv.Digest) || count < 1 || count > 5 || len(inv.Entries) != count {
		return errors.New("invalid qualification epoch inventory")
	}
	var previous uint64
	for i, e := range inv.Entries {
		n, err := epochs.Number(e.Name)
		if err != nil || (i > 0 && n <= previous) || e.Protected != (i >= count-epochs.ProtectedSuffix) || e.LogicalBytes < 0 || e.AllocatedBytes < 0 || e.SnapshotBytes < 0 || (n > 0 && e.SnapshotBytes == 0) || e.FileCount < 3 || e.FileCount > epochs.MaxFilesPerEpoch {
			return errors.New("invalid qualification epoch entry")
		}
		previous = n
	}
	return nil
}

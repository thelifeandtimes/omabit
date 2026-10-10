//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"path/filepath"
	"time"
)

func (q *qualifier) meld(ctx context.Context) error {
	if q.d.managed || q.d.profile != ProfileMeld || !q.r.LimitsObserved || q.r.Meld != nil {
		return errors.New("meld requires a fresh disposable qualification and verified limits")
	}
	for _, c := range q.r.Containers {
		if !c.Removed || c.CreateUncertain {
			return errors.New("meld requires all preceding containers removed with resolved outcomes")
		}
	}
	if err := q.pier(); err != nil {
		return err
	}
	measure := func() (int64, error) {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		return pierBytes(ctx, filepath.Join(q.r.Directory, "data/fake-zod"))
	}
	before, err := measure()
	if err != nil {
		return err
	}
	// Boot can take minutes. Re-sample capacity immediately before the utility,
	// using physical available memory rather than swap or the initial observation.
	mem, disk, err := q.d.admission(q.r.Directory)
	if err != nil || mem < 5<<30 || disk < 8<<30 {
		return errors.New("meld admission requires MemAvailable >= 5 GiB and free disk >= 8 GiB")
	}
	m := &MeldRecord{StartedAt: time.Now().UTC(), BeforePierBytes: before,
		MemAvailableBytes: mem, DiskAvailableBytes: disk, Uncertain: true}
	q.r.Meld = m
	if err := q.save(); err != nil {
		return err
	}
	if err := q.offlineUtility(ctx, "meld"); err != nil {
		return err
	}
	if err := q.pier(); err != nil {
		return err
	}
	after, err := measure()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	m.FinishedAt, m.AfterPierBytes, m.Uncertain = &now, &after, false
	if err := q.save(); err != nil {
		// A failed final persistence is not proof of a durable clean outcome.
		// Final cleanup/reporting must retain the conservative review requirement.
		m.FinishedAt, m.AfterPierBytes, m.Uncertain = nil, nil, true
		return err
	}
	return nil
}

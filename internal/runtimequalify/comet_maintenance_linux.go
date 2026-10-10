//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"path/filepath"
	"time"
)

var errCometIdentityChanged = errors.New("public comet identity is invalid or changed across runtime replacement")

func isCometUtility(kind string) bool { return kind == "comet-pack" || kind == "comet-meld" }

func isManagedCometUtility(kind string) bool { return isCometUtility(kind) || kind == "comet-roll" }

func (q *qualifier) cometMaintenanceResumeReady(kind string) error {
	if kind == "comet-resume" {
		if len(q.r.CometUtilities) != 2 || !q.r.CometUtilities[0].RestartVerified || !q.r.CometUtilities[1].RestartVerified {
			return errors.New("final comet resume requires both verified maintenance restarts")
		}
		return nil
	}
	index := 0
	want := "comet-pack"
	if kind == "comet-post-meld" {
		index = 1
		want = "comet-meld"
	}
	if len(q.r.CometUtilities) != index+1 {
		return errors.New("comet restart lacks its maintenance record")
	}
	m := q.r.CometUtilities[index]
	if m.Kind != want || m.Uncertain || m.FinishedAt == nil || m.RestartVerified {
		return errors.New("comet utility outcome is not ready for restart")
	}
	return nil
}

func (q *qualifier) cometUtility(ctx context.Context, kind string) error {
	if q.d.managed || q.d.profile != ProfileCometMaintenance || q.r.Comet == nil || !q.r.Comet.BootCompleted || !q.r.Comet.Uncertain {
		return errors.New("comet maintenance requires its fresh disposable workflow")
	}
	index := len(q.r.CometUtilities)
	if index == 0 && kind != "comet-pack" || index == 1 && (kind != "comet-meld" || !q.r.CometUtilities[0].RestartVerified) || index > 1 {
		return errors.New("comet utilities must execute exactly once in order")
	}
	for _, c := range q.r.Containers {
		if !c.Removed || c.CreateUncertain {
			return errors.New("comet utility requires all previous containers removed")
		}
	}
	if err := q.cometPier(); err != nil {
		return err
	}
	measure := func() (int64, error) {
		c, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		return pierBytes(c, filepath.Join(q.r.Directory, "data/comet"))
	}
	before, err := measure()
	if err != nil {
		return err
	}
	mem, disk, err := q.d.admission(q.r.Directory)
	if err != nil || mem < 5<<30 || disk < 8<<30 {
		return errors.New("comet utility requires 5 GiB available RAM and 8 GiB free disk")
	}
	q.r.CometUtilities = append(q.r.CometUtilities, CometUtilityRecord{Kind: kind, MeldRecord: MeldRecord{StartedAt: time.Now().UTC(), BeforePierBytes: before, MemAvailableBytes: mem, DiskAvailableBytes: disk, Uncertain: true}})
	if err := q.save(); err != nil {
		return err
	}
	if err := q.offlineUtility(ctx, kind); err != nil {
		return err
	}
	if err := q.cometPier(); err != nil {
		return err
	}
	after, err := measure()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	m := &q.r.CometUtilities[index]
	m.FinishedAt = &now
	m.AfterPierBytes = &after
	m.Uncertain = false
	if err := q.save(); err != nil {
		m.Uncertain = true
		m.FinishedAt = nil
		m.AfterPierBytes = nil
		return err
	}
	return nil
}

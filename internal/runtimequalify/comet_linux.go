//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"
)

func (q *qualifier) cometPier() error {
	data := filepath.Join(q.r.Directory, "data")
	entries, err := os.ReadDir(data)
	if err != nil || len(entries) != 1 || entries[0].Name() != "comet" {
		return errors.New("comet data mount must contain only its new pier")
	}
	for _, p := range []string{data, filepath.Join(data, "comet"), filepath.Join(data, "comet/.urb")} {
		if err := q.owned(p, os.ModeDir, false); err != nil {
			return errors.New("comet pier metadata unavailable or unsafe")
		}
	}
	q.r.PierMetadataSeen = true
	return nil
}

func (q *qualifier) cometRuntime(ctx context.Context, kind string) error {
	if q.d.managed || !cometProfile(q.d.profile) || !(isCometKind(kind) || q.d.profile == ProfileCometMaintenance && isCometRestart(kind) || cometEpochContainer(q.d.profile, kind) && isCometEpochRestart(kind) || q.d.profile == ProfileCometPrefix && isCometPrefixRuntime(kind)) || !q.r.LimitsObserved {
		return errors.New("comet bootstrap requires its isolated qualification profile and live limits")
	}
	for _, c := range q.r.Containers {
		if !c.Removed || c.CreateUncertain {
			return errors.New("previous qualification runtime is unresolved")
		}
	}
	mem, disk, err := q.d.admission(q.r.Directory)
	if err != nil || mem < 5<<30 || disk < 8<<30 {
		return errors.New("comet bootstrap/resume requires 5 GiB available RAM and 8 GiB free disk")
	}
	if kind == "comet-boot" {
		entries, err := os.ReadDir(filepath.Join(q.r.Directory, "data"))
		if err != nil || len(entries) != 0 || q.r.Comet != nil {
			return errors.New("comet creation requires empty new storage")
		}
		q.r.Comet = &CometRecord{StartedAt: time.Now().UTC(), Uncertain: true}
		if err := q.save(); err != nil {
			return err
		}
	} else {
		if q.d.profile == ProfileCometMaintenance {
			if err := q.cometMaintenanceResumeReady(kind); err != nil {
				return err
			}
		}
		if q.d.profile == ProfileCometEpochs {
			if err := q.cometEpochResumeReady(kind); err != nil {
				return err
			}
		}
		if q.d.profile == ProfileCometPrefix {
			if err := q.cometPrefixResumeReady(kind); err != nil {
				return err
			}
		}
		if q.r.Comet == nil || !q.r.Comet.BootCompleted || !cometName.MatchString(q.r.Comet.Ship) || q.r.Comet.ResumeCompleted {
			return errors.New("comet resume requires its completed initial boot")
		}
		if err := q.cometPier(); err != nil {
			return err
		}
	}
	index, err := q.create(ctx, kind)
	if err != nil {
		return err
	}
	if _, err := q.control(ctx, index, q.d.times.command, "start"); err != nil {
		return err
	}
	if err := q.readyComet(ctx, index); err != nil {
		return err
	}
	if err := q.stop(ctx, index, true); err != nil {
		return err
	}
	if err := q.cometPier(); err != nil {
		return err
	}
	if kind == "comet-boot" {
		q.r.Comet.BootCompleted = true
	} else if kind == "comet-resume" {
		if q.d.profile == ProfileCometPrefix {
			if err := q.cometPrefixFinish(ctx); err != nil {
				return err
			}
		}
		if q.d.profile == ProfileCometEpochs {
			// Reinspect after the final replacement, then freeze a preview-only
			// selection before clearing overall qualification uncertainty.
			if err := q.epochPreview(ctx); err != nil {
				return err
			}
		}
		now := time.Now().UTC()
		q.r.Comet.ResumeCompleted = true
		q.r.Comet.FinishedAt = &now
		q.r.Comet.Uncertain = false
	} else if isCometRestart(kind) {
		index := 0
		if kind == "comet-post-meld" {
			index = 1
		}
		q.r.CometUtilities[index].RestartVerified = true
	}
	return q.save()
}

func (q *qualifier) readyComet(ctx context.Context, index int) error {
	ship, err := q.readyCometIdentity(ctx, index, q.r.Comet.Ship)
	if err != nil {
		return err
	}
	q.r.Comet.Ship = ship
	return q.save()
}

func (q *qualifier) readyCometIdentity(ctx context.Context, index int, expected string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, q.d.times.ready)
	defer cancel()
	for ctx.Err() == nil {
		c, err := q.inspect(ctx, index)
		if err != nil {
			return "", err
		}
		if !*c.State.Running || c.State.Paused || c.State.Restarting || c.State.Dead || *c.State.OOMKilled {
			return "", errors.New("comet runtime exited before identity readiness")
		}
		if _, err := q.inspectNetwork(ctx); err != nil {
			return "", err
		}
		q.r.Containers[index].AliveObserved = true
		endpoint, err := q.publishedEndpoint(c, q.r.Containers[index].Kind)
		if err != nil && !errors.Is(err, errBindingPending) {
			return "", err
		}
		if err == nil {
			q.r.Containers[index].HostEndpoint = endpoint
			if err := q.save(); err != nil {
				return "", err
			}
			ship, readErr := q.d.hostIdentity(ctx, endpoint)
			if readErr == nil {
				valid := cometName.MatchString(ship)
				if q.d.managed && q.d.nativeShip != "" {
					valid = ship == q.d.nativeShip
				}
				if !valid || expected != "" && ship != expected {
					return "", errCometIdentityChanged
				}
				after, err := q.inspect(ctx, index)
				if err != nil {
					return "", err
				}
				if !*after.State.Running || after.State.Paused || after.State.Restarting || after.State.Dead || *after.State.OOMKilled {
					return "", errors.New("comet runtime exited during identity check")
				}
				again, err := q.publishedEndpoint(after, q.r.Containers[index].Kind)
				if err != nil || again != endpoint {
					return "", errors.New("comet binding changed during identity check")
				}
				if _, err := q.inspectNetwork(ctx); err != nil {
					return "", err
				}
				q.r.Containers[index].HostHTTPVerified = true
				return ship, q.save()
			}
			if !errors.Is(readErr, errHostNotReady) {
				return "", errors.New("comet identity endpoint failed response-policy checks")
			}
		}
		timer := time.NewTimer(q.d.times.poll)
		select {
		case <-ctx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
	return "", errors.New("comet bootstrap/identity deadline elapsed; no automatic retry")
}

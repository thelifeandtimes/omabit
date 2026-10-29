//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"fmt"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"time"
)

// AwaitInitialProjectReadiness tolerates a freshly started Vere process still
// creating its pier and HTTP listener. Opening the observer is itself read-only
// and can fail while the child/parent layout is settling, so it belongs inside
// the bounded observation loop too. No Create or Start is repeated here.
// The caller subsequently collects its own fresh proof under the same fence.
func AwaitInitialProjectReadiness(ctx context.Context, s ProjectInitialReadinessSelection) (projectnetwork.InitialReadinessDriver, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	return awaitProjectReadiness(ctx, 2*time.Second, func(ctx context.Context) (projectnetwork.InitialReadinessDriver, error) {
		return OpenInitialProjectReadiness(ctx, s)
	}, func(ctx context.Context, d projectnetwork.InitialReadinessDriver) error {
		for _, m := range s.Project.Plan.Members {
			if _, err := projectnetwork.CollectInitialMemberReadiness(ctx, s.Project.Plan, s.Launches, m.InstanceID, d, time.Now); err != nil {
				return err
			}
		}
		return nil
	})
}

func awaitProjectReadiness(ctx context.Context, interval time.Duration, open func(context.Context) (projectnetwork.InitialReadinessDriver, error), collect func(context.Context, projectnetwork.InitialReadinessDriver) error) (projectnetwork.InitialReadinessDriver, error) {
	var last error
	for {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("project readiness observation ended: %w (last check: %v)", ctx.Err(), last)
		}
		d, err := open(ctx)
		if err == nil && d == nil {
			err = errors.New("project readiness observer unavailable")
		}
		if err == nil {
			err = collect(ctx, d)
		}
		if err == nil && ctx.Err() == nil {
			return d, nil
		}
		last = err
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, fmt.Errorf("project readiness observation ended: %w (last check: %v)", ctx.Err(), last)
		case <-timer.C:
		}
	}
}

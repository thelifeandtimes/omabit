//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"testing"
	"time"
)

type waitingReadinessDriver struct {
	projectnetwork.InitialReadinessDriver
}

func TestProjectReadinessWaitIncludesObserverSetup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	opened, collected := 0, 0
	want := &waitingReadinessDriver{}
	got, err := awaitProjectReadiness(ctx, time.Millisecond, func(context.Context) (projectnetwork.InitialReadinessDriver, error) {
		opened++
		if opened == 1 {
			return nil, errors.New("pier parent still settling")
		}
		return want, nil
	}, func(context.Context, projectnetwork.InitialReadinessDriver) error {
		collected++
		if collected == 1 {
			return errors.New("HTTP identity not ready")
		}
		return nil
	})
	if err != nil || got != want || opened != 3 || collected != 2 {
		t.Fatal(got, err, opened, collected)
	}
}
func TestProjectReadinessWaitCancellationNeverReturnsProof(t *testing.T) {
	for _, phase := range []string{"before-open", "during-open", "during-collection", "permanent-refusal"} {
		t.Run(phase, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			defer cancel()
			opened, collected := 0, 0
			if phase == "before-open" {
				cancel()
			}
			got, err := awaitProjectReadiness(ctx, time.Millisecond, func(context.Context) (projectnetwork.InitialReadinessDriver, error) {
				opened++
				if phase == "during-open" {
					cancel()
					return nil, context.Canceled
				}
				if phase == "permanent-refusal" {
					return nil, errors.New("ownership changed")
				}
				return &waitingReadinessDriver{}, nil
			}, func(context.Context, projectnetwork.InitialReadinessDriver) error { collected++; cancel(); return nil })
			if got != nil || err == nil || (!errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)) {
				t.Fatal(got, err)
			}
			if phase == "before-open" && opened != 0 || phase != "during-collection" && collected != 0 {
				t.Fatal(opened, collected)
			}
		})
	}
}

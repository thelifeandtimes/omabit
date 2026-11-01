package collection

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestCadenceBroadcastCoalescingAndValidation(t *testing.T) {
	c, err := NewWatchCadence(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	interval, refresh, changed := c.read()
	if interval != time.Hour || refresh != 0 {
		t.Fatal(interval, refresh)
	}
	if err = c.Update(time.Millisecond, true); err == nil {
		t.Fatal("accepted busy polling")
	}
	if _, err = NewWatchCadence(-time.Second); err == nil {
		t.Fatal("accepted negative interval")
	}
	if err = c.Update(time.Hour, false); err != nil {
		t.Fatal(err)
	}
	select {
	case <-changed:
		t.Fatal("no-op update woke watchers")
	default:
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var wg sync.WaitGroup
	results := make(chan error, 8)
	for n := 0; n < 8; n++ {
		wg.Add(1)
		go func() { defer wg.Done(); results <- c.wait(ctx, time.Now(), refresh, 0) }()
	}
	c.Update(100*time.Millisecond, true)
	// Events need no reader to make progress and do not build an unbounded queue.
	for n := 0; n < 10000; n++ {
		c.Update(time.Hour, true)
	}
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal("broadcast missed a runner", err)
		}
	}
	select {
	case <-changed:
	default:
		t.Fatal("old revision not notified")
	}
}

func TestCadenceLongerWaitCancellationAndFailureBackoff(t *testing.T) {
	c, _ := NewWatchCadence(100 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 160*time.Millisecond)
	defer cancel()
	go func() { time.Sleep(20 * time.Millisecond); c.Update(time.Hour, false) }()
	if err := c.wait(ctx, time.Now(), 0, 0); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("slowing cadence caused an extra observation", err)
	}
	c, _ = NewWatchCadence(100 * time.Millisecond)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 160*time.Millisecond)
	defer cancel2()
	if err := c.wait(ctx2, time.Now(), 0, 1); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("failure lost exponential delay", err)
	}
}

func TestCadenceShorterWaitUsesElapsedTimeAndRefreshWakesImmediately(t *testing.T) {
	c, _ := NewWatchCadence(time.Hour)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	start := time.Now()
	c.Update(100*time.Millisecond, false)
	if err := c.wait(ctx, start.Add(-time.Second), 0, 0); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > 50*time.Millisecond {
		t.Fatal("shorter cadence ignored elapsed wait")
	}
	c.Update(time.Hour, true)
	if err := c.wait(ctx, time.Now(), 0, 8); err != nil {
		t.Fatal("explicit opening refresh was blocked", err)
	}
}

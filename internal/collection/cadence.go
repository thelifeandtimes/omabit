package collection

import (
	"context"
	"errors"
	"sync"
	"time"
)

// WatchCadence adjusts only the observation schedule, never the runner selection
// or transport identity. Updates are coalesced and broadcast to all runners.
// A visibility change therefore need not discard authenticated connections.
type WatchCadence struct {
	mu       sync.Mutex
	interval time.Duration
	refresh  uint64
	changed  chan struct{}
}

func NewWatchCadence(interval time.Duration) (*WatchCadence, error) {
	if interval < 100*time.Millisecond {
		return nil, errors.New("inventory interval must be at least 100ms")
	}
	return &WatchCadence{interval: interval, changed: make(chan struct{})}, nil
}

func (c *WatchCadence) Update(interval time.Duration, refresh bool) error {
	if interval < 100*time.Millisecond {
		return errors.New("inventory interval must be at least 100ms")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if interval == c.interval && !refresh {
		return nil
	}
	c.interval = interval
	if refresh {
		c.refresh++
	}
	close(c.changed)
	c.changed = make(chan struct{})
	return nil
}

func (c *WatchCadence) read() (time.Duration, uint64, <-chan struct{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.interval, c.refresh, c.changed
}

// wait uses the last completed poll, so slowing an idle watcher extends its
// existing wait instead of triggering another poll. Only an explicit refresh
// bypasses the delay. Repeated failures retain bounded exponential backoff.
func (c *WatchCadence) wait(ctx context.Context, completed time.Time, refresh uint64, failures uint) error {
	timer := time.NewTimer(time.Hour)
	defer timer.Stop()
	for {
		interval, current, changed := c.read()
		if current != refresh {
			return ctx.Err()
		}
		delay := interval
		for n := uint(0); n < failures && delay < 30*time.Second; n++ {
			delay = min(30*time.Second, delay*2)
		}
		remaining := time.Until(completed.Add(delay))
		if remaining <= 0 {
			return ctx.Err()
		}
		timer.Reset(remaining)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-changed:
		case <-timer.C:
			return nil
		}
	}
}

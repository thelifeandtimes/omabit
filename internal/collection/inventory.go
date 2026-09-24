package collection

import (
	"context"
	"errors"
	"sync"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// Observation keeps instance IDs scoped to a connection and stable runner ID.
// Unavailable runners have no current snapshot; absence is not a ship crash.
type Observation struct {
	Name       string             `json:"name"`
	RunnerID   string             `json:"runner_id"`
	Mode       string             `json:"mode"`
	AccessRole string             `json:"access_role,omitempty"`
	Features   []string           `json:"features,omitempty"`
	Status     string             `json:"status"`
	CheckedAt  time.Time          `json:"checked_at"`
	Snapshot   *protocol.Snapshot `json:"snapshot,omitempty"`
	Error      *protocol.APIError `json:"error,omitempty"`
}

type Inventory struct {
	Connections []Observation `json:"connections"`
	Complete    bool          `json:"complete"`
}

func inspect(ctx context.Context, entry Connection) Observation {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	c := entry.Client(true)
	defer c.Close()
	result := Observation{Name: entry.Name, RunnerID: entry.RunnerID, Mode: entry.Mode, Status: "available"}
	// Retain capabilities/access for UI affordances; the runner remains the
	// authorization authority and rechecks every mutation.
	hello, err := c.Hello(ctx)
	var snapshot protocol.Snapshot
	if err == nil {
		err = entry.Identity().Check(hello.RunnerID, hello.Mode)
	}
	if err == nil {
		result.AccessRole = hello.AccessRole
		result.Features = hello.Features
		snapshot, err = c.Snapshot(ctx)
	}
	if err == nil {
		err = entry.Identity().Check(snapshot.RunnerID, snapshot.Mode)
	}
	result.CheckedAt = time.Now().UTC()
	if err == nil {
		result.Snapshot = &snapshot
		return result
	}
	result.Status = "unavailable"
	problem := protocol.APIError{Code: "connection_unavailable", Message: "runner could not be observed; ship state is unknown"}
	if errors.Is(err, client.ErrIdentityMismatch) {
		result.Status = "identity_mismatch"
		problem = protocol.APIError{Code: "runner_identity_mismatch", Message: client.ErrIdentityMismatch.Error()}
	} else if errors.Is(err, context.DeadlineExceeded) {
		problem.Code = "connection_timeout"
	}
	// Do not copy arbitrary remote error bodies or transport diagnostics into the
	// combined inventory; they can contain data belonging to another connection.
	result.Error = &problem
	return result
}

// Collect contacts connections concurrently with a five-second per-runner budget.
func Collect(ctx context.Context, entries []Connection) Inventory {
	result := Inventory{Connections: make([]Observation, len(entries)), Complete: true}
	var wg sync.WaitGroup
	for i, entry := range entries {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result.Connections[i] = inspect(ctx, entry)
		}()
	}
	wg.Wait()
	for _, observation := range result.Connections {
		if observation.Status != "available" {
			result.Complete = false
		}
	}
	return result
}

// Watch emits one observation per completed poll. Each runner owns its polling
// loop, so an unavailable/slow runner cannot delay healthy runners' next poll.
// Entries are fixed for this invocation; restarting picks up enrollment changes.
func Watch(ctx context.Context, entries []Connection, interval time.Duration, emit func(Observation) error) error {
	if interval < 100*time.Millisecond {
		return errors.New("inventory interval must be at least 100ms")
	}
	if len(entries) == 0 {
		return errors.New("no enrolled runners to watch")
	}
	ctx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	defer func() { cancel(); wg.Wait() }()
	updates := make(chan Observation, len(entries))
	for _, entry := range entries {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ctx.Err() == nil {
				observation := inspect(ctx, entry)
				select {
				case updates <- observation:
				case <-ctx.Done():
					return
				}
				timer := time.NewTimer(interval)
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
			}
		}()
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case observation := <-updates:
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if err := emit(observation); err != nil {
				return err
			}
		}
	}
}

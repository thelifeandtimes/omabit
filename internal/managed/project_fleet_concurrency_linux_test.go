//go:build linux && amd64

package managed

import (
	"context"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type blockedFleetBackend struct {
	*queuedProjectBackend
	entered chan struct{}
	release chan struct{}
}

func (b *blockedFleetBackend) ProjectFleet(ctx context.Context, s runtimequalify.ProjectCurrentReadinessSelection) ([]protocol.Instance, error) {
	close(b.entered)
	select {
	case <-b.release:
		return []protocol.Instance{{ID: s.Project.Plan.Members[0].InstanceID}}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
func (*blockedFleetBackend) ProjectLoginCode(context.Context, runtimequalify.ProjectCurrentReadinessSelection, string, string) (string, error) {
	return "", nil
}

func TestProjectFleetSlowObservationDoesNotHoldMutationLock(t *testing.T) {
	r, original, _, _ := projectQueueFixture(t)
	p, _, _ := readyStopProject(t, r.s)
	b := &blockedFleetBackend{original, make(chan struct{}), make(chan struct{})}
	r.b = b
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var snapshot protocol.Snapshot
	done := make(chan struct{})
	go func() { r.attachProjectFleet(ctx, &snapshot); close(done) }()
	select {
	case <-b.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("fleet observation did not begin")
	}
	unlocked := make(chan struct{})
	go func() { r.mu.Lock(); r.mu.Unlock(); close(unlocked) }()
	select {
	case <-unlocked:
	case <-time.After(time.Second):
		t.Error("slow observation blocks lifecycle admission")
	}
	close(b.release)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("fleet observation did not finish")
	}
	found := false
	for _, row := range snapshot.Instances {
		if row.ID == p.Plan.Members[0].InstanceID {
			found = true
		}
	}
	if !found {
		t.Fatal(snapshot.Instances)
	}
}

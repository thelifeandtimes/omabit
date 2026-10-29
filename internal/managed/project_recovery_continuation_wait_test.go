//go:build linux && amd64

package managed

import (
	"context"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"reflect"
	"testing"
	"time"
)

type uncertainDrainWaitProbe struct {
	calls   int
	saved   projectnetwork.AnchorDrain
	forever bool
	t       *testing.T
}

func (p *uncertainDrainWaitProbe) InspectDrain(ctx context.Context, s projectnetwork.AnchorDrain) (projectnetwork.AnchorDrainObservation, error) {
	p.calls++
	if !reflect.DeepEqual(s, p.saved) {
		p.t.Fatal("wait changed saved uncertainty")
	}
	// Intentionally mutate the passed copy: the next attempt must retain the intent.
	s.Members[0].Phase = "exited"
	if p.forever || p.calls == 1 {
		return projectnetwork.AnchorDrainObservation{}, errors.New("still running")
	}
	return projectnetwork.AnchorDrainObservation{}, nil
}
func TestVMInterruptedDrainWaitUsesOnlyObservations(t *testing.T) {
	saved := projectnetwork.AnchorDrain{Members: []projectnetwork.AnchorDrainMember{{Phase: "stop-dispatched"}}}
	p := &uncertainDrainWaitProbe{saved: saved.Clone(), t: t}
	if err := vmWaitInterruptedDrainExit(context.Background(), p, saved); err != nil {
		t.Fatal(err)
	}
	if p.calls != 2 || saved.Members[0].Phase != "stop-dispatched" || saved.Members[0].SignalAcknowledged {
		t.Fatal(p.calls, saved)
	}
	p = &uncertainDrainWaitProbe{saved: saved.Clone(), forever: true, t: t}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	if err := vmWaitInterruptedDrainExit(ctx, p, saved); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	if p.calls > 1 {
		t.Fatal(p.calls)
	}
}

//go:build linux && amd64

package managed

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func appRuntimeReadinessSnapshot() protocol.Snapshot {
	return protocol.Snapshot{RunnerID: "runner", Mode: "development-managed", Instances: []protocol.Instance{{ID: "member", Ship: "~zod", ConfigRevision: apppolicy.MaxRevision, ObservedState: "running", Resources: &protocol.ResourceMetrics{ContainerID: strings.Repeat("a", 64), Status: "available"}}}}
}

func TestRuntimeVMReadinessWaitsForSocketAndActualPinnedProcess(t *testing.T) {
	good := appRuntimeReadinessSnapshot()
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	calls := 0
	got, err := appRuntimeVMWaitRunning(ctx, good.RunnerID, good.Mode, good.Instances[0], func(ctx context.Context) (protocol.Snapshot, error) {
		calls++
		switch calls {
		case 1:
			return protocol.Snapshot{}, fmt.Errorf("wrapped snapshot transport: %w", os.ErrNotExist)
		case 2:
			return protocol.Snapshot{}, fmt.Errorf("wrapped snapshot transport: %w", syscall.ECONNREFUSED)
		case 3:
			return protocol.Snapshot{}, io.EOF
		case 4:
			return protocol.Snapshot{}, io.ErrUnexpectedEOF
		case 5:
			unready := appRuntimeReadinessSnapshot()
			unready.Instances[0].Resources.ContainerID = ""
			return unready, nil
		default:
			return good, nil
		}
	}, time.Microsecond)
	if err != nil || got.Resources == nil || got.Resources.ContainerID != good.Instances[0].Resources.ContainerID || calls != 6 {
		t.Fatal("readiness skipped pinning, process evidence or transient startup", got, calls, err)
	}
}

func TestRuntimeVMReadinessRefusesInvalidExpectationWithoutProbe(t *testing.T) {
	good := appRuntimeReadinessSnapshot()
	good.Instances[0].ConfigRevision = apppolicy.MaxRevision + 1
	_, err := appRuntimeVMWaitRunning(t.Context(), good.RunnerID, good.Mode, good.Instances[0], func(context.Context) (protocol.Snapshot, error) {
		t.Fatal("invalid expectation was probed")
		return protocol.Snapshot{}, nil
	}, time.Second)
	if err == nil {
		t.Fatal("invalid expectation became a readiness wait")
	}
}

func TestRuntimeVMReadinessDoesNotRetryIdentityProtocolOrPermissionFailures(t *testing.T) {
	for _, scenario := range []string{"runner", "mode", "instance", "ship", "revision", "extra-instance", "permission", "protocol"} {
		t.Run(scenario, func(t *testing.T) {
			good, changed := appRuntimeReadinessSnapshot(), appRuntimeReadinessSnapshot()
			var failure error
			switch scenario {
			case "runner":
				changed.RunnerID = "replacement"
			case "mode":
				changed.Mode = "replacement"
			case "instance":
				changed.Instances[0].ID = "unrelated"
			case "ship":
				changed.Instances[0].Ship = "~nec"
			case "revision":
				changed.Instances[0].ConfigRevision++
			case "extra-instance":
				changed.Instances = append(changed.Instances, changed.Instances[0])
			case "permission":
				failure = os.ErrPermission
			case "protocol":
				failure = errors.New("incompatible protocol")
			}
			calls := 0
			_, err := appRuntimeVMWaitRunning(t.Context(), good.RunnerID, good.Mode, good.Instances[0], func(context.Context) (protocol.Snapshot, error) { calls++; return changed, failure }, time.Second)
			if err == nil || calls != 1 || failure != nil && !errors.Is(err, failure) {
				t.Fatal("changed selection or nontransient refusal was retried", calls, err)
			}
		})
	}
}

func TestRuntimeVMReadinessCancellationBoundsProbeAndPause(t *testing.T) {
	for _, scenario := range []string{"before", "probe", "pause"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if scenario == "before" {
				cancel()
			}
			good := appRuntimeReadinessSnapshot()
			calls := 0
			_, err := appRuntimeVMWaitRunning(ctx, good.RunnerID, good.Mode, good.Instances[0], func(probe context.Context) (protocol.Snapshot, error) {
				calls++
				cancel()
				if scenario == "probe" {
					<-probe.Done()
					return protocol.Snapshot{}, probe.Err()
				}
				return protocol.Snapshot{}, os.ErrNotExist
			}, time.Hour)
			if !errors.Is(err, context.Canceled) || scenario == "before" && calls != 0 || scenario != "before" && calls != 1 {
				t.Fatal("readiness ignored its bounded context", calls, err)
			}
		})
	}
}

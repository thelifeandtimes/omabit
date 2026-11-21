//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
)

func recordingSupervisorFixture(t *testing.T) (ManagedRecordingJob, RecordingWorkerIdentity, string) {
	t.Helper()
	job, _, _, m := recordingJobFixture(t)
	unit, err := RecordingWorkerUnitName(job)
	if err != nil {
		t.Fatal(err)
	}
	i := RecordingWorkerIdentity{ExecutableSHA256: job.ExecutableSHA256, PID: 4242, StartTicks: 123, InvocationID: strings.Repeat("a", 32), HostBootID: m.Identity.HostBootID, Namespaces: m.Identity.RecorderNamespaces}
	properties := "Id=" + unit + "\nType=notify\nNotifyAccess=main\nActiveState=active\nSubState=running\nMainPID=4242\nInvocationID=" + i.InvocationID + "\nControlGroup=/user.slice/app.slice/" + unit + "\n"
	return job, i, properties
}

func TestRecordingSupervisorProbeBracketsExactProcessAndRefusesChangedBindings(t *testing.T) {
	for _, scenario := range []string{"pass", "read-fails", "second-read-fails", "changed-unit", "pid-reused", "wrong-cgroup", "process-fails", "canceled", "legacy", "different-main"} {
		t.Run(scenario, func(t *testing.T) {
			job, claimed, properties := recordingSupervisorFixture(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			reads, probes := 0, 0
			if scenario == "legacy" {
				claimed.StartTicks, claimed.InvocationID = 0, ""
			}
			unitRead := func(ctx context.Context, name string) ([]byte, error) {
				reads++
				want, _ := RecordingWorkerUnitName(job)
				if name != want {
					t.Fatal("selected unrelated unit")
				}
				if scenario == "read-fails" || scenario == "second-read-fails" && reads == 2 {
					return nil, errors.New("private command detail")
				}
				raw := properties
				if scenario == "different-main" {
					raw = strings.Replace(raw, "MainPID=4242", "MainPID=4243", 1)
				}
				if scenario == "changed-unit" && reads == 2 {
					raw = strings.Replace(raw, claimed.InvocationID, strings.Repeat("b", 32), 1)
				}
				return []byte(raw), nil
			}
			processRead := func(ctx context.Context, unit RecordingSupervisorState) (RecordingWorkerIdentity, string, error) {
				probes++
				current, group := claimed, unit.ControlGroup
				if scenario == "pid-reused" {
					current.StartTicks++
				}
				if scenario == "wrong-cgroup" {
					group += "/child"
				}
				if scenario == "process-fails" {
					return current, group, errors.New("private process detail")
				}
				if scenario == "canceled" {
					cancel()
				}
				return current, group, nil
			}
			_, err := observeRecordingSupervisor(ctx, job, claimed, true, unitRead, processRead)
			if scenario == "pass" {
				if err != nil || reads != 2 || probes != 1 {
					t.Fatal("no exact bracket", reads, probes, err)
				}
			} else if !errors.Is(err, ErrRecordingWorker) {
				t.Fatal("stale/unready worker admitted or private error leaked", err)
			}
			if (scenario == "legacy" && reads != 0) || (scenario == "different-main" && probes != 0) {
				t.Fatal("probed without an exact live identity")
			}
		})
	}
}

func TestRecordingSupervisorStartupStateCannotGrantReadyAuthority(t *testing.T) {
	job, worker, raw := recordingSupervisorFixture(t)
	raw = strings.ReplaceAll(strings.ReplaceAll(raw, "ActiveState=active", "ActiveState=activating"), "SubState=running", "SubState=start")
	if _, err := ParseRecordingSupervisorState(job, []byte(raw)); !errors.Is(err, ErrRecordingWorker) {
		t.Fatal("activating unit granted readiness")
	}
	unit, err := observeRecordingSupervisor(t.Context(), job, worker, false,
		func(context.Context, string) ([]byte, error) { return []byte(raw), nil },
		func(_ context.Context, unit RecordingSupervisorState) (RecordingWorkerIdentity, string, error) {
			return worker, unit.ControlGroup, nil
		})
	if err != nil || unit.MainPID != worker.PID {
		t.Fatal("exact startup main process refused", err)
	}
	if _, err := parseRecordingSupervisorState(job, []byte(raw), true); err == nil {
		t.Fatal("startup authority promoted")
	}
}

func TestRecordingSupervisorProbeInputBounds(t *testing.T) {
	job, _, _ := recordingSupervisorFixture(t)
	unit, _ := RecordingWorkerUnitName(job)
	for _, name := range []string{"../other.service", "--all", unit + "\n", strings.Replace(unit, ".service", ".scope", 1), strings.ToUpper(unit)} {
		if validRecordingUnitName(name) {
			t.Fatal("unsafe unit accepted", name)
		}
	}
	var out recordingSupervisorOutput
	if _, err := out.Write(bytes.Repeat([]byte("a"), 8192)); err != nil {
		t.Fatal(err)
	}
	if n, err := out.Write([]byte("b")); n != 0 || err == nil || out.Len() != 8192 {
		t.Fatal("unbounded command output", n, err)
	}
	for _, raw := range []string{"0::/\n", "0::/unit\n1:cpu:/unit\n", "1:cpu:/unit\n", "0::/unit/../other\n", "0::/unit\x00\n"} {
		if _, err := recordingProcessCgroup([]byte(raw)); err == nil {
			t.Fatal("ambiguous cgroup accepted")
		}
	}
	if group, err := recordingProcessCgroup([]byte("0::/user.slice/worker.service\n")); err != nil || group != "/user.slice/worker.service" {
		t.Fatal("ordinary unified cgroup refused", err)
	}
	for _, raw := range []string{"4243 (worker) S " + strings.Repeat("0 ", 18) + "123", "4242 (worker) Z " + strings.Repeat("0 ", 18) + "123", "4242 (worker) T " + strings.Repeat("0 ", 18) + "123"} {
		if _, err := recordingProcessStat([]byte(raw), 4242); err == nil {
			t.Fatal("wrong/dead process accepted")
		}
	}
}

func TestRecordingProcessObserverReadsOnlySelfAndPreservesKernelIdentity(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("non-root owner boundary")
	}
	q := &qualifier{d: defaults()}
	self, err := observeRecordingWorkerIdentity(q)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		t.Fatal(err)
	}
	group, err := recordingProcessCgroup(raw)
	if err != nil {
		t.Skip("self cgroup is not an absolute unified user service context")
	}
	// Synthetic supervisor descriptor, actual self proc. This is explicitly not
	// a unit/service qualification and reads no other process or environment.
	unit := RecordingSupervisorState{MainPID: uint32(os.Getpid()), InvocationID: strings.Repeat("a", 32), ControlGroup: group}
	current, actualGroup, err := observeRecordingProcess(t.Context(), unit)
	self.InvocationID = unit.InvocationID
	if err != nil || current != self || actualGroup != group {
		t.Fatal("self identity changed", err)
	}
	unit.ControlGroup = "/different.service"
	if _, _, err = observeRecordingProcess(t.Context(), unit); !errors.Is(err, ErrRecordingWorker) {
		t.Fatal("actual cgroup mismatch accepted", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, _, err = observeRecordingProcess(ctx, unit); err == nil {
		t.Fatal("canceled observation accepted")
	}
	stat, _ := os.ReadFile("/proc/self/stat")
	if ticks, err := recordingProcessStat(stat, uint32(os.Getpid())); err != nil || ticks != self.StartTicks {
		t.Fatal("self stat differs", strconv.FormatUint(ticks, 10), err)
	}
}

//go:build linux && amd64

package runtimequalify

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecordingSupervisorUnitNamespaceBindsRunnerModeRootAndJob(t *testing.T) {
	job, _, _, _ := recordingJobFixture(t)
	want, err := RecordingWorkerUnitName(job)
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"runner", "mode", "root", "job"} {
		changed := job
		switch scenario {
		case "runner":
			changed.Operation.RunnerID = "other"
		case "mode":
			changed.Operation.Mode = "development-managed-fake"
		case "root":
			changed.State.Runtime.Directory += "-other"
			changed.State.Runtime.PillPath = filepath.Join(changed.State.Runtime.Directory, filepath.Base(job.State.Runtime.PillPath))
		case "job":
			changed.ID = strings.Repeat("4", 32)
		}
		got, err := RecordingWorkerUnitName(changed)
		if err != nil || got == want {
			t.Fatal("recording units overlapped namespaces", scenario, got, err)
		}
	}
}

func TestRecordingSupervisorRequiresNotifyMainAndExactLiveProcess(t *testing.T) {
	job, _, _, _ := recordingJobFixture(t)
	want, err := RecordingWorkerUnitName(job)
	if err != nil {
		t.Fatal(err)
	}
	properties := "Id=" + want + "\nType=notify\nNotifyAccess=main\nActiveState=active\nSubState=running\nMainPID=4242\nInvocationID=" + strings.Repeat("a", 32) + "\nControlGroup=/user.slice/app.slice/" + want + "\n"
	unit, err := ParseRecordingSupervisorState(job, []byte(properties))
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, m := recordingJobFixture(t)
	worker := RecordingWorkerIdentity{ExecutableSHA256: job.ExecutableSHA256, PID: 4242, StartTicks: 123, InvocationID: unit.InvocationID, HostBootID: m.Identity.HostBootID, Namespaces: m.Identity.RecorderNamespaces}
	if err = CheckRecordingSupervisor(job, unit, worker, worker); err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"exec", "all", "activating", "dead", "wrong-unit", "missing", "duplicate", "extra", "zero-pid", "alternate-pid", "no-invocation", "wrong-cgroup"} {
		changed := properties
		switch scenario {
		case "exec":
			changed = strings.ReplaceAll(changed, "Type=notify", "Type=exec")
		case "all":
			changed = strings.ReplaceAll(changed, "NotifyAccess=main", "NotifyAccess=all")
		case "activating":
			changed = strings.ReplaceAll(changed, "ActiveState=active", "ActiveState=activating")
		case "dead":
			changed = strings.ReplaceAll(changed, "SubState=running", "SubState=dead")
		case "wrong-unit":
			changed = strings.ReplaceAll(changed, "Id="+want, "Id=unrelated.service")
		case "missing":
			changed = strings.ReplaceAll(changed, "Type=notify\n", "")
		case "duplicate":
			changed += "MainPID=4242\n"
		case "extra":
			changed += "Unexpected=private\n"
		case "zero-pid":
			changed = strings.ReplaceAll(changed, "MainPID=4242", "MainPID=0")
		case "alternate-pid":
			changed = strings.ReplaceAll(changed, "MainPID=4242", "MainPID=04242")
		case "no-invocation":
			changed = strings.ReplaceAll(changed, "InvocationID="+worker.InvocationID, "InvocationID=")
		case "wrong-cgroup":
			changed = strings.ReplaceAll(changed, "/user.slice/app.slice/"+want, "/user.slice/app.slice/unrelated.service")
		}
		if _, err = ParseRecordingSupervisorState(job, []byte(changed)); !errors.Is(err, ErrRecordingWorker) {
			t.Fatal("unready/wrong supervisor accepted", scenario, err)
		}
	}
	for _, scenario := range []string{"pid-reuse", "executable", "boot", "namespace", "invocation", "legacy"} {
		current := worker
		claimed := worker
		switch scenario {
		case "pid-reuse":
			current.StartTicks++
		case "executable":
			current.ExecutableSHA256 = strings.Repeat("f", 64)
		case "boot":
			current.HostBootID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
		case "namespace":
			current.Namespaces.PID++
		case "invocation":
			current.InvocationID = strings.Repeat("b", 32)
		case "legacy":
			claimed.StartTicks = 0
			claimed.InvocationID = ""
		}
		if err = CheckRecordingSupervisor(job, unit, claimed, current); !errors.Is(err, ErrRecordingWorker) {
			t.Fatal("changed live worker admitted", scenario, err)
		}
	}
}

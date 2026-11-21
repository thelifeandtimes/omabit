//go:build linux && amd64

package managed

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/localrpc"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

const issuesVMControl = "/home/review/issues-recovery-helper"

// Set only when freezing the test helper. Go test binaries do not carry Go's
// automatic VCS settings; runtime admission requires this explicit source pin.
var issuesVMSourceRevision string

func TestIssuesStopVMBuildIdentity(t *testing.T) {
	if len(issuesVMSourceRevision) != 40 {
		t.Skip("unfrozen test helper")
	}
	t.Log("OMABIT_ISSUES_HELPER_SOURCE", issuesVMSourceRevision)
}

type issuesVMBackend struct {
	realBackend
	mode, project string
}

func issuesVMWrite(name string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(issuesVMControl, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err = f.Write(append(b, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

func (b issuesVMBackend) ProjectShipAction(ctx context.Context, s *runtimequalify.ProjectExpansion, id, action, operation string, save func(runtimequalify.ProjectExpansion) error) error {
	if b.mode != "interrupt-stop" || action != "stop" {
		return b.realBackend.ProjectShipAction(ctx, s, id, action, operation, save)
	}
	if s.Project.ProjectID != b.project || id != "issues-added-nec" {
		return errors.New("VM fault target changed")
	}
	return b.realBackend.ProjectShipAction(ctx, s, id, action, operation, func(next runtimequalify.ProjectExpansion) error {
		for _, m := range next.Members {
			if m.Member.InstanceID == id && m.Phase == "stopped" {
				// Native code has observed the exact clean exit. Retain the
				// already committed stop-dispatched ledger by blocking BEFORE
				// the final save. The parent kills only this owned helper.
				if m.StopIntent == nil || m.StopIntent.OperationID != operation {
					return errors.New("VM stop lacks original dispatch binding")
				}
				if err := issuesVMWrite("clean-exit-before-save.json", map[string]any{"project": b.project, "member": id, "operation": operation, "native_observed_phase": m.Phase, "container": m.Creator.Container.ID, "pier": m.Initialized}); err != nil {
					return err
				}
				<-ctx.Done()
				return ctx.Err()
			}
		}
		return save(next)
	})
}

func (b issuesVMBackend) ExpandedProjectStopObservation(ctx context.Context, s runtimequalify.ProjectExpansion, id string) (protocol.ProjectStopObservation, error) {
	if b.mode == "lookup-no-observer" {
		_ = issuesVMWrite("unexpected-recovery-observer.json", map[string]string{"project": s.Project.ProjectID, "member": id})
		return protocol.ProjectStopObservation{}, errors.New("VM exact receipt lookup must not observe runtime")
	}
	return b.realBackend.ExpandedProjectStopObservation(ctx, s, id)
}

// This is a dependency-injected manager, NOT the installed runner artifact.
// Its only modes are selected by the private guarded guest; no production CLI
// flag or general command/path/signal interface is introduced.
func TestIssuesStopRecoveryDisposableVM(t *testing.T) {
	mode := os.Getenv("OMABIT_ISSUES_VM_HELPER")
	if mode == "" {
		t.Skip("explicit disposable VM helper only")
	}
	vendor, e1 := os.ReadFile("/sys/class/dmi/id/sys_vendor")
	cmdline, e2 := os.ReadFile("/proc/cmdline")
	marker, e3 := os.ReadFile("/opt/omabit-release-input/scenario.json")
	var scenario struct {
		Kind         string `json:"kind"`
		SourceCommit string `json:"source_commit"`
	}
	if e1 != nil || e2 != nil || e3 != nil || json.Unmarshal(marker, &scenario) != nil || scenario.Kind != "issues16-21-development-v1" || len(issuesVMSourceRevision) != 40 || scenario.SourceCommit != issuesVMSourceRevision || strings.TrimSpace(string(vendor)) != "QEMU" || !strings.Contains(" "+string(cmdline)+" ", " omabit.disposable-guest=release-upgrade-v1 ") || os.Geteuid() != 1000 || os.Getenv("HOME") != "/home/review" || (mode != "interrupt-stop" && mode != "lookup-no-observer") {
		t.Fatal("refused outside the explicit fresh offline issues guest")
	}
	var control struct {
		Project string `json:"project"`
	}
	data, err := os.ReadFile(filepath.Join(issuesVMControl, "selection.json"))
	if err != nil || json.Unmarshal(data, &control) != nil || !protocol.ValidID(control.Project) {
		t.Fatal("missing exact VM project selection")
	}
	host, err := runtimequalify.OpenManagedHost()
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	if host.Directory != "/home/review/.local/share/urbit-runner/managed" || host.Socket != "/run/user/1000/urbit-runner/runner.sock" {
		t.Fatal("VM owner paths changed")
	}
	if info, err := os.Lstat(host.Socket); err == nil {
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok || st.Uid != 1000 || st.Nlink != 1 || info.Mode().Type() != os.ModeSocket || info.Mode().Perm() != 0600 {
			t.Fatal("unexpected retained VM socket")
		}
		c, dialErr := net.DialTimeout("unix", host.Socket, 500*time.Millisecond)
		if dialErr == nil {
			c.Close()
			t.Fatal("VM manager still listening")
		}
		if !errors.Is(dialErr, syscall.ECONNREFUSED) {
			t.Fatal("VM socket ownership uncertain")
		}
		if err := os.Remove(host.Socket); err != nil {
			t.Fatal(err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	r, err := newServerMode(host.Directory, issuesVMBackend{mode: mode, project: control.Project}, time.Hour, true)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	r.keys = access.DefaultKeys
	expansion, err := loadProjectExpansion(r.s.db, control.Project)
	if err != nil || len(expansion.Members) != 2 {
		t.Fatal("VM project is not the exact two fresh fakes")
	}
	for _, m := range expansion.Members {
		if (m.Member.InstanceID != "issues-original-zod" || m.Member.Ship != "~zod") && (m.Member.InstanceID != "issues-added-nec" || m.Member.Ship != "~nec") {
			t.Fatal("unexpected VM member")
		}
	}
	l, err := localrpc.Listen(host.Socket)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	if err := os.Chmod(host.Socket, os.ModeSticky|0600); err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: r, ConnContext: PeerContext, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second}
	defer server.Close()
	if err := issuesVMWrite(mode+"-ready.json", map[string]any{"pid": os.Getpid(), "mode": mode, "project": control.Project, "source_commit": issuesVMSourceRevision}); err != nil {
		t.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	done := make(chan error, 1)
	go func() { done <- server.Serve(l) }()
	select {
	case err := <-done:
		t.Fatal("VM helper server ended unexpectedly", err)
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			t.Fatal(err)
		}
		if err := <-done; !errors.Is(err, http.ErrServerClosed) {
			t.Fatal(err)
		}
	}
}

//go:build linux && amd64 && omabit_disposable_qualification

package managed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/localrpc"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

// Present only in the explicitly tagged test binary. It preserves the actual
// manager's queued operation, save callbacks, Stop intent and terminal receipt.
// The normal runner has neither this backend nor this entry point.
type projectRaceVMBackend struct {
	realBackend
	project string
}

func (b projectRaceVMBackend) ProjectShipAction(ctx context.Context, s *runtimequalify.ProjectExpansion, id, action, operation string, save func(runtimequalify.ProjectExpansion) error) error {
	if s == nil || s.Project.ProjectID != b.project || id != "issues-race-bud" || action != "stop" {
		return errors.New("native race helper refuses an unselected action")
	}
	target := ""
	for _, m := range s.Members {
		if m.Member.InstanceID == id {
			target = m.Creator.Container.ID
		}
	}
	var mu sync.Mutex
	sequence := 0
	return runtimequalify.RunDisposableProjectStopRace(ctx, s, id, operation, issuesVMSourceRevision, save, func(event runtimequalify.DisposableProjectStopRaceEvent) error {
		switch event.Step {
		case "pause-intent", "paused", "term-dispatched", "preliminary-running", "continued", "exited-before-bracket", "cleanup-continued":
		default:
			return errors.New("unknown native race trace step")
		}
		if event.ContainerID != target || event.OperationID != operation || event.StartedAt.IsZero() || event.At.IsZero() {
			return errors.New("changed native race trace binding")
		}
		mu.Lock()
		defer mu.Unlock()
		sequence++
		return issuesVMWrite(fmt.Sprintf("race-%02d-%s.json", sequence, event.Step), event)
	})
}

// Environment, complete VM/source guard and fixed member selection precede any
// store/socket opening. This helper is never used as an installed runner.
func TestIssuesProjectRaceDisposableVM(t *testing.T) {
	if os.Getenv("OMABIT_PROJECT_RACE_VM") == "" {
		t.Skip("explicit separately pinned disposable native race helper only")
	}
	if err := runtimequalify.CheckDisposableProjectStopRaceVM(t.Context(), issuesVMSourceRevision); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(issuesVMControl)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		t.Fatal("private fresh race control directory required", err)
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || st.Uid != 1000 || st.Gid != 1000 {
		t.Fatal("foreign race control directory")
	}
	selection := filepath.Join(issuesVMControl, "selection.json")
	info, err = os.Lstat(selection)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Size() > 4096 {
		t.Fatal("private exact race selection required", err)
	}
	st, ok = info.Sys().(*syscall.Stat_t)
	if !ok || st.Uid != 1000 || st.Nlink != 1 {
		t.Fatal("foreign or linked race selection")
	}
	raw, err := os.ReadFile(selection)
	if err != nil {
		t.Fatal(err)
	}
	var control struct {
		Project string `json:"project"`
	}
	if json.Unmarshal(raw, &control) != nil || !protocol.ValidID(control.Project) {
		t.Fatal("invalid exact race project")
	}
	host, err := runtimequalify.OpenManagedHost()
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	if host.Directory != "/home/review/.local/share/urbit-runner/managed" || host.Socket != "/run/user/1000/urbit-runner/runner.sock" {
		t.Fatal("race owner paths changed")
	}
	// The acquired host lock excludes the installed manager. Remove only its
	// exact owned stale socket, using the same restrictions as the older helper.
	if info, err := os.Lstat(host.Socket); err == nil {
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok || st.Uid != 1000 || st.Nlink != 1 || info.Mode().Type() != os.ModeSocket || info.Mode().Perm() != 0600 {
			t.Fatal("unexpected retained race socket")
		}
		if err = os.Remove(host.Socket); err != nil {
			t.Fatal(err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	r, err := newServerMode(host.Directory, projectRaceVMBackend{project: control.Project}, time.Hour, true)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	r.keys = access.DefaultKeys
	expansion, err := loadProjectExpansion(r.s.db, control.Project)
	if err != nil || expansion.Project.Plan.Label != "issues native transition" || len(expansion.Members) != 2 || len(expansion.History) != 0 {
		t.Fatal("race project changed", err)
	}
	for _, m := range expansion.Members {
		if (m.Member.InstanceID != "issues-race-nec" || m.Member.Ship != "~nec") && (m.Member.InstanceID != "issues-race-bud" || m.Member.Ship != "~bud") {
			t.Fatal("foreign race member")
		}
		if m.Phase != "ready" || m.StopIntent != nil || m.Creator.Generation != 1 || m.Creator.Mode != "boot" {
			t.Fatal("race selection is not fresh")
		}
	}
	listener, err := localrpc.Listen(host.Socket)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if err = os.Chmod(host.Socket, os.ModeSticky|0600); err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: r, ConnContext: PeerContext, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second}
	defer server.Close()
	if err = issuesVMWrite("native-race-ready.json", map[string]any{"pid": os.Getpid(), "project": control.Project, "source_commit": issuesVMSourceRevision, "build_tag": "omabit_disposable_qualification"}); err != nil {
		t.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(t.Context(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	select {
	case err := <-done:
		t.Fatal("native race helper ended unexpectedly", err)
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

func TestIssuesProjectRaceBackendRefusesUnselectedAction(t *testing.T) {
	b := projectRaceVMBackend{project: "project"}
	var s runtimequalify.ProjectExpansion
	s.Project.ProjectID = "project"
	saved := false
	save := func(runtimequalify.ProjectExpansion) error { saved = true; return nil }
	for _, test := range []struct {
		s          *runtimequalify.ProjectExpansion
		id, action string
	}{{nil, "issues-race-bud", "stop"}, {&s, "other", "stop"}, {&s, "issues-race-bud", "start"}} {
		err := b.ProjectShipAction(t.Context(), test.s, test.id, test.action, "op-stop", save)
		if err == nil || err.Error() != "native race helper refuses an unselected action" || saved {
			t.Fatal("unselected action escaped", err)
		}
	}
	s.Project.ProjectID = "other"
	if err := b.ProjectShipAction(t.Context(), &s, "issues-race-bud", "stop", "op-stop", save); err == nil || saved {
		t.Fatal("foreign project escaped", err)
	}
	s.Project.ProjectID = "project"
	// The selected shape still cannot open a host store or dispatch: the complete
	// VM/source guard rejects the ordinary source-test environment first.
	if err := b.ProjectShipAction(t.Context(), &s, "issues-race-bud", "stop", "op-stop", save); err == nil || saved {
		t.Fatal("native race backend entered host")
	}
}

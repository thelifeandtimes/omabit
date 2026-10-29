//go:build linux && amd64

package managed

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/localrpc"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// Only called after the enclosing no-NIC QEMU admission and completed recovery.
func vmProjectStatusCLI(ctx context.Context, r *Server, p storedProject, recovery string) (map[string]any, error) {
	result := map[string]any{"ok": false}
	const socket = "/home/lab/project-status.sock"
	listener, err := localrpc.Listen(socket)
	if err != nil {
		return result, err
	}
	server := &http.Server{Handler: r, ConnContext: PeerContext, ReadHeaderTimeout: 5 * time.Second}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	defer func() { server.Close(); <-done }()
	var before, after int
	if err = r.s.db.QueryRow("SELECT total_changes()").Scan(&before); err != nil {
		return result, err
	}
	for _, format := range []string{"json", "text"} {
		commandCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		command := exec.CommandContext(commandCtx, "/opt/omabit-lab/urbit-collector", "--socket", socket, "--format", format, "project-status", p.ID)
		command.Env = []string{"HOME=/home/lab", "XDG_RUNTIME_DIR=/run/user/1000", "PATH=/usr/bin:/bin", "LC_ALL=C"}
		data, err := command.Output()
		cancel()
		if err != nil {
			return result, err
		}
		if len(data) > 1024*1024 {
			return result, errors.New("oversized saved-status output")
		}
		if format == "text" {
			if !strings.Contains(string(data), "Saved history only.") || !strings.Contains(string(data), "do not prove current ship health") {
				return result, errors.New("status text loses historical boundary")
			}
			result["text"] = string(data)
			continue
		}
		var status protocol.ProjectStatus
		if json.Unmarshal(data, &status) != nil || status.RunnerID != r.s.runnerID || status.Project.ID != p.ID || status.Project.Revision != p.Revision || status.RuntimeObservation != "not-performed" || status.HasOlderOperations {
			return result, errors.New("status differs from selected runner/project")
		}
		continuations, observations := 0, 0
		parent := false
		for _, op := range status.Operations {
			if op.ID == recovery && op.Kind == "recover-project-anchor" && op.Status == "succeeded" {
				parent = true
			}
			if op.Kind == "continue-project-recovery" {
				continuations++
			}
			if op.Kind == "observe-project-recovery" {
				observations++
			}
		}
		if !parent || continuations != 3 || observations != 2 {
			return result, errors.New("status lost recovery attempts")
		}
		result["json"] = status
	}
	if err = r.s.db.QueryRow("SELECT total_changes()").Scan(&after); err != nil {
		return result, err
	}
	if before != after {
		return result, errors.New("status read changed database")
	}
	result["ok"], result["store_unchanged"] = true, true
	return result, nil
}

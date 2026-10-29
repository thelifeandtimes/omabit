//go:build linux && amd64

package managed

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

// This entry point has its own explicit token and all common no-NIC guest guards.
func TestProjectApplicationStateDisposableVM(t *testing.T) {
	exerciseProjectDisposableVM(t, true, true, true, true, false, false, false, false)
}

const vmAppSpecPath = "/home/lab/project-state-spec.json"
const vmAppStatePath = "/home/lab/project-app-state.json"

func vmApplicationSpec(t *testing.T, s *store, p storedProject, members []map[string]any, docker func(...string) json.RawMessage) (map[string]any, error) {
	t.Helper()
	anchor, err := loadProjectAnchorStart(s.db, p.ID)
	if err != nil {
		return nil, err
	}
	var inspected []struct {
		ID     string `json:"Id"`
		Config struct{ Labels map[string]string }
	}
	if json.Unmarshal(docker("container", "inspect", anchor.Prepared.Anchor.ID), &inspected) != nil || len(inspected) != 1 || inspected[0].ID != anchor.Prepared.Anchor.ID {
		return nil, errors.New("application fixture anchor binding")
	}
	ships := map[string]any{}
	for _, item := range members {
		ready, ok := item["readiness"].(map[string]any)
		if !ok || ready["ok"] != true {
			return nil, errors.New("application fixture readiness missing")
		}
		job, ok := ready["journal"].(projectMemberReadinessJob)
		if !ok || job.Evidence == nil || anchor.Namespace == nil || job.Binding.Namespace != *anchor.Namespace {
			return nil, errors.New("application fixture readiness binding")
		}
		member, found := projectMember{}, false
		for _, m := range p.Members {
			if m.InstanceID == job.InstanceID {
				member, found = m, true
			}
		}
		if !found || (member.Ship != "~zod" && member.Ship != "~nec") {
			return nil, errors.New("application fixture identity")
		}
		path, err := pierstore.Check("/home/lab", job.Evidence.Storage)
		if err != nil {
			return nil, err
		}
		ships[strings.TrimPrefix(member.Ship, "~")] = map[string]any{"http_port": member.InternalHTTPPort, "published_port": member.HTTPPort, "pier": path, "identity": job.Evidence.Storage.Identity}
	}
	if len(ships) != 2 {
		return nil, errors.New("application fixture requires both fresh peers")
	}
	return map[string]any{"format": 1, "http_transport": "published-loopback", "runner_id": s.runnerID, "project_id": p.ID, "anchor_id": anchor.Prepared.Anchor.ID, "namespace": fmt.Sprintf("net:[%d]", anchor.Namespace.Inode), "anchor_labels": inspected[0].Config.Labels, "ships": ships}, nil
}

func vmApplicationSetup(t *testing.T, ctx context.Context, b *vmMemberLaunchBackend, p storedProject, members []map[string]any, docker func(...string) json.RawMessage) {
	t.Helper()
	b.appResult = map[string]any{"ok": false, "phases": []map[string]any{}}
	for _, m := range members {
		if m["instance_id"] == "vm-zod" {
			m["application_state"] = b.appResult
		}
	}
	spec, err := vmApplicationSpec(t, b.store, p, members, docker)
	if err != nil {
		t.Error("application specification", err)
		return
	}
	if b.livePeer {
		spec["peer_mode"] = "running"
	}
	if b.currentLifecycle {
		spec["peer_cycle"] = true
	}
	if b.anchorRecovery {
		spec["anchor_cycle"] = true
	}
	b.appResult["spec"] = spec
	raw, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		t.Error(err)
		return
	}
	f, err := os.OpenFile(vmAppSpecPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Error("application spec must be fresh", err)
		return
	}
	_, err = f.Write(append(raw, '\n'))
	err = errors.Join(err, f.Sync(), f.Close())
	if err != nil {
		t.Error("application spec write", err)
		return
	}
	vmApplicationPhase(t, ctx, b, "setup-before-first")
}

func vmApplicationPhase(t *testing.T, ctx context.Context, b *vmMemberLaunchBackend, phase string) {
	t.Helper()
	if b.livePeer {
		if b.checkLivePeer == nil {
			t.Error("running peer observer missing")
			return
		}
		if err := b.checkLivePeer(); err != nil {
			t.Error("running peer changed", err)
			return
		}
	}
	result := map[string]any{"phase": phase, "ok": false}
	b.appResult["phases"] = append(b.appResult["phases"].([]map[string]any), result)
	child, cancel := context.WithTimeout(ctx, 600*time.Second)
	defer cancel()
	cmd := exec.CommandContext(child, "/usr/bin/python3", "-B", "/opt/omabit-lab/runner_app_probe.py", phase)
	cmd.Env = []string{"HOME=/home/lab", "XDG_RUNTIME_DIR=/run/user/1000", "PATH=/usr/bin:/bin", "LC_ALL=C", "PYTHONDONTWRITEBYTECODE=1"}
	output, err := cmd.CombinedOutput()
	var receipt struct {
		OK      bool   `json:"ok"`
		Phase   string `json:"phase"`
		All     bool   `json:"all_phases_complete"`
		Receipt struct {
			Path  string `json:"path"`
			Bytes int    `json:"bytes"`
			SHA   string `json:"sha256"`
		} `json:"receipt"`
	}
	validOutput := len(output) <= 8192 && json.Unmarshal(output, &receipt) == nil
	if validOutput {
		result["response"] = json.RawMessage(append([]byte(nil), output...))
	}
	f, readErr := os.Open(vmAppStatePath)
	var raw []byte
	if readErr == nil {
		raw, readErr = io.ReadAll(io.LimitReader(f, 2*1024*1024+1))
		readErr = errors.Join(readErr, f.Close())
	}
	if readErr == nil && len(raw) <= 2*1024*1024 && json.Valid(raw) {
		b.appResult["retained_state"] = json.RawMessage(append([]byte(nil), raw...))
	} else {
		t.Error("application receipt unavailable or oversized")
		return
	}
	if err != nil || !validOutput || !receipt.OK || receipt.Phase != phase || receipt.Receipt.Path != vmAppStatePath || receipt.Receipt.Bytes != len(raw) || receipt.Receipt.SHA != fmt.Sprintf("%x", sha256.Sum256(raw)) || receipt.All != (phase == "after-second" && !b.currentLifecycle || phase == "after-peer" && b.currentLifecycle && !b.anchorRecovery || phase == "after-anchor" && b.anchorRecovery) {
		t.Error("application phase failed; private structured receipt retained", phase)
		return
	}
	result["ok"] = true
	b.appResult["ok"] = receipt.All
}

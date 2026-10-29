//go:build linux && amd64 && omabit_private_member_boot_inspection

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
	"golang.org/x/sys/unix"
)

func init() {
	inspectionGuestMarker = "offline-managed-member-boot-inspect-v1"
	inspectionSchema = 46
	expectedOperations = 5
	memberBootAudit = inspectMemberBoot
}

type memberBootProof struct {
	Version         int    `json:"version"`
	ProjectID       string `json:"project_id"`
	InstanceID      string `json:"instance_id"`
	OperationID     string `json:"operation_id"`
	OperationSHA256 string `json:"operation_sha256"`
	LaunchSHA256    string `json:"launch_sha256"`
	ManagerReopened bool   `json:"manager_reopened"`
	ReplayPreserved bool   `json:"replay_preserved"`
}
type memberBootReceipt struct {
	Version            int                              `json:"version"`
	Phase              string                           `json:"phase"`
	LaunchSHA256       string                           `json:"launch_sha256"`
	ContainerID        string                           `json:"container_id"`
	Ship               string                           `json:"ship"`
	PublicHTTPIdentity bool                             `json:"public_http_identity"`
	LensIdentity       bool                             `json:"lens_identity"`
	GatewayBefore      *privatechain.GatewayObservation `json:"gateway_before,omitempty"`
	GatewayAfter       *privatechain.GatewayObservation `json:"gateway_after,omitempty"`
	StopDispatched     bool                             `json:"stop_dispatched"`
	StopAcknowledged   bool                             `json:"stop_acknowledged"`
	ExitCode           int                              `json:"exit_code"`
	FinishedAt         time.Time                        `json:"finished_at"`
	StoppedMetadata    bool                             `json:"stopped_metadata"`
}

func memberBootContinuation(first, reopened memberBootProof) bool {
	if first.Version != 1 || first.ManagerReopened || first.ReplayPreserved || !reopened.ManagerReopened || !reopened.ReplayPreserved {
		return false
	}
	reopened.ManagerReopened, reopened.ReplayPreserved = false, false
	return first == reopened
}
func memberBootSequence(rows []memberBootReceipt) bool {
	if len(rows) != 4 {
		return false
	}
	base := rows[0]
	if base.Version != 1 || base.Phase != "ready" || !base.PublicHTTPIdentity || !base.LensIdentity || base.GatewayBefore == nil || base.GatewayAfter != nil || base.StopDispatched || base.StopAcknowledged || base.ExitCode != 0 || !base.FinishedAt.IsZero() || base.StoppedMetadata {
		return false
	}
	for i, phase := range []string{"ready", "stop-dispatched", "stop-acknowledged", "stopped"} {
		row := rows[i]
		if row.Phase != phase || row.StopDispatched != (i >= 1) || row.StopAcknowledged != (i >= 2) || row.ExitCode != 0 || row.StoppedMetadata != (i == 3) || (row.GatewayAfter != nil) != (i == 3) || row.FinishedAt.IsZero() != (i != 3) {
			return false
		}
		row.Phase = "ready"
		row.StopDispatched = false
		row.StopAcknowledged = false
		row.StoppedMetadata = false
		row.GatewayAfter = nil
		row.FinishedAt = time.Time{}
		if !reflect.DeepEqual(row, base) {
			return false
		}
	}
	return true
}
func memberBootDirectory(path string, expected pierstore.DirectoryIdentity) {
	var st unix.Stat_t
	must(unix.Lstat(path, &st))
	need(st.Mode&unix.S_IFMT == unix.S_IFDIR && st.Mode&07022 == 0 && st.Uid == 1000, "member-directory-metadata")
	var x unix.Statx_t
	must(unix.Statx(unix.AT_FDCWD, path, unix.AT_SYMLINK_NOFOLLOW, unix.STATX_BTIME, &x))
	need(x.Mask&unix.STATX_BTIME != 0, "member-directory-birth")
	id := pierstore.DirectoryIdentity{Device: uint64(st.Dev), Inode: st.Ino, Birth: fmt.Sprintf("%d:%d", x.Btime.Sec, x.Btime.Nsec)}
	need(id.SameObject(expected), "member-retained-object")
}
func inspectMemberBoot(db *sql.DB, p privatechain.Plan, w privatechain.Workspace, g privatechain.Gateway) map[string]any {
	out := map[string]any{"ok": false, "schema": 1, "ship_boots": 0, "raw_state_exported": false, "managed_stop_qualified": false, "application_state_qualified": false}
	logs := map[string][]byte{}
	for _, stage := range []string{"launch", "reopen"} {
		path := "/newroot/opt/omabit-lab/results/managed-member-" + stage + ".log"
		if _, err := os.Lstat(path); os.IsNotExist(err) {
			continue
		}
		logs[stage] = file(path, 8<<20, 0, 0)
		out[stage+"_log_sha256"] = digest(logs[stage])
		out[stage+"_passed"] = bytes.Contains(logs[stage], []byte("--- PASS: TestPrivateMemberBootVM"))
		// Export only fixed fixture stage identifiers, never arbitrary Go/log text.
		stages := []string{}
		for _, code := range []string{"member-host-lock", "member-manager", "member-project", "member-workspace", "member-gateway", "member-pill-stage", "member-launch-queue", "member-launch-record", "member-launch-validation", "member-replay", "member-http-lens-shutdown", "member-retained-gateway"} {
			if bytes.Contains(logs[stage], []byte("fixture failed at "+code)) {
				stages = append(stages, code)
			}
		}
		out[stage+"_failure_stages"] = stages
	}
	var count int
	must(db.QueryRow("SELECT count(*) FROM private_member_launches").Scan(&count))
	need(count == 1, "one-member-launch")
	var raw []byte
	must(db.QueryRow("SELECT body FROM private_member_launches WHERE instance_id='managed-vm-zod'").Scan(&raw))
	var launch privatechain.MemberLaunch
	decode(raw, &launch)
	must(launch.Validate(p, w))
	out["launch_phase"] = launch.Phase
	out["capture_present"] = launch.Capture != nil
	out["launch_version"] = launch.Version
	var opRaw, mirror []byte
	must(db.QueryRow("SELECT body,runtime FROM operations WHERE id=?", launch.OperationID).Scan(&opRaw, &mirror))
	var op protocol.Operation
	decode(opRaw, &op)
	out["operation_status"] = op.Status
	need(bytes.Equal(raw, mirror), "member-operation-mirror")
	if _, err := os.Lstat(root + "/qualification-private-member-boot-stopped.json"); os.IsNotExist(err) {
		return out
	}
	need(out["launch_passed"] == true && out["reopen_passed"] == true, "member-guest-stages-passed")
	need(launch.Version == 3 && launch.Capture != nil && launch.Phase == "running" && launch.Running != nil && launch.Running.Birth != nil && len(launch.Peers) == 0, "captured-first-member")
	need(op.Status == "succeeded" && op.Actor == "qualification:private-initializer" && op.Kind == "launch-private-project-member" && op.InstanceID == launch.InstanceID && op.ProjectID == g.Selection.Binding.ProjectID, "member-operation-attribution")
	var first, reopened memberBootProof
	decode(file(root+"/qualification-private-member-launch.json", 16<<10, 1000, 0600), &first)
	decode(file(root+"/qualification-private-member-reopen.json", 16<<10, 1000, 0600), &reopened)
	need(memberBootContinuation(first, reopened) && first.ProjectID == op.ProjectID && first.InstanceID == op.InstanceID && first.OperationID == op.ID && first.OperationSHA256 == digest(opRaw) && first.LaunchSHA256 == digest(raw), "member-manager-reopen-preserved")
	rows := []memberBootReceipt{}
	for _, phase := range []string{"ready", "stop-dispatched", "stop-acknowledged", "stopped"} {
		var v memberBootReceipt
		decode(file(root+"/qualification-private-member-boot-"+phase+".json", 256<<10, 1000, 0600), &v)
		rows = append(rows, v)
	}
	need(memberBootSequence(rows), "member-shutdown-sequence")
	final := rows[3]
	need(final.LaunchSHA256 == digest(raw) && final.ContainerID == launch.ContainerID && final.Ship == "~zod" && !final.FinishedAt.Before(launch.Running.StartedAt), "member-shutdown-binding")
	for _, o := range []*privatechain.GatewayObservation{final.GatewayBefore, final.GatewayAfter} {
		got, want := o.Clone(), g.Observation.Clone()
		got.ObservedAt = time.Time{}
		want.ObservedAt = time.Time{}
		need(reflect.DeepEqual(got, want), "member-gateway-unchanged")
	}
	need(!final.GatewayAfter.ObservedAt.Before(final.FinishedAt) && !final.GatewayBefore.ObservedAt.Before(launch.Running.ObservedAt), "member-observation-order")
	for _, m := range w.Mounts {
		path := root + "/" + m.Path
		entries, err := os.ReadDir(path)
		must(err)
		if m.InstanceID != launch.InstanceID {
			need(len(entries) == 0, "unlaunched-peer-empty")
			continue
		}
		need(len(entries) == 1 && entries[0].Name() == "zod" && entries[0].IsDir(), "single-captured-pier")
		id := launch.Running.Birth.Pier
		memberBootDirectory(path+"/zod", pierstore.DirectoryIdentity{Device: id.Device, Inode: id.Inode, Birth: id.Birth})
		memberBootDirectory(path+"/zod/.urb", pierstore.DirectoryIdentity{Device: id.UrbDevice, Inode: id.UrbInode, Birth: id.UrbBirth})
		for _, suffix := range []string{"/.vere.lock", "/.urb/vere.lock", "/.urb/lock"} {
			_, err = os.Lstat(path + "/zod" + suffix)
			need(os.IsNotExist(err), "member-stopped-locks-absent")
		}
	}
	directory(root+"/"+launch.RuntimeID, launch.Artifact)
	pill := file(root+"/"+launch.RuntimeID+"/"+candidate.PillName, candidate.PillBytes, 1000, 0400)
	need(int64(len(pill)) == candidate.PillBytes && digest(pill) == candidate.PillSHA256, "member-pinned-pill")
	key := file(root+"/"+w.Path+"/secrets/zod.key", protocol.MaxBootKeyBytes+1, 1000, 0600)
	defer clear(key)
	keyText := strings.TrimSuffix(string(key), "\n")
	need(protocol.ValidBootKey(keyText) && digest(key) == launch.Key.KeySHA256, "member-retained-key")
	for _, log := range logs {
		need(!bytes.Contains(log, []byte(keyText)), "member-no-key-in-log")
	}
	var disk struct {
		ID    string
		State struct {
			Running, OOMKilled, Dead bool
			ExitCode                 int
			StartedAt, FinishedAt    time.Time
		}
	}
	must(json.Unmarshal(file("/newroot/home/lab/.local/share/urbit-runner/docker/containers/"+launch.ContainerID+"/config.v2.json", 1<<20, 1000, 0), &disk))
	need(disk.ID == launch.ContainerID && !disk.State.Running && !disk.State.OOMKilled && !disk.State.Dead && disk.State.ExitCode == 0 && disk.State.StartedAt.Equal(launch.Running.StartedAt) && disk.State.FinishedAt.Equal(final.FinishedAt), "member-docker-clean-exit")
	out["ok"] = true
	out["ship_boots"] = 1
	out["manager_reopen_preserved"] = true
	out["captured_pier_matches"] = true
	out["public_http_identity"] = true
	out["lens_identity"] = true
	out["fixture_lens_shutdown_acknowledged"] = true
	out["exit_code"] = 0
	out["key_contents_read"] = true
	out["key_contents_exported"] = false
	return out
}

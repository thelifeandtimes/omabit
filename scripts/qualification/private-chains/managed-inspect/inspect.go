package main

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"golang.org/x/sys/unix"
	_ "modernc.org/sqlite"
)

const root = "/newroot/home/lab/.local/share/urbit-runner/managed"

var check = "guest-admission"
var memberKeyAudit func(privatechain.Plan, privatechain.Workspace, privatechain.Gateway)
var memberBootAudit func(*sql.DB, privatechain.Plan, privatechain.Workspace, privatechain.Gateway) map[string]any
var inspectionSchema = 43
var expectedOperations = 4

func need(ok bool, code string) {
	check = code
	if !ok {
		panic("managed audit refused")
	}
}
func digest(raw []byte) string {
	s := sha256.Sum256(raw)
	return hex.EncodeToString(s[:])
}
func hash(value any) string {
	raw, err := json.Marshal(value)
	must(err)
	return digest(raw)
}
func emit(value any) {
	raw, err := json.Marshal(value)
	must(err)
	fmt.Println("OMABIT_MANAGED_INSPECTION " + string(raw))
}
func readerFailureProjection(value any) map[string]any {
	return map[string]any{"check": check, "error_sha256": hash(fmt.Sprint(value)), "raw_message_exported": false}
}
func file(path string, limit int64, owner uint32, mode uint32) []byte {
	check = "bounded-private-file"
	need(strings.HasPrefix(path, "/newroot/") && filepath.Clean(path) == path, "file-path")
	real, err := filepath.EvalSymlinks(path)
	must(err)
	need(real == path, "file-alias")
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	must(err)
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	var st unix.Stat_t
	must(unix.Fstat(fd, &st))
	need(st.Mode&unix.S_IFMT == unix.S_IFREG && st.Uid == owner && st.Nlink == 1 && st.Size <= limit && (mode == 0 || st.Mode&0777 == mode), "file-metadata")
	raw, err := io.ReadAll(io.LimitReader(f, limit+1))
	must(err)
	need(int64(len(raw)) == st.Size && int64(len(raw)) <= limit, "file-size")
	return raw
}
func decode(raw []byte, value any) {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	must(d.Decode(value))
	need(d.Decode(new(any)) == io.EOF, "trailing-json")
}

type proof struct {
	Version      int                             `json:"version"`
	ProjectID    string                          `json:"project_id"`
	ProjectSHA   string                          `json:"project_sha256"`
	WorkspaceSHA string                          `json:"workspace_sha256"`
	GatewaySHA   string                          `json:"gateway_sha256"`
	OperationSHA string                          `json:"operation_sha256"`
	Observation  privatechain.GatewayObservation `json:"initial_observation"`
	RecoveryID   string                          `json:"recovery_id,omitempty"`
	RecoverySHA  string                          `json:"recovery_sha256,omitempty"`
	ReadySHA     string                          `json:"ready_sha256,omitempty"`
}
type observation struct {
	Version           int                   `json:"version"`
	ProjectID         string                `json:"project_id"`
	OperationID       string                `json:"operation_id"`
	SourceOperationID string                `json:"source_operation_id"`
	SourceDigest      string                `json:"source_digest"`
	Phase             string                `json:"phase"`
	Result            *privatechain.Gateway `json:"result,omitempty"`
}

func logProjection(path string) map[string]any {
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		return map[string]any{"present": false}
	}
	raw := file(path, 8<<20, 0, 0)
	text := string(raw)
	stages := regexp.MustCompile(`private managed gateway fixture failed at ([a-z-]{1,64})`).FindAllStringSubmatch(text, 8)
	codes := []string{}
	for _, stage := range stages {
		codes = append(codes, stage[1])
	}
	known := []string{}
	for _, message := range []string{"real backend never reported successful readiness", "lost readiness did not retain exact started resources", "private operation deadline expired", "read-only recovery changed initialized runtime evidence"} {
		if strings.Contains(text, message) {
			known = append(known, message)
		}
	}
	return map[string]any{"present": true, "bytes": len(raw), "sha256": digest(raw), "failure_stages": codes, "known_failures": known, "passed": strings.Contains(text, "--- PASS: TestPrivateManagedGatewayVM"), "raw_log_exported": false, "observer_trace": observerTraceProjection(text)}
}

func runtimeDiagnostic(g privatechain.Gateway) map[string]any {
	need(protocol.ValidID(g.Selection.Binding.ProjectID) && strings.HasPrefix(g.Selection.Binding.ProjectID, "private-project-"), "diagnostic-project-binding")
	workspace := root + "/private-projects/" + g.Selection.Binding.ProjectID
	out := map[string]any{}
	projectText := func(raw []byte) map[string]any {
		text := string(raw)
		codes, stages := []string{}, []string{}
		for _, code := range []string{"EACCES", "ENOENT", "EEXIST", "ETIMEDOUT", "ENOMEM", "EROFS", "ENOSPC", "ERR_ASSERTION", "ECONNREFUSED"} {
			if regexp.MustCompile(`\b` + code + `\b`).MatchString(text) {
				codes = append(codes, code)
			}
		}
		for _, stage := range []string{"canonical Hoon hex regression passed", "fresh-keys-generated", "keys-configured-on-private-chain", "both-key-verdicts-checked", "galaxy-chain-reads-complete", "GATEWAY_READY", "private evaluator failed", "initial gateway network mode differs", "fresh key output format refused", "reopened startup state differs"} {
			if strings.Contains(text, stage) {
				stages = append(stages, stage)
			}
		}
		return map[string]any{"sha256": digest(raw), "bytes": len(raw), "codes": codes, "known_stages": stages, "source_locations": regexp.MustCompile(`/gateway/docker_gateway[.]cjs:[0-9]{1,5}:[0-9]{1,5}`).FindAllString(text, 16), "raw_text_exported": false}
	}
	for _, name := range []string{"failure.txt", "evaluator-failure.json", "gateway-ready.json"} {
		path := workspace + "/" + name
		if _, err := os.Lstat(path); os.IsNotExist(err) {
			out[name] = map[string]any{"present": false}
			continue
		}
		raw := file(path, 64<<10, 1000, 0600)
		entry := projectText(raw)
		entry["present"] = true
		if name == "evaluator-failure.json" {
			var v map[string]any
			must(json.Unmarshal(raw, &v))
			for _, key := range []string{"timeout_ms", "elapsed_ms", "status", "stdout_bytes", "stderr_bytes"} {
				if number, ok := v[key].(float64); ok && number == float64(int64(number)) && number >= -1 && number <= 1<<30 {
					entry[key] = number
				}
			}
		}
		out[name] = entry
	}
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(g.ContainerID) {
		return out
	}
	containerRoot := "/newroot/home/lab/.local/share/urbit-runner/docker/containers/" + g.ContainerID
	if _, err := os.Lstat(containerRoot + "/config.v2.json"); err == nil {
		raw := file(containerRoot+"/config.v2.json", 1<<20, 1000, 0)
		var v struct {
			ID    string `json:"ID"`
			State struct {
				Running, OOMKilled, Dead bool
				ExitCode                 int
				StartedAt, FinishedAt    string
			} `json:"State"`
			LogPath     string `json:"LogPath"`
			MountPoints map[string]struct {
				Type, Destination, Source, Propagation string
				RW                                     bool
			}
		}
		must(json.Unmarshal(raw, &v))
		need(v.ID == g.ContainerID, "diagnostic-container-binding")
		out["container"] = map[string]any{"running": v.State.Running, "oom_killed": v.State.OOMKilled, "dead": v.State.Dead, "exit": v.State.ExitCode, "metadata_sha256": digest(raw)}
		start, se := time.Parse(time.RFC3339Nano, v.State.StartedAt)
		end, ee := time.Parse(time.RFC3339Nano, v.State.FinishedAt)
		if se == nil && ee == nil && !start.IsZero() && end.After(start) {
			out["container_lifetime_ms"] = end.Sub(start).Milliseconds()
		}
		mounts := []map[string]any{}
		for _, target := range []string{"/lab", "/node", "/tools", "/source", "/compiled", "/deployment", "/vere", "/gateway.cjs", "/tmp", "/urbit"} {
			m, present := v.MountPoints[target]
			entry := map[string]any{"target": target, "present": present}
			if present {
				entry["bind"] = m.Type == "bind"
				entry["tmpfs"] = m.Type == "tmpfs"
				entry["writable"] = m.RW
				entry["destination_matches"] = m.Destination == target
				entry["source_empty"] = m.Source == ""
				entry["rprivate"] = m.Propagation == "rprivate"
			}
			mounts = append(mounts, entry)
		}
		out["stored_mounts"] = mounts
		out["stored_mount_count"] = len(v.MountPoints)
		// Docker's local driver stores length-delimited protobuf messages. Only
		// known fixed diagnostic tokens are projected; no bytes are exported.
		localLog := containerRoot + "/local-logs/container.log"
		if _, err := os.Lstat(localLog); err == nil {
			out["local_container_log"] = projectText(file(localLog, 4<<20, 1000, 0))
		}
		log := containerRoot + "/" + g.ContainerID + "-json.log"
		if v.LogPath == strings.TrimPrefix(log, "/newroot") {
			out["container_log"] = projectText(file(log, 4<<20, 1000, 0))
		}
	}
	return out
}

func directory(path string, expected pierstore.DirectoryIdentity) {
	real, err := filepath.EvalSymlinks(path)
	must(err)
	need(real == path, "directory-alias")
	var st unix.Stat_t
	must(unix.Lstat(path, &st))
	need(st.Mode&unix.S_IFMT == unix.S_IFDIR && st.Mode&0777 == 0700 && st.Uid == 1000, "directory-metadata")
	var x unix.Statx_t
	must(unix.Statx(unix.AT_FDCWD, path, unix.AT_SYMLINK_NOFOLLOW, unix.STATX_BTIME, &x))
	need(x.Mask&unix.STATX_BTIME != 0, "directory-birth")
	id := pierstore.DirectoryIdentity{Device: uint64(st.Dev), Inode: st.Ino, Birth: fmt.Sprintf("%d:%d", x.Btime.Sec, x.Btime.Nsec)}
	need(expected.Inode == 0 || id.SameObject(expected), "retained-directory-identity")
}

func inspectManaged() {
	logs := map[string]any{}
	for _, mode := range []string{"initialize", "recover", "verify"} {
		logs[mode] = logProjection("/newroot/opt/omabit-lab/results/managed-gateway-" + mode + ".log")
	}
	out := map[string]any{"schema": 1, "ok": false, "managed_private_gateway": true, "read_only_disk": true, "raw_state_exported": false, "logs": logs, "key_contents_read": false, "chain_contents_read": false}
	if _, err := os.Lstat(root + "/runner.sqlite"); os.IsNotExist(err) {
		out["database_present"] = false
		emit(out)
		return
	}
	file(root+"/runner.sqlite", 64<<20, 1000, 0600)
	for _, suffix := range []string{"-wal", "-journal"} {
		info, err := os.Lstat(root + "/runner.sqlite" + suffix)
		need(os.IsNotExist(err) || err == nil && info.Size() == 0 && info.Mode().IsRegular(), "database-fully-checkpointed")
	}
	check = "readonly-sqlite"
	db, err := sql.Open("sqlite", "file:"+root+"/runner.sqlite?mode=ro&immutable=1")
	must(err)
	defer db.Close()
	db.SetMaxOpenConns(1)
	var schema int
	must(db.QueryRow("PRAGMA user_version").Scan(&schema))
	need(schema == inspectionSchema, "selected-schema")
	var integrity string
	must(db.QueryRow("PRAGMA quick_check").Scan(&integrity))
	need(integrity == "ok", "sqlite-integrity")
	counts := map[string]int{}
	for _, name := range []string{"instances", "projects", "private_projects", "private_project_workspaces", "private_project_gateways", "private_gateway_observations", "operations"} {
		var count int
		must(db.QueryRow("SELECT count(*) FROM " + name).Scan(&count))
		counts[name] = count
	}
	out["rows"] = counts
	if counts["private_project_gateways"] == 1 {
		var raw []byte
		must(db.QueryRow("SELECT body FROM private_project_gateways").Scan(&raw))
		var g privatechain.Gateway
		decode(raw, &g)
		if g.Phase == "planned" || g.Phase == "network-dispatched" || g.Phase == "network-created" || g.Phase == "gateway-dispatched" || g.Phase == "gateway-created" || g.Phase == "start-dispatched" || g.Phase == "started" || g.Phase == "ready" {
			out["source_phase"] = g.Phase
		}
		out["runtime_diagnostic"] = runtimeDiagnostic(g)
	}
	if _, err := os.Lstat(root + "/qualification-private-gateway-verify.json"); os.IsNotExist(err) {
		emit(out)
		return
	}
	need(counts["instances"] == 0 && counts["projects"] == 0 && counts["private_projects"] == 1 && counts["private_project_workspaces"] == 1 && counts["private_project_gateways"] == 1 && counts["private_gateway_observations"] == 1 && counts["operations"] == expectedOperations, "exact-lifecycle-records")
	var initial, recovered, verified proof
	decode(file(root+"/qualification-private-gateway-initialize.json", 256<<10, 1000, 0600), &initial)
	decode(file(root+"/qualification-private-gateway-recover.json", 256<<10, 1000, 0600), &recovered)
	decode(file(root+"/qualification-private-gateway-verify.json", 256<<10, 1000, 0600), &verified)
	need(initial.Version == 1 && protocol.ValidID(initial.ProjectID) && initial.RecoveryID == "" && reflect.DeepEqual(recovered, verified), "independent-manager-reopens")
	copy := recovered
	copy.RecoveryID, copy.RecoverySHA, copy.ReadySHA = "", "", ""
	need(reflect.DeepEqual(copy, initial), "original-observation-preserved")
	query := func(statement string, arg string, want string) []byte {
		var raw []byte
		must(db.QueryRow(statement, arg).Scan(&raw))
		need(len(raw) <= 256<<10 && digest(raw) == want, "persisted-source-hash")
		return raw
	}
	praw := query("SELECT body FROM private_projects WHERE id=?", initial.ProjectID, initial.ProjectSHA)
	var project struct {
		Plan privatechain.Plan `json:"plan"`
	}
	must(json.Unmarshal(praw, &project))
	var w privatechain.Workspace
	decode(query("SELECT body FROM private_project_workspaces WHERE project_id=?", initial.ProjectID, initial.WorkspaceSHA), &w)
	var g privatechain.Gateway
	decode(query("SELECT body FROM private_project_gateways WHERE project_id=?", initial.ProjectID, initial.GatewaySHA), &g)
	var original protocol.Operation
	decode(query("SELECT body FROM operations WHERE id=?", g.Selection.OperationID, initial.OperationSHA), &original)
	query("SELECT runtime FROM operations WHERE id=?", g.Selection.OperationID, initial.GatewaySHA)
	need(g.Phase == "started" && original.Status == "interrupted" && original.Actor == "qualification:private-initializer", "original-interruption-preserved")
	must(w.Validate(project.Plan))
	must(g.Validate(project.Plan, w))
	var recovery protocol.Operation
	decode(query("SELECT body FROM operations WHERE id=?", recovered.RecoveryID, recovered.RecoverySHA), &recovery)
	need(recovery.Status == "succeeded" && recovery.Actor == "qualification:private-observer" && recovery.Kind == "observe-private-project-gateway" && recovery.ProjectID == initial.ProjectID, "attributed-recovery")
	var raw []byte
	must(db.QueryRow("SELECT body FROM private_gateway_observations WHERE operation_id=?", recovery.ID).Scan(&raw))
	var j observation
	decode(raw, &j)
	need(j.Phase == "observed" && j.Result != nil && j.ProjectID == initial.ProjectID && j.SourceOperationID == original.ID && j.OperationID == recovery.ID && hash(j.Result) == recovered.ReadySHA, "recovery-journal")
	ready := j.Result
	must(ready.Validate(project.Plan, w))
	need(ready.Phase == "ready" && ready.ContainerID == g.ContainerID && ready.NetworkID == g.NetworkID && ready.Selection == g.Selection && ready.Observation != nil, "same-runtime-selection")
	o, first := ready.Observation.Clone(), initial.Observation.Clone()
	o.ObservedAt, first.ObservedAt = time.Time{}, time.Time{}
	need(reflect.DeepEqual(o, first), "same-runtime-observation")
	directory(root+"/private-projects", w.Parent)
	workspace := root + "/" + w.Path
	directory(workspace, w.Root)
	directory(workspace+"/chain", o.Chain)
	directory(workspace+"/secrets", o.Secrets)
	directory(workspace+"/piers", w.Observation.PiersRoot)
	for i, mount := range w.Mounts {
		directory(root+"/"+mount.Path, w.Observation.Mounts[i])
		entries, err := os.ReadDir(root + "/" + mount.Path)
		must(err)
		if memberBootAudit == nil {
			need(len(entries) == 0, "no-ship-boots")
		}
	}
	client := root + "/private-gateway-client-" + original.ID
	directory(client, pierstore.DirectoryIdentity{})
	directory(client+"/docker-config", pierstore.DirectoryIdentity{})
	entries, err := os.ReadDir(client + "/docker-config")
	must(err)
	need(len(entries) == 0, "empty-docker-config")
	var marker struct {
		Version   int                           `json:"version"`
		Selection privatechain.GatewaySelection `json:"selection"`
	}
	decode(file(client+"/session.json", 16<<10, 1000, 0600), &marker)
	need(marker.Version == 1 && marker.Selection == g.Selection, "client-selection")
	for _, value := range logs {
		need(value.(map[string]any)["passed"] == true, "all-guest-stages-passed")
	}
	out["ok"] = true
	out["schema_version"] = schema
	out["original_receipt_preserved"] = true
	out["separate_actor_recovery"] = true
	out["manager_reopens"] = 2
	out["ship_boots"] = 0
	out["filesystem_identities_preserved"] = true
	out["gateway_snapshot_sha256"] = recovered.ReadySHA
	out["full_gateway_restart_qualified"] = false
	if memberBootAudit != nil {
		member := memberBootAudit(db, project.Plan, w, *ready)
		out["member_boot"] = member
		out["ok"] = member["ok"] == true
		out["ship_boots"] = member["ship_boots"]
		out["key_contents_read"] = member["key_contents_read"] == true
	}
	if memberKeyAudit != nil {
		memberKeyAudit(project.Plan, w, *ready)
	}
	emit(out)
}

type observerTraceEvent struct {
	Stage        string `json:"stage"`
	Event        string `json:"event"`
	Milliseconds uint64 `json:"milliseconds"`
	Succeeded    bool   `json:"succeeded"`
	Expired      bool   `json:"expired"`
}

func observerTraceProjection(text string) map[string]any {
	allowed := map[string]bool{
		"open-contract": true, "open-input-root": true, "open-inputs": true, "open-engine": true, "open-image": true, "open-running": true, "open-state": true,
		"observe-preparation": true, "observe-engine-before": true, "observe-engine-after": true, "observe-image-before": true, "observe-image-after": true,
		"observe-running-before": true, "observe-running-after": true, "observe-host-before": true, "observe-host-after": true,
		"observe-namespace-before": true, "observe-namespace-after": true, "observe-state-before": true, "observe-state-after": true,
		"observe-mounts-before": true, "observe-mounts-after": true, "observe-rpc": true, "observe-inputs-after": true, "observe-final-validation": true,
	}
	pattern := regexp.MustCompile(`(?m)^OMABIT_PRIVATE_OBSERVER ([a-z-]{1,48}) (begin|end) ([0-9]{1,7}) (true|false) (true|false)$`)
	rows := pattern.FindAllStringSubmatch(text, 1025)
	if len(rows) > 1024 {
		return map[string]any{"over_limit": true, "raw_text_exported": false}
	}
	records := []observerTraceEvent{}
	for _, row := range rows {
		ms, err := strconv.ParseUint(row[3], 10, 64)
		if err != nil || ms > 900000 || !allowed[row[1]] || row[2] == "begin" && (ms != 0 || row[4] != "false") {
			continue
		}
		records = append(records, observerTraceEvent{Stage: row[1], Event: row[2], Milliseconds: ms, Succeeded: row[4] == "true", Expired: row[5] == "true"})
	}
	count := len(records)
	if len(records) > 128 {
		records = records[len(records)-128:]
	}
	return map[string]any{"count": count, "retained_count": len(records), "records": records, "over_limit": false, "raw_text_exported": false, "diagnostic_only": true}
}

//go:build linux && amd64

package managed

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const recordingUsageVMKind = "issues20-recording-usage-v1"
const recordingUsageVMCase = "closed-native-capture-allocation-v1"
const recordingUsageVMPredecessor = "e3a728a4b1dcc090a10560481dd8414e801c4ef12632164fb05e0f4db5d022d9"

type recordingUsageVMScenario struct {
	Kind        string `json:"kind"`
	Case        string `json:"case"`
	Source      string `json:"source_commit"`
	Predecessor string `json:"predecessor_root_sha256"`
	Original    int    `json:"original_schema"`
	Target      int    `json:"target_schema"`
}

func recordingUsageVMGuard(raw []byte, env string, uid int, home, vendor, cmdline, source string) bool {
	var scenario recordingUsageVMScenario
	return json.Unmarshal(raw, &scenario) == nil && scenario.Kind == recordingUsageVMKind && scenario.Case == recordingUsageVMCase && scenario.Source == source && len(source) == 40 && scenario.Predecessor == recordingUsageVMPredecessor && scenario.Original == 65 && scenario.Target == StateSchemaVersion && uid == 1000 && home == "/home/review" && strings.TrimSpace(vendor) == "QEMU" && strings.Contains(" "+strings.TrimSpace(cmdline)+" ", " omabit.disposable-guest=release-upgrade-v1 ") && env == recordingUsageVMCase
}

func TestIssuesRecordingUsageVMGuardRejectsHostAndForeignCase(t *testing.T) {
	source := strings.Repeat("a", 40)
	scenario := recordingUsageVMScenario{Kind: recordingUsageVMKind, Case: recordingUsageVMCase, Source: source, Predecessor: recordingUsageVMPredecessor, Original: 65, Target: StateSchemaVersion}
	raw, _ := json.Marshal(scenario)
	if !recordingUsageVMGuard(raw, recordingUsageVMCase, 1000, "/home/review", "QEMU\n", "quiet omabit.disposable-guest=release-upgrade-v1", source) {
		t.Fatal("fixed synthetic guard refused")
	}
	for _, alter := range []func(*recordingUsageVMScenario){func(s *recordingUsageVMScenario) { s.Kind = "other" }, func(s *recordingUsageVMScenario) { s.Case = "other" }, func(s *recordingUsageVMScenario) { s.Source = strings.Repeat("b", 40) }, func(s *recordingUsageVMScenario) { s.Predecessor = strings.Repeat("b", 64) }, func(s *recordingUsageVMScenario) { s.Original = 64 }, func(s *recordingUsageVMScenario) { s.Target = 64 }} {
		changed := scenario
		alter(&changed)
		body, _ := json.Marshal(changed)
		if recordingUsageVMGuard(body, recordingUsageVMCase, 1000, "/home/review", "QEMU", "omabit.disposable-guest=release-upgrade-v1", source) {
			t.Fatal("foreign guard accepted")
		}
	}
	for _, input := range []struct {
		env                           string
		uid                           int
		home, vendor, cmdline, source string
	}{{"", 1000, "/home/review", "QEMU", "omabit.disposable-guest=release-upgrade-v1", source}, {recordingUsageVMCase, 0, "/home/review", "QEMU", "omabit.disposable-guest=release-upgrade-v1", source}, {recordingUsageVMCase, 1000, "/home/other", "QEMU", "omabit.disposable-guest=release-upgrade-v1", source}, {recordingUsageVMCase, 1000, "/home/review", "other", "omabit.disposable-guest=release-upgrade-v1", source}, {recordingUsageVMCase, 1000, "/home/review", "QEMU", "omabit.disposable-guest=release-upgrade-v1-extra", source}, {recordingUsageVMCase, 1000, "/home/review", "QEMU", "omabit.disposable-guest=release-upgrade-v1", ""}} {
		if recordingUsageVMGuard(raw, input.env, input.uid, input.home, input.vendor, input.cmdline, input.source) {
			t.Fatal("host/foreign guard accepted")
		}
	}
}

func TestIssuesRecordingUsageDisposableVM(t *testing.T) {
	env := os.Getenv("OMABIT_RECORDING_USAGE_VM")
	if env == "" {
		t.Skip("fixed offline read-only-backed recording fixture only")
	}
	raw, e1 := os.ReadFile("/opt/omabit-release-input/scenario.json")
	vendor, e2 := os.ReadFile("/sys/class/dmi/id/sys_vendor")
	cmdline, e3 := os.ReadFile("/proc/cmdline")
	if e1 != nil || e2 != nil || e3 != nil || !recordingUsageVMGuard(raw, env, os.Geteuid(), os.Getenv("HOME"), string(vendor), string(cmdline), issuesVMSourceRevision) {
		t.Fatal("refused outside exact recording usage VM")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	proofRoot := "/home/review/omabit-recording-usage-proof"
	if _, err := os.Lstat(proofRoot); !os.IsNotExist(err) {
		t.Fatal("fresh proof directory required")
	}
	g02Check(t, os.Mkdir(proofRoot, 0700))
	base := "/home/review/.local/share/urbit-runner"
	measure := func() (recording.StorageTotals, recording.StorageTotals, map[string]recording.StorageTotals, map[string]string) {
		t.Helper()
		var total, shared recording.StorageTotals
		instances := map[string]recording.StorageTotals{}
		hashes := map[string]string{}
		for _, kind := range []string{"recording-jobs", "recordings"} {
			root := filepath.Join(base, kind)
			g02Check(t, filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
				g02Check(t, err)
				info, err := os.Lstat(path)
				g02Check(t, err)
				if !info.IsDir() && !info.Mode().IsRegular() {
					t.Fatal("unexpected fixture object")
				}
				stat, ok := info.Sys().(*syscall.Stat_t)
				if !ok || stat.Blocks < 0 {
					t.Fatal("fixture stat unavailable")
				}
				value := recording.StorageTotals{AllocatedBytes: uint64(stat.Blocks) * 512}
				if info.IsDir() {
					value.Directories = 1
				} else {
					value.Files = 1
					value.LogicalBytes = uint64(info.Size())
					data, err := os.ReadFile(path)
					g02Check(t, err)
					sum := sha256.Sum256(data)
					hashes[strings.TrimPrefix(path, base+"/")] = hex.EncodeToString(sum[:])
				}
				total, err = total.Add(value)
				g02Check(t, err)
				relative, err := filepath.Rel(root, path)
				g02Check(t, err)
				if relative == "." || kind == "recording-jobs" && relative == "admission.lock" {
					shared, err = shared.Add(value)
					g02Check(t, err)
					return nil
				}
				child := strings.Split(relative, string(filepath.Separator))[0]
				var instance string
				if kind == "recording-jobs" {
					data, err := os.ReadFile(filepath.Join(root, child, "job.json"))
					g02Check(t, err)
					job, err := runtimequalify.ReadManagedRecordingJob(data)
					g02Check(t, err)
					instance = job.Operation.Target.InstanceID
				} else {
					data, err := os.ReadFile(filepath.Join(root, child, "metadata.json"))
					g02Check(t, err)
					metadata, err := recording.ReadMetadata(data)
					g02Check(t, err)
					instance = metadata.InstanceID
				}
				current := instances[instance]
				current, err = current.Add(value)
				g02Check(t, err)
				instances[instance] = current
				return nil
			}))
		}
		return total, shared, instances, hashes
	}
	expected, shared, instances, before := measure()
	registry, err := collection.Open(filepath.Join(proofRoot, "registry"))
	g02Check(t, err)
	entry, err := registry.Enroll(ctx, "fixture", "/run/user/1000/urbit-runner/runner.sock")
	g02Check(t, err)
	config := filepath.Join(proofRoot, "registry")
	binary := "/run/omabit-runtime-candidate/candidate/omabit"
	stderr, err := os.OpenFile(filepath.Join(proofRoot, "mcp-stderr.log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	g02Check(t, err)
	defer stderr.Close()
	command := exec.CommandContext(ctx, binary, "--config-dir", config, "mcp", "serve", "--allow-runner", "fixture:"+entry.RunnerID)
	command.Stderr = stderr
	session, err := mcp.NewClient(&mcp.Implementation{Name: "recording-usage-native", Version: "1"}, nil).Connect(ctx, &mcp.CommandTransport{Command: command}, &mcp.ClientSessionOptions{ProtocolVersion: "2025-11-25"})
	g02Check(t, err)
	defer session.Close()
	input := service.RecordingControlsInput{Target: service.Target{Name: "fixture", RunnerID: entry.RunnerID}, IncludePrivateData: true}
	encoded, err := json.Marshal(input)
	g02Check(t, err)
	var policies []recording.ControlState
	for _, adapter := range []string{"cli", "mcp"} {
		var body []byte
		if adapter == "cli" {
			body, err = exec.CommandContext(ctx, binary, "--config-dir", config, "call", "runtime.recording-controls.v1", "--input-json", string(encoded)).Output()
			g02Check(t, err)
		} else {
			result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit.runtime.recording-controls.v1", Arguments: json.RawMessage(encoded)})
			g02Check(t, err)
			if result.IsError || len(result.Content) != 1 {
				t.Fatal("native SDK refused")
			}
			text, ok := result.Content[0].(*mcp.TextContent)
			if !ok {
				t.Fatal("native JSON fallback missing")
			}
			body = []byte(text.Text)
		}
		g02Check(t, os.WriteFile(filepath.Join(proofRoot, adapter+"-status.json"), body, 0600))
		var response struct {
			Data  recording.ControlStatus `json:"data"`
			Error *service.Problem        `json:"error"`
		}
		g02Check(t, json.Unmarshal(body, &response))
		status := response.Data
		if response.Error != nil || status.CapacityStatus != "available" || status.Capacity == nil || status.Capacity.Validate() != nil || status.Capacity.Usage == nil || status.Capacity.Usage.Status != "measured" || status.Capacity.Usage.Totals == nil || status.Capacity.Usage.Shared == nil || *status.Capacity.Usage.Totals != expected || *status.Capacity.Usage.Shared != shared {
			t.Fatal("native allocation differs from independent filesystem census")
		}
		if len(status.Capacity.Usage.Instances) != len(instances) {
			t.Fatal("native instance attribution differs")
		}
		for _, item := range status.Capacity.Usage.Instances {
			value, ok := instances[item.InstanceID]
			if !ok || item.Totals != value {
				t.Fatal("native per-instance usage differs")
			}
		}
		policies = append(policies, status.State)
	}
	_, _, _, after := measure()
	if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(policies[0], policies[1]) {
		t.Fatal("saved recording/control evidence changed during reads")
	}
	proof := map[string]any{"passed": true, "independent_totals": expected, "independent_shared": shared, "independent_instances": instances, "original_recording_bytes_unchanged": true, "policy_head_unchanged": true, "actual_cli_and_stdio_measurements": true, "scope": "closed native captures after copied fixture reboot only; no running capture, pressure/retirement/export, installed upgrade, DEV or production qualification"}
	body, err := json.MarshalIndent(proof, "", "  ")
	g02Check(t, err)
	g02Check(t, os.WriteFile(filepath.Join(proofRoot, "proof.json"), body, 0600))
	t.Log("native closed-capture allocation qualification passed")
}

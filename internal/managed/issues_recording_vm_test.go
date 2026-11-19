//go:build linux && amd64

package managed

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

// A read-only full-prefix observer for the explicitly guarded fresh guest. The
// actual installed runner, worker and engine own every native lifecycle effect.
func TestIssuesRecordingSnapshotDisposableVM(t *testing.T) {
	if os.Getenv("OMABIT_RECORDING_PILOT_VM") == "" {
		t.Skip("explicit fresh offline recording VM only")
	}
	vendor, e1 := os.ReadFile("/sys/class/dmi/id/sys_vendor")
	cmdline, e2 := os.ReadFile("/proc/cmdline")
	raw, e3 := os.ReadFile("/opt/omabit-release-input/scenario.json")
	var scenario struct {
		Kind   string `json:"kind"`
		Source string `json:"source_commit"`
	}
	if e1 != nil || e2 != nil || e3 != nil || json.Unmarshal(raw, &scenario) != nil || scenario.Kind != "issues20-recording-pilot-v1" || len(issuesVMSourceRevision) != 40 || scenario.Source != issuesVMSourceRevision || strings.TrimSpace(string(vendor)) != "QEMU" || !strings.Contains(" "+string(cmdline)+" ", " omabit.disposable-guest=release-upgrade-v1 ") || os.Geteuid() != 1000 || os.Getenv("HOME") != "/home/review" || os.Getenv("OMABIT_RECORDING_PILOT_VM") != "snapshot" {
		t.Fatal("refused outside exact frozen recording guest")
	}
	base := "/home/review/.local/share/urbit-runner"
	entries, err := os.ReadDir(filepath.Join(base, "recording-jobs"))
	g02Check(t, err)
	if len(entries) < 1 || len(entries) > 16 {
		t.Fatal("pilot job count", len(entries))
	}
	type prefix struct {
		Unit         string                               `json:"unit"`
		Status       runtimequalify.RecordingWorkerStatus `json:"status"`
		Frames       int                                  `json:"frames"`
		StdoutBytes  int64                                `json:"stdout_bytes"`
		StderrBytes  int64                                `json:"stderr_bytes"`
		StdoutSHA256 string                               `json:"stdout_sha256"`
		StderrSHA256 string                               `json:"stderr_sha256"`
		LastSequence uint64                               `json:"last_sequence"`
		LastDigest   string                               `json:"last_digest"`
		Tail         string                               `json:"tail"`
		ExitFrames   int                                  `json:"exit_frames"`
		GapFrames    int                                  `json:"gap_frames"`
	}
	var out []prefix
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(base, "recording-jobs", entry.Name(), "job.json"))
		g02Check(t, err)
		job, err := runtimequalify.ReadManagedRecordingJob(raw)
		g02Check(t, err)
		digest := sha256.Sum256(raw)
		status, err := runtimequalify.ReadManagedRecordingStatus(t.Context(), job.ID, hex.EncodeToString(digest[:]))
		g02Check(t, err)
		unit, err := runtimequalify.RecordingWorkerUnitName(job)
		g02Check(t, err)
		p := prefix{Status: status, Unit: unit}
		if status.Metadata != nil {
			store, err := recording.OpenStore(filepath.Join(base, "recordings"), recording.Limits{MaxSessionBytes: job.Limits.MaxSessionBytes, MaxSegmentBytes: job.Limits.MaxSegmentBytes, ReserveFreeBytes: job.Limits.ReserveFreeBytes})
			g02Check(t, err)
			stdout, stderr := sha256.New(), sha256.New()
			var cursor *recording.Cursor
			for pages := 0; ; pages++ {
				if pages > 16384 {
					t.Fatal("bounded prefix scan exceeded")
				}
				page, err := store.ReadPage(t.Context(), status.SessionID, cursor, 256, 1<<20)
				g02Check(t, err)
				for _, f := range page.Frames {
					p.Frames++
					switch f.Kind {
					case recording.Stdout:
						stdout.Write(f.Data)
						p.StdoutBytes += int64(len(f.Data))
					case recording.Stderr:
						stderr.Write(f.Data)
						p.StderrBytes += int64(len(f.Data))
					case recording.Exit:
						p.ExitFrames++
					case recording.Gap:
						p.GapFrames++
					}
				}
				p.LastSequence, p.LastDigest, p.Tail = page.Next.Sequence, page.Next.Digest, page.Tail
				if page.Tail != "not-scanned" {
					break
				}
				next := page.Next
				cursor = &next
			}
			p.StdoutSHA256, p.StderrSHA256 = hex.EncodeToString(stdout.Sum(nil)), hex.EncodeToString(stderr.Sum(nil))
			g02Check(t, store.Close())
		}
		out = append(out, p)
	}
	raw, err = json.Marshal(out)
	g02Check(t, err)
	fmt.Println("OMABIT_RECORDING_VM_SNAPSHOT " + string(raw))
}

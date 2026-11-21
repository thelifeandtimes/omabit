//go:build linux || darwin

package checkplan

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func savedReport(t *testing.T) runtimequalify.Result {
	t.Helper()
	plan, err := Read(Lifecycle)
	if err != nil {
		t.Fatal(err)
	}
	p := plan.Runtime
	return runtimequalify.Result{RunID: strings.Repeat("a", 32), Profile: p.Profile, Image: p.Image, ImageID: p.ConfigDigest, RepoDigest: strings.TrimPrefix(p.Image, "docker.io/"), Platform: p.Platform, EngineVersion: p.EngineVersion, VereVersion: p.VereVersion, PillURL: p.Pill.URL, PillSHA256: p.Pill.SHA256, PillBytes: p.Pill.Bytes, CheckpointPassed: true, CleanupComplete: true, LimitsObserved: true, PierMetadataSeen: true, Phases: []runtimequalify.Phase{{Name: "private-name", Passed: true, Detail: "secret-cookie-value"}}, Containers: []runtimequalify.ContainerRecord{{ID: strings.Repeat("b", 64), Removed: true}}, Directory: "/private/pier", ReportPath: "/private/report.json", PillPath: "/private/pill"}
}

func reportBytes(t *testing.T, report runtimequalify.Result) []byte {
	t.Helper()
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func reportDigest(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }
func writeReport(t *testing.T, raw []byte) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "report.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSelectedReportIsImmutableBoundedAndNotExecutionAttestation(t *testing.T) {
	raw := reportBytes(t, savedReport(t))
	path := writeReport(t, raw)
	report, err := OpenReport("incident", path, reportDigest(raw))
	if err != nil {
		t.Fatal(err)
	}
	out := report.Summary()
	if !out.ReportedCheckpointPassed || !out.ReportedCleanupComplete || !out.ReportedPinsMatchCurrentPlan || out.ExecutedByThisService || out.RuntimeQualified || !strings.Contains(out.SourceProvenance, "unavailable") {
		t.Fatal("report truth/provenance boundary lost", out)
	}
	encoded, _ := json.Marshal(out)
	for _, private := range []string{"private-name", "secret-cookie-value", "/private/", strings.Repeat("b", 64)} {
		if bytes.Contains(encoded, []byte(private)) {
			t.Fatal("free-form/path/container data escaped summary")
		}
	}
	if err := os.WriteFile(path, []byte("replaced after startup"), 0600); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(out, report.Summary()) {
		t.Fatal("tool acquired fresh file bytes")
	}
	out.ReportedPhases[0].Passed = false
	if !report.Summary().ReportedPhases[0].Passed {
		t.Fatal("caller mutated immutable snapshot")
	}
	for _, selection := range []ReportSelection{{ReportBinding: ReportBinding{Name: "incident", PlanID: Lifecycle, SHA256: report.SHA256()}, IncludePrivateData: true}} {
		if selection.Validate() != nil {
			t.Fatal("valid selection rejected")
		}
	}
}

func TestReportRefusesUnselectedDigestUnsafeFileAndAmbiguousFormat(t *testing.T) {
	raw := reportBytes(t, savedReport(t))
	good := writeReport(t, raw)
	for _, digest := range []string{strings.Repeat("0", 64), strings.ToUpper(reportDigest(raw)), "missing"} {
		if _, err := OpenReport("incident", good, digest); err == nil {
			t.Fatal("wrong digest accepted")
		}
	}
	for _, kind := range []string{"symlink", "hardlink", "fifo", "public", "oversize", "directory"} {
		t.Run(kind, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "candidate")
			var err error
			switch kind {
			case "symlink":
				err = os.Symlink(good, path)
			case "hardlink":
				err = os.Link(good, path)
			case "fifo":
				err = syscall.Mkfifo(path, 0600)
			case "directory":
				err = os.Mkdir(path, 0700)
			case "public":
				err = os.WriteFile(path, raw, 0644)
			case "oversize":
				err = os.WriteFile(path, bytes.Repeat([]byte("x"), MaxReportBytes+1), 0600)
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err = OpenReport("incident", path, reportDigest(raw)); err == nil {
				t.Fatal("unsafe report accepted", kind)
			}
		})
	}
	for _, kind := range []string{"unknown-field", "duplicate", "missing", "null", "runtime-claim", "contradictory-cleanup", "failed-phase", "unknown-profile", "depth", "trailing"} {
		t.Run(kind, func(t *testing.T) {
			report := savedReport(t)
			var value map[string]any
			_ = json.Unmarshal(reportBytes(t, report), &value)
			switch kind {
			case "unknown-field":
				value["command"] = "shell private"
			case "missing":
				delete(value, "checkpoint_passed")
			case "null":
				value["checkpoint_passed"] = nil
			case "runtime-claim":
				value["runtime_qualified"] = true
			case "contradictory-cleanup":
				value["cleanup_complete"] = false
			case "failed-phase":
				value["phases"] = []map[string]any{{"name": "failed", "passed": false, "detail": "private"}}
			case "unknown-profile":
				value["profile"] = "arbitrary-executor"
			}
			data, _ := json.Marshal(value)
			switch kind {
			case "duplicate":
				data = append([]byte(`{"checkpoint_passed":false,`), data[1:]...)
			case "depth":
				data = []byte(strings.Repeat("[", 34) + "0" + strings.Repeat("]", 34))
			case "trailing":
				data = append(data, []byte(` {}`)...)
			}
			path := writeReport(t, data)
			if _, err := OpenReport("incident", path, reportDigest(data)); err == nil {
				t.Fatal("ambiguous/unsupported report accepted", kind)
			}
		})
	}
}

func TestFailedAndDifferentPinReportsRemainExplicitReportedEvidence(t *testing.T) {
	r := savedReport(t)
	r.CheckpointPassed = false
	r.CleanupComplete = false
	r.Phases[0].Passed = false
	r.Failure = "private failure"
	r.PillSHA256 = strings.Repeat("b", 64)
	raw := reportBytes(t, r)
	report, err := OpenReport("failed", writeReport(t, raw), reportDigest(raw))
	if err != nil {
		t.Fatal(err)
	}
	out := report.Summary()
	if out.ReportedCheckpointPassed || out.ReportedCleanupComplete || out.ReportedPinsMatchCurrentPlan || out.ReportedPhases[0].Passed || out.RuntimeQualified || out.ExecutedByThisService {
		t.Fatal("failed report promoted", out)
	}
	for _, in := range []ReportSelection{{}, {ReportBinding: ReportBinding{Name: "failed", PlanID: Lifecycle, SHA256: report.SHA256()}}, {ReportBinding: ReportBinding{Name: "failed", PlanID: "run_shell", SHA256: report.SHA256()}, IncludePrivateData: true}} {
		if in.Validate() == nil {
			t.Fatal("invalid selection accepted")
		}
	}
}

func TestReportPhaseCategoriesDoNotReflectArbitraryNames(t *testing.T) {
	for _, tc := range []struct{ profile, name, want string }{
		{runtimequalify.ProfileLifecycle, "engine", "engine"},
		{runtimequalify.ProfileLifecycle, "pack", "pack"},
		{runtimequalify.ProfileHTTP, "host-http-restart", "host-http-restart"},
		{runtimequalify.ProfileHTTP, "pack", "other"},
		{runtimequalify.ProfileLifecycle, "cookie=private-value", "other"},
	} {
		r := savedReport(t)
		r.Profile = tc.profile
		r.Phases[0].Name = tc.name
		raw := reportBytes(t, r)
		report, err := OpenReport("saved", writeReport(t, raw), reportDigest(raw))
		if err != nil {
			t.Fatal(err)
		}
		if report.Summary().ReportedPhases[0].Category != tc.want {
			t.Fatal("unbounded or wrong phase category", tc)
		}
	}
}

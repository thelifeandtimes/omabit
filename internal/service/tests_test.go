package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/checkplan"
)

func TestNamedPlanDiscoveryAndReportSelectionStayOffline(t *testing.T) {
	root, rootErr := filepath.EvalSymlinks(t.TempDir())
	if rootErr != nil {
		t.Fatal(rootErr)
	}
	registry := filepath.Join(root, "uncreated-registry")
	raw := []byte(`{"profile":"isolated-lifecycle","run_id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","checkpoint_passed":false,"runtime_qualified":false,"cleanup_complete":false,"phases":[],"containers":[],"failure":"private-cookie-value","directory":"/private/path"}`)
	file := filepath.Join(root, "report.json")
	if err := os.WriteFile(file, raw, 0600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	digest := hex.EncodeToString(sum[:])
	report, err := checkplan.OpenReport("saved", file, digest)
	if err != nil {
		t.Fatal(err)
	}
	reports := []*checkplan.Report{report}
	core, err := New(Options{ConfigDir: registry, RequireAllowed: true, TestReports: reports})
	if err != nil {
		t.Fatal(err)
	}
	reports[0] = nil
	for _, tc := range []struct{ id, input, code string }{
		{"tests.plans.v1", `{}`, ""},
		{"tests.plan.v1", `{"id":"runtime-lifecycle-v1"}`, ""},
		{"tests.plan.v1", `{"id":"runtime-http-v1"}`, ""},
		{"tests.plan.v1", `{"id":"run_shell"}`, "invalid_input"},
		{"tests.plan.v1", `{"id":"runtime-lifecycle-v1","command":"shell"}`, "invalid_input"},
		{"tests.reports.v1", `{}`, "invalid_input"},
		{"tests.reports.v1", `{"include_private_data":true}`, ""},
		{"tests.report.v1", `{"name":"saved","plan_id":"runtime-lifecycle-v1","sha256":"` + digest + `","include_private_data":true}`, ""},
		{"tests.report.v1", `{"name":"saved","plan_id":"runtime-http-v1","sha256":"` + digest + `","include_private_data":true}`, "test_report_not_allowed"},
		{"tests.report.v1", `{"name":"saved","plan_id":"runtime-lifecycle-v1","sha256":"` + strings.Repeat("b", 64) + `","include_private_data":true}`, "test_report_not_allowed"},
		{"tests.report.v1", `{"name":"saved","plan_id":"runtime-lifecycle-v1","sha256":"` + digest + `","include_private_data":false}`, "invalid_input"},
		{"tests.report.v1", `{"name":"saved","plan_id":"runtime-lifecycle-v1","sha256":"` + digest + `","include_private_data":true,"path":"/unselected"}`, "invalid_input"},
		{"tests.run.v1", `{}`, "unknown_operation"},
	} {
		t.Run(tc.id+tc.code, func(t *testing.T) {
			out := core.Call(context.Background(), tc.id, json.RawMessage(tc.input))
			if tc.code != "" {
				if out.Error == nil || out.Error.Code != tc.code || out.Error.Dispatch != "not-dispatched" {
					t.Fatal("refusal differs", out)
				}
				return
			}
			if out.Error != nil {
				t.Fatal("read failed", out.Error)
			}
			encoded, _ := json.Marshal(out)
			if strings.Contains(string(encoded), "private-cookie-value") || strings.Contains(string(encoded), "/private/path") {
				t.Fatal("raw report exposed")
			}
		})
	}
	if _, err := os.Lstat(registry); !os.IsNotExist(err) {
		t.Fatal("offline operation created registry")
	}
	for _, d := range core.Catalog() {
		if strings.HasPrefix(d.ID, "tests.") {
			if slices.Contains(d.Effects, "runtime-mutation") || slices.Contains(d.Effects, "journal-write") || slices.Contains(d.Effects, "disposable-test-execution") || d.Role != "local-user" {
				t.Fatal("read acquired execution effects", d.ID)
			}
		}
	}
	for _, opts := range []Options{{TestReports: []*checkplan.Report{nil}}, {TestReports: []*checkplan.Report{report, report}}, {TestReports: []*checkplan.Report{&checkplan.Report{}}}, {TestReports: make([]*checkplan.Report, checkplan.MaxReports+1)}} {
		if _, err := New(opts); err == nil {
			t.Fatal("invalid report startup selection accepted")
		}
	}
}

func TestApprovedSourceRootDoesNotAuthorizeReportReads(t *testing.T) {
	root, err := OpenSourceRoot("source", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	core, err := New(Options{Sources: []*SourceRoot{root}, RequireAllowed: true})
	if err != nil {
		t.Fatal(err)
	}
	out := core.Call(context.Background(), "tests.report.v1", json.RawMessage(`{"name":"source","plan_id":"runtime-lifecycle-v1","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","include_private_data":true}`))
	if out.Error == nil || out.Error.Code != "test_report_not_allowed" {
		t.Fatal("source selection broadened authority", out)
	}
	out = core.Call(context.Background(), "tests.reports.v1", json.RawMessage(`{"include_private_data":true}`))
	if out.Error != nil {
		t.Fatal(out.Error)
	}
	if len(out.Data.([]checkplan.ReportBinding)) != 0 {
		t.Fatal("implicit report discovery")
	}
}

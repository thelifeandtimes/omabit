package client

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectusage"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestProjectAllocationClientValidatesExactPrivateResponseAndScopedTimeout(t *testing.T) {
	in := projectusage.Request{RunnerID: "runner", ProjectID: "project", ExpectedRevision: 9007199254740993, ReviewDigest: strings.Repeat("b", 64), IncludePrivateData: true}
	r := projectusage.Root{ID: "mount-0", Kind: "member-mount", InstanceID: "member", RuntimeID: strings.Repeat("d", 32), Path: "/private/mount", Expected: pierstore.DirectoryIdentity{Device: 1, Inode: 2, Birth: "1:1"}}
	totals := projectusage.Totals{Directories: 1, RegularFiles: 1, UniqueFileInodes: 1, LogicalBytes: 9007199254740993, AllocatedBytes: 4096}
	out := projectusage.Observation{Version: 1, Binding: projectusage.Binding{RunnerID: in.RunnerID, Mode: "development-managed", ProjectID: in.ProjectID, Revision: in.ExpectedRevision, Generation: 1, PlanDigest: in.ReviewDigest, LedgerSHA256: strings.Repeat("c", 64)}, Scope: projectusage.Scope, Status: "measured", DeclaredRoots: []projectusage.Root{r}, Roots: []projectusage.RootUsage{{Root: r, Observed: r.Expected, Totals: totals}}, Totals: &totals, ExcludedComponents: projectusage.Exclusions(), SharedAllowanceStatus: "not-configured", StartedAt: time.Now().UTC(), ReadAt: time.Now().UTC(), Basis: "inert client fixture"}
	growth := projectusage.Compare(projectusage.Observation{}, out, 0)
	out.Growth = &growth
	variant, calls := "ok", 0
	c := &Client{httpClient: &http.Client{Timeout: 5 * time.Second, Transport: exportRoundTrip(func(req *http.Request) (*http.Response, error) {
		calls++
		deadline, ok := req.Context().Deadline()
		if req.Method != "POST" || req.URL.Path != protocol.ProjectAllocationPath || !ok || time.Until(deadline) < 31*time.Second || time.Until(deadline) > 33*time.Second {
			t.Error("allocation scope changed")
		}
		var got projectusage.Request
		if json.NewDecoder(req.Body).Decode(&got) != nil || got != in {
			t.Error("selection changed")
		}
		next := out.Clone()
		switch variant {
		case "runner":
			next.Binding.RunnerID = "other"
		case "revision":
			next.Binding.Revision++
		case "plan":
			next.Binding.PlanDigest = strings.Repeat("a", 64)
		case "partial":
			next.Status, next.Reason = "unknown", "unavailable"
		case "unknown":
			next.Status, next.Reason, next.Totals, next.Roots = "unknown", "unavailable", nil, nil
			g := projectusage.Compare(projectusage.Observation{}, next, 0)
			next.Growth = &g
		case "growth":
			next.Growth = nil
		case "stale":
			next.ReadAt = time.Now().Add(-time.Hour)
			next.StartedAt = next.ReadAt
		case "future":
			next.ReadAt = time.Now().Add(time.Hour)
		case "disconnect":
			return nil, errors.New("injected disconnect")
		}
		body, _ := json.Marshal(next)
		return &http.Response{StatusCode: 200, Header: http.Header{protocol.Header: []string{"1"}}, Body: io.NopCloser(strings.NewReader(string(body)))}, nil
	})}}
	for _, name := range []string{"ok", "unknown", "runner", "revision", "plan", "partial", "growth", "stale", "future", "disconnect"} {
		variant = name
		before := calls
		got, err := c.ProjectAllocation(t.Context(), in)
		if calls != before+1 || c.httpClient.Timeout != 5*time.Second {
			t.Fatal("read retried or changed shared client")
		}
		if name == "ok" || name == "unknown" {
			if err != nil || got.Validate() != nil {
				t.Fatal(name, err)
			}
			if name == "ok" && got.Totals.LogicalBytes != 9007199254740993 {
				t.Fatal("lost integer precision")
			}
		} else if err == nil || got.Totals != nil || len(got.Roots) != 0 {
			t.Fatal("invalid response leaked metrics", name, err)
		}
	}
	in.IncludePrivateData = false
	before := calls
	if _, err := c.ProjectAllocation(t.Context(), in); err == nil || calls != before {
		t.Fatal("invalid consent contacted runner")
	}
}

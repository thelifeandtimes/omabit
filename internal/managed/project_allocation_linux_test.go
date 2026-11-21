//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectusage"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type allocationFixtureBackend struct {
	*stopRecoveryFixtureBackend
	reads  atomic.Int32
	change func(*projectusage.Observation)
	onRead func()
}

// Synthetic metadata only; the filesystem walker is independently tested with
// real disposable roots. These paths never authorize filesystem access.
func allocationFixtureObservation(s runtimequalify.ProjectExpansion, b projectusage.Binding) projectusage.Observation {
	out := projectusage.Observation{Version: 1, Binding: b, Scope: projectusage.Scope, Status: "measured", ExcludedComponents: projectusage.Exclusions(), SharedAllowanceStatus: "not-configured", StartedAt: time.Now().UTC(), ReadAt: time.Now().UTC(), Basis: "synthetic API selection fixture; no filesystem scan"}
	base := "/fixture/.local/share/urbit-runner/managed"
	for i, m := range s.Members {
		selected := m.Creator.Storage
		if m.Initialized.Identity.Inode != 0 {
			selected = m.Initialized
		}
		out.DeclaredRoots = append(out.DeclaredRoots, projectusage.Root{ID: fmt.Sprintf("mount-%d", i), Kind: "member-mount", InstanceID: m.Member.InstanceID, RuntimeID: m.Creator.RuntimeID, Path: "/fixture/" + pierstore.MountPath(pierstore.Relative(selected.Path)), Expected: selected.MountIdentity}, projectusage.Root{ID: fmt.Sprintf("runtime-%d", i), Kind: "runtime-cache", InstanceID: m.Member.InstanceID, RuntimeID: m.Creator.RuntimeID, Path: filepath.Join(base, m.Creator.RuntimeID)})
	}
	out.DeclaredRoots = append(out.DeclaredRoots, projectusage.Root{ID: "project-cache", Kind: "project-cache", Path: filepath.Join(base, "project-resources-"+s.Project.ProjectID)})
	var total projectusage.Totals
	for i, r := range out.DeclaredRoots {
		id := r.Expected
		if id.Inode == 0 {
			id = pierstore.DirectoryIdentity{Device: 7, Inode: uint64(100 + i), Birth: "1000000000:1"}
		}
		value := projectusage.Totals{RegularFiles: 1, Directories: 1, UniqueFileInodes: 1, LogicalBytes: 100, AllocatedBytes: 4096}
		out.Roots = append(out.Roots, projectusage.RootUsage{Root: r, Observed: id, Totals: value})
		total, _ = total.Add(value)
	}
	out.Totals = &total
	return out
}
func (b *allocationFixtureBackend) ExpandedProjectAllocation(_ context.Context, s runtimequalify.ProjectExpansion, binding projectusage.Binding) (projectusage.Observation, error) {
	b.reads.Add(1)
	out := allocationFixtureObservation(s, binding)
	if b.change != nil {
		b.change(&out)
	}
	if b.onRead != nil {
		b.onRead()
	}
	return out, nil
}
func allocationCall(t *testing.T, r *Server, in projectusage.Request, role string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(in)
	g02Check(t, err)
	w := httptest.NewRecorder()
	q := httptest.NewRequest("POST", protocol.ProjectAllocationPath, bytes.NewReader(body))
	q.Header.Set("Content-Type", "application/json")
	if role == "operator" {
		q.Header.Set(protocol.Header, "1")
		q = q.WithContext(context.WithValue(q.Context(), peerKey{}, r.uid))
		r.ServeHTTP(w, q)
	} else {
		r.serveProjectAllocation(w, q, role)
	}
	return w
}
func allocationAPIFixture(t *testing.T) (*Server, *allocationFixtureBackend, protocol.Operation, projectusage.Request) {
	t.Helper()
	r, old, op := interruptedExpandedStopFixture(t, true)
	b := &allocationFixtureBackend{stopRecoveryFixtureBackend: old}
	r.b = b
	state, err := loadProjectExpansion(r.s.db, op.ProjectID)
	g02Check(t, err)
	digest, err := state.Project.Plan.Digest()
	g02Check(t, err)
	return r, b, op, projectusage.Request{RunnerID: r.s.runnerID, ProjectID: op.ProjectID, ExpectedRevision: state.Project.Revision, ReviewDigest: digest, IncludePrivateData: true}
}

func TestProjectAllocationReadPreservesEveryRowAndInterruptedReceipt(t *testing.T) {
	r, b, original, in := allocationAPIFixture(t)
	r.failed.Store(true) // Read-only diagnosis remains available with writes fenced.
	before := appAllRows(t, r.s.db)
	w := allocationCall(t, r, in, "operator")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var out projectusage.Observation
	g02Check(t, json.Unmarshal(w.Body.Bytes(), &out))
	if out.Validate() != nil || out.Growth == nil || out.Growth.Reason != "need-two-samples" || out.Binding.LedgerSHA256 == "" || out.FullProjectBudgetKnown || b.reads.Load() != 1 || b.observations.Load() != 0 {
		t.Fatal("invalid read binding", out)
	}
	if !reflect.DeepEqual(before, appAllRows(t, r.s.db)) {
		t.Fatal("allocation wrote persistent state")
	}
	after, err := loadOperation(r.s.db, original.ID)
	g02Check(t, err)
	if !reflect.DeepEqual(after, original) {
		t.Fatal("original interrupted receipt changed")
	}
	count, err := projectPendingOperationCount(r.s.db, in.ProjectID, "")
	g02Check(t, err)
	if count != 1 {
		t.Fatal("read resolved pending fence", count)
	}
	b.change = func(out *projectusage.Observation) {
		out.Status, out.Reason, out.UnknownRootID, out.Roots, out.Totals = "unknown", "changed", out.DeclaredRoots[0].ID, nil, nil
	}
	w = allocationCall(t, r, in, "operator")
	out = projectusage.Observation{}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &out) != nil || out.Status != "unknown" || out.Totals != nil || out.Growth == nil || out.Growth.Reason != "current-unavailable" || len(r.projectAllocations) != 0 {
		t.Fatal("unknown scan retained old metrics or growth", w.Code, w.Body.String())
	}
	if !reflect.DeepEqual(before, appAllRows(t, r.s.db)) {
		t.Fatal("unknown scan wrote persistent state")
	}
}

func TestProjectAllocationCapabilityAndUnsupportedReservationStayExplicit(t *testing.T) {
	r, _, id, _ := projectQueueFixture(t)
	if len(r.projectAllocationFeatures()) != 0 {
		t.Fatal("backend without adapter advertised allocation")
	}
	b := &allocationFixtureBackend{stopRecoveryFixtureBackend: &stopRecoveryFixtureBackend{}}
	r.b = b
	if !reflect.DeepEqual(r.projectAllocationFeatures(), []string{protocol.ProjectAllocationFeature}) {
		t.Fatal("adapter capability missing")
	}
	p, err := loadProject(r.s.db, id)
	g02Check(t, err)
	digest, err := p.Plan.Digest()
	g02Check(t, err)
	in := projectusage.Request{RunnerID: r.s.runnerID, ProjectID: id, ExpectedRevision: p.Revision, ReviewDigest: digest, IncludePrivateData: true}
	w := allocationCall(t, r, in, "operator")
	if w.Code != 409 || !bytes.Contains(w.Body.Bytes(), []byte("project_allocation_unsupported_phase")) || b.reads.Load() != 0 {
		t.Fatal("reservation implied known storage", w.Code, w.Body.String())
	}
}

func TestProjectAllocationRejectsSelectionBeforeBackend(t *testing.T) {
	r, b, _, selected := allocationAPIFixture(t)
	before := appAllRows(t, r.s.db)
	for _, variant := range []string{"observer", "no-consent", "runner", "project", "revision", "plan"} {
		in, role := selected, "operator"
		switch variant {
		case "observer":
			role = "observer"
		case "no-consent":
			in.IncludePrivateData = false
		case "runner":
			in.RunnerID = "other"
		case "project":
			in.ProjectID = "unknown"
		case "revision":
			in.ExpectedRevision++
		case "plan":
			in.ReviewDigest = fmt.Sprintf("%064x", 123)
		}
		if w := allocationCall(t, r, in, role); w.Code == 200 || bytes.Contains(w.Body.Bytes(), []byte("/fixture")) {
			t.Fatal("private read accepted", variant, w.Code, w.Body.String())
		}
	}
	if b.reads.Load() != 0 || !reflect.DeepEqual(before, appAllRows(t, r.s.db)) {
		t.Fatal("refusal probed or wrote")
	}
}

func TestProjectAllocationWithholdsInvalidBackendAndChangedLedger(t *testing.T) {
	r, b, _, in := allocationAPIFixture(t)
	before := appAllRows(t, r.s.db)
	for _, variant := range []string{"binding", "root", "growth", "partial-unknown", "future", "stale", "ledger"} {
		t.Run(variant, func(t *testing.T) {
			var original []byte
			b.onRead = nil
			b.change = func(out *projectusage.Observation) {
				switch variant {
				case "binding":
					out.Binding.Generation++
				case "root":
					out.DeclaredRoots[0].Path = "/other/bound"
					out.Roots[0].Root = out.DeclaredRoots[0]
				case "growth":
					g := projectusage.Compare(projectusage.Observation{}, *out, 0)
					out.Growth = &g
				case "partial-unknown":
					out.Status, out.Reason = "unknown", "unavailable"
				case "future":
					out.StartedAt = time.Now().Add(time.Hour)
					out.ReadAt = out.StartedAt
				case "stale":
					out.StartedAt = time.Now().Add(-time.Hour)
					out.ReadAt = out.StartedAt
				}
			}
			if variant == "ledger" {
				g02Check(t, r.s.db.QueryRow("SELECT body FROM project_expansions WHERE project_id=?", in.ProjectID).Scan(&original))
				b.onRead = func() {
					_, err := r.s.db.Exec("UPDATE project_expansions SET body=? WHERE project_id=?", append(append([]byte(nil), original...), byte(' ')), in.ProjectID)
					g02Check(t, err)
				}
			}
			w := allocationCall(t, r, in, "operator")
			if variant == "ledger" {
				_, err := r.s.db.Exec("UPDATE project_expansions SET body=? WHERE project_id=?", original, in.ProjectID)
				g02Check(t, err)
			}
			if w.Code != 409 || bytes.Contains(w.Body.Bytes(), []byte("/fixture")) || len(r.projectAllocations) != 0 || r.projectAllocationActive {
				t.Fatal("invalid allocation escaped", w.Code, w.Body.String())
			}
			if !reflect.DeepEqual(before, appAllRows(t, r.s.db)) {
				t.Fatal("original rows changed")
			}
		})
	}
}

func TestProjectAllocationAllowsOneCensusWithoutHoldingManagerLock(t *testing.T) {
	r, b, _, in := allocationAPIFixture(t)
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan *httptest.ResponseRecorder, 1)
	b.onRead = func() { close(entered); <-release }
	go func() { done <- allocationCall(t, r, in, "operator") }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("backend did not start")
	}
	w := allocationCall(t, r, in, "operator")
	if w.Code != 409 || !bytes.Contains(w.Body.Bytes(), []byte("project_allocation_busy")) {
		close(release)
		t.Fatal("duplicate census allowed", w.Code, w.Body.String())
	}
	close(release)
	select {
	case w := <-done:
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
	case <-time.After(time.Second):
		t.Fatal("census failed to complete")
	}
	if b.reads.Load() != 1 {
		t.Fatal("duplicate backend read")
	}
}

func TestProjectAllocationGrowthKeepsOwnedBaselineAndBoundedMemory(t *testing.T) {
	r, _, _, in := allocationAPIFixture(t)
	state, binding, err := r.s.projectAllocationSelection(t.Context(), in)
	g02Check(t, err)
	out := allocationFixtureObservation(state, binding)
	now := time.Now()
	if got := r.projectAllocationGrowth(out, now); got.Reason != "need-two-samples" {
		t.Fatal(got)
	}
	before := out.Clone()
	out.Roots[0].Totals.AllocatedBytes += 512
	out.Totals.AllocatedBytes += 512
	if got := r.projectAllocationGrowth(out, now.Add(time.Millisecond)); got.Reason != "window-too-short" {
		t.Fatal(got)
	}
	if !reflect.DeepEqual(r.projectAllocations[in.ProjectID].observation, before) {
		t.Fatal("fast poll or caller changed held sample")
	}
	if got := r.projectAllocationGrowth(out, now.Add(2*time.Second)); got.Status != "measured" || got.Allocated.Bytes != 512 || got.ElapsedNanos != int64(2*time.Second) {
		t.Fatal(got)
	}
	out.Binding.LedgerSHA256 = fmt.Sprintf("%064x", 1)
	if got := r.projectAllocationGrowth(out, now.Add(4*time.Second)); got.Reason != "binding-changed" {
		t.Fatal(got)
	}
	if got := r.projectAllocationGrowth(out, now.Add(10*time.Minute)); got.Reason != "sample-expired" {
		t.Fatal(got)
	}
	for i := 0; i < 34; i++ {
		next := out.Clone()
		next.Binding.ProjectID = fmt.Sprintf("project-%d", i)
		r.projectAllocationGrowth(next, now.Add(time.Duration(i+1)*time.Hour))
	}
	if len(r.projectAllocations) != 32 {
		t.Fatal("unbounded census memory")
	}
	if _, exists := r.projectAllocations[in.ProjectID]; exists {
		t.Fatal("oldest baseline not evicted")
	}
	out.Status, out.Reason, out.Roots, out.Totals = "unknown", "unavailable", nil, nil
	r.projectAllocations[in.ProjectID] = projectAllocationSample{before, now}
	if got := r.projectAllocationGrowth(out, now.Add(time.Second)); got.Reason != "current-unavailable" {
		t.Fatal(got)
	}
	if _, exists := r.projectAllocations[in.ProjectID]; exists {
		t.Fatal("unknown retained stale growth")
	}
}

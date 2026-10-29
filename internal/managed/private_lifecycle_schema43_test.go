package managed

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

func originalPrivate42(t *testing.T) g02Fixture {
	t.Helper()
	for _, f := range g02Fixtures(t) {
		if f.Schema == 42 {
			return f
		}
	}
	t.Fatal("original42 fixture is absent from supported inventory")
	return g02Fixture{}
}

func lifecycleWorkspace43(t *testing.T, s *store, phase string) privatechain.Workspace {
	t.Helper()
	claims, err := privateWorkspaceClaims(s.db)
	g02Check(t, err)
	for _, w := range claims {
		if w.Phase == phase {
			return w
		}
	}
	t.Fatal("original42 workspace missing", phase)
	return privatechain.Workspace{}
}

func workspaceResult43(w privatechain.Workspace) privatechain.Workspace {
	result := w.Clone()
	piers, a, b := w.Root, w.Root, w.Root
	piers.Inode++
	a.Inode += 2
	b.Inode += 3
	result.Phase = "observed"
	result.Observation = &privatechain.WorkspaceObservation{ObservedAt: time.Now().UTC(), Parent: w.Parent, Root: w.Root, PiersRoot: piers, Mounts: []pierstore.DirectoryIdentity{a, b}, Complete: true, Private: true, Empty: true, Synced: true}
	return result
}

func lifecycleGateway43(t *testing.T, s *store, w privatechain.Workspace, stopAt string) privatechain.Gateway {
	t.Helper()
	project := w.Binding.ProjectID
	p, err := loadPrivateProject(s.db, project)
	g02Check(t, err)
	inputs := privatechain.RuntimeInputInspection{Format: privatechain.RuntimeInputFormat, ManifestSHA256: strings.Repeat("1", 64), FoundationDigest: p.Binding.FoundationDigest, Files: 12352, Bytes: 834035190}
	planned, err := privatechain.NewGateway(p.Plan, w, "synthetic-gateway-operation", "private-engine", inputs)
	g02Check(t, err)
	req := privateGatewayRequest{RequestID: "initialize-" + project, ExpectedRevision: 1, ReviewDigest: planned.Selection.PlanDigest, WorkspaceOperationID: w.OperationID, WorkspaceDigest: planned.Selection.WorkspaceDigest, InputManifestSHA256: inputs.ManifestSHA256}
	op, hit, err := s.acceptPrivateGateway("synthetic-initializer", project, req, "private-engine", inputs)
	g02Check(t, err)
	if hit {
		t.Fatal("new lifecycle fixture initialization replayed")
	}
	g, err := s.beginPrivateGateway(project, op.ID)
	g02Check(t, err)
	for _, phase := range privateGatewayPhases[1:] {
		network, container := g.NetworkID, g.ContainerID
		if phase == "network-created" {
			network = fmt.Sprintf("%x", sha256.Sum256([]byte(project+":network")))
		}
		if phase == "gateway-created" {
			container = fmt.Sprintf("%x", sha256.Sum256([]byte(project+":container")))
		}
		var observation *privatechain.GatewayObservation
		if phase == "ready" {
			o := privateGatewayReady(t, s, g)
			observation = &o
		}
		next, err := g.Advance(p.Plan, w, g.Phase, phase, network, container, observation)
		g02Check(t, err)
		g02Check(t, s.advancePrivateGateway(project, op.ID, g.Phase, next))
		g = next
		if phase == stopAt {
			break
		}
	}
	if stopAt != "ready" {
		g02Check(t, s.interruptPrivateGateway(project, op.ID))
	}
	return g
}

// Populate every schema43 table over genuine original42 history. All identities,
// keys and observations here are synthetic; these helpers perform no host I/O.
func populatePrivateLifecycle43(t *testing.T, s *store) {
	t.Helper()
	complete := lifecycleWorkspace43(t, s, "observed")
	pending := lifecycleWorkspace43(t, s, "layout-dispatched")
	workspaceReq := privateWorkspaceObservationRequest{RequestID: "observe-original-workspace-1", ExpectedRevision: 1, ReviewDigest: pending.PlanDigest, SourceOperationID: pending.OperationID, SourceDigest: privateWorkspaceSourceDigest(pending)}
	first, _, err := s.acceptPrivateWorkspaceObservation("synthetic-observer", pending.Binding.ProjectID, workspaceReq)
	g02Check(t, err)
	_, err = s.beginPrivateWorkspaceObservation(first.ID)
	g02Check(t, err)
	g02Check(t, s.interruptPrivateWorkspaceObservation(first.ID))
	workspaceReq.RequestID = "observe-original-workspace-2"
	second, _, err := s.acceptPrivateWorkspaceObservation("synthetic-observer", pending.Binding.ProjectID, workspaceReq)
	g02Check(t, err)
	_, err = s.beginPrivateWorkspaceObservation(second.ID)
	g02Check(t, err)
	g02Check(t, s.completePrivateWorkspaceObservation(second.ID, workspaceResult43(pending)))
	recovered, err := readyPrivateWorkspace(s.db, pending.Binding.ProjectID)
	g02Check(t, err)
	unknown := lifecycleGateway43(t, s, complete, "start-dispatched")
	lifecycleGateway43(t, s, recovered, "ready")
	req := privateGatewayObservationRequest{RequestID: "observe-original-gateway-1", ExpectedRevision: 1, ReviewDigest: unknown.Selection.PlanDigest, SourceOperationID: unknown.Selection.OperationID, SourceDigest: privateGatewaySourceDigest(unknown)}
	first, _, err = s.acceptPrivateGatewayObservation("synthetic-observer", complete.Binding.ProjectID, req)
	g02Check(t, err)
	_, err = s.beginPrivateGatewayObservation(first.ID)
	g02Check(t, err)
	g02Check(t, s.interruptPrivateGatewayObservation(first.ID))
	req.RequestID = "observe-original-gateway-2"
	second, _, err = s.acceptPrivateGatewayObservation("synthetic-observer", complete.Binding.ProjectID, req)
	g02Check(t, err)
	g02Check(t, s.runPrivateGatewayObservation(context.Background(), second.ID, &gatewayObservationOnly{t: t, s: s, operation: second.ID}))
	_, err = projectReservations(s.db)
	g02Check(t, err)
}

func lifecycleRows43(t *testing.T, s *store) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, table := range []string{"private_workspace_observations", "private_project_gateways", "private_gateway_observations"} {
		rows, err := s.db.Query("SELECT operation_id,body FROM " + table + " ORDER BY operation_id")
		g02Check(t, err)
		count := 0
		for rows.Next() {
			var id string
			var raw []byte
			g02Check(t, rows.Scan(&id, &raw))
			out[table+":"+id] = string(raw)
			count++
		}
		g02Check(t, rows.Err())
		g02Check(t, rows.Close())
		if count != 2 {
			t.Fatal("fixture lacks completed/interrupted lifecycle history", table, count)
		}
	}
	rows, err := s.db.Query("SELECT id,body,runtime FROM operations WHERE json_extract(body,'$.kind') IN (?,?,?) ORDER BY id", privateGatewayKind, privateWorkspaceObservationKind, privateGatewayObservationKind)
	g02Check(t, err)
	defer rows.Close()
	for rows.Next() {
		var id string
		var body, mirror []byte
		g02Check(t, rows.Scan(&id, &body, &mirror))
		out["operation:"+id] = string(body)
		out["mirror:"+id] = string(mirror)
	}
	g02Check(t, rows.Err())
	return out
}

// Compare every original operation by its retained primary key: schema43 adds
// six receipts, so the older helper's exact table-size assertion is inappropriate
// after population. All other original tables remain exactly the same size.
func assertOriginalPrivate42Rows(t *testing.T, s *store, f g02Fixture) {
	t.Helper()
	assertOriginalPrivateRowsWithAdditional(t, s, f, 6)
}

func assertOriginalPrivateRowsWithAdditional(t *testing.T, s *store, f g02Fixture, additional int) {
	t.Helper()
	rest := f
	rest.Tables = nil
	for _, table := range f.Tables {
		if table.Name != "operations" {
			rest.Tables = append(rest.Tables, table)
			continue
		}
		for _, expected := range table.Rows {
			id := g02Cell(t, expected[0])
			rows, err := s.db.Query("SELECT * FROM operations WHERE id=?", id)
			g02Check(t, err)
			columns, err := rows.Columns()
			g02Check(t, err)
			if !reflect.DeepEqual(columns, table.Columns) || !rows.Next() {
				t.Fatal("original operation missing or columns changed", id)
			}
			got := make([]any, len(columns))
			ptrs := make([]any, len(columns))
			for i := range got {
				ptrs[i] = &got[i]
			}
			g02Check(t, rows.Scan(ptrs...))
			for i, cell := range expected {
				if !reflect.DeepEqual(g02Cell(t, cell), got[i]) {
					t.Fatal("original operation changed", id, columns[i])
				}
			}
			if rows.Next() {
				t.Fatal("duplicated original operation", id)
			}
			g02Check(t, rows.Err())
			g02Check(t, rows.Close())
		}
		var count int
		g02Check(t, s.db.QueryRow("SELECT count(*) FROM operations").Scan(&count))
		if count != len(table.Rows)+additional {
			t.Fatal("unexpected lifecycle receipt count", count)
		}
	}
	g02AssertRows(t, s.db, rest, nil)
}

func TestPrivateLifecycleSchema43PreservesOriginalAndNewReceipts(t *testing.T) {
	f := originalPrivate42(t)
	dir := g02Materialize(t, f)
	s := g02Open(t, dir)
	g02AssertRows(t, s.db, f, nil)
	populatePrivateLifecycle43(t, s)
	assertOriginalPrivate42Rows(t, s, f)
	before := lifecycleRows43(t, s)
	g02Check(t, s.close())
	for pass := 0; pass < 3; pass++ {
		s = g02Open(t, dir)
		assertOriginalPrivate42Rows(t, s, f)
		if !reflect.DeepEqual(before, lifecycleRows43(t, s)) {
			t.Fatal("reopen changed schema43 history")
		}
		claims, err := privateWorkspaceClaims(s.db)
		g02Check(t, err)
		for _, w := range claims {
			ready, evidence, err := readyPrivateGateway(s.db, w.Binding.ProjectID)
			g02Check(t, err)
			if ready.Phase != "ready" || evidence.Status != "succeeded" {
				t.Fatal("lost durable readiness")
			}
			if w.Phase == "observed" && evidence.ID == ready.Selection.OperationID {
				t.Fatal("recovered readiness lost separate observation attribution")
			}
			if w.Phase == "layout-dispatched" && evidence.ID != ready.Selection.OperationID {
				t.Fatal("completed initialization attribution changed")
			}
		}
		g02Check(t, s.close())
	}
	g02Sidecars(t, dir, f)
}

func TestPrivateLifecycleSchema43MigrationFailureRollsBackTables(t *testing.T) {
	f := originalPrivate42(t)
	dir := g02Materialize(t, f)
	db := g02RawDB(t, dir)
	var project string
	var body []byte
	g02Check(t, db.QueryRow("SELECT project_id,body FROM private_project_workspaces LIMIT 1").Scan(&project, &body))
	_, err := db.Exec("UPDATE private_project_workspaces SET body='{}' WHERE project_id=?", project)
	g02Check(t, err)
	g02Check(t, db.Close())
	path := filepath.Join(dir, "runner.sqlite")
	before, err := os.ReadFile(path)
	g02Check(t, err)
	for pass := 0; pass < 2; pass++ {
		failed, err := openStore(dir)
		if err == nil {
			failed.close()
			t.Fatal("corrupt original workspace migrated")
		}
		after, err := os.ReadFile(path)
		g02Check(t, err)
		if !bytes.Equal(before, after) {
			t.Fatal("failed migration changed original42 database")
		}
	}
	db = g02RawDB(t, dir)
	var version, count int
	g02Check(t, db.QueryRow("PRAGMA user_version").Scan(&version))
	g02Check(t, db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name IN ('private_workspace_observations','private_project_gateways','private_gateway_observations')").Scan(&count))
	if version != 42 || count != 0 {
		t.Fatal("partial migration committed", version, count)
	}
	_, err = db.Exec("UPDATE private_project_workspaces SET body=? WHERE project_id=?", body, project)
	g02Check(t, err)
	g02Check(t, db.Close())
	s := g02Open(t, dir)
	g02AssertRows(t, s.db, f, nil)
	g02Check(t, s.close())
	g02Sidecars(t, dir, f)
}

func TestPrivateLifecycleSchema43CorruptionRefusesOpenWithoutChanges(t *testing.T) {
	for _, table := range []string{"private_workspace_observations", "private_project_gateways", "private_gateway_observations"} {
		for _, target := range []string{"journal", "mirror"} {
			t.Run(table+"/"+target, func(t *testing.T) {
				dir := g02Materialize(t, originalPrivate42(t))
				s := g02Open(t, dir)
				populatePrivateLifecycle43(t, s)
				var id string
				g02Check(t, s.db.QueryRow("SELECT operation_id FROM "+table+" LIMIT 1").Scan(&id))
				var err error
				if target == "journal" {
					_, err = s.db.Exec("UPDATE "+table+" SET body='{}' WHERE operation_id=?", id)
				} else {
					_, err = s.db.Exec("UPDATE operations SET runtime='{}' WHERE id=?", id)
				}
				g02Check(t, err)
				g02Check(t, s.close())
				path := filepath.Join(dir, "runner.sqlite")
				before, err := os.ReadFile(path)
				g02Check(t, err)
				failed, err := openStore(dir)
				if err == nil {
					failed.close()
					t.Fatal("corrupt lifecycle state opened")
				}
				after, err := os.ReadFile(path)
				g02Check(t, err)
				if !bytes.Equal(before, after) {
					t.Fatal("failed open modified lifecycle evidence")
				}
			})
		}
	}
}

func TestPrivateLifecycleSchema43WorkspaceCannotClaimGatewayObjects(t *testing.T) {
	s := g02Open(t, g02Materialize(t, originalPrivate42(t)))
	complete := lifecycleWorkspace43(t, s, "observed")
	pending := lifecycleWorkspace43(t, s, "layout-dispatched")
	gateway := lifecycleGateway43(t, s, complete, "ready")
	before, err := json.Marshal(gateway)
	g02Check(t, err)
	req := privateWorkspaceObservationRequest{RequestID: "observe-cross-domain", ExpectedRevision: 1, ReviewDigest: pending.PlanDigest, SourceOperationID: pending.OperationID, SourceDigest: privateWorkspaceSourceDigest(pending)}
	op, _, err := s.acceptPrivateWorkspaceObservation("observer", pending.Binding.ProjectID, req)
	g02Check(t, err)
	_, err = s.beginPrivateWorkspaceObservation(op.ID)
	g02Check(t, err)
	result := workspaceResult43(pending)
	result.Observation.PiersRoot = gateway.Observation.Chain
	if err = s.completePrivateWorkspaceObservation(op.ID, result); err == nil {
		t.Fatal("workspace claimed another project's chain directory")
	}
	j, err := loadPrivateWorkspaceObservation(s.db, op.ID)
	g02Check(t, err)
	if j.Phase != "observing" || j.Result != nil {
		t.Fatal("invalid observation partly committed")
	}
	_, err = projectReservations(s.db)
	g02Check(t, err)
	g02Check(t, s.completePrivateWorkspaceObservation(op.ID, workspaceResult43(pending)))
	saved, err := loadPrivateGateway(s.db, complete.Binding.ProjectID)
	g02Check(t, err)
	after, err := json.Marshal(saved)
	g02Check(t, err)
	if !bytes.Equal(before, after) {
		t.Fatal("recovery changed another project's gateway")
	}
	g02Check(t, s.close())
}

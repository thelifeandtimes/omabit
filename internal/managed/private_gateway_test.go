package managed

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

// Fixtures use the ordinary schema43 store initialization and ownership checks.
func privateGatewayStoreFixture(t *testing.T) (*store, storedPrivateProject, privateGatewayRequest, privatechain.RuntimeInputInspection) {
	t.Helper()
	s := testStore(t)
	p, req := privateWorkspaceFixture(t, s)
	op, _, err := s.acceptPrivateWorkspace("owner", p.Binding.ProjectID, req, privateWorkspaceParent())
	g02Check(t, err)
	g02Check(t, s.runPrivateWorkspace(context.Background(), p.Binding.ProjectID, op.ID, &privateWorkspaceStoreDriver{t: t, s: s}))
	w, err := loadPrivateWorkspace(s.db, p.Binding.ProjectID)
	g02Check(t, err)
	inputs := privatechain.RuntimeInputInspection{Format: privatechain.RuntimeInputFormat, ManifestSHA256: strings.Repeat("1", 64), FoundationDigest: p.Binding.FoundationDigest, Files: 12352, Bytes: 834035190}
	g, err := privatechain.NewGateway(p.Plan, w, "prototype-operation", "private-engine", inputs)
	g02Check(t, err)
	return s, p, privateGatewayRequest{RequestID: "initialize-private-gateway", ExpectedRevision: 1, ReviewDigest: g.Selection.PlanDigest, WorkspaceOperationID: w.OperationID, WorkspaceDigest: g.Selection.WorkspaceDigest, InputManifestSHA256: inputs.ManifestSHA256}, inputs
}

func privateGatewayReady(t *testing.T, s *store, g privatechain.Gateway) privatechain.GatewayObservation {
	t.Helper()
	p, err := loadPrivateProject(s.db, g.Selection.Binding.ProjectID)
	g02Check(t, err)
	w, err := readyPrivateWorkspace(s.db, g.Selection.Binding.ProjectID)
	g02Check(t, err)
	ns := projectnetwork.NamespaceIdentity{BootID: "01234567-89ab-cdef-0123-456789abcdef", Device: 4, Inode: 101}
	host := ns
	host.Inode = 100
	chain, secrets := w.Root, w.Root
	chain.Inode = w.Root.Inode + 4
	secrets.Inode = w.Root.Inode + 5
	o := privatechain.GatewayObservation{Selection: g.Selection, ObservedAt: time.Now().UTC(), NetworkID: g.NetworkID, ContainerID: g.ContainerID, ImageID: runtimecandidate.ConfigDigest, Namespace: ns, HostNamespace: host, Root: w.Root, Chain: chain, Secrets: secrets, Mounts: w.Observation.Mounts, Foundation: p.Plan.Foundation, Head: 0x50, HeadHash: "0x" + strings.Repeat("f", 64), CompleteContract: true, Running: true, Private: true, Synced: true, RPCReadOnly: true}
	for _, m := range p.Plan.Members {
		auth := sha256.Sum256([]byte("public synthetic auth:" + g.Selection.Binding.ProjectID + ":" + m.Ship))
		crypt := sha256.Sum256([]byte("public synthetic crypt:" + g.Selection.Binding.ProjectID + ":" + m.Ship))
		o.PublicKeys = append(o.PublicKeys, privatechain.GatewayPublicKey{Ship: m.Ship, Auth: fmt.Sprintf("0x%x", auth), Crypt: fmt.Sprintf("0x%x", crypt), Life: 1, Suite: 1})
		o.Publications = append(o.Publications, projectnetwork.Publication{InternalPort: m.InternalHTTPPort, HostAddress: "127.0.0.1", HostPort: m.HTTPPort, Protocol: "tcp"})
	}
	return o.Clone()
}

type privateGatewayStoreDriver struct {
	t     *testing.T
	s     *store
	calls []string
	fail  string
}

func (d *privateGatewayStoreDriver) check(name string, g privatechain.Gateway) error {
	d.t.Helper()
	d.calls = append(d.calls, name)
	saved, err := loadPrivateGateway(d.s.db, g.Selection.Binding.ProjectID)
	g02Check(d.t, err)
	if !reflect.DeepEqual(saved, g) {
		d.t.Fatal("gateway effect preceded durable checkpoint", name)
	}
	if d.fail == name {
		return errors.New("synthetic private driver detail")
	}
	return nil
}
func (d *privateGatewayStoreDriver) VerifyPreparation(_ context.Context, g privatechain.Gateway) error {
	return d.check("verify-"+g.Phase, g)
}
func (d *privateGatewayStoreDriver) CreateNetwork(_ context.Context, g privatechain.Gateway) (string, error) {
	return strings.Repeat("2", 64), d.check("create-network", g)
}
func (d *privateGatewayStoreDriver) InspectNetwork(_ context.Context, g privatechain.Gateway) error {
	return d.check("inspect-network", g)
}
func (d *privateGatewayStoreDriver) CreateGateway(_ context.Context, g privatechain.Gateway) (string, error) {
	return strings.Repeat("3", 64), d.check("create-gateway", g)
}
func (d *privateGatewayStoreDriver) InspectGateway(_ context.Context, g privatechain.Gateway) error {
	return d.check("inspect-gateway", g)
}
func (d *privateGatewayStoreDriver) StartGateway(_ context.Context, g privatechain.Gateway) error {
	return d.check("start-gateway", g)
}
func (d *privateGatewayStoreDriver) ObserveGateway(_ context.Context, g privatechain.Gateway) (privatechain.GatewayObservation, error) {
	return privateGatewayReady(d.t, d.s, g), d.check("observe-gateway", g)
}

func TestPrivateGatewayDurableExecutionAndActorReplay(t *testing.T) {
	s, p, req, inputs := privateGatewayStoreFixture(t)
	workspace, err := loadPrivateWorkspace(s.db, p.Binding.ProjectID)
	g02Check(t, err)
	op, hit, err := s.acceptPrivateGateway("owner", p.Binding.ProjectID, req, "private-engine", inputs)
	g02Check(t, err)
	if hit || op.Status != "queued" {
		t.Fatal("initial admission", op, hit)
	}
	d := &privateGatewayStoreDriver{t: t, s: s}
	g02Check(t, s.runPrivateGateway(context.Background(), p.Binding.ProjectID, op.ID, d))
	g, err := loadPrivateGateway(s.db, p.Binding.ProjectID)
	g02Check(t, err)
	actual, err := loadOperation(s.db, op.ID)
	g02Check(t, err)
	if g.Phase != "ready" || actual.Status != "succeeded" || len(d.calls) != 9 {
		t.Fatal(g.Phase, actual.Status, d.calls)
	}
	after, err := loadPrivateWorkspace(s.db, p.Binding.ProjectID)
	g02Check(t, err)
	if !reflect.DeepEqual(workspace, after) {
		t.Fatal("initialization rewrote workspace receipt")
	}
	req.ReplayOnly = true
	replay, hit, err := s.acceptPrivateGateway("owner", p.Binding.ProjectID, req, "", privatechain.RuntimeInputInspection{})
	g02Check(t, err)
	if !hit || !reflect.DeepEqual(replay, actual) {
		t.Fatal("replay substituted backend input")
	}
	_, _, err = s.acceptPrivateGateway("another-actor", p.Binding.ProjectID, req, "private-engine", inputs)
	requireFault(t, err, "replay_not_found")
	req.InputManifestSHA256 = strings.Repeat("f", 64)
	_, _, err = s.acceptPrivateGateway("owner", p.Binding.ProjectID, req, "private-engine", inputs)
	requireFault(t, err, "request_conflict")
	if err = s.runPrivateGateway(context.Background(), p.Binding.ProjectID, op.ID, d); err == nil || len(d.calls) != 9 {
		t.Fatal("completed initialization was replayed")
	}
}

var privateGatewayPhases = []string{"planned", "network-dispatched", "network-created", "gateway-dispatched", "gateway-created", "start-dispatched", "started", "ready"}

func advancePrivateGatewayFixture(t *testing.T, s *store, p storedPrivateProject, g privatechain.Gateway, phase string) privatechain.Gateway {
	t.Helper()
	w, err := readyPrivateWorkspace(s.db, p.Binding.ProjectID)
	g02Check(t, err)
	network, container := g.NetworkID, g.ContainerID
	if phase == "network-created" {
		network = strings.Repeat("2", 64)
	}
	if phase == "gateway-created" {
		container = strings.Repeat("3", 64)
	}
	var o *privatechain.GatewayObservation
	if phase == "ready" {
		ready := privateGatewayReady(t, s, g)
		o = &ready
	}
	next, err := g.Advance(p.Plan, w, g.Phase, phase, network, container, o)
	g02Check(t, err)
	g02Check(t, s.advancePrivateGateway(p.Binding.ProjectID, g.Selection.OperationID, g.Phase, next))
	return next
}

func TestPrivateGatewayInterruptedPhasesNeverRestartInitialization(t *testing.T) {
	for _, phase := range privateGatewayPhases {
		t.Run(phase, func(t *testing.T) {
			s, p, req, inputs := privateGatewayStoreFixture(t)
			op, _, err := s.acceptPrivateGateway("owner", p.Binding.ProjectID, req, "private-engine", inputs)
			g02Check(t, err)
			g, err := loadPrivateGateway(s.db, p.Binding.ProjectID)
			g02Check(t, err)
			if phase != "planned" {
				g, err = s.beginPrivateGateway(p.Binding.ProjectID, op.ID)
				g02Check(t, err)
				for _, next := range privateGatewayPhases[1:] {
					g = advancePrivateGatewayFixture(t, s, p, g, next)
					if next == phase {
						break
					}
				}
			}
			before, _ := json.Marshal(g)
			dir := s.directory
			g02Check(t, s.close())
			s = g02Open(t, dir)
			g02Check(t, s.reconcileInterrupted())
			g02Check(t, s.reconcileInterrupted())
			after, err := loadPrivateGateway(s.db, p.Binding.ProjectID)
			g02Check(t, err)
			afterRaw, _ := json.Marshal(after)
			if !reflect.DeepEqual(before, afterRaw) {
				t.Fatal("restart changed the original attempt")
			}
			actual, err := loadOperation(s.db, op.ID)
			g02Check(t, err)
			want := "interrupted"
			if phase == "ready" {
				want = "succeeded"
			}
			if actual.Status != want {
				t.Fatal(actual.Status)
			}
			_, err = s.beginPrivateGateway(p.Binding.ProjectID, op.ID)
			requireFault(t, err, "stale_project_operation")
			req.ReplayOnly = true
			replayed, hit, err := s.acceptPrivateGateway("owner", p.Binding.ProjectID, req, "", privatechain.RuntimeInputInspection{})
			g02Check(t, err)
			if !hit || replayed.ID != op.ID || replayed.Status != want {
				t.Fatal("receipt replay changed")
			}
			req.RequestID = "another-initialization"
			req.ReplayOnly = false
			_, _, err = s.acceptPrivateGateway("owner", p.Binding.ProjectID, req, "private-engine", inputs)
			requireFault(t, err, "project_recovery_required")
			g02Check(t, privateGatewayState(s.db))
		})
	}
}

func TestPrivateGatewayLostEffectsRetainKnownResources(t *testing.T) {
	for failure, phase := range map[string]string{"create-network": "network-created", "create-gateway": "gateway-created", "start-gateway": "start-dispatched", "observe-gateway": "started"} {
		t.Run(failure, func(t *testing.T) {
			s, p, req, inputs := privateGatewayStoreFixture(t)
			op, _, err := s.acceptPrivateGateway("owner", p.Binding.ProjectID, req, "private-engine", inputs)
			g02Check(t, err)
			d := &privateGatewayStoreDriver{t: t, s: s, fail: failure}
			if err = s.runPrivateGateway(context.Background(), p.Binding.ProjectID, op.ID, d); !errors.Is(err, privatechain.ErrGatewayUnresolved) || strings.Contains(err.Error(), "synthetic private") {
				t.Fatal(err)
			}
			g, err := loadPrivateGateway(s.db, p.Binding.ProjectID)
			g02Check(t, err)
			actual, err := loadOperation(s.db, op.ID)
			g02Check(t, err)
			if g.Phase != phase || actual.Status != "interrupted" {
				t.Fatal(g.Phase, actual.Status)
			}
			calls := len(d.calls)
			if err = s.runPrivateGateway(context.Background(), p.Binding.ProjectID, op.ID, d); err == nil || len(d.calls) != calls {
				t.Fatal("uncertain mutation retried")
			}
		})
	}
}

func TestPrivateGatewayTransactionsRollbackBothMirrors(t *testing.T) {
	for _, failure := range []string{"insert", "mirror", "advance"} {
		t.Run(failure, func(t *testing.T) {
			s, p, req, inputs := privateGatewayStoreFixture(t)
			var g privatechain.Gateway
			if failure == "advance" {
				op, _, err := s.acceptPrivateGateway("owner", p.Binding.ProjectID, req, "private-engine", inputs)
				g02Check(t, err)
				g, err = s.beginPrivateGateway(p.Binding.ProjectID, op.ID)
				g02Check(t, err)
			}
			stmt := `CREATE TRIGGER fail_gateway BEFORE UPDATE OF runtime ON operations BEGIN SELECT RAISE(ABORT,'synthetic mirror failure'); END;`
			if failure == "insert" {
				stmt = `CREATE TRIGGER fail_gateway BEFORE INSERT ON private_project_gateways BEGIN SELECT RAISE(ABORT,'synthetic journal failure'); END;`
			}
			_, err := s.db.Exec(stmt)
			g02Check(t, err)
			if failure == "advance" {
				w, err := readyPrivateWorkspace(s.db, p.Binding.ProjectID)
				g02Check(t, err)
				next, err := g.Advance(p.Plan, w, g.Phase, "network-dispatched", "", "", nil)
				g02Check(t, err)
				if err = s.advancePrivateGateway(p.Binding.ProjectID, g.Selection.OperationID, g.Phase, next); err == nil {
					t.Fatal("partial checkpoint succeeded")
				}
				after, err := loadPrivateGateway(s.db, p.Binding.ProjectID)
				g02Check(t, err)
				if !reflect.DeepEqual(g, after) {
					t.Fatal("checkpoint partially committed")
				}
			} else {
				if _, _, err = s.acceptPrivateGateway("owner", p.Binding.ProjectID, req, "private-engine", inputs); err == nil {
					t.Fatal("partial acceptance succeeded")
				}
				var count int
				g02Check(t, s.db.QueryRow("SELECT count(*) FROM private_project_gateways").Scan(&count))
				if count != 0 {
					t.Fatal("partial journal")
				}
				g02Check(t, s.db.QueryRow("SELECT count(*) FROM operations WHERE request_id=?", req.RequestID).Scan(&count))
				if count != 0 {
					t.Fatal("partial operation")
				}
			}
			g02Check(t, privateGatewayState(s.db))
		})
	}
}

func TestPrivateGatewayCorruptionRefusesReplayAndExecution(t *testing.T) {
	for name, query := range map[string]string{
		"journal":       `UPDATE private_project_gateways SET body=json_set(body,'$.selection.engine_id','changed')`,
		"unknown-field": `UPDATE private_project_gateways SET body=json_set(body,'$.unknown',true)`,
		"mirror":        `UPDATE operations SET runtime='{}' WHERE json_extract(body,'$.kind')='initialize-private-project-gateway'`,
		"fingerprint":   `UPDATE operations SET fingerprint='changed' WHERE json_extract(body,'$.kind')='initialize-private-project-gateway'`,
		"actor":         `UPDATE operations SET actor='changed' WHERE json_extract(body,'$.kind')='initialize-private-project-gateway'`,
		"orphan":        `DELETE FROM private_project_gateways`,
		"phase":         `UPDATE private_project_gateways SET body=json_set(body,'$.phase','ready')`,
	} {
		t.Run(name, func(t *testing.T) {
			s, p, req, inputs := privateGatewayStoreFixture(t)
			op, _, err := s.acceptPrivateGateway("owner", p.Binding.ProjectID, req, "private-engine", inputs)
			g02Check(t, err)
			_, err = s.db.Exec(query)
			g02Check(t, err)
			if _, _, err = s.acceptPrivateGateway("owner", p.Binding.ProjectID, req, "", privatechain.RuntimeInputInspection{}); err == nil {
				t.Fatal("corrupt replay passed")
			}
			d := &privateGatewayStoreDriver{t: t, s: s}
			if err = s.runPrivateGateway(context.Background(), p.Binding.ProjectID, op.ID, d); err == nil || len(d.calls) != 0 {
				t.Fatal("corruption reached a driver")
			}
		})
	}
}

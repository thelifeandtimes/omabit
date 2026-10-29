package managed

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

func privateGatewayObservationFixture(t *testing.T) (*store, storedPrivateProject, privateGatewayObservationRequest) {
	t.Helper()
	s, p, req, inputs := privateGatewayStoreFixture(t)
	op, _, err := s.acceptPrivateGateway("owner", p.Binding.ProjectID, req, "private-engine", inputs)
	g02Check(t, err)
	d := &privateGatewayStoreDriver{t: t, s: s, fail: "start-gateway"}
	if err = s.runPrivateGateway(context.Background(), p.Binding.ProjectID, op.ID, d); !errors.Is(err, privatechain.ErrGatewayUnresolved) {
		t.Fatal(err)
	}
	g, err := loadPrivateGateway(s.db, p.Binding.ProjectID)
	g02Check(t, err)
	return s, p, privateGatewayObservationRequest{RequestID: "observe-private-gateway", ExpectedRevision: 1, ReviewDigest: g.Selection.PlanDigest, SourceOperationID: op.ID, SourceDigest: privateGatewaySourceDigest(g)}
}

type gatewayObservationOnly struct {
	t         *testing.T
	s         *store
	operation string
	calls     int
	fail      bool
	modify    func(*privatechain.GatewayObservation)
}

func (o *gatewayObservationOnly) ObserveGateway(_ context.Context, g privatechain.Gateway) (privatechain.GatewayObservation, error) {
	o.t.Helper()
	o.calls++
	j, err := loadPrivateGatewayObservation(o.s.db, o.operation)
	g02Check(o.t, err)
	if j.Phase != "observing" || j.SourceOperationID != g.Selection.OperationID || j.SourceDigest != privateGatewaySourceDigest(g) {
		o.t.Fatal("gateway inspection preceded attributed receipt")
	}
	if o.fail {
		return privatechain.GatewayObservation{}, errors.New("private gateway observation detail")
	}
	result := privateGatewayReady(o.t, o.s, g)
	if o.modify != nil {
		o.modify(&result)
	}
	return result, nil
}

func TestPrivateGatewayObservationPreservesUnknownStart(t *testing.T) {
	s, p, req := privateGatewayObservationFixture(t)
	g, err := loadPrivateGateway(s.db, p.Binding.ProjectID)
	g02Check(t, err)
	source, err := loadOperation(s.db, req.SourceOperationID)
	g02Check(t, err)
	if _, _, err = readyPrivateGateway(s.db, p.Binding.ProjectID); err == nil {
		t.Fatal("unknown Start claimed readiness")
	}
	op, hit, err := s.acceptPrivateGatewayObservation("observer", p.Binding.ProjectID, req)
	g02Check(t, err)
	if hit {
		t.Fatal("new request replayed")
	}
	o := &gatewayObservationOnly{t: t, s: s, operation: op.ID}
	g02Check(t, s.runPrivateGatewayObservation(context.Background(), op.ID, o))
	after, err := loadPrivateGateway(s.db, p.Binding.ProjectID)
	g02Check(t, err)
	old, err := loadOperation(s.db, source.ID)
	g02Check(t, err)
	if !reflect.DeepEqual(g, after) || !reflect.DeepEqual(source, old) || o.calls != 1 {
		t.Fatal("observation changed unknown initialization")
	}
	ready, evidence, err := readyPrivateGateway(s.db, p.Binding.ProjectID)
	g02Check(t, err)
	if ready.Phase != "ready" || ready.Selection.OperationID != source.ID || evidence.ID != op.ID || evidence.Status != "succeeded" {
		t.Fatal("readiness lost source attribution")
	}
	req.ReplayOnly = true
	replayed, hit, err := s.acceptPrivateGatewayObservation("observer", p.Binding.ProjectID, req)
	g02Check(t, err)
	if !hit || !reflect.DeepEqual(replayed, evidence) {
		t.Fatal("observation replay changed")
	}
	_, _, err = s.acceptPrivateGatewayObservation("another-actor", p.Binding.ProjectID, req)
	requireFault(t, err, "replay_not_found")
	if err = s.runPrivateGatewayObservation(context.Background(), op.ID, o); err == nil || o.calls != 1 {
		t.Fatal("completed observation reexecuted")
	}
	_, err = s.beginPrivateGateway(p.Binding.ProjectID, source.ID)
	requireFault(t, err, "stale_project_operation")
	req.RequestID = "duplicate-observation"
	req.ReplayOnly = false
	_, _, err = s.acceptPrivateGatewayObservation("observer", p.Binding.ProjectID, req)
	requireFault(t, err, "project_operation_pending")
}

func TestPrivateGatewayObservationRestartRequiresNewInspectionIntent(t *testing.T) {
	for _, phase := range []string{"planned", "observing", "failed-observer"} {
		t.Run(phase, func(t *testing.T) {
			s, p, req := privateGatewayObservationFixture(t)
			op, _, err := s.acceptPrivateGatewayObservation("observer", p.Binding.ProjectID, req)
			g02Check(t, err)
			if phase == "observing" {
				_, err = s.beginPrivateGatewayObservation(op.ID)
				g02Check(t, err)
			}
			if phase == "failed-observer" {
				o := &gatewayObservationOnly{t: t, s: s, operation: op.ID, fail: true}
				if err = s.runPrivateGatewayObservation(context.Background(), op.ID, o); err == nil || strings.Contains(err.Error(), "private gateway observation detail") {
					t.Fatal(err)
				}
			}
			dir := s.directory
			g02Check(t, s.close())
			s = g02Open(t, dir)
			g02Check(t, s.reconcileInterrupted())
			g02Check(t, s.reconcileInterrupted())
			actual, err := loadOperation(s.db, op.ID)
			g02Check(t, err)
			if actual.Status != "interrupted" {
				t.Fatal(actual.Status)
			}
			_, err = s.beginPrivateGatewayObservation(op.ID)
			requireFault(t, err, "stale_project_operation")
			req.RequestID = "observe-gateway-again"
			second, _, err := s.acceptPrivateGatewayObservation("another-observer", p.Binding.ProjectID, req)
			g02Check(t, err)
			o := &gatewayObservationOnly{t: t, s: s, operation: second.ID}
			g02Check(t, s.runPrivateGatewayObservation(context.Background(), second.ID, o))
			g, err := loadPrivateGateway(s.db, p.Binding.ProjectID)
			g02Check(t, err)
			if g.Phase != "start-dispatched" {
				t.Fatal("observation modified original start")
			}
		})
	}
}

func TestPrivateGatewayObservationRejectsChangedEvidence(t *testing.T) {
	for name, modify := range map[string]func(*privatechain.GatewayObservation){
		"stopped":  func(o *privatechain.GatewayObservation) { o.Running = false },
		"resource": func(o *privatechain.GatewayObservation) { o.ContainerID = strings.Repeat("f", 64) },
		"head":     func(o *privatechain.GatewayObservation) { o.Head++ },
		"key-life": func(o *privatechain.GatewayObservation) { o.PublicKeys[0].Life++ },
		"ports":    func(o *privatechain.GatewayObservation) { o.Publications[0].HostPort++ },
	} {
		t.Run(name, func(t *testing.T) {
			s, p, req := privateGatewayObservationFixture(t)
			op, _, err := s.acceptPrivateGatewayObservation("observer", p.Binding.ProjectID, req)
			g02Check(t, err)
			o := &gatewayObservationOnly{t: t, s: s, operation: op.ID, modify: modify}
			if err = s.runPrivateGatewayObservation(context.Background(), op.ID, o); err == nil {
				t.Fatal("changed evidence accepted")
			}
			j, err := loadPrivateGatewayObservation(s.db, op.ID)
			g02Check(t, err)
			actual, err := loadOperation(s.db, op.ID)
			g02Check(t, err)
			if j.Result != nil || j.Phase != "observing" || actual.Status != "interrupted" {
				t.Fatal("failed observation fabricated readiness")
			}
		})
	}
}

func TestPrivateGatewayObservationCompletionRollback(t *testing.T) {
	s, p, req := privateGatewayObservationFixture(t)
	op, _, err := s.acceptPrivateGatewayObservation("observer", p.Binding.ProjectID, req)
	g02Check(t, err)
	_, err = s.beginPrivateGatewayObservation(op.ID)
	g02Check(t, err)
	g, err := loadPrivateGateway(s.db, p.Binding.ProjectID)
	g02Check(t, err)
	result := g.Clone()
	result.Phase = "ready"
	o := privateGatewayReady(t, s, g)
	result.Observation = &o
	_, err = s.db.Exec(`CREATE TRIGGER fail_gateway_observation BEFORE UPDATE OF runtime ON operations WHEN json_extract(new.body,'$.kind')='observe-private-project-gateway' BEGIN SELECT RAISE(ABORT,'synthetic mirror failure'); END;`)
	g02Check(t, err)
	if err = s.completePrivateGatewayObservation(op.ID, result); err == nil {
		t.Fatal("partial completion committed")
	}
	j, err := loadPrivateGatewayObservation(s.db, op.ID)
	g02Check(t, err)
	if j.Phase != "observing" || j.Result != nil {
		t.Fatal("partial observation")
	}
	g02Check(t, privateGatewayState(s.db))
}

func TestPrivateGatewayObservationCannotResolveMissingStartDispatch(t *testing.T) {
	s, p, req, inputs := privateGatewayStoreFixture(t)
	op, _, err := s.acceptPrivateGateway("owner", p.Binding.ProjectID, req, "private-engine", inputs)
	g02Check(t, err)
	d := &privateGatewayStoreDriver{t: t, s: s, fail: "inspect-gateway"}
	if err = s.runPrivateGateway(context.Background(), p.Binding.ProjectID, op.ID, d); err == nil {
		t.Fatal("missing synthetic failure")
	}
	g, err := loadPrivateGateway(s.db, p.Binding.ProjectID)
	g02Check(t, err)
	observed := privateGatewayObservationRequest{RequestID: "observe-unstarted", ExpectedRevision: 1, ReviewDigest: g.Selection.PlanDigest, SourceOperationID: op.ID, SourceDigest: privateGatewaySourceDigest(g)}
	_, _, err = s.acceptPrivateGatewayObservation("observer", p.Binding.ProjectID, observed)
	requireFault(t, err, "project_recovery_required")
}

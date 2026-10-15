package managed

import (
	"context"
	"net"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestPortAdmissionPreservesReplayAndChecksChangesAndStart(t *testing.T) {
	s := testStore(t)
	free := true
	s.portAvailable = func(string, uint16) bool { return free }
	req := registration("alpha", "register", 18080)
	op, _, err := s.register("actor", req)
	if err != nil {
		t.Fatal(err)
	}
	free = false
	if replay, hit, err := s.register("actor", req); err != nil || !hit || replay.ID != op.ID {
		t.Fatal("replay rechecked a now-occupied port", err)
	}
	if _, _, err = s.register("actor", registration("other", "other", 18081)); err == nil {
		t.Fatal("occupied registration accepted")
	} else {
		requireFault(t, err, "port_conflict")
	}
	unchanged := protocol.ConfigureRequest{ExpectedRevision: 1, Config: req.Config}
	if _, err = s.configure("actor", "alpha", unchanged); err != nil {
		t.Fatal("unchanged config probed its owned listener", err)
	}
	unchanged.Config.HTTPPort = 18081
	if _, err = s.configure("actor", "alpha", unchanged); err == nil {
		t.Fatal("occupied changed port accepted")
	}
	if _, _, err = s.accept("actor", "alpha", protocol.OperationRequest{Kind: "start", RequestID: "start", ExpectedRevision: 1}); err == nil {
		t.Fatal("occupied start accepted")
	}
	v, _ := loadInstance(s.db, "alpha")
	if v.Public.ActiveOperationID != "" || v.Runtime.Prepared {
		t.Fatal("refused start changed runtime")
	}
	free = true
	if _, _, err = s.accept("actor", "alpha", protocol.OperationRequest{Kind: "start", RequestID: "start", ExpectedRevision: 1}); err != nil {
		t.Fatal(err)
	}
}

func TestSuggestionChecksActualHostListenerAndDoesNotReserve(t *testing.T) {
	s := testStore(t)
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	occupied := uint16(listener.Addr().(*net.TCPAddr).Port)
	s.portAvailable = func(kind string, port uint16) bool { return port >= occupied && portAvailable(kind, port) }
	port, err := s.suggestHTTPPort(context.Background())
	if err != nil || port <= occupied {
		t.Fatal(port, err)
	}
	listener.Close()
	port, err = s.suggestHTTPPort(context.Background())
	if err != nil || port != occupied {
		t.Fatal(port, err)
	}
	if _, _, err = s.register("actor", registration("reserved", "reserve", port)); err != nil {
		t.Fatal(err)
	}
	if next, err := s.suggestHTTPPort(context.Background()); err != nil || next == port {
		t.Fatal(next, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.suggestHTTPPort(ctx); err == nil {
		t.Fatal("ignored cancelled probe")
	}
}

func TestPortSuggestionRPCIsBoundAndReadOnly(t *testing.T) {
	r, c, _ := testEndpoint(t, privateDir(t), &fakeBackend{})
	if _, err := c.CreateFake(context.Background(), registration("reserved", "reserve", 18080)); err != nil {
		t.Fatal(err)
	}
	p, err := c.SuggestHTTPPort(context.Background())
	if err != nil || p.RunnerID != r.s.runnerID || p.HTTPPort != 18081 || p.ContainerPort != 8080 || p.ObservedAt.IsZero() {
		t.Fatal(p, err)
	}
	s, err := c.Snapshot(context.Background())
	if err != nil || len(s.Instances) != 1 || len(s.Operations) != 1 {
		t.Fatal(s, err)
	}
}

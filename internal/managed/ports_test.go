package managed

import (
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"testing"
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

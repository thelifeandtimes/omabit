package managed

import (
	"fmt"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestRegistrationPathsBeyondFormerLimitKeepConflictChecks(t *testing.T) {
	for _, kind := range []string{"fake", "comet", "qualified-import", "owned", "enroll"} {
		t.Run(kind, func(t *testing.T) {
			s := testStore(t)
			s.mode = DevelopmentMode
			for n := range 16 {
				id := fmt.Sprintf("existing-%d", n)
				if _, _, err := s.register("actor", registration(id, id, uint16(18100+n))); err != nil {
					t.Fatalf("registration %d: %v", n+1, err)
				}
			}
			req := registration("additional", "additional", 18200)
			var err error
			switch kind {
			case "fake":
				_, _, err = s.register("actor", req)
			case "comet":
				_, _, err = s.registerKind("actor", req, nil, true)
			case "qualified-import":
				source := sourceFixture()
				_, _, err = s.registerInstance("actor", req, &source)
			case "owned":
				_, _, err = s.registerShip("actor", protocol.CreateShipRequest{Ship: "~sampel-palnet", Key: "0w" + strings.Repeat("a", 128), RequestID: req.RequestID, Config: req.Config, ConfirmNetwork: true})
			case "enroll":
				identity := pierstore.Identity{Device: 1, Inode: 2, UrbDevice: 1, UrbInode: 3}
				candidate := pierstore.Candidate{Path: "fakeships/additional/pier", Kind: "fake", Identity: identity, Fingerprint: pierstore.Fingerprint(identity), MountIdentity: pierstore.DirectoryIdentity{Device: 1, Inode: 4, Birth: "1:0"}}
				_, err = s.enrollPier("actor", protocol.EnrollPierRequest{ID: req.ID, Path: candidate.Path, Ship: "~zod", Fingerprint: candidate.Fingerprint, HTTPPort: req.Config.HTTPPort, RequestID: req.RequestID, ConfirmStopped: true}, candidate, "enroll-additional")
			}
			if err != nil {
				t.Fatal(err)
			}
			snapshot, err := s.snapshot()
			if err != nil || len(snapshot.Instances) != 17 {
				t.Fatalf("registrations not preserved: %d, %v", len(snapshot.Instances), err)
			}
			_, _, err = s.register("actor", registration("existing-0", "duplicate-id", 18300))
			requireFault(t, err, "instance_exists")
			_, _, err = s.register("actor", registration("duplicate-port", "duplicate-port", 18100))
			requireFault(t, err, "port_conflict")
		})
	}
}

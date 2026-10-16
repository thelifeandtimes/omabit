package managed

import (
	"fmt"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func TestLivenetEnrollmentAllocatesAmesAndPreservesConfiguredPort(t *testing.T) {
	for _, external := range []bool{false, true} {
		t.Run(fmt.Sprintf("external=%t", external), func(t *testing.T) {
			s := testStore(t)
			s.mode = DevelopmentMode
			s.portAvailable = func(kind string, port uint16) bool { return kind != "udp" || port != 34343 }
			for i, ship := range []string{"~dibpen-fogweg-dozzod-sarlev", "~sampel-palnet", "~doznec-salbud-salwex-sogref--tasweb-navfun-batwet-sigfet"} {
				id := fmt.Sprintf("imported-%d", i)
				kind := protocol.ShipClass(ship)
				tier := map[string]string{"moon": "moons", "planet": "planets", "comet": "comets"}[kind]
				source := pierstore.Candidate{Path: "livenet/" + tier + "/" + id + "/pier", Kind: kind}
				req := protocol.EnrollPierRequest{ID: id, Path: source.Path, Ship: ship, HTTPPort: uint16(18080 + i), RequestID: id, ConfirmStopped: true}
				var ext *pierstore.ExternalSource
				if external {
					ext = &pierstore.ExternalSource{Path: "/unused-test-source/" + id}
				}
				op, err := s.enrollPierSource("actor", req, source, id, ext)
				if err != nil {
					t.Fatal(err)
				}
				v, err := loadInstance(s.db, id)
				if err != nil {
					t.Fatal(err)
				}
				want := uint16(34344 + i)
				if v.Public.Config.AmesPort != want || v.Runtime.AmesPort != want || !v.Public.AmesPublished || v.Runtime.Boot == nil {
					t.Fatalf("enrolled livenet pier lacks reserved Ames publication: config=%d runtime=%d published=%t boot=%t; want %d", v.Public.Config.AmesPort, v.Runtime.AmesPort, v.Public.AmesPublished, v.Runtime.Boot != nil, want)
				}
				if v.Runtime.NativeShip != ship || v.Runtime.NativeKeyDigest != "" || !v.Runtime.Adoption.Uncertain {
					t.Fatal("lost identity quarantine or introduced a boot key")
				}
				// Complete only the backend enrollment receipt; no real pier or service is touched.
				_, v, err = s.beginOperation(op.ID)
				if err != nil {
					t.Fatal(err)
				}
				v.Runtime.Initialized = true
				v.Runtime.Adoption.Uncertain = false
				v.Runtime.Adoption.FinishedAt = &op.CreatedAt
				v.Runtime.Comet.StartedAt, v.Runtime.Comet.FinishedAt = &op.CreatedAt, &op.CreatedAt
				if ext != nil {
					v.Runtime.ExternalImport.Stage = "complete"
				}
				if err = s.finish(op, v.Runtime, runtimequalify.ManagedObservation{State: "stopped", Health: "unknown"}, nil, false); err != nil {
					t.Fatal(err)
				}
			}
			v, err := loadInstance(s.db, "imported-0")
			if err != nil {
				t.Fatal(err)
			}
			cfg := protocol.ConfigureRequest{ExpectedRevision: v.Public.ConfigRevision, Config: v.Public.Config}
			cfg.Config.AmesPort = 34345
			_, err = s.configure("actor", "imported-0", cfg)
			requireFault(t, err, "port_conflict")
			cfg.Config.AmesPort = 34343
			_, err = s.configure("actor", "imported-0", cfg)
			requireFault(t, err, "port_conflict")
			cfg.Config.AmesPort = 34567
			updated, err := s.configure("actor", "imported-0", cfg)
			if err != nil {
				t.Fatal(err)
			}
			s.portAvailable = func(kind string, port uint16) bool { return kind != "udp" || port != 34567 }
			_, _, err = s.accept("actor", "imported-0", protocol.OperationRequest{Kind: "start", ExpectedRevision: updated.ConfigRevision, RequestID: "start-imported"})
			requireFault(t, err, "port_conflict")
			s.portAvailable = func(string, uint16) bool { return true }
			op, _, err := s.accept("actor", "imported-0", protocol.OperationRequest{Kind: "start", ExpectedRevision: updated.ConfigRevision, RequestID: "start-imported"})
			if err != nil {
				t.Fatal(err)
			}
			_, v, err = s.beginOperation(op.ID)
			if err != nil {
				t.Fatal(err)
			}
			if v.Runtime.AmesPort != 34567 || !v.Runtime.Initialized || v.Runtime.NativeKeyDigest != "" {
				t.Fatal("start lost configured port or initialized-pier state")
			}
		})
	}
}

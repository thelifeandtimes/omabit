package managed

import (
	"bytes"
	"fmt"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func TestOpeningExistingStatePreservesLegacyAmesAndRecovery(t *testing.T) {
	for _, version := range []int{15, 16} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			seedRecovery(t, s)
			op, _, err := s.register("operator", registration("legacy-live", "legacy-register", 18081))
			if err != nil {
				t.Fatal(err)
			}
			legacy, err := loadInstance(s.db, "legacy-live")
			if err != nil {
				t.Fatal(err)
			}
			legacy.Public.Mode = "planet"
			legacy.Public.Ship = "~sampel-palnet"
			legacy.Public.DesiredState, legacy.Public.ObservedState = "running", "running"
			legacy.Public.Config.AmesPort = 34567
			legacy.Public.ConfigRevision = 4
			legacy.Public.PierPath = "livenet/planets/legacy-live/pier"
			legacy.Runtime.Storage.Path = legacy.Public.PierPath
			legacy.Runtime.Fake, legacy.Runtime.Bootstrap, legacy.Runtime.Boot = nil, nil, nil
			legacy.Runtime.Comet = &runtimequalify.ManagedComet{Ship: legacy.Public.Ship}
			legacy.Runtime.NativeShip = legacy.Public.Ship
			legacy.Runtime.Prepared, legacy.Runtime.Initialized = true, true
			legacy.Runtime.Runtime.Containers = []runtimequalify.ContainerRecord{{ID: "retained-container", Kind: "comet-resume"}}
			saveRecoveryFixture(t, s, legacy)
			before := map[string][]byte{}
			for _, id := range []string{"one", "legacy-live"} {
				var body []byte
				if err = s.db.QueryRow("SELECT body FROM instances WHERE id=?", id).Scan(&body); err != nil {
					t.Fatal(err)
				}
				before[id] = body
			}
			var operation, fingerprint []byte
			if err = s.db.QueryRow("SELECT body,fingerprint FROM operations WHERE id=?", op.ID).Scan(&operation, &fingerprint); err != nil {
				t.Fatal(err)
			}
			runnerID := s.runnerID
			if _, err = s.db.Exec(fmt.Sprintf("PRAGMA user_version=%d", version)); err != nil {
				t.Fatal(err)
			}
			if err = s.close(); err != nil {
				t.Fatal(err)
			}
			for pass := 0; pass < 2; pass++ {
				s, err = testOpenStore(dir)
				if err != nil {
					t.Fatal(err)
				}
				if err = s.reconcileInterrupted(); err != nil {
					t.Fatal(err)
				}
				if s.runnerID != runnerID {
					t.Fatal("upgrade replaced runner identity")
				}
				for id, body := range before {
					var after []byte
					if err = s.db.QueryRow("SELECT body FROM instances WHERE id=?", id).Scan(&after); err != nil || !bytes.Equal(body, after) {
						t.Fatal("opening state rewrote runtime/enrollment", id, err)
					}
				}
				var after []byte
				if err = s.db.QueryRow("SELECT body FROM operations WHERE id=?", op.ID).Scan(&after); err != nil || !bytes.Equal(after, operation) {
					t.Fatal("historical receipt changed", err)
				}
				got, hit, err := replay(s.db, "operator", "legacy-register", string(fingerprint))
				if err != nil || !hit || got.ID != op.ID {
					t.Fatal("replay changed", got, err)
				}
				v, err := loadInstance(s.db, "legacy-live")
				if err != nil || v.Runtime.Boot != nil || v.Runtime.AmesPort != 0 || v.Public.AmesPublished || v.Public.Config.AmesPort != 34567 || v.Public.DesiredState != "running" {
					t.Fatal("legacy network/intent changed", v.Public, err)
				}
				v, err = loadInstance(s.db, "one")
				if err != nil || !v.Runtime.ImportRecoveryEligible() || !v.Public.RecoveryRequired {
					t.Fatal("recovery fence lost", err)
				}
				_, _, err = s.accept("operator", "one", protocol.OperationRequest{Kind: "start", ExpectedRevision: 1, RequestID: "must-remain-blocked"})
				requireFault(t, err, "recovery_required")
				if err = s.close(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

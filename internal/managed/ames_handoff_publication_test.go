package managed

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type amesHandoffBackend struct {
	handoffBackend
	checks int
}

func (b *amesHandoffBackend) ValidateAmesPublication(_ context.Context, s runtimequalify.ManagedState) error {
	b.checks++
	if s.NativeShip != "" || s.Comet == nil || s.Handoff == nil || !s.Handoff.ValidCommitment(s.Handoff.Commitment) {
		return fmt.Errorf("validation did not receive the original completed comet contract")
	}
	return nil
}

func TestAmesPublicationReservedLegacyCometHandoffAfterActivationAndReopen(t *testing.T) {
	s := testStore(t)
	s.mode = DevelopmentMode
	actor, _, _ := receiptActors(t, s.directory)
	b := &amesHandoffBackend{handoffBackend: handoffBackend{dir: s.directory}}
	r := &Server{s: s, b: b, uid: uint32(os.Geteuid()), keys: filepath.Join(s.directory, "keys")}
	offer, key := receiptOffer(t, r, 0)
	offer.Mode, offer.Ship = "comet", "~doznec-salbud-salwex-sogref--tasweb-navfun-batwet-sigfet"
	offer.RuntimeSettings = &protocol.RuntimeSettings{Loom: 29, NativeControl: true}
	receiptCall(t, r, actor, "/v1/handoffs", protocol.ReceiveHandoffRequest{Offer: offer, HTTPPort: 18080, RequestID: "reserve-comet"}, 201)
	v := receiptLoad(t, r, offer.DestinationInstance)
	if v.Runtime.NativeShip != "" || v.Runtime.Boot != nil || v.Runtime.Comet.Ship != offer.Ship {
		t.Fatal("actual handoff reservation no longer represents the legacy contract")
	}
	// Model the receive driver's completion callback at the journal boundary;
	// the paired runtime regression executes extraction and the real driver.
	now := time.Now().UTC()
	v.Runtime.Prepared, v.Runtime.Initialized = true, true
	v.Runtime.Storage.Identity = pierstore.Identity{Device: 1, Inode: 2, UrbInode: 3}
	v.Runtime.Storage.MountIdentity = pierstore.DirectoryIdentity{Device: 1, Inode: 4}
	v.Runtime.Adoption = &runtimequalify.ManagedAdoption{Fingerprint: pierstore.Fingerprint(v.Runtime.Storage.Identity), FinishedAt: &now}
	v.Runtime.Comet.StartedAt, v.Runtime.Comet.FinishedAt = &now, &now
	v.Runtime.Runtime.RunID = v.Runtime.ID
	v.Runtime.Handoff.State = "received"
	receiptSave(t, r, v)
	proof := fmt.Sprintf("%x", ed25519.Sign(key, offer.CommitmentBytes()))
	receiptCall(t, r, actor, "/v1/handoffs/"+offer.ID+"/activate", protocol.FinishHandoffRequest{RequestID: "activate-comet", ExpectedRevision: 1, SHA256: offer.SHA256, Commitment: proof}, 200)
	r.s = rc11ReopenStore(t, r.s)
	before := receiptLoad(t, r, offer.DestinationInstance)
	if before.Public.DesiredState != "stopped" || before.Public.ObservedState != "stopped" || before.Runtime.NativeShip != "" {
		t.Fatal("activation or reopen changed stopped legacy intent")
	}
	path := "/v1/instances/" + offer.DestinationInstance + "/publish-ames"
	first := receiptCall(t, r, actor, path, amesRequest(), 201)
	after := receiptLoad(t, r, offer.DestinationInstance)
	if after.Runtime.NativeShip != offer.Ship || after.Runtime.Boot == nil || !after.Public.AmesPublished || after.Public.ConfigRevision != 2 || after.Public.Mode != "comet" || after.Public.Ship != offer.Ship || after.Public.DesiredState != "stopped" || after.Public.ObservedState != "stopped" || after.Public.Config.Loom != 29 || !after.Public.Config.NativeControl || !reflect.DeepEqual(before.Runtime.Storage, after.Runtime.Storage) || !reflect.DeepEqual(before.Runtime.Handoff, after.Runtime.Handoff) || !after.Runtime.Handoff.ValidCommitment(proof) {
		t.Fatal("publication did not preserve the activated comet handoff")
	}
	r.s = rc11ReopenStore(t, r.s)
	second := receiptCall(t, r, actor, path, amesRequest(), 200)
	if first.Body.String() != second.Body.String() || b.checks != 1 {
		t.Fatal("replay changed receipt or revalidated migrated state")
	}
}

func TestAmesPublicationNormalizedCometStillRequiresMatchingPublicIdentity(t *testing.T) {
	for _, field := range []string{"mode", "ship"} {
		t.Run(field, func(t *testing.T) {
			s := testStore(t)
			v := seedLegacyAmes(t, s, "second", 18081)
			v.Runtime.NativeShip = "" // valid comet semantics without the optional native pin
			if field == "mode" {
				v.Public.Mode = "fake"
			} else {
				v.Public.Ship = "~sampel-palnet"
			}
			saveRecoveryFixture(t, s, v)
			_, _, err := s.publishAmes(context.Background(), "actor", v.Public.ID, amesRequest(), func(context.Context, runtimequalify.ManagedState) error {
				t.Fatal("public identity mismatch reached runtime validation")
				return nil
			})
			requireFault(t, err, "invalid_state")
			after, err := loadInstance(s.db, v.Public.ID)
			if err != nil || !reflect.DeepEqual(v, after) {
				t.Fatal("public identity refusal modified state", err)
			}
		})
	}
}

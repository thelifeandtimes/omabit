//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func receivedLegacyCometHandoff(t *testing.T) (*fakeEngine, ManagedState, func(ManagedState) error) {
	t.Helper()
	f, state, persist := rc11Incoming(t)
	// Match the historical reservation shape, and the fake engine's expected
	// namespace, to the actual comet receive driver (not a renamed fake runtime).
	f.d.profile = ProfileComet
	f.d.hostIdentity = func(context.Context, string) (string, error) { return fixtureComet, nil }
	state.NativeShip, state.Fake = "", nil
	state.Comet = &ManagedComet{Ship: fixtureComet}
	state.Storage.Path = "livenet/comets/fixture/pier"
	state.Loom, state.NativeControl = 29, true
	state.Handoff.Ship, state.Handoff.Mode = fixtureComet, "comet"
	state.Handoff.RuntimeSettings = &protocol.RuntimeSettings{Loom: 29, NativeControl: true}
	key := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{7}, ed25519.SeedSize))
	state.Handoff.VerifyKey = fmt.Sprintf("%x", key.Public())
	var err error
	state, _, err = applyManaged(context.Background(), "receive-handoff", state, persist, f.d)
	if err != nil || state.Handoff.State != "received" || !state.Initialized || state.NativeShip != "" || state.Adoption == nil {
		t.Fatal("actual receive driver did not produce the legacy comet shape", err)
	}
	// Activation's signed commitment is also exercised through the managed API
	// regression. Here retain the exact source-bound proof in the driver state.
	state.Handoff.Commitment = fmt.Sprintf("%x", ed25519.Sign(key, state.Handoff.CommitmentBytes()))
	state.Handoff.State = "complete"
	return f, rc11Reload(t, rc11Save(t, state)), persist
}

func TestAmesPublicationReceivedLegacyCometHandoffResumesSamePier(t *testing.T) {
	f, state, persist := receivedLegacyCometHandoff(t)
	if state.UncertainProvisioning() || state.Boot != nil || state.NativeShip != "" || !state.Handoff.ValidCommitment(state.Handoff.Commitment) {
		t.Fatal("fixture is not a completed unpublished legacy comet handoff")
	}
	before := rc11Save(t, state)
	if err := validateAmesPublication(context.Background(), state, f.d); err != nil {
		t.Fatal("valid received comet rejected", err)
	}
	proposed, err := AmesPublicationState(state, 34567)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, rc11Save(t, state)) || proposed.NativeShip != fixtureComet || proposed.Boot == nil || proposed.Boot.Pill != nil || proposed.Bootstrap != nil || proposed.NativeKeyDigest != "" || !proposed.Initialized || proposed.Loom != 29 || !proposed.NativeControl || !reflect.DeepEqual(state.Storage, proposed.Storage) || !reflect.DeepEqual(state.Comet, proposed.Comet) || !reflect.DeepEqual(state.Adoption, proposed.Adoption) || !reflect.DeepEqual(state.Handoff, proposed.Handoff) {
		t.Fatal("publication changed existing comet identity, settings or transfer provenance")
	}
	proposed = rc11Reload(t, rc11Save(t, proposed))
	if _, err := managedSession(&proposed, persist, f.d); err != nil {
		t.Fatal("normalized publication contract is invalid", err)
	}
	calls := len(f.calls)
	proposed, obs, err := applyManaged(context.Background(), "start", proposed, persist, f.d)
	if err != nil || obs.Health != "healthy" || proposed.Comet.Ship != fixtureComet || !proposed.Adoption.Verified || f.downloads != 0 || !reflect.DeepEqual(proposed.Storage, state.Storage) || !proposed.Handoff.ValidCommitment(proposed.Handoff.Commitment) {
		t.Fatal("published handoff failed to resume the original pier", obs, err)
	}
	verified, resumed := false, false
	for _, call := range f.calls[calls:] {
		if len(call) < 2 || call[0] != "container" || call[1] != "create" {
			continue
		}
		joined := " " + strings.Join(call, " ") + " "
		for _, boot := range []string{"comet-boot", " -B ", " -F ", " -w ", " -k "} {
			if strings.Contains(joined, boot) {
				t.Fatal("existing handoff was bootstrapped", call)
			}
		}
		if strings.Contains(joined, "comet-verify") {
			verified = true
			if strings.Contains(joined, "/udp") || !strings.Contains(joined, " -L ") {
				t.Fatal("offline identity verification published Ames", call)
			}
		}
		if strings.Contains(joined, "comet-resume") {
			resumed = true
			if !verified || !strings.Contains(joined, "0.0.0.0:34567:34567/udp") || !strings.Contains(joined, " --loom 29 ") || !strings.Contains(joined, "umask 077;") || strings.Count(joined, " -c ") != 1 {
				t.Fatal("published resume lost settings or verification order", call)
			}
		}
	}
	if !verified || !resumed {
		t.Fatal("missing verification or published resume")
	}
	if _, err := pierstore.Check(f.d.home, *proposed.Storage); err != nil {
		t.Fatal("published runtime no longer owns the received pier", err)
	}
}

func TestAmesPublicationCometNormalizationPreservesIdentityFences(t *testing.T) {
	for _, fault := range []string{"non-comet-without-native-pin", "different-native-pin", "fake-storage", "uncertain-comet", "missing-adoption", "uncommitted-handoff", "foreign-handoff-ship", "foreign-handoff-mode", "published-handoff"} {
		t.Run(fault, func(t *testing.T) {
			_, state, _ := receivedLegacyCometHandoff(t)
			switch fault {
			case "non-comet-without-native-pin":
				state.Comet.Ship = "~sampel-palnet"
				state.Storage.Path = "livenet/planets/fixture/pier"
			case "different-native-pin":
				state.NativeShip = "~sampel-palnet"
			case "fake-storage":
				state.Storage.Path = "fakeships/fixture/pier"
			case "uncertain-comet":
				state.Comet.Uncertain = true
			case "missing-adoption":
				state.Adoption = nil
			case "uncommitted-handoff":
				state.Handoff.State = "received"
			case "foreign-handoff-ship":
				state.Handoff.Ship = "~sampel-palnet"
			case "foreign-handoff-mode":
				state.Handoff.Mode = "fake"
			case "published-handoff":
				state.Handoff.AmesPublished = true
			}
			before := rc11Save(t, state)
			if _, err := AmesPublicationState(state, 34567); err == nil {
				t.Fatal("unproven live identity normalized into publication")
			}
			if !bytes.Equal(before, rc11Save(t, state)) || state.Boot != nil {
				t.Fatal("refusal modified private state")
			}
		})
	}
}

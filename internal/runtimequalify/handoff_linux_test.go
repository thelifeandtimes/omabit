//go:build linux && amd64

package runtimequalify

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestStoppedHandoffPreservesDataFencesSourceAndResumesDestination(t *testing.T) {
	for _, comet := range []bool{false, true} {
		name := "fake"
		if comet {
			name = "comet"
		}
		t.Run(name, func(t *testing.T) { exerciseStoppedHandoff(t, comet) })
	}
}

func exerciseStoppedHandoff(t *testing.T, comet bool) {
	ctx := context.Background()
	fixture := managedFixture
	if comet {
		fixture = managedCometFixture
	}
	s, source, persist := fixture(t)
	s.d.admission = func(string) (uint64, uint64, error) { return 8 << 30, 32 << 30, nil }
	var err error
	source, _, err = applyManaged(ctx, "start", source, persist, s.d)
	if err != nil {
		t.Fatal(err)
	}
	source, _, err = applyManaged(ctx, "stop", source, persist, s.d)
	if err != nil {
		t.Fatal(err)
	}
	writePierEpochPath(t, managedPier(s, source), "0i0", false)
	if err = os.MkdirAll(filepath.Join(managedPier(s, source), ".urb/chk"), 0700); err != nil {
		t.Fatal(err)
	}
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ship, mode := "~zod", "fake"
	if comet {
		ship, mode = source.Comet.Ship, "comet"
	}
	source.Handoff = &protocol.Handoff{ID: strings.Repeat("b", 32), Direction: "out", State: "preparing", SourceRunner: "source", SourceInstance: "original", DestinationRunner: "destination", DestinationInstance: "received", Ship: ship, Mode: mode, VerifyKey: fmt.Sprintf("%x", public)}
	source.HandoffSigningKey = private
	source, _, err = applyManaged(ctx, "prepare-handoff", source, persist, s.d)
	if err != nil {
		t.Fatal(err)
	}
	// Retry a lost preparation outcome against the same immutable artifact.
	source.Handoff.State = "preparing"
	source, _, err = applyManaged(ctx, "prepare-handoff", source, persist, s.d)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = applyManaged(ctx, "start", source, persist, s.d); err == nil {
		t.Fatal("fenced source started")
	}
	d, dest, persistDest := fixture(t)
	d.d.admission = s.d.admission
	d.d.hostIdentity = func(context.Context, string) (string, error) { return ship, nil }
	if comet {
		dest.Comet.Ship = ship
	} else {
		dest.Fake = &ManagedFake{Ship: ship}
	}
	h := *source.Handoff
	h.Direction = "in"
	h.State = "receiving"
	dest.Handoff = &h
	artifact := HandoffArchive(filepath.Join(d.d.home, ".local/share/urbit-runner/managed"), &h)
	if err = os.Mkdir(filepath.Dir(artifact), 0700); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(HandoffArchive(filepath.Join(s.d.home, ".local/share/urbit-runner/managed"), source.Handoff))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(artifact, data, 0400); err != nil {
		t.Fatal(err)
	}
	// Simulate a journal failure after the incoming mount is durably identified.
	failed := false
	dest, _, err = applyManaged(ctx, "receive-handoff", dest, func(v ManagedState) error {
		if v.Storage.MountIdentity.Inode != 0 && !failed {
			failed = true
			return errors.New("interrupted")
		}
		return persistDest(v)
	}, d.d)
	if err == nil || !failed {
		t.Fatal("missing fault injection")
	}
	dest, _, err = applyManaged(ctx, "receive-handoff", dest, persistDest, d.d)
	if err != nil {
		t.Fatal(err)
	}
	if !dest.Initialized || dest.Handoff.State != "received" || dest.Epochs == nil {
		t.Fatal("destination not verified")
	}
	if _, _, err = applyManaged(ctx, "start", dest, persistDest, d.d); err == nil {
		t.Fatal("uncommitted destination started")
	}
	changedSource := filepath.Join(managedPier(s, source), "unexpected-after-freeze")
	if err = os.WriteFile(changedSource, []byte("advanced"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err = applyManaged(ctx, "commit-handoff", source, persist, s.d); err == nil {
		t.Fatal("committed a changed frozen source")
	}
	if err = os.Remove(changedSource); err != nil {
		t.Fatal(err)
	}
	// A failed final journal write leaves the source physically archived. Reconcile
	// using its exact inode/birth identities, never by copying old data back.
	source, _, err = applyManaged(ctx, "commit-handoff", source, func(ManagedState) error { return errors.New("lost commit") }, s.d)
	if err == nil {
		t.Fatal("missing commit failure")
	}
	source.Handoff.State = "ready"
	source.Handoff.Commitment = ""
	source, _, err = applyManaged(ctx, "commit-handoff", source, persist, s.d)
	if err != nil {
		t.Fatal(err)
	}
	if !h.ValidCommitment(source.Handoff.Commitment) {
		t.Fatal("commitment does not bind transferred bytes")
	}
	changed := h
	changed.DestinationInstance = "another"
	if changed.ValidCommitment(source.Handoff.Commitment) {
		t.Fatal("commitment reusable at another destination")
	}
	if _, err = os.Stat(managedPier(s, source)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("source still in active tree")
	}
	if _, _, err = applyManaged(ctx, "start", source, persist, s.d); err == nil {
		t.Fatal("moved source started")
	}
	dest.Handoff.State = "complete"
	dest.Handoff.Commitment = source.Handoff.Commitment
	identity := dest.Storage.Identity
	dest, obs, err := applyManaged(ctx, "start", dest, persistDest, d.d)
	if err != nil || obs.Health != "healthy" || dest.Storage.Identity != identity {
		t.Fatal(obs, err)
	}
	if d.downloads != 0 {
		t.Fatal("transferred pier re-bootstrapped")
	}
	if _, err = pierstore.Check(d.d.home, *dest.Storage); err != nil {
		t.Fatal(err)
	}
}

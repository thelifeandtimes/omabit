package managed

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type guidedFixture struct {
	storageActionBackend
	identity protocol.PierIdentity
	occupied string
}

func (b *guidedFixture) InspectPierIdentity(context.Context, string) (protocol.PierIdentity, error) {
	return b.identity, nil
}
func (b *guidedFixture) ImportDestinationAvailable(path, source string) (bool, error) {
	return path != b.occupied, nil
}

func TestGuidedImportDerivesPlanAndRefusesChangedIdentity(t *testing.T) {
	b := &guidedFixture{storageActionBackend: storageActionBackend{home: privateDir(t)}, identity: protocol.PierIdentity{Ship: "~zod", Network: "fakenet", Kind: "fake"}, occupied: "fakeships/zod/zod"}
	if err := pierstore.Init(b.home); err != nil {
		t.Fatal(err)
	}
	r, c, _ := testEndpoint(t, privateDir(t), b)
	ctx := context.Background()
	source := filepath.Join(privateDir(t), "arbitrary-folder-name")
	for _, dir := range []string{".urb/chk", ".urb/log"} {
		if err := os.MkdirAll(filepath.Join(source, dir), 0700); err != nil {
			t.Fatal(err)
		}
	}
	preview, err := c.PreviewGuidedPierImport(ctx, r.s.runnerID, source, "default")
	if err != nil || preview.Ship != "~zod" || preview.InstanceID != "zod-2" || preview.Destination != "fakeships/zod-2/zod" || preview.Network != "fakenet" || preview.HTTPPort < 1024 {
		t.Fatal(preview, err)
	}
	if _, err := c.PreviewGuidedPierImport(ctx, r.s.runnerID, source, "unknown"); err == nil {
		t.Fatal("unknown location accepted")
	}
	request := protocol.ImportPierRequest{Source: source, Identify: true, EnrollPierRequest: protocol.EnrollPierRequest{ID: preview.InstanceID, Ship: preview.Ship, Path: preview.Destination, HTTPPort: preview.HTTPPort, Fingerprint: preview.Source.Fingerprint, ConfirmStopped: true, RequestID: "guided"}}
	b.identity = protocol.PierIdentity{Ship: "~nec", Network: "fakenet", Kind: "fake"}
	if _, err := c.ImportPier(ctx, request); err == nil {
		t.Fatal("changed identity admitted")
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatal("source changed before authorization", err)
	}
	b.identity = protocol.PierIdentity{Ship: "~zod", Network: "livenet", Kind: "galaxy"}
	if _, err := c.ImportPier(ctx, request); err == nil {
		t.Fatal("fake/live change admitted")
	}
	b.identity = protocol.PierIdentity{Ship: "~zod", Network: "fakenet", Kind: "fake"}
	op, err := c.ImportPier(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	waitOperation(t, c, op.ID, "succeeded")
	again, err := c.ImportPier(ctx, request)
	if err != nil || again.ID != op.ID {
		t.Fatal("guided request replay changed", again, err)
	}
	v, err := loadInstance(r.s.db, preview.InstanceID)
	if err != nil || !v.Runtime.ExternalImport.Identify {
		t.Fatal("worker lost identity revalidation", err)
	}
}

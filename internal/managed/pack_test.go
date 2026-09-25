package managed

import (
	"context"
	"errors"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestPackRequiresStoppedPierAndReplaysDurably(t *testing.T) {
	b := &fakeBackend{}
	r, c, closeServer := testEndpoint(t, privateDir(t), b)
	defer closeServer()
	ctx := context.Background()
	if _, err := c.CreateFake(ctx, registration("alpha", "register", 18080)); err != nil {
		t.Fatal(err)
	}
	pack := protocol.OperationRequest{RequestID: "pack", Kind: "pack", ExpectedRevision: 1}
	if _, err := c.Operate(ctx, "alpha", pack); err == nil {
		t.Fatal("uninitialized pier admitted for pack")
	}
	start, err := c.Operate(ctx, "alpha", protocol.OperationRequest{RequestID: "start", Kind: "start", ExpectedRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	waitOperation(t, c, start.ID, "succeeded")
	if _, err := c.Operate(ctx, "alpha", pack); err == nil {
		t.Fatal("running pier admitted for pack")
	}
	stop, err := c.Operate(ctx, "alpha", protocol.OperationRequest{RequestID: "stop", Kind: "stop", ExpectedRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	waitOperation(t, c, stop.ID, "succeeded")
	op, err := c.Operate(ctx, "alpha", pack)
	if err != nil {
		t.Fatal(err)
	}
	waitOperation(t, c, op.ID, "succeeded")
	replayed, err := c.Operate(ctx, "alpha", pack)
	if err != nil || replayed.ID != op.ID {
		t.Fatal("pack request replay changed operation")
	}
	// An unresolved maintenance flag is authoritative even if an observation
	// was previously stopped/healthy enough for ordinary admission.
	tx, err := r.s.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	v, err := loadInstance(tx, "alpha")
	if err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	v.Runtime.MaintenanceUncertain = true
	v.RuntimeRevision++
	if err := saveInstance(tx, v); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	_, err = c.Operate(ctx, "alpha", protocol.OperationRequest{RequestID: "blocked-resume", Kind: "start", ExpectedRevision: 1})
	var remote *client.RemoteError
	if !errors.As(err, &remote) || remote.Problem.Code != "recovery_required" {
		t.Fatalf("uncertainty gate: %v", err)
	}
}

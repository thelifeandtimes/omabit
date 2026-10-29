package applicationauth

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestAcknowledgementCommitFailureNeverReportsConnected(t *testing.T) {
	f, r := setup(t), newReceiver()
	a := f.approve(t, r)
	journal := filepath.Join(f.dir, "applications.json")
	r.deliver = func(_ context.Context, data []byte) ([]byte, error) {
		if err := os.Chmod(journal, 0644); err != nil {
			t.Fatal(err)
		}
		return ack(data), nil
	}
	got, err := f.e.Connect(context.Background(), a.ID, r)
	if err == nil || got.Phase != "delivering" {
		t.Fatal("accepted before durable commit", got, err)
	}
	if err := os.Chmod(journal, 0600); err != nil {
		t.Fatal(err)
	}
	f.reopen(t)
	got, err = f.e.Recover(context.Background(), a.ID)
	if err != nil || got.Phase != "closed" || !got.ReceiverReconciliation {
		t.Fatal(got, err)
	}
	if _, _, active := f.counts(); active != 1 {
		t.Fatal("session escaped cleanup")
	}
}

func TestArtifactChangeAfterIssuanceCleansUpWithoutDelivery(t *testing.T) {
	f, r := setup(t), newReceiver()
	a := f.approve(t, r)
	checks := 0
	r.verify = func(context.Context) error {
		checks++
		if checks == 2 {
			return errors.New("changed artifact")
		}
		return nil
	}
	got, err := f.e.Connect(context.Background(), a.ID, r)
	if !errors.Is(err, ErrDelivery) || got.Phase != "closed" || got.ReceiverReconciliation || r.calls != 0 {
		t.Fatal(got, err)
	}
	if _, _, active := f.counts(); active != 1 {
		t.Fatal("session escaped cleanup")
	}
}

func TestMissingConnectedCredentialIsNotConfirmedRevocation(t *testing.T) {
	ctx := context.Background()
	f, r := setup(t), newReceiver()
	a := f.approve(t, r)
	if _, err := f.e.Connect(ctx, a.ID, r); err != nil {
		t.Fatal(err)
	}
	if err := f.store.Delete(ctx, credentialKey(a)); err != nil {
		t.Fatal(err)
	}
	f.reopen(t)
	got, err := f.e.Disconnect(ctx, a.ID)
	if !errors.Is(err, ErrCleanup) || got.Phase != "unresolved" || got.Outcome != "" {
		t.Fatal(got, err)
	}
	if _, out, active := f.counts(); out != 0 || active != 2 {
		t.Fatal("missing token invented revocation", out, active)
	}
}

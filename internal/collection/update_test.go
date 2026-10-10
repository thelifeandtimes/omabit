package collection

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
)

func TestConnectionRebindPreservesIdentityAndRejectsReplacement(t *testing.T) {
	r := registry(t)
	ctx := context.Background()
	old, err := r.Enroll(ctx, "home", fixture(t, "runner-home", nil))
	if err != nil {
		t.Fatal(err)
	}
	other := fixture(t, "runner-other", nil)
	if _, err := r.Update(ctx, old.Name, old.RunnerID, old.Revision, Connection{Name: "renamed", Transport: "unix", Socket: other}); !errors.Is(err, client.ErrIdentityMismatch) {
		t.Fatalf("replacement: %v", err)
	}
	got, _ := r.Get(ctx, "home")
	if got != old {
		t.Fatal("failed verification changed enrollment")
	}
	endpoint := fixture(t, "runner-home", nil)
	next, err := r.Update(ctx, "home", old.RunnerID, 1, Connection{Name: "renamed", Transport: "unix", Socket: endpoint})
	if err != nil || next.Revision != 2 || next.RunnerID != old.RunnerID || next.Mode != old.Mode || !next.EnrolledAt.Equal(old.EnrolledAt) {
		t.Fatalf("update: %+v %v", next, err)
	}
	if _, err := r.Get(ctx, "home"); err == nil {
		t.Fatal("old name retained")
	}
	if _, err := r.Update(ctx, "renamed", next.RunnerID, 1, next); err == nil {
		t.Fatal("stale update accepted")
	}
	noop, err := r.Update(ctx, "renamed", next.RunnerID, 2, next)
	if err != nil || noop != next {
		t.Fatal("no-op churned revision")
	}
	if _, err := r.Enroll(ctx, "occupied", other); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Update(ctx, "renamed", next.RunnerID, 2, Connection{Name: "occupied", Transport: "unix", Socket: endpoint}); err == nil {
		t.Fatal("name collision accepted")
	}
}

func TestConcurrentConnectionUpdatesHaveSingleWinner(t *testing.T) {
	r := registry(t)
	ctx := context.Background()
	socket := fixture(t, "runner-home", nil)
	old, err := r.Enroll(ctx, "home", socket)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var winners atomic.Int32
	for _, name := range []string{"first", "second"} {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if _, err := r.Update(ctx, "home", old.RunnerID, 1, Connection{Name: name, Transport: "unix", Socket: socket}); err == nil {
				winners.Add(1)
			}
		}(name)
	}
	wg.Wait()
	all, err := r.List(ctx)
	if err != nil || winners.Load() != 1 || len(all) != 1 || all[0].Revision != 2 {
		t.Fatalf("lost update: %v %d %v", all, winners.Load(), err)
	}
}

func TestLegacyConnectionRevisionIsReadWithoutRewriting(t *testing.T) {
	r := registry(t)
	ctx := context.Background()
	if _, err := r.Enroll(ctx, "home", fixture(t, "runner-home", nil)); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(r.dir, "runners.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	legacy := strings.Replace(string(raw), ",\n      \"revision\": 1", "", 1)
	if legacy == string(raw) {
		t.Fatal("fixture did not remove revision")
	}
	if err := os.WriteFile(path, []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := r.Get(ctx, "home")
	if err != nil || c.Revision != 1 {
		t.Fatal("legacy revision not normalized", err)
	}
	after, _ := os.ReadFile(path)
	if string(after) != legacy {
		t.Fatal("read mutated legacy registry")
	}
}

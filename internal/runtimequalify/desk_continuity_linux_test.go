//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDeskSourceIdentityTracksPierObjectAcrossRestartAndMove(t *testing.T) {
	parent := t.TempDir()
	pier := filepath.Join(parent, "pier")
	if err := os.MkdirAll(filepath.Join(pier, ".urb"), 0700); err != nil {
		t.Fatal(err)
	}
	checks := 0
	s := &DeskRuntime{pier: pier, check: func(context.Context) error { checks++; return nil }}
	first, err := s.SourceIdentity(context.Background())
	if err != nil || len(first) != 64 || checks != 2 {
		t.Fatal(first, checks, err)
	}
	moved := filepath.Join(parent, "moved")
	if err = os.Rename(pier, moved); err != nil {
		t.Fatal(err)
	}
	s.pier = moved
	s.containerID = "a different selected runtime"
	after, err := s.SourceIdentity(context.Background())
	if err != nil || after != first {
		t.Fatal("same pier lost continuity", after, err)
	}
	if err = os.Rename(filepath.Join(moved, ".urb"), filepath.Join(moved, ".urb-retained")); err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(filepath.Join(moved, ".urb"), 0700); err != nil {
		t.Fatal(err)
	}
	replaced, err := s.SourceIdentity(context.Background())
	if err != nil || replaced == first {
		t.Fatal("replacement inherited source ownership", replaced, err)
	}
	s.check = func(context.Context) error { return errors.New("runtime changed") }
	if _, err = s.SourceIdentity(context.Background()); err == nil {
		t.Fatal("unverified runtime accepted")
	}
}
func TestDeskRuntimeSelectsConfiguredShipIdentity(t *testing.T) {
	for _, tc := range []struct {
		state ManagedState
		ship  string
	}{
		{ManagedState{}, "~zod"},
		{ManagedState{NativeShip: "~bus"}, "~bus"},
		{ManagedState{Fake: &ManagedFake{Ship: "~nec"}}, "~nec"},
		{ManagedState{Comet: &ManagedComet{Ship: "~marzod-marzod-marzod-marzod"}}, "~marzod-marzod-marzod-marzod"},
	} {
		if got := managedDeskShip(tc.state); got != tc.ship {
			t.Fatal("wrong selected ship", got, tc.ship)
		}
	}
}

//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestPortAdmissionBeforePreparationAndBeforeDispatch(t *testing.T) {
	for _, race := range []bool{false, true} {
		f, s, persist := managedFixture(t)
		calls := 0
		f.d.portAvailable = func(string, uint16) bool { calls++; return race && calls == 1 }
		s, _, err := applyManaged(context.Background(), "start", s, persist, f.d)
		if err == nil || !strings.Contains(err.Error(), "already in use") {
			t.Fatal(err)
		}
		for _, event := range f.events {
			if event == "start-http-boot" {
				t.Fatal("port conflict launched Vere")
			}
		}
		if !race && (s.Prepared || len(f.calls) != 0) {
			t.Fatal("admission mutated before refusal")
		}
	}
}

func TestDockerLaunchReasonIsUsefulAndDoesNotReflectSecrets(t *testing.T) {
	f, s, persist := managedFixture(t)
	execute := f.d.execute
	f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
		if len(args) > 6 && args[4] == "container" && args[5] == "start" {
			for _, c := range f.containers {
				if c.id == args[6] && c.kind == "http-boot" {
					mapAt(c.doc, "State")["Error"] = "RootlessKit: bind: address already in use PRIVATE-SECRET 0wabc"
					mapAt(c.doc, "State")["ExitCode"] = 128
					return "", errors.New("PRIVATE-STDERR")
				}
			}
		}
		return execute(ctx, args, env)
	}
	s, _, err := applyManaged(context.Background(), "start", s, persist, f.d)
	if err == nil || !strings.Contains(err.Error(), "Host port") || strings.Contains(err.Error(), "PRIVATE") || strings.Contains(err.Error(), "0wabc") {
		t.Fatal(err)
	}
	last := s.Runtime.Containers[len(s.Runtime.Containers)-1].LastState
	if last == nil || !strings.Contains(last.LaunchError, "Host port") {
		t.Fatal("launch evidence missing")
	}
}

package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRecordingPilotRequiresExplicitDevelopmentModeAndAcknowledgment(t *testing.T) {
	for _, tc := range []struct {
		comets bool
		args   []string
		want   int
	}{
		{true, []string{"--help"}, 0},
		{false, []string{"--acknowledge-development", "--record-runtime"}, 2},
		{true, []string{"--record-runtime"}, 2},
	} {
		var out, diagnostic bytes.Buffer
		code := runManagedMode(context.Background(), tc.args, &out, &diagnostic, tc.comets)
		if code != tc.want {
			t.Fatalf("unexpected opt-in admission: %v: %d %s", tc.args, code, &diagnostic)
		}
		if tc.want == 0 && (!strings.Contains(out.String(), "sixteen 64 MiB") || !strings.Contains(out.String(), "no automatic pruning")) {
			t.Fatal("pilot limits absent from explicit choice")
		}
	}
}

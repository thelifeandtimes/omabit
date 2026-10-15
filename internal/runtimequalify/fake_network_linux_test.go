//go:build linux && amd64

package runtimequalify

import (
	"context"
	"strings"
	"testing"
)

func TestFakeCannotStartWithoutLocalAmesRestriction(t *testing.T) {
	f, state, persist := managedFixture(t)
	f.editInspect = func(kind string, doc map[string]any) {
		if !isHTTPKind(kind) {
			return
		}
		config := mapAt(doc, "Config")
		args := config["Cmd"].([]any)
		filtered := []any{}
		for _, arg := range args {
			if arg != "-L" {
				filtered = append(filtered, arg)
			}
		}
		config["Cmd"] = filtered
	}
	state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err == nil || state.Initialized {
		t.Fatal("fake without -L accepted")
	}
	for _, event := range f.events {
		if strings.HasPrefix(event, "start-http-") {
			t.Fatal("unsafe fake was started", event)
		}
	}
}

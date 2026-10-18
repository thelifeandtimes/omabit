package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRuntimeSettingsManagedHelp(t *testing.T) {
	for _, development := range []bool{false, true} {
		var out, errs bytes.Buffer
		if code := runManagedMode(context.Background(), []string{"--help"}, &out, &errs, development); code != 0 {
			t.Fatal(code, errs.String())
		}
		if development && strings.Contains(out.String(), "fake-only native control") {
			t.Error("development help incorrectly restricts native control by ship class")
		}
		for _, want := range []string{"loom 20..34", "default 31", "native control", "default off", "full ship-control", "Primarily for development", "Leave disabled for routine use", "known crash bug", "Keep clients connected until every reply has been received", "Do not kill clients or use short timeouts"} {
			if !strings.Contains(out.String(), want) {
				t.Errorf("managed help missing %q", want)
			}
		}
	}
}

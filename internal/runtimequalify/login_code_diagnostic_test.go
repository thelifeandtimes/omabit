package runtimequalify

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestLoginCodeDiagnosticNeverExposesCauseOrUnknownStage(t *testing.T) {
	for _, err := range []error{nil, (*codeReadError)(nil), errors.New("PRIVATE secret"), &codeReadError{stage: "PRIVATE secret", deadline: true}} {
		if stage, deadline := LoginCodeDiagnostic(err); stage != "unavailable" || deadline {
			t.Fatal("unknown diagnostics exposed", stage, deadline)
		}
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	err := codeReadFailure(ctx, "identity-after")
	if strings.Contains(err.Error(), "identity") || strings.Contains(err.Error(), "PRIVATE") {
		t.Fatal("unexpected public error")
	}
	if stage, deadline := LoginCodeDiagnostic(fmt.Errorf("outer: %w", err)); stage != "identity-after" || !deadline {
		t.Fatal(stage, deadline)
	}
	if stage, deadline := LoginCodeDiagnostic(codeReadFailure(context.Background(), "engine")); stage != "engine" || deadline {
		t.Fatal(stage, deadline)
	}
}

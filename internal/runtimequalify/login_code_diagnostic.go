package runtimequalify

import (
	"context"
	"errors"
)

// codeReadError contains only a fixed stage and whether the outer deadline
// expired. It must never wrap the command error, response body, code or cookie.
type codeReadError struct {
	stage    string
	deadline bool
}

func (e *codeReadError) Error() string {
	return "login code unavailable for the selected registered runtime"
}
func codeReadFailure(ctx context.Context, stage string) error {
	return &codeReadError{stage: stage, deadline: errors.Is(ctx.Err(), context.DeadlineExceeded)}
}

// LoginCodeDiagnostic exposes a finite nonsecret vocabulary for the access audit.
// Unknown backend errors are deliberately discarded, not rendered or persisted.
func LoginCodeDiagnostic(err error) (stage string, deadline bool) {
	var failure *codeReadError
	if !errors.As(err, &failure) || failure == nil {
		return "unavailable", false
	}
	switch failure.stage {
	case "selection", "session", "engine", "container", "port-inventory", "identity-before", "inventory-before-code", "read-code", "inventory-after-code", "identity-after":
		return failure.stage, failure.deadline
	default:
		return "unavailable", false
	}
}

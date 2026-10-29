package client

import "fmt"

// OutcomeUncertainError means a mutation lacks a trustworthy runner response.
// It does not assert acceptance, rejection or whether bytes reached the runner.
// Callers must reconcile the original request rather than automatically retry.
// No request payload or credential is retained here.
type OutcomeUncertainError struct {
	Method string
	Path   string
	Err    error
}

func (e *OutcomeUncertainError) Error() string {
	return fmt.Sprintf("%s %s outcome uncertain; the runner may have accepted the request: %v", e.Method, e.Path, e.Err)
}

func (e *OutcomeUncertainError) Unwrap() error { return e.Err }

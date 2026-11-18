//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

var errGrubberyReadScope = errors.New("private app read scope unavailable; no effect was submitted")

type grubberyReadScopeKey struct{}
type grubberyScopedRequest struct {
	scope   *grubberyReadScope
	request grubbery.Request
}
type grubberyReadScope struct {
	mu      sync.Mutex
	runtime *DeskRuntime
	binding grubbery.Binding
	ctx     context.Context
	owner   *eyre.Client
	closed  bool
}

// A request-local sealed selection replaces no shared transport state. Only
// typed GETs made by this exact live scope can reach the application endpoint.
func (s *grubberyReadScope) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL != nil && (req.URL.Path == "/~/login" || req.URL.Path == "/~/logout" || req.URL.Path == "/~/host" || req.URL.Path == "/~/name") {
		return (grubberyHTTPTransport{runtime: s.runtime}).RoundTrip(req)
	}
	selected, ok := req.Context().Value(grubberyReadScopeKey{}).(grubberyScopedRequest)
	if !ok || selected.scope != s || selected.request.Method != http.MethodGet {
		return nil, errGrubberyReadScope
	}
	return (grubberyHTTPTransport{runtime: s.runtime, selected: selected.request}).RoundTrip(req)
}

func (s *grubberyReadScope) GrubberyRead(ctx context.Context, read grubbery.ReadRequest) (grubbery.NativeResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.ctx.Err() != nil {
		return grubbery.NativeResponse{}, errGrubberyReadScope
	}
	binding, err := read.Binding()
	if err != nil || binding != s.binding {
		return grubbery.NativeResponse{}, errGrubberyReadScope
	}
	request, err := read.Request()
	if err != nil || request.Method != http.MethodGet {
		return grubbery.NativeResponse{}, errGrubberyReadScope
	}
	// A retained callback cannot provide an independent background lifetime.
	ctx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(s.ctx, cancel)
	defer cancel()
	defer stop()
	ctx = context.WithValue(ctx, grubberyReadScopeKey{}, grubberyScopedRequest{s, request})
	response, err := s.owner.GrubberyRead(ctx, read)
	response.SessionCleanup = grubbery.NativeReadScopeActive
	return response, err
}

// WithGrubberyReads preserves sequential/conditional reads and authenticates
// once for one read-only capture. Every transport call retains its exact
// process/URL/method/body checks; framework/pier bindings bracket the scope.
// The server must hold its lifecycle/policy guard throughout. No write method
// is exposed to the callback. Provisional reads cannot become journal evidence
// until logout is confirmed, and retained readers refuse after the callback.
func (s *DeskRuntime) WithGrubberyReads(ctx context.Context, binding grubbery.Binding, run func(grubbery.NativeReader) error) (cleanup string, err error) {
	cleanup = "not-created"
	if run == nil {
		return cleanup, errGrubberyReadScope
	}
	if err = s.grubberyBinding(ctx, binding); err != nil {
		return cleanup, err
	}
	code, e := s.call(ctx, deskwork.FrontendOwnerCode())
	if e != nil || !protocol.ValidLoginCode(strings.TrimSpace(code)) {
		return cleanup, errGrubberyReadScope
	}
	scope := &grubberyReadScope{runtime: s, binding: binding, ctx: ctx}
	owner, e := eyre.NewWithTransport(s.httpEndpoint, s.ship, scope)
	if e != nil {
		return cleanup, errGrubberyReadScope
	}
	scope.owner = owner
	defer owner.Close()
	cleanup = "unconfirmed"
	defer func() {
		scope.mu.Lock()
		scope.closed = true
		scope.mu.Unlock()
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if owner.Logout(cleanupCtx) == nil {
			cleanup = "confirmed"
		} else if err == nil {
			err = errGrubberyReadScope
		}
	}()
	raw := []byte(strings.TrimSpace(code))
	code = ""
	defer clear(raw)
	if owner.Login(ctx, raw) != nil {
		return cleanup, errGrubberyReadScope
	}
	if err = s.grubberyBinding(ctx, binding); err != nil {
		return cleanup, err
	}
	err = run(scope)
	if changed := s.grubberyBinding(ctx, binding); changed != nil {
		return cleanup, errGrubberyReadScope
	}
	return cleanup, err
}

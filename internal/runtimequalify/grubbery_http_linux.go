//go:build linux && amd64

package runtimequalify

import (
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// This script has no configurable destination, command, retry, redirect, proxy,
// file or auth argument. The adapter independently pins one private URL/method/
// body before invoking it; credentials and binary content are separate stdin.
const grubberyHTTPScript = `set -eu
IFS= read -r omabit_content_type
IFS= read -r omabit_cookie
exec 3<&0
{
 printf '%s\n' 'Expect:' 'Accept-Encoding: identity'
 if [ -n "$omabit_content_type" ]; then printf 'Content-Type: %s\n' "$omabit_content_type"; fi
 if [ -n "$omabit_cookie" ]; then printf 'Cookie: %s\n' "$omabit_cookie"; fi
} | if [ "$1" = PUT ] || [ "$1" = POST ]; then
 /usr/bin/curl --disable --noproxy '*' --proxy '' --silent --include --raw --max-time 30 --max-filesize 1048576 --request "$1" --header @- --data-binary @/dev/fd/3 "$2"
else
 /usr/bin/curl --disable --noproxy '*' --proxy '' --silent --include --raw --max-time 30 --max-filesize 1048576 --request GET --header @- "$2"
fi
`

type grubberyHTTPTransport struct {
	runtime  *DeskRuntime
	selected grubbery.Request
}

func (t grubberyHTTPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	unavailable := errors.New("private app HTTP transport unavailable; an effect was not retried")
	s := t.runtime
	if req.Body != nil {
		defer req.Body.Close()
	}
	if s == nil || s.check == nil || s.q == nil || req.URL == nil || req.URL.User != nil || req.URL.Fragment != "" || req.URL.RawFragment != "" || req.URL.RawPath != "" || req.URL.Opaque != "" || req.URL.ForceQuery || req.URL.Scheme+"://"+req.URL.Host != s.httpEndpoint || req.Host != "" && req.Host != req.URL.Host {
		return nil, unavailable
	}
	// Fixed authentication/identity/cleanup routes retain their existing stricter
	// implementation. This does not widen ordinary desk HTTP transport.
	if req.URL.RawQuery == "" && (req.URL.Path == "/~/host" || req.URL.Path == "/~/name" || req.URL.Path == "/~/login" || req.URL.Path == "/~/logout") {
		return (deskHTTPTransport{s}).RoundTrip(req)
	}
	q := t.selected
	if req.Method != q.Method || req.URL.Path != q.Path || req.URL.RawQuery != q.Query || len(req.TransferEncoding) != 0 || req.ContentLength > int64(len(q.Body)) || req.Header.Get("Content-Type") != q.ContentType {
		return nil, unavailable
	}
	if q.Method != "GET" && q.Method != "PUT" && q.Method != "POST" || !strings.HasPrefix(q.Path, "/grubbery/api/") || len(q.Body) > grubbery.MaxAppFileBytes {
		return nil, unavailable
	}
	for name, values := range req.Header {
		if name != "Content-Type" && name != "Cookie" || len(values) != 1 || len(values[0]) > 16384 || strings.ContainsAny(values[0], "\r\n\x00") {
			return nil, unavailable
		}
	}
	body := []byte{}
	var err error
	if req.Body != nil {
		body, err = io.ReadAll(io.LimitReader(req.Body, int64(len(q.Body))+1))
		defer clear(body)
	}
	if err != nil || len(body) != len(q.Body) || sha256.Sum256(body) != sha256.Sum256(q.Body) {
		return nil, unavailable
	}
	if s.check == nil || s.q == nil {
		return nil, unavailable
	}
	if s.check(req.Context()) != nil {
		return nil, unavailable
	}
	input := append([]byte(req.Header.Get("Content-Type")+"\n"+req.Header.Get("Cookie")+"\n"), body...)
	defer clear(input)
	output, err := s.q.dockerInput(req.Context(), 35*time.Second, input, "container", "exec", "--interactive", s.containerID, "/bin/sh", "-c", grubberyHTTPScript, "omabit-private-app-http", req.Method, req.URL.String())
	if err != nil || s.check(req.Context()) != nil {
		return nil, unavailable
	}
	response, err := decodeDeskHTTP(output, req)
	if err != nil {
		return nil, unavailable
	}
	return response, nil
}
func (s *DeskRuntime) grubberyBinding(ctx context.Context, b grubbery.Binding) error {
	if b.Validate() != nil || s == nil || s.containerID != b.RuntimeID || s.check == nil || s.check(ctx) != nil {
		return errors.New("app runtime binding changed")
	}
	continuity, err := s.SourceIdentity(ctx)
	if err != nil || continuity != b.Continuity {
		return errors.New("app pier continuity changed")
	}
	observation, err := s.Observe(ctx, b.Target.Desk)
	if err != nil || observation != b.FrameworkBaseline {
		return errors.New("app framework baseline changed")
	}
	return nil
}
func (s *DeskRuntime) withGrubberyOwner(ctx context.Context, b grubbery.Binding, q grubbery.Request, run func(*eyre.Client) (eyre.GrubberyResponse, error)) (result eyre.GrubberyResponse, err error) {
	result.SessionCleanup = "not-created"
	if err = s.grubberyBinding(ctx, b); err != nil {
		return result, err
	}
	code, e := s.call(ctx, deskwork.FrontendOwnerCode())
	if e != nil || !protocol.ValidLoginCode(strings.TrimSpace(code)) {
		return result, errors.New("app owner session unavailable")
	}
	owner, e := eyre.NewWithTransport(s.httpEndpoint, s.ship, grubberyHTTPTransport{s, q})
	if e != nil {
		return result, e
	}
	defer owner.Close()
	result.SessionCleanup = "unconfirmed"
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if owner.Logout(cleanup) == nil {
			result.SessionCleanup = "confirmed"
		}
	}()
	raw := []byte(strings.TrimSpace(code))
	code = ""
	defer clear(raw)
	if owner.Login(ctx, raw) != nil {
		return result, errors.New("app owner session unavailable")
	}
	// Admission/authentication is not atomic with native writes. Refuse drift in
	// the interval and require the caller's lifecycle/policy guard across entry.
	if err = s.grubberyBinding(ctx, b); err != nil {
		return result, err
	}
	response, err := run(owner)
	response.SessionCleanup = result.SessionCleanup
	result = response
	if check := s.grubberyBinding(ctx, b); check != nil {
		return result, errors.New("app runtime changed after request; outcome unconfirmed")
	}
	return result, err
}

// GrubberyStep requires the server's lifecycle/policy guard and a durably saved
// original dispatch. It never admits policy, creates a journal or retries a call.
func (s *DeskRuntime) GrubberyStep(ctx context.Context, p grubbery.Plan, stepID string, files []deskwork.SourceFile) (eyre.GrubberyResponse, error) {
	if err := grubbery.CheckNativeSource(p.Source); err != nil {
		return eyre.GrubberyResponse{}, err
	}
	q, err := p.Request(stepID, files)
	if err != nil {
		return eyre.GrubberyResponse{}, err
	}
	defer clear(q.Body)
	return s.withGrubberyOwner(ctx, p.Binding, q, func(owner *eyre.Client) (eyre.GrubberyResponse, error) {
		return owner.GrubberyStep(ctx, p, stepID, files)
	})
}
func (s *DeskRuntime) GrubberyRead(ctx context.Context, read grubbery.ReadRequest) (eyre.GrubberyResponse, error) {
	q, err := read.Request()
	if err != nil {
		return eyre.GrubberyResponse{}, err
	}
	b, err := read.Binding()
	if err != nil {
		return eyre.GrubberyResponse{}, err
	}
	return s.withGrubberyOwner(ctx, b, q, func(owner *eyre.Client) (eyre.GrubberyResponse, error) { return owner.GrubberyRead(ctx, read) })
}

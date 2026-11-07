//go:build linux && amd64

package runtimequalify

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

// Headers and body use separate stdin descriptors. Only fixed code, a method
// and the pinned loopback URL appear in argv. No shell expansion of input,
// temporary files, curl config, proxy, redirects, or retry options are allowed.
const deskHTTPScript = `set -eu
IFS= read -r omabit_content_type
IFS= read -r omabit_cookie
exec 3<&0
{
  printf '%s\n' 'Expect:' 'Accept-Encoding: identity'
  if [ -n "$omabit_content_type" ]; then printf 'Content-Type: %s\n' "$omabit_content_type"; fi
  if [ -n "$omabit_cookie" ]; then printf 'Cookie: %s\n' "$omabit_cookie"; fi
} | if [ "$1" = POST ]; then
  /usr/bin/curl --disable --noproxy '*' --proxy '' --silent --include --raw --max-time 30 --max-filesize 1048576 --request POST --header @- --data-binary @/dev/fd/3 "$2"
else
  /usr/bin/curl --disable --noproxy '*' --proxy '' --silent --include --raw --max-time 30 --max-filesize 1048576 --request GET --header @- "$2"
fi
`

type deskHTTPTransport struct{ runtime *DeskRuntime }

func (t deskHTTPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	unavailable := errors.New("private ship HTTP transport unavailable; an effect was not retried")
	s := t.runtime
	if req.Body != nil {
		defer req.Body.Close()
	}
	if s == nil || s.httpEndpoint == "" || req.URL == nil || req.URL.User != nil || req.URL.RawQuery != "" || req.URL.ForceQuery || req.URL.Fragment != "" || req.URL.RawFragment != "" || req.URL.RawPath != "" || req.URL.Opaque != "" || req.URL.Scheme+"://"+req.URL.Host != s.httpEndpoint || req.Host != "" && req.Host != req.URL.Host {
		return nil, unavailable
	}
	var limit int64
	switch {
	case req.Method == "GET" && (req.URL.Path == "/~/host" || req.URL.Path == "/~/name"):
	case req.Method == "POST" && req.URL.Path == "/~/login":
		limit = 4096
	case req.Method == "POST" && req.URL.Path == "/~/logout":
	case req.Method == "POST" && req.URL.Path == "/docket/upload":
		limit = 36 << 20
	default:
		return nil, unavailable
	}
	if len(req.TransferEncoding) != 0 || req.ContentLength > limit {
		return nil, unavailable
	}
	for name, values := range req.Header {
		if name != "Content-Type" && name != "Cookie" || len(values) != 1 || len(values[0]) > 16384 || strings.ContainsAny(values[0], "\r\n\x00") {
			return nil, unavailable
		}
	}
	var body []byte
	var err error
	if req.Body != nil {
		body, err = io.ReadAll(io.LimitReader(req.Body, limit+1))
		defer clear(body)
		if err != nil || int64(len(body)) > limit {
			return nil, unavailable
		}
	}
	input := append([]byte(req.Header.Get("Content-Type")+"\n"+req.Header.Get("Cookie")+"\n"), body...)
	defer clear(input)
	if err = s.check(req.Context()); err != nil {
		return nil, unavailable
	}
	out, err := s.q.dockerInput(req.Context(), 35*time.Second, input, "container", "exec", "--interactive", s.containerID, "/bin/sh", "-c", deskHTTPScript, "omabit-private-http", req.Method, req.URL.String())
	if err != nil {
		return nil, unavailable
	}
	if err = s.check(req.Context()); err != nil {
		return nil, unavailable
	}
	response, err := decodeDeskHTTP(out, req)
	if err != nil {
		return nil, unavailable
	}
	return response, nil
}

func decodeDeskHTTP(raw string, req *http.Request) (*http.Response, error) {
	bad := errors.New("invalid private HTTP response")
	if len(raw) > outputLimit {
		return nil, bad
	}
	r := bufio.NewReader(strings.NewReader(raw))
	for i := 0; i < 8; i++ {
		response, err := http.ReadResponse(r, req)
		if err != nil {
			return nil, bad
		}
		if response.StatusCode == 100 {
			response.Body.Close()
			continue
		}
		if response.StatusCode < 200 || response.ContentLength > outputLimit {
			response.Body.Close()
			return nil, bad
		}
		data, err := io.ReadAll(io.LimitReader(response.Body, outputLimit+1))
		response.Body.Close()
		if err != nil || len(data) > outputLimit {
			clear(data)
			return nil, bad
		}
		if _, err = r.Peek(1); err != io.EOF {
			clear(data)
			return nil, bad
		}
		response.Body = io.NopCloser(bytes.NewReader(data))
		response.Request = req
		return response, nil
	}
	return nil, bad
}

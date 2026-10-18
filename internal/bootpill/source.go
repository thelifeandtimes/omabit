package bootpill

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// Resolve runs as the desktop/collector user. Local paths never reach the runner.
func Resolve(ctx context.Context, source, dir string) (string, protocol.BootPill, error) {
	return ResolveExpected(ctx, source, dir, nil)
}

// ValidateSource rejects insecure network sources before any download or upload.
// Explicit local files remain available for custom provisioning.
func ValidateSource(source string) error {
	u, err := url.Parse(source)
	if err != nil {
		return errors.New("invalid boot pill source")
	}
	if u.Scheme == "" && u.Host == "" && !strings.Contains(source, "://") {
		return nil
	}
	if u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" || u.Opaque != "" {
		return errors.New("network boot pills require an HTTPS URL without user credentials or fragment; custom local files are also supported")
	}
	return nil
}

// ResolveExpected verifies a trusted descriptor before returning any staged
// content. Custom sources may omit it; the built-in release never does.
func ResolveExpected(ctx context.Context, source, dir string, expected *protocol.BootPill) (string, protocol.BootPill, error) {
	if err := ValidateSource(source); err != nil {
		return "", protocol.BootPill{}, err
	}
	if IsRecommended(source) {
		pin := RecommendedDescriptor()
		if expected != nil && *expected != pin {
			return "", protocol.BootPill{}, errors.New("expected descriptor conflicts with the Recommended Apps release")
		}
		source, expected = RecommendedURL, &pin
	}
	if expected != nil && !protocol.ValidBootPill(expected) {
		return "", protocol.BootPill{}, errors.New("invalid expected boot pill descriptor")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	var body io.ReadCloser
	if u, _ := url.Parse(source); u.Scheme == "https" {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.Proxy = nil
		transport.DisableCompression = true
		client := &http.Client{Transport: transport, Timeout: 5 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 5 || ValidateSource(req.URL.String()) != nil || req.URL.Scheme != "https" {
				return errors.New("unsupported pill redirect")
			}
			// Custom signed URLs may carry secrets in their query. Do not leak
			// those to a redirect target in an automatically generated Referer.
			req.Header.Del("Referer")
			return nil
		}}
		defer client.CloseIdleConnections()
		req, err := http.NewRequestWithContext(ctx, "GET", source, nil)
		if err != nil {
			return "", protocol.BootPill{}, errors.New("cannot prepare pill download")
		}
		req.Header.Set("Accept-Encoding", "identity")
		resp, err := client.Do(req)
		if err != nil {
			return "", protocol.BootPill{}, errors.New("boot pill download failed")
		}
		if resp.StatusCode != 200 || resp.ContentLength > protocol.MaxBootPillBytes || resp.Header.Get("Content-Encoding") != "" && resp.Header.Get("Content-Encoding") != "identity" {
			resp.Body.Close()
			return "", protocol.BootPill{}, errors.New("boot pill response is unsupported or too large")
		}
		body = resp.Body
	} else {
		f, err := Open(source)
		if err != nil {
			return "", protocol.BootPill{}, err
		}
		body = f
	}
	defer body.Close()
	return Stage(ctx, dir, body, expected)
}

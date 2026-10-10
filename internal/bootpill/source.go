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
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	var body io.ReadCloser
	if strings.HasPrefix(source, "https://") || strings.HasPrefix(source, "http://") {
		u, err := url.Parse(source)
		if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" {
			return "", protocol.BootPill{}, errors.New("invalid boot pill URL")
		}
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.Proxy = nil
		transport.DisableCompression = true
		client := &http.Client{Transport: transport, Timeout: 5 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 5 || req.URL.User != nil || (req.URL.Scheme != "https" && req.URL.Scheme != "http") || via[0].URL.Scheme == "https" && req.URL.Scheme != "https" {
				return errors.New("unsupported pill redirect")
			}
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
	return Stage(ctx, dir, body, nil)
}

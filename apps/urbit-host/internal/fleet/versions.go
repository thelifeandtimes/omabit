package fleet

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type VersionImage struct {
	Repo  string `json:"repo"`
	Tag   string `json:"tag"`
	AMD64 string `json:"amd64_sha256"`
	ARM64 string `json:"arm64_sha256"`
}

func ParseVersion(b []byte, channel, arch string) (string, error) {
	var m struct {
		Groundseg map[string]struct {
			Vere VersionImage `json:"vere"`
		} `json:"groundseg"`
	}
	if e := json.Unmarshal(b, &m); e != nil {
		return "", e
	}
	c, ok := m.Groundseg[channel]
	if !ok {
		return "", fmt.Errorf("release channel %q missing; refusing silent fallback", channel)
	}
	hash := c.Vere.AMD64
	if arch == "arm64" {
		hash = c.Vere.ARM64
	} else if arch != "amd64" {
		return "", fmt.Errorf("unsupported architecture: %s", arch)
	}
	if !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(hash) {
		return "", fmt.Errorf("channel has no valid image digest for %s", arch)
	}
	if c.Vere.Repo == "" || c.Vere.Tag == "" {
		return "", fmt.Errorf("incomplete version manifest")
	}
	ref := c.Vere.Repo + ":" + c.Vere.Tag + "@sha256:" + hash
	return ref, ValidateImage(ref)
}
func ResolveVersion(ctx context.Context, urlString, channel string) (string, error) {
	if channel != "latest" && channel != "edge" && channel != "canary" {
		return "", fmt.Errorf("channel must be latest, edge or canary")
	}
	u, e := url.Parse(urlString)
	if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return "", fmt.Errorf("version URL must be HTTPS")
	}
	req, e := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if e != nil {
		return "", e
	}
	req.Header.Set("User-Agent", "Omarchy-Urbit/"+Version)
	client := &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if r.URL.Scheme != "https" || len(via) > 5 {
			return fmt.Errorf("unsafe redirect")
		}
		return nil
	}}
	resp, e := client.Do(req)
	if e != nil {
		return "", e
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("version server HTTP %d", resp.StatusCode)
	}
	b, e := io.ReadAll(io.LimitReader(resp.Body, (2<<20)+1))
	if e != nil {
		return "", e
	}
	if len(b) > 2<<20 {
		return "", fmt.Errorf("version manifest exceeds 2 MiB")
	}
	return ParseVersion(b, strings.TrimSpace(channel), runtime.GOARCH)
}

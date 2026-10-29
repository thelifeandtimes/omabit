// Package eyre implements origin-bound owner sessions without runner access.
package eyre

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	ErrAuth      = errors.New("ship session is missing, expired, or rejected; log in again")
	ErrIdentity  = errors.New("ship identity differs from enrollment")
	ErrTransport = errors.New("ship connection failed")
	ErrRedirect  = errors.New("ship redirect refused; enroll the intended origin explicitly")
	ErrProtocol  = errors.New("invalid or oversized Eyre response")
	ErrUncertain = errors.New("Eyre request outcome is uncertain; it was not retried")
)

const maxBody = 1 << 20

var shipPattern = regexp.MustCompile(`^~[a-z]+(-{1,2}[a-z]+)*$`)
var termPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)
var codePattern = regexp.MustCompile(`^[a-z]{6}(-[a-z]{6}){3}$`)

func ValidShip(s string) bool { return len(s) >= 4 && len(s) <= 128 && shipPattern.MatchString(s) }

// Origin never contains a path, userinfo, query, or fragment. Plain HTTP is
// supported for explicitly selected local, LAN/VPN and public endpoints. HTTP
// provides no transport encryption; redirects and ambient proxies remain disabled.
func Origin(raw string) (string, error) {
	if len(raw) > 2048 || strings.ContainsAny(raw, "\r\n\t ") {
		return "", errors.New("invalid ship origin")
	}
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || u.Opaque != "" || (u.Path != "" && u.Path != "/") || u.RawPath != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.Contains(raw, "#") || u.Host == "" {
		return "", errors.New("ship URL must be an origin without credentials, path, query, or fragment")
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return "", errors.New("ship origin requires HTTPS or HTTP")
	}
	host := strings.ToLower(u.Hostname())
	if strings.ContainsAny(host, "%\\") || host == "" {
		return "", errors.New("invalid ship host")
	}
	for _, r := range host {
		if r > 127 {
			return "", errors.New("use an ASCII or punycode ship host")
		}
	}
	if ip, err := netip.ParseAddr(host); err == nil {
		ip = ip.Unmap()
		if ip.IsUnspecified() || ip.IsMulticast() {
			return "", errors.New("ship host must be a unicast address")
		}
	} else {
		if len(host) > 253 || regexp.MustCompile(`^[0-9.]+$`).MatchString(host) {
			return "", errors.New("invalid ship hostname")
		}
		for _, label := range strings.Split(host, ".") {
			if !regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`).MatchString(label) {
				return "", errors.New("invalid ship hostname")
			}
		}
	}
	if strings.HasSuffix(u.Host, ":") {
		return "", errors.New("invalid ship port")
	}
	port := u.Port()
	if port != "" {
		n, err := strconv.ParseUint(port, 10, 16)
		if err != nil || n == 0 {
			return "", errors.New("invalid ship port")
		}
		port = strconv.FormatUint(n, 10)
	}
	if (u.Scheme == "https" && port == "443") || (u.Scheme == "http" && port == "80") {
		port = ""
	}
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	if port != "" {
		host += ":" + port
	}
	return u.Scheme + "://" + host, nil
}

func ValidPath(p string) bool {
	if len(p) == 0 || len(p) > 2048 || p[0] != '/' || strings.HasPrefix(p, "//") || path.Clean(p) != p {
		return false
	}
	for _, r := range p {
		if r < 33 || r > 126 || strings.ContainsRune("%?#\\", r) {
			return false
		}
	}
	return true
}

type session struct {
	Origin  string    `json:"origin"`
	Ship    string    `json:"ship"`
	Name    string    `json:"cookie_name"`
	Value   string    `json:"cookie_value"`
	Expires time.Time `json:"expires"`
}

type Client struct {
	origin, ship string
	http         *http.Client
	session      session
}

func New(origin, ship string) (*Client, error) {
	canonical, err := Origin(origin)
	if err != nil {
		return nil, err
	}
	if !ValidShip(ship) {
		return nil, errors.New("require an expected ship name beginning with ~")
	}
	transport := &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 5 * time.Second, MaxResponseHeaderBytes: 16384, IdleConnTimeout: 30 * time.Second}
	return &Client{origin: canonical, ship: ship, http: &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

func (c *Client) Close() { c.http.CloseIdleConnections(); c.session = session{} }

func (c *Client) request(ctx context.Context, method, p string, body []byte, auth bool, headers http.Header) (*http.Response, error) {
	return c.requestWithSession(ctx, method, p, body, auth, headers, false)
}

// allowExpired is restricted to exact-session cleanup, never data requests.
func (c *Client) requestWithSession(ctx context.Context, method, p string, body []byte, auth bool, headers http.Header, allowExpired bool) (*http.Response, error) {
	if !ValidPath(p) {
		return nil, errors.New("invalid Eyre path")
	}
	if allowExpired && (method != "POST" || p != "/~/logout" || !auth || len(body) != 0) {
		return nil, ErrProtocol
	}
	if auth && (c.session.Value == "" || !allowExpired && !time.Now().Before(c.session.Expires)) {
		return nil, ErrAuth
	}
	req, err := http.NewRequestWithContext(ctx, method, c.origin+p, bytes.NewReader(body))
	if err != nil {
		return nil, ErrProtocol
	}
	req.Header = headers.Clone()
	if req.Header == nil {
		req.Header = make(http.Header)
	}
	// No cookie jar: ordinary jars share cookies across ports. This client sends
	// exactly one cookie only to its immutable, canonical scheme/host/port origin.
	if auth {
		req.AddCookie(&http.Cookie{Name: c.session.Name, Value: c.session.Value})
	}
	req.GetBody = nil // POST/PUT are never automatically replayed
	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if method != "GET" {
			return nil, ErrUncertain
		}
		return nil, ErrTransport
	}
	if resp.StatusCode >= 300 && resp.StatusCode < 400 && !(method == "POST" && p == "/~/logout" && resp.StatusCode == 303) {
		resp.Body.Close()
		return nil, ErrRedirect
	}
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		resp.Body.Close()
		return nil, ErrAuth
	}
	return resp, nil
}

func readResponse(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil || len(data) > maxBody {
		return nil, ErrProtocol
	}
	return data, nil
}

func (c *Client) getText(ctx context.Context, p string, auth bool) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	resp, err := c.request(ctx, "GET", p, nil, auth, nil)
	if err != nil {
		return "", err
	}
	data, err := readResponse(resp)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Eyre identity check returned HTTP %d", resp.StatusCode)
	}
	return strings.TrimSpace(string(data)), nil
}

func (c *Client) Probe(ctx context.Context) error {
	ship, err := c.getText(ctx, "/~/host", false)
	if err != nil {
		return err
	}
	if ship != c.ship {
		return ErrIdentity
	}
	return nil
}

func (c *Client) Check(ctx context.Context) error {
	if err := c.Probe(ctx); err != nil {
		return err
	}
	name, err := c.getText(ctx, "/~/name", true)
	if err != nil {
		return err
	}
	if name != c.ship {
		return ErrAuth
	}
	return nil
}

func (c *Client) Login(ctx context.Context, code []byte) error {
	defer clear(code)
	code = bytes.TrimSpace(code)
	if !codePattern.Match(code) {
		return errors.New("require a four-part web login code on stdin")
	}
	if err := c.Probe(ctx); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	body := []byte(url.Values{"password": {string(code)}}.Encode())
	defer clear(body)
	resp, err := c.request(ctx, "POST", "/~/login", body, false, http.Header{"Content-Type": []string{"application/x-www-form-urlencoded"}})
	if err != nil {
		return err
	}
	// A successful login body may itself contain a session token: never expose it.
	data, err := readResponse(resp)
	clear(data)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		return ErrAuth
	}
	var found *http.Cookie
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "urbauth-"+c.ship {
			if found != nil {
				return ErrProtocol
			}
			found = cookie
		}
	}
	u, _ := url.Parse(c.origin)
	if found == nil || found.Value == "" || found.Valid() != nil || found.MaxAge < 0 || (found.Path != "" && found.Path != "/") || (found.Domain != "" && strings.TrimPrefix(found.Domain, ".") != u.Hostname()) || (found.Secure && u.Scheme != "https") {
		return ErrProtocol
	}
	expires := time.Now().Add(30 * 24 * time.Hour)
	if found.MaxAge > 0 {
		expiry := time.Now().Add(time.Duration(min(found.MaxAge, 30*24*3600)) * time.Second)
		if expiry.Before(expires) {
			expires = expiry
		}
	}
	if !found.Expires.IsZero() && found.Expires.Before(expires) {
		expires = found.Expires
	}
	if !time.Now().Before(expires) {
		return ErrAuth
	}
	c.session = session{Origin: c.origin, Ship: c.ship, Name: found.Name, Value: found.Value, Expires: expires}
	if err := c.Check(ctx); err != nil {
		_ = c.Logout(context.Background())
		return err
	}
	return nil
}

// ExportSession is for credential adapters only. It is never a public CLI result.
func (c *Client) ExportSession() ([]byte, error) {
	if c.session.Value == "" {
		return nil, ErrAuth
	}
	return json.Marshal(c.session)
}

func (c *Client) RestoreSession(data []byte) error {
	return c.restoreSession(data, false)
}

func (c *Client) restoreSession(data []byte, allowExpired bool) error {
	defer clear(data)
	var s session
	if len(data) > 16384 || json.Unmarshal(data, &s) != nil || s.Origin != c.origin || s.Ship != c.ship || s.Name != "urbauth-"+c.ship || s.Value == "" || (&http.Cookie{Name: s.Name, Value: s.Value}).Valid() != nil {
		return errors.New("stored ship session is invalid or belongs to another origin")
	}
	if s.Expires.IsZero() || !allowExpired && !time.Now().Before(s.Expires) {
		return ErrAuth
	}
	c.session = s
	return nil
}

func (c *Client) Logout(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	defer func() { c.session = session{} }()
	// Eyre intentionally responds to logout with 303. It is accepted here without
	// following its Location; other redirects remain errors.
	resp, err := c.request(ctx, "POST", "/~/logout", nil, true, nil)
	if err != nil {
		return err
	}
	data, err := readResponse(resp)
	clear(data)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 && resp.StatusCode != 204 && resp.StatusCode != 303 {
		return errors.New("ship logout was not confirmed")
	}
	return nil
}

func (c *Client) Scry(ctx context.Context, app, p string) (json.RawMessage, error) {
	if !termPattern.MatchString(app) || !ValidPath(p) {
		return nil, errors.New("invalid scry app or path")
	}
	if err := c.Check(ctx); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	resp, err := c.request(ctx, "GET", "/~/scry/"+app+p+".json", nil, true, nil)
	if err != nil {
		return nil, err
	}
	data, err := readResponse(resp)
	if err != nil {
		return nil, err
	}
	media, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("scry returned HTTP %d", resp.StatusCode)
	}
	if media != "application/json" || !json.Valid(data) {
		return nil, ErrProtocol
	}
	return json.RawMessage(data), nil
}

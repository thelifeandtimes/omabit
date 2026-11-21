package mcpserver

import (
	"context"
	"errors"
	"net"
	"net/http"
	"slices"
	"strconv"
	"sync"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/service"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HTTPOptions selects an explicit local manual-bearer deployment. It does not
// configure an OAuth authorization server, remote proxy, TLS or client settings.
type HTTPOptions struct {
	Listen    string
	TokenFile string
	Ready     func(string)
}

var httpProtocolVersions = []string{"2025-11-25", "2025-06-18", "2025-03-26"}

func localHTTPAddress(value string) bool {
	host, port, err := net.SplitHostPort(value)
	ip := net.ParseIP(host)
	n, e := strconv.Atoi(port)
	return err == nil && e == nil && ip != nil && ip.IsLoopback() && n >= 0 && n <= 65535 && strconv.Itoa(n) == port
}

// NewHTTPHandler shares the exact stdio tools and authorization with HTTP. The
// supplied verifier authenticates every request and must not reflect secrets in
// errors. Browser origins, proxies, query credentials and nonlocal peers refuse.
func NewHTTPHandler(core *service.Service, authority string, verifier auth.TokenVerifier) (http.Handler, func(), error) {
	if !localHTTPAddress(authority) || verifier == nil {
		return nil, nil, errors.New("explicit numeric loopback authority and authentication required")
	}
	server := newWithProtocols(core, httpProtocolVersions)
	safeVerifier := func(ctx context.Context, token string, r *http.Request) (*auth.TokenInfo, error) {
		info, err := verifier(ctx, token, r)
		if err != nil || info == nil || info.UserID == "" {
			return nil, auth.ErrInvalidToken
		}
		return info, nil
	}
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		count := 0
		for range server.Sessions() {
			count++
		}
		if count >= 16 {
			return nil
		}
		return server
	}, &mcp.StreamableHTTPOptions{JSONResponse: true, SessionTimeout: 5 * time.Minute, MaxRequestBodyBytes: service.MaxInputBytes + 8192})
	authenticated := auth.RequireBearerToken(safeVerifier, nil)(handler)
	active := make(chan struct{}, 16)
	var mu sync.Mutex
	var initialize sync.Mutex
	closed := false
	wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w = &httpBearerResponse{ResponseWriter: w}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		remote, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil || net.ParseIP(remote) == nil || !net.ParseIP(remote).IsLoopback() || r.Host != authority || r.Header.Get("Origin") != "" || len(r.Header.Values("Origin")) != 0 {
			http.Error(w, "local MCP request refused", http.StatusForbidden)
			return
		}
		if r.URL.Path != "/mcp" || r.URL.RawPath != "" || r.URL.RawQuery != "" || r.Header.Get("Cookie") != "" {
			http.Error(w, "invalid local MCP endpoint", http.StatusBadRequest)
			return
		}
		if len(r.Header.Values("Authorization")) != 1 {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		if versions := r.Header.Values("MCP-Protocol-Version"); len(versions) > 1 || len(versions) == 1 && !slices.Contains(httpProtocolVersions, versions[0]) {
			http.Error(w, "unsupported MCP protocol", http.StatusBadRequest)
			return
		}
		select {
		case active <- struct{}{}:
			defer func() { <-active }()
		default:
			http.Error(w, "MCP request limit", http.StatusServiceUnavailable)
			return
		}
		// This deployment has no unsolicited server messages. MCP permits405 for
		// standalone GET; POST replies remain standard Streamable HTTP JSON.
		if r.Method == http.MethodGet {
			authenticatedGet := auth.RequireBearerToken(safeVerifier, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Allow", "POST, DELETE")
				w.WriteHeader(http.StatusMethodNotAllowed)
			}))
			authenticatedGet.ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodPost && r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		mu.Lock()
		ending := closed
		mu.Unlock()
		if ending {
			http.Error(w, "MCP server closing", http.StatusServiceUnavailable)
			return
		}
		if r.Method == http.MethodPost && r.Header.Get("Mcp-Session-Id") == "" {
			initialize.Lock()
			defer initialize.Unlock()
			mu.Lock()
			ending = closed
			mu.Unlock()
			if ending {
				http.Error(w, "MCP server closing", http.StatusServiceUnavailable)
				return
			}
		}
		authenticated.ServeHTTP(w, r)
	})
	closeSessions := func() {
		initialize.Lock()
		defer initialize.Unlock()
		mu.Lock()
		closed = true
		mu.Unlock()
		for session := range server.Sessions() {
			session.Close()
		}
	}
	return wrapped, closeSessions, nil
}

type httpBearerResponse struct{ http.ResponseWriter }

func (w *httpBearerResponse) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (w *httpBearerResponse) WriteHeader(code int) {
	if code == http.StatusUnauthorized {
		w.Header().Set("WWW-Authenticate", `Bearer realm="omabit-local"`)
	}
	w.ResponseWriter.WriteHeader(code)
}

func ServeHTTP(ctx context.Context, core *service.Service, opts HTTPOptions) error {
	if !localHTTPAddress(opts.Listen) {
		return errors.New("HTTP MCP requires a numeric loopback address")
	}
	authority, err := openHTTPAuthority(opts.TokenFile)
	if err != nil {
		return errors.New("private HTTP credential unavailable or invalid")
	}
	listener, err := net.Listen("tcp", opts.Listen)
	if err != nil {
		return errors.New("HTTP MCP listener unavailable")
	}
	defer listener.Close()
	handler, closeSessions, err := NewHTTPHandler(core, listener.Addr().String(), authority.Verify)
	if err != nil {
		return err
	}
	defer closeSessions()
	if opts.Ready != nil {
		opts.Ready("http://" + listener.Addr().String() + "/mcp")
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 8192}
	result := make(chan error, 1)
	go func() { result <- server.Serve(limitHTTPListener(listener, 32)) }()
	select {
	case err = <-result:
	case <-ctx.Done():
		stop, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = server.Shutdown(stop)
		cancel()
		if err != nil {
			server.Close()
		}
		<-result
	}
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

type httpLimitedListener struct {
	net.Listener
	slots chan struct{}
	done  chan struct{}
	once  sync.Once
}
type httpLimitedConn struct {
	net.Conn
	release func()
	once    sync.Once
}

func (c *httpLimitedConn) Close() error { err := c.Conn.Close(); c.once.Do(c.release); return err }
func limitHTTPListener(l net.Listener, max int) *httpLimitedListener {
	return &httpLimitedListener{Listener: l, slots: make(chan struct{}, max), done: make(chan struct{})}
}
func (l *httpLimitedListener) Accept() (net.Conn, error) {
	select {
	case l.slots <- struct{}{}:
	case <-l.done:
		return nil, net.ErrClosed
	}
	c, err := l.Listener.Accept()
	if err != nil {
		<-l.slots
		return nil, err
	}
	return &httpLimitedConn{Conn: c, release: func() { <-l.slots }}, nil
}
func (l *httpLimitedListener) Close() error {
	l.once.Do(func() { close(l.done) })
	return l.Listener.Close()
}

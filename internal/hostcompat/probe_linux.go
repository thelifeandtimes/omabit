//go:build linux

package hostcompat

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

var errProbe = errors.New("private engine identity or bounded API probe failed")

// privatePath rejects symlink ancestors and writable ownership boundaries. A
// private directory owned by the engine UID must enclose its Unix socket.
func privatePath(name string, uid uint32, socket bool) (os.FileInfo, error) {
	if !filepath.IsAbs(name) || filepath.Clean(name) != name || len(name) > 4096 {
		return nil, errProbe
	}
	parts := strings.Split(strings.TrimPrefix(name, "/"), "/")
	prefix := "/"
	private := false
	var last os.FileInfo
	for i, part := range parts {
		prefix = filepath.Join(prefix, part)
		info, err := os.Lstat(prefix)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return nil, errProbe
		}
		s, ok := info.Sys().(*syscall.Stat_t)
		if !ok || (s.Uid != 0 && s.Uid != uid) {
			return nil, errProbe
		}
		if i == len(parts)-1 && socket {
			if info.Mode()&os.ModeSocket == 0 || s.Uid != uid || !private {
				return nil, errProbe
			}
		} else {
			if !info.IsDir() {
				return nil, errProbe
			}
			// Root-owned sticky /tmp cannot replace an owner's private child.
			if info.Mode().Perm()&0022 != 0 && !(s.Uid == 0 && info.Mode()&os.ModeSticky != 0) {
				return nil, errProbe
			}
			if s.Uid == uid && info.Mode().Perm()&0077 == 0 {
				private = true
			}
			if i == len(parts)-1 && s.Uid != uid {
				return nil, errProbe
			}
		}
		last = info
	}
	return last, nil
}

// ProbeEngine queries only /version and /v1.47/info over an owner-private Unix
// socket. Root may inspect a non-root engine; other callers must own it. Neither
// proxy environment variables nor redirects can redirect these requests.
func ProbeEngine(ctx context.Context, socket, expectedDataRoot string) (Engine, error) {
	info, err := os.Lstat(socket)
	if err != nil {
		return Engine{}, errProbe
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid == 0 || (os.Geteuid() != 0 && stat.Uid != uint32(os.Geteuid())) {
		return Engine{}, errProbe
	}
	uid := stat.Uid
	if _, err := privatePath(socket, uid, true); err != nil {
		return Engine{}, errProbe
	}
	if _, err := privatePath(expectedDataRoot, uid, false); err != nil {
		return Engine{}, errProbe
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	transport := &http.Transport{Proxy: nil, DisableKeepAlives: true, MaxResponseHeaderBytes: 16 << 10, ResponseHeaderTimeout: 2 * time.Second}
	defer transport.CloseIdleConnections()
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		before, err := privatePath(socket, uid, true)
		if err != nil || !os.SameFile(info, before) {
			return nil, errProbe
		}
		conn, err := (&net.Dialer{}).DialContext(ctx, "unix", socket)
		if err != nil {
			return nil, errProbe
		}
		u, ok := conn.(*net.UnixConn)
		if !ok {
			conn.Close()
			return nil, errProbe
		}
		raw, err := u.SyscallConn()
		var cred *unix.Ucred
		var credErr error
		if err == nil {
			err = raw.Control(func(fd uintptr) { cred, credErr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED) })
		}
		after, se := os.Lstat(socket)
		if err != nil || credErr != nil || cred == nil || cred.Uid != uid || se != nil || !os.SameFile(info, after) {
			conn.Close()
			return nil, errProbe
		}
		return conn, nil
	}
	client := &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return errProbe }}
	get := func(path string, target any) error {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://engine"+path, nil)
		res, err := client.Do(req)
		if err != nil {
			return errProbe
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return errProbe
		}
		b, err := io.ReadAll(io.LimitReader(res.Body, (1<<20)+1))
		if err != nil || len(b) > 1<<20 || json.Unmarshal(b, target) != nil {
			return errProbe
		}
		return nil
	}
	var version struct{ Version, APIVersion, MinAPIVersion string }
	if get("/version", &version) != nil || !token(version.Version) || !supportsAPI(version.MinAPIVersion, version.APIVersion) {
		return Engine{}, errProbe
	}
	var data struct {
		ServerVersion                              string
		SecurityOptions                            []string
		CgroupDriver, CgroupVersion, DockerRootDir string
	}
	if get("/v"+APIVersion+"/info", &data) != nil || data.ServerVersion != version.Version {
		return Engine{}, errProbe
	}
	e := Engine{Version: version.Version, APIVersion: version.APIVersion, MinAPIVersion: version.MinAPIVersion, CgroupDriver: data.CgroupDriver, CgroupVersion: data.CgroupVersion, DataRoot: data.DockerRootDir}
	for _, option := range data.SecurityOptions {
		if option == "name=rootless" {
			e.Rootless = true
		}
	}
	if CheckEngine(e, expectedDataRoot, true) != nil {
		return Engine{}, errProbe
	}
	return e, nil
}

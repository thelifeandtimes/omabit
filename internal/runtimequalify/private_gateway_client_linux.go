//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"syscall"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"golang.org/x/sys/unix"
)

// Private client metadata contains only the public accepted selection. Holding
// the original directories prevents inode reuse while commands are in flight.
// This guards accidental replacement/configuration drift under a trusted UID;
// it is not a security boundary against another application of that same UID.
type privateGatewayClient struct {
	mu     sync.Mutex
	paths  []string
	files  []*os.File
	marker []byte
	closed bool
}

func privateClientFile(path string, directory bool) (*os.File, error) {
	flags := unix.O_RDONLY | unix.O_NOFOLLOW | unix.O_CLOEXEC | unix.O_NONBLOCK
	if directory {
		flags |= unix.O_DIRECTORY
	}
	fd, err := unix.Open(path, flags, 0)
	if err != nil {
		return nil, errPrivateGatewayResource
	}
	return os.NewFile(uintptr(fd), "private gateway client metadata"), nil
}

func openPrivateGatewayClient(ctx context.Context, q *qualifier, g privatechain.Gateway, fresh bool) (*privateGatewayClient, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// The caller validates the complete plan/workspace binding before this helper.
	if q == nil || !privateGatewayMountPath(q.base) || !validPrivateGatewayOperation(g.Selection.OperationID) {
		return nil, errPrivateGatewayResource
	}
	marker, err := json.Marshal(struct {
		Version   int                           `json:"version"`
		Selection privatechain.GatewaySelection `json:"selection"`
	}{1, g.Selection})
	if err != nil || len(marker) > 16<<10 {
		return nil, errPrivateGatewayResource
	}
	name := filepath.Join(q.base, "private-gateway-client-"+g.Selection.OperationID)
	c := &privateGatewayClient{paths: []string{q.base, name, filepath.Join(name, "docker-config"), filepath.Join(name, "session.json")}, marker: marker}
	f, err := privateClientFile(q.base, true)
	if err != nil {
		return nil, err
	}
	c.files = append(c.files, f)
	fail := func() (*privateGatewayClient, error) { c.Close(); return nil, errPrivateGatewayResource }
	if !privateClientOwned(f, true) {
		return fail()
	}
	canonical, err := filepath.EvalSymlinks(q.base)
	if err != nil || canonical != q.base {
		return fail()
	}
	if fresh {
		if g.Phase != "planned" || os.Mkdir(name, 0700) != nil {
			return fail()
		}
		// A failed partial setup is retained; never adopt or repair it on retry.
		if os.Mkdir(c.paths[2], 0700) != nil {
			return fail()
		}
		m, e := os.OpenFile(c.paths[3], os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0600)
		if e != nil {
			return fail()
		}
		_, writeErr := m.Write(marker)
		syncErr := m.Sync()
		closeErr := m.Close()
		if errors.Join(writeErr, syncErr, closeErr) != nil {
			return fail()
		}
	} else if g.Phase != "start-dispatched" && g.Phase != "started" && g.Phase != "ready" {
		return fail()
	}
	for n, p := range c.paths[1:] {
		f, err = privateClientFile(p, n < 2)
		if err != nil {
			return fail()
		}
		c.files = append(c.files, f)
	}
	if err = c.current(); err != nil {
		return fail()
	}
	if fresh {
		for _, f := range []*os.File{c.files[2], c.files[1], c.files[0]} {
			if f.Sync() != nil {
				return fail()
			}
		}
	}
	if ctx.Err() != nil {
		return fail()
	}
	q.r.Directory = name
	q.config = c.paths[2]
	// Every CLI command uses the same empty config. Return a known create ID even
	// if post-command validation fails, so the controller can retain that outcome.
	execute := q.d.execute
	q.d.execute = func(ctx context.Context, args, env []string) (string, error) {
		if err := c.current(); err != nil {
			return "", err
		}
		out, err := execute(ctx, args, env)
		return out, errors.Join(err, c.current())
	}
	return c, nil
}

func privateClientOwned(f *os.File, directory bool) bool {
	i, err := f.Stat()
	if err != nil {
		return false
	}
	s, ok := i.Sys().(*syscall.Stat_t)
	if !ok || s.Uid != uint32(os.Geteuid()) {
		return false
	}
	if directory {
		return i.Mode() == os.ModeDir|0700
	}
	return i.Mode() == 0600 && s.Nlink == 1
}

func (c *privateGatewayClient) current() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || len(c.files) != 4 {
		return errPrivateGatewayResource
	}
	for n, p := range c.paths {
		canonical, err := filepath.EvalSymlinks(p)
		if err != nil || canonical != p || !privateClientOwned(c.files[n], n < 3) {
			return errPrivateGatewayResource
		}
		held, err := c.files[n].Stat()
		if err != nil {
			return errPrivateGatewayResource
		}
		named, err := os.Lstat(p)
		if err != nil || !os.SameFile(held, named) {
			return errPrivateGatewayResource
		}
	}
	for _, n := range []int{1, 2} {
		if _, err := c.files[n].Seek(0, 0); err != nil {
			return errPrivateGatewayResource
		}
		entries, err := c.files[n].ReadDir(3)
		if err != nil && !errors.Is(err, io.EOF) {
			return errPrivateGatewayResource
		}
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		slices.Sort(names)
		if (n == 1 && !slices.Equal(names, []string{"docker-config", "session.json"})) || (n == 2 && len(names) != 0) {
			return errPrivateGatewayResource
		}
	}
	m := c.files[3]
	info, err := m.Stat()
	if err != nil || info.Size() != int64(len(c.marker)) {
		return errPrivateGatewayResource
	}
	raw := make([]byte, len(c.marker))
	n, err := m.ReadAt(raw, 0)
	if err != nil || n != len(raw) || !bytes.Equal(raw, c.marker) {
		return errPrivateGatewayResource
	}
	return nil
}

func (c *privateGatewayClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	var errs []error
	for _, f := range c.files {
		errs = append(errs, f.Close())
	}
	return errors.Join(errs...)
}

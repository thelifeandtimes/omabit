//go:build linux

package privatechain

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/sys/unix"
)

const gatewayBootstrapMarker = "private-chain-docker-gateway-v1\n"

// GatewayBootstrap owns only the small filesystem effect required by the guest
// gateway. Open and VerifyEmpty are read-only. Initialize belongs to an already
// committed start-dispatched receipt, after engine/container/input checks and
// before the one container Start. It never creates a chain, key or pier, executes
// a program, or contacts Docker. A failed/partial initialization is retained;
// neither this handle nor recovery may automatically retry it.
//
// The current gateway remains a disposable-QEMU qualification candidate. This
// helper does not relax its guest/namespace guards or authorize public startup.
type GatewayBootstrap struct {
	mu                sync.Mutex
	verifier          *WorkspaceVerifier
	plan              Plan
	workspace         Workspace
	selection         GatewaySelection
	attempted, closed bool
}

func OpenGatewayBootstrap(ctx context.Context, directory string, p Plan, w Workspace, g Gateway) (*GatewayBootstrap, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if g.Validate(p, w) != nil || g.Phase != "planned" {
		return nil, ErrGatewayJournal
	}
	p, w = gatewayInputs(p, w)
	v, err := OpenWorkspaceVerifier(ctx, directory, p, w)
	if err != nil {
		return nil, err
	}
	return &GatewayBootstrap{verifier: v, plan: p, workspace: w, selection: g.Selection}, nil
}

func (b *GatewayBootstrap) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil
	}
	b.closed = true
	return b.verifier.Close()
}

func (b *GatewayBootstrap) VerifyEmpty(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed || b.attempted {
		return ErrGatewayUnresolved
	}
	return b.verifier.VerifyEmpty(ctx)
}

// Initialize reads namespace identities from this process, never from caller
// JSON. Those names let the guest check it actually entered distinct namespaces.
// They are public metadata, not credentials or proof of a no-NIC VM boundary.
func (b *GatewayBootstrap) Initialize(ctx context.Context, g Gateway) error {
	return b.initialize(ctx, g, (*os.File).Sync)
}

func (b *GatewayBootstrap) initialize(ctx context.Context, g Gateway, syncFile func(*os.File) error) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if b.closed || b.attempted || syncFile == nil {
		return ErrGatewayUnresolved
	}
	if g.Validate(b.plan, b.workspace) != nil || g.Selection != b.selection || g.Phase != "start-dispatched" {
		return ErrGatewayJournal
	}
	if err := b.verifier.VerifyEmpty(ctx); err != nil {
		return err
	}
	namespaces := map[string]string{}
	for _, kind := range []string{"net", "pid", "mnt"} {
		name, err := os.Readlink("/proc/self/ns/" + kind)
		if err != nil || !strings.HasPrefix(name, kind+":[") || !strings.HasSuffix(name, "]") {
			return errWorkspaceTree
		}
		number := strings.TrimSuffix(strings.TrimPrefix(name, kind+":["), "]")
		n, err := strconv.ParseUint(number, 10, 64)
		if err != nil || n == 0 || strconv.FormatUint(n, 10) != number {
			return errWorkspaceTree
		}
		namespaces[kind] = name
	}
	raw, err := json.Marshal(namespaces)
	if err != nil {
		return errWorkspaceTree
	}
	raw = append(raw, '\n')
	d := b.verifier.filesystem
	root, err := d.root(b.workspace)
	if err != nil {
		return err
	}
	defer root.Close()
	current := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := d.parentCurrent(); err != nil {
			return err
		}
		r, err := d.root(b.workspace)
		if err != nil {
			return err
		}
		return r.Close()
	}
	if err = current(); err != nil {
		return err
	}
	// Fence the attempt before the first write, including a failed mkdir. All
	// later errors keep the journal and whatever reached storage for inspection.
	b.attempted = true
	refuse := func(err error) error { return errors.Join(ErrGatewayUnresolved, err) }
	if err = unix.Mkdirat(int(root.Fd()), "home", 0700); err != nil {
		return refuse(errWorkspaceTree)
	}
	home, err := workspaceOpen(root, "home")
	if err != nil {
		return refuse(err)
	}
	err = syncFile(home)
	closeErr := home.Close()
	if err != nil || closeErr != nil {
		return refuse(errWorkspaceTree)
	}
	for _, entry := range []struct {
		name string
		data []byte
	}{{"marker", []byte(gatewayBootstrapMarker)}, {"guest-namespaces.json", raw}} {
		if err = current(); err != nil {
			return refuse(err)
		}
		fd, e := unix.Openat2(int(root.Fd()), entry.name, &unix.OpenHow{
			Flags: unix.O_WRONLY | unix.O_CREAT | unix.O_EXCL | unix.O_NOFOLLOW | unix.O_CLOEXEC,
			Mode:  0600, Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_XDEV,
		})
		if e != nil {
			return refuse(errWorkspaceTree)
		}
		f := os.NewFile(uintptr(fd), "private gateway bootstrap metadata")
		n, e := f.Write(entry.data)
		if e == nil && n == len(entry.data) {
			e = syncFile(f)
		} else if e == nil {
			e = errWorkspaceTree
		}
		closeErr = f.Close()
		if e != nil || closeErr != nil {
			return refuse(errWorkspaceTree)
		}
	}
	if err = current(); err != nil {
		return refuse(err)
	}
	if workspaceContents(root, []string{"piers", "home", "marker", "guest-namespaces.json"}) != nil || syncFile(root) != nil {
		return refuse(errWorkspaceTree)
	}
	if err = current(); err != nil {
		return refuse(err)
	}
	return nil
}

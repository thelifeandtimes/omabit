//go:build linux

package projectusage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"golang.org/x/sys/unix"
)

const measurementBasis = "non-atomic metadata census of declared current member mounts/private caches only; inode blocks may share extents; no file contents, event counts, complete project budget, free-space reservation or action authority"

type scanError struct{ reason string }

func (e *scanError) Error() string { return "project allocation unavailable: " + e.reason }
func refused(reason string) error  { return &scanError{reason} }
func Reason(err error) string {
	var typed *scanError
	if errors.As(err, &typed) {
		return typed.reason
	}
	return "unavailable"
}

type inodeKey struct{ dev, inode uint64 }
type fingerprint struct {
	dev, inode, links, mount uint64
	mode, uid, gid           uint32
	size, blocks             int64
	modified, changed        syscall.Timespec
	birth                    string
}

func object(file *os.File) (fingerprint, os.FileInfo, error) {
	info, err := file.Stat()
	if err != nil {
		return fingerprint{}, nil, refused("unavailable")
	}
	s, ok := info.Sys().(*syscall.Stat_t)
	var x unix.Statx_t
	flags := unix.STATX_BTIME | unix.STATX_MNT_ID
	if !ok {
		return fingerprint{}, nil, refused("unavailable")
	}
	if s.Uid != uint32(os.Geteuid()) || s.Nlink == 0 || info.Mode()&os.ModeSymlink != 0 {
		return fingerprint{}, nil, refused("unsafe-storage")
	}
	if unix.Statx(int(file.Fd()), "", unix.AT_EMPTY_PATH, flags, &x) != nil || x.Mask&uint32(flags) != uint32(flags) {
		return fingerprint{}, nil, refused("unavailable")
	}
	if s.Blocks < 0 || uint64(s.Blocks) > ^uint64(0)/512 || s.Size < 0 {
		return fingerprint{}, nil, refused("overflow")
	}
	if !info.IsDir() && !info.Mode().IsRegular() && info.Mode()&(os.ModeSocket|os.ModeNamedPipe) == 0 {
		return fingerprint{}, nil, refused("unsafe-storage")
	}
	return fingerprint{s.Dev, s.Ino, s.Nlink, x.Mnt_id, s.Mode, s.Uid, s.Gid, s.Size, s.Blocks, s.Mtim, s.Ctim, fmt.Sprintf("%d:%d", x.Btime.Sec, x.Btime.Nsec)}, info, nil
}

func (f fingerprint) identity() pierstore.DirectoryIdentity {
	return pierstore.DirectoryIdentity{Device: f.dev, Inode: f.inode, Birth: f.birth}
}

func samePath(root *os.Root, name string, info os.FileInfo) bool {
	after, err := root.Lstat(name)
	return err == nil && os.SameFile(info, after) && after.Mode()&os.ModeSymlink == 0
}

type linkCount struct {
	object   fingerprint
	observed uint64
}
type treeScan struct {
	ctx           context.Context
	remaining     int
	mount, device uint64
	files         map[inodeKey]*linkCount
	directories   map[inodeKey]bool
	totals        Totals
}

func (s *treeScan) count(fp fingerprint, info os.FileInfo) error {
	if s.ctx.Err() != nil {
		return refused("cancelled")
	}
	if s.remaining <= 0 {
		return refused("entry-limit")
	}
	s.remaining--
	if fp.mount != s.mount || fp.dev != s.device {
		return refused("cross-mount")
	}
	value := Totals{}
	key := inodeKey{fp.dev, fp.inode}
	if info.IsDir() {
		if s.directories[key] {
			return refused("cross-mount")
		}
		s.directories[key] = true
		value.Directories = 1
		value.AllocatedBytes = uint64(fp.blocks) * 512
	} else {
		if info.Mode().IsRegular() {
			value.RegularFiles = 1
		} else {
			value.SpecialEntries = 1
		}
		if fp.links > 1 {
			value.HardlinkEntries = 1
		}
		if previous, ok := s.files[key]; ok {
			if previous.object != fp {
				return refused("changed")
			}
			previous.observed++
		} else {
			s.files[key] = &linkCount{fp, 1}
			value.UniqueFileInodes = 1
			value.AllocatedBytes = uint64(fp.blocks) * 512
			if info.Mode().IsRegular() {
				value.LogicalBytes = uint64(fp.size)
			}
		}
	}
	next, err := s.totals.Add(value)
	if err != nil {
		return refused("overflow")
	}
	s.totals = next
	return nil
}

func (s *treeScan) directory(root *os.Root, depth int) error {
	if depth > MaxDepth {
		return refused("depth-limit")
	}
	f, err := root.Open(".")
	if err != nil {
		return refused("unavailable")
	}
	defer f.Close()
	before, info, err := object(f)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return refused("unsafe-storage")
	}
	if err := s.count(before, info); err != nil {
		return err
	}
	entries, err := f.ReadDir(s.remaining + 1)
	if err != nil && err != io.EOF {
		return refused("unavailable")
	}
	if len(entries) > s.remaining {
		return refused("entry-limit")
	}
	for _, entry := range entries {
		if s.ctx.Err() != nil {
			return refused("cancelled")
		}
		name := entry.Name()
		before, err := root.Lstat(name)
		if err != nil {
			return refused("changed")
		}
		if before.Mode()&os.ModeSymlink != 0 {
			return refused("unsafe-storage")
		}
		if before.IsDir() {
			child, err := root.OpenRoot(name)
			if err != nil {
				return refused("unavailable")
			}
			inside, e := child.Stat(".")
			if e != nil || !os.SameFile(before, inside) || !samePath(root, name, before) {
				child.Close()
				return refused("changed")
			}
			err = s.directory(child, depth+1)
			child.Close()
			if err != nil {
				return err
			}
			if !samePath(root, name, before) {
				return refused("changed")
			}
		} else {
			// O_PATH inspects a FIFO/socket without reading, blocking or connecting.
			file, err := root.OpenFile(name, unix.O_PATH|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
			if err != nil {
				return refused("unavailable")
			}
			fp, inside, err := object(file)
			if err != nil {
				file.Close()
				return err
			}
			if !os.SameFile(before, inside) || !samePath(root, name, inside) {
				file.Close()
				return refused("changed")
			}
			err = s.count(fp, inside)
			after, _, readErr := object(file)
			file.Close()
			if err != nil {
				return err
			}
			if readErr != nil || fp != after || !samePath(root, name, inside) {
				return refused("changed")
			}
		}
	}
	after, _, err := object(f)
	if err != nil || before != after || !samePath(root, ".", info) {
		return refused("changed")
	}
	return nil
}

// ScanBoundTree is metadata-only. Its path must come from a separately validated
// runner journal/cache selection; this library does not authorize arbitrary paths.
// A missing root is unknown, never empty, disposable or permission to recreate it.
func ScanBoundTree(ctx context.Context, selected Root, limit int) (RootUsage, int, error) {
	if selected.Validate() != nil || limit < 1 || limit > MaxEntries {
		return RootUsage{}, 0, ErrInvalid
	}
	if ctx.Err() != nil {
		return RootUsage{}, 0, refused("cancelled")
	}
	actual, err := filepath.EvalSymlinks(selected.Path)
	if err != nil {
		return RootUsage{}, 0, refused("unavailable")
	}
	if actual != selected.Path {
		return RootUsage{}, 0, refused("unsafe-storage")
	}
	before, err := os.Lstat(actual)
	if err != nil || !before.IsDir() {
		return RootUsage{}, 0, refused("unavailable")
	}
	if selected.Kind != "member-mount" && before.Mode().Perm() != 0700 {
		return RootUsage{}, 0, refused("unsafe-storage")
	}
	root, err := os.OpenRoot(actual)
	if err != nil {
		return RootUsage{}, 0, refused("unavailable")
	}
	defer root.Close()
	f, err := root.Open(".")
	if err != nil {
		return RootUsage{}, 0, refused("unavailable")
	}
	fp, info, err := object(f)
	f.Close()
	if err != nil {
		return RootUsage{}, 0, err
	}
	if !os.SameFile(before, info) || selected.Kind == "member-mount" && !selected.Expected.SameObject(fp.identity()) {
		return RootUsage{}, 0, refused("changed")
	}
	s := treeScan{ctx: ctx, remaining: limit, mount: fp.mount, device: fp.dev, files: map[inodeKey]*linkCount{}, directories: map[inodeKey]bool{}}
	if err := s.directory(root, 0); err != nil {
		return RootUsage{}, limit - s.remaining, err
	}
	for _, file := range s.files {
		if file.observed != file.object.links {
			return RootUsage{}, limit - s.remaining, refused("external-hardlink")
		}
	}
	current, e1 := os.Lstat(actual)
	resolved, e2 := filepath.EvalSymlinks(actual)
	if e1 != nil || e2 != nil || resolved != actual || !os.SameFile(before, current) {
		return RootUsage{}, limit - s.remaining, refused("changed")
	}
	f, err = root.Open(".")
	if err != nil {
		return RootUsage{}, limit - s.remaining, refused("unavailable")
	}
	after, _, err := object(f)
	f.Close()
	if err != nil || fp != after {
		return RootUsage{}, limit - s.remaining, refused("changed")
	}
	return RootUsage{Root: selected, Observed: fp.identity(), Totals: s.totals}, limit - s.remaining, nil
}

// Measure observes only declared roots, without content/engine reads or writes.
// Unknown refuses all partial totals. Shared/historical exclusions remain visible
// even when the entire declared scope measures successfully.
func Measure(ctx context.Context, binding Binding, roots []Root, limit int) (Observation, error) {
	out := Observation{Version: 1, Binding: binding, Scope: Scope, Status: "unknown", Reason: "unavailable", DeclaredRoots: append([]Root(nil), roots...), ExcludedComponents: Exclusions(), SharedAllowanceStatus: "not-configured", StartedAt: time.Now().UTC(), Basis: measurementBasis}
	out.ReadAt = out.StartedAt
	if out.Validate() != nil || limit < 1 || limit > MaxEntries {
		return Observation{}, ErrInvalid
	}
	remaining := limit
	var usages []RootUsage
	var totals Totals
	for _, root := range roots {
		if remaining <= 0 {
			out.Reason, out.UnknownRootID = "entry-limit", root.ID
			out.ReadAt = time.Now().UTC()
			return out, out.Validate()
		}
		usage, used, err := ScanBoundTree(ctx, root, remaining)
		remaining -= used
		if err != nil {
			out.Reason, out.UnknownRootID = Reason(err), root.ID
			out.ReadAt = time.Now().UTC()
			return out, out.Validate()
		}
		for _, old := range usages {
			if old.Observed.Device == usage.Observed.Device && old.Observed.Inode == usage.Observed.Inode && old.Observed.Birth == usage.Observed.Birth {
				out.Reason, out.UnknownRootID = "unsafe-storage", root.ID
				out.ReadAt = time.Now().UTC()
				return out, out.Validate()
			}
		}
		totals, err = totals.Add(usage.Totals)
		if err != nil {
			out.Reason, out.UnknownRootID = "overflow", root.ID
			out.ReadAt = time.Now().UTC()
			return out, out.Validate()
		}
		usages = append(usages, usage)
	}
	out.ReadAt = time.Now().UTC()
	if ctx.Err() != nil {
		out.Reason = "cancelled"
		return out, out.Validate()
	}
	out.Status, out.Reason, out.Roots, out.Totals = "measured", "", usages, &totals
	return out, out.Validate()
}

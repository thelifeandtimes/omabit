//go:build linux

package hostcompat

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

type downloadPolicy struct {
	stage          string
	aptUID, aptGID uint32
	aptReady       bool
}

var aptWritable = []string{"state", "cache", "archives", "lists", "log", "home", "tmp"}

func aptDownloadIdentity(fsys fs.FS) (uint32, uint32, error) {
	b, err := readTrusted(fsys, "etc/passwd", 1<<20)
	if err != nil {
		return 0, 0, errAPTSandboxUnavailable
	}
	var uid, gid uint32
	found := false
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Split(line, ":")
		if len(fields) == 0 || fields[0] != "_apt" {
			continue
		}
		if found || len(fields) != 7 {
			return 0, 0, errAPTSandboxUnavailable
		}
		u, ue := strconv.ParseUint(fields[2], 10, 32)
		g, ge := strconv.ParseUint(fields[3], 10, 32)
		if ue != nil || ge != nil || u == 0 || g == 0 || u == uint64(^uint32(0)) || g == uint64(^uint32(0)) {
			return 0, 0, errAPTSandboxUnavailable
		}
		uid, gid, found = uint32(u), uint32(g), true
	}
	if !found {
		return 0, 0, errAPTSandboxUnavailable
	}
	return uid, gid, nil
}

func (p *downloadPolicy) aptArea(name string) bool {
	if p == nil || !p.aptReady {
		return false
	}
	for _, dir := range aptWritable {
		base := p.stage + "/apt/" + dir
		if name == base || strings.HasPrefix(name, base+"/") {
			return true
		}
	}
	return false
}
func (p *downloadPolicy) allows(name string, info fs.FileInfo) bool {
	if info == nil || info.Mode()&fs.ModeSymlink != 0 || info.Mode().Perm()&0022 != 0 {
		return false
	}
	s, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return false
	}
	return s.Uid == 0 || (p.aptArea(name) && s.Uid == p.aptUID)
}
func (p *downloadPolicy) checkPath(root rootFiles, name string) error {
	if !fs.ValidPath(name) || (name != p.stage && !strings.HasPrefix(name, p.stage+"/")) {
		return errFetch
	}
	prefix := ""
	for _, part := range strings.Split(name, "/") {
		prefix = path.Join(prefix, part)
		info, err := root.Lstat(prefix)
		if err != nil || !p.allows(prefix, info) {
			return errFetch
		}
	}
	return nil
}

func prepareAPTSandbox(root *os.Root, p *downloadPolicy, fsys fs.FS, base string) error {
	uid, gid, err := aptDownloadIdentity(fsys)
	if err != nil {
		return err
	}
	if base != p.stage+"/apt" {
		return errFetch
	}
	// Read-only traversal through root-owned parents. The only writable
	// locations granted to the existing account are these acquisition subtrees.
	for _, dir := range []string{p.stage, base} {
		if err := root.Chmod(dir, 0711); err != nil {
			return errFetch
		}
	}
	for _, dir := range []string{"empty-config", "empty-sources", "empty-trusted"} {
		if err := root.Chmod(base+"/"+dir, 0755); err != nil {
			return errFetch
		}
	}
	for _, name := range []string{"apt.conf", "sources.sources"} {
		if err := root.Chmod(base+"/"+name, 0644); err != nil {
			return errFetch
		}
	}
	for _, dir := range append(append([]string{}, aptWritable...), "lists/partial") {
		name := base + "/" + dir
		if err := root.Chown(name, int(uid), int(gid)); err != nil {
			return errFetch
		}
		if err := root.Chmod(name, 0700); err != nil {
			return errFetch
		}
	}
	p.aptUID, p.aptGID, p.aptReady = uid, gid, true
	return nil
}

// promoteDownload reads only a direct child of a designated acquisition
// directory, via a pinned directory fd and O_NOFOLLOW|O_NONBLOCK. It never
// blesses an _apt-owned object in place. Stable bytes are copied, hashed, synced,
// atomically published and rehashed inside an inaccessible root-owned directory.
// Publisher authenticity is still checked by the normal verifier afterwards.
func promoteDownload(ctx context.Context, root rootFiles, p *downloadPolicy, source, destination string, limit int64, fsys fs.FS) error {
	parent := path.Dir(source)
	if !p.aptReady || (parent != p.stage+"/apt/lists" && parent != p.stage+"/apt/archives") || path.Base(source) == "." || !fs.ValidPath(source) {
		return errFetch
	}
	if p.checkPath(root, parent) != nil {
		return errFetch
	}
	dir, err := root.Open(parent)
	if err != nil {
		return errFetch
	}
	defer dir.Close()
	parentBefore, err := dir.Stat()
	if err != nil {
		return errFetch
	}
	fd, err := unix.Openat(int(dir.Fd()), path.Base(source), unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if err != nil {
		return errFetch
	}
	sourceFile := os.NewFile(uintptr(fd), source)
	defer sourceFile.Close()
	before, err := sourceFile.Stat()
	if err != nil {
		return errFetch
	}
	st, ok := before.Sys().(*syscall.Stat_t)
	if !ok || st.Uid != p.aptUID || st.Nlink != 1 || !before.Mode().IsRegular() || before.Mode().Perm()&0022 != 0 || before.Size() < 0 || before.Size() > limit {
		return errFetch
	}
	if chargeBytes(fsys, "read", before.Size()+1) != nil {
		return errBudget
	}
	if !strings.HasPrefix(destination, p.stage+"/apt-evidence/") || !fs.ValidPath(destination) {
		return errFetch
	}
	resolved, destDir, err := resolve(root.FS(), path.Dir(destination))
	if err != nil || resolved != path.Dir(destination) || !destDir.IsDir() {
		return errFetch
	}
	if info, err := root.Lstat(destination); err == nil {
		if !trustedInfo(info) || !info.Mode().IsRegular() {
			return errFetch
		}
	} else if !os.IsNotExist(err) {
		return errFetch
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return errFetch
	}
	tmp := path.Dir(destination) + "/.seal-" + hex.EncodeToString(nonce[:])
	out, err := root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return errFetch
	}
	defer root.Remove(tmp)
	h := sha256.New()
	reader := &contextReader{ctx: ctx, r: io.LimitReader(sourceFile, before.Size()+1)}
	n, copyErr := io.Copy(io.MultiWriter(out, h), reader)
	after, se := sourceFile.Stat()
	current, ce := root.Lstat(source)
	parentAfter, pe := root.Lstat(parent)
	if copyErr != nil || ctx.Err() != nil || n != before.Size() || se != nil || ce != nil || pe != nil || !sameFileState(before, after) || !sameFileState(before, current) || !os.SameFile(parentBefore, parentAfter) || p.checkPath(root, parent) != nil {
		out.Close()
		return errFetch
	}
	if err := out.Sync(); err != nil {
		out.Close()
		return errFetch
	}
	if err := out.Close(); err != nil {
		return errFetch
	}
	if err := root.Rename(tmp, destination); err != nil {
		return errFetch
	}
	syncDir, err := root.Open(path.Dir(destination))
	if err != nil {
		return errFetch
	}
	err = syncDir.Sync()
	syncDir.Close()
	if err != nil {
		return errFetch
	}
	actual, err := hashTrusted(fsys, destination, limit)
	if err != nil || actual != hex.EncodeToString(h.Sum(nil)) {
		return errFetch
	}
	return nil
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r *contextReader) Read(b []byte) (int, error) {
	if r.ctx.Err() != nil {
		return 0, r.ctx.Err()
	}
	return r.r.Read(b)
}

func acquisitionEntries(root rootFiles, p *downloadPolicy, dir string) ([]fs.DirEntry, error) {
	if p.checkPath(root, dir) != nil {
		return nil, errFetch
	}
	f, err := root.Open(dir)
	if err != nil {
		return nil, errFetch
	}
	defer f.Close()
	entries, err := f.ReadDir(maxFetchFiles + 1)
	if (err != nil && err != io.EOF) || len(entries) > maxFetchFiles {
		return nil, errFetch
	}
	return entries, nil
}

func sealAPTDownloads(ctx context.Context, root *os.Root, p *downloadPolicy, v *verifier, base string, owner nativePackage, indexes bool) (string, error) {
	if base != p.stage+"/apt" || !p.aptReady {
		return "", errFetch
	}
	sealed := p.stage + "/apt-evidence"
	for _, dir := range []string{sealed, sealed + "/lists", sealed + "/archives"} {
		if err := root.Mkdir(dir, 0700); err != nil && !os.IsExist(err) {
			return "", errFetch
		}
		actual, info, err := resolve(root.FS(), dir)
		if err != nil || actual != dir || !info.IsDir() {
			return "", errFetch
		}
	}
	if indexes {
		sources, _, err := aptSources(v.fsys)
		if err != nil {
			return "", err
		}
		entries, err := acquisitionEntries(root, p, base+"/lists")
		if err != nil {
			return "", err
		}
		copied := 0
		for _, entry := range entries {
			name := entry.Name()
			allowed := false
			limit := int64(maxIndex)
			for _, source := range sources {
				if name == path.Base(source.releasePath()) {
					allowed = true
					limit = 4 << 20
				}
				if source.permitsRetainedIndex(name, owner.arch) {
					allowed = true
				}
			}
			if !allowed {
				continue
			}
			if promoteDownload(ctx, root, p, base+"/lists/"+name, sealed+"/lists/"+name, limit, v.fsys) != nil {
				return "", errFetch
			}
			copied++
		}
		if copied == 0 {
			return "", errFetch
		}
	} else {
		entries, err := acquisitionEntries(root, p, base+"/archives")
		if err != nil {
			return "", err
		}
		copied := 0
		for _, entry := range entries {
			name := entry.Name()
			if !strings.HasPrefix(name, owner.name+"_") || !strings.HasSuffix(name, "_"+owner.arch+".deb") {
				continue
			}
			if copied >= 8 || promoteDownload(ctx, root, p, base+"/archives/"+name, sealed+"/archives/"+name, maxArtifact, v.fsys) != nil {
				return "", errFetch
			}
			copied++
		}
		if copied == 0 {
			return "", errFetch
		}
	}
	return sealed, nil
}

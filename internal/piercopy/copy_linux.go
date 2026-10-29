//go:build linux

// Package piercopy copies a stopped, caller-owned tree into empty private storage.
// It does not infer that a pier is stopped; the runtime coordinator proves that.
package piercopy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path"
	"reflect"
	"sort"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

const MaxEntries = 8192
const MaxBytes int64 = pierstore.MaxLocalPierBytes
const MaxDepth = 32

type Summary struct {
	Digest string `json:"digest"`
	Files  int    `json:"files"`
	Bytes  int64  `json:"bytes"`
}

type entry struct {
	Name      string `json:"name"`
	Directory bool   `json:"directory"`
	Size      int64  `json:"size"`
	Hash      string `json:"hash,omitempty"`
	info      os.FileInfo
}

func stable(a, b os.FileInfo) bool {
	x, xok := a.Sys().(*syscall.Stat_t)
	y, yok := b.Sys().(*syscall.Stat_t)
	return xok && yok && x.Dev == y.Dev && x.Ino == y.Ino && x.Uid == y.Uid && x.Nlink == y.Nlink && x.Size == y.Size && x.Mtim == y.Mtim && x.Ctim == y.Ctim && a.Mode() == b.Mode()
}

func inventory(ctx context.Context, root *os.Root) ([]entry, error) {
	var out []entry
	var total int64
	type linkKey struct{ device, inode uint64 }
	type links struct{ expected, seen uint64 }
	docked := map[linkKey]links{}
	base, err := root.Lstat(".")
	if err != nil {
		return nil, err
	}
	owner, ok := base.Sys().(*syscall.Stat_t)
	if !ok || !base.IsDir() || owner.Uid != uint32(os.Geteuid()) {
		return nil, errors.New("copy root must be an owned directory")
	}
	var walk func(string, int) error
	walk = func(name string, depth int) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if depth > MaxDepth || len(name) > 4096 || !utf8.ValidString(name) || strings.ContainsAny(name, "\x00\r\n") || len(out) >= MaxEntries {
			return errors.New("pier tree entry, path or depth limit exceeded")
		}
		info, err := root.Lstat(name)
		if err != nil {
			return errors.New("pier tree metadata unavailable")
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok || st.Uid != owner.Uid || st.Dev != owner.Dev || info.Mode()&(os.ModeSymlink|os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
			return errors.New("pier tree has unsafe ownership, device or file type")
		}
		e := entry{Name: name, Directory: info.IsDir(), info: info}
		if !info.IsDir() {
			if !info.Mode().IsRegular() || st.Nlink != 1 && !pierstore.InertDockedFile(name) || info.Size() < 0 || info.Size() > MaxBytes-total {
				return errors.New("pier tree has shared/special files or exceeds byte limit")
			}
			if st.Nlink > 1 {
				key := linkKey{st.Dev, st.Ino}
				old := docked[key]
				if old.expected != 0 && old.expected != st.Nlink {
					return errors.New("docked executable links changed")
				}
				docked[key] = links{st.Nlink, old.seen + 1}
			}
			e.Size = info.Size()
			total += e.Size
		}
		out = append(out, e)
		if info.IsDir() {
			f, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
			if err != nil {
				return errors.New("pier directory unavailable")
			}
			actual, err := f.Stat()
			if err != nil || !stable(info, actual) {
				f.Close()
				return errors.New("pier directory changed")
			}
			children, err := f.ReadDir(MaxEntries - len(out) + 1)
			f.Close()
			if (err != nil && !errors.Is(err, io.EOF)) || len(children) > MaxEntries-len(out) {
				return errors.New("pier directory exceeds entry limit")
			}
			sort.Slice(children, func(i, j int) bool { return children[i].Name() < children[j].Name() })
			for _, child := range children {
				if err := walk(path.Join(name, child.Name()), depth+1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(".", 0); err != nil {
		return nil, err
	}
	for _, links := range docked {
		if links.seen != links.expected {
			return nil, errors.New("docked executable shares links outside the pier")
		}
	}
	return out, nil
}

func digest(entries []entry) Summary {
	h := sha256.New()
	s := Summary{}
	for _, e := range entries {
		data, _ := json.Marshal(e)
		h.Write(data)
		h.Write([]byte{'\n'})
		if !e.Directory {
			s.Files++
			s.Bytes += e.Size
		}
	}
	s.Digest = hex.EncodeToString(h.Sum(nil))
	return s
}

// transfer reads an exact bounded file, optionally writing it, and verifies both
// its open descriptor and current path still match the original metadata.
func transfer(ctx context.Context, root *os.Root, e entry, to io.Writer) (string, error) {
	f, err := root.OpenFile(e.Name, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return "", errors.New("source file unavailable")
	}
	defer f.Close()
	actual, err := f.Stat()
	if err != nil || !stable(e.info, actual) {
		return "", errors.New("source file changed before read")
	}
	h := sha256.New()
	writer := io.Writer(h)
	if to != nil {
		writer = io.MultiWriter(to, h)
	}
	buf := make([]byte, 256<<10)
	remaining := e.Size
	for remaining > 0 {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		n, err := f.Read(buf[:min(int64(len(buf)), remaining)])
		if n > 0 {
			if written, err := writer.Write(buf[:n]); err != nil || written != n {
				return "", errors.New("pier copy write failed")
			}
			remaining -= int64(n)
		}
		if err != nil || n == 0 {
			return "", errors.New("source file truncated during copy")
		}
	}
	var extra [1]byte
	n, err := f.Read(extra[:])
	if n != 0 || err != io.EOF {
		return "", errors.New("source file grew during copy")
	}
	actual, err = f.Stat()
	if err != nil || !stable(e.info, actual) {
		return "", errors.New("source file changed during read")
	}
	actual, err = root.Lstat(e.Name)
	if err != nil || !stable(e.info, actual) {
		return "", errors.New("source path changed during read")
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func unchanged(root *os.Root, entries []entry) error {
	for _, e := range entries {
		actual, err := root.Lstat(e.Name)
		if err != nil || !stable(e.info, actual) {
			return errors.New("pier tree changed during traversal")
		}
	}
	return nil
}

// Measure hashes all regular-file contents and the sorted relative tree layout.
// It is a consistency fingerprint, not proof of application or backup integrity.
func Measure(ctx context.Context, root *os.Root) (Summary, error) {
	entries, err := inventory(ctx, root)
	if err != nil {
		return Summary{}, err
	}
	for i, e := range entries {
		if !e.Directory {
			entries[i].Hash, err = transfer(ctx, root, e, nil)
			if err != nil {
				return Summary{}, err
			}
		}
	}
	if err := unchanged(root, entries); err != nil {
		return Summary{}, err
	}
	return digest(entries), nil
}

// Copy does not overwrite or remove anything. Failure retains the partial target
// for the caller's quarantine/recovery workflow. Permissions are normalized to
// owner-only. The caller journals intent before creating the destination root.
func Copy(ctx context.Context, source, destination *os.Root, progress func(int, int64) error, verifying ...func() error) (Summary, error) {
	if progress == nil {
		return Summary{}, errors.New("copy requires durable progress callback")
	}
	dst, err := inventory(ctx, destination)
	if err != nil || len(dst) != 1 {
		return Summary{}, errors.New("copy destination must be an empty owned directory")
	}
	if dst[0].info.Mode().Perm() != 0700 {
		return Summary{}, errors.New("copy destination must be private")
	}
	entries, err := inventory(ctx, source)
	if err != nil {
		return Summary{}, err
	}
	if os.SameFile(entries[0].info, dst[0].info) {
		return Summary{}, errors.New("source and target must be distinct")
	}
	files := 0
	var copied int64
	for i, e := range entries {
		if err := ctx.Err(); err != nil {
			return Summary{}, err
		}
		if e.Name == "." {
			continue
		}
		if e.Directory {
			if err := destination.Mkdir(e.Name, 0700); err != nil {
				return Summary{}, errors.New("cannot create private target directory")
			}
			continue
		}
		f, err := destination.OpenFile(e.Name, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0600)
		if err != nil {
			return Summary{}, errors.New("target file must be fresh")
		}
		lastReport := time.Now()
		writer := &progressWriter{Writer: f, notify: func(written int64) error {
			if time.Since(lastReport) < 250*time.Millisecond {
				return nil
			}
			lastReport = time.Now()
			return progress(files, copied+written)
		}}
		entries[i].Hash, err = transfer(ctx, source, e, writer)
		if err == nil {
			err = f.Sync()
		}
		closeErr := f.Close()
		if err != nil {
			return Summary{}, err
		}
		if closeErr != nil {
			return Summary{}, errors.New("target file close failed")
		}
		files++
		copied += e.Size
		if err := progress(files, copied); err != nil {
			return Summary{}, err
		}
	}
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].Directory {
			f, err := destination.Open(entries[i].Name)
			if err != nil {
				return Summary{}, err
			}
			err = f.Sync()
			f.Close()
			if err != nil {
				return Summary{}, errors.New("target directory sync failed")
			}
		}
	}
	if err := unchanged(source, entries); err != nil {
		return Summary{}, err
	}
	if len(verifying) > 0 && verifying[0] != nil {
		if err := verifying[0](); err != nil {
			return Summary{}, err
		}
	}
	expected := digest(entries)
	actual, err := Measure(ctx, destination)
	if err != nil || !reflect.DeepEqual(actual, expected) {
		return Summary{}, errors.New("copied pier content verification failed")
	}
	// Re-read source content as well; content fingerprints must agree on both sides.
	still, err := Measure(ctx, source)
	if err != nil || still != expected {
		return Summary{}, errors.New("source content changed during copy")
	}
	return expected, nil
}

type progressWriter struct {
	io.Writer
	written int64
	notify  func(int64) error
}

func (w *progressWriter) Write(p []byte) (int, error) {
	n, err := w.Writer.Write(p)
	w.written += int64(n)
	if err == nil {
		err = w.notify(w.written)
	}
	return n, err
}

//go:build linux

package hostcompat

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path"
	"strings"
	"syscall"
	"time"
)

// Production uses *os.Root. The narrow interface also permits testing real
// atomic filesystem operations with injected ownership in an unprivileged temp
// root; no production entry point accepts an alternate backend.
type rootFiles interface {
	FS() fs.FS
	Lstat(string) (os.FileInfo, error)
	OpenFile(string, int, fs.FileMode) (*os.File, error)
	Open(string) (*os.File, error)
	Rename(string, string) error
	Remove(string) error
}

type budgetRoot struct {
	rootFiles
	fsys fs.FS
}

func (r budgetRoot) FS() fs.FS { return r.fsys }

func prepareVerificationDirectory(root *os.Root) error {
	return prepareVerificationDirectories(root)
}

type directoryRoot interface {
	rootFiles
	Mkdir(string, fs.FileMode) error
}

func prepareVerificationDirectories(root directoryRoot) error {
	for _, name := range []string{"var", "var/lib", "var/lib/omabit", "var/lib/omabit/hostcompat", "var/lib/omabit/hostcompat/evidence-v2"} {
		created := false
		if _, err := root.Lstat(name); os.IsNotExist(err) {
			if name == "var" || name == "var/lib" {
				return errEvidence
			}
			mode := fs.FileMode(0755)
			if strings.HasSuffix(name, "/evidence-v2") {
				mode = 0700
			}
			if err := root.Mkdir(name, mode); err != nil {
				return errEvidence
			}
			created = true
		}
		resolved, info, err := resolve(root.FS(), name)
		if err != nil || resolved != name || !info.IsDir() {
			return errEvidence
		}
		// Exact application paths only. Never chmod /var, /var/lib, aliases,
		// foreign-owned directories, or an unrelated preexisting omabit tree.
		mode := fs.FileMode(0755)
		if strings.HasSuffix(name, "/evidence-v2") {
			mode = 0700
		}
		owned := created || knownVerificationDirectory(root, name)
		if name == "var/lib/omabit" || name == "var/lib/omabit/hostcompat" || name == "var/lib/omabit/hostcompat/evidence-v2" {
			if owned {
				d, err := root.Open(name)
				if err != nil {
					return errEvidence
				}
				before, err := d.Stat()
				if err != nil || !sameFileObject(info, before) {
					d.Close()
					return errEvidence
				}
				owner, ok := before.Sys().(*syscall.Stat_t)
				if !ok || owner.Uid != uint32(os.Geteuid()) || before.Mode().Perm()&0022 != 0 {
					d.Close()
					return errEvidence
				}
				if err := d.Chmod(mode); err != nil {
					d.Close()
					return errEvidence
				}
				after, err := d.Stat()
				if err != nil || after.Mode().Perm() != mode || !after.IsDir() {
					d.Close()
					return errEvidence
				}
				if err := d.Sync(); err != nil {
					d.Close()
					return errEvidence
				}
				d.Close()
			} else if (name != "var/lib/omabit/hostcompat/evidence-v2" && info.Mode().Perm()&0055 != 0055) || (name == "var/lib/omabit/hostcompat/evidence-v2" && info.Mode().Perm() != 0700) {
				return errEvidence
			}
		}
	}
	return nil
}

// Recovery of an earlier restrictive-umask generation is limited to the exact
// root-owned application subtree with a readable, valid versioned receipt and
// no unrelated entries at its public parent boundaries.
func knownVerificationDirectory(root rootFiles, name string) bool {
	if name != "var/lib/omabit" && name != "var/lib/omabit/hostcompat" && name != "var/lib/omabit/hostcompat/evidence-v2" {
		return false
	}
	b, err := readTrusted(root.FS(), receiptPath, 1<<20)
	if err != nil {
		return false
	}
	var set receiptSet
	if decodeStrict(b, &set) != nil || set.SchemaVersion != 2 || set.PolicyID != PolicyID {
		return false
	}
	if name == "var/lib/omabit" {
		entries, err := readDir(root.FS(), name)
		return err == nil && len(entries) == 1 && entries[0].Name() == "hostcompat"
	}
	if name == "var/lib/omabit/hostcompat" {
		entries, err := readDir(root.FS(), name)
		if err != nil {
			return false
		}
		for _, entry := range entries {
			switch entry.Name() {
			case "receipts-v2.json", "receipts-v1.json", ".refresh.lock", "evidence-v2":
			default:
				return false
			}
		}
	}
	return true
}

func lockRefresh(ctx context.Context, root *os.Root) (func(), error) {
	name := "var/lib/omabit/hostcompat/.refresh.lock"
	f, err := root.OpenFile(name, os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, errEvidence
	}
	info, err := f.Stat()
	if err != nil || !trustedInfo(info) || !info.Mode().IsRegular() {
		f.Close()
		return nil, errEvidence
	}
	for {
		if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err == nil {
			return func() { syscall.Flock(int(f.Fd()), syscall.LOCK_UN); f.Close() }, nil
		} else if err != syscall.EWOULDBLOCK && err != syscall.EAGAIN {
			f.Close()
			return nil, errEvidence
		}
		select {
		case <-ctx.Done():
			f.Close()
			return nil, errEvidence
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// atomicRootFile publishes only beneath a verified application-owned directory.
// Existing aliases, writable files and non-regular objects are never overwritten.
func atomicRootFile(root rootFiles, name string, data []byte, mode fs.FileMode) error {
	if mode != 0600 && mode != 0644 {
		return errEvidence
	}
	dir := path.Dir(name)
	resolved, info, err := resolve(root.FS(), dir)
	if err != nil || resolved != dir || !info.IsDir() {
		return errEvidence
	}
	if info, err := root.Lstat(name); err == nil {
		if !trustedInfo(info) || !info.Mode().IsRegular() {
			return errEvidence
		}
	} else if !os.IsNotExist(err) {
		return errEvidence
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return errEvidence
	}
	tmp := dir + "/.publish-" + hex.EncodeToString(nonce[:])
	f, err := root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, mode)
	if err != nil {
		return errEvidence
	}
	defer root.Remove(tmp)
	// Creation modes are filtered by the hook's umask. Set and inspect the
	// descriptor mode explicitly before any sync or atomic publication.
	if err := f.Chmod(mode); err != nil {
		f.Close()
		return errEvidence
	}
	if info, err := f.Stat(); err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != mode {
		f.Close()
		return errEvidence
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return errEvidence
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return errEvidence
	}
	if err := f.Close(); err != nil {
		return errEvidence
	}
	if err := root.Rename(tmp, name); err != nil {
		return errEvidence
	}
	d, err := root.Open(dir)
	if err != nil {
		return errEvidence
	}
	defer d.Close()
	return d.Sync()
}

func cacheStore(root rootFiles) func(context.Context, []byte, string) (string, error) {
	return func(ctx context.Context, b []byte, suffix string) (string, error) {
		if ctx.Err() != nil || len(b) == 0 || len(b) > maxArtifact {
			return "", errEvidence
		}
		switch suffix {
		case ".gpg", ".asc", ".sig", ".lz4", ".xz", ".gz", ".zst", ".deb":
		default:
			suffix = ".evidence"
		}
		h := sha256.Sum256(b)
		hash := hex.EncodeToString(h[:])
		name := strings.TrimPrefix(VerificationCacheDirectory, "/") + "/" + hash + suffix
		if _, err := root.Lstat(name); err == nil {
			existing, err := hashTrusted(root.FS(), name, maxArtifact)
			if err != nil || existing != hash {
				return "", errEvidence
			}
			return name, nil
		} else if !os.IsNotExist(err) {
			return "", errEvidence
		}
		if err := atomicRootFile(root, name, b, 0600); err != nil {
			return "", err
		}
		actual, err := hashTrusted(root.FS(), name, maxArtifact)
		if err != nil || actual != hash {
			return "", errEvidence
		}
		return name, nil
	}
}

// Prune only recognizable, unreferenced blobs inside our own dedicated cache,
// after the new receipt is durable. Recovery proofs for unauthorized roles count
// as references. OS package caches and unrelated files are never touched.
func pruneVerificationCache(root rootFiles, set receiptSet) {
	live := map[string]bool{}
	for _, rec := range set.Packages {
		for _, item := range rec.Retained {
			live[item.Path] = true
		}
	}
	dir := strings.TrimPrefix(VerificationCacheDirectory, "/")
	entries, err := readDir(root.FS(), dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if len(name) < 65 || name[64] != '.' {
			continue
		}
		if _, err := hex.DecodeString(name[:64]); err != nil {
			continue
		}
		full := dir + "/" + name
		if live[full] {
			continue
		}
		info, err := root.Lstat(full)
		if err != nil || !trustedInfo(info) || !info.Mode().IsRegular() {
			continue
		}
		root.Remove(full)
	}
}

package hostcompat

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"path"
	"strings"
)

const maxMetadata = 32 << 20
const maxIndex = 512 << 20
const maxMetadataTotal = 64 << 20
const maxBinary = 256 << 20
const maxArtifact = 512 << 20
const maxEntries = 16384

var errEvidence = errors.New("local evidence is missing, unsafe, malformed or exceeds limits")

// resolve checks every ancestor and follows at most sixteen links within the
// injected root. A filesystem without Unix ownership evidence is not trusted.
func resolve(fsys fs.FS, name string) (string, fs.FileInfo, error) {
	name = strings.TrimPrefix(name, "/")
	if !fs.ValidPath(name) {
		return "", nil, errEvidence
	}
	root, err := fs.Stat(fsys, ".")
	if err != nil || !trustedInfo(root) || !root.IsDir() {
		return "", nil, errEvidence
	}
	for links := 0; links <= 16; links++ {
		parts := strings.Split(name, "/")
		prefix := ""
		restart := false
		for i, part := range parts {
			prefix = path.Join(prefix, part)
			info, err := fs.Lstat(fsys, prefix)
			if err != nil || !trustedInfo(info) {
				return "", nil, errEvidence
			}
			if info.Mode()&fs.ModeSymlink != 0 {
				target, err := fs.ReadLink(fsys, prefix)
				if err != nil || len(target) > 4096 {
					return "", nil, errEvidence
				}
				if strings.HasPrefix(target, "/") {
					target = strings.TrimPrefix(target, "/")
				} else {
					target = path.Join(path.Dir(prefix), target)
				}
				name = path.Join(target, path.Join(parts[i+1:]...))
				if !fs.ValidPath(name) {
					return "", nil, errEvidence
				}
				restart = true
				break
			}
			if i < len(parts)-1 && !info.IsDir() {
				return "", nil, errEvidence
			}
			if i == len(parts)-1 {
				return prefix, info, nil
			}
		}
		if !restart {
			break
		}
	}
	return "", nil, errEvidence
}

func openTrusted(fsys fs.FS, name string, limit int64) (fs.File, error) {
	resolved, info, err := resolve(fsys, name)
	if err != nil || !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > limit {
		return nil, errEvidence
	}
	f, err := fsys.Open(resolved)
	if err != nil {
		return nil, errEvidence
	}
	after, err := f.Stat()
	if err != nil || !trustedInfo(after) || !after.Mode().IsRegular() || !sameFileState(info, after) {
		f.Close()
		return nil, errEvidence
	}
	return f, nil
}

func readTrusted(fsys fs.FS, name string, limit int64) ([]byte, error) {
	f, err := openTrusted(fsys, name, limit)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	before, err := f.Stat()
	if err != nil {
		return nil, errEvidence
	}
	kind := "read"
	if limit <= maxMetadata {
		kind = "metadata"
	}
	if chargeBytes(fsys, kind, before.Size()+1) != nil {
		return nil, errBudget
	}
	b, err := io.ReadAll(io.LimitReader(f, before.Size()+1))
	if err != nil || int64(len(b)) > limit || int64(len(b)) != before.Size() || !stableRead(fsys, name, f, before) {
		return nil, errEvidence
	}
	return b, nil
}

func hashTrusted(fsys fs.FS, name string, limit int64) (string, error) {
	h, _, err := hashTrustedState(fsys, name, limit)
	return h, err
}

func hashTrustedState(fsys fs.FS, name string, limit int64) (string, fs.FileInfo, error) {
	f, err := openTrusted(fsys, name, limit)
	if err != nil {
		return "", nil, err
	}
	defer f.Close()
	before, err := f.Stat()
	if err != nil {
		return "", nil, errEvidence
	}
	stamp, cacheable := stampFor(before)
	var cache *fileDigestCache
	if w := workFor(fsys); w != nil {
		cache = w.cache
	}
	if cacheable && cache != nil {
		if digest, ok := cache.get(stamp); ok {
			if !stableRead(fsys, name, f, before) {
				return "", nil, errEvidence
			}
			return digest, before, nil
		}
	}
	if chargeBytes(fsys, "hash", before.Size()+1) != nil {
		return "", nil, errBudget
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, before.Size()+1))
	if err != nil || n > limit || n != before.Size() || !stableRead(fsys, name, f, before) {
		return "", nil, errEvidence
	}
	digest := hex.EncodeToString(h.Sum(nil))
	if cacheable && cache != nil {
		cache.put(stamp, digest)
	}
	return digest, before, nil
}

func stableRead(fsys fs.FS, name string, f fs.File, before fs.FileInfo) bool {
	after, err := f.Stat()
	if err != nil || !sameFileState(before, after) {
		return false
	}
	_, current, err := resolve(fsys, name)
	return err == nil && sameFileState(before, current)
}

func readDir(fsys fs.FS, name string) ([]fs.DirEntry, error) {
	resolved, info, err := resolve(fsys, name)
	if err != nil || !info.IsDir() {
		return nil, errEvidence
	}
	f, err := fsys.Open(resolved)
	if err != nil {
		return nil, errEvidence
	}
	defer f.Close()
	d, ok := f.(fs.ReadDirFile)
	if !ok {
		return nil, errEvidence
	}
	entries, err := d.ReadDir(maxEntries + 1)
	if (err != nil && err != io.EOF) || len(entries) > maxEntries {
		return nil, errEvidence
	}
	if chargeCount(fsys, "entry", len(entries)) != nil {
		return nil, errBudget
	}
	return entries, nil
}

func token(s string) bool {
	if s == "" || len(s) > 256 {
		return false
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || strings.ContainsRune("._+~:-", c)) {
			return false
		}
	}
	return true
}

package fleet

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func ReadJSON(path string, v any) error {
	b, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	return json.Unmarshal(b, v)
}

// AtomicJSON commits via fsync + same-directory rename + directory fsync. The
// daemon holds a lifetime flock; requests never modify this file themselves.
func AtomicJSON(path string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	b = append(b, '\n')
	dir := filepath.Dir(path)
	if e = os.MkdirAll(dir, 0700); e != nil {
		return e
	}
	f, e := os.CreateTemp(dir, ".commit-")
	if e != nil {
		return e
	}
	name := f.Name()
	defer os.Remove(name)
	if e = f.Chmod(0600); e == nil {
		_, e = f.Write(b)
	}
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e != nil {
		return e
	}
	if ce != nil {
		return ce
	}
	if e = os.Rename(name, path); e != nil {
		return e
	}
	d, e := os.Open(dir)
	if e != nil {
		return e
	}
	defer d.Close()
	return d.Sync()
}
func AcquireLock(path string) (*os.File, error) {
	f, e := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return nil, e
	}
	if e = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		f.Close()
		return nil, fmt.Errorf("another host service owns this state directory: %w", e)
	}
	return f, nil
}

// CheckPath rejects symlink ancestors beneath root. The pilot assumes the owning
// Linux account is trusted, not an adversarial same-UID process racing renames.
func CheckPath(root, relative string) (string, error) {
	if filepath.IsAbs(relative) || relative == "." || relative == "" {
		return "", fmt.Errorf("unsafe relative path")
	}
	clean := filepath.Clean(relative)
	if clean == ".." || len(clean) > 3 && clean[:3] == "../" {
		return "", fmt.Errorf("path escapes data root")
	}
	// Resolve the configured root once; it may deliberately be a separate disk.
	abs, e := filepath.Abs(root)
	if e != nil {
		return "", e
	}
	p := abs
	for _, part := range splitPath(clean) {
		p = filepath.Join(p, part)
		st, e := os.Lstat(p)
		if os.IsNotExist(e) {
			continue
		}
		if e != nil {
			return "", e
		}
		if st.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("refusing symlink: %s", p)
		}
	}
	return filepath.Join(abs, clean), nil
}
func splitPath(p string) []string {
	var a []string
	for p != "." && p != "" {
		d, b := filepath.Split(p)
		if b != "" {
			a = append([]string{b}, a...)
		}
		p = filepath.Clean(d)
	}
	return a
}

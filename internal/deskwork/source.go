// Package deskwork contains bounded, typed desk development operations.
package deskwork

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
)

const (
	ManifestVersion       = 1
	MaxFiles              = 8192
	MaxFileBytes    int64 = 32 << 20
	MaxTreeBytes    int64 = 256 << 20
)

// Files are relative Unix mount paths. No hidden files or host metadata belong
// in a Clay source payload. The terminal extension is the Clay mark.
var segment = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$`)
var mark = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)
var digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

func ValidPath(name string) bool {
	if len(name) > 1024 || path.Clean(name) != name || strings.HasPrefix(name, "/") || strings.Contains(name, "\\") {
		return false
	}
	parts := strings.Split(name, "/")
	if len(parts) > 16 {
		return false
	}
	for _, p := range parts {
		if !segment.MatchString(p) {
			return false
		}
	}
	base := parts[len(parts)-1]
	i := strings.LastIndexByte(base, '.')
	return i > 0 && mark.MatchString(base[i+1:])
}

type File struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}
type Manifest struct {
	Version int    `json:"version"`
	Files   []File `json:"files"`
	Digest  string `json:"digest"`
}

func treeDigest(files []File) string {
	b, _ := json.Marshal(files)
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}
func NewManifest(files []File) (Manifest, error) {
	copyFiles := append([]File{}, files...)
	sort.Slice(copyFiles, func(i, j int) bool { return copyFiles[i].Path < copyFiles[j].Path })
	m := Manifest{ManifestVersion, copyFiles, treeDigest(copyFiles)}
	return m, m.Validate()
}
func (m Manifest) Validate() error {
	if m.Version != ManifestVersion || m.Files == nil || len(m.Files) > MaxFiles || m.Digest != treeDigest(m.Files) {
		return errors.New("invalid source manifest")
	}
	var size int64
	seen := map[string]bool{}
	for i, f := range m.Files {
		if !ValidPath(f.Path) || !digestPattern.MatchString(f.SHA256) || f.Bytes < 0 || f.Bytes > MaxFileBytes || i > 0 && m.Files[i-1].Path >= f.Path {
			return errors.New("invalid source file")
		}
		size += f.Bytes
		if size > MaxTreeBytes {
			return errors.New("source tree exceeds byte limit")
		}
		seen[f.Path] = true
	}
	for name := range seen {
		for p := path.Dir(name); p != "."; p = path.Dir(p) {
			if seen[p] {
				return errors.New("source file is also a directory")
			}
		}
	}
	return nil
}
func (m Manifest) Index() map[string]File {
	out := map[string]File{}
	for _, f := range m.Files {
		out[f.Path] = f
	}
	return out
}

// Snapshot accepts only ordinary files and directories. Root confinement also
// prevents a path swapped to an escaping symlink from opening host data. A
// changed file fails this observation instead of becoming a mixed source build.
func Snapshot(directory string) (Manifest, error) {
	st, err := os.Lstat(directory)
	if err != nil {
		return Manifest{}, err
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return Manifest{}, errors.New("source must be an ordinary directory")
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return Manifest{}, err
	}
	defer root.Close()
	files := []File{}
	var total int64
	entries := 0
	err = fs.WalkDir(root.FS(), ".", func(name string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if name == "." {
			return nil
		}
		entries++
		if entries > MaxFiles*4 || len(name) > 1024 || len(strings.Split(name, "/")) > 16 {
			return errors.New("source traversal limit exceeded")
		}
		if d.IsDir() {
			for _, p := range strings.Split(name, "/") {
				if !segment.MatchString(p) {
					return fmt.Errorf("unsupported source directory %q", name)
				}
			}
			return nil
		}
		if !ValidPath(name) || !d.Type().IsRegular() {
			return fmt.Errorf("unsupported source entry %q", name)
		}
		if len(files) >= MaxFiles {
			return errors.New("source file limit exceeded")
		}
		before, err := root.Lstat(name)
		if err != nil {
			return err
		}
		if !before.Mode().IsRegular() || before.Size() > MaxFileBytes {
			return fmt.Errorf("invalid source file %q", name)
		}
		f, err := root.Open(name)
		if err != nil {
			return err
		}
		opened, err := f.Stat()
		if err != nil {
			f.Close()
			return err
		}
		if !os.SameFile(before, opened) {
			f.Close()
			return errors.New("source file replaced during observation")
		}
		h := sha256.New()
		n, readErr := io.Copy(h, io.LimitReader(f, MaxFileBytes+1))
		after, statErr := f.Stat()
		closeErr := f.Close()
		if err = errors.Join(readErr, statErr, closeErr); err != nil {
			return err
		}
		named, err := root.Lstat(name)
		if err != nil {
			return err
		}
		if n > MaxFileBytes || n != before.Size() || after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) || !os.SameFile(after, named) || !named.Mode().IsRegular() {
			return errors.New("source changed during observation")
		}
		total += n
		if total > MaxTreeBytes {
			return errors.New("source byte limit exceeded")
		}
		files = append(files, File{name, hex.EncodeToString(h.Sum(nil)), n})
		return nil
	})
	if err != nil {
		return Manifest{}, err
	}
	return NewManifest(files)
}

type Change struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Before *File  `json:"before,omitempty"`
	After  *File  `json:"after,omitempty"`
}
type Plan struct {
	Managed   Manifest `json:"managed"`
	Preserved []string `json:"preserved"`
	Current   string   `json:"current"`
	Source    string   `json:"source"`
	Previous  string   `json:"previous"`
	Changes   []Change `json:"changes"`
	Conflicts []string `json:"conflicts"`
}

// PlanTransfer removes/replaces only files owned by the previous source binding
// whose bytes still match. Ship-generated and untracked files are preserved.
// An identical untracked file can be shared, but is never silently adopted.
func PlanTransfer(previous, current, source Manifest) (Plan, error) {
	for _, m := range []Manifest{previous, current, source} {
		if err := m.Validate(); err != nil {
			return Plan{}, err
		}
	}
	out := Plan{Current: current.Digest, Source: source.Digest, Previous: previous.Digest, Changes: []Change{}, Conflicts: []string{}, Preserved: []string{}}
	managed := []File{}
	old, now, next := previous.Index(), current.Index(), source.Index()
	names := map[string]bool{}
	for n := range old {
		names[n] = true
	}
	for n := range next {
		names[n] = true
	}
	ordered := []string{}
	for n := range names {
		ordered = append(ordered, n)
	}
	sort.Strings(ordered)
	for _, name := range ordered {
		prior, owned := old[name]
		before, exists := now[name]
		after, wanted := next[name]
		if wanted && exists && before == after {
			if owned {
				managed = append(managed, after)
			} else {
				out.Preserved = append(out.Preserved, name)
			}
			continue
		}
		if !wanted && !exists {
			continue
		}
		if owned && (!exists || before != prior) || !owned && exists {
			out.Conflicts = append(out.Conflicts, name)
			continue
		}
		// A directory/file collision with preserved content is also an explicit
		// conflict, not a reason to delete the enclosing tree.
		collision := false
		if wanted {
			for n := range now {
				if n != name && (strings.HasPrefix(n, name+"/") || strings.HasPrefix(name, n+"/")) {
					collision = true
					break
				}
			}
		}
		if collision {
			out.Conflicts = append(out.Conflicts, name)
			continue
		}
		c := Change{Path: name}
		if exists {
			v := before
			c.Before = &v
		}
		if wanted {
			v := after
			c.After = &v
		}
		switch {
		case !wanted:
			c.Kind = "delete"
		case exists:
			c.Kind = "replace"
		default:
			c.Kind = "add"
		}
		out.Changes = append(out.Changes, c)
		if wanted {
			managed = append(managed, after)
		}
	}
	out.Managed, _ = NewManifest(managed)
	return out, nil
}

type Comparison struct {
	State       string   `json:"state"`
	UnixChanges []string `json:"unix_changes"`
	ClayChanges []string `json:"clay_changes"`
	Conflicts   []string `json:"conflicts"`
}

// Compare expects mark-normalized manifests (or an exact, revision-fenced
// prior commit baseline). Raw Unix and rendered Clay bytes can legitimately
// differ; Snapshot alone does not establish semantic equality.
func Compare(base, unix, clay Manifest) (Comparison, error) {
	for _, m := range []Manifest{base, unix, clay} {
		if err := m.Validate(); err != nil {
			return Comparison{}, err
		}
	}
	out := Comparison{"synchronized", []string{}, []string{}, []string{}}
	b, u, c := base.Index(), unix.Index(), clay.Index()
	names := map[string]bool{}
	for _, m := range []map[string]File{b, u, c} {
		for n := range m {
			names[n] = true
		}
	}
	ordered := []string{}
	for n := range names {
		ordered = append(ordered, n)
	}
	sort.Strings(ordered)
	for _, n := range ordered {
		if u[n] != b[n] {
			out.UnixChanges = append(out.UnixChanges, n)
		}
		if c[n] != b[n] {
			out.ClayChanges = append(out.ClayChanges, n)
		}
		if u[n] != b[n] && c[n] != b[n] && u[n] != c[n] {
			out.Conflicts = append(out.Conflicts, n)
		}
	}
	switch {
	case unix.Digest == clay.Digest:
		out.State = "synchronized"
	case len(out.UnixChanges) > 0 && len(out.ClayChanges) > 0:
		out.State = "diverged"
	case len(out.UnixChanges) > 0:
		out.State = "unix-changes"
	case len(out.ClayChanges) > 0:
		out.State = "clay-changes"
	}
	return out, nil
}

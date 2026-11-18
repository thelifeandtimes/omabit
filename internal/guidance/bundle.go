// Package guidance reads only explicitly pinned, release-bound skill documents.
// Reading a document neither installs it nor grants operation authority.
package guidance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strings"
	"syscall"
	"unicode/utf8"
)

var revision = regexp.MustCompile(`^[a-f0-9]{40}$`)
var digest = regexp.MustCompile(`^[a-f0-9]{64}$`)
var relative = regexp.MustCompile(`^[A-Za-z0-9._/-]{1,256}$`)

type provenance struct {
	Version      string `json:"version"`
	SourceCommit string `json:"source_commit"`
	Development  any    `json:"development,omitempty"`
}
type manifest struct {
	provenance
	Format int               `json:"format"`
	Skills []string          `json:"skills"`
	Files  map[string]string `json:"files"`
}
type Entry struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}
type Index struct {
	Version        int     `json:"version"`
	ReleaseVersion string  `json:"release_version"`
	SourceCommit   string  `json:"source_commit"`
	ManifestSHA256 string  `json:"manifest_sha256"`
	Files          []Entry `json:"files"`
}
type Document struct {
	Entry
	ReleaseVersion string `json:"release_version"`
	SourceCommit   string `json:"source_commit"`
	ManifestSHA256 string `json:"manifest_sha256"`
	Text           string `json:"text"`
}
type Bundle struct {
	root, manifestSHA string
	manifest          manifest
}

func hash(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func ordinary(path string, limit int) ([]byte, error) {
	// O_NOFOLLOW and descriptor checks close the final-component link race;
	// every parent below the configured bundle is checked by checkedPath.
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0111 != 0 || info.Size() > int64(limit) {
		return nil, errors.New("invalid bounded skill file")
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); !ok || stat.Nlink != 1 {
		return nil, errors.New("skill file must have one ordinary link")
	}
	data, err := io.ReadAll(io.LimitReader(f, int64(limit)+1))
	if err != nil || len(data) > limit || !utf8.Valid(data) {
		return nil, errors.New("skill file exceeds limit or is not UTF-8")
	}
	return data, nil
}
func (b *Bundle) checkedPath(path string) (string, error) {
	if !relative.MatchString(path) || filepath.IsAbs(path) || filepath.ToSlash(filepath.Clean(path)) != path {
		return "", errors.New("invalid bundled resource path")
	}
	parts := strings.Split(path, "/")
	current := b.root
	for _, part := range parts[:len(parts)-1] {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("skill parent is not an ordinary directory")
		}
	}
	return filepath.Join(b.root, filepath.FromSlash(path)), nil
}

// Open requires the SKILLS.json hash from the verified matching bundle, supplied
// by local startup configuration. A self-consistent unpinned directory is refused.
func Open(root, expectedManifestSHA string) (*Bundle, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || !digest.MatchString(expectedManifestSHA) {
		return nil, errors.New("select an absolute skill bundle and verified manifest hash")
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("skill bundle root is not an ordinary directory")
	}
	data, err := ordinary(filepath.Join(root, "SKILLS.json"), 65536)
	if err != nil || hash(data) != expectedManifestSHA {
		return nil, errors.New("skill manifest does not match its configured pin")
	}
	b := &Bundle{root: root, manifestSHA: expectedManifestSHA}
	if json.Unmarshal(data, &b.manifest) != nil {
		return nil, errors.New("invalid skill manifest")
	}
	buildData, err := ordinary(filepath.Join(root, "BUILD.json"), 65536)
	if err != nil {
		return nil, errors.New("skill build provenance unavailable")
	}
	var build provenance
	if json.Unmarshal(buildData, &build) != nil || !reflect.DeepEqual(build, b.manifest.provenance) || build.Version == "" || !revision.MatchString(build.SourceCommit) {
		return nil, errors.New("skill manifest and build provenance differ")
	}
	known := []string{"omabit-operator", "omabit-plugin-developer", "omabit-app-developer"}
	m := b.manifest
	if m.Format != 1 || len(m.Skills) < 2 || len(m.Skills) > 3 || len(m.Files) < 2 || len(m.Files) > 64 {
		return nil, errors.New("unsupported skill inventory")
	}
	seen := map[string]bool{}
	for _, name := range m.Skills {
		if !slices.Contains(known, name) || seen[name] {
			return nil, errors.New("unknown or repeated skill")
		}
		seen[name] = true
		if _, ok := m.Files["skills/"+name+"/SKILL.md"]; !ok {
			return nil, errors.New("skill entry point missing")
		}
	}
	if !seen["omabit-operator"] || !seen["omabit-plugin-developer"] {
		return nil, errors.New("legacy skill pair missing")
	}
	total := 0
	for name, want := range m.Files {
		parts := strings.Split(name, "/")
		if len(parts) < 3 || parts[0] != "skills" || !seen[parts[1]] || !strings.HasSuffix(name, ".md") || !digest.MatchString(want) {
			return nil, errors.New("invalid manifest resource")
		}
		path, err := b.checkedPath(name)
		if err != nil {
			return nil, err
		}
		content, err := ordinary(path, 65536)
		if err != nil || hash(content) != want {
			return nil, errors.New("skill bytes differ from manifest")
		}
		total += len(content)
		if total > 1048576 {
			return nil, errors.New("skill inventory exceeds limit")
		}
	}
	// Extra resources are not silently omitted from the release inventory.
	count := 0
	err = filepath.WalkDir(filepath.Join(root, "skills"), func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("linked skill resource")
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if _, ok := m.Files[filepath.ToSlash(rel)]; !ok {
			return errors.New("unlisted skill resource")
		}
		count++
		return nil
	})
	if err != nil || count != len(m.Files) {
		return nil, errors.New("skill tree does not match manifest")
	}
	return b, nil
}

func (b *Bundle) Index() Index {
	out := Index{Version: 1, ReleaseVersion: b.manifest.Version, SourceCommit: b.manifest.SourceCommit, ManifestSHA256: b.manifestSHA, Files: []Entry{}}
	for path, sha := range b.manifest.Files {
		out.Files = append(out.Files, Entry{Path: path, SHA256: sha})
	}
	sort.Slice(out.Files, func(i, j int) bool { return out.Files[i].Path < out.Files[j].Path })
	return out
}
func (b *Bundle) Read(path string) (Document, error) {
	want, ok := b.manifest.Files[path]
	if !ok {
		return Document{}, errors.New("resource is outside the verified skill inventory")
	}
	full, err := b.checkedPath(path)
	if err != nil {
		return Document{}, err
	}
	data, err := ordinary(full, 65536)
	if err != nil || hash(data) != want {
		return Document{}, errors.New("bundled skill changed after verification")
	}
	return Document{Entry: Entry{Path: path, SHA256: want}, ReleaseVersion: b.manifest.Version, SourceCommit: b.manifest.SourceCommit, ManifestSHA256: b.manifestSHA, Text: string(data)}, nil
}

package hostcompat

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"net/url"
	"path"
	"slices"
	"sort"
	"strings"
)

type aptSource struct {
	URI           string   `json:"uri"`
	Suite         string   `json:"suite"`
	Components    []string `json:"components"`
	Architectures []string `json:"architectures"`
	Keyrings      []string `json:"keyrings"`
}

func (s aptSource) id() string {
	b, _ := json.Marshal(s)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// proofID binds repository location/suite independently of current keyring
// filenames and enabled components. Reuse also checks the exact retained index
// against current component/architecture scope before using its current keys.
func (s aptSource) proofID() string {
	s.Keyrings = nil
	s.Components = nil
	s.Architectures = nil
	return s.id()
}

func (s aptSource) permitsRetainedIndex(name, arch string) bool {
	base := path.Base(strings.TrimSuffix(s.releasePath(), "_InRelease")) + "_"
	name = path.Base(name)
	for _, component := range s.Components {
		for _, architecture := range []string{arch, "all"} {
			index := component + "/binary-" + architecture + "/Packages"
			if !s.permitsIndex(index, arch) {
				continue
			}
			for _, ext := range []string{"", ".lz4", ".xz", ".gz"} {
				if name == base+strings.ReplaceAll(index, "/", "_")+ext {
					return true
				}
			}
		}
	}
	return false
}
func (s aptSource) releasePath() string {
	u, _ := url.Parse(s.URI)
	return "var/lib/apt/lists/" + strings.ReplaceAll(u.Host+strings.TrimSuffix(u.Path, "/")+"/dists/"+s.Suite+"/InRelease", "/", "_")
}
func (s aptSource) permitsIndex(name, arch string) bool {
	if len(s.Architectures) > 0 && !containsString(s.Architectures, arch) {
		return false
	}
	for _, component := range s.Components {
		if name == component+"/binary-"+arch+"/Packages" || name == component+"/binary-all/Packages" {
			return true
		}
	}
	return false
}
func containsString(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

func aptSources(fsys fs.FS) ([]aptSource, []string, error) {
	if _, err := aptConfigurationInputs(fsys); err != nil {
		return nil, nil, err
	}
	var paths []string
	if _, err := fs.Stat(fsys, "etc/apt/sources.list"); err == nil {
		paths = append(paths, "etc/apt/sources.list")
	} else if !isMissing(err) {
		return nil, nil, errEvidence
	}
	if _, err := fs.Stat(fsys, "etc/apt/sources.list.d"); err == nil {
		entries, e := readDir(fsys, "etc/apt/sources.list.d")
		if e != nil || len(entries) > 128 {
			return nil, nil, errEvidence
		}
		for _, entry := range entries {
			if nativeAPTFragment(entry.Name(), false) {
				name := "etc/apt/sources.list.d/" + entry.Name()
				regular, err := nativeAPTPartFile(fsys, name)
				if err != nil {
					return nil, nil, err
				}
				if regular {
					paths = append(paths, name)
				}
			}
		}
	} else if !isMissing(err) {
		return nil, nil, errEvidence
	}
	sort.Strings(paths)
	var result []aptSource
	total := 0
	for _, name := range paths {
		b, err := readTrusted(fsys, name, 256<<10)
		if err != nil {
			return nil, nil, err
		}
		total += len(b)
		if total > 1<<20 {
			return nil, nil, errEvidence
		}
		var sources []aptSource
		if strings.HasSuffix(name, ".sources") {
			sources, err = parseDeb822Sources(string(b))
		} else {
			sources, err = parseListSources(string(b))
		}
		if err != nil {
			return nil, nil, err
		}
		result = append(result, sources...)
		if len(result) > 64 {
			return nil, nil, errEvidence
		}
	}
	if len(result) == 0 {
		return nil, nil, errEvidence
	}
	seen := map[string]int{}
	var unique []aptSource
	for _, s := range result {
		key := s.releasePath()
		if index, ok := seen[key]; ok {
			prior := &unique[index]
			// Ubuntu's default sources.list lists main/restricted, universe and
			// multiverse separately for the same signed Release. Combine only
			// component scopes with identical URI, architecture and key authority.
			// Never let a second source broaden another source's signing keys.
			if prior.URI != s.URI || prior.Suite != s.Suite || !slices.Equal(prior.Architectures, s.Architectures) || !slices.Equal(prior.Keyrings, s.Keyrings) {
				return nil, nil, errEvidence
			}
			prior.Components = append(prior.Components, s.Components...)
			sort.Strings(prior.Components)
			prior.Components = uniqueStrings(prior.Components)
			if len(prior.Components) > 32 {
				return nil, nil, errEvidence
			}
			continue
		}
		seen[key] = len(unique)
		unique = append(unique, s)
	}
	sort.Slice(unique, func(i, j int) bool { return unique[i].id() < unique[j].id() })
	return unique, paths, nil
}

func isMissing(err error) bool { return errors.Is(err, fs.ErrNotExist) }

func sourceTuples(uris, suites, components, arches, keys []string) ([]aptSource, error) {
	if len(uris) == 0 || len(uris) > 8 || len(suites) == 0 || len(suites) > 8 || len(components) == 0 || len(components) > 32 || len(arches) > 8 || len(keys) > 8 {
		return nil, errEvidence
	}
	for _, v := range append(append(append([]string{}, suites...), components...), arches...) {
		if !token(v) || strings.ContainsAny(v, ":~") {
			return nil, errEvidence
		}
	}
	components = append([]string{}, components...)
	sort.Strings(components)
	components = uniqueStrings(components)
	arches = append([]string{}, arches...)
	sort.Strings(arches)
	arches = uniqueStrings(arches)
	var result []aptSource
	for _, raw := range uris {
		u, err := url.Parse(raw)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.Host == "" || u.Port() != "" || u.RawQuery != "" || u.Fragment != "" || u.RawPath != "" || len(raw) > 2048 {
			return nil, errEvidence
		}
		for _, c := range u.Host + u.Path {
			if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789./-", c) {
				return nil, errEvidence
			}
		}
		if strings.Contains(u.Path, "..") {
			return nil, errEvidence
		}
		selected := append([]string{}, keys...)
		if len(selected) == 0 {
			host := strings.ToLower(u.Hostname())
			switch {
			case host == "archive.ubuntu.com" || strings.HasSuffix(host, ".archive.ubuntu.com") || host == "security.ubuntu.com" || host == "ports.ubuntu.com":
				selected = []string{"/usr/share/keyrings/ubuntu-archive-keyring.gpg"}
			case host == "deb.debian.org" || host == "security.debian.org" || host == "ftp.debian.org":
				selected = []string{"/usr/share/keyrings/debian-archive-keyring.gpg"}
			default:
				return nil, errEvidence
			}
		}
		for i, key := range selected {
			if !strings.HasPrefix(key, "/") || !fs.ValidPath(strings.TrimPrefix(key, "/")) || len(key) > 4096 || (path.Ext(key) != ".gpg" && path.Ext(key) != ".asc") {
				return nil, errEvidence
			}
			selected[i] = strings.TrimPrefix(key, "/")
		}
		sort.Strings(selected)
		for _, suite := range suites {
			result = append(result, aptSource{strings.TrimSuffix(raw, "/"), suite, append([]string{}, components...), append([]string{}, arches...), selected})
		}
	}
	return result, nil
}

func parseListSources(text string) ([]aptSource, error) {
	var result []aptSource
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
		if line == "" {
			continue
		}
		space := strings.IndexAny(line, " \t")
		if space < 0 {
			return nil, errEvidence
		}
		typ, rest := line[:space], line[space+1:]
		rest = strings.TrimSpace(rest)
		if typ != "deb" && typ != "deb-src" {
			return nil, errEvidence
		}
		var keys, arches []string
		if strings.HasPrefix(rest, "[") {
			opts, after, ok := strings.Cut(rest[1:], "]")
			if !ok {
				return nil, errEvidence
			}
			rest = strings.TrimSpace(after)
			seen := map[string]bool{}
			for _, option := range strings.Fields(opts) {
				k, v, ok := strings.Cut(option, "=")
				if !ok || seen[k] {
					return nil, errEvidence
				}
				seen[k] = true
				switch k {
				case "signed-by":
					keys = strings.Split(v, ",")
				case "arch":
					arches = strings.Split(v, ",")
				case "trusted":
					// yes bypasses native authentication; no explicitly withholds
					// source trust. Neither grants an authority for this verifier.
					return nil, errEvidence
				default:
					return nil, errEvidence
				}
			}
		}
		f := strings.Fields(rest)
		if len(f) < 3 {
			return nil, errEvidence
		}
		if typ == "deb-src" {
			continue
		}
		sources, err := sourceTuples(f[:1], f[1:2], f[2:], arches, keys)
		if err != nil {
			return nil, err
		}
		result = append(result, sources...)
	}
	return result, nil
}

func parseDeb822Sources(text string) ([]aptSource, error) {
	var clean []string
	for _, line := range strings.Split(text, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "#") {
			clean = append(clean, line)
		}
	}
	stanzas, err := parseControl(strings.Join(clean, "\n"))
	if err != nil {
		return nil, err
	}
	var result []aptSource
	for _, m := range stanzas {
		// Debian control field names are case-insensitive. Canonicalize here
		// before enforcing security options so casing cannot bypass them.
		normalized := map[string]string{}
		for key, value := range m {
			key = strings.ToLower(key)
			if _, ok := normalized[key]; ok {
				return nil, errEvidence
			}
			normalized[key] = value
		}
		m = map[string]string{}
		for _, key := range []string{"Enabled", "Trusted", "Allow-Insecure", "Allow-Weak", "Allow-Downgrade-To-Insecure", "Check-Valid-Until", "Check-Date", "Architectures-Add", "Architectures-Remove", "InRelease-Path", "Types", "URIs", "Suites", "Components", "Architectures", "Signed-By"} {
			m[key] = normalized[strings.ToLower(key)]
		}
		if strings.EqualFold(m["Enabled"], "no") {
			continue
		}
		if m["Enabled"] != "" && !strings.EqualFold(m["Enabled"], "yes") {
			return nil, errEvidence
		}
		if m["Trusted"] != "" {
			return nil, errEvidence
		}
		for _, field := range []string{"Allow-Insecure", "Allow-Weak", "Allow-Downgrade-To-Insecure"} {
			if m[field] != "" && !strings.EqualFold(m[field], "no") {
				return nil, errEvidence
			}
		}
		for _, field := range []string{"Check-Valid-Until", "Check-Date"} {
			if m[field] != "" && !strings.EqualFold(m[field], "yes") {
				return nil, errEvidence
			}
		}
		for _, field := range []string{"Architectures-Add", "Architectures-Remove", "InRelease-Path"} {
			if m[field] != "" {
				return nil, errEvidence
			}
		}
		types := strings.Fields(m["Types"])
		for _, typ := range types {
			if typ != "deb" && typ != "deb-src" {
				return nil, errEvidence
			}
		}
		if !containsString(types, "deb") {
			continue
		}
		sources, err := sourceTuples(strings.Fields(m["URIs"]), strings.Fields(m["Suites"]), strings.Fields(m["Components"]), strings.Fields(m["Architectures"]), strings.Fields(m["Signed-By"]))
		if err != nil {
			return nil, err
		}
		result = append(result, sources...)
	}
	return result, nil
}

// Native dearmor is a pure conversion: it imports no key and confers no trust.
// Only this source's root-configured keyrings are passed to gpgv.
func (v *verifier) aptKeyArgs(ctx context.Context, source aptSource) ([]string, error) {
	var args []string
	for _, name := range source.Keyrings {
		b, err := readTrusted(v.fsys, name, 8<<20)
		if err != nil {
			return nil, err
		}
		key := name
		if strings.HasSuffix(name, ".asc") {
			if !bytesHasArmor(b) || v.store == nil {
				return nil, errEvidence
			}
			converted, e := v.run(ctx, "/usr/bin/gpg", []string{"--no-options", "--batch", "--dearmor", "--output", "-", "/" + name}, 8<<20)
			if e != nil || len(converted) == 0 {
				return nil, errEvidence
			}
			key, e = v.store(ctx, converted, ".gpg")
			if e != nil {
				return nil, e
			}
		} else if bytesHasArmor(b) {
			return nil, errEvidence
		}
		args = append(args, "--keyring", "/"+key)
	}
	return args, nil
}

func bytesHasArmor(b []byte) bool {
	return strings.HasPrefix(strings.TrimSpace(string(b)), "-----BEGIN PGP PUBLIC KEY BLOCK-----")
}

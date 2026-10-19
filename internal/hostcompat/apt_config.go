package hostcompat

import (
	"io/fs"
	"os"
	"sort"
	"strings"
)

const aptAuthorityModel = "native-default-source-selection-v2"

// APT 2.6/2.8 ReadConfigDir and ReadSourceDir both use the extension-aware
// GetListOfFilesInDir overload (apt-pkg/contrib/fileutl.cc), which permits ':'.
// Do not substitute the different extension-free overload or the manpage's
// narrower character list. Config accepts no extension/.conf; sources require
// .list/.sources. Both ignore dotfiles; explicit main files are separate.
func nativeAPTFragment(name string, configuration bool) bool {
	if name == "" || name[0] == '.' {
		return false
	}
	for _, c := range name {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-' || c == '.' || c == ':') {
			return false
		}
	}
	if !configuration {
		return strings.HasSuffix(name, ".list") || strings.HasSuffix(name, ".sources")
	}
	if dot := strings.LastIndexByte(name, '.'); dot >= 0 {
		return name[dot+1:] == "conf"
	}
	return true
}

func nativeAPTPartFile(fsys fs.FS, name string) (bool, error) {
	info, err := fs.Stat(fsys, name)
	if isMissing(err) {
		return false, nil
	}
	if err != nil {
		return false, errEvidence
	}
	return info.Mode().IsRegular(), nil
}

// Effective selection overrides are deliberately unresolved rather than
// approximated. This routine is read-only: it never invokes apt-config or a shell.
// Unrelated hooks/cache/proxy configuration is parsed as data and does not become
// source authority. Its root-trusted input hashes are retained in the audit set.
func aptConfigurationInputs(fsys fs.FS) ([]string, error) {
	if env, ok := fsys.(interface{ aptEnvironment() string }); ok && env.aptEnvironment() != "" {
		return nil, errEvidence
	}
	var paths []string
	if _, err := fs.Stat(fsys, "etc/apt/apt.conf.d"); err == nil {
		entries, err := readDir(fsys, "etc/apt/apt.conf.d")
		if err != nil || len(entries) > 128 {
			return nil, errEvidence
		}
		for _, entry := range entries {
			if nativeAPTFragment(entry.Name(), true) {
				name := "etc/apt/apt.conf.d/" + entry.Name()
				regular, err := nativeAPTPartFile(fsys, name)
				if err != nil {
					return nil, err
				}
				if regular {
					paths = append(paths, name)
				}
			}
		}
	} else if !isMissing(err) {
		return nil, errEvidence
	}
	sort.Strings(paths)
	if _, err := fs.Stat(fsys, "etc/apt/apt.conf"); err == nil {
		paths = append(paths, "etc/apt/apt.conf")
	} else if !isMissing(err) {
		return nil, errEvidence
	}
	total := 0
	for _, name := range paths {
		b, err := readTrusted(fsys, name, 256<<10)
		if err != nil {
			return nil, err
		}
		total += len(b)
		if total > 1<<20 {
			return nil, errEvidence
		}
		if validateAPTConfiguration(string(b)) != nil {
			return nil, errEvidence
		}
	}
	return paths, nil
}

type aptConfigToken struct {
	text   string
	quoted bool
}

func aptConfigTokens(text string) ([]aptConfigToken, error) {
	var tokens []aptConfigToken
	for i := 0; i < len(text); {
		c := text[i]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			i++
			continue
		}
		if c == '#' {
			end := strings.IndexByte(text[i:], '\n')
			if end < 0 {
				end = len(text) - i
			}
			line := strings.TrimSpace(text[i+1 : i+end])
			lower := strings.ToLower(line)
			if strings.HasPrefix(lower, "include") || strings.HasPrefix(lower, "clear") {
				return nil, errEvidence
			}
			i += end
			continue
		}
		if c == '/' && i+1 < len(text) && text[i+1] == '/' {
			end := strings.IndexByte(text[i:], '\n')
			if end < 0 {
				break
			}
			i += end
			continue
		}
		if c == '/' && i+1 < len(text) && text[i+1] == '*' {
			end := strings.Index(text[i+2:], "*/")
			if end < 0 {
				return nil, errEvidence
			}
			i += end + 4
			continue
		}
		if c == '"' {
			i++
			var b strings.Builder
			closed := false
			for i < len(text) {
				ch := text[i]
				i++
				if ch == '"' {
					closed = true
					break
				}
				if ch == '\\' {
					if i == len(text) {
						return nil, errEvidence
					}
					b.WriteByte(ch)
					ch = text[i]
					i++
				}
				b.WriteByte(ch)
				if b.Len() > 65536 {
					return nil, errEvidence
				}
			}
			if !closed {
				return nil, errEvidence
			}
			tokens = append(tokens, aptConfigToken{b.String(), true})
		} else if c == '{' || c == '}' || c == ';' {
			tokens = append(tokens, aptConfigToken{text[i : i+1], false})
			i++
		} else {
			start := i
			for i < len(text) {
				c = text[i]
				if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '{' || c == '}' || c == ';' || c == '"' {
					break
				}
				i++
			}
			if i == start || i-start > 256 {
				return nil, errEvidence
			}
			tokens = append(tokens, aptConfigToken{text[start:i], false})
		}
		if len(tokens) > 32768 {
			return nil, errEvidence
		}
	}
	return tokens, nil
}

func aptAuthorityOverride(key string) bool {
	key = strings.ToLower(strings.TrimSuffix(key, "::"))
	// Include binary-specific scopes: Binary::apt-get::Dir::Etc is an
	// effective override too, not an unrelated namespace.
	parts := strings.Split(key, "::")
	for i, part := range parts {
		if part == "rootdir" {
			return true
		}
		if part == "dir" && (i == len(parts)-1 || parts[i+1] == "etc" || parts[i+1] == "ignore-files-silently") {
			return true
		}
		if part == "apt" && i+1 < len(parts) && (parts[i+1] == "architecture" || parts[i+1] == "architectures") {
			return true
		}
	}
	return strings.Contains("::"+key, "::acquire::indextargets::deb::packages")
}

func validateAPTConfiguration(text string) error {
	tokens, err := aptConfigTokens(text)
	if err != nil {
		return err
	}
	position := 0
	var scope func(string, int) error
	scope = func(prefix string, depth int) error {
		if depth > 16 {
			return errEvidence
		}
		for position < len(tokens) {
			t := tokens[position]
			position++
			if t.text == "}" && !t.quoted {
				if depth == 0 {
					return errEvidence
				}
				return nil
			}
			if t.quoted {
				if position >= len(tokens) || tokens[position].text != ";" || tokens[position].quoted {
					return errEvidence
				}
				position++
				continue
			}
			if t.text == ";" || t.text == "{" || t.text == "" {
				return errEvidence
			}
			for _, c := range t.text {
				if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == ':' || c == '_' || c == '-' || c == '.' || c == '/' || c == '+') {
					return errEvidence
				}
			}
			key := t.text
			if prefix != "" {
				key = prefix + "::" + key
			}
			if aptAuthorityOverride(key) {
				return errEvidence
			}
			if position >= len(tokens) {
				return errEvidence
			}
			next := tokens[position]
			position++
			if next.text == "{" && !next.quoted {
				if err := scope(key, depth+1); err != nil {
					return err
				}
			} else if !next.quoted && (next.text == "}" || next.text == ";" || next.text == "") {
				return errEvidence
			}
			if position >= len(tokens) || tokens[position].text != ";" || tokens[position].quoted {
				return errEvidence
			}
			position++
		}
		if depth != 0 {
			return errEvidence
		}
		return nil
	}
	return scope("", 0)
}

// Environment overrides are an invocation-specific authority input. Pure
// Inspect(fsys,...) remains environment-independent; live entry points capture
// the native environment and fail closed if APT_CONFIG changes selection.
type aptEnvironmentFS struct {
	fs.FS
	override string
}

func (f aptEnvironmentFS) Stat(name string) (fs.FileInfo, error)  { return fs.Stat(f.FS, name) }
func (f aptEnvironmentFS) Lstat(name string) (fs.FileInfo, error) { return fs.Lstat(f.FS, name) }
func (f aptEnvironmentFS) ReadLink(name string) (string, error)   { return fs.ReadLink(f.FS, name) }
func (f aptEnvironmentFS) aptEnvironment() string                 { return f.override }
func currentEvidenceFS(fsys fs.FS) fs.FS                          { return aptEnvironmentFS{fsys, os.Getenv("APT_CONFIG")} }

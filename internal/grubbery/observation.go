package grubbery

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"path"
	"regexp"
	"strings"
)

var ErrObservation = errors.New("native app observation is incomplete, changed or unsupported")
var nativeName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$`)
var nativeArm = regexp.MustCompile(`^[a-z][a-z0-9-]{0,255}$`)

// Pinned kernel build-code-with forces these sources into every code namespace.
// They are framework-owned, not an app's staged source or ownership claim.
var foundationPaths = []string{"mar/hoon.hoon", "mar/kelvin.hoon", "mar/mime.hoon", "mar/tang.hoon"}

func foundation(name string) bool {
	for _, p := range foundationPaths {
		if name == p {
			return true
		}
	}
	return false
}

// CheckNativeSource refuses attempts the pinned kernel would overwrite by fiat.
// Historical metadata validation remains separate; old receipts are not rewritten.
func CheckNativeSource(source Source) error {
	if source.ValidateMetadata() != nil {
		return ErrSource
	}
	for _, f := range source.Manifest.Files {
		if foundation(f.Path) {
			return errors.New("pinned framework owns foundational mark sources")
		}
	}
	return nil
}

func uniqueNativeJSON(raw []byte) bool {
	if len(raw) == 0 || len(raw) > 1<<20 {
		return false
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var value func(int) bool
	value = func(depth int) bool {
		if depth > 32 {
			return false
		}
		token, err := d.Token()
		if err != nil {
			return false
		}
		delimiter, container := token.(json.Delim)
		if !container {
			return true
		}
		if delimiter != '{' && delimiter != '[' {
			return false
		}
		seen := map[string]bool{}
		for d.More() {
			if delimiter == '{' {
				key, err := d.Token()
				name, ok := key.(string)
				if err != nil || !ok || seen[name] {
					return false
				}
				seen[name] = true
			}
			if !value(depth + 1) {
				return false
			}
		}
		end, err := d.Token()
		return err == nil && (delimiter == '{' && end == json.Delim('}') || delimiter == '[' && end == json.Delim(']'))
	}
	return value(0) && d.Decode(new(any)) == io.EOF
}
func nativeObject(raw []byte, allowed ...string) (map[string]json.RawMessage, error) {
	var out map[string]json.RawMessage
	if json.Unmarshal(raw, &out) != nil || out == nil {
		return nil, ErrObservation
	}
	for key := range out {
		found := false
		for _, name := range allowed {
			if name == key {
				found = true
				break
			}
		}
		if !found {
			return nil, ErrObservation
		}
	}
	return out, nil
}

// EqualNativeJSON refuses duplicate keys and preserves exact numeric spelling;
// there is no float conversion of large identity/revision integers.
func EqualNativeJSON(actual, expected []byte) bool {
	if !uniqueNativeJSON(actual) || !uniqueNativeJSON(expected) {
		return false
	}
	decode := func(raw []byte) ([]byte, error) {
		d := json.NewDecoder(bytes.NewReader(raw))
		d.UseNumber()
		var v any
		if err := d.Decode(&v); err != nil {
			return nil, err
		}
		return json.Marshal(v)
	}
	a, err := decode(actual)
	if err != nil {
		return false
	}
	b, err := decode(expected)
	return err == nil && bytes.Equal(a, b)
}

type NativeDirectory struct {
	HasLump     bool
	Neck        *string
	Files       map[string]string
	Directories map[string]NativeDirectory
}

func ParseDirectory(raw []byte) (NativeDirectory, error) {
	if !uniqueNativeJSON(raw) {
		return NativeDirectory{}, ErrObservation
	}
	count := 0
	var parse func([]byte, int) (NativeDirectory, error)
	parse = func(data []byte, depth int) (NativeDirectory, error) {
		var out NativeDirectory
		count++
		if depth > 24 || count > 4096 {
			return out, ErrObservation
		}
		object, err := nativeObject(data, "neck", "files", "dirs")
		if err != nil {
			return out, err
		}
		dirs, ok := object["dirs"]
		if !ok {
			return out, ErrObservation
		}
		var children map[string]json.RawMessage
		if json.Unmarshal(dirs, &children) != nil || children == nil {
			return out, ErrObservation
		}
		out.Directories = map[string]NativeDirectory{}
		files, hasFiles := object["files"]
		neck, hasNeck := object["neck"]
		if hasFiles != hasNeck {
			return out, ErrObservation
		}
		if hasFiles {
			out.HasLump = true
			if json.Unmarshal(files, &out.Files) != nil || out.Files == nil || json.Unmarshal(neck, &out.Neck) != nil || out.Neck != nil && !nativeArm.MatchString(*out.Neck) {
				return out, ErrObservation
			}
			for name, mark := range out.Files {
				count++
				if !nativeName.MatchString(name) || !nativeArm.MatchString(mark) || count > 4096 {
					return out, ErrObservation
				}
			}
		}
		for name, body := range children {
			if !nativeName.MatchString(name) {
				return out, ErrObservation
			}
			child, err := parse(body, depth+1)
			if err != nil {
				return out, err
			}
			out.Directories[name] = child
		}
		return out, nil
	}
	return parse(raw, 0)
}

// CheckStagedTree checks only this release's namespace structure, independently
// of its bytes and compiled cache. Pinned virtual foundational grubs are read
// separately; their existence does not put files/directories in this tree.
// Missing files may be allowed while
// staging, but unknown paths, wrong marks or nested nexuses always refuse.
func CheckStagedTree(source Source, tree NativeDirectory, complete, versionAllowed bool) error {
	if CheckNativeSource(source) != nil || tree.Neck == nil || *tree.Neck != "code" {
		return ErrObservation
	}
	expected := map[string]string{}
	dirs := map[string]bool{}
	for _, f := range source.Manifest.Files {
		mark := "mime"
		if codePath(f.Path) {
			mark = "hoon"
		}
		expected[f.Path] = mark
	}
	if versionAllowed {
		expected["version.json"] = "mime"
	}
	for name := range expected {
		for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
			dirs[parent] = true
		}
	}
	seen := map[string]bool{}
	var check func(NativeDirectory, string, bool) error
	check = func(node NativeDirectory, prefix string, root bool) error {
		if !root && node.Neck != nil {
			return ErrObservation
		}
		for name, mark := range node.Files {
			full := prefix + name
			if expected[full] != mark {
				return ErrObservation
			}
			seen[full] = true
		}
		for name, child := range node.Directories {
			full := prefix + name
			if !dirs[full] {
				return ErrObservation
			}
			if err := check(child, full+"/", false); err != nil {
				return err
			}
		}
		return nil
	}
	if check(tree, "", true) != nil {
		return ErrObservation
	}
	if complete {
		for name := range expected {
			if !seen[name] {
				return ErrObservation
			}
		}
	}
	return nil
}

type BuildFile struct {
	Source string   `json:"source"`
	Status string   `json:"status"`
	Error  []string `json:"-"` // private native compiler text; never implicit export
}
type BuildReport struct {
	Namespace string      `json:"namespace"`
	Total     int         `json:"total"`
	Failed    int         `json:"failed"`
	Files     []BuildFile `json:"files"`
}

func ParseBuildReport(raw []byte, namespace string) (BuildReport, error) {
	var out BuildReport
	if !uniqueNativeJSON(raw) {
		return out, ErrObservation
	}
	object, err := nativeObject(raw, "namespace", "total", "failed", "files")
	if err != nil || len(object) != 4 {
		return out, ErrObservation
	}
	if json.Unmarshal(object["namespace"], &out.Namespace) != nil || out.Namespace != namespace || json.Unmarshal(object["total"], &out.Total) != nil || json.Unmarshal(object["failed"], &out.Failed) != nil || out.Total < 1 || out.Total > 8192 || out.Failed < 0 || out.Failed > out.Total {
		return BuildReport{}, ErrObservation
	}
	var rows []json.RawMessage
	if json.Unmarshal(object["files"], &rows) != nil || rows == nil || len(rows) != out.Total {
		return BuildReport{}, ErrObservation
	}
	seen := map[string]bool{}
	failed, errorBytes := 0, 0
	for _, row := range rows {
		fields, err := nativeObject(row, "source", "status", "error")
		if err != nil || len(fields) != 3 {
			return BuildReport{}, ErrObservation
		}
		var f BuildFile
		if json.Unmarshal(fields["source"], &f.Source) != nil || json.Unmarshal(fields["status"], &f.Status) != nil || !strings.HasPrefix(f.Source, "/") || !validNativeSourcePath(strings.TrimPrefix(f.Source, "/")) || seen[f.Source] {
			return BuildReport{}, ErrObservation
		}
		seen[f.Source] = true
		switch f.Status {
		case "vase", "mime":
			if !bytes.Equal(bytes.TrimSpace(fields["error"]), []byte("null")) {
				return BuildReport{}, ErrObservation
			}
		case "tang":
			failed++
			if json.Unmarshal(fields["error"], &f.Error) != nil || f.Error == nil || len(f.Error) > 256 {
				return BuildReport{}, ErrObservation
			}
			for _, line := range f.Error {
				errorBytes += len(line)
				if len(line) > 4096 || errorBytes > 128<<10 {
					return BuildReport{}, ErrObservation
				}
			}
		default:
			return BuildReport{}, ErrObservation
		}
		out.Files = append(out.Files, f)
	}
	if failed != out.Failed {
		return BuildReport{}, ErrObservation
	}
	return out, nil
}
func validNativeSourcePath(name string) bool {
	if name == "" || len(name) > 1024 || path.Clean(name) != name || strings.HasPrefix(name, "/") {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if !nativeName.MatchString(part) {
			return false
		}
	}
	return true
}
func (r BuildReport) CheckStaged(source Source, versionAllowed bool) error {
	if CheckNativeSource(source) != nil || r.Failed != 0 {
		return ErrObservation
	}
	expected := map[string]string{}
	for _, name := range foundationPaths {
		expected["/"+name] = "vase"
	}
	for _, f := range source.Manifest.Files {
		kind := "mime"
		if codePath(f.Path) {
			kind = "vase"
		}
		expected["/"+f.Path] = kind
	}
	if versionAllowed {
		expected["/version.json"] = "mime"
	}
	if len(r.Files) != len(expected) || r.Total != len(expected) {
		return ErrObservation
	}
	for _, f := range r.Files {
		if expected[f.Source] != f.Status {
			return ErrObservation
		}
		delete(expected, f.Source)
	}
	if len(expected) != 0 {
		return ErrObservation
	}
	return nil
}

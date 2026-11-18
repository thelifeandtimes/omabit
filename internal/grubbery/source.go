// Package grubbery defines the pinned framework's application adapter contracts.
// These constructors perform no I/O and grant no policy or runtime authority.
package grubbery

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

const FrameworkCommit = "7f7701d546ce115eea5f37f66465b362cb121774"
const FrameworkSourceDigest = "7d2ae430acd4868bb56268188dff57addcdf4520abcc7aabaa6b9a342b22febc"
const FrameworkDesk = "grubbery"
const HTTPContract = "grubbery-darts-7f7701d-v1"
const MaxAppFiles = 64
const MaxAppBytes = 512 << 10
const MaxAppFileBytes = 64 << 10
const MaxInstances = 8

var term = regexp.MustCompile(`^[a-z][a-z0-9-]{0,47}$`)
var digest = regexp.MustCompile(`^[0-9a-f]{64}$`)

type Instance struct {
	Name string `json:"name"`
	Neck string `json:"neck"`
}

type Source struct {
	Version   int               `json:"version"`
	Manifest  deskwork.Manifest `json:"manifest"`
	Instances []Instance        `json:"instances"`
}

// ReviewSource accepts just one bounded code tree, not a Clay desk/repository.
// version.json is controller-owned and written last in a unique release tree.
// Every declared instance must resolve to a nexus in this exact source; a bill
// cannot instantiate a privileged built-in by pointing outside its own nex/.
func ReviewSource(manifest deskwork.Manifest, files []deskwork.SourceFile) (Source, error) {
	var out Source
	if manifest.Validate() != nil || len(files) != len(manifest.Files) || len(files) < 2 || len(files) > MaxAppFiles {
		return out, errors.New("invalid bounded Grubbery app source")
	}
	index := manifest.Index()
	seen := map[string]bool{}
	total := 0
	var bill []byte
	for _, file := range files {
		entry, ok := index[file.Path]
		sum := sha256.Sum256(file.Bytes)
		total += len(file.Bytes)
		if !ok || seen[file.Path] || len(file.Bytes) > MaxAppFileBytes || total > MaxAppBytes || entry.Bytes != int64(len(file.Bytes)) || entry.SHA256 != hex.EncodeToString(sum[:]) {
			return out, errors.New("app source differs from its exact stage")
		}
		seen[file.Path] = true
		if file.Path == "bill.json" {
			bill = file.Bytes
			continue
		}
		if file.Path == "README.md" {
			continue
		}
		parts := strings.Split(file.Path, "/")
		if len(parts) < 2 {
			return out, errors.New("unexpected app source root file")
		}
		switch parts[0] {
		case "nex", "lib", "mar":
			if !strings.HasSuffix(file.Path, ".hoon") || !utf8.Valid(file.Bytes) || bytes.ContainsRune(file.Bytes, 0) {
				return out, errors.New("invalid app Hoon source")
			}
		case "assets":
		default:
			return out, errors.New("app source must contain only code and declared assets")
		}
		for _, part := range parts[:len(parts)-1] {
			if parts[0] != "assets" && (part == "bundle" || strings.HasSuffix(part, "-bundle")) {
				return out, errors.New("this adapter does not compile nested code bundles")
			}
		}
	}
	instances, err := readBill(bill, index)
	if err != nil {
		return out, err
	}
	return Source{Version: 1, Manifest: manifest, Instances: instances}, nil
}

func readBill(raw []byte, files map[string]deskwork.File) ([]Instance, error) {
	if len(raw) == 0 || len(raw) > 8192 {
		return nil, errors.New("app requires a bounded bill.json")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	first, err := d.Token()
	if err != nil || first != json.Delim('{') {
		return nil, errors.New("app bill must be an object")
	}
	instances := []Instance{}
	seen := map[string]bool{}
	for d.More() {
		key, err := d.Token()
		if err != nil {
			return nil, errors.New("invalid app bill")
		}
		name, ok := key.(string)
		var neck string
		if !ok || !term.MatchString(name) || seen[name] || len(instances) >= MaxInstances || d.Decode(&neck) != nil || !validNeck(neck) {
			return nil, errors.New("invalid or duplicate app instance")
		}
		if _, ok := files["nex"+neck+".hoon"]; !ok {
			return nil, errors.New("app bill references a nexus outside the exact source")
		}
		seen[name] = true
		instances = append(instances, Instance{Name: name, Neck: neck})
	}
	last, err := d.Token()
	if err != nil || last != json.Delim('}') || len(instances) == 0 || d.Decode(new(any)) != io.EOF {
		return nil, errors.New("incomplete or trailing app bill")
	}
	sort.Slice(instances, func(i, j int) bool { return instances[i].Name < instances[j].Name })
	return instances, nil
}

func validNeck(neck string) bool {
	if len(neck) > 192 || !strings.HasPrefix(neck, "/") || path.Clean(neck) != neck {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(neck, "/"), "/")
	if len(parts) > 4 {
		return false
	}
	for _, part := range parts {
		if !term.MatchString(part) {
			return false
		}
	}
	return true
}

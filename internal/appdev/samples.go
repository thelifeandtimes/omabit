package appdev

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"sort"
	"strings"
	"unicode/utf8"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

//go:embed samples
var sampleFiles embed.FS

type SampleDefinition struct {
	Version        int               `json:"version"`
	Name           string            `json:"name"`
	Framework      string            `json:"framework"`
	APIVersion     int               `json:"api_version"`
	StateVersion   int               `json:"state_version"`
	Agent          string            `json:"agent"`
	ScryPath       string            `json:"scry_path"`
	WatchPath      string            `json:"watch_path"`
	PokeMark       string            `json:"poke_mark"`
	TestShips      []string          `json:"test_ships"`
	MaxEntries     int               `json:"max_entries"`
	VereSHA256     string            `json:"vere_sha256"`
	PillSHA256     string            `json:"pill_sha256"`
	UpstreamSource string            `json:"upstream_source"`
	Qualification  string            `json:"qualification"`
	Files          map[string]string `json:"files"`
}
type SampleSource struct {
	Path    string `json:"path"`
	SHA256  string `json:"sha256"`
	Content string `json:"content"`
}
type Sample struct {
	Definition       SampleDefinition  `json:"definition"`
	DefinitionSHA256 string            `json:"definition_sha256"`
	Source           deskwork.Manifest `json:"source"`
	Files            []SampleSource    `json:"files"`
	Notices          string            `json:"notices"`
}

func sampleHash(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

// ReadSample returns only a compiled named fixture, never arbitrary filesystem
// input. Source files are separate from notices/metadata and remain outside a
// live mount until explicitly staged through the normal policy-enforced path.
func ReadSample(name string) (Sample, error) {
	var out Sample
	if name != "gall-state-v1" {
		return out, errors.New("unsupported application sample")
	}
	root := "samples/" + name + "/"
	data, err := sampleFiles.ReadFile(root + "manifest.json")
	if err != nil {
		return out, err
	}
	if len(data) > 65536 || json.Unmarshal(data, &out.Definition) != nil {
		return out, errors.New("invalid frozen sample definition")
	}
	d := out.Definition
	if d.Version != 1 || d.Name != name || d.Framework != "gall" || d.APIVersion != 1 || d.StateVersion != 0 || len(d.Files) != 5 || d.Qualification != "current-workflow-pending" {
		return out, errors.New("unsupported frozen sample contract")
	}
	out.DefinitionSHA256 = sampleHash(data)
	out.Files = []SampleSource{}
	var files []deskwork.File
	total := 0
	err = fs.WalkDir(sampleFiles, root+"source", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel := strings.TrimPrefix(path, root+"source/")
		want, ok := d.Files[rel]
		if !ok || !deskwork.ValidPath(rel) {
			return errors.New("unlisted sample source")
		}
		data, err := sampleFiles.ReadFile(path)
		if err != nil {
			return err
		}
		total += len(data)
		if len(data) > 65536 || total > 65536 || !utf8.Valid(data) || sampleHash(data) != want {
			return errors.New("frozen sample bytes changed")
		}
		out.Files = append(out.Files, SampleSource{Path: rel, SHA256: want, Content: string(data)})
		files = append(files, deskwork.File{Path: rel, SHA256: want, Bytes: int64(len(data))})
		return nil
	})
	if err != nil {
		return Sample{}, err
	}
	if len(files) != len(d.Files) {
		return Sample{}, errors.New("incomplete sample source")
	}
	sort.Slice(out.Files, func(i, j int) bool { return out.Files[i].Path < out.Files[j].Path })
	out.Source, err = deskwork.NewManifest(files)
	if err != nil {
		return Sample{}, err
	}
	notices, err := sampleFiles.ReadFile(root + "THIRD-PARTY-NOTICES.md")
	if err != nil || len(notices) > 8192 {
		return Sample{}, errors.New("sample notices unavailable")
	}
	out.Notices = string(notices)
	return out, nil
}

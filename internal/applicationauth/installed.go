package applicationauth

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"unicode"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
)

const DeclarationName = "application-connection.json"
const OwnerAuthority = "eyre-owner-session"

// Declaration is a local plugin capability, not an installation catalog. The
// entire declaration and inventory are included in the approved artifact hash.
type Declaration struct {
	Version   int      `json:"version"`
	Runtime   string   `json:"runtime"`
	Authority string   `json:"authority"`
	Entry     string   `json:"entry"`
	Files     []string `json:"files"`
}

type InstalledPlugin struct {
	Target collection.ApplicationTarget `json:"target"`
	Name   string                       `json:"name"`
	Status string                       `json:"status"`
}

// ResolveInstalled never executes plugin code. The directory must be the
// manifest ID beneath the selected Omarchy plugin root; no arbitrary path/argv
// is accepted. A later explicit approval must compare the returned digest.
func ResolveInstalled(root, id string, environment map[string]string) (*Artifact, InstalledPlugin, error) {
	info := InstalledPlugin{Target: collection.ApplicationTarget{PluginID: id}, Status: "invalid_declaration"}
	if !validInstalledID(id) || !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return nil, info, ErrArtifact
	}
	path := filepath.Join(root, id)
	if resolved, err := filepath.EvalSymlinks(path); err != nil || resolved != path {
		return nil, info, ErrArtifact
	}
	reader := &Artifact{root: path}
	data, err := reader.read(DeclarationName)
	if err != nil {
		return nil, info, ErrArtifact
	}
	var declaration Declaration
	if !strictDeclaration(data, &declaration) || declaration.Version != 1 || declaration.Runtime != "python3" || declaration.Authority != OwnerAuthority || !slices.Contains(declaration.Files, DeclarationName) || !slices.Contains(declaration.Files, "manifest.json") {
		return nil, info, ErrArtifact
	}
	files := append([]string(nil), declaration.Files...)
	// Profiles are instance metadata, so a canonical declaration works in a
	// separately labeled installation without rewriting its helper inventory.
	if _, err := os.Lstat(filepath.Join(path, "application-profile.json")); err == nil && !slices.Contains(files, "application-profile.json") {
		files = append(files, "application-profile.json")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, info, ErrArtifact
	}
	a, err := OpenReceiver(path, declaration.Entry, files, environment)
	if err != nil || a.Target().PluginID != id || !bytes.Equal(a.files[DeclarationName], data) {
		return nil, info, ErrArtifact
	}
	var manifest struct {
		Name string `json:"name"`
	}
	if json.Unmarshal(a.files["manifest.json"], &manifest) != nil || len(manifest.Name) > 160 {
		return nil, info, ErrArtifact
	}
	for _, r := range manifest.Name {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return nil, info, ErrArtifact
		}
	}
	if manifest.Name == "" {
		manifest.Name = id
	}
	return a, InstalledPlugin{Target: a.Target(), Name: manifest.Name, Status: "available"}, nil
}

func validInstalledID(id string) bool {
	return (collection.ApplicationTarget{PluginID: id, Profile: "default", SHA256: "0000000000000000000000000000000000000000000000000000000000000000"}).Valid() && id != "." && id != ".."
}

// DiscoverInstalled only advertises installed directories with a declaration.
// Invalid declarations remain visible as unavailable, without opening a link or
// executing a helper. Missing install roots are an ordinary empty result.
func DiscoverInstalled(root string, environment map[string]string) ([]InstalledPlugin, error) {
	result := []InstalledPlugin{}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return nil, ErrArtifact
	}
	resolved, err := filepath.EvalSymlinks(root)
	if errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err != nil || resolved != root {
		return nil, ErrArtifact
	}
	f, err := os.Open(root)
	if err != nil {
		return nil, ErrArtifact
	}
	defer f.Close()
	entries, err := f.ReadDir(513)
	if err != nil && err != io.EOF || len(entries) > 512 {
		return nil, ErrArtifact
	}
	for _, entry := range entries {
		if !entry.IsDir() || !validInstalledID(entry.Name()) {
			continue
		}
		if _, err := os.Lstat(filepath.Join(root, entry.Name(), DeclarationName)); errors.Is(err, os.ErrNotExist) {
			continue
		}
		_, info, _ := ResolveInstalled(root, entry.Name(), environment)
		result = append(result, info)
	}
	slices.SortFunc(result, func(a, b InstalledPlugin) int {
		if a.Target.PluginID < b.Target.PluginID {
			return -1
		}
		if a.Target.PluginID > b.Target.PluginID {
			return 1
		}
		return 0
	})
	return result, nil
}

func strictDeclaration(data []byte, value *Declaration) bool {
	// Refuse duplicate keys, including keys a permissive JSON reader overwrites.
	d := json.NewDecoder(bytes.NewReader(data))
	if token, err := d.Token(); err != nil || token != json.Delim('{') {
		return false
	}
	seen := map[string]bool{}
	for d.More() {
		token, err := d.Token()
		key, ok := token.(string)
		if err != nil || !ok || seen[key] {
			return false
		}
		seen[key] = true
		if d.Decode(new(json.RawMessage)) != nil {
			return false
		}
	}
	if token, err := d.Token(); err != nil || token != json.Delim('}') || d.Decode(new(any)) != io.EOF {
		return false
	}
	d = json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	return d.Decode(value) == nil && d.Decode(new(any)) == io.EOF
}

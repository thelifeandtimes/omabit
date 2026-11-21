package appdev

import (
	"embed"
	"encoding/json"
	"errors"
	"slices"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

// This is source metadata, not framework source, a downloader or an installer.
// The complete desk is verified separately under a startup-approved source root.
//
//go:embed frameworks/*.json
var frameworkFiles embed.FS

const GrubberyFramework = "grubbery-7f7701d-v1"
const grubberyDefinitionSHA256 = "ad2ea8615c95483232eb2a07c019fade42787775e9520b158e26c6cbc43cce77"

type Framework struct {
	Version         int               `json:"version"`
	Name            string            `json:"name"`
	Framework       string            `json:"framework"`
	Repository      string            `json:"repository"`
	Commit          string            `json:"commit"`
	SourceDirectory string            `json:"source_directory"`
	Agent           string            `json:"agent"`
	Kelvins         []int             `json:"kelvins"`
	FrameworkKelvin int               `json:"framework_kelvin"`
	HTTPContract    string            `json:"http_contract"`
	Source          deskwork.Manifest `json:"source"`
	License         string            `json:"license"`
	Qualification   string            `json:"qualification"`
}

func ReadFramework(name string) (Framework, error) {
	var out Framework
	if name != GrubberyFramework {
		return out, errors.New("unsupported frozen application framework")
	}
	data, err := frameworkFiles.ReadFile("frameworks/" + name + ".json")
	if err != nil || len(data) > 512<<10 || sampleHash(data) != grubberyDefinitionSHA256 || json.Unmarshal(data, &out) != nil {
		return Framework{}, errors.New("frozen framework definition changed")
	}
	if out.Version != 1 || out.Name != name || out.Framework != "grubbery" || out.Commit != "7f7701d546ce115eea5f37f66465b362cb121774" || out.SourceDirectory != "desk" || out.Agent != "grubbery" || out.HTTPContract != "grubbery-darts-7f7701d-v1" || out.FrameworkKelvin != 21000000 || !slices.Equal(out.Kelvins, []int{410, 409, 408}) || out.Source.Validate() != nil || len(out.Source.Files) != 1443 || out.Qualification != "native-workflow-pending" {
		return Framework{}, errors.New("unsupported frozen framework contract")
	}
	var size int64
	for _, file := range out.Source.Files {
		size += file.Bytes
	}
	if size != 8013304 {
		return Framework{}, errors.New("frozen framework size changed")
	}
	return out, nil
}

// VerifySource checks ordinary bounded source observed by the existing confined
// snapshot implementation. Equal version labels or a subset of the files never
// establish equality. This grants no policy and says nothing about compilation,
// activation, loaded app state, live permits or served bytes.
func (framework Framework) VerifySource(observed deskwork.Manifest) error {
	frozen, err := ReadFramework(framework.Name)
	if err != nil || framework.Source.Validate() != nil || framework.Source.Digest != frozen.Source.Digest || observed.Validate() != nil || observed.Digest != frozen.Source.Digest {
		return errors.New("source differs from the complete frozen framework desk")
	}
	return nil
}

package runtimeprofile

import "fmt"

const (
	ManifestDigest     = "sha256:47fbcf0710ded939931ce3c6408864cee5e8848c5765da0b51a90ca0b1ab1afd"
	Image              = "docker.io/tloncorp/vere@" + ManifestDigest
	ConfigDigest       = "sha256:d6cc5107ed7fa92174e336af9c7f25ee433073bd9961e1c3ed23a225b866b245"
	Platform           = "linux/amd64"
	VereVersion        = "4.6-5a7614f"
	ProfilesFeature    = "runtime-profiles-v1"
	OfficialProfile    = "vere-4.6-5a7614f"
	JSONFixProfile     = "vere-4.6-jsonfix-0774857c"
	JSONFixRequirement = "vere-json-array-error-1145-v1"
)

// Profile is a reviewed, immutable execution contract. IDs never follow a
// mutable channel or version tag. Extending this catalogue requires a reviewed
// release, artifact provenance and runtime qualification; callers cannot supply
// paths, commands, images or replacement metadata.
type Profile struct {
	Format         int    `json:"format"`
	ID             string `json:"id"`
	Kind           string `json:"kind"`
	Platform       string `json:"platform"`
	VereVersion    string `json:"vere_version"`
	Image          string `json:"image"`
	ManifestDigest string `json:"manifest_digest"`
	ConfigDigest   string `json:"config_digest"`
	BinarySHA256   string `json:"binary_sha256"`
	SourceBase     string `json:"source_base"`
	PatchRevision  string `json:"patch_revision,omitempty"`
	Requirement    string `json:"requirement,omitempty"`
	ReviewURL      string `json:"review_url,omitempty"`
	ArchiveSHA256  string `json:"archive_sha256,omitempty"`
	ArchiveBytes   int64  `json:"archive_bytes,omitempty"`
}

// Resolve preserves the exact historical official runtime for omitted fields.
// A future default must never change what an existing empty selection means.
func Resolve(id string) (Profile, error) {
	switch id {
	case "", OfficialProfile:
		return Profile{Format: 1, ID: OfficialProfile, Kind: "official", Platform: Platform,
			VereVersion: VereVersion, Image: Image, ManifestDigest: ManifestDigest, ConfigDigest: ConfigDigest,
			BinarySHA256: "b5c54439ae5b9f0ff94d75319e67e04f04369776fa5e2180a5ab1a5ebd4c2ff0",
			SourceBase:   "5a7614f69bbc783736b08965fdc0a2534bb42e82"}, nil
	case JSONFixProfile:
		return Profile{Format: 1, ID: JSONFixProfile, Kind: "reviewed", Platform: Platform,
			VereVersion:    "4.6-82f4558",
			Image:          "omabit/vere@sha256:2b6b7939b09aa0be68bf5a80b796f89382040aeb0d3469c032063a4c93fb5b1d",
			ManifestDigest: "sha256:2b6b7939b09aa0be68bf5a80b796f89382040aeb0d3469c032063a4c93fb5b1d",
			ConfigDigest:   "sha256:9aff0a010f4bfecff2233cc5508a0a6deec078239038456c94bbe0666b7a6f58",
			BinarySHA256:   "0774857cdb4460fc288bfa55e778249456a7d8bd7b6203922455d887a92fc96c",
			SourceBase:     "5a7614f69bbc783736b08965fdc0a2534bb42e82",
			PatchRevision:  "fcc9dd214ae4af07e683490a6cc3f2ded1ab7d47",
			Requirement:    JSONFixRequirement,
			ReviewURL:      "https://github.com/nisfeb/talon/issues/112#issuecomment-6997450705",
			ArchiveSHA256:  "c8a412c42d53fa0c9a8c97aa55a580ddf2864bbdf32f80074dfb6bdfd837f510", ArchiveBytes: 54750277}, nil
	default:
		return Profile{}, fmt.Errorf("unknown immutable runtime profile %q; use a matching runner and reviewed catalogue", id)
	}
}

func Catalogue() []Profile {
	a, _ := Resolve(OfficialProfile)
	b, _ := Resolve(JSONFixProfile)
	return []Profile{a, b}
}

func (p Profile) AcceptsImageID(id string) bool {
	return id == p.ConfigDigest || id == p.ManifestDigest
}

// CompatibleSuccessor requires actual correction evidence, never version order.
func CompatibleSuccessor(from, to string) error {
	a, err := Resolve(from)
	if err != nil {
		return err
	}
	b, err := Resolve(to)
	if err != nil {
		return err
	}
	if a.Requirement != "" && b.Requirement != a.Requirement {
		return fmt.Errorf("pier requires %s: stock Vere can crash replay; restoring a pre-patch pier is a separate reviewed recovery, not a runtime downgrade", a.Requirement)
	}
	return nil
}

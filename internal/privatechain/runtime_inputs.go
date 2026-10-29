package privatechain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"path"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

const RuntimeInputFormat = "private-chain-runtime-inputs-v1"
const MaxRuntimeInputManifest = 8 * 1024 * 1024
const MaxRuntimeInputFiles = 40000
const MaxRuntimeInputBytes int64 = 1024 * 1024 * 1024

// RuntimeInputs describes an inert, locally reviewed dependency generation. Its
// hash is not a signature or a claim of runtime qualification. In particular the
// current gateway programs are guest-only candidates, not installed services.
// An eventual start must bind its own project/operation and freshly revalidate
// inputs and retained state; an inspection receipt cannot authorize that start.
type RuntimeInputs struct {
	Version    int                `json:"version"`
	Format     string             `json:"format"`
	Platform   string             `json:"platform"`
	Image      string             `json:"image"`
	Foundation Foundation         `json:"foundation"`
	Files      []RuntimeInputFile `json:"files"`
}

type RuntimeInputFile struct {
	Path       string `json:"path"`
	Bytes      int64  `json:"bytes"`
	SHA256     string `json:"sha256"`
	Executable bool   `json:"executable"`
}

// RuntimeInputInspection intentionally contains no local paths, private project
// values or inferred startup authority. It describes one read-only inspection.
type RuntimeInputInspection struct {
	Format            string `json:"format"`
	ManifestSHA256    string `json:"manifest_sha256"`
	FoundationDigest  string `json:"foundation_digest"`
	Files             int    `json:"files"`
	Bytes             int64  `json:"bytes"`
	RuntimeAuthorized bool   `json:"runtime_authorized"`
}

var errRuntimeInputs = errors.New("private-chain runtime inputs differ from the bounded reviewed manifest")

var runtimeInputPins = map[string]string{
	"vere":                       "b5c54439ae5b9f0ff94d75319e67e04f04369776fa5e2180a5ab1a5ebd4c2ff0",
	"boot.pill":                  runtimecandidate.PillSHA256,
	"node/usr/bin/node":          "6dcdb9bbe2bf22bd053ba06d5ba7b8a2d92b2d3d8c714f3b2670be3d2728c9c8",
	"tools/package-lock.json":    "3775ca05fc44270b64dac13bee82042eb86e6ce16dba3e9698157a9a1b35c3a8",
	"source/bridge/package.json": "362f076b8b046606f9c84ef0f223c5de4992a0f628dc8f52fbabf8668cdeb7f2",
	"tools/node_modules/ganache/node_modules/leveldown/prebuilds/linux-x64/node.napi.glibc.node": "35164284059b66d0de99a33674d99ff2fafeb27b928e9014adb6a06838d7c05a",
}

var runtimeInputRequired = []string{
	"gateway/docker_gateway.cjs", "gateway/docker_gateway_resume.cjs", "gateway/gateway_resume_contract.cjs",
	"node/usr/lib/ld-linux-x86-64.so.2", "node-closure.json",
	"compiled/compiler-output.json", "compiled/result.json",
	"deployment/result.json", "deployment/chain/CURRENT",
}

func inputPath(name string) bool {
	if name == "" || len(name) > 1024 || !utf8.ValidString(name) || strings.IndexFunc(name, unicode.IsControl) >= 0 || name != path.Clean(name) || strings.HasPrefix(name, "/") || strings.Contains(name, "\\") {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if part == "." || part == ".." || part == "" {
			return false
		}
	}
	if name == "vere" || name == "boot.pill" || name == "node-closure.json" || name == "source/bridge/package.json" {
		return true
	}
	for _, prefix := range []string{"node/", "tools/", "gateway/", "compiled/", "deployment/"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func (m RuntimeInputs) validate() (string, int64, error) {
	foundation, err := m.Foundation.Digest()
	if err != nil || m.Version != 1 || m.Format != RuntimeInputFormat || m.Platform != runtimecandidate.Platform || m.Image != runtimecandidate.Image || len(m.Files) == 0 || len(m.Files) > MaxRuntimeInputFiles {
		return "", 0, errRuntimeInputs
	}
	seen := map[string]RuntimeInputFile{}
	var total int64
	previous := ""
	for _, f := range m.Files {
		if !inputPath(f.Path) || f.Path <= previous || f.Bytes < 0 || f.Bytes > MaxRuntimeInputBytes-total || !hash(f.SHA256, false) {
			return "", 0, errRuntimeInputs
		}
		previous = f.Path
		for p := path.Dir(f.Path); p != "."; p = path.Dir(p) {
			if _, exists := seen[p]; exists {
				return "", 0, errRuntimeInputs
			}
		}
		seen[f.Path] = f
		total += f.Bytes
	}
	for name, pin := range runtimeInputPins {
		f, ok := seen[name]
		if !ok || f.SHA256 != pin || f.Bytes == 0 {
			return "", 0, errRuntimeInputs
		}
	}
	for _, name := range runtimeInputRequired {
		if f, ok := seen[name]; !ok || f.Bytes == 0 {
			return "", 0, errRuntimeInputs
		}
	}
	if seen["boot.pill"].Bytes != runtimecandidate.PillBytes || seen["boot.pill"].Executable || !seen["vere"].Executable || !seen["node/usr/bin/node"].Executable || !seen["node/usr/lib/ld-linux-x86-64.so.2"].Executable {
		return "", 0, errRuntimeInputs
	}
	return foundation, total, nil
}

// EncodeRuntimeInputs sorts a copy and produces the single supported canonical
// encoding. It does not inspect files or establish trust in their contents.
func EncodeRuntimeInputs(m RuntimeInputs) ([]byte, string, error) {
	m.Files = slices.Clone(m.Files)
	slices.SortFunc(m.Files, func(a, b RuntimeInputFile) int { return strings.Compare(a.Path, b.Path) })
	if _, _, err := m.validate(); err != nil {
		return nil, "", err
	}
	raw, err := json.Marshal(m)
	if err != nil || len(raw) > MaxRuntimeInputManifest {
		return nil, "", errRuntimeInputs
	}
	sum := sha256.Sum256(raw)
	return raw, hex.EncodeToString(sum[:]), nil
}

// DecodeRuntimeInputs requires a separately supplied exact-byte review hash and
// the caller's reviewed foundation. Neither is learned from this same document.
// Unknown/duplicate/case-alias fields and alternate encodings refuse unchanged.
func DecodeRuntimeInputs(raw []byte, expectedSHA, expectedFoundation string) (RuntimeInputs, error) {
	var m RuntimeInputs
	if len(raw) > MaxRuntimeInputManifest || !hash(expectedSHA, false) || !hash(expectedFoundation, false) {
		return m, errRuntimeInputs
	}
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != expectedSHA || !uniqueJSONDepth(raw, 6) {
		return m, errRuntimeInputs
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&m) != nil || d.Decode(new(any)) != io.EOF {
		return RuntimeInputs{}, errRuntimeInputs
	}
	foundation, _, err := m.validate()
	canonical, _ := json.Marshal(m)
	if err != nil || foundation != expectedFoundation || !bytes.Equal(raw, canonical) {
		return RuntimeInputs{}, errRuntimeInputs
	}
	return m, nil
}

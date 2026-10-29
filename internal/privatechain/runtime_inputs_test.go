package privatechain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

func runtimeManifestFixture(t *testing.T) RuntimeInputs {
	t.Helper()
	raw, err := os.ReadFile("testdata/qualified-foundation.json")
	if err != nil {
		t.Fatal(err)
	}
	f, err := DecodeFoundation(raw)
	if err != nil {
		t.Fatal(err)
	}
	m := RuntimeInputs{Version: 1, Format: RuntimeInputFormat, Platform: runtimecandidate.Platform, Image: runtimecandidate.Image, Foundation: f}
	for name, pin := range runtimeInputPins {
		file := RuntimeInputFile{Path: name, Bytes: 1, SHA256: pin, Executable: name == "vere" || name == "node/usr/bin/node"}
		if name == "boot.pill" {
			file.Bytes = runtimecandidate.PillBytes
		}
		m.Files = append(m.Files, file)
	}
	for _, name := range runtimeInputRequired {
		m.Files = append(m.Files, RuntimeInputFile{Path: name, Bytes: 1, SHA256: strings.Repeat("a", 64), Executable: name == "node/usr/lib/ld-linux-x86-64.so.2"})
	}
	slices.SortFunc(m.Files, func(a, b RuntimeInputFile) int { return strings.Compare(a.Path, b.Path) })
	return m
}

func inputSHA(raw []byte) string { h := sha256.Sum256(raw); return hex.EncodeToString(h[:]) }

func TestRuntimeInputsReviewedIdentity(t *testing.T) {
	m := runtimeManifestFixture(t)
	before, _ := json.Marshal(m)
	slices.Reverse(m.Files)
	raw, sha, err := EncodeRuntimeInputs(m)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != string(before) {
		t.Fatal("canonical output differs")
	}
	if m.Files[0].Path == runtimeManifestFixture(t).Files[0].Path {
		t.Fatal("encoder changed caller-owned ordering")
	}
	foundation, _ := m.Foundation.Digest()
	got, err := DecodeRuntimeInputs(raw, sha, foundation)
	if err != nil || len(got.Files) != len(m.Files) {
		t.Fatal("review decode", err)
	}
	for _, pair := range [][2]string{{"", foundation}, {strings.Repeat("b", 64), foundation}, {sha, ""}, {sha, strings.Repeat("b", 64)}} {
		if _, err := DecodeRuntimeInputs(raw, pair[0], pair[1]); err == nil {
			t.Fatal("unreviewed or foreign input admitted")
		}
	}
}

func TestRuntimeInputsMetadataRefusals(t *testing.T) {
	changes := map[string]func(*RuntimeInputs){
		"version":       func(m *RuntimeInputs) { m.Version++ },
		"format":        func(m *RuntimeInputs) { m.Format = "fake-galaxies-v1" },
		"platform":      func(m *RuntimeInputs) { m.Platform = "linux/arm64" },
		"mutable-image": func(m *RuntimeInputs) { m.Image = "docker.io/tloncorp/vere:latest" },
		"foundation":    func(m *RuntimeInputs) { m.Foundation.ChainID = 1 },
		"missing-role":  func(m *RuntimeInputs) { m.Files = m.Files[1:] },
		"duplicate":     func(m *RuntimeInputs) { m.Files = append(m.Files, m.Files[0]) },
		"negative-size": func(m *RuntimeInputs) { m.Files[0].Bytes = -1 },
		"overflow":      func(m *RuntimeInputs) { m.Files[0].Bytes = 1<<63 - 1 },
		"budget":        func(m *RuntimeInputs) { m.Files[0].Bytes = MaxRuntimeInputBytes },
		"invalid-hash":  func(m *RuntimeInputs) { m.Files[0].SHA256 = "private-data" },
		"changed-pin": func(m *RuntimeInputs) {
			for i := range m.Files {
				if m.Files[i].Path == "vere" {
					m.Files[i].SHA256 = strings.Repeat("f", 64)
				}
			}
		},
		"boot-payload-size": func(m *RuntimeInputs) {
			for i := range m.Files {
				if m.Files[i].Path == "boot.pill" {
					m.Files[i].Bytes--
				}
			}
		},
		"nonexecutable-node": func(m *RuntimeInputs) {
			for i := range m.Files {
				if m.Files[i].Path == "node/usr/bin/node" {
					m.Files[i].Executable = false
				}
			}
		},
		"executable-pill": func(m *RuntimeInputs) {
			for i := range m.Files {
				if m.Files[i].Path == "boot.pill" {
					m.Files[i].Executable = true
				}
			}
		},
		"ancestor-file": func(m *RuntimeInputs) {
			m.Files = append(m.Files, RuntimeInputFile{Path: "node/usr", Bytes: 1, SHA256: strings.Repeat("a", 64)})
		},
		"too-many-files": func(m *RuntimeInputs) { m.Files = make([]RuntimeInputFile, MaxRuntimeInputFiles+1) },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			m := runtimeManifestFixture(t)
			change(&m)
			if _, _, err := EncodeRuntimeInputs(m); err == nil {
				t.Fatal("invalid review accepted")
			}
		})
	}
	for _, p := range []string{"/etc/passwd", "../secret", "node/../../secret", "node//a", "node/./a", "node/a\x00b", "node/a\nb", "node/a\tb", "node/a\u0085b", "node/\xff", "node\\file", "piers/zod", "secrets/key", "manifest.json", "."} {
		t.Run(p, func(t *testing.T) {
			m := runtimeManifestFixture(t)
			m.Files = append(m.Files, RuntimeInputFile{Path: p, Bytes: 1, SHA256: strings.Repeat("a", 64)})
			if _, _, err := EncodeRuntimeInputs(m); err == nil {
				t.Fatal("unsafe path admitted")
			}
		})
	}
}

func TestRuntimeInputsUnambiguousWireFormat(t *testing.T) {
	m := runtimeManifestFixture(t)
	raw, _, err := EncodeRuntimeInputs(m)
	if err != nil {
		t.Fatal(err)
	}
	foundation, _ := m.Foundation.Digest()
	changes := map[string][]byte{
		"unsupported-version": []byte(strings.Replace(string(raw), `"version":1`, `"version":2`, 1)),
		"duplicate":           []byte(strings.Replace(string(raw), `"version":1`, `"version":1,"version":1`, 1)),
		"alias":               []byte(strings.Replace(string(raw), `"version":1`, `"Version":1`, 1)),
		"unknown":             []byte(strings.Replace(string(raw), `"version":1`, `"version":1,"secret":"PRIVATE-SENTINEL"`, 1)),
		"whitespace":          append(append([]byte{}, raw...), '\n'),
		"two-values":          append(append([]byte{}, raw...), raw...),
		"too-large":           make([]byte, MaxRuntimeInputManifest+1),
		"null":                []byte("null"),
	}
	for name, bad := range changes {
		t.Run(name, func(t *testing.T) {
			_, err := DecodeRuntimeInputs(bad, inputSHA(bad), foundation)
			if err == nil || strings.Contains(err.Error(), "PRIVATE-SENTINEL") {
				t.Fatal("invalid input accepted or reflected", err)
			}
		})
	}
	slices.Reverse(m.Files)
	bad, _ := json.Marshal(m)
	if _, err := DecodeRuntimeInputs(bad, inputSHA(bad), foundation); err == nil {
		t.Fatal("noncanonical file ordering accepted")
	}
}

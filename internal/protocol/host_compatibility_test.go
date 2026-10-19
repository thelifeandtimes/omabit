package protocol

import (
	"encoding/json"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
)

func TestHostCompatibilityPublicBoundedWireReport(t *testing.T) {
	r := hostcompat.Inspect(fstest.MapFS{}, "linux", "amd64")
	r.Engine = &hostcompat.Engine{DataRoot: "/private/owner/data"}
	public := PublicHostCompatibility(r)
	if public.Engine.DataRoot != "" || r.Engine.DataRoot != "/private/owner/data" {
		t.Fatal("public projection leaked or mutated private evidence")
	}
	bound := HostCompatibilityReport{SchemaVersion: 1, ProtocolVersion: 1, RunnerID: "runner", Mode: "development-managed", Features: []string{HostCompatibilityFeature}, Report: public, ExpiresAt: r.ObservedAt.Add(HostCompatibilityMaxAge)}
	if !ValidHostCompatibility(bound) {
		t.Fatal("valid metadata evidence rejected")
	}
	data, err := json.Marshal(bound)
	if err != nil || strings.Contains(string(data), "/private/") || !strings.Contains(string(data), `"schema_version":1`) || !strings.Contains(string(data), `"protocol_version":1`) || !strings.Contains(string(data), `"runner_id":"runner"`) {
		t.Fatal(string(data), err)
	}
	for _, text := range []string{"\x1b[31muntrusted", "bidi\u202esecret", strings.Repeat("x", 1025)} {
		bad := bound
		bad.Report.Guidance = []string{text}
		if ValidHostCompatibility(bad) {
			t.Fatal("unbounded/control diagnostic accepted")
		}
	}
	bound.ExpiresAt = r.ObservedAt.Add(2 * HostCompatibilityMaxAge)
	if ValidHostCompatibility(bound) {
		t.Fatal("unbounded freshness claim accepted")
	}
	bound.ExpiresAt = r.ObservedAt.Add(time.Second)
	oldMetadata := r.ObservedAt.Add(-HostCompatibilityMaxAge)
	bound.MetadataObservedAt = &oldMetadata
	if ValidHostCompatibility(bound) {
		t.Fatal("live refresh extended stale native evidence")
	}
}

func TestHostCompatibilityNativeEvidenceVocabularyAndPrivateProofOmission(t *testing.T) {
	r := hostcompat.Inspect(fstest.MapFS{}, "linux", "amd64")
	r.Provenance = "verified"
	r.Packages = []hostcompat.Package{
		{Role: "engine", Name: "docker.io", Version: "5:29.7.2+dfsg1-1", UpstreamVersion: "29.7.2+dfsg1", Provenance: "verified", Detail: "Authenticated cached evidence; method: apt-archive-keyring-inrelease+sha256-index+sha256-deb+sha256-installed-payload."},
		{Role: "cli", Name: "docker", Version: "29.7.2-1", UpstreamVersion: "29.7.2", Provenance: "verified", Detail: "Authenticated cached evidence; method: pacman-keyring-gpg-trusted-signature+sha256-installed-payload."},
	}
	r.Checks = []hostcompat.Check{
		{ID: "packages.receipt", Status: "current", Detail: "Age does not expire authorization."},
		{ID: "packages.receipt", Status: "historical", Detail: "Historical evidence remains authorized while package, executable and public authority bindings match."},
	}
	bound := HostCompatibilityReport{SchemaVersion: 1, ProtocolVersion: 1, RunnerID: "runner", Mode: "development-managed", Features: []string{HostCompatibilityFeature}, Report: r, ExpiresAt: r.ObservedAt.Add(5 * time.Second)}
	if !ValidHostCompatibility(bound) {
		t.Fatal("native build labels or authentication diagnostics rejected")
	}
	data, err := json.Marshal(bound)
	if err != nil {
		t.Fatal(err)
	}
	// Private provider proof is not part of the report wire type. A peer's extra
	// proof/receipt fields must not survive the typed decode into fleet output.
	var wire map[string]any
	if err := json.Unmarshal(data, &wire); err != nil {
		t.Fatal(err)
	}
	private := "https://private-user:private-secret@example.invalid/package"
	body := wire["report"].(map[string]any)
	body["proof_scope"] = map[string]any{"uri": private}
	body["packages"].([]any)[0].(map[string]any)["proof_scope"] = private
	data, err = json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	var decoded HostCompatibilityReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	decoded.Report = PublicHostCompatibility(decoded.Report)
	data, err = json.Marshal(decoded)
	if err != nil || !ValidHostCompatibility(decoded) || strings.Contains(string(data), "private-secret") || strings.Contains(string(data), "proof_scope") || decoded.Report.Packages[0].Version != "5:29.7.2+dfsg1-1" {
		t.Fatal("public report lost native metadata or exposed private proof", string(data), err)
	}
	decoded.Report.Provenance = strings.Repeat("x", 1025)
	if ValidHostCompatibility(decoded) {
		t.Fatal("provenance status escaped the string bound")
	}
}

package grubbery_test

import (
	"encoding/json"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

var foundationFixture = []string{"mar/hoon.hoon", "mar/kelvin.hoon", "mar/mime.hoon", "mar/tang.hoon"}

func buildFixture(t *testing.T, p grubbery.Plan) map[string]any {
	t.Helper()
	namespace, err := p.Binding.ReleaseRoot(p.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	rows := []any{}
	for _, name := range foundationFixture {
		rows = append(rows, map[string]any{"source": "/" + name, "status": "vase", "error": nil})
	}
	for _, f := range p.Source.Manifest.Files {
		status := "mime"
		if strings.HasPrefix(f.Path, "nex/") || strings.HasPrefix(f.Path, "lib/") || strings.HasPrefix(f.Path, "mar/") {
			status = "vase"
		}
		rows = append(rows, map[string]any{"source": "/" + f.Path, "status": status, "error": nil})
	}
	return map[string]any{"namespace": namespace, "total": len(rows), "failed": 0, "files": rows}
}
func encoded(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func TestPinnedCompilerReportRequiresActualCompleteSourceAndFoundations(t *testing.T) {
	p, _ := plan(t)
	native := buildFixture(t, p)
	namespace, _ := p.Binding.ReleaseRoot(p.RequestID)
	report, err := grubbery.ParseBuildReport(encoded(t, native), namespace)
	if err != nil || report.CheckStaged(p.Source, false) != nil {
		t.Fatal("native observed shape did not verify", err)
	}
	rows := native["files"].([]any)
	rows[0].(map[string]any)["status"] = "tang"
	rows[0].(map[string]any)["error"] = []string{"PRIVATE compiler diagnostic"}
	native["failed"] = 1
	report, err = grubbery.ParseBuildReport(encoded(t, native), namespace)
	if err != nil {
		t.Fatal(err)
	}
	if report.CheckStaged(p.Source, false) == nil {
		t.Fatal("compile failure became healthy")
	}
	if strings.Contains(string(encoded(t, report)), "PRIVATE") {
		t.Fatal("compiler text implicitly exported")
	}
}
func TestCompilerReportRefusesAmbiguityWrongNamespaceAndMissingEvidence(t *testing.T) {
	p, _ := plan(t)
	namespace, _ := p.Binding.ReleaseRoot(p.RequestID)
	for _, kind := range []string{"wrong-namespace", "alias-key", "unknown-key", "wrong-count", "wrong-failure-count", "duplicate-source", "outside-source", "wrong-kind", "missing-foundation", "missing-error", "non-null-success-error"} {
		t.Run(kind, func(t *testing.T) {
			native := buildFixture(t, p)
			rows := native["files"].([]any)
			switch kind {
			case "wrong-namespace":
				native["namespace"] = "/other"
			case "alias-key":
				native["Namespace"] = native["namespace"]
				delete(native, "namespace")
			case "unknown-key":
				native["invented"] = true
			case "wrong-count":
				native["total"] = 1
			case "wrong-failure-count":
				native["failed"] = 1
			case "duplicate-source":
				rows[1].(map[string]any)["source"] = rows[0].(map[string]any)["source"]
			case "outside-source":
				rows[0].(map[string]any)["source"] = "/../private"
			case "wrong-kind":
				rows[0].(map[string]any)["status"] = "mime"
			case "missing-foundation":
				native["files"] = rows[1:]
				native["total"] = len(rows) - 1
			case "missing-error":
				delete(rows[0].(map[string]any), "error")
			case "non-null-success-error":
				rows[0].(map[string]any)["error"] = []string{}
			}
			report, err := grubbery.ParseBuildReport(encoded(t, native), namespace)
			if err == nil && report.CheckStaged(p.Source, false) == nil {
				t.Fatal("changed compiler evidence accepted")
			}
		})
	}
	raw := encoded(t, buildFixture(t, p))
	duplicate := append([]byte(`{"failed":0,`), raw[1:]...)
	if _, err := grubbery.ParseBuildReport(duplicate, namespace); err == nil {
		t.Fatal("duplicate key accepted")
	}
}
func stagedTreeFixture(p grubbery.Plan) map[string]any {
	root := map[string]any{"neck": "code", "files": map[string]string{}, "dirs": map[string]any{}}
	insert := func(name, mark string) {
		parts := strings.Split(name, "/")
		node := root
		for _, part := range parts[:len(parts)-1] {
			children := node["dirs"].(map[string]any)
			if children[part] == nil {
				children[part] = map[string]any{"neck": nil, "files": map[string]string{}, "dirs": map[string]any{}}
			}
			node = children[part].(map[string]any)
		}
		node["files"].(map[string]string)[parts[len(parts)-1]] = mark
	}
	for _, name := range foundationFixture {
		insert(name, "hoon")
	}
	for _, file := range p.Source.Manifest.Files {
		mark := "mime"
		if strings.HasPrefix(file.Path, "nex/") || strings.HasPrefix(file.Path, "lib/") || strings.HasPrefix(file.Path, "mar/") {
			mark = "hoon"
		}
		insert(file.Path, mark)
	}
	return root
}
func TestStagedTreeSeparatesFrameworkSeedsAndRejectsUnexpectedPaths(t *testing.T) {
	p, _ := plan(t)
	for _, kind := range []string{"complete", "partial", "foreign-file", "foreign-dir", "nested-nexus", "wrong-root", "missing-foundation", "wrong-mark", "alias-key", "duplicate"} {
		t.Run(kind, func(t *testing.T) {
			tree := stagedTreeFixture(p)
			dirs := tree["dirs"].(map[string]any)
			switch kind {
			case "partial":
				delete(tree["files"].(map[string]string), "bill.json")
			case "foreign-file":
				tree["files"].(map[string]string)["unowned.json"] = "mime"
			case "foreign-dir":
				dirs["unowned"] = map[string]any{"dirs": map[string]any{}}
			case "nested-nexus":
				dirs["nex"].(map[string]any)["neck"] = "shell"
			case "wrong-root":
				tree["neck"] = "desk"
			case "missing-foundation":
				delete(dirs["mar"].(map[string]any)["files"].(map[string]string), "mime.hoon")
			case "wrong-mark":
				dirs["nex"].(map[string]any)["files"].(map[string]string)["state.hoon"] = "mime"
			case "alias-key":
				tree["Neck"] = tree["neck"]
				delete(tree, "neck")
			}
			raw := encoded(t, tree)
			if kind == "duplicate" {
				raw = append([]byte(`{"neck":"code",`), raw[1:]...)
			}
			parsed, err := grubbery.ParseDirectory(raw)
			valid := err == nil && grubbery.CheckStagedTree(p.Source, parsed, true, false) == nil
			if (kind == "complete") != valid {
				t.Fatal("staged tree qualification wrong", kind, err)
			}
			if kind == "partial" && (err != nil || grubbery.CheckStagedTree(p.Source, parsed, false, false) != nil) {
				t.Fatal("bounded staging subset refused", err)
			}
		})
	}
}
func TestNativeJSONPreservesLargeIntegersAndRefusesDuplicateKeys(t *testing.T) {
	if !grubbery.EqualNativeJSON([]byte(`{"version":"v2","revision":18446744073709551615}`), []byte(`{"revision":18446744073709551615,"version":"v2"}`)) {
		t.Fatal("valid JSON order/numeric identity changed")
	}
	for _, raw := range []string{`{"revision":18446744073709551614,"version":"v2"}`, `{"revision":18446744073709551615,"revision":0,"version":"v2"}`, `{"Revision":18446744073709551615,"version":"v2"}`} {
		if grubbery.EqualNativeJSON([]byte(raw), []byte(`{"revision":18446744073709551615,"version":"v2"}`)) {
			t.Fatal("changed integer/key accepted")
		}
	}
}
func TestNativeSourceRejectsForcedFoundationalMarkOverrides(t *testing.T) {
	for _, name := range foundationFixture {
		m, files := source(t, `{"state":"/state"}`, deskwork.SourceFile{Path: name, Bytes: []byte(":: caller override\n")})
		attested, err := grubbery.ReviewSource(m, files)
		if err != nil {
			t.Fatal(err)
		}
		if grubbery.CheckNativeSource(attested) == nil {
			t.Fatal("kernel-overwritten source advertised as supported")
		}
	}
}

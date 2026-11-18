package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestCatalogAndInputGuardsStayOffline(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "must-not-exist")
	s, err := New(Options{ConfigDir: dir, RequireAllowed: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Catalog()) < 35 {
		t.Fatal("operations missing")
	}
	for _, tc := range []struct{ id, input, code string }{
		{"unknown", `{}`, "unknown_operation"},
		{"project.status.v1", `{}`, "invalid_input"},
		{"project.status.v1", `{"name":"dev","runner_id":"runner","project_id":"project","extra":true}`, "invalid_input"},
		{"project.status.v1", `{"name":"dev","runner_id":"runner","project_id":"project"}`, "target_not_allowed"},
		{"project.stop-reconcile.v1", `{}`, "mutation_disabled"},
		{"runners.list.v1", `null`, "invalid_input"},
		{"runners.list.v1", `{} {}`, "invalid_input"},
		{"project.create-wait.v1", `{"name":"dev","runner_id":"runner","plan":{},"request_id":"parent","timeout_millis":30001}`, "invalid_input"},
	} {
		out := s.Call(context.Background(), tc.id, json.RawMessage(tc.input))
		if out.Error == nil || out.Error.Code != tc.code || out.Error.Dispatch != "not-dispatched" {
			t.Fatalf("%s: %+v", tc.id, out)
		}
	}
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("offline discovery/refusal created registry")
	}
	for _, d := range s.Catalog() {
		if d.ID == "" || d.InputSchema == nil || d.OutputSchema == nil || len(d.CLI) == 0 || d.MCP == "" || d.Bindings == "" || d.Replay == "" || len(d.Effects) == 0 {
			t.Fatalf("incomplete descriptor %+v", d)
		}
	}
}

func TestSharedLegacyCreationBindingRetainsAllEffects(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, binding := range s.Discovery().CLIBindings {
		if binding.CLI == "urbit-collector project-create" {
			if len(binding.Operations) != 2 || !slices.Contains(binding.Effects, "journal-write") || !slices.Contains(binding.Effects, "runtime-mutation") || !slices.Contains(binding.Effects, "network-download") {
				t.Fatal("shared legacy alias underreported its effects", binding)
			}
			return
		}
	}
	t.Fatal("legacy creation binding missing")
}

func TestCatalogAccountsForEveryDocumentedCLI(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	bindings := map[string]CLIBinding{}
	for _, b := range s.Discovery().CLIBindings {
		if _, exists := bindings[b.CLI]; exists {
			t.Fatalf("duplicate CLI %s", b.CLI)
		}
		bindings[b.CLI] = b
		if len(b.Operations) == 0 && b.ExclusionReason == "" {
			t.Fatalf("silent exclusion: %s", b.CLI)
		}
	}
	for _, tool := range []string{"omabit", "urbit-collector", "urbit-runner"} {
		body, err := os.ReadFile(filepath.Join("..", "..", "docs", "cli", tool+".md"))
		if err != nil {
			t.Fatal(err)
		}
		for _, match := range regexp.MustCompile(`(?m)^## ([^\n]+)$`).FindAllStringSubmatch(string(body), -1) {
			if match[1] == "Overview" {
				continue
			}
			if _, ok := bindings[tool+" "+match[1]]; !ok {
				t.Errorf("new CLI command lacks catalog binding/exclusion: %s %s", tool, match[1])
			}
		}
	}
}

func TestCoreErrorsDoNotReflectUntrustedDiagnostics(t *testing.T) {
	for _, err := range []error{errors.New("cookie=private-value"), &client.RemoteError{Status: 403, Problem: protocol.APIError{Code: "policy_denied", Message: "cookie=private-value"}}} {
		encoded, _ := json.Marshal(problem(err, true))
		if strings.Contains(string(encoded), "private-value") {
			t.Fatal("diagnostic disclosed")
		}
	}
}

func TestSchemaValidationPreservesLargeIntegerBindings(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	type input struct {
		Binding uint64 `json:"binding"`
	}
	err = bind(s, description("integer-fixture.v1", "Exact binding fixture", nil, []string{"offline-read"}, nil, "local-user", "exact uint64", "read-only"), func(_ context.Context, _ *Service, in input) (input, error) { return in, nil })
	if err != nil {
		t.Fatal(err)
	}
	for _, number := range []string{"9007199254740993", "18446744073709551615"} {
		out := s.Call(context.Background(), "integer-fixture.v1", json.RawMessage(`{"binding":`+number+`}`))
		encoded, _ := json.Marshal(out)
		if out.Error != nil || !strings.Contains(string(encoded), `"binding":`+number) {
			t.Fatalf("binding rounded or refused: %s", encoded)
		}
	}
}

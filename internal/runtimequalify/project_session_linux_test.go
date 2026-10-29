//go:build linux && amd64

package runtimequalify

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

func TestProjectSessionOnlyAllocatesFreshMetadataAndReadsPinnedImage(t *testing.T) {
	f, _, _ := managedFixture(t)
	contract, _, _ := resourceContractFixture(t)
	f.engine.ID = contract.Preparation.EngineID
	original := f.d.execute
	var calls [][]string
	f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
		calls = append(calls, append([]string(nil), args[4:]...))
		if len(args) != 7 || args[4] != "image" || args[5] != "inspect" || args[6] != candidate.Image {
			t.Fatal("session setup mutated engine", args)
		}
		return original(ctx, args, env)
	}
	prepared, adapter, err := openProjectPreparation(context.Background(), contract.Selection, f.d)
	if err != nil || adapter == nil || prepared.EngineID != f.engine.ID || prepared.Sequence != 0 || prepared.Validate(contract.Selection.Plan) != nil {
		t.Fatal(prepared, err)
	}
	if len(calls) != 1 {
		t.Fatal(calls)
	}
	dir := filepath.Join(f.d.home, ".local/share/urbit-runner/managed/project-resources-"+prepared.ProjectID)
	if entries, err := os.ReadDir(dir); err != nil || len(entries) != 1 || entries[0].Name() != "docker-config" {
		t.Fatal(entries, err)
	}
	if _, _, err = openProjectPreparation(context.Background(), contract.Selection, f.d); err == nil || len(calls) != 1 {
		t.Fatal("existing workspace adopted", err, calls)
	}
}
func TestProjectSessionRefusesInvalidSelectionAndUnidentifiedEngine(t *testing.T) {
	for _, mode := range []string{"path", "generation", "missing-engine-id", "missing-image"} {
		t.Run(mode, func(t *testing.T) {
			f, _, _ := managedFixture(t)
			contract, _, _ := resourceContractFixture(t)
			f.engine.ID = contract.Preparation.EngineID
			selection := contract.Selection
			switch mode {
			case "path":
				selection.ProjectID = "../foreign"
			case "generation":
				selection.Generation = 2
			case "missing-engine-id":
				f.engine.ID = ""
			case "missing-image":
				f.imageID = "sha256:" + strings.Repeat("e", 64)
			}
			p, engine, err := openProjectPreparation(context.Background(), selection, f.d)
			if err == nil || engine != nil || p.ProjectID != "" {
				t.Fatal("invalid project session returned authority", p, err)
			}
			if len(f.containers) != 0 || len(f.networks) != 0 {
				t.Fatal("preflight created engine resources")
			}
			if mode != "missing-image" {
				dir := filepath.Join(f.d.home, ".local/share/urbit-runner/managed")
				if entries, err := os.ReadDir(dir); err != nil || len(entries) != 0 {
					t.Fatal("refusal created metadata", entries, err)
				}
			}
		})
	}
}

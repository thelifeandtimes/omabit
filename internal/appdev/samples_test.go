package appdev

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

func TestFrozenGallSampleIsCompleteBoundedClaySource(t *testing.T) {
	for _, sampleName := range []string{"gall-state-v1", "gall-state-v2"} {
		t.Run(sampleName, func(t *testing.T) {
			sample, err := ReadSample(sampleName)
			if err != nil {
				t.Fatal(err)
			}
			root := t.TempDir()
			for _, file := range sample.Files {
				path := filepath.Join(root, file.Path)
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(file.Content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			observed, err := deskwork.Snapshot(root)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(sample.Source, observed) || sample.Source.Validate() != nil {
				t.Fatal("sample cannot be staged exactly", sample.Source, observed)
			}
			index := sample.Source.Index()
			for _, required := range []string{"app/omabit-state.hoon", "lib/default-agent.hoon", "lib/skeleton.hoon", "mar/bill.hoon", "desk.bill"} {
				if _, ok := index[required]; !ok {
					t.Fatal("implicit scaffold dependency", required)
				}
			}
			if sample.Definition.Qualification != "current-workflow-pending" {
				t.Fatal("source test invented live qualification")
			}
			if sampleName == "gall-state-v1" && sample.DefinitionSHA256 != "a3183a6d19dcb83828d723048348fcad0e47e0b014bc30492c204d07bff10e04" {
				t.Fatal("frozen v1 definition changed")
			}
			if sampleName == "gall-state-v2" {
				if sample.Evidence == nil || sample.Evidence.DefinitionSHA256 != sample.DefinitionSHA256 || sample.Evidence.SourceDigest != sample.Source.Digest || !sample.Evidence.CleanShutdown || len(sample.Evidence.NotQualified) == 0 {
					t.Fatal("native evidence lost its exact source binding or limits")
				}
				for _, required := range []string{"mar/omabit-entry.hoon", "mar/json.hoon", "mar/mime.hoon"} {
					if _, ok := index[required]; !ok {
						t.Fatal("JSON wire dependency absent", required)
					}
				}
				if sample.Definition.PokeMark != "omabit-entry" {
					t.Fatal("JSON wire mark changed")
				}
			}
			if sampleName == "gall-state-v1" && sample.Evidence != nil {
				t.Fatal("v2 evidence was applied to the frozen v1 source")
			}
		})
	}
	for _, name := range []string{"grubbery", "../gall-state-v1", "gall-state-v1/source/app/omabit-state.hoon"} {
		if _, err := ReadSample(name); err == nil {
			t.Fatal("arbitrary sample selection accepted")
		}
	}
}

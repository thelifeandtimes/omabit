package appdev

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

func TestFrozenGallSampleIsCompleteBoundedClaySource(t *testing.T) {
	sample, err := ReadSample("gall-state-v1")
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
	for _, name := range []string{"grubbery", "../gall-state-v1", "gall-state-v1/source/app/omabit-state.hoon"} {
		if _, err := ReadSample(name); err == nil {
			t.Fatal("arbitrary sample selection accepted")
		}
	}
}

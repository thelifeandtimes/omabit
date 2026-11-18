package appdev_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/appdev"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

func TestMaintainedGrubberySourcesMatchQualifiedBytesAndStageSeparately(t *testing.T) {
	pins := map[string]string{"grubbery-state-v1": "840fb9fa309325af2ec44087342999dd95595ceacebd04335ba250103e5859b7", "grubbery-state-v2": "5be1882ece1a6c07f58dce29bace4df9bbf3bd2ab9feae164a3ddc3c874a3a3b"}
	for name, digest := range pins {
		sample, err := appdev.ReadSample(name)
		if err != nil {
			t.Fatal(err)
		}
		if sample.Definition.Version != 2 || sample.Definition.Framework != "grubbery" || sample.Source.Digest != digest || sample.Evidence == nil || sample.Evidence.SourceDigest != digest || !sample.Evidence.CleanShutdown || len(sample.Evidence.NotQualified) == 0 {
			t.Fatal("source qualification binding or limits changed", name)
		}
		if sample.Definition.Grubbery == nil || sample.Definition.Grubbery.FrameworkCommit != grubbery.FrameworkCommit || sample.Definition.Grubbery.Adapter != grubbery.HTTPContract {
			t.Fatal("sample and pinned adapter diverged")
		}
		directory := t.TempDir()
		files := []deskwork.SourceFile{}
		for _, file := range sample.Files {
			path := filepath.Join(directory, file.Path)
			if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(path, []byte(file.Content), 0600); err != nil {
				t.Fatal(err)
			}
			files = append(files, deskwork.SourceFile{Path: file.Path, Bytes: []byte(file.Content)})
		}
		manifest, err := deskwork.Snapshot(directory)
		if err != nil || !reflect.DeepEqual(manifest, sample.Source) {
			t.Fatal("sample cannot be staged exactly without metadata/notices", err)
		}
		source, err := grubbery.ReviewSource(manifest, files)
		if err != nil || len(source.Instances) != 1 || source.Instances[0].Name != "state" {
			t.Fatal("sample bill/nexus contract changed", err)
		}
	}
}

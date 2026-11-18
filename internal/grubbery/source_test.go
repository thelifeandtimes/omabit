package grubbery_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/appdev"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

func source(t *testing.T, bill string, extra ...deskwork.SourceFile) (deskwork.Manifest, []deskwork.SourceFile) {
	t.Helper()
	files := append([]deskwork.SourceFile{{Path: "bill.json", Bytes: []byte(bill)}, {Path: "nex/state.hoon", Bytes: []byte(":: explicit source; not a compilation test\n")}}, extra...)
	entries := []deskwork.File{}
	for _, f := range files {
		sum := sha256.Sum256(f.Bytes)
		entries = append(entries, deskwork.File{Path: f.Path, SHA256: hex.EncodeToString(sum[:]), Bytes: int64(len(f.Bytes))})
	}
	m, err := deskwork.NewManifest(entries)
	if err != nil {
		t.Fatal(err)
	}
	return m, files
}

func TestAppSourcePinsAndExactOwnedNexusBill(t *testing.T) {
	f, err := appdev.ReadFramework(appdev.GrubberyFramework)
	if err != nil || f.Commit != grubbery.FrameworkCommit || f.Source.Digest != grubbery.FrameworkSourceDigest || f.Agent != grubbery.FrameworkDesk || f.HTTPContract != grubbery.HTTPContract {
		t.Fatal("adapter/framework source pins diverged", err)
	}
	m, files := source(t, `{"state":"/state"}`, deskwork.SourceFile{Path: "assets/index.html", Bytes: []byte("<p>source-owned asset</p>")})
	r, err := grubbery.ReviewSource(m, files)
	if err != nil || len(r.Instances) != 1 || r.Instances[0].Neck != "/state" || r.Manifest.Digest != m.Digest {
		t.Fatal(r, err)
	}
	files[0].Bytes = []byte(`{"state":"/shell"}`)
	if _, err := grubbery.ReviewSource(m, files); err == nil {
		t.Fatal("changed staged source accepted")
	}
	for _, bill := range []string{`{}`, `[]`, `null`, `{"state":null}`, `{"state":"/shell"}`, `{"state":"/state","state":"/state"}`, `{"state":"/../state"}`, `{"state":"//state"}`, `{"state":"/state/"}`, `{"state":"/state?force=1"}`, `{"state":"/state"} {}`, `{"State":"/state"}`, `{"state":{"neck":"/state"}}`} {
		t.Run(bill, func(t *testing.T) {
			m, files := source(t, bill)
			if _, err := grubbery.ReviewSource(m, files); err == nil {
				t.Fatal("invalid, external or duplicate bill accepted")
			}
		})
	}
}

func TestAppSourceRefusesRepositoryMetadataAndReservedMarkers(t *testing.T) {
	for _, extra := range []deskwork.SourceFile{
		{Path: "version.json", Bytes: []byte("1")}, {Path: "sys.kelvin", Bytes: []byte("[%zuse 410]")},
		{Path: "app/state.hoon", Bytes: []byte("Clay agent, not app namespace")},
		{Path: "secrets/code.txt", Bytes: []byte("unrequested metadata")}, {Path: "nex/nested-bundle/state.hoon", Bytes: []byte(":: not compiled here")},
		{Path: "lib/raw.hoon", Bytes: []byte{0, 255}}, {Path: "assets/huge.bin", Bytes: make([]byte, grubbery.MaxAppFileBytes+1)},
	} {
		m, files := source(t, `{"state":"/state"}`, extra)
		if _, err := grubbery.ReviewSource(m, files); err == nil {
			t.Fatal("unexpected source accepted", extra.Path)
		}
	}
	m, files := source(t, `{"state":"/state"}`, deskwork.SourceFile{Path: "assets/vendor-bundle/main.js", Bytes: []byte{0, 255}})
	if _, err := grubbery.ReviewSource(m, files); err != nil {
		t.Fatal("binary source-owned assets rejected", err)
	}
	m, files = source(t, `{"state":"/state"}`)
	files = append(files, files[0])
	if _, err := grubbery.ReviewSource(m, files); err == nil {
		t.Fatal("duplicate files accepted")
	}
	if _, err := grubbery.ReviewSource(m, files[:1]); err == nil {
		t.Fatal("partial tree accepted")
	}
}

func TestAppSourceEnforcesAggregateFileAndInstanceBounds(t *testing.T) {
	for _, count := range []int{9, 65} {
		extra := []deskwork.SourceFile{}
		for i := 0; i < count; i++ {
			size := grubbery.MaxAppFileBytes
			if count == 65 {
				size = 1
			}
			extra = append(extra, deskwork.SourceFile{Path: fmt.Sprintf("assets/item-%02d.bin", i), Bytes: make([]byte, size)})
		}
		m, files := source(t, `{"state":"/state"}`, extra...)
		if _, err := grubbery.ReviewSource(m, files); err == nil {
			t.Fatal("oversized application stage accepted", count)
		}
	}
	bill := "{"
	for i := 0; i < grubbery.MaxInstances+1; i++ {
		if i != 0 {
			bill += ","
		}
		bill += fmt.Sprintf(`"state-%d":"/state"`, i)
	}
	m, files := source(t, bill+"}")
	if _, err := grubbery.ReviewSource(m, files); err == nil {
		t.Fatal("unbounded instance bill accepted")
	}
}

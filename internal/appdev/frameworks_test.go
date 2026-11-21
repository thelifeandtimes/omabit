package appdev

import (
	"os"
	"slices"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

func TestFrameworkRequiresExactCompletePinnedDesk(t *testing.T) {
	f, err := ReadFramework(GrubberyFramework)
	if err != nil {
		t.Fatal(err)
	}
	if f.VerifySource(f.Source) != nil || f.Qualification != "native-workflow-pending" || !strings.Contains(f.License, "Groundwire Foundation") || !strings.Contains(f.License, "Permission is hereby granted") {
		t.Fatal("lost source-only boundary or upstream license")
	}
	index := f.Source.Index()
	if index["app/grubbery.hoon"].SHA256 != "d6d6f5bf4fc0ddb55ecfbfab784a87de024c6c2703f59e016d648154d0bf8087" || index["lib/ball-api.hoon"].Bytes == 0 || index["gub/nex/desk.hoon"].Bytes == 0 || index["gub/nex/shell.hoon"].Bytes == 0 {
		t.Fatal("kernel/API/desk/permit implementation absent")
	}
	for _, change := range []string{"missing", "extra", "changed", "metadata-only"} {
		t.Run(change, func(t *testing.T) {
			files := slices.Clone(f.Source.Files)
			switch change {
			case "missing":
				files = files[1:]
			case "extra":
				files = append(files, deskwork.File{Path: "unreviewed.txt", SHA256: strings.Repeat("a", 64), Bytes: 1})
			case "changed":
				files[0].SHA256 = strings.Repeat("b", 64)
			case "metadata-only":
				files = []deskwork.File{index["sys.kelvin"], index["desk.bill"]}
			}
			observed, err := deskwork.NewManifest(files)
			if err != nil || f.VerifySource(observed) == nil {
				t.Fatal("nonmatching bytes accepted", err)
			}
			// A caller cannot make a different source authoritative by changing
			// the definition it passes to VerifySource.
			changed := f
			changed.Source = observed
			if changed.VerifySource(observed) == nil {
				t.Fatal("caller replaced compiled definition")
			}
		})
	}
	for _, name := range []string{"grubbery", "grubbery-v1", "../frameworks", "grubbery-ff2b66b-v1"} {
		if _, err := ReadFramework(name); err == nil {
			t.Fatal("unqualified revision accepted", name)
		}
	}
}

// Optional source-only gate for a separately downloaded public source tree. It
// performs the same bounded ordinary-file observation used for staging, never
// Hoon execution or a live connection. No checkout is fetched by the test.
func TestFrozenGrubberyDownloadedSource(t *testing.T) {
	directory := os.Getenv("OMABIT_TEST_GRUBBERY_SOURCE")
	if directory == "" {
		t.Skip("explicit public pinned desk source required")
	}
	f, err := ReadFramework(GrubberyFramework)
	if err != nil {
		t.Fatal(err)
	}
	observed, err := deskwork.Snapshot(directory)
	if err != nil || f.VerifySource(observed) != nil {
		t.Fatal("downloaded source does not match frozen complete desk", err)
	}
}

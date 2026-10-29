//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/bootpill"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectDependenciesFreshCacheAndReuse(t *testing.T) {
	f, _, _ := managedFixture(t)
	original := f.d.execute
	available := false
	pulls := 0
	f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
		if len(args) < 6 || args[4] != "image" {
			t.Fatal("dependency step operated on a runtime", args)
		}
		if args[5] == "inspect" && !available {
			return "", errors.New("not downloaded")
		}
		if args[5] == "pull" {
			pulls++
			available = true
		}
		return original(ctx, args, env)
	}
	for i := 0; i < 2; i++ {
		if err := prepareProjectDependencies(context.Background(), f.d); err != nil {
			t.Fatal(err)
		}
	}
	if f.downloads != 1 || pulls != 1 || len(f.containers) != 0 || len(f.networks) != 0 {
		t.Fatal(f.downloads, pulls)
	}
	base := filepath.Join(f.d.home, ".local/share/urbit-runner/managed")
	meta := protocol.BootPill{SHA256: f.d.pillSHA, Bytes: f.d.pillBytes}
	if err := bootpill.Verify(context.Background(), bootpill.CachePath(base, meta), meta); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(base)
	if len(entries) != 1 || entries[0].Name() != "boot-pills" {
		t.Fatal("temporary cache step leaked", entries)
	}
}
func TestProjectDependenciesPreserveInvalidCacheAndRefuseWrongImage(t *testing.T) {
	for _, scenario := range []string{"image", "pill", "cancelled"} {
		t.Run(scenario, func(t *testing.T) {
			f, _, _ := managedFixture(t)
			ctx := context.Background()
			if scenario == "image" {
				f.imageID = "sha256:" + strings.Repeat("a", 64)
			}
			var path string
			if scenario == "pill" {
				path = bootpill.CachePath(filepath.Join(f.d.home, ".local/share/urbit-runner/managed"), protocol.BootPill{SHA256: f.d.pillSHA, Bytes: f.d.pillBytes})
				if err := os.Mkdir(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("preserve invalid bytes"), 0400); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "cancelled" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			if err := prepareProjectDependencies(ctx, f.d); err == nil {
				t.Fatal("invalid dependency accepted")
			}
			if f.downloads != 0 {
				t.Fatal("invalid artifact replaced")
			}
			for _, event := range f.events {
				if event == "pull" {
					t.Fatal("mismatched image was pulled over")
				}
			}
			if path != "" {
				data, _ := os.ReadFile(path)
				if string(data) != "preserve invalid bytes" {
					t.Fatal("cache altered")
				}
			}
		})
	}
}

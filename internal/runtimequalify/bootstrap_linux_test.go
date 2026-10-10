//go:build linux && amd64

package runtimequalify

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/bootpill"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestFakeAndCometBootstrapSelectionOnlyAffectsFirstBoot(t *testing.T) {
	for _, comet := range []bool{false, true} {
		for _, custom := range []bool{false, true} {
			t.Run(fmt.Sprintf("comet-%t-custom-%t", comet, custom), func(t *testing.T) {
				fixture := managedFixture
				if comet {
					fixture = managedCometFixture
				}
				f, state, persist := fixture(t)
				state.Bootstrap = &protocol.Bootstrap{Kind: "bare"}
				if custom {
					data := []byte("recommended apps fixture")
					p := protocol.BootPill{SHA256: fmt.Sprintf("%x", sha256.Sum256(data)), Bytes: int64(len(data))}
					state.Bootstrap = &protocol.Bootstrap{Kind: "recommended", Pill: &p}
					base := filepath.Join(f.d.home, ".local/share/urbit-runner/managed")
					if err := os.Mkdir(filepath.Join(base, "boot-pills"), 0700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(bootpill.CachePath(base, p), data, 0400); err != nil {
						t.Fatal(err)
					}
				}
				var err error
				state, obs, err := applyManaged(context.Background(), "start", state, persist, f.d)
				if err != nil || obs.Health != "healthy" || f.downloads != 0 {
					t.Fatal(obs, err, f.downloads)
				}
				bootSeen := false
				for _, c := range state.Runtime.Containers {
					if c.Removed {
						continue
					}
					bootSeen = true
					actual := f.containers[c.Name]
					cmd := mapAt(actual.doc, "Config")["Cmd"].([]string)
					joined := " " + strings.Join(cmd, " ") + " "
					if strings.Contains(joined, " -B ") != custom {
						t.Fatal(cmd)
					}
					if strings.Contains(joined, " -L ") == comet {
						t.Fatal("Ames mode changed", cmd)
					}
				}
				if !bootSeen {
					t.Fatal("no live bootstrap")
				}
				state, _, err = applyManaged(context.Background(), "stop", state, persist, f.d)
				if err != nil {
					t.Fatal(err)
				}
				if custom {
					base := filepath.Join(f.d.home, ".local/share/urbit-runner/managed")
					if err = os.Remove(bootpill.CachePath(base, *state.Bootstrap.Pill)); err != nil {
						t.Fatal(err)
					}
					if err = os.Remove(state.Runtime.PillPath); err != nil {
						t.Fatal(err)
					}
				}
				state, obs, err = applyManaged(context.Background(), "start", state, persist, f.d)
				if err != nil || obs.Health != "healthy" || f.downloads != 0 {
					t.Fatal("resume required bootstrap material", obs, err)
				}
				for _, c := range state.Runtime.Containers {
					if c.Removed {
						continue
					}
					cmd := mapAt(f.containers[c.Name].doc, "Config")["Cmd"].([]string)
					for _, a := range cmd {
						if a == "-B" || a == "-c" || a == "-F" || a == "-k" {
							t.Fatal("resume repeated bootstrap", cmd)
						}
					}
				}
			})
		}
	}
}

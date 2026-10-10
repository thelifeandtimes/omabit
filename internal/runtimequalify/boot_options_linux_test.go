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
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestOwnedCanonicalAndCustomPillWithPublishedAmes(t *testing.T) {
	for _, custom := range []bool{false, true} {
		t.Run(fmt.Sprint(custom), func(t *testing.T) {
			f, state, persist := managedCometFixture(t)
			state.NativeShip = "~sampel-palnet"
			state.Comet.Ship = state.NativeShip
			state.AmesPort = 34567
			state.Boot = &ManagedBoot{}
			state.Storage = &pierstore.Location{Path: "livenet/planets/sampel-palnet/pier"}
			base := filepath.Join(f.d.home, ".local/share/urbit-runner/managed")
			os.Mkdir(filepath.Join(base, "keys"), 0700)
			key := "0w" + strings.Repeat("a", 128)
			state.NativeKeyDigest = fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
			os.WriteFile(filepath.Join(base, "keys", state.ID+".key"), []byte(key), 0600)
			if custom {
				data := []byte("explicitly selected test pill")
				p := protocol.BootPill{SHA256: fmt.Sprintf("%x", sha256.Sum256(data)), Bytes: int64(len(data))}
				state.Boot.Pill = &p
				os.Mkdir(filepath.Join(base, "boot-pills"), 0700)
				os.WriteFile(bootpill.CachePath(base, p), data, 0400)
			}
			f.d.hostIdentity = func(context.Context, string) (string, error) { return "~sampel-palnet", nil }
			state, obs, err := applyManaged(context.Background(), "start", state, persist, f.d)
			if err != nil || obs.Health != "healthy" || f.downloads != 0 {
				t.Fatal(obs, err, f.downloads)
			}
			bootSeen := false
			for _, call := range f.calls {
				joined := " " + strings.Join(call, " ") + " "
				if strings.Contains(joined, " -w ~sampel-palnet ") {
					bootSeen = true
					if !strings.Contains(joined, " --snap-time 60 ") {
						t.Fatal("owned boot omitted snapshot interval")
					}
					if strings.Contains(joined, " -B ") != custom {
						t.Fatal("canonical/custom pill flags", call)
					}
					if !strings.Contains(joined, " --ames-port 34567 ") || !strings.Contains(joined, "0.0.0.0:34567:34567/udp") {
						t.Fatal("Ames configuration lost", call)
					}
				}
			}
			if !bootSeen {
				t.Fatal("no key boot")
			}
			_, obs, err = observeManaged(context.Background(), state, f.d)
			if err != nil || obs.Health != "healthy" {
				t.Fatal(obs, err)
			}
			f.editInspect = func(_ string, doc map[string]any) {
				mapAt(mapAt(doc, "HostConfig"), "PortBindings")["34567/udp"] = []any{map[string]any{"HostIp": "127.0.0.1", "HostPort": "34568"}}
			}
			if _, _, err := observeManaged(context.Background(), state, f.d); err == nil {
				t.Fatal("wrong Ames publication accepted")
			}
		})
	}
}

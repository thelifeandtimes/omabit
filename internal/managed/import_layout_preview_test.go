package managed

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// Exercise the public guided-preview path with real private source/root trees
// and fixture identity detection. No runtime, Docker or host pier is accessed.
func TestGuidedImportPreviewIdentityLayoutMatrix(t *testing.T) {
	const comet = "doznec-salbud-salwex-sogref--tasweb-navfun-batwet-sigfet"
	for _, test := range []struct {
		name, ship, network, kind, source, destination          string
		customRoot, inPlace, registeredCollision, pathCollision bool
	}{
		{name: "arbitrary-source", ship: "~zod", network: "fakenet", kind: "fake", destination: "fakeships/zod/zod"},
		{name: "planet-repeats-identity", ship: "~sampel-palnet", network: "livenet", kind: "planet", destination: "livenet/planets/sampel-palnet/sampel-palnet"},
		{name: "known-comet", ship: "~" + comet, network: "livenet", kind: "comet", destination: "@disk/livenet/comets/" + comet + "/" + comet, customRoot: true},
		{name: "legacy-child", ship: "~zod", network: "fakenet", kind: "fake", source: "fakeships/project/friendly/pier", destination: "fakeships/project/friendly/zod"},
		{name: "custom-legacy-child", ship: "~zod", network: "fakenet", kind: "fake", source: "fakeships/project/friendly/pier", destination: "@disk/fakeships/project/friendly/zod", customRoot: true},
		{name: "custom-named-in-place", ship: "~zod", network: "fakenet", kind: "fake", source: "fakeships/project/friendly/zod", destination: "@disk/fakeships/project/friendly/zod", customRoot: true, inPlace: true},
		{name: "directory-collision", ship: "~zod", network: "fakenet", kind: "fake", destination: "fakeships/zod-2/zod", pathCollision: true},
		{name: "registered-name-collision", ship: "~zod", network: "fakenet", kind: "fake", destination: "fakeships/zod-2/zod", registeredCollision: true},
		{name: "source-label-does-not-set-network", ship: "~zod", network: "fakenet", kind: "fake", source: "livenet/galaxies/friendly/pier", destination: "fakeships/zod/zod"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			b := &guidedFixture{storageActionBackend: storageActionBackend{home: privateDir(t)}, identity: protocol.PierIdentity{Ship: test.ship, Network: test.network, Kind: test.kind}}
			if err := pierstore.Init(b.home); err != nil {
				t.Fatal(err)
			}
			roots, err := b.StorageRoots()
			if err != nil || len(roots) != 1 {
				t.Fatal(roots, err)
			}
			root, rootID := roots[0].Path, "default"
			if test.customRoot {
				root, rootID = privateDir(t), "disk"
				if _, err := b.RegisterStorageRoot(rootID, root); err != nil {
					t.Fatal(err)
				}
			}
			r, client, _ := testEndpoint(t, privateDir(t), b)
			wantRows := 0
			if test.registeredCollision {
				if _, _, err := r.s.register("fixture", registration("zod", "existing", 18080)); err != nil {
					t.Fatal(err)
				}
				wantRows = 1
			}
			if test.pathCollision {
				b.occupied = "fakeships/zod/zod"
			}
			source := filepath.Join(privateDir(t), "arbitrary-source-folder")
			if test.source != "" {
				source = filepath.Join(root, test.source)
			}
			for _, child := range []string{".urb/log", ".urb/chk"} {
				if err := os.MkdirAll(filepath.Join(source, child), 0700); err != nil {
					t.Fatal(err)
				}
			}
			marker := filepath.Join(source, ".urb/log/fixture")
			if err := os.WriteFile(marker, []byte("disposable source remains unchanged"), 0600); err != nil {
				t.Fatal(err)
			}
			before, err := b.InspectExternalPier(ctx, source)
			if err != nil {
				t.Fatal(err)
			}
			preview, err := client.PreviewGuidedPierImport(ctx, r.s.runnerID, source, rootID)
			if err != nil {
				t.Fatal(err)
			}
			if preview.Destination != test.destination || preview.InPlace != test.inPlace || preview.StorageRoot != rootID || preview.PierIdentity != b.identity || preview.InstanceID != filepath.Base(filepath.Dir(pierstore.Relative(test.destination))) {
				t.Fatalf("wrong import layout: %+v", preview)
			}
			if !pierstore.ValidPath(preview.Destination) || filepath.Base(preview.Destination) != test.ship[1:] {
				t.Fatal("destination lost detected identity", preview.Destination)
			}
			after, err := b.InspectExternalPier(ctx, source)
			if err != nil || before != after || preview.Source != before {
				t.Fatal("preview changed source", err)
			}
			rows, err := instances(r.s.db)
			if err != nil || len(rows) != wantRows {
				t.Fatal("preview registered an instance", rows, err)
			}
			if !test.inPlace {
				if _, err := os.Lstat(filepath.Join(root, pierstore.Relative(preview.Destination))); !os.IsNotExist(err) {
					t.Fatal("preview published destination", err)
				}
			}
		})
	}
}

//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"golang.org/x/sys/unix"
	_ "modernc.org/sqlite"
)

func TestRC11EmptyMountDeletionNeverAdoptsUnrecordedPier(t *testing.T) {
	for _, phase := range []string{"empty", "review", "renaming", "removing"} {
		t.Run(phase, func(t *testing.T) {
			f, state, persist := managedFixture(t)
			exhausted := false
			f.editImage = func(map[string]any) { exhausted = true }
			f.d.admission = func(string) (uint64, uint64, error) {
				if exhausted {
					return 0, 0, nil
				}
				return 5 << 30, 8 << 30, nil
			}
			var saved []byte
			_, _, err := applyManaged(context.Background(), "start", state, func(v ManagedState) error {
				saved = rc11Save(t, v)
				return persist(v)
			}, f.d)
			if err == nil || len(saved) == 0 {
				t.Fatal("preparation/admission boundary missed")
			}
			state = rc11Reload(t, saved)
			if !state.Prepared || state.Storage.MountIdentity.Inode == 0 || state.Storage.Identity.Inode != 0 || state.Initialized {
				t.Fatal("expected a real prepared empty mount")
			}
			f.editImage, exhausted = nil, false
			state, _, err = applyManaged(context.Background(), "stop", state, persist, f.d)
			rc11Must(t, err)
			path := managedPier(f, state)
			if phase == "renaming" || phase == "removing" {
				_, _, err = applyManaged(context.Background(), "delete-pier", state, func(v ManagedState) error {
					if v.Deletion != nil && v.Deletion.Stage == phase {
						saved = rc11Save(t, v)
						return errors.New("durable empty-mount deletion interruption")
					}
					return persist(v)
				}, f.d)
				if err == nil {
					t.Fatal("deletion boundary missed")
				}
				state = rc11Reload(t, saved)
				if phase == "removing" {
					path = filepath.Join(f.d.home, "piers/archive/deleted-"+state.ID, filepath.Base(path))
				}
			}
			file := filepath.Join(path, ".urb/foreign-data")
			if phase != "empty" {
				rc11Must(t, os.MkdirAll(filepath.Dir(file), 0700))
				rc11Must(t, os.WriteFile(file, []byte("unrecorded pier"), 0600))
			}
			_, obs, err := applyManaged(context.Background(), "delete-pier", state, persist, f.d)
			if phase == "empty" {
				if err != nil || obs.State != "deleted" {
					t.Fatal("legitimate empty-mount deletion stranded", obs, err)
				}
				return
			}
			if err == nil || !obs.RecoveryRequired {
				t.Fatal("mount-only ownership authorized its unknown child", obs, err)
			}
			if data, err := os.ReadFile(file); err != nil || string(data) != "unrecorded pier" {
				t.Fatal("unrecorded data deleted", err)
			}
		})
	}
}

func TestRC11CopiedSourceIsFrozenAcrossPublicationBarriers(t *testing.T) {
	for _, change := range []string{"source-bytes", "source-restored-mtime", "source-replaced-file", "source-new-child", "source-after-stage", "destination-bytes", "destination-directory"} {
		t.Run(change, func(t *testing.T) {
			f, state, persist, source := externalFixture(t)
			const file = "post-copy-data"
			rc11Must(t, os.WriteFile(filepath.Join(source, file), []byte("old-data"), 0600))
			info, err := os.Stat(filepath.Join(source, file))
			rc11Must(t, err)
			state.ExternalImport.Source, err = pierstore.InspectExternal(context.Background(), f.d.home, source)
			rc11Must(t, err)
			state.Adoption.Fingerprint = state.ExternalImport.Source.Fingerprint
			f.d.renamePier = func(string, string) error { return unix.EXDEV }
			held := filepath.Join(filepath.Dir(source), ".omabit-import-"+state.ID, "pier")
			target := managedPier(f, state)
			changed, unlinks := false, 0
			mutate := func() {
				if changed {
					return
				}
				changed = true
				switch change {
				case "source-bytes", "source-after-stage":
					rc11Must(t, os.WriteFile(filepath.Join(held, file), []byte("newer source data"), 0600))
				case "source-restored-mtime":
					rc11Must(t, os.WriteFile(filepath.Join(held, file), []byte("new-data"), 0600))
					rc11Must(t, os.Chtimes(filepath.Join(held, file), info.ModTime(), info.ModTime()))
				case "source-replaced-file":
					rc11Must(t, os.Rename(filepath.Join(held, file), filepath.Join(t.TempDir(), "original")))
					rc11Must(t, os.WriteFile(filepath.Join(held, file), []byte("old-data"), 0600))
				case "source-new-child":
					rc11Must(t, os.WriteFile(filepath.Join(held, "new-child"), []byte("newer source data"), 0600))
				case "destination-bytes":
					rc11Must(t, os.WriteFile(filepath.Join(target, file), []byte("changed destination"), 0600))
				case "destination-directory":
					rc11Must(t, os.Rename(target, target+"-preserved"))
					rc11Must(t, os.MkdirAll(filepath.Join(target, ".urb"), 0700))
				}
			}
			f.d.remove = func(*os.Root, string) error { unlinks++; return errors.New("unexpected source unlink") }
			_, obs, err := applyExternalImport(context.Background(), state, func(v ManagedState) error {
				if change == "source-after-stage" && v.ExternalImport.Stage == "removing-source" {
					mutate()
				}
				return persist(v)
			}, f.d, func(path string) error {
				if change != "source-after-stage" {
					mutate()
				}
				return pierstore.SyncDirectory(path)
			})
			if err == nil || !changed || unlinks != 0 || !obs.RecoveryRequired {
				t.Fatal("post-copy change did not fence all source deletion", obs, err, unlinks)
			}
			if _, err := os.Stat(filepath.Join(held, file)); err != nil {
				t.Fatal("held source lost", err)
			}
			if change == "source-bytes" || change == "source-after-stage" || change == "source-restored-mtime" {
				data, err := os.ReadFile(filepath.Join(held, file))
				if err != nil || bytes.Equal(data, []byte("old-data")) {
					t.Fatal("new source bytes not retained", err)
				}
			}
		})
	}
}

func TestRC11ImportRecoversEverySerializedEnrollmentCallback(t *testing.T) {
	for _, mode := range []string{"rename", "copy", "in-place"} {
		for _, boundary := range []string{"unprepared", "prepared", "complete"} {
			for _, live := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/live=%v", mode, boundary, live), func(t *testing.T) {
					f, state, persist, source := externalFixture(t)
					if live {
						state.NativeShip, state.Fake = "~sampel-palnet", nil
						state.Comet = &ManagedComet{Ship: state.NativeShip}
						state.Storage.Path = "livenet/planets/imported/sampel-palnet"
						state.Boot, state.AmesPort = &ManagedBoot{}, 34567
					}
					journal := filepath.Join(t.TempDir(), "callbacks.sqlite")
					db, err := sql.Open("sqlite", journal)
					rc11Must(t, err)
					t.Cleanup(func() { db.Close() })
					_, err = db.Exec("CREATE TABLE callback_state(body BLOB NOT NULL); INSERT INTO callback_state VALUES('{}')")
					rc11Must(t, err)
					persistCallback := func(v ManagedState) error {
						if _, err := db.Exec("UPDATE callback_state SET body=?", rc11Save(t, v)); err != nil {
							return err
						}
						return persist(v)
					}
					if mode == "copy" {
						f.d.renamePier = func(string, string) error { return unix.EXDEV }
					}
					if mode == "in-place" {
						target := managedPier(f, state)
						rc11Must(t, os.MkdirAll(filepath.Dir(target), 0700))
						rc11Must(t, os.Rename(source, target))
						var err error
						state.ExternalImport.Source, err = pierstore.InspectExternal(context.Background(), f.d.home, target)
						rc11Must(t, err)
						state.Adoption.Fingerprint = state.ExternalImport.Source.Fingerprint
					}
					var saved []byte
					_, _, err = applyManaged(context.Background(), "import-pier", state, func(v ManagedState) error {
						if err := persistCallback(v); err != nil {
							return err
						}
						if v.ExternalImport.Stage == "enrolling" && !v.Prepared && boundary == "unprepared" ||
							v.ExternalImport.Stage == "enrolling" && v.Prepared && boundary == "prepared" ||
							v.ExternalImport.Stage == "complete" && boundary == "complete" {
							saved = rc11Save(t, v)
							return errors.New("crash after durable enrollment callback")
						}
						return nil
					}, f.d)
					if err == nil || len(saved) == 0 {
						t.Fatal("enrollment callback not interrupted")
					}
					rc11Must(t, db.Close())
					db, err = sql.Open("sqlite", journal)
					rc11Must(t, err)
					var durable []byte
					rc11Must(t, db.QueryRow("SELECT body FROM callback_state").Scan(&durable))
					if !bytes.Equal(durable, saved) {
						t.Fatal("reopened journal differs from the actual enrollment callback")
					}
					state = rc11Reload(t, durable)
					before, _, _, err := pierstore.TreeReview(context.Background(), managedPier(f, state))
					rc11Must(t, err)
					location := *state.Storage
					if boundary != "complete" {
						if !state.ImportRecoveryEligible() {
							t.Fatal("serialized enrollment state is not recovery eligible")
						}
						failed, _, err := applyManaged(context.Background(), "recover-import", state, func(ManagedState) error { return errors.New("completion transaction failed") }, f.d)
						if err == nil || !bytes.Equal(saved, rc11Save(t, failed)) {
							t.Fatal("failed completion changed the original enrollment state", err)
						}
						state, _, err = applyManaged(context.Background(), "recover-import", rc11Reload(t, saved), persistCallback, f.d)
						rc11Must(t, err)
					}
					if !state.Prepared || !state.Initialized || state.ExternalImport.Stage != "complete" || state.UncertainProvisioning() || *state.Storage != location {
						t.Fatal("enrollment callback has no completed path")
					}
					after, _, _, err := pierstore.TreeReview(context.Background(), managedPier(f, state))
					if err != nil || before != after || len(f.calls) != 0 || f.downloads != 0 {
						t.Fatal("callback recovery changed pier data or provisioned a runtime", err)
					}
				})
			}
		}
	}
}

func TestRC11UnpreparedEnrollmentRecoveryRetainsEvidenceGuards(t *testing.T) {
	for _, fault := range []string{"lock", "replacement", "source-reappeared", "runtime-header", "runtime-result", "fingerprint", "epoch-version"} {
		t.Run(fault, func(t *testing.T) {
			f, state, persist, _ := externalFixture(t)
			var saved []byte
			_, _, err := applyManaged(context.Background(), "import-pier", state, func(v ManagedState) error {
				if v.ExternalImport.Stage == "enrolling" && !v.Prepared {
					saved = rc11Save(t, v)
					return errors.New("enrolling callback interruption")
				}
				return persist(v)
			}, f.d)
			if err == nil || len(saved) == 0 {
				t.Fatal("missing unprepared enrollment fixture")
			}
			state = rc11Reload(t, saved)
			path := managedPier(f, state)
			switch fault {
			case "lock":
				rc11Must(t, os.WriteFile(filepath.Join(path, ".urb/lock"), []byte("native"), 0600))
			case "replacement":
				rc11Must(t, os.Rename(filepath.Join(path, ".urb"), filepath.Join(t.TempDir(), "original-urb")))
				rc11Must(t, os.Mkdir(filepath.Join(path, ".urb"), 0700))
			case "source-reappeared":
				rc11Must(t, os.Mkdir(state.ExternalImport.Source.Path, 0700))
			case "runtime-header":
				state.Runtime.RunID = state.ID
			case "runtime-result":
				state.Runtime.PierMetadataSeen = true
			case "fingerprint":
				state.Adoption.Fingerprint = "wrong"
			case "epoch-version":
				rc11Must(t, os.WriteFile(filepath.Join(path, ".urb/log/0i0/vere.txt"), []byte("foreign-version"), 0600))
			}
			before, _, _, err := pierstore.TreeReview(context.Background(), path)
			rc11Must(t, err)
			writes := 0
			_, obs, err := applyManaged(context.Background(), "recover-import", state, func(ManagedState) error { writes++; return nil }, f.d)
			if err == nil || !obs.RecoveryRequired || writes != 0 {
				t.Fatal("unprepared recovery bypassed evidence", obs, err, writes)
			}
			workspace := filepath.Join(f.d.home, ".local/share/urbit-runner/managed", state.ID)
			if _, err := os.Stat(workspace); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("unvalidated recovery created a workspace", err)
			}
			after, _, _, err := pierstore.TreeReview(context.Background(), path)
			if err != nil || before != after || len(f.calls) != 0 {
				t.Fatal("unsafe recovery changed data or dispatched runtime", err)
			}
		})
	}
}

//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"golang.org/x/sys/unix"
)

func rc11PreparedImport(t *testing.T, mode string, live bool) (*fakeEngine, ManagedState, func(ManagedState) error) {
	t.Helper()
	f, state, persist, source := externalFixture(t)
	state.Storage.Path = "fakeships/imported/zod"
	if live {
		state.NativeShip, state.Fake = "~sampel-palnet", nil
		state.Comet = &ManagedComet{Ship: state.NativeShip}
		state.Storage.Path = "livenet/planets/sampel-palnet/sampel-palnet"
		state.Boot, state.AmesPort = &ManagedBoot{}, 34567
	}
	if mode == "copy" {
		_, err := pierstore.RegisterRoot(f.d.home, "fixture", t.TempDir())
		rc11Must(t, err)
		state.Storage.Path = "@fixture/" + state.Storage.Path
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
	_, _, err := applyManaged(context.Background(), "import-pier", state, func(v ManagedState) error {
		saved = rc11Save(t, v)
		if v.Prepared {
			return errors.New("crash after prepared callback, before workspace mkdir")
		}
		return persist(v)
	}, f.d)
	if err == nil {
		t.Fatal("prepared callback not interrupted")
	}
	state = rc11Reload(t, saved)
	if !state.ImportRecoveryEligible() || !state.Prepared || state.Initialized {
		t.Fatal("prepared callback lost its recovery intent")
	}
	if _, err := os.Lstat(state.Runtime.Directory); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("fixture did not stop before workspace creation", err)
	}
	return f, state, persist
}

func TestRC11PreparedImportRecoveryCreatesOnlyMissingWorkspace(t *testing.T) {
	for _, mode := range []string{"rename", "copy", "in-place"} {
		for _, live := range []bool{false, true} {
			for _, workspace := range []string{"absent", "directory-only", "complete"} {
				t.Run(fmt.Sprintf("%s/live=%v/%s", mode, live, workspace), func(t *testing.T) {
					f, state, persist := rc11PreparedImport(t, mode, live)
					path, err := pierstore.Absolute(f.d.home, state.Storage.Path)
					rc11Must(t, err)
					config := filepath.Join(state.Runtime.Directory, "docker-config")
					if workspace != "absent" {
						rc11Must(t, os.Mkdir(state.Runtime.Directory, 0700))
						rc11Must(t, os.WriteFile(filepath.Join(state.Runtime.Directory, "retained"), []byte("keep"), 0600))
					}
					if workspace == "complete" {
						rc11Must(t, os.Mkdir(config, 0700))
					}
					before, _, _, err := pierstore.TreeReview(context.Background(), path)
					rc11Must(t, err)
					writes := 0
					next, obs, err := applyManaged(context.Background(), "recover-import", state, func(v ManagedState) error {
						writes++
						if !v.Initialized || v.ExternalImport.Stage != "complete" {
							t.Fatal("intermediate state escaped completion proposal")
						}
						return persist(v)
					}, f.d)
					if err != nil || writes != 1 || obs.State != "stopped" || obs.RecoveryRequired || next.UncertainProvisioning() || next.ImportRecoveryEligible() {
						t.Fatal("prepared boundary remains stranded", obs, err, writes)
					}
					for _, p := range []string{state.Runtime.Directory, config} {
						i, err := os.Lstat(p)
						if err != nil || !i.IsDir() || i.Mode().Perm() != 0700 {
							t.Fatal("workspace was not privately restored", err)
						}
					}
					if workspace != "absent" {
						data, err := os.ReadFile(filepath.Join(state.Runtime.Directory, "retained"))
						if err != nil || string(data) != "keep" {
							t.Fatal("existing workspace content changed", err)
						}
					}
					after, _, _, err := pierstore.TreeReview(context.Background(), path)
					if err != nil || before != after || len(f.calls) != 0 || next.Storage.Identity != state.Storage.Identity || next.Port != state.Port || next.AmesPort != state.AmesPort {
						t.Fatal("recovery changed pier, ports, identity or dispatched runtime", err)
					}
				})
			}
		}
	}
}

func TestRC11MissingRecoveryWorkspaceStillRequiresAllEvidence(t *testing.T) {
	for _, fault := range []string{"workspace-link", "config-link", "writable", "config-file", "late-workspace", "workspace-replaced", "lock", "urb-lock", "urb-vere-lock", "source-reappeared", "pier-replaced", "journal"} {
		t.Run(fault, func(t *testing.T) {
			f, state, persist := rc11PreparedImport(t, "rename", false)
			path, dir := managedPier(f, state), state.Runtime.Directory
			config := filepath.Join(dir, "docker-config")
			retained := t.TempDir()
			switch fault {
			case "workspace-link":
				rc11Must(t, os.Symlink(retained, dir))
			case "config-link", "config-file", "workspace-replaced", "writable":
				rc11Must(t, os.Mkdir(dir, 0700))
				if fault == "config-link" {
					rc11Must(t, os.Symlink(retained, config))
				} else if fault == "config-file" {
					rc11Must(t, os.WriteFile(config, []byte("keep"), 0600))
				} else if fault == "writable" {
					rc11Must(t, os.Chmod(dir, 0777))
				} else {
					rc11Must(t, os.Mkdir(config, 0700))
				}
			case "lock", "urb-lock", "urb-vere-lock":
				name := map[string]string{"lock": ".vere.lock", "urb-lock": ".urb/lock", "urb-vere-lock": ".urb/vere.lock"}[fault]
				rc11Must(t, os.WriteFile(filepath.Join(path, name), []byte("native"), 0600))
			case "source-reappeared":
				rc11Must(t, os.Mkdir(state.ExternalImport.Source.Path, 0700))
			case "pier-replaced":
				rc11Must(t, os.Rename(path, filepath.Join(retained, "original")))
				rc11Must(t, os.MkdirAll(filepath.Join(path, ".urb"), 0700))
			}
			if fault == "late-workspace" || fault == "workspace-replaced" {
				original, changed := f.d.stat, false
				f.d.stat = func(p string) (node, error) {
					if !changed && p == filepath.Join(path, ".urb/log/0i0/vere.txt") {
						changed = true
						if fault == "workspace-replaced" {
							rc11Must(t, os.Rename(dir, filepath.Join(retained, "original-workspace")))
						}
						rc11Must(t, os.Mkdir(dir, 0700))
						rc11Must(t, os.WriteFile(filepath.Join(dir, "unexpected"), []byte("keep"), 0600))
					}
					return original(p)
				}
			}
			before, _, _, err := pierstore.TreeReview(context.Background(), path)
			rc11Must(t, err)
			writes := 0
			next, obs, err := applyManaged(context.Background(), "recover-import", state, func(ManagedState) error {
				writes++
				return errors.New("publication failed")
			}, f.d)
			if err == nil || !obs.RecoveryRequired || !next.ImportRecoveryEligible() || (fault == "journal") != (writes == 1) {
				t.Fatal("invalid evidence or failed publication cleared recovery", obs, err, writes)
			}
			if fault == "journal" {
				next, obs, err = applyManaged(context.Background(), "recover-import", next, persist, f.d)
				if err != nil || next.UncertainProvisioning() || obs.State != "stopped" {
					t.Fatal("created workspace stranded publication retry", obs, err)
				}
			}
			if fault == "lock" || fault == "urb-lock" || fault == "urb-vere-lock" || fault == "source-reappeared" || fault == "pier-replaced" {
				if _, err := os.Lstat(dir); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("workspace created before evidence validation", err)
				}
			}
			after, _, _, err := pierstore.TreeReview(context.Background(), path)
			if err != nil || before != after || len(f.calls) != 0 {
				t.Fatal("recovery changed pier data or dispatched runtime", err)
			}
		})
	}
}

func TestRC11LegacyCompletionEligibilityDoesNotMutateEvidence(t *testing.T) {
	f, state, persist := retainedImport(t, true)
	state, _, err := applyManaged(context.Background(), "recover-import", state, persist, f.d)
	rc11Must(t, err)
	state.ExternalImport.Stage = "enrolling"
	seed := rc11Save(t, state)
	for _, fault := range []string{"", "uncertain", "verified", "missing-finish", "zero-finish", "missing-start", "zero-start", "comet-finish", "comet-zero-finish", "reversed-times", "adoption-before-comet", "comet-uncertain", "identity", "phase", "storage", "fingerprint", "container", "network", "maintenance", "bootstrap"} {
		t.Run(fault, func(t *testing.T) {
			state := rc11Reload(t, seed)
			zero := time.Time{}
			switch fault {
			case "uncertain":
				state.Adoption.Uncertain = true
			case "verified":
				state.Adoption.Verified = true
			case "missing-finish":
				state.Adoption.FinishedAt = nil
			case "zero-finish":
				state.Adoption.FinishedAt = &zero
			case "missing-start":
				state.Comet.StartedAt = nil
			case "zero-start":
				state.Comet.StartedAt = &zero
			case "comet-finish":
				state.Comet.FinishedAt = nil
			case "comet-zero-finish":
				state.Comet.FinishedAt = &zero
			case "reversed-times":
				later := state.Comet.FinishedAt.Add(time.Second)
				state.Comet.StartedAt = &later
			case "adoption-before-comet":
				earlier := state.Comet.FinishedAt.Add(-time.Second)
				state.Adoption.FinishedAt = &earlier
			case "comet-uncertain":
				state.Comet.Uncertain = true
			case "identity":
				state.Comet.Ship = "~different"
			case "phase":
				state.ExternalImport.Stage = "complete"
			case "storage":
				state.Storage.Uncertain = true
			case "fingerprint":
				state.Adoption.Fingerprint = "wrong"
			case "container":
				state.Runtime.Containers = []ContainerRecord{{Removed: true}}
			case "network":
				state.Runtime.Network = &NetworkRecord{Removed: true}
			case "maintenance":
				state.MaintenanceUncertain = true
			case "bootstrap":
				state.NativeKeyDigest = "pending"
			}
			before := rc11Save(t, state)
			for i := 0; i < 3; i++ {
				if got := state.ImportRecoveryEligible(); got != (fault == "") {
					t.Fatal("wrong historical completion eligibility", got)
				}
			}
			if !bytes.Equal(before, rc11Save(t, state)) {
				t.Fatal("eligibility query mutated retained evidence")
			}
		})
	}
}

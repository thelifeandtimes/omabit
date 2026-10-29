//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

func memberStorageDriverFixture(t *testing.T) (*fakeEngine, projectnetwork.Plan, projectnetwork.MemberStorage, projectnetwork.MemberStorageDriver) {
	t.Helper()
	f := fixture(t)
	c, _, _ := resourceContractFixture(t)
	s, err := projectnetwork.NewMemberStorage(c.Selection.Plan, c.Selection.ProjectID, 1, "sender", "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", "prepare-storage")
	if err != nil {
		t.Fatal(err)
	}
	d, err := openProjectMemberStorage(context.Background(), c.Selection.Plan, s, f.d)
	if err != nil {
		t.Fatal(err)
	}
	return f, c.Selection.Plan, s, d
}
func TestProjectMemberStorageCreatesOnlyAnExclusiveEmptyParent(t *testing.T) {
	f, plan, s, d := memberStorageDriverFixture(t)
	saved := s
	var phases []string
	result, err := projectnetwork.ExecuteMemberStorage(context.Background(), plan, s, d, func(expected, phase string, id pierstore.DirectoryIdentity) error {
		if expected != saved.Phase {
			t.Fatal("stale callback")
		}
		next, err := saved.Advance(plan, expected, phase, id)
		if err != nil {
			return err
		}
		saved = next
		phases = append(phases, phase)
		return nil
	})
	if err != nil || result.Phase != "observed" || !reflect.DeepEqual(saved, result) || !reflect.DeepEqual(phases, []string{"dispatched", "created", "observed"}) {
		t.Fatal(result, phases, err)
	}
	path, err := pierstore.CheckMount(f.d.home, result.Location)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(path)
	if err != nil || len(entries) != 0 {
		t.Fatal(entries, err)
	}
	if len(f.calls) != 0 || len(f.containers) != 0 || len(f.networks) != 0 || f.downloads != 0 {
		t.Fatal("filesystem preparation accessed Docker or downloads")
	}
	if _, err = os.Stat(filepath.Join(path, "zod")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("storage preparation created a pier")
	}
	if _, err = projectnetwork.ExecuteMemberStorage(context.Background(), plan, result, d, func(string, string, pierstore.DirectoryIdentity) error { t.Fatal("replayed callback"); return nil }); err == nil {
		t.Fatal("observed preparation replayed")
	}
}
func TestProjectMemberStorageNeverAdoptsExistingParent(t *testing.T) {
	f, plan, s, d := memberStorageDriverFixture(t)
	parent := filepath.Join(f.d.home, "piers/fakeships/sender")
	if err := os.MkdirAll(parent, 0700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(parent, "keep")
	if err := os.WriteFile(marker, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := projectnetwork.ExecuteMemberStorage(context.Background(), plan, s, d, func(string, string, pierstore.DirectoryIdentity) error { return nil })
	if err == nil || result.Phase != "dispatched" || result.Location.MountIdentity != (pierstore.DirectoryIdentity{}) {
		t.Fatal(result, err)
	}
	data, err := os.ReadFile(marker)
	if err != nil || string(data) != "original" {
		t.Fatal("existing storage changed", err)
	}
}
func TestProjectMemberStorageFailedCommitAndCancellationKeepEvidence(t *testing.T) {
	for _, cut := range []string{"dispatched", "created", "observed", "cancel-dispatched", "replace-created"} {
		t.Run(cut, func(t *testing.T) {
			f, plan, s, d := memberStorageDriverFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			parent := filepath.Join(f.d.home, "piers/fakeships/sender")
			saved := s
			result, err := projectnetwork.ExecuteMemberStorage(ctx, plan, s, d, func(expected, phase string, id pierstore.DirectoryIdentity) error {
				if phase == cut {
					return errors.New("durable callback failed")
				}
				var err error
				saved, err = saved.Advance(plan, expected, phase, id)
				if err != nil {
					return err
				}
				if cut == "cancel-dispatched" && phase == "dispatched" {
					cancel()
				}
				if cut == "replace-created" && phase == "created" {
					if err = os.Rename(parent, parent+"-original"); err != nil {
						return err
					}
					return os.Mkdir(parent, 0700)
				}
				return nil
			})
			if err == nil || !reflect.DeepEqual(saved, result) {
				t.Fatal("failed work completed or rewrote durable state", result, saved, err)
			}
			_, statErr := os.Stat(parent)
			if cut == "dispatched" || cut == "cancel-dispatched" {
				if !errors.Is(statErr, os.ErrNotExist) {
					t.Fatal("mkdir preceded durable intent/cancellation check", statErr)
				}
			} else if statErr != nil {
				t.Fatal("created evidence removed", statErr)
			}
			if result.Phase != "planned" {
				if _, err = projectnetwork.ExecuteMemberStorage(context.Background(), plan, result, d, func(string, string, pierstore.DirectoryIdentity) error { return nil }); err == nil {
					t.Fatal("interrupted execution replayed")
				}
			}
		})
	}
}

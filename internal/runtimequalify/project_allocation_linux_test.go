//go:build linux && amd64

package runtimequalify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectusage"
)

func allocationAdapterFixture(t *testing.T) (*memberStopFixture, ProjectExpansion, projectusage.Binding) {
	t.Helper()
	f, input, _ := resumedStopAdapterFixture(t, false, 2)
	state, err := SeedProjectExpansion(ProjectCurrentReadinessSelection{Project: input.Project, Launches: input.Launches, Resumes: input.Resumes, Original: f.readiness["receiver"]})
	if err != nil {
		t.Fatal(err)
	}
	for i := range state.Members {
		m := &state.Members[i]
		m.Initialized = m.Creator.Storage
		m.Initialized.Identity, err = pierstore.Identify(f.paths[m.Member.InstanceID])
		if err != nil {
			t.Fatal(err)
		}
	}
	digest, err := state.Project.Plan.Digest()
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(body)
	binding := projectusage.Binding{RunnerID: state.Project.RunnerID, Mode: "development-managed", ProjectID: state.Project.ProjectID, Revision: state.Project.Revision, Generation: state.Project.Generation, PlanDigest: digest, LedgerSHA256: hex.EncodeToString(h[:])}
	return f, state, binding
}

// All engine adapters are inert, with real disposable filesystem objects. This
// test grants no native runtime or installed-upgrade qualification.
func TestExpandedProjectAllocationReadsOnlyBoundCurrentStorage(t *testing.T) {
	f, state, binding := allocationAdapterFixture(t)
	forbid := func(context.Context, []string, []string) (string, error) {
		t.Error("allocation invoked an engine command")
		return "", errors.New("forbidden")
	}
	f.f.d.execute = forbid
	f.f.d.hostIdentity = func(context.Context, string) (string, error) {
		t.Error("allocation probed HTTP")
		return "", errors.New("forbidden")
	}
	before := map[string]string{}
	err := filepath.WalkDir(f.f.d.home, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			data, e := os.ReadFile(path)
			if e != nil {
				return e
			}
			before[path] = string(data)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	signals, calls := f.signals, len(f.f.calls)
	out, err := readExpandedProjectAllocation(t.Context(), state, binding, f.f.d)
	if err != nil || out.Validate() != nil || out.Status != "measured" || out.Binding != binding || len(out.Roots) != len(state.Members)*2+1 || out.Growth != nil || out.FullProjectBudgetKnown || out.Totals.RegularFiles == 0 {
		t.Fatal("bound allocation failed", err, out)
	}
	if f.signals != signals || len(f.f.calls) != calls {
		t.Fatal("read changed runtime history")
	}
	for path, data := range before {
		got, err := os.ReadFile(path)
		if err != nil || string(got) != data {
			t.Fatal("read changed original bytes", path, err)
		}
	}
	roots, err := projectAllocationRoots(t.Context(), state, binding, f.f.d)
	if err != nil || !reflect.DeepEqual(roots, out.DeclaredRoots) {
		t.Fatal("root selection drift", err)
	}
	for _, r := range out.Roots {
		if r.Root.Kind == "member-mount" && !r.Root.Expected.SameObject(r.Observed) {
			t.Fatal("lost mount identity")
		}
	}
}

func TestExpandedProjectAllocationRefusesChangedBindingsAndUnsafeCaches(t *testing.T) {
	for _, variant := range []string{"generation", "plan", "creator", "initialized", "unsafe-cache", "missing-cache"} {
		t.Run(variant, func(t *testing.T) {
			f, state, binding := allocationAdapterFixture(t)
			switch variant {
			case "generation":
				binding.Generation++
			case "plan":
				binding.PlanDigest = strings.Repeat("a", 64)
			case "creator":
				state.Members[0].Creator.Container.ContractDigest = strings.Repeat("b", 64)
			case "initialized":
				state.Members[0].Initialized.Identity.Inode++
			case "unsafe-cache":
				base := filepath.Join(f.f.d.home, ".local/share/urbit-runner/managed", state.Members[0].Creator.RuntimeID)
				if err := os.Symlink("/outside-fixture", filepath.Join(base, "unsafe")); err != nil {
					t.Fatal(err)
				}
			case "missing-cache":
				base := filepath.Join(f.f.d.home, ".local/share/urbit-runner/managed", state.Members[0].Creator.RuntimeID)
				if err := os.Rename(base, base+"-retained"); err != nil {
					t.Fatal(err)
				}
			}
			out, err := readExpandedProjectAllocation(t.Context(), state, binding, f.f.d)
			if variant == "unsafe-cache" || variant == "missing-cache" {
				if err != nil || out.Validate() != nil || out.Status != "unknown" || out.Totals != nil || len(out.Roots) != 0 {
					t.Fatal("unsafe cache yielded partial totals", err, out)
				}
			} else if err == nil {
				t.Fatal("changed binding accepted", out)
			}
		})
	}
}

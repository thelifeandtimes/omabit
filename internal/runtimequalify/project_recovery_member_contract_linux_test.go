//go:build linux && amd64

package runtimequalify

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

func completedRecoveryRetirement(t *testing.T) *anchorRetirementAdapterFixture {
	t.Helper()
	f := newAnchorRetirementAdapterFixture(t)
	h := anchorDrainHistory(f.selection.Source)
	for !f.saved.Complete() {
		s := f.saved.Clone()
		e := f.open(t, s)
		if _, err := projectnetwork.ExecuteNextAnchorRetirement(context.Background(), h, s, e, func(seq uint64, next projectnetwork.AnchorRetirement) error { return f.commit(t, seq, next) }, time.Now); err != nil {
			t.Fatal(err)
		}
	}
	return f
}
func TestRecoveryMemberContractKeepsOriginalAndBindsSuccessor(t *testing.T) {
	f := completedRecoveryRetirement(t)
	h := anchorDrainHistory(f.selection.Source)
	original := f.saved.Clone()
	history := h.Clone()
	for _, m := range original.Namespace.Drain.Review.Context.Members {
		member := m.Context.Member
		c, err := newRecoveryProjectMemberContract(f.selection.Source.Project, h, original, member.InstanceID, "recover-"+member.InstanceID, f.x.f.f.d.home, filepath.Dir(f.x.f.paths[member.InstanceID]), candidate.ManifestDigest)
		if err != nil {
			t.Fatal(err)
		}
		if c.recovery == nil || c.recovery.Validate(h, original) != nil || c.selection.ContainerGeneration != member.Generation+1 || c.selection.Mode != "resume" || c.pillPath != "" || c.selection.Storage != m.Context.Storage || c.selection.Anchor.Namespace == nil || *c.selection.Anchor.Namespace == c.recovery.Namespace {
			t.Fatal("replacement changed original authority or bootstrap mode")
		}
		legacy, err := newProjectMemberContract(c.selection, c.home, c.mountSource, c.imageID)
		if err != nil || legacy.digest == c.digest {
			t.Fatal("successor contract not distinct", err)
		}
		// Independently reproduce the original v1 serialization. New private contract
		// fields cannot silently change any ordinary historical digest or labels.
		body, err := json.Marshal(struct {
			Domain     string
			Selection  projectMemberSelection
			Arguments  []string
			PillSHA256 string
			PillBytes  int64
		}{"omabit-project-member-v1", legacy.selection, legacy.createArgs(), candidate.PillSHA256, candidate.PillBytes})
		if err != nil || legacy.digest != fmt.Sprintf("%x", sha256.Sum256(body)) {
			t.Fatal("legacy contract digest changed")
		}
		if _, ok := legacy.labels()[projectLabel+".namespace-operation"]; ok {
			t.Fatal("legacy labels changed")
		}
		v := projectMemberContainerFixture(t, c)
		v.ID = strings.Repeat("1", 64)
		if c.validateContainer(v, v.ID, false) != nil || legacy.validateContainer(v, v.ID, false) == nil {
			t.Fatal("successor labels not enforced")
		}
		if v.Config.Labels[projectLabel+".namespace-operation"] != original.Namespace.OperationID || v.Config.Labels[projectLabel+".retirement-operation"] != original.OperationID {
			t.Fatal("missing succession links")
		}
		delete(v.Config.Labels, projectLabel+".namespace-operation")
		if c.validateContainer(v, v.ID, false) == nil {
			t.Fatal("missing succession label accepted")
		}
		if strings.Contains(strings.Join(c.createArgs(), " "), "boot.pill") || !reflect.DeepEqual(c.selection.Anchor, c.recovery.OriginalAnchor) {
			t.Fatal("replacement bootstraps or rewrites anchor")
		}
		c.recovery.OriginalAnchor.Namespace.Inode++
		if !reflect.DeepEqual(h, history) || !reflect.DeepEqual(original, f.saved) {
			t.Fatal("replacement aliases retained history")
		}
	}
}
func TestRecoveryMemberContractRefusesIncompleteAuthority(t *testing.T) {
	f := completedRecoveryRetirement(t)
	h := anchorDrainHistory(f.selection.Source)
	instance := f.saved.Members[0].InstanceID
	for _, mode := range []string{"partial", "missing-pier", "old-namespace", "foreign-instance", "reused-operation", "changed-plan", "changed-runner"} {
		t.Run(mode, func(t *testing.T) {
			selected := f.selection.Source.Project
			selected.Plan.Members = append([]projectnetwork.Member{}, selected.Plan.Members...)
			r := f.saved.Clone()
			id := instance
			op := "new-replacement"
			switch mode {
			case "partial":
				r.Members[len(r.Members)-1].Phase = "remove-dispatched"
			case "missing-pier":
				r.Members[len(r.Members)-1].After.Piers = nil
			case "old-namespace":
				r.Namespace.After.Runtime.Anchor.Namespace = r.Namespace.Drain.Review.Context.Binding.Namespace
			case "foreign-instance":
				id = "foreign"
			case "reused-operation":
				op = r.Namespace.OperationID
			case "changed-plan":
				selected.Plan.Members[0].HTTPPort++
			case "changed-runner":
				selected.RunnerID = "foreign-runner"
			}
			if _, err := newRecoveryProjectMemberContract(selected, h, r, id, op, f.x.f.f.d.home, filepath.Dir(f.x.f.paths[instance]), candidate.ManifestDigest); err == nil {
				t.Fatal("invalid recovery authority accepted")
			}
		})
	}
}

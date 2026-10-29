//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

type anchorReplacementAdapterFixture struct {
	f               *anchorRetirementAdapterFixture
	selection       ProjectAnchorReplacementSelection
	saved           projectnetwork.AnchorReplacement
	contracts       []projectMemberContract
	creates, starts int
	mode            string
	cancel          context.CancelFunc
}

func newAnchorReplacementAdapterFixture(t *testing.T) *anchorReplacementAdapterFixture {
	t.Helper()
	f := completedRecoveryRetirement(t)
	h := anchorDrainHistory(f.selection.Source)
	x := &anchorReplacementAdapterFixture{f: f}
	var members []projectnetwork.RecoveredMember
	for _, m := range f.saved.Members {
		c, err := newRecoveryProjectMemberContract(f.selection.Source.Project, h, f.saved, m.InstanceID, "replace-"+m.InstanceID, f.x.f.f.d.home, filepath.Dir(f.x.f.paths[m.InstanceID]), candidate.ManifestDigest)
		if err != nil {
			t.Fatal(err)
		}
		x.contracts = append(x.contracts, c)
		members = append(members, projectnetwork.RecoveredMember{Creation: c.recovery.Clone(), Container: projectnetwork.ResourceIdentity{ContractDigest: c.digest}, Phase: "planned"})
	}
	s, err := projectnetwork.NewAnchorReplacement(h, f.saved, "replace-all-members", members)
	if err != nil {
		t.Fatal(err)
	}
	x.saved = s.Clone()
	x.selection = ProjectAnchorReplacementSelection{Source: f.selection.Source, Replacement: s}
	original := f.x.f.f.d.execute
	f.x.f.f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
		a := args[4:]
		index := 0
		for index < len(x.saved.Members) && x.saved.Members[index].Phase == "running" {
			index++
		}
		switch {
		case len(a) > 1 && a[0] == "container" && a[1] == "create":
			if index >= len(x.contracts) || x.saved.Members[index].Phase != "create-dispatched" || !reflect.DeepEqual(a, x.contracts[index].createArgs()) {
				t.Fatal("unbound replacement Create", a)
			}
			x.creates++
			if x.mode == "create-error" {
				return "", errors.New("private Create failure")
			}
			id := fmt.Sprintf("%064x", 100+index)
			v := projectMemberContainerFixture(t, x.contracts[index])
			v.ID = id
			f.x.f.containers[id] = &v
			if x.mode == "create-lost-id" {
				return "", errors.New("private lost reply")
			}
			if x.mode == "create-returned-id-error" {
				return id, errors.New("private lost reply")
			}
			return id, nil
		case len(a) > 1 && a[0] == "container" && a[1] == "start":
			if index >= len(x.contracts) || x.saved.Members[index].Phase != "start-dispatched" || !reflect.DeepEqual(a, []string{"container", "start", x.saved.Members[index].Container.ID}) {
				t.Fatal("unbound replacement Start")
			}
			x.starts++
			if x.mode == "start-error" {
				return "", errors.New("private Start failure")
			}
			v := f.x.f.containers[a[2]]
			if x.mode != "false-start-ack" {
				*v.State.Running = true
				v.State.Status = "running"
				v.State.Pid = 100 + index
				v.State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
			}
			if x.mode == "start-lost-reply" {
				return "", errors.New("private lost reply")
			}
			if x.mode == "bad-start-ack" {
				return "wrong-id", nil
			}
			if x.mode == "cancel-start" {
				x.cancel()
			}
			return v.ID, nil
		case len(a) == 8 && a[0] == "container" && a[1] == "exec" && f.x.f.containers[a[2]] != nil:
			if !*f.x.f.containers[a[2]].State.Running {
				t.Fatal("namespace probe of stopped replacement")
			}
			n := x.saved.Retirement.Namespace.After.Runtime.Anchor.Namespace
			if x.mode == "wrong-member-namespace" {
				n.Inode++
			}
			return fmt.Sprintf("%d:%d", n.Device, n.Inode), nil
		case len(a) > 4 && a[0] == "container" && a[1] == "inspect" && a[2] == "--format" && a[3] == projectMountFormat:
			query := append(append([]string{}, args[:4]...), "container", "inspect")
			query = append(query, a[4:]...)
			out, err := original(ctx, query, env)
			if err != nil {
				return "", err
			}
			var values []container
			if err = json.Unmarshal([]byte(out), &values); err != nil {
				t.Fatal(err)
			}
			var rows strings.Builder
			for _, v := range values {
				b, _ := json.Marshal(projectContainerMounts{ID: v.ID, Mounts: v.Mounts})
				rows.Write(b)
				rows.WriteByte('\n')
			}
			return rows.String(), nil
		}
		return original(ctx, args, env)
	}
	return x
}
func (f *anchorReplacementAdapterFixture) open(t *testing.T, s projectnetwork.AnchorReplacement) *projectAnchorReplacementEngine {
	t.Helper()
	selected := f.selection
	selected.Replacement = s
	raw, err := openProjectAnchorReplacement(context.Background(), selected, f.f.x.f.f.d)
	if err != nil {
		t.Fatal(err)
	}
	e := raw.(*projectAnchorReplacementEngine)
	e.review.source.anchor.readBootID = func() (string, error) { return f.f.x.f.ns.BootID, nil }
	return e
}
func TestProjectAnchorReplacementAdapter(t *testing.T) {
	for _, mode := range []string{"pass", "caller-alias", "create-error", "create-lost-id", "create-returned-id-error", "start-error", "start-lost-reply", "bad-start-ack", "false-start-ack", "cancel-start", "commit-create-dispatched", "commit-created", "commit-start-dispatched", "commit-start-acknowledged", "commit-running", "anchor-process", "anchor-namespace", "pier-before-create", "lock-before-create", "foreign-mount", "changed-created", "wrong-member-namespace"} {
		t.Run(mode, func(t *testing.T) {
			f := newAnchorReplacementAdapterFixture(t)
			s := f.saved.Clone()
			e := f.open(t, s)
			h := e.history.Clone()
			original := s.Retirement.Clone()
			signals := f.f.x.f.signals
			removes := len(f.f.removes)
			sentinel := f.f.x.sentinel
			data, err := os.ReadFile(sentinel)
			if err != nil {
				t.Fatal(err)
			}
			stat, err := os.Stat(sentinel)
			if err != nil {
				t.Fatal(err)
			}
			targetPath := f.f.x.f.paths[s.Members[0].Creation.InstanceID]
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			f.cancel = cancel
			f.mode = mode
			if mode == "caller-alias" {
				f.selection.Source.Launches[0].Storage.RuntimeID = "changed"
				f.selection.Replacement.Members[0].Creation.Storage.Path = "changed"
			}
			commit := func(seq uint64, next projectnetwork.AnchorReplacement) error {
				if seq != f.saved.Sequence || next.Validate(h) != nil {
					t.Fatal("invalid replacement commit")
				}
				phase := next.Members[0].Phase
				if mode == "commit-"+phase {
					return errors.New("commit failed")
				}
				f.saved = next.Clone()
				if phase == "create-dispatched" {
					switch mode {
					case "anchor-process", "anchor-namespace":
						f.f.x.fault = mode
					case "pier-before-create":
						if err := os.Rename(targetPath, targetPath+"-retained"); err != nil {
							t.Fatal(err)
						}
						if err := os.Mkdir(targetPath, 0700); err != nil {
							t.Fatal(err)
						}
						if strings.HasPrefix(sentinel, targetPath+"/") {
							sentinel = targetPath + "-retained" + strings.TrimPrefix(sentinel, targetPath)
						}
					case "lock-before-create":
						if err := os.WriteFile(filepath.Join(targetPath, ".vere.lock"), []byte("99"), 0600); err != nil {
							t.Fatal(err)
						}
					case "foreign-mount":
						v := projectMemberContainerFixture(t, f.contracts[0])
						v.ID = fmt.Sprintf("%064x", 999)
						v.Name = "/foreign"
						v.HostConfig.NetworkMode = "none"
						f.f.x.f.containers[v.ID] = &v
					}
				}
				if phase == "start-dispatched" && mode == "changed-created" {
					f.f.x.f.containers[next.Members[0].Container.ID].HostConfig.Privileged = true
				}
				return nil
			}
			got, err := projectnetwork.ExecuteNextAnchorReplacement(ctx, h, s, e, commit, time.Now)
			pass := mode == "pass" || mode == "caller-alias"
			if (err == nil) != pass || !reflect.DeepEqual(got, f.saved) || !reflect.DeepEqual(got.Retirement, original) || f.f.x.f.signals != signals || len(f.f.removes) != removes {
				t.Fatal("unexpected outcome", mode, err, got.Members[0].Phase)
			}
			phases := map[string]string{"pass": "running", "caller-alias": "running", "create-error": "create-dispatched", "create-lost-id": "create-dispatched", "create-returned-id-error": "created", "start-error": "start-dispatched", "start-lost-reply": "start-dispatched", "bad-start-ack": "start-dispatched", "false-start-ack": "start-acknowledged", "cancel-start": "start-dispatched", "commit-create-dispatched": "planned", "commit-created": "create-dispatched", "commit-start-dispatched": "created", "commit-start-acknowledged": "start-dispatched", "commit-running": "start-acknowledged", "anchor-process": "create-dispatched", "anchor-namespace": "create-dispatched", "pier-before-create": "create-dispatched", "lock-before-create": "create-dispatched", "foreign-mount": "create-dispatched", "changed-created": "start-dispatched", "wrong-member-namespace": "start-acknowledged"}
			if got.Members[0].Phase != phases[mode] {
				t.Fatal("failed before intended boundary", got.Members[0].Phase, phases[mode])
			}
			creates, starts := f.creates, f.starts
			if got.Members[0].Phase == "start-dispatched" || got.Members[0].Phase == "start-acknowledged" {
				reopened := f.open(t, got)
				if _, er := reopened.CreateReplacement(context.Background(), got, 0); er == nil {
					t.Fatal("reopened Create replayed")
				}
				if er := reopened.StartReplacement(context.Background(), got, 0); er == nil {
					t.Fatal("reopened Start replayed")
				}
				recoverable := mode == "start-lost-reply" || mode == "bad-start-ack" || mode == "cancel-start" || mode == "commit-start-acknowledged" || mode == "commit-running"
				resolved, er := projectnetwork.ObserveAnchorReplacementRunning(context.Background(), h, got, reopened, func(seq uint64, next projectnetwork.AnchorReplacement) error {
					if seq != f.saved.Sequence {
						t.Fatal("stale recovery")
					}
					f.saved = next.Clone()
					return nil
				}, time.Now)
				if (er == nil) != recoverable {
					t.Fatal("unexpected read-only reconciliation", er)
				}
				if recoverable && resolved.Members[0].StartAcknowledged != got.Members[0].StartAcknowledged {
					t.Fatal("recovery invented acknowledgement")
				}
			}
			if f.creates != creates || f.starts != starts {
				t.Fatal("reconciliation repeated effect")
			}
			after, er := os.ReadFile(sentinel)
			if er != nil || string(after) != string(data) {
				t.Fatal("replacement changed test pier data", er)
			}
			info, er := os.Stat(sentinel)
			if er != nil || !os.SameFile(stat, info) {
				t.Fatal("replacement replaced pier data", er)
			}
		})
	}
}
func TestProjectAnchorReplacementBothMembers(t *testing.T) {
	f := newAnchorReplacementAdapterFixture(t)
	s := f.saved.Clone()
	h := anchorDrainHistory(f.selection.Source)
	for !s.Running() {
		e := f.open(t, s)
		var err error
		s, err = projectnetwork.ExecuteNextAnchorReplacement(context.Background(), h, s, e, func(seq uint64, next projectnetwork.AnchorReplacement) error {
			if seq != f.saved.Sequence {
				t.Fatal("stale")
			}
			f.saved = next.Clone()
			return nil
		}, time.Now)
		if err != nil {
			t.Fatal(err)
		}
	}
	if s.Validate(h) != nil || f.creates != 2 || f.starts != 2 || len(f.f.x.f.containers) != 2 {
		t.Fatal("replacement incomplete")
	}
	for i, m := range s.Members {
		if m.Creation.Generation != s.Retirement.Namespace.Drain.Review.Context.Members[i].Context.Member.Generation+1 {
			t.Fatal("wrong generation")
		}
	}
	if _, err := openProjectAnchorReplacement(context.Background(), ProjectAnchorReplacementSelection{Source: f.selection.Source, Replacement: s}, f.f.x.f.f.d); err == nil {
		t.Fatal("completed replacement reopened")
	}
}

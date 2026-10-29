package projectnetwork

import (
	"context"
	"errors"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// MemberStorage records only a fresh member's exclusive mount preparation. It
// creates no pier and conveys no engine/anchor/ship authority. The runtime ID is
// allocated by project registration; instance storage remains a separate claim.
type MemberStorage struct {
	ProjectID       string             `json:"project_id"`
	InstanceID      string             `json:"instance_id"`
	RuntimeID       string             `json:"runtime_id"`
	OperationID     string             `json:"operation_id"`
	ProjectRevision uint64             `json:"project_revision"`
	PlanDigest      string             `json:"plan_digest"`
	Phase           string             `json:"phase"`
	Location        pierstore.Location `json:"location"`
}

func NewMemberStorage(plan Plan, project string, revision uint64, instance, runtime, operation string) (MemberStorage, error) {
	digest, err := plan.Digest()
	if err != nil {
		return MemberStorage{}, err
	}
	s := MemberStorage{ProjectID: project, ProjectRevision: revision, PlanDigest: digest, InstanceID: instance, RuntimeID: runtime, OperationID: operation, Phase: "planned"}
	for _, m := range plan.Members {
		if m.InstanceID == instance {
			s.Location.Path = "fakeships/" + instance + "/" + strings.TrimPrefix(m.Ship, "~")
		}
	}
	return s, s.Validate(plan)
}
func (s MemberStorage) Validate(plan Plan) error {
	digest, err := plan.Digest()
	if err != nil || digest != s.PlanDigest || !protocol.ValidID(s.ProjectID) || s.ProjectRevision == 0 || !runtimePattern.MatchString(s.RuntimeID) || !protocol.ValidID(s.OperationID) || !pierstore.ValidPath(s.Location.Path) || s.Location.Uncertain || s.Location.Identity != (pierstore.Identity{}) {
		return errors.New("invalid project member storage intent")
	}
	found := false
	for _, m := range plan.Members {
		if m.InstanceID == s.InstanceID {
			found = s.Location.Path == "fakeships/"+m.InstanceID+"/"+strings.TrimPrefix(m.Ship, "~")
		}
	}
	if !found {
		return errors.New("member storage must use its reviewed instance and ship-named leaf")
	}
	switch s.Phase {
	case "planned", "dispatched":
		if s.Location.MountIdentity != (pierstore.DirectoryIdentity{}) {
			return errors.New("uncreated member storage claims a filesystem identity")
		}
	case "created", "observed":
		v := s.Location.MountIdentity
		if v.Device == 0 || v.Inode == 0 || v.Birth == "" {
			return errors.New("created member storage lacks its filesystem identity")
		}
	default:
		return errors.New("unsupported member storage phase")
	}
	return nil
}
func (s MemberStorage) Advance(plan Plan, expected, phase string, identity pierstore.DirectoryIdentity) (MemberStorage, error) {
	if s.Validate(plan) != nil || s.Phase != expected {
		return MemberStorage{}, errors.New("stale or invalid member storage phase")
	}
	next := map[string]string{"planned": "dispatched", "dispatched": "created", "created": "observed"}
	if next[s.Phase] != phase {
		return MemberStorage{}, errors.New("member storage cannot skip or replay a phase")
	}
	if phase == "observed" && identity != s.Location.MountIdentity {
		return MemberStorage{}, errors.New("member storage identity changed before observation")
	}
	s.Phase = phase
	s.Location.MountIdentity = identity
	return s, s.Validate(plan)
}

type MemberStorageDriver interface {
	CreateMount(context.Context, MemberStorage) (pierstore.DirectoryIdentity, error)
	InspectMount(context.Context, MemberStorage) error
}
type MemberStorageCommit func(expected, phase string, identity pierstore.DirectoryIdentity) error

// ExecuteMemberStorage requires a durable store callback and the common runner
// mutation/update fence. Never retry uncertain mkdir or adopt an existing parent.
func ExecuteMemberStorage(ctx context.Context, plan Plan, s MemberStorage, d MemberStorageDriver, commit MemberStorageCommit) (MemberStorage, error) {
	if s.Validate(plan) != nil || s.Phase != "planned" || d == nil || commit == nil {
		return s, errors.New("member storage requires fresh intent and durable dispatch")
	}
	advance := func(phase string, id pierstore.DirectoryIdentity) error {
		next, err := s.Advance(plan, s.Phase, phase, id)
		if err != nil {
			return err
		}
		if err = commit(s.Phase, phase, id); err != nil {
			return err
		}
		s = next
		return nil
	}
	if err := ctx.Err(); err != nil {
		return s, err
	}
	if err := advance("dispatched", pierstore.DirectoryIdentity{}); err != nil {
		return s, err
	}
	if err := ctx.Err(); err != nil {
		return s, err
	}
	id, createErr := d.CreateMount(ctx, s)
	// A successful identity observation remains evidence even when directory sync
	// or cancellation failed afterward. Commit it before returning uncertainty.
	if id != (pierstore.DirectoryIdentity{}) {
		if err := advance("created", id); err != nil {
			return s, err
		}
	}
	if createErr != nil {
		return s, errors.Join(errors.New("member mount creation did not complete; retain its original receipt"), ctx.Err())
	}
	if s.Phase != "created" {
		return s, errors.New("member mount creation returned no filesystem identity")
	}
	if err := ctx.Err(); err != nil {
		return s, err
	}
	if err := d.InspectMount(ctx, s); err != nil {
		return s, errors.Join(errors.New("member mount observation failed; retain its original receipt"), ctx.Err())
	}
	if err := ctx.Err(); err != nil {
		return s, err
	}
	if err := advance("observed", id); err != nil {
		return s, err
	}
	return s, nil
}

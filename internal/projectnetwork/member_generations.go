package projectnetwork

import (
	"errors"
	"reflect"
	"slices"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

// MemberGeneration selects an immutable creation contract, not a live process.
// Initial storage may still lack initialized pier identity; readiness establishes
// that separately. Project/anchor generation is independent of this counter.
type MemberGeneration struct {
	InstanceID  string             `json:"instance_id"`
	RuntimeID   string             `json:"runtime_id"`
	Generation  uint64             `json:"generation"`
	OperationID string             `json:"operation_id"`
	Mode        string             `json:"mode"`
	Container   ResourceIdentity   `json:"container"`
	Storage     pierstore.Location `json:"storage"`
}
type MemberGenerationSet struct {
	Binding RuntimeBinding     `json:"binding"`
	Members []MemberGeneration `json:"members"`
}

// ResolveMemberGenerations reconstructs a single causal sequence of completed
// replacements. It never adopts discovered resources, chooses by timestamps, or
// treats unfinished dispatch as success. Callers must first load validated store
// receipts. This pure selector neither persists a new ledger nor authorizes effects.
func ResolveMemberGenerations(plan Plan, launches []MemberLaunch, resumes []MemberResume) (MemberGenerationSet, error) {
	return resolveMemberGenerations(plan, launches, resumes, nil)
}

// MemberGenerationsAtBinding validates the complete causal history, then returns
// the exact historical prefix. Old receipts can remain valid after later creates;
// callers must still use ResolveMemberGenerations for current action admission.
func MemberGenerationsAtBinding(plan Plan, launches []MemberLaunch, resumes []MemberResume, binding RuntimeBinding) (MemberGenerationSet, error) {
	var found *MemberGenerationSet
	if binding.Validate(plan) != nil {
		return MemberGenerationSet{}, errors.New("invalid historical member binding")
	}
	_, err := resolveMemberGenerations(plan, launches, resumes, func(current MemberGenerationSet, _ []MemberResume) {
		if reflect.DeepEqual(current.Binding, binding) {
			v := current
			v.Binding.Members = slices.Clone(current.Binding.Members)
			v.Members = slices.Clone(current.Members)
			found = &v
		}
	})
	if err != nil {
		return MemberGenerationSet{}, err
	}
	if found == nil {
		return MemberGenerationSet{}, errors.New("member binding is not a prefix of retained history")
	}
	return *found, nil
}
func resolveMemberGenerations(plan Plan, launches []MemberLaunch, resumes []MemberResume, visit func(MemberGenerationSet, []MemberResume)) (MemberGenerationSet, error) {
	fail := func() (MemberGenerationSet, error) {
		return MemberGenerationSet{}, errors.New("member generations do not form one complete retained history")
	}
	b, err := InitialRuntimeBinding(plan, launches)
	if err != nil {
		return fail()
	}
	out := MemberGenerationSet{Binding: b}
	usedResources := map[string]bool{b.Bridge.ID: true, b.Anchor.ID: true}
	usedOperations := map[string]bool{}
	for _, l := range launches {
		usedOperations[l.OperationID] = true
		usedOperations[l.Storage.OperationID] = true
		usedOperations[l.Anchor.OperationID] = true
		usedOperations[l.Anchor.Prepared.OperationID] = true
	}
	for _, m := range b.Members {
		l, _, err := initialMember(plan, launches, m.InstanceID)
		if err != nil {
			return fail()
		}
		usedResources[m.Container.ID] = true
		out.Members = append(out.Members, MemberGeneration{InstanceID: m.InstanceID, RuntimeID: m.RuntimeID, Generation: 1, OperationID: l.OperationID, Mode: "boot", Container: m.Container, Storage: l.Storage.Location})
	}
	var through []MemberResume
	if visit != nil {
		visit(out, through)
	}
	seenResume, seenRetirement := map[string]bool{}, map[string]bool{}
	for _, s := range resumes {
		if s.Validate(plan) != nil || s.Phase != "running" || seenResume[s.OperationID] || seenRetirement[s.Retirement.OperationID] || usedOperations[s.OperationID] {
			return fail()
		}
		seenResume[s.OperationID], seenRetirement[s.Retirement.OperationID] = true, true
	}
	applied := make([]bool, len(resumes))
	for n := 0; n < len(resumes); n++ {
		selected, member := -1, -1
		for i, s := range resumes {
			if applied[i] || !reflect.DeepEqual(s.Retirement.Exit.Stop.Binding, out.Binding) {
				continue
			}
			for k, m := range out.Members {
				old := s.Retirement.Exit.Stop
				if m.InstanceID != old.InstanceID || m.Generation != s.PreviousGeneration || m.OperationID != old.LaunchOperationID || m.Storage.Path != old.Storage.Path || m.Storage.MountIdentity != old.Storage.MountIdentity {
					continue
				}
				if m.Mode == "resume" && m.Storage != old.Storage {
					continue
				}
				if selected != -1 {
					return fail()
				}
				selected, member = i, k
			}
		}
		if selected == -1 {
			return fail()
		}
		s := resumes[selected]
		if usedResources[s.Container.ID] || usedOperations[s.OperationID] {
			return fail()
		}
		applied[selected] = true
		usedResources[s.Container.ID] = true
		usedOperations[s.OperationID] = true
		m := &out.Members[member]
		m.Generation = s.PreviousGeneration + 1
		m.OperationID = s.OperationID
		m.Mode = "resume"
		m.Container = s.Container
		m.Storage = s.Retirement.Exit.Stop.Storage
		out.Binding.Members[member].Container = s.Container
		if out.Binding.Validate(plan) != nil {
			return fail()
		}
		through = append(through, s)
		if visit != nil {
			visit(out, through)
		}
	}
	return out, nil
}

// MemberResumesAtBinding returns the ordered completed replacements through one
// exact historical binding. Factories reopening a later creation need the history
// before that creation, not the current tail. The entire supplied history is still
// validated, and returned receipts never alias the caller's nested observations.
func MemberResumesAtBinding(plan Plan, launches []MemberLaunch, resumes []MemberResume, binding RuntimeBinding) ([]MemberResume, error) {
	if binding.Validate(plan) != nil {
		return nil, errors.New("invalid historical member binding")
	}
	var result []MemberResume
	found := false
	_, err := resolveMemberGenerations(plan, launches, resumes, func(current MemberGenerationSet, through []MemberResume) {
		if !reflect.DeepEqual(current.Binding, binding) {
			return
		}
		found = true
		result = make([]MemberResume, 0, len(through))
		for _, s := range through {
			result = append(result, cloneMemberResume(s))
		}
	})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.New("member binding is not a prefix of retained history")
	}
	return result, nil
}

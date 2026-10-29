package projectnetwork

import (
	"errors"
	"reflect"
	"slices"
	"time"
)

// ResolveRecoveredMemberGenerations extends one complete coordinated recovery
// with ordinary same-pier replacements. The recovery boundary remains explicit:
// pre-fault receipts stay in h, and tail contains only subsequent resumes. This
// is a pure selector, not persistence, a contract factory, or a second recovery.
func ResolveRecoveredMemberGenerations(h AnchorRecoveryHistory, r AnchorRecoveryReadiness, tail []MemberResume) (MemberGenerationSet, error) {
	out, _, _, err := resolveRecoveredMemberGenerations(h, r, tail, nil)
	return out, err
}
func cloneMemberGenerationSet(s MemberGenerationSet) MemberGenerationSet {
	s.Binding.Members = slices.Clone(s.Binding.Members)
	s.Members = slices.Clone(s.Members)
	return s
}
func recoveredLifecycleOperations(s MemberResume) []string {
	return []string{s.OperationID, s.Retirement.OperationID, s.Retirement.Exit.OperationID, s.Retirement.Exit.Stop.OperationID, s.Retirement.Exit.Stop.ReadinessOperationID}
}
func resolveRecoveredMemberGenerations(h AnchorRecoveryHistory, r AnchorRecoveryReadiness, tail []MemberResume, visit func(MemberGenerationSet, []MemberResume)) (MemberGenerationSet, map[string]time.Time, time.Time, error) {
	fail := func() (MemberGenerationSet, map[string]time.Time, time.Time, error) {
		return MemberGenerationSet{}, nil, time.Time{}, errors.New("post-recovery replacements do not form one complete retained history")
	}
	if r.Validate(h) != nil {
		return fail()
	}
	out := MemberGenerationSet{Binding: r.Binding}
	out.Binding.Members = slices.Clone(r.Binding.Members)
	processes := map[string]time.Time{}
	for _, m := range r.Replacement.Members {
		c := m.Creation
		out.Members = append(out.Members, MemberGeneration{InstanceID: c.InstanceID, RuntimeID: c.RuntimeID, Generation: c.Generation, OperationID: c.OperationID, Mode: "recovery", Container: m.Container, Storage: c.Storage})
	}
	for _, p := range r.After.Processes {
		processes[p.InstanceID] = p.Process.StartedAt
	}
	usedIDs := map[string]bool{r.Binding.Anchor.ID: true, r.Binding.Bridge.ID: true}
	for id := range recoveryHistoryContainerIDs(h) {
		usedIDs[id] = true
	}
	for _, m := range r.Replacement.Members {
		usedIDs[m.Container.ID] = true
	}
	usedOps := map[string]bool{r.Replacement.OperationID: true, r.Replacement.Retirement.OperationID: true, r.Replacement.Retirement.Namespace.OperationID: true, r.Replacement.Retirement.Namespace.Drain.OperationID: true}
	for _, m := range r.Replacement.Members {
		usedOps[m.Creation.OperationID] = true
	}
	notBefore := r.CapturedAt
	through := []MemberResume{}
	if visit != nil {
		visit(cloneMemberGenerationSet(out), through)
	}
	for _, s := range tail {
		if s.Validate(h.Plan) != nil || s.Phase != "running" {
			return fail()
		}
	}
	applied := make([]bool, len(tail))
	for range tail {
		chosen, index := -1, -1
		for i, s := range tail {
			if applied[i] || !reflect.DeepEqual(s.Retirement.Exit.Stop.Binding, out.Binding) {
				continue
			}
			for k, m := range out.Members {
				old := s.Retirement.Exit.Stop
				if m.InstanceID != old.InstanceID || m.Generation != s.PreviousGeneration || m.OperationID != old.LaunchOperationID || m.Storage != old.Storage {
					continue
				}
				if chosen != -1 {
					return fail()
				}
				chosen, index = i, k
			}
		}
		if chosen < 0 {
			return fail()
		}
		s := tail[chosen]
		old := s.Retirement.Exit.Stop
		if usedIDs[s.Container.ID] || old.Before == nil || !old.Before.StartedAt.Equal(processes[old.InstanceID]) || old.Before.CapturedAt.Before(notBefore) || old.Before.Runtime.ObservedAt.Before(notBefore) {
			return fail()
		}
		for _, id := range recoveredLifecycleOperations(s) {
			if usedOps[id] || recoveryOperationExists(h, id) {
				return fail()
			}
			usedOps[id] = true
		}
		applied[chosen] = true
		usedIDs[s.Container.ID] = true
		m := &out.Members[index]
		m.Generation++
		m.OperationID = s.OperationID
		m.Mode = "resume"
		m.Container = s.Container
		out.Binding.Members[index].Container = s.Container
		if out.Binding.Validate(h.Plan) != nil {
			return fail()
		}
		processes[m.InstanceID] = s.After.StartedAt
		notBefore = s.After.CapturedAt
		through = append(through, cloneMemberResume(s))
		if visit != nil {
			visit(cloneMemberGenerationSet(out), through)
		}
	}
	return out, processes, notBefore, nil
}

// RecoveredResumesAtBinding validates the full suffix before returning an exact
// causal prefix. A matching early binding cannot hide later ambiguous history.
func RecoveredResumesAtBinding(h AnchorRecoveryHistory, r AnchorRecoveryReadiness, tail []MemberResume, binding RuntimeBinding) ([]MemberResume, error) {
	if binding.Validate(h.Plan) != nil {
		return nil, errors.New("invalid recovered historical binding")
	}
	var result []MemberResume
	found := false
	_, _, _, err := resolveRecoveredMemberGenerations(h, r, tail, func(s MemberGenerationSet, through []MemberResume) {
		if !reflect.DeepEqual(s.Binding, binding) {
			return
		}
		found = true
		result = make([]MemberResume, 0, len(through))
		for _, v := range through {
			result = append(result, cloneMemberResume(v))
		}
	})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.New("binding is not a prefix after this recovery")
	}
	return result, nil
}

// SelectRecoveredMemberContext pins a recovered or later resumed creator and its
// exact process. Original readiness still owns the initialized pier; later work
// cannot replace that identity or relabel the original anchor receipt.
func SelectRecoveredMemberContext(h AnchorRecoveryHistory, r AnchorRecoveryReadiness, tail []MemberResume, instance string) (MemberContext, error) {
	out, processes, notBefore, err := resolveRecoveredMemberGenerations(h, r, tail, nil)
	if err != nil {
		return MemberContext{}, err
	}
	for _, m := range out.Members {
		if m.InstanceID != instance {
			continue
		}
		for _, original := range h.Originals {
			if original.InstanceID == instance && original.Storage == m.Storage && !processes[instance].IsZero() {
				return MemberContext{Member: m, Binding: out.Binding, Storage: m.Storage, KnownStartedAt: processes[instance], NotBefore: notBefore}, nil
			}
		}
	}
	return MemberContext{}, errors.New("recovered member lost original initialized pier or process")
}

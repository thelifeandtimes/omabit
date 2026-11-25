package managed

import candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"

func validateRuntimeProfile(id string) error {
	if _, err := candidate.Resolve(id); err != nil {
		return bad(err.Error())
	}
	return nil
}

func validateRuntimeChange(from, to string) error {
	if err := candidate.CompatibleSuccessor(from, to); err != nil {
		return conflict("runtime_incompatible", err.Error())
	}
	a, _ := candidate.Resolve(from)
	b, _ := candidate.Resolve(to)
	if a.ID != b.ID {
		return conflict("runtime_transition_required", "stop the ship, select an installed compatible runtime, and explicitly confirm the runtime change; preserve a backup before starting an untested build")
	}
	// Avoid changing the persisted encoding of an existing equivalent binding.
	if from != to {
		return conflict("runtime_binding_immutable", "keep the existing runtime selection encoding")
	}
	return nil
}

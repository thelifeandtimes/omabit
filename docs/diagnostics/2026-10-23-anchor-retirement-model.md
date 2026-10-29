# Preserve piers while retiring drained old containers

The pure retirement intent requires a completed clean drain and separately
observed successor namespace. Members retire serially: each exact container needs
committed dispatch and complete absence before the next dispatch. An uncertain
removal is resolved only by observation and never receives another removal.
Acknowledged and unacknowledged outcomes remain distinct.

Every observation retains all initialized pier identities, including those whose
containers have already disappeared. Remaining old process starts, clean exits,
anchor process/namespace, bridge, publications and original history must match.
Tests cover both reviewed anchor fault paths, failed commits, stale/changed
observations, lost replies, missing piers, forged order and input/callback aliases.
Focused tests pass in 2.054s and full projectnetwork race in 52.778s; vet/diff pass.
No persistent format or runner route changed. The concrete Docker adapter is a
separate unqualified draft; no coordinated runtime recovery is claimed. VM28 is
still running its earlier read-only anchor review fixture and has no retirement.

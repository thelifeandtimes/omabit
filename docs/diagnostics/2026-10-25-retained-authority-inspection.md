# Retained-authority inspection qualification

M314 adds a read-only internal inspector and explicit source CLI. It validates
retained descriptor/journal/reservation/admission/binding/closure evidence without
constructing a live Authority or changing active admission checks. Existing state
formats and immutable bundled generations remain unchanged. Shared read locks,
file identity/hash checks, journal replay and original process-domain observation
separate coherent evidence, owned-process closure and unresolved transport.

20 focused tests pass, including owned compositor process termination, socket-name
reuse, exact guardian closure, missing/wrong proof, altered claim/binding/mirror,
missing journals/locks, future markers, conflicting leases, publication during
inspection, other PID namespaces and CLI summary/details/error results. All280
collection tests pass. No failing runtime or unit trial occurred.

The actual native fixture inspects the live collection, original compositor/shell
closure, replacement compositor and final fenced state. CLI and internal results
match at every point. The original guardian proof becomes verified after death,
while two undeleted channels remain explicitly unresolved and replacement plugins
stay inert. No state is changed by inspection. A second native run inspects actual
reservation3 after initial-owner recovery, validating both original closure records
and retained initial-recovery evidence. All four scopes are inactive; three private
compositors exit zero. Sources, original histories, artifact pins, snapshot files
and closure proofs pass independent checks.

This supersedes M313's missing retained-evidence inspection path only. It neither
cleans up transport nor provisions a successor, and does not support losing/moving
the original runtime directory, copied evidence as authority, reboot/keyring unlock
or installed full-shell/Hyprland recovery. Exports exclude keyrings, account/session/
application-state files, piers and credentials. No production resource was touched.

Evidence: /home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-25-retained-authority-inspection

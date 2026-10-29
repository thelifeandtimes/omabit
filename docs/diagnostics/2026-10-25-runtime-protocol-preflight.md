# Protocol-pinned collection generations and preflight

Artifact4 adds an immutable runtime protocol declaration checked against literal
constants used by the bundled shared admission, authority and journal readers.
Static inspection executes no bundled code. The original artifact3 writer,
dependencies, support and deterministic generation are frozen and reproduce;
old readers refuse4. Formats1–3 remain readable with undeclared protocol, never
silently gaining capabilities. Nine actual historical native generations also
verify unchanged. Existing persisted authority/reservation/journal/admission
formats and original writer paths are unchanged.

233 focused tests pass, including16 new cases: exact complete-set checks, mixed
versions, old/new writer behavior, legacy byte preservation, missing pins,
unsupported/dynamic/duplicate declarations, mismatched readers, changed payload,
wrong profile, future requirements, lost publication response and CLI0/2/1 exits.
The read-only command and its limits are documented separately.

The actual private native protocol4 test attempts to enable a mixed generation and
to begin forward drain while a stable collection is active. Both refuse before any
manifest, reservation, gate, journal, constructor or channel-deletion change.
All stable/dev/edge generation sets then pass preflight. Seven controller crashes
recover post-adoption state and complete stable/dev/stable with15 subscriptions,
ten closed channels, six unchanged sessions and the original unknown DM.

Protocol4 initial-owner recovery writes/reads reservation3 and survives five
controller crashes with ten subscriptions/six channels. DMs-only candidate failure
and rejected deletion recover through original-owner cleanup and rollback, with
two recovery controller crashes,14 subscriptions/seven channels and stable
appearance restored. An ordinary artifact3 recovery regression passes five crashes,
15 subscriptions/ten channels. All preserve six sessions and unknown DM intent.
Independent canonical history, original closure/recovery and pinned artifact checks
pass; all seven scopes are inactive and six private compositors exit zero.

Two early native passes preceded a private-adapter refinement: compatibility gates
forward draining, not candidate cleanup during rollback. Final runs requalify that
boundary plus initial recovery and old-format behavior. No failing runtime trial
occurred. Qualification uses synthetic Eyre, private keyrings and headless Sway;
it does not certify installed Hyprland/full shell, publisher trust, public profile
activation, legacy live upgrade migration or cross-session bootstrap.

M311 supersedes M308–M310's missing explicit-generation capability check for
protocol4 preparation/read-only preflight and the isolated adapter. Persistent
bootstrap, interrupted provisioning and supported activation/upgrade delivery
remain. Production ships/services/plugins, tmux and temp-ur were untouched.
Exports exclude keyrings, account/application state, piers and raw credentials.

Evidence: /home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-25-runtime-protocol-preflight

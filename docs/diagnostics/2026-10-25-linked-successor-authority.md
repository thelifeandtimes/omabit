# Linked authority2 preserves canonical transition history

A new descriptor2 reader links the immutable successor intent, checkpoint, original
cleanup and original canonical mirror. It uses a separate successor mirror, retaining
the old one. A publication record binds its own preallocated inode and all staged
root/seal/mirror objects before promotion. Atomic no-replace publication supports
exact retries; unbound partial stages remain retained, and missing bound objects
are never recreated. The old authority reader and shared gate refuse descriptor2.
Existing descriptor1, ticket1, admission2, reservation1/2/3 and journal1/2/3 implementations
are unchanged. The explicit new gate can read authority1 and2; cleanup remains
possible if a historical link disappears and live guards become fenced.

Initial reservation on authority2 requires the canonical replacement transition at
enable-candidate, its exact request/effect, a new owned guardian/shell and an empty
gate. It revalidates the closed original plugin cohort while the adapter holds
admission. The logical profile, paths, IDs and endpoints must match; artifact hashes
may advance in new immutable generations. Once reserved, retry preserves its bytes
without demanding that an already-running app still have its pre-start state.
Normal journal enable/verify events record and commit the replacement, retaining
its predecessor and all prior events. No fake initial adoption or journal reset is
used. Atomic journal appends are read as snapshots rather than immutable inode links.

19 focused and all355 collection tests pass. Four actual publication controllers
are killed after prepared intent, seal, mirror and root promotion in each final
run. Exact retries retain all objects. Three real guards admit, bind a replacement,
and end successfully; a completed journal transition preserves the old activation.
Forty-eight atomic journal appends leave live guards running. Missing checkpoint
fences all three guards while allowing cleanup. Source, snapshots, closure, binding,
tickets and canonical history verify independently. Six scopes inactive. Both
failed early trials are retained: omitted test-shell permit helper, then an overly
strict refusal snapshot that missed the old reader's own lock initialization. No
core failure was hidden or retried on an existing lab. Final source review also
found a lost-publication-record allocation gap after the earlier354-test pass.
A still-present original temporary-record inode now proves an unbound stage;
a missing bound publication record refuses before allocating a new authority.
The new regression passes in the final19/355-test runs.

M319 supersedes M318's missing descriptor/admission contract at the private core
level. Application state, appearance and readiness observations in this new test
are synthetic; native Theme/DMs successor activation is not yet qualified. Complete
immutable generations must explicitly declare/bundle authority2 and pass compatible
set preflight before native construction. Public profiles, subsequent session loss,
full-shell/Hyprland, logout/reboot/keyring unlock and installed upgrades remain open.
The original descriptor2/publication writer and populated metadata are retained for
future compatibility tests. No production resources or current plugin installs were
changed; exported metadata contains no raw sessions/keyrings/piers.

Evidence: /home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-25-linked-successor-authority

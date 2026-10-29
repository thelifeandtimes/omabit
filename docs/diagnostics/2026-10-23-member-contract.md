# Project member storage/container binding — 2026-10-23

The private member contract now binds a reserved runtime ID, member identity,
container generation, distinct operation, completed anchor and boot-bound namespace
to its creation digest. It selects exactly `fakeships/<instance>/<ship>`, an already
recorded exclusive parent, and the pinned image. The member shares the anchor by
exact container ID and has no independent bridge, publication or persistent volume.
Resume requires the original pier/.urb identities and omits boot flags/pill mounts.

Read-only checks against temporary filesystem fixtures refuse a partial boot,
replaced or linked parent, substituted pier, sibling data and Vere locks without
changing those objects. Inspection tests reject peer mounts, writable pills,
unexpected bind options, replacement resources and independent network ownership.
Source race regressions and vet pass. The first resume fixture lacked `.urb/chk`;
that source-test failure is retained and the complete synthetic fixture passed.

This is a contract/validator slice; it performs no engine mutation and is not yet
connected to the runner store. Storage creation/claims, fresh full namespace/mount
observations, pill verification, dispatch journals, readiness and Stop/recovery
still need integration. No persistent format changed. Exact member Docker behavior
requires a separate private VM gate. Existing standalone execution is unchanged.

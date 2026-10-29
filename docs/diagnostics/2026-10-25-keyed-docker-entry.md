# Guest-only keyed Docker entry integration — 2026-10-25

The exact gateway, peer transport and case sources from appliance07 are now in
`scripts/qualification/private-chains/`. They match the active VM's frozen hashes;
no runtime is changed by this source integration. The existing native launcher and
public project/runner contracts remain unchanged. There is not yet a reusable
Docker VM preparation/launch command.

All21 inert tests pass:17 existing lifecycle checks and four host-entry refusals
for the case, peers, direct HTTP and SSE paths. Refusal occurs before engine or
subprocess construction. JavaScript syntax and diff whitespace checks pass.

The next stopped-disk inspector also passes four top-level tests and10 negative
subcases. It exports evaluator failure metadata only for exact scalar names and
allowlisted values; extra/private fields fall back to hashes and fixed diagnostics.
The original inspector and all prior evidence remain unchanged.

The fresh VM has reached gateway readiness and begun its first keyed ship boot.
Its messaging/recovery result is still pending. No public runner support, installed
plugin update or complete Docker qualification is implied. No user action is needed.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-25-keyed-docker-entry`.

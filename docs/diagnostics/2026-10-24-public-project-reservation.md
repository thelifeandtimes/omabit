# Public project reservation

`project-registration-v1` now exposes operator-only `POST /v1/projects`.
The server reconstructs the canonical fresh-galaxy plan from primitive fields,
requires the reviewed digest and explicit reservation/shared-administration
acknowledgements, and uses the existing actor/request-bound transaction. It
reserves membership and loopback ports only: no instance, pier, bridge, container,
worker or ship is created. Ordinary host-port checks remain part of admission.

`project-register` consumes the saved JSON from the offline planner, requires an
enrolled runner with an explicit identity pin and returns text or JSON receipts.
A lost or malformed response is uncertain. `--replay-only` can retrieve only the
original matching actor/request receipt; it cannot create a missing reservation
or perform fresh port checks. The client does not retry a mutation automatically.
Its error guidance and the bundled operator skill explain the lookup path.

Focused applied race tests pass: managed14.868s, client1.045s and collector1.028s.
Vet, diff and skill validation pass. Tests cover actual Unix RPC/store round trips,
same-request replay, missing/other-actor lookup, plan/consent refusal, observer and
origin refusal, management quiescence, bounded strict request parsing, lost reply
without repeat mutation, malformed/unbound success responses and CLI help.

During review, several initial refusal cases were found to stop at a missing
Content-Type header. The corrected tests send valid JSON headers and assert their
specific intended statuses/codes; their separate refined/applied passes are retained.
The request bound is 64 KiB for the full member set, separately from ordinary
4 KiB action bodies. The schema and historical registration fingerprints are
unchanged. The full schema40 baseline predates these public-route additions.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-public-project-reservation`.
VM33 is frozen before registration and therefore cannot qualify this new route.
Public runtime lifecycle APIs, actual combined CLI/runner guest dispatch and QML
remain open. No installed plugin, existing ship or production service was touched.

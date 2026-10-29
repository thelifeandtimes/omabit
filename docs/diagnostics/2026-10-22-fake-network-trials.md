# Private fake-network trial 01

2026-10-22. **Transport gate failed; no networking pass is claimed.** The fresh
no-NIC VM used the pinned Vere image and its own rootless Docker engine. It exited
cleanly after 1,403.7 seconds. Both newly created test ships stopped with exit 0
and no OOM. Production ships, services, installed plugins and the active desktop
were not accessed. [Machine-readable evidence](2026-10-22-fake-network-trials.json).

## Confirmed behavior

- Fresh `~zod` and `~nec` shared the private anchor's network namespace, distinct
  from the VM host. It had only loopback, no IPv4 route and no port publications.
- Public HTTP ports were 8081 and 8082. Independently verified administrative
  identities used ports 12321 and 12322, as recorded by each pier's `.http.ports`.
- Each independent cookie could read its own desk inventory. Each was refused
  when presented to the other ship.
- Retained container logs report local-only UDP ports 31337 and 31338.

The first authenticated `hi` request failed after these checks. No acknowledged
message was recorded. The original wrapper discarded the subprocess exit status
and stderr, so the elapsed time suggests a request timeout but does not establish
its cause. The logs alone do not distinguish a thread/request-path failure from
peer transport failure. Rejection and restart checks were not reached.

## Evidence and next trial

Run: `/tmp/omabit-import-guest-run-20261022-12`, appliance11. Offline extraction:
`/tmp/omabit-import-inspection-20261022-12`, 18 result files / 68,421 bytes.
The original backing and overlay disks are retained with verified hashes in
`/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-22-network01`;
original paths retain compatibility links. Only the verified redundant raw
conversion was removed. Raw container logs remain private in the inspection
directory; the report includes their hashes and selected public transport facts.

The next fresh trial adds a same-ship `hi` control on each ship, records pending
message attempts before sending, preserves credential-free command/HTTP failure
metadata, and collects bounded network/socket and Spider-thread diagnostics before
cleanup. An uncertain message is never automatically retried. These changes have
passed Python parsing, known jam vectors, atom-response checks and credential
redaction checks; they are not a runtime result.

The requested URL follows the [official Spider HTTP API](https://docs.urbit.org/urbit-os/base/threads/http-api).
The binary noun handling was checked against local Arvo source; the actual pinned
pill's compatibility still needs this runtime proof. Runner project ownership,
external endpoint allocation, cross-project duplicate identities, subscriptions,
two-runner networks and private Ethereum chains remain separate acceptance gates.

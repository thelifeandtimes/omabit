# Concurrent private project fixture — source qualified, runtime pending

M373, 2026-10-25. This is preparation evidence, not a live isolation pass.

The new guest-only case keeps project A's keyed `~zod`/`~nec` pair live while
project B independently boots the same names and internal ports, completes its
receiver restart and four-write recovery, then joins the comparison. Fixed A/B
receipt labels preserve both original recovery transcripts. Each project has a
separate chain, generated network keys, namespace and pier directories.

The comparison checks all six current container identities before and after:
four ships plus two gateways. Four same-name crossed-cookie requests must refuse.
Two fresh subscription/write cycles must stay within their own projects. Stopping
B must leave A's containers and five-write state intact. Existing recovery records
are not mutated by the comparison; extensions keep separate channels/receipts.
The independent Go reader requires all live snapshots, distinct public keys and
objects, both original peer proofs, exact state transitions, subscriptions and
clean shutdowns. Only fixed metadata and hashes leave its read-only audit VM.

Validation: 90 Python suite checks, then 11 focused reader-entry checks including
one new case; seven boot and three CPU admission checks; 30 Go reader tests/161
including subcases with race detection and vet. The compiled reader refuses host
execution with exit80 and no output. Tests use mocked Docker/systemd/HTTP or new
temporary files. No host engine or existing ship was accessed.

Appliance16 prepared successfully in20.14s. All12,358 payload entries and artifact
hashes were verified against the prior qualified inputs; only the selected
fixture code changed. Build SHA256:
`bdaa1f8bcb05f786ecf255964ce1ec49487f11bedeecdf8c6054496b12947b37`.

The launch has not been attempted: available memory is below the existing four-ship
admission requirement of32GiB. The new case uses a24GiB scope,14GiB guest, no swap,
two host CPU cores, a7200s deadline and no network device. It cannot combine with
single-project gateway/publication modes. Do not lower the admission check or
reuse a partial run. Prepare the next fresh runtime only when admission passes;
a stopped-disk audit remains mandatory. Managed schema42 regression is running
independently; public private-chain runtime dispatch remains unfinished.

This updates M372's next-step status and does not supersede its publication pass.
Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-25-private-project-isolation-source`.

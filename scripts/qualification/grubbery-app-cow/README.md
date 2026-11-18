# Fresh peer on a read-only task-fixture backing

This narrowly bound capsule tests newer app-source artifacts without duplicating
a full OS/engine/pill installation. It accepts only the exact ended task VM05 root
hash and original failed-result/source identity. That root, its pristine backing
and portable inputs are exposed read-only inside the isolated QEMU mount
namespace. A fresh output receives its own COW root and128MiB input filesystem.
The original failed capsule is never booted writable or relabeled as passed.

Before systemd starts, the private PID1 validates the predecessor marker and adds
an override only to the new overlay. A missing `/run` marker prevents automatic
manager startup, and its source-service invocation omits `--resume-running`.
The guest checks original journal rows and the cached standard pill before
allowing that service. It never exercises or bypasses installer override checks;
this is a source service/native product gate, not an installed-upgrade gate.

The helper accepts only two fixed cases. This case creates a new ordinary~nec,
instance, HTTP port, request IDs and result directory through the public runner
API, binding the already verified cached pill descriptor without another upload.
The original~zod container must remain stopped with unchanged StartedAt. Its
acknowledged pending app receipt and original operation/events/lease rows must
remain exact. No pending effect is reconciled, redispatched, rebound or abandoned.

The fresh peer runs the same framework create/adopt/commit/activate, two app source
versions, CLI/stdio parity, state/code checks, same-pier replacement and manager
restart replay gates. Its later native state probes remain fixed test-only HTTP,
not new production API authority. Only the new peer is started/stopped. An original
receipt replay cannot advance the new operation after a lost reply.

No work occurs without `--run`, explicit pinned inputs, clean matching artifacts,
closed original scopes and storage admission. Declare896MiB additional scratch
under the separately authorized48GiB aggregate cap; monitor every5seconds and
retain any over-limit failure. One runtime experiment only. The unchanged8GiB
no-NIC guest,10GiB/no-swap/one-CPU QEMU scope, independent2GiB staging supervisor
and deadlines apply. COW allocation of the full case remains unqualified until
measured. No existing installed namespace, service, user ship or tmux is exposed.

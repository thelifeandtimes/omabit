# Shared private-chain VM launcher — 2026-10-25

The repository's `scripts/qualification/imports/appliance.py boot` now accepts a
prepared private-chain appliance with the M346 runtime settings: separate read-only
pristine backing, fresh overlay, no NIC, up to two host CPU cores,7GiB guest RAM,
12GiB/no-swap/128-task scope,16GiB available-memory admission and3600s deadline.
Storage admission stays8GiB; its running reserve is2GiB. Existing import/fake
cases retain their original1GiB running reserve and resource policies.

The private path normalizes byte-for-byte to the M346 frozen driver after removing
the new malformed/mixed-case guards, restoring the experiment's absolute repository
constant and its blanket2GiB reserve. Eleven inert source tests pass: five launch
checks plus existing CPU-admission and console-parser regressions. They start no
process and verify exact writable/read-only mounts, changed backing refusal,
memory-floor refusal, mixed-case refusal before output and unchanged import limits.
This is source integration of qualified runtime settings; the new source entry
point has not yet run an end-to-end VM trial. A reusable preparation command is next.

M346's key-bearing successful disk is now preserved under the owner-only private
lab at `omabit-private-docker-labs/runtime-20261025-08`. The old lab path resolves
to that copy. Both original scopes were inactive, the complete bytes were verified
before removal of the temporary copy, and SHA256 remains
`ab362efea839f5121dcf73494fa2cb783ced077851e470e7c09150061a5aedb3`.
No runtime state is included in this code/metadata evidence export.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-25-private-chain-vm-launcher`.
The goal remains active. Runner private-chain classification/lifecycle and broader
public/release gates remain separate. No production or installed-plugin changes.

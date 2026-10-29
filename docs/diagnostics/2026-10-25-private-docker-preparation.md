# Reusable private Docker preparation — 2026-10-25

The repository now prepares a fresh private-chain appliance through
`scripts/qualification/private-chains/docker_appliance.py`. Its explicit inputs
are the pristine 12GiB VM, pinned source/toolchain/compiler results, original
stopped unkeyed foundation deployment, verified Node/addon library set, candidate
runner and pinned official Vere. No Docker, ship, key generation or VM starts
occur during preparation. The common launcher remains a separate command.

Thirty-two inert Python checks pass across preparation, guest entry and lifecycle;
the Go payload library passes five top-level tests/eighteen including subcases.
Coverage includes deterministic sealed input, input hashes, path escape, cycles,
links/special files, duplicates, byte/count bounds, malformed cpio, reused/nested
outputs, host refusal and the existing exact-container lifecycle checks. Both old
and candidate runner identities are bound into the guest init; the original guest
hash also remains required after PID1/QEMU/no-NIC/root-marker admission.

The source command completed preparation after two preserved precreation refusals:
`/tmp` was about14MiB below its8GiB quota admission, and the home filesystem could
not report user quota through the harness. Four exact stopped historical code
appliances were copied into private home storage, checked byte-for-byte and left
accessible through their original paths. The preparation then used a fresh `/tmp`
output with both storage checks passing. No admission threshold was lowered.

The sealed inventory has12,357 files and642,437,372 expanded bytes. Every name,
size and SHA256 matches M346's successful appliance. The init implementation now
comes from the repository payload package with explicit old/candidate bindings;
the compressed archive is deterministic and checked after writing. These facts
qualify source preparation, not yet the new source-entry runtime. A fresh no-NIC
VM09 is running that end-to-end gate with the common source launcher.

The new build is `/tmp/omabit-import-appliance-private-chain-20261025-08`.
Evidence is `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-25-private-docker-preparation`.
Only source, logs without credentials, public metadata and inventories are exported;
no payload archive, chain database, key, pier or raw application state is included.
Public runner private-chain lifecycle and the broader goal remain open.

# Real named-pier import and resumed runtime

The same-filesystem gate passed on fresh disposable guest run `05`, using appliance
`06`. It completed in 727.05 seconds and powered off cleanly. The
[frozen evidence](2026-10-22-import-runtime.json) records source/artifact hashes,
scope admission, receipts, snapshots and the extracted disk evidence. Earlier
[failed trials](2026-10-22-import-runtime-trials.md) remain preserved.

The test booted a newly created fake `~zod` from the pinned Vere image and pill,
observed its public identity and stopped it cleanly. The real collector's guided
preview inspected the arbitrarily named source without changing it. Import moved
that exact filesystem object to `fakeships/zod/zod`, preserving its inode and every
reviewed file's bytes and metadata. Completion stayed stopped with identity pending.
Re-presenting the terminal request returned the same operation; no implicit runtime
started.

An explicit Start then reached running/healthy state and resolved the identity.
The actual rootless container had one bind mount: its exclusive instance parent
`/srv/urbit/piers/fakeships/zod` at `/urbit`, with `/urbit/zod` selected as the pier.
The guest's published loopback endpoint reported `~zod`. Explicit Stop completed
with no active runtime, followed by clean guest shutdown.

This used the real Omabit installer, collector, runner and rootless Docker 29.9.1.
The guest had two virtual CPUs and 7 GiB RAM, under an actively verified outer
one-core/12 GiB/no-swap/128-task scope. Its early port/resource probe passed after
adding missing guest networking modules. No host kernel modules, engine, production
service, installed plugin or protected ship was inspected or changed. The VM had
no NIC or host filesystem/desktop/credential shares.

This qualifies the same-filesystem CLI/runtime path only. Cross-filesystem copy,
interruption/recovery, full disks, replaced sources and the guided UI connected to
a real runner remain separate gates. The existing offscreen UI test uses inert
dispatch; this run does not change that distinction. The runner's generic hosting
qualification remains `unqualified`; this evidence does not rewrite that status.

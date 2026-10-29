# Interrupted cross-filesystem copy

Trial `09` passed the **partial-copy preservation** checkpoint using frozen
appliance `08`. The private guest powered off cleanly in 728.66 seconds. The
[evidence](2026-10-22-import-interruption.json) binds its binaries, appliance,
operation, original disks, extracted receipts and console. This run used the
existing import implementation; it does not test the later restoration prototype.

The fresh fake source booted on the separate guest disk and stopped cleanly.
Guided preview left it unchanged. The actual collector submitted import operation
`op-de4fda159752badde367726cc5f9989e` for `fakeships/zod/zod`.
The test observed `copying` with 24,379,392 bytes reported, then stopped only the
guest owner's runner service. Cancellation left a 44,564,480-byte partial
destination and the complete held source (7 files / 1,016,840,204 bytes).

The original source name stayed absent. Its held object retained its identity and
all recorded source bytes/metadata. Restarting the guest runner marked the original
operation interrupted, preserved stopped intent and the recovery fence, and started
no runtime. An actual `recover-import` request returned `invalid_state`; the test
verified that this refusal changed neither retained tree and started nothing.

Original guest disks, backing image and receipts remain. Extraction used an
offline, isolated conversion without mounting the disk. Its redundant converted
image was removed only after verifying the original disks, backing image and every
extracted result hash. No production ships, services, plugins or desktop state were
accessed.

This establishes preservation through a graceful runner stop/restart, including
an unchanged recovery refusal. It does not establish power-loss behavior or a
usable recovery command. The [restoration prototype](2026-10-22-import-restore-prototype.md)
is the next implementation checkpoint; runner integration and real recovery
qualification remain open.

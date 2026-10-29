# Authority3 admission from a linked checkpoint

A separate continuation_authority publishes descriptor3 from an existing
continuation-intent2/checkpoint2. Its publication sidecar and mirror have distinct
names; every staged inode is bound before promotion. Original descriptors, intents,
mirrors and checkpoints remain unchanged. The new reader dispatches authority1/2/3;
older descriptor readers refuse3. Ticket1, admission2, reservation1/2/3 and canonical
journal1/2/3 formats do not change.

The checkpoint_links reader checks checkpoint2 and its checkpoint1 ancestor,
canonical immutable source identities/hashes, both cleanup receipts and journal
prefixes without acquiring another canonical/runtime lock or opening an ancestor
runtime. Current journal publication is atomic and mutable; it is read as a snapshot,
not pinned as an immutable inode. Missing parent or grandparent evidence prevents
continued guard use; no missing object is reconstructed.

Initial reservation requires the exact canonical forward replacement effect, a
fresh guardian/gate and a verified closed predecessor cohort. The logical profile,
paths and endpoint references remain fixed while compatible generation hashes may
change. Exact reservation retries retain the same original permit and binding.
Historical state is not reset or re-adopted to get through admission.

21 focused and475 full tests pass in fresh namespaces. The first focused run had
two fixture failures: its preservation comparison included newly published files.
The corrected fixture distinguishes those files, keeps the old binding at its own
path and checks that binding unchanged. The focused pass exercises same-generation
continuation; the final full suite explicitly changes all three candidate generation
hashes. The runtime implementation is identical across these trials.

Each passing run kills four actual publication controllers and later admits three
owned guard processes. The guards exit0 and the original guardian closes with a
waitpid-echild receipt.48 atomic canonical appends retain live guards. Deleting the
parent or grandparent checkpoint fences them while allowing clean end. Lost replies,
missing bound stages/records, changed cohorts, invalid requests and old readers are
also covered. All three scopes are inactive. Exported source hashes and metadata
shapes/link consistency are independently checked; application, readiness and
appearance observations are synthetic in this slice. These unit fixtures close and
remove their own temporary runtime directories after exporting results.

This supersedes M326's missing authority/guard contract from checkpoint2 at the
private core level only. No compatible native immutable package currently advertises
authority3. Prepare a complete new generation and capability preflight, then verify
actual Theme/DMs activation/subscription recovery. Claimed-successor loss, repeated
cleanup/checkpoint from authority3, public installed profiles, reboot/keyring and
real-backend release gates remain. No production resources were touched.

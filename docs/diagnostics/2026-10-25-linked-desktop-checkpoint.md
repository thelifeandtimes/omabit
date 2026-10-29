# Linked-runtime checkpoint with referenced ancestry

The separate successor_checkpoint writer publishes immutable format2 checkpoints
outside the runtime directory after verified linked desktop cleanup. It copies the
current metadata and keeps the original checkpoint as an inode/hash reference. The
parent checkpoint's body is excluded from the copied inventory, avoiding recursive
embedding. Current canonical immutable links, both cleanup receipts and the original
checkpoint must remain available; copied runtime metadata does not replace them.
The source inventory and total encoded record retain explicit size/file bounds.

Reading validates the copied descriptor2/publication/claim, committed reservation,
original guardian closure, cleanup2 evidence, parent checkpoint and canonical journal
prefix. Later append-only history is permitted. Missing/replaced ancestors, unknown
formats, conflicting targets or changed copied bytes refuse intact. Existing writers
and their contracts remain unchanged; the original checkpoint reader refuses2.
No checkpoint reconstructs runtime files or grants activation authority.

20 focused and430 full tests pass on the first runs. The native test repeats the
artifact4-to5 activation and second desktop cleanup, then kills the checkpoint
controller before and after no-replace publication. One original checkpoint remains
stable through replay. After the private keyring/probe and compositors have stopped,
the second runtime directory is renamed intact and replaced by an empty directory.
A new controller reads/retries the checkpoint without restoring any runtime objects;
opening the retired authority or live inspector refuses. The parent checkpoint body
is absent from the current copied files. All31 injected controller kills across the
combined scenario pass; three compositor exits are0 and three test scopes inactive.

The native application proof uses copied Theme/DMs, inert Omabit/appearance adapters,
private unlocked keyring, software Wayland and synthetic Eyre. Before retirement,
six sessions and the unknown DM remain unchanged without login or message replay.
Independent verification checks source hashes, metadata inodes/bytes, both checkpoints,
cleanup, canonical history and immutable artifact hashes. Unchanged plugin packages
are referenced from retained M323 evidence. No keyrings, raw sessions/account/app
state or piers are exported. Production ships/plugins/services remain untouched.

Supersedes M324's missing durable linked checkpoint only. This writer accepts the
existing authority2 with a checkpoint1 ancestor; it is not yet a repeated-admission
protocol. Next extend explicit successor ownership/admission to consume checkpoint2,
with bounded lineage references and capability refusal for older bundles. Public
installed profiles, pending/unbound sessions, reboot/keyring and real-backend release
qualification remain separate gates.

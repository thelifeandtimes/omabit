# Preserve verified desktop evidence outside the runtime directory

A separate immutable format1 desktop-checkpoint receipt captures the original
verified cleanup record, authority descriptor/reservation/history, file identities,
exact metadata bytes and canonical journal prefix. Capture requires the original
files and verified cleanup. The coordinator must be outside the selected runtime
directory. Existing reservation/history parsers were extracted unchanged into pure
validators; active wrappers retain their original filesystem checks.

Publication syncs an exclusive staging file and uses no-replace rename. Failed
publication preserves the stage. Lost replies reopen the same checkpoint; retries
never replace its inode. Reads validate copied evidence and require the original
canonical coordinator, lock, cleanup receipt and journal prefix. Later journal
appends are allowed; changing that prefix is not. Copies are historical evidence,
not live capabilities. No successor admission or public mutating command exists.

17 focused tests and all314 collection tests pass. Tests cover failed/lost publish,
exact retries, inventory limits, changed bytes/digests/history, absent or unverified
cleanup, missing original runtime before first capture, and completed read/retry
after runtime loss. Native qualification uses real private Theme/DMs transports:
13 provisioning kills, two cleanup kills and two checkpoint-publication kills.
Four channels/five subscriptions close; six sessions and unknown DM are preserved.
After both private compositors and keyring stop, the runtime directory is renamed
aside and a fresh empty directory replaces its pathname. Checkpoint read and a new
controller retry succeed unchanged, while old Authority and live-file inspection
refuse. Three scopes are inactive; both compositors exit zero. Source,10 immutable
artifacts, original parser bodies, metadata bytes/inodes and receipts independently
verify. No production resources changed.

M316 supersedes M315's inability to read already-captured evidence after runtime
path loss, only following verified committed-state cleanup. First capture after
loss, lost cleanup records, pending/unbound activation, successor ownership,
logout/reboot/keyring unlock, power-loss durability, full-shell/Hyprland and public
activation remain unqualified. The original writer and populated receipt are
retained for future compatibility tests. No old runtime is deleted or reconstructed;
archives exclude keyrings, private application/session state, piers and credentials.

Evidence: /home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-25-desktop-runtime-checkpoint

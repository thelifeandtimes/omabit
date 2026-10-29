# Reviewed private gateway input generation — 2026-10-25

M386 seals and independently inspects generation06: 12,352 files / 834,036,904
bytes. It includes M384's qualified first-start initializer and both unchanged
retained-gateway programs. Fresh unbooted appliance21 supplies the complete source
set; initial-only appliance20 intentionally cannot supply it.

The preparer verifies appliance/build/inventory/artifact hashes and produces
owner-only immutable files and directories. A newly built temporary collector
then runs the public local private-chain-inputs command in private HOME/XDG roots.
Its independent Go implementation re-reads the manifest, hashes every file and
rejects links, mounts, unexpected entries, permissions or changed objects. It
reports runtime_authorized=false. It contacts no runner/engine, opens no database
through a library and executes none of the input programs.

The manifest is44782315f0cf0bc1732b57a76387dde975dd7d23decaf6f5cf7a0258c514d9db.
The reviewed foundation is unchanged. Generation04 retains its original
2e501616bb6421b701a5c9faffdabdd286c4599e44658a6721b2d419e583a560 manifest.
No installed/default generation, service, plugin or ship changed.

Three orchestration mistakes remain recorded: the initial review omitted the
existing init artifact and refused before output; an early inspector ran before
sealing finished and refused before building; a renamed inspector accidentally
changed the source-package path and refused before compilation. Fresh inspector03
uses independently checked source paths and passes. No failed output was repaired
or old bundle relabeled. Generation05 was never created.

This supersedes M366's applicability to the current initializer and M385's pending
generation inspection, while retaining earlier evidence and bytes. It supplies
an inert reviewed generation, not gateway startup authority or supported delivery.
The programs retain their disposable-VM guards. Concrete backend/workspace/input
admission, runtime/public lifecycle and concurrent projects remain separate gates.

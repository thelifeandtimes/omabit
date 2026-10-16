# Pier storage, discovery and real-ship setup

For the distinction between independent fake identities and double-booting a real
ship, see the [live fake-zod networking inspection](diagnostics/2026-10-14-fake-zod-networking.md).
Fakes use container-local Ames; their containers still permit outbound HTTP.

**New development workflows:** [Storage locations, import and deletion](storage-management.md)
documents configurable roots, GUI host-path move/import and permanent pier deletion
introduced in schema 14 and retained in RC.3/schema 16. The layout below explains
the default convention retained by each configured location.

The owner-user layout lives below `/srv/urbit/piers/`. Additional locations use
the same layout. Each instance has an exclusive mount directory containing one
actual pier child. New imports and known-identity creations use the ship name
without `~` as that child; legacy `pier/` children are also recognized:

```text
piers/
├── fakeships/
│   ├── project-name/
│   │   ├── zod/zod/.urb/
│   │   └── nec/nec/.urb/
│   ├── project-name2/
│   ├── zod/zod/.urb/
│   ├── bus/bus/.urb/
│   └── marzod/marzod/.urb/
├── livenet/
│   ├── galaxies/zod/zod/.urb/
│   ├── stars/tocwex/tocwex/.urb/
│   ├── planets/lismyl-lismyl/lismyl-lismyl/.urb/
│   ├── moons/sampel-palnet-moons/<instance>/<moon-name>/.urb/
│   └── comets/
│       ├── testing-comet-1/pier/.urb/
│       └── random-comet-2/pier/.urb/
└── archive/
    ├── zod.20260901.bak/
    └── sampel-palnet.20240303.bak/
```

Project/group folders are optional. Use names without the leading `~`; the pier
parent directory can have a friendly name independent of the instance name and actual
Urbit identity. Active paths use letters, digits, hyphens and underscores, with
alphanumeric-leading components (up to eight components / 384 characters).
Archive filenames are not constrained by the active-path parser because archive
contents are never scanned, enrolled or started automatically.

The API's `pier_path` names the actual child containing `.urb`, independently of
its basename. The parent is reserved exclusively for that registration and may
contain only that pier directory. Docker binds the parent at `/urbit`; Vere uses
`/urbit/<pier-name>` for boot, resume and maintenance. No sibling ship or project
directory is mounted. Both mount-root and pier directory identities are recorded.

A newly mined comet has no known `@p` before first boot, so that flow still uses
its provisional `pier` child. Importing an existing comet uses its detected name.
Changing a registered path requires a stopped, reviewed move/relink; installation
does not rename existing live piers.

## Discover and enroll

**Setup → Pier storage & livenet import** shows the runner's root, discovered piers,
registered locations and moved-directory reviews. The fleet also shows a prompt
when discovery finds an unenrolled or moved pier. Discovery is read-only, bounded
and does not descend into a candidate's `.urb` contents or read its keys.

1. Fully stop the source ship. A directory such as
   `.../livenet/planets/lismyl-lismyl/lismyl-lismyl` is discovered by its `.urb`
   child; renaming it to `pier` is unnecessary.
2. Copy as the selected pier-owning user. Ordinary user-owned workspace modes are
   supported, while `.urb` retains private state checks and symlinked storage and
   shared/special state files are refused. Owners can use ordinary remote
   SSH/SFTP; the separate management-only
   SSH endpoint does not become a shell or file-transfer endpoint.
3. Review the discovered path in the guided import flow and choose its configured
   storage location. The runner detects identity/fake-live mode and derives the
   destination, instance name and available HTTP port. Confirm the stopped source.
4. Enrollment validates compatible Vere epoch history and leaves the pier stopped.
   Start verifies its identity. Imported livenet ships first resume with Ames
   disabled, then stop and resume normally only after the expected identity matches.
   Mutating maintenance is unavailable until identity verification completes.

Galaxy/star/planet/moon classifications must match the expected identity. The same
livenet identity cannot be registered twice on one runner. Independent development
fakes may share an `@p`, but cannot share a physical pier or overlapping directory.
There is no fixed registration-count cap. Identity, port and storage reservations
still apply to each registration.

CLI examples (global runner selection precedes the command):

```bash
./build/urbit-collector --runner HOST --expected-runner-id RUNNER_ID piers
./build/urbit-collector --runner HOST --expected-runner-id RUNNER_ID \
  enroll-pier my-planet --path livenet/planets/my-planet/sampel-palnet \
  --ship '~sampel-palnet' --fingerprint DISCOVERY_FINGERPRINT \
  --http-port 18090 --request-id enroll-my-planet --confirm-stopped
```

This livenet import is **in-place enrollment of a fully transferred stopped
pier in the tree**. Moving between two managed runners uses the distinct
[stopped host-handoff workflow](host-handoff.md), with source fencing and a streamed
verified transfer. Docked executables are not used; execution stays on the pinned image.

## Moves, missing directories and archive

Stop a managed ship before moving it. **Move…** in Pier storage performs a durable,
same-filesystem no-overwrite rename of the entire mount directory and its named
child. Keep the child name when relocating its parent. Renaming a leaf inside the
same parent is also supported. It leaves the ship stopped;
Start is separate.
New project parents can be created, but a pier cannot be nested inside another pier.

For a direct filesystem rename, move the whole ship directory. The runner keeps the old location binding and
reports the missing directory. Discovery recognizes the same pier at the new path
and offers **Reconnect location…**. This requires the stopped registration, current
revision and fresh discovery fingerprint. Directory device/inode and Linux birth
timestamps distinguish a moved object from a replaced or inode-reused one. The
filesystem must expose stable directory birth timestamps for enrollment.

Removing/replacing a registered directory never triggers a fresh bootstrap or
automatic adoption. Reconnection handles the same filesystem object, not an
arbitrary replacement copy. Cross-host handoff remains distinct from this local
no-overwrite rename. Historical pier restoration is not an interface operation.

The runner deliberately excludes `archive/` from discovery and startup. Completed
cross-host moves put their fenced source there while retaining the journal tombstone.
Manually archiving a registered pier does not erase its history; its old enrollment
reports missing storage. Archive contents are not selectable restore points.

## New ships and key boot

New fake/comet registration accepts a convention-relative **Pier path**. Omabit
creates only the dedicated parent mount directory. The official
Vere binary creates the absent named child directly in its permanent host
location. Identity/HTTP verification marks the ship initialized and leaves that
same container running. There is no staging pier, promotion, automatic Stop or
replacement container during first boot.

**Setup → Add a ship → Galaxy / star / planet / moon** requires only an owned `@p`
and private key file.
The instance name is the ID without `~`; the path is inferred from its tier.
The runner allocates available HTTP and Ames ports atomically. Advanced configuration
contains port overrides. A Boot pill dropdown chooses bare bootstrap or Recommended
Apps for first boot. Only the key filename
enters QML/argv. The collector reads a bounded, owned,
private regular file; key contents travel in the authenticated RPC body and are
staged as mode 0600 under the runner's private `managed/keys` directory. The journal
contains a digest, never the key. Registration leaves the ship stopped.

Explicit Start invokes the official runtime with `-w` and a read-only `-k` file
and checks the resulting identity. After success the private host key file is
removed; its read-only bind mount lasts until that first container is stopped.
Subsequent explicit starts create initialized runtimes without key/creation flags
at the recorded `/urbit/<pier-name>` path. Interrupted provisioning remains a review state,
never an automatic second bootstrap over existing data.

```bash
./build/urbit-collector --runner HOST --expected-runner-id RUNNER_ID \
  create-ship --ship '~sampel-palnet' --key-file /private/ship.key \
  --request-id register-my-planet --confirm-network
```

Optional CLI overrides are `--http-port`, `--ames-port`, `--bootstrap bare|recommended`
and `--pill URL_OR_FILE`.
New owned ships publish the selected Ames UDP port on the host; HTTP stays on
loopback. Existing development registrations retain their previous networking.
The Omarchy plugin's `recommendedPillUrl` setting supplies the direct release-asset
URL for Recommended Apps. Bare bootstrap ignores it and sends no pill override to
Vere. New comet/fake registrations use the same selection. URLs/local CLI files are
copied/downloaded by the collector, bounded at
4 GB (4,000,000,000 bytes), hashed and uploaded into private runner artifact storage
before registration. See the [release-pill plan](pill-release.md) for the confirmed
desk bundle and artifact distribution plan.

The local moon has passed real key boot and same-identity recovery. The 0.20 live
acceptance booted a fresh fake directly in its permanent directory, kept the first
container running, then explicitly stopped/resumed it without changing its mount
or pier identities. All six initialized local piers and eight registrations have
been migrated. Owned livenet import and broader network qualification remain.

## Management data and format

The owner-user SQLite management database is at
`~/.local/share/urbit-runner/managed/runner.sqlite`.
Opaque IDs remain internal identifiers for ownership, work directories and history.
They do not determine the operator-facing pier path. The runner accepts only
**schema 16** and the exclusive mount-root/named-pier layout. The original layout migration
was an offline one-shot; old-layout execution remains unsupported. The additive
11/12→13 format upgrade preserves existing layout/runtime records, including
handoff fences, and records explicit first-boot selection so older runners cannot
ignore it. Historical operation records remain evidence.

Each pier's actual Urbit state remains in its own `.urb` directory. A management
database backup is not a backup of those event logs/checkpoints.

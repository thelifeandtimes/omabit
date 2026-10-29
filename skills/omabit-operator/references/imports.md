# Import a stopped pier

The source path is an absolute path **on the selected runner host**, not necessarily
on the desktop. Confirm that it is the intended, permitted source and is stopped.
If stopping it is already authorized, use the normal management path and verify
completion; otherwise resolve that missing authorization before stopping it. Never
probe an explicitly excluded source to satisfy this checklist.

## Review

```sh
urbit-collector --config-dir REGISTRY --runner NAME --expected-runner-id RUNNER_ID \
  preview-pier --source /absolute/source --storage-root default
```

The guided preview reads the pier's identity/fake-live metadata and binds the
filesystem review. Use its `source.fingerprint`, `ship`, `instance_id`,
`destination`, `http_port`, `storage_root` and `in_place` values. Do not infer the
ship or network from a directory label or guess a free port.

Current source permits local imports up to 8 TiB / 8192 entries / depth 32;
published0.3.2 retains the former 100 GB bound until upgraded. Cross-host handoff
is separately capped at 100 GB. A remote host-path import runs locally on that
runner. Cross-filesystem imports need the source's logical size plus 8 GiB free;
same-filesystem renames need 8 GiB headroom. Do not bypass these checks or infer
free space from allocated sparse-file blocks. Transfer deadlines scale with size.

New import destinations use `<instance>/<patp-without-tilde>`; `zod/zod` is valid.
Only that exclusive instance parent is mounted into Docker. Existing legacy
`pier` children can be imported through the guided path; do not rename a registered
or running pier manually as a cosmetic repair. Newly mined comets are a separate
flow: they have no known identity before first boot and may use a provisional leaf.

A preview rejection can name the quoted relative entry and the failed ownership,
permissions, mount, link or file-type check. Keep that evidence. Do not recommend a
recursive chmod/chown or lock deletion as a generic fix. A foreign owner, nested
mount or link may indicate that the selected tree is not the intended import source.

## Import and observe

After reviewing the actual preview and the task's authorization:

```sh
urbit-collector --config-dir REGISTRY --runner NAME --expected-runner-id RUNNER_ID \
  import-pier INSTANCE --source /absolute/source --path DESTINATION \
  --ship '~SHIP' --fingerprint FINGERPRINT --http-port PORT \
  --request-id REQUEST_ID --identify --confirm-stopped
```

The confirmation authorizes moving the stopped source into managed storage. A
same-filesystem import renames it; a cross-filesystem import copies, verifies and
then removes the retained source according to its journal. In-place enrollment
can have no move. Keep the original request ID and poll its operation receipt as
described in [operations](operations.md).

A completed import leaves the ship stopped. Its identity may remain pending until
the first explicit Start. Start only when that effect is part of the authorized
task, using the current instance revision; then verify healthy state, identity and
endpoint. Import success alone does not prove that the runtime can start.

## Failure or uncertain acknowledgement

The public `external_import` fields describe stages, progress and the retained
holding path. The original source pathname may have disappeared while its contents
remain in a private holding directory. Preserve that path, the destination, the
operation and their recorded bindings. Do not copy the source again, recreate its
old path, clean up a partial target or start another copy to resolve uncertainty.

Review the latest snapshot. If `runtime_status.import_recovery_available` is true
and recovery is appropriate to the confirmed failure and authorized task, submit:

```sh
urbit-collector --config-dir REGISTRY --runner NAME --expected-runner-id RUNNER_ID \
  operate INSTANCE recover-import --revision REVISION --request-id NEW_REQUEST_ID
```

This is a new explicit recovery operation, not replay of the original import.
Use the current revision and observe its receipt to completion. If recovery is
unavailable, retain the evidence and report what remains unresolved; do not bypass
the runner's source, ownership, identity or epoch checks.


## Restore a source after partial copying

`recover-import` finishes retained enrollment; it does not restore a partial copy.
On a runner advertising `restore-import`, a stopped failed copy can instead offer
`runtime_status.import_restore_available`. This hint still requires filesystem
review. Do not infer permission to move any excluded source from a status flag.

```sh
urbit-collector --config-dir REGISTRY --runner NAME --expected-runner-id RUNNER_ID \
  preview-import-restore INSTANCE --revision REVISION
```

Use `--expected-actor ACTOR` on both restoration commands to bind the current
inventory connection's `actor`. The preview also returns `actor`; compare it with
that same identity. If a saved management credential changes, refresh and review
again before submitting. This is a public management identity, not a ship secret.

Review `source` → `restore_to`, `destination` → `retain_at`, the byte/file counts,
the stage/continuation flag and `review_digest`. This restores the held source's
original name, archives the current partial destination and retires the failed
registration. It does not complete the import or start either copy. When that
effect is authorized and the preview matches the intended objects:

```sh
urbit-collector --config-dir REGISTRY --runner NAME --expected-runner-id RUNNER_ID \
  restore-import INSTANCE --revision REVISION --restore-digest REVIEW_DIGEST \
  --request-id NEW_REQUEST_ID --confirm-restore
```

Track the exact operation. A successful receipt retains `import_restoration`
paths and digest after the failed instance disappears. Preserve the archive.
A fresh import is a separate reviewed action and gets a new instance revision.
Never retry the original import implicitly after restoration.

If restoration stops, preview the same failed registration again. Once move
intent is durable, continuation is bound to its original review; a changed or
ambiguous object is refused. Keep the old operation receipt. After its terminal
state is known, an authorized continuation uses a new request ID and current
revision with the same reviewed digest. Reusing an old request ID retrieves that
attempt's receipt; it cannot restart the work. Do not rename files by hand to force
a pending review to pass. Older runners without the capability remain observable.

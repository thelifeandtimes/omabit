# Source restoration: runner and CLI integration

The development runner now exposes reviewed partial-copy restoration, backed by
schema22. This advances the [fixture-only prototype](2026-10-22-import-restore-prototype.md).
It has not been installed on a live runner or qualified through a real pier
restoration in the private VM. The [evidence](2026-10-22-import-restore-integration.json)
records exact source/artifact identities and the tests below.

## Public flow

With a pinned enrolled runner advertising `restore-import`:

```sh
urbit-collector --runner LAB --expected-runner-id RUNNER_ID \
  preview-import-restore INSTANCE --revision REVISION

urbit-collector --runner LAB --expected-runner-id RUNNER_ID \
  restore-import INSTANCE --revision REVISION --restore-digest REVIEW_DIGEST \
  --request-id NEW_REQUEST_ID --confirm-restore
```

Preview requires an idle, stopped failed copy and operator access. It exposes the
two proposed moves, source/destination file and byte counts, current revision,
digest and continuation phase. Private per-object manifests stay inside the runner.
The CLI checks capabilities, runner/instance/revision bindings and bounded results;
its help is available offline. Operator skill guidance and `help imports` describe
this flow separately from `operate … recover-import`, which finishes retained
enrollment.

Acceptance records the actor-scoped request and approved digest. The serialized
worker revalidates the filesystem review before publishing move intent. Once that
intent exists, continuation cannot replace its digest or object inventory. Every
phase persists transactionally with the operation's runtime evidence. Failed
publication keeps the latest durable journal; it cannot publish a proposed phase
that did not commit. Stop and observation retain the recovery fence. Configuration,
Start, deletion and other recovery paths cannot bypass a pending restoration.

After the filesystem driver verifies both retained trees and directory durability,
completion atomically retires only the failed registration, retains its tombstone
revision and publishes the successful receipt. The receipt's `import_restoration`
field retains paths/digest after the instance is gone. No pier data is deleted,
no new import is submitted and no runtime starts. A new registration advances the
revision. Old requests still replay their original receipts and cannot affect a
replacement instance.

## Validation

- Full race suites passed for runtime qualification, managed runner, client,
  protocol, collector CLI and runner CLI. Focused tests cover invalid approvals,
  phase skipping, changed reviews, crash after each durable phase, terminal write
  failure, cancellation, retirement/replay/name reuse and stale observations.
- Runtime wrapper fixtures preserve both trees while enforcing durable callbacks,
  Stop/Start fences and continuation. RPC fixtures check authentication, observer
  refusal, revisions, busy instances, host serialization and private-manifest
  omission. Client fixtures reject mismatched capabilities, identities, actors,
  revisions, digests, paths and completion receipts.
- The pinned original schema21 writer at `b23312876c611e1d083bdcc9468469f7e716d749`
  produced a reproducible baseline fixture. The current migrator preserved it,
  and the original reader refused the resulting schema22 database twice without
  changing it. Earlier historical-writer migration tests also passed. This does
  not extend the frozen RC.12 host/release qualification claims.
- Twelve built-binary smoke cases passed in fresh private namespaces without host
  HOME, credentials, sockets or engine access. One earlier harness expectation
  incorrectly assumed that Go help renders flags with two hyphens; the command
  succeeded, and the corrected harness passed. Vet, skill validation, local links
  and whitespace checks passed.

## Remaining qualification

Run the actual preview/restore/replay and a fresh import against a disposable
partially copied fake pier in the private VM. Integrate the guided UI with the new
preview and retained completion receipt. Empty targets, full disks, process crashes,
power loss and device renumbering remain separate gates. No installed plugin,
production service, protected ship, tmux ship or `~/temp-ur` data was accessed.

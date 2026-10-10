# Checkpoint 6P: durable import of a qualified stopped fake

Runner/collector import integration and panel **0.6.0** are implemented. Go,
vet/race and QML/model checks passed. **Managed live acceptance passed** on
September 28 UTC; see the [evidence](../diagnostics/2026-09-28-managed-qualified-import.md).
The 6O copier already passed its real Vere qualification.
Desktop 0.6.0 and its bundled binaries are installed; shell status IPC confirmed
the loaded version after restart. The upgraded runner advertises the import
capability. Alpha and panel-check were
independently rechecked running/healthy. The prior desktop snapshot was archived at
`/tmp/opencode/omabit-panel-before-6p-20260928.tar`.

## Supported source and operation

The runner lists completed `stopped-pier-copy` qualifications with matching image,
Vere, engine and pill pins; resolved container outcomes; completed transfer and
both resumes; and preserved original content. Only the original `source/` tree is
eligible. No caller-supplied filesystem path, key or external pier is accepted.

The operator selects source run ID, content digest, new instance name and HTTP
port, with explicit stopped-source confirmation. Registration, port/source
reservation and the queued operation commit atomically. Each source is reserved
once, including failed imports. Actor-scoped exact replay precedes current source
lookup and returns the original operation without another copy.

The worker uses the host work gate and rechecks qualification metadata, current
source content, ownership, containment and disk admission. It creates a fresh
runner-owned target, journals intent before filesystem work and progress per file,
and uses the qualified content-verifying copier. Success leaves an initialized,
**stopped** instance. Its next explicit start uses resume flags, not bootstrap flags
or a pill mount. The original qualification source is retained unchanged.

Partial/failed/interrupted imports remain quarantined. Start, observation and Stop
cannot turn them into new bootstrap requests. Even a completed private copy is
quarantined if management restarts before its operation outcome was committed.

**Private schema compatibility:** the first accepted import atomically advances
SQLite `user_version` to 2. Afterward, use **6P or newer** for all management starts.
Older runners refuse this version rather than discard import quarantine metadata.

## Interfaces

Capability: `import-qualified-fake` (operator only).

- `GET /v1/imports`: runner-bound source catalogue, including `claimed_by`.
- `POST /v1/imports`: confirmed registration/import request; returns a durable
  `import-fake` operation, normally 202 or 200 on exact replay.
- Snapshot `instance.import`: source/digest, expected/copied file and byte counts,
  timestamps and uncertainty.

```bash
/home/thelifeandtimes/repos/omabit/build/urbit-collector \
  --format text --runner workstation \
  --expected-runner-id runner-f496feed1f7b08f31a5d82b94ac2f5fd import-sources
```

Setup → Add a ship → Existing pier exposes the same catalogue and an immutable
source/destination review. It requires fresh operator access and a separate
stopped-source checkbox. The panel does not refresh a selected digest silently,
start the imported runtime, or retry a failed submission.

## Required upgrade

1. Press **Ctrl+C** in the terminal running management `serve-fake`. The runtime
   containers continue independently; keep the SSH listener running.
2. Install the rebuilt runner:

   ```bash
   sudo /usr/bin/install -D -o root -g root -m 0755 \
     /home/thelifeandtimes/repos/omabit/build/urbit-runner \
     /usr/local/lib/urbit-runner/checkpoints/6p/urbit-runner
   ```

3. Start management and leave it running:

   ```bash
   sudo runuser -u urbit-runner -- /usr/bin/env -i \
     HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
     /usr/local/lib/urbit-runner/checkpoints/6p/urbit-runner \
     serve-fake --acknowledge-development
   ```

No standalone qualification rerun is needed. The agent will use the already
accepted source `185ff2ed58b9abdd6861dfe64d6650f5`, digest
`7d897a1765a4b0d0b511b2a3d5ec5d721aad07f413d0cdd219aca71a98908ff8`, to test import,
progress, replay, duplicate-source refusal, and explicit initialized-pier resume.
Source bytes plus 8 GiB disk are required for copy; resume retains the 5 GiB
available-RAM admission. A low-memory host can complete import and remain stopped.

External upload/staging and real-pier identity/format validation, key/comet boot,
Ames networking and scheduled maintenance remain later backend increments.

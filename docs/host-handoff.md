# Stopped cross-host handoff (0.22)

This development workflow moves a ship's **current, cleanly stopped pier** between
two enrolled operator runners. It does not offer historical pier backup/restore:
resuming an old copy after the original advances can regress Urbit networking state.

## Operator workflow

1. Stop the source ship and wait for a clean stopped observation. Resolve retained
   containers or uncertain maintenance before moving it.
2. Open its **Overview → Move to another host → Choose destination**. Select the
   destination runner and a new instance name. Blank HTTP/Ames ports allocate free
   ports; explicit ports are checked for reservations and actual host listeners.
3. Choose **Move stopped ship**. This permanently fences the source against normal
   operations, configuration, relinking, routines and startup resume.
4. After completion, open the destination ship. It remains stopped. Review browser
   routing, published Ames and maintenance schedules, then explicitly Start it.
   Livenet identity verification first runs with Ames disabled; networking is enabled
   only after the expected identity matches. Fake identity is checked on Start.

Destination networking is host-specific. The source's @p, fake/livenet class, Ames
publication intent, current pier contents and outstanding roll-before-run guard are
preserved. A new destination name starts at revision 1; a reused name advances
past its deleted revision. The collector reads the actual revision from the
matching transfer's snapshot. Host ports can differ.
Source history and maintenance policies remain on the source; policies are not
automatically enabled on the destination. A moved source remains a fenced journal
record. There is no fixed registration-count cap, so retained move records do not
consume a limited allocation of registration slots.

CLI equivalent:

```bash
urbit-collector --runner SOURCE --expected-runner-id SOURCE_ID \
  move-host INSTANCE --revision N \
  --to-runner DESTINATION --to-runner-id DESTINATION_ID --to-instance NEW_INSTANCE \
  --request-id move-attempt-1
```

The collector streams directly between authenticated RPC connections. It writes
no pier archive to the desktop and does not require management shell/SFTP access.

## Interruption and fencing

Phase receipts are durable on both runners, independently of the desktop process:

- Source: `preparing → ready → moved`, permanently fenced from preparation onward.
- Destination: `reserved → receiving → received → complete`, fenced until complete.
- Failed/interrupted attempts retain their phase and error. **Continue stopped move**
  (or the same CLI selection with a new request ID) reconciles that phase. It never
  starts either ship or automatically rolls back to the old source.
- A changed source tree, replaced directories or unrecorded filesystem outcomes stay
  quarantined for review. They are not silently adopted or overwritten.

The source verifies its frozen content again before commitment, atomically moves
the whole mount root under `piers/archive/moved-<handoff-id>/pier`, and signs a receipt
binding the transfer, both runner/instance IDs, ship, content digest and verification
key. The destination requires that receipt before releasing its fence. The collector
obtains the verification key from the pinned source connection and commits only
after the destination reports verified receipt. This is a coordinated operator
workflow, not a federation-wide identity/ownership protocol.

Source files and private spool artifacts are retained as development evidence, not
exposed as selectable restore points. Archive is excluded from discovery and Start.

### Reservation replay compatibility

`POST /v1/handoffs` is both a reservation and continuation endpoint. First
acceptance returns HTTP 201 with the reserved transfer. Repeating the same
actor/request ID and input returns HTTP 200 with the **current phase of that exact
transfer**, including `receiving`, `received`, or `complete`. Existing RC.11
collectors depend on this response to resume without retransmitting received data.
This behavior requires no new header or protocol version.

Schema20 runners atomically retain the original acceptance status and transfer
body in the private operation audit receipt alongside the reservation. Replay
does not rewrite that receipt or repeat the mutation. Changed input under the
same actor/request ID is rejected before lookup of current transfer state. A
missing or replaced transfer is refused rather than re-reserved or substituted.
Activation retains its original completed response on exact replay, including
after later configuration changes. Legacy transfer signature bytes are preserved.

## Bounds and storage format

- Linux/AMD64, pinned official runtime and permanent `/urbit/pier` layout.
- Maximum stopped tree: **100 GB (100,000,000,000 logical bytes)**, 8192 entries,
  depth 32. Stream overhead allowance:
  32 MiB. Files/directories only; links, traversal, duplicate entries, special files
  and trailing archive data are refused. Extracted content is hashed again.
- Export requires the transfer size plus 8 GiB free disk. Upload admission requires
  two transfer sizes plus 8 GiB; extraction rechecks space. Bulk disk work shares the
  host work gate. Each stream/phase wait has a 30-minute deadline.
- Host-owned spool files are private, hashed and fsynced. The source's ephemeral
  signing key stays in its private journal; snapshots expose only the verification key.
- Schema **12** adds fencing to the permanent schema-11 format. Upgrade 11→12 is
  additive and preserves registrations; older layout schemas remain unsupported.
  Schema-11 binaries refuse the new database rather than ignoring fences.
  Source version 0.23 advances to schema **13** for first-boot selection and preserves
  all existing handoff state when upgrading schema 11 or 12.

## Qualification

Automated tests cover fake/comet stopped transfer and initialized resume, source
changes, failed receive and commit, continuation after manager restart, signature
binding, ordinary-operation/startup fences, observer refusal, bounded SSH streaming,
unsafe tar entries, format migration and the mapped desktop interaction.

The optional published-RC.11 collector regression uses a SHA-256-pinned old binary
against temporary Unix sockets, SQLite journals and synthetic handoff backends:

```bash
OMABIT_RC11_COLLECTOR=/path/to/urbit-collector-linux-amd64 \
  go test ./internal/managed -run '^TestHandoffPublishedRC11CollectorContinuation$' -count=1
```

It interrupts before source commitment after destination receipt, restarts the
destination, and repeats the same old-collector command. Completion must preserve
the original reservation audit receipt and perform no second data upload.

Real two-host handoff acceptance passed with the already-stopped `panel-check` fake:
about 1.05 GB transferred, source Start refused, destination identity verified by
explicit Start, then returned to stopped. Other fleet states were preserved.
See the [live acceptance record](diagnostics/2026-10-09-stopped-host-handoff.md).
Owned-ship movement and broader failures remain development gates. A subsequent
sourcebox reboot preserved the stopped destination and completed handoff record;
see the [reboot acceptance](diagnostics/2026-10-10-sourcebox-reboot.md).

The 100 GB bound is configured and boundary-tested with sparse-file metadata;
a full 100 GB live transfer has not been qualified.

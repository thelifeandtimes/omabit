# Runtime recording development contract

Issue20's recorder is in development. Current installed log behavior is unchanged.
The initial [inert lifecycle comparison](diagnostics/2026-11-17-recording-backend-inert.json)
supports proceeding with a runner-owned stream recorder outside the container.
This choice remains provisional until actual Docker/Vere lifecycle, interruption,
retention and pressure gates pass. The existing bounded sanitized tail cannot be
treated as a comprehensive original recording.

The fixture compares fixed inert processes, not existing ships or tmux sessions.
All four paths retain350-line bursts, a12288-byte line, diagnostic atoms, fake
credential text, terminal redraws and stderr. Pipes preserve separate stream
labels and raw LF; PTYs combine streams and apply terminal LF→CRLF processing.
Receive-clock timing and offsets cover all raw pipe/PTY/tmux bytes in the final
run. Cross-stream ordering is observed read order, not proof of simultaneous
process-write order. `script` has its own timing plus untimed wrapper headers.

| Tested stop path | Observed child/lifecycle result |
| --- | --- |
| Signal the owned pipe workload | TERM, observed child exit7, final output retained; first trial timeout remains unresolved, three following runs pass |
| Signal the owned PTY workload | TERM, observed child exit7, final output retained; streams combined |
| Signal util-linux script2.42.4 wrapper | Inner child receives TERM and exits7; wrapper returns0 and its recording omits the child's accepted final write |
| Stop tmux3.8 private server | Inner child receives HUP and exits129; final terminal write fails EIO and is absent; new-session client status0 was only submission |

These results apply to the exact fixture/version/stop paths. A purpose-built
wrapper could behave differently and would need its own signal/reaping/status
tests. They do not invalidate the user's useful manual script/tmux incident
capture. They show why a plain wrapper cannot yet replace the frozen container
lifecycle contract. No container entrypoint or TTY configuration was changed.

The planned integration attaches the runner-owned recorder before Start to a
verified prepared container's non-TTY streams. It preserves native runtime signal
delivery, exit status and one-runtime ownership. Capture must live independently
of desktop UI/SSH connections. Recorder restart and host disappearance need
explicit continuity/gap evidence; reconnecting a transport is not proof that
intervening bytes survived. Docker buffering/rotation and process bootstrap capture
remain to qualify. A future interactive console can have a separately qualified
PTY contract without changing ordinary recording semantics implicitly.

The persistent format must keep original timed bytes in owner-private runner
storage outside the writable container layer, with stable session/runner/instance/
container/configuration/operation and runtime-artifact bindings. It must preserve
prior sessions after Stop/replacement, expose durable paging/follow cursors and
explicit gaps, and distinguish queued Start from actual process exit/errors.
Metadata excludes bootstrap keys, environment credentials and secret contents.
Terminal display must contain escape/control bytes; merely opening a record must
not execute terminal commands.

Retention must have configurable per-instance/host limits, incident pins, oldest
retained coverage, visible retirement markers and a persistence reserve. A failed
record write must surface and resume with a gap, without consuming space reserved
for the pier. Pressure, rotation, crash/reopen and replacement are required
disposable gates, not reasons to prune existing user state during development.

Original bytes remain immutable. Redacted displays/exports are derived and labeled
with transformation/omission reasons; diagnostic `0w...` atoms are not erased by
the historical blanket rule. Authenticated owners can choose an original export;
sharing/publishing remains an explicit choice. CLI/API and safe MCP diagnostics,
history/recovery UI and versioned migration/support evidence remain to implement.
No production deployment follows from this design checkpoint.

## Source-only frame format

`internal/recording` now supplies a development codec and non-TTY Docker stream
decoder. It is not wired into runner launch or advertised as recording support.
No existing persistent state is changed. The version1 segment header binds a
session ID, exact metadata hash, segment/first-sequence, wall-clock origin and
previous digest. Each bounded64KiB frame retains kind, sequence, observed wall
time, monotonic elapsed time, exact bytes and a chained SHA256. These checks detect
corruption/rebinding/reordering; file ownership and authorization remain separate
storage responsibilities. Gap and exit payloads now have separate version1 typed
canonical validation. Unknown/duplicate fields, alternate encodings, unsupported
reason/version and invalid counts/process bindings refuse before write. Even a
valid frame checksum cannot promote semantically invalid lifecycle metadata.

A failed/short write permanently poisons that encoder. Reads expose only complete
verified frames and retain the last complete offset/digest on a partial tail.
Partial data never satisfies `errors.Is(err, io.EOF)`. Continue in a new segment
with explicit gap evidence; never truncate an original to make it readable.
Complete encoded offsets do not claim fsync durability. Wall-clock adjustments
may go backwards; monotonic elapsed order cannot. Segments are written by one
serialized coordinator, with transport observation order stated honestly.

The transport decoder streams arbitrarily large declared payloads through bounded
chunks and keeps stdout/stderr identity. It preserves received partial payload
prefixes before reporting interrupted coverage. Writer failure reports known
unrecorded bytes and performs no replay. Transport EOF remains distinct from a
verified process exit. Callers must verify container identity/ownership and TTY
mode and close blocked transports on cancellation. This library opens no path,
connects to no engine and runs no command.

Canonical owner-private attachment metadata is bounded to4KiB and binds the exact
runner/mode, instance/project, configuration revision, original operation/request,
prepared container, launch contract, image digest, Vere version and transport.
It contains no command/environment/secret-content fields. A random session ID and
canonical UTC attachment time bind the first segment header and metadata digest.
Equivalent time zones encode identically; full uint64 revisions stay exact.
Metadata content binding does not establish filesystem ownership or authority.

Typed gaps use finite reasons and optional **known** missing-byte counts; omitted
counts mean unknown, never zero. A gap cannot become a process exit. Typed exit
evidence requires a nonzero process start/finish, exact container and bounded exit
code, keeping OOM/dead/native-error flags separate. It records an independently
observed terminal process, not a Start submission or stream EOF. Wall-clock
reversal is preserved independently from monotonic receive ordering. A failed
Start with no launched process cannot fabricate an exit. Before-container setup
failures still need a separate runner diagnostic/recording contract; these types
describe attachment to an already prepared container.

Later segments may start another monotonic epoch; durable storage must verify the
preceding digest, exact metadata, sequence and explicit interruption gap instead
of inventing continuous coverage. A later recorder attachment can link a distinct
previous session ID. That link is evidence association, not proof that intervening
bytes survived. Storage/runtime admission must compare recorded container/process
facts with the independently verified selected runtime.

Source race tests cover bursts/long lines, raw atoms/fake credentials/controls/
invalid UTF-8, NTP-style wall-clock reversal, every partial-tail boundary, injected
short/disk-pressure writes, bounded lengths, corruption/reordering, segment links,
interrupted Docker payloads and cancellation. Owner-private filesystem storage,
locking/index/cursors, retention/reserves, lifecycle coordination, durable sync,
actual runtime capture and APIs/UI/export remain to implement and qualify.

## Source-only private session files and paging

The Linux storage layer pins an explicitly authorized, already private owner
directory. New session directories require unique validated IDs; metadata, lock
and original frame files are0600 within0700 directories. Metadata/header and
directory entries sync before creation returns. Append serializes and fsyncs
before successful writer acknowledgement. It checks exact stream/container
binding and refuses replaced, linked, exposed or unexpectedly resized files.
An existing or incomplete session is never reused or repaired.

Read-only pages verify the bounded original prefix and bind each cursor to its
session, exact metadata, sequence and chained digest. Reconnecting returns the
same subsequent bytes. A changed or lost cursor refuses instead of silently
rewinding. Partial/corrupt snapshot tails remain explicit and originals remain
unchanged. A live append can transiently expose a partial snapshot; rereading the
same verified cursor can later observe completion. Writer-lock presence is an
observation at read time, not process health. Snapshot EOF requires separate
terminal-process evidence. Merely reading returns bytes, not terminal commands.

This first implementation scans a single bounded segment; it has no trusted
persistent index, writer reopen/continuation, rotation, global/per-ship retention,
incident pins or runtime service. Metadata/root selection and operator API
authorization remain the caller's responsibility. Reader pages establish the
verified prefix visible at read time; only successful append acknowledgement
claims the writer's fsync path. Crash/power-loss and fsync-failure gates remain
unqualified, including any cursor whose bytes disappear after an interrupted write.

An explicit session byte ceiling and free-space reserve are checked before writes;
refusal preserves originals. This reserve is admission against current free space,
not an exclusive reservation against other filesystem writers. Hitting a limit
sends no ship signal and prunes nothing; the future capture coordinator must
surface the incident and record missing coverage when storage can resume.
Synthetic source tests cover close/reopen, long/raw bytes, exact/live paging,
partial/corrupt/lost coverage, links/root replacement, stream/exit rebinding and
quota/reserve refusal. No installed recording or persistent migration is created
by adding this unwired library.

## Source-only explicit raw JSON boundary

Ordinary Frame/Page JSON omits original payload bytes, including their base64
representation. `ExportOriginalFrame` explicitly selects a bounded original-byte
encoding while retaining its frame timing/kind/digest metadata. It copies bytes;
changing an export cannot change the retained frame. Binary storage/decoding is
unchanged, including credentials, controls, NULs and invalid UTF-8.

This is an encoding boundary, not a redacted view, complete session export,
authorization grant or independent digest-chain verifier. The future API must
select actual verified retained data and enforce exact target/operator/raw-export
authorization. No recorder, public raw endpoint or UI is enabled. Source/race
checks cover implicit omission, exact explicit octets/metadata, export isolation
and invalid/oversize frames. Redaction and lifecycle integration remain pending.

Development metadata2 adds an immutable attachment identity: full project target
with generation/creator, native ship, pier continuity, actual daemon identity,
endpoint digest, host boot ID and the external recorder's host namespaces. A
project also binds its already verified anchor network namespace. An ordinary
pre-Start attachment omits that project field; this metadata makes no claim that
a container process or its eventual namespace has been observed. Independent
process/lifecycle/coverage evidence remains required.

New store captures require metadata2. Version1 remains readable with its exact
original canonical bytes and header digest; a missing identity stays unknown.
`CaptureIdentity` refuses legacy names instead of assigning a current generation,
and new capture with legacy metadata refuses before creating files. Writer
admission copies nested metadata so a retained caller pointer cannot rebind it.
The binary frame codec stays at1 and binds the versioned metadata by SHA256.
Focused race and pointer/binding/refusal checks pass. The source rehearsal
`scripts/test-recording-metadata-upgrade.py` is prepared to build an actual
3218e6f writer/reader plus a declared helper, verify original bytes through new
reads/cursors/reopen, and prove old-reader refusal of metadata2 unchanged. That
actual original-writer gate is still pending; no live capture, native/installed
upgrade or public recording API follows from these unit checks.

The [actual original-writer rehearsal](diagnostics/2026-11-18-recording-metadata2-source-upgrade.json)
now passes for frozen candidate5041406. Original3218e6f bytes, lifecycle evidence,
cursor paging and repeated reopen remain unchanged; the old reader refuses
metadata2 without modifying it. The first helper expectation failure remains
failed and retained. A compact original-writer fixture and source/helper input
manifest now run in ordinary focused tests. This supersedes the preceding
original-writer deferral for this exact source boundary, with no live/native or
installed qualification implied.

## Source-only derived redacted windows

`RenderRedactedFrames` produces an explicitly selected, bounded derived view of
already verified retained frames. It never changes original bytes or includes
them implicitly in JSON. Diagnostic `0w` atoms, multiline context and long traces
remain visible. Finite credential-field rules omit values after Authorization,
Proxy-Authorization, Cookie, Set-Cookie, boot.key and login-code labels. Explicit
owner-selected literal secrets can supplement those rules; their bytes are
ephemeral inputs and never become metadata or result fields. These rules do not
identify all private application data, so a redacted export still needs review
before sharing. No export is published by this library.

Matching crosses original frame boundaries within each stream. Interleaved
stdout/stderr retain their own context and their original frame timing/order.
Every rendered or omitted byte range carries its original frame offset/length;
omissions explain their reason. Typed gaps/exits remain separate evidence. Without
independently verified line-start context, a leading partial line is withheld;
every trailing partial line is withheld. A gap invalidates earlier context.
Page edges cannot silently leak a split credential fragment. Stream EOF and an
omitted fragment are never represented as a quiet or successfully exited ship.

Text uses escaped literal plain-text representation, including terminal controls,
invalid UTF-8 and non-ASCII formatting controls. Consumers must render it as plain
text. Input, secret count/size and transformation spans have explicit bounds;
excess refuses with no partial export. Overlapping/repeated secret matching uses
linear scanning. Source race tests cover all frame split positions, stream
interleaving, gaps, context loss, long traces, control containment, unchanged
originals and output bounds. The recording format is unchanged. Runtime capture,
operator/target authorization, retained-prefix selection, complete bundles,
redaction preview/API/UI and native/installed qualification remain pending.

## Unwired private Docker attachment primitive

The prepared transport requests only output from one full container ID on the
already selected owner-private Unix socket. It verifies socket ownership/layout,
unchanged socket identity and the actual Unix peer UID. It accepts only the
explicit101/TCP upgrade with non-TTY multiplex content type for pinned API1.47;
the [primary implementation](https://github.com/moby/moby/blob/v27.5.1/api/server/router/container/container_routes.go#L847)
distinguishes that format from raw TTY output. There are no proxy, redirect,
stdin, log-replay, signal, Start or Stop options. Consumers receive only a reader.
Headers/dialing are bounded; the lifetime belongs to the future independent
recorder rather than a UI/SSH request. Explicit close/cancellation interrupts a
blocked read by closing only that attachment connection.

Inert private Unix-peer race tests preserve multiplexed bursts/long traces,
diagnostic atoms, fake credential text, controls and arbitrary bytes. They verify
the buffered first bytes, no writable console surface, no extra request on close,
blocked-read cancellation, wrong framing/upgrade/headers/redirect refusal and
invalid ID/socket layout refusal before connection. These are source transport
gates, not Docker, Vere or recorder-service lifecycle evidence.

The primitive is unexported and not called by runtime lifecycle code. A trusted
factory still must establish exact daemon/container/TTY/launch ownership before
and after attachment, persist admission and complete metadata, and coordinate
capture before Start. Peer UID alone establishes none of those facts. Persistent
writer/cursors/retention integration, independent worker lifetime, process/exit
and missing-coverage observations and actual disposable Docker/Vere qualification
remain required. No installed logging behavior changes.

## Source-only linked segments and recorder continuation

Metadata3 explicitly selects a segmented layout. Metadata1/2 retain their exact
canonical bytes and single-segment read/cursor contracts; their writers do not
acquire segmented continuation. An older reader must refuse3 before presenting
an incomplete first-segment view. The original binary frame/header format stays
at1. Frame results now include segment number and receive-clock epoch origin,
derived from the verified header, so a restart cannot silently reset elapsed time.

Size rotation and explicit coordinator-scheduled rotation seal the old file and
fsync a uniquely numbered successor header/directory before accepting new output.
Each header binds exact metadata, next sequence and previous verified digest.
Rotation in the same capture retains its clock epoch and inserts no invented
transport gap. The default segment bound is16MiB, with an explicit smaller bound
allowed; all original encoded segment/header/partial bytes count toward the
session ceiling. At most1024segments are admitted. Metadata and filesystem
allocation overhead require separate host retention/reserve accounting.

Continuation requires exact saved metadata, an exclusive original writer lock,
verified bounded prefix and unchanged original file identities/sizes at admission.
It never appends to, truncates, replaces or repairs an old segment. A new clock
epoch begins with an fsynced `recorder-restarted` gap with unknown missing bytes.
A torn final frame instead requires `corrupt-tail`; crossing that preserved tail
without the matching gap refuses, including across an empty successor header.
Corrupt complete frames, missing/reordered segments, incomplete headers, changed
bindings and an already observed terminal process refuse. An observed exit seals
a metadata3 writer. Transport EOF alone still establishes no process outcome.

The caller must independently revalidate the same runtime/host attachment before
continuation. Metadata equality and a released writer lock are not runtime
authority. Changed host/attachment identity needs a distinct linked session;
this library does not reconnect a transport or update saved identity. A failed
or pressure-refused write still needs the future capture coordinator's incident
and recovery-gap handling. There is no automatic pruning, ship signal or resume.

Source race tests cover exact paging/reconnect across size/explicit rotation,
concurrent rotation/readers, real owned recorder-process death, changed binding,
live writer, quota/reserve, file replacement, torn-tail preservation and gaps,
wall-clock reversal/new epochs, terminal refusal, links and segment corruption.
A quota test's first bound did not force a new header; its corrected bound proves
header-inclusive admission. The original2 writer/reader rehearsal is prepared in
`scripts/test-recording-segment-upgrade.py`; actual compatibility remains pending
until that exact clean-source gate runs. Indexed seeking, host/ship retention,
independent capture worker, real Docker/Vere, API/UI and installed qualification
remain required. No installed recording or runner schema changes here.

The [actual original2 writer/reader rehearsal](diagnostics/2026-11-18-recording-segments3-source-upgrade.json)
now passes for frozen2b5af45. Repeated new-reader reopen/paging preserve every
original2 file byte, legacy continuation refuses unchanged, and the actual old
reader refuses the rotated/continued metadata3 store without changing it. A
retained original-writer fixture and source/helper pins keep that boundary in
focused tests. This supersedes the preceding compatibility deferral for these
exact source artifacts; runtime/native/installed gates remain unqualified.

## Source capture coordinator

`Store.CaptureDocker` composes current-version session creation, an already trusted
output-only attachment factory, durable-readiness acknowledgement, multiplexed
raw-byte decoding and independent exit observation. Metadata syncs before attach;
readiness follows successful attachment. Its single receive clock retains a Go
monotonic origin while wall time is recorded independently. Bounded original
chunks, including long traces, controls and credentials, sync through existing
linked segments using versioned metadata and the existing frame/segment formats.

EOF with no independently verified matching exit produces a gap. Interrupted
transport can retain its received prefix, an explicit gap and a separately
verified exit. Cancellation closes only this output connection and records lost
coverage; it establishes no process exit. Read/stream completion detaches before
storage diagnostics or bounded exit observation, preventing an unread connection
from applying backpressure while diagnosis runs. Neither this coordinator nor its
transport sends a lifecycle action, signal, console input or replay request.

Storage refusal is visible even when no gap can be written: the returned finite
reason and `gap_saved=false` retain that distinction. A successfully closed file
is not proof of complete coverage. Raw diagnostics and original payloads are not
copied into status. Existing/incomplete sessions are never reused automatically;
reconnection, pressure recovery and retention require separate admission.

Full recording race tests and focused vet pass with inert streams/private files,
covering durable readiness, exact raw bytes across rotation, wall-clock reversal,
metadata-pointer isolation, interrupted payloads, EOF without exit, wrong-process
exit, failed attach/readiness, quota/reserve/write refusal and cancellation of a
blocked read. The existing actual-original1/2 fixtures remain readable unchanged.
These source gates do not observe a real engine/process. A trusted factory,
independent supervised worker, launch/start-failure integration, retention/pins,
indexed seek, API/UI/export and real Docker/Vere/installed qualification remain
required. Recording support is not advertised by the runner.

## Prepared metadata4 bootstrap boundary — unqualified

A first launch can precede both comet identity and a pier/.urb object. The current
working model labels pending-comet-mining with an empty ship name and distinguishes
an existing-pier fingerprint from prepared-storage intent. Registered means an
expected known identity, not a successful native response. These immutable labels
never acquire future names/inodes through opening, continuation or exit. Later
observed identity requires separate linked evidence and trusted runtime factories.

Metadata1–3 retains exact original bytes and omitted labels; current4 uses the
same segmented frame codec1. Full recording race tests and focused vet now pass.
The [actual original3 writer rehearsal](diagnostics/2026-11-18-recording-metadata4-source-upgrade.json)
passes for frozen0b28bac: old segmented bytes/cursors/reopen stay exact, and the
actual old reader refuses populated4 unchanged. A retained original3 fixture with
source/helper/writer pins runs in ordinary tests. No installed recording,
bootstrap coverage or native identity observation is qualified by this model.

## Ordinary native factory — native qualification pending

`PrepareManagedRecording` is prepared source for a newly created ordinary Vere
container. The authenticated manager still owns the original job, actor, public
target/revision and lifecycle fence. The factory verifies the private account and
daemon, full selected container, non-TTY launch/containment, network and unchanged
exclusive storage. It refuses an absent or already started container. A pending
comet keeps its name unknown; empty prepared storage is checked without creating
a pier. Only digests of private paths/launch configuration enter metadata.

Before and after attachment, and immediately before readiness, recheck the same
created-container snapshot. Replaced storage, a newly appearing pier, changed
boot/daemon, caller metadata or launch settings refuse and detach. Independently
read terminal evidence twice on stream completion; a missing process timestamp,
changed launch/exit or running container refuses. Preserve nonzero/OOM/error flags
and wall-clock reversal without copying daemon diagnostics. This code sends no
Start, Stop, signal, wait, stdin or guest command.

The source/race tests pass with inert engine fixtures and private temporary files,
including exact output capture, independent terminal evidence, caller-pointer
isolation, artifact/launch/storage drift and ambiguous selection refusal. Focused
vet passes. No factory is wired into installed launch. Independent supervised
workers, project factories, later native identity evidence, launch-failure and
pressure recovery, retention, API/UI and real-runtime gates remain required.

## Source-only worker handoff model

The version1 private job codec binds the original ordinary runner operation,
actor, revision, prepared runtime and exact executable hash to its native
admission. It copies only required launch inputs. Signing keys, retained log
bodies, recorded endpoints and runtime diagnostics are omitted; source quarantine
cannot be stripped into a clean job. Unknown versions/fields, duplicate or
noncanonical JSON, invalid bounds and sensitive extra state refuse. Nested values
are copied and full uint64 fields remain exact.

An independent worker must repeat native admission and match engine/endpoint,
host boot, target, expected-name/storage basis, launch and runtime-artifact facts.
Its new session clock and its own recorder namespaces must be independently
obtained. A matching hash is evidence association, not authentication, an engine
probe or Start permission. The authenticated manager retains the durable job and
lifecycle guard. Source/race/vet checks pass with inert engine/private fixtures.
The source claim/readiness layer is described below. Actual executable checking,
independent supervision, launch integration and upgrade guards remain required;
this codec alone starts nothing and changes no installed file or database.

## Source-only durable job handoff files

An explicitly authorized0700 job root now supports bounded, immutable0600 job,
claim and readiness files. Cross-process admission locks enforce an explicit job
count ceiling; free-space reserve refusal creates no job. Job bytes and parent/
child directory entries sync before acknowledgement. A worker lock serializes a
single claim, which syncs before further native admission. A closed/crashed claim
refuses replay; a partial original is preserved without repair. Supplied worker
executable/boot/namespace observations must match the job and actual attachment
metadata. Actual self-executable observation remains the future worker's duty.

Readiness binds the exact job, session and canonical metadata hash. Replaced,
linked, exposed, malformed or rebound files/directories refuse. Repeated readiness
does not overwrite its original. Lookups contact no engine, mutate no file and
remain available after writer closure. A saved ready receipt does not prove a
currently live worker or authorize Start; supervision, fresh native admission
and the manager's lifecycle fence remain separate mandatory checks.

Source/race/vet checks pass with private temporary files and inert native seams,
including independent handles, claim/reclaim, concurrent quotas, disk reserve and
metadata/ownership changes. This is not process-death/power-loss qualification.
No default job root, service, launch integration or installed database is enabled.

## Segment-sized capture reads

The capture coordinator now selects transport read chunks that fit a complete
frame plus header in the configured segment. A long output frame therefore
rotates through small segments without forcing an avoidable limit failure.
Append callbacks still acknowledge a whole received chunk or fail; no callback
splits a prefix and then overstates the number of unrecorded bytes. The public
transport decoder keeps its existing64KiB default and frame/metadata formats are
unchanged. Full source/race checks preserve long binary/multiline output through
4KiB segments and verify exact known received/unrecorded counts under pressure.
The original1/2/3 byte fixtures stay unchanged. Native capture remains pending.

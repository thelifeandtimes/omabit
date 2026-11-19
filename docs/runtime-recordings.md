# Runtime recording development contract

Issue20's recorder is in development. Current installed log behavior is unchanged.
The initial [inert lifecycle comparison](diagnostics/2026-11-17-recording-backend-inert.json)
supports proceeding with a runner-owned stream recorder outside the container.
This choice remains provisional until actual Docker/Vere lifecycle, interruption,
retention and pressure gates pass. The existing bounded sanitized tail cannot be
treated as a comprehensive original recording.

The corrected [native schema64→65 active-worker case](diagnostics/2026-11-19-recording-policy-native-upgrade.json)
passes process/history preservation, independent original-byte API comparison,
policy replay/lookup/pin, immutable old/new limits, two same-pier clean Stops,
unchanged genuine old-reader refusal and final reopen. Its runtime artifacts are
frozen at703de9f2. Earlier retries exposed an original-window schema mismatch and
a fixture SQLite reader leak; their failures remain recorded separately. The
shared schema now matches JSON base64 encoding without changing persisted bytes.
This is not the stable59→65 or preserved DEV upgrade, or qualification of the
remaining retention/pressure/export/factory features below.

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
The source claim/readiness and worker layers are described below. Independent
supervision, launch integration and upgrade guards remain required;
this codec alone starts nothing and changes no installed file or database.

## Source-only durable job handoff files

An explicitly authorized0700 job root now supports bounded, immutable0600 job,
claim and readiness files. Cross-process admission locks enforce an explicit job
count ceiling; free-space reserve refusal creates no job. Job bytes and parent/
child directory entries sync before acknowledgement. A worker lock serializes a
single claim, which syncs before further native admission. A closed/crashed claim
refuses replay; a partial original is preserved without repair. Supplied worker
executable/boot/namespace observations must match the job and actual attachment
metadata. The source worker now observes its actual self image and context.

Readiness binds the exact job, session and canonical metadata hash. Replaced,
linked, exposed, malformed or rebound files/directories refuse. Repeated readiness
does not overwrite its original. Lookups contact no engine, mutate no file and
remain available after writer closure. A saved ready receipt does not prove a
currently live worker or authorize Start. A complete file may be visible before
its writer's fsync call returns: require a separate post-sync acknowledgement.
Supervision, fresh native admission
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

## Source worker composition and acknowledgement

The unwired worker function verifies the installed-account boundary, hashes the
kernel-selected self executable, observes own host/namespace/process-start facts,
claims its exact private job and repeats native pre-Start admission. It persists
capture metadata/header and readiness, then calls a separate trusted supervisor
acknowledgement only after sync returns. A readable marker with failed
acknowledgement remains an incomplete capture. Neither file visibility nor a
queued service invocation can authorize Start.

Claim2 binds PID/start ticks, host boot, namespaces, executable and service
invocation when present. Legacy claim1 lookups leave start/invocation unknown and
cannot establish a live worker. Synthetic legacy-shape checks pass; actual old
claim-writer compatibility still needs qualification. Immutable version1 result
files contain finite status, byte counts and gap/exit/durability flags, with no
raw output or private exception text. Lookups/repeated outcomes preserve originals.

For the upcoming service, systemd's [notify contract](https://www.freedesktop.org/software/systemd/man/latest/sd_notify.html)
and [service types](https://www.freedesktop.org/software/systemd/man/latest/systemd.service.html)
require the main worker to acknowledge readiness through Type=notify, followed
by exact live-worker checks in the manager. Parent/client lifetime must not own
the capture service. The inspected host uses systemd262; no service is installed
or started by this source checkpoint.

Source/race/vet checks cover raw bootstrap/exit composition, actual self identity,
changed job/native bindings, failed acknowledgement, EOF without terminal proof,
storage refusal, blocked-read cancellation, immutable finite results and legacy
unknowns. Cancellation closes only the output connection. This is not independent
process/service, Docker/Vere, crash/power-loss or installed qualification. Fixed
configuration/CLI, supervision and authenticated launch/upgrade integration remain.

## Prepared notify supervisor primitive

Source support now sends only fixed READY=1 to a bounded local supervisor socket
with a valid invocation identity. No raw status, output or secret is sent. Unit
names bind runner/mode/root/job. Exact bounded property parsing requires notify,
NotifyAccess=main, active/running, full unit identity, invocation and cgroup;
current process/image/boot/namespaces/start ticks must match the original claim.
These comparisons do not perform a probe or grant launch authority.

Focused source/race/vet checks pass. A gated inert test helper is prepared to
verify actual acknowledgement and survival after its launcher returns, using
only its own process and fresh private files. Actual service, native worker and
installed launch qualification remain pending; no existing unit is touched.

The [fresh inert service gate](diagnostics/2026-11-18-recording-notify-inert-service.json)
now passes for frozen source fb355f8. Its exact main process acknowledged
readiness, remained active after the launcher returned, wrote a continuation
proof, and ended successfully. The launcher compared PID, start ticks, executable,
invocation and cgroup; source and binary stayed unchanged. This supersedes the
pending inert-helper gate above only. It does not qualify a recording worker,
native capture, authenticated launch or active-job upgrades.

The source observer now runs a fixed bounded property read for only the selected
unit, reads only its exact main PID's kernel identity/image/cgroup, and requires
the unit observation to match afterward. It refuses changed process start,
image, owner, namespaces, cgroup or invocation. It reads no process arguments or
environment. The startup main-process check is separate from ready authority.
Focused race checks use inert supervisor seams and actual self-only proc; actual
worker-service/native/installed gates remain pending.

The internal `recording-worker --job ID --sha256 DIGEST` entry now derives
`recording-jobs` and `recordings` only below the verified owner's selected
`.local/share/urbit-runner` home. It requires existing private directories and
real starting notify/main supervision before consuming a job claim. The separate
development-root command boundary remains explicit. No arbitrary paths or native
inputs are accepted; ordinary unsupervised invocation refuses. Source/race gates
pass, but normal launch still creates/submits no jobs. The database downgrade
guard and authenticated journal binding must precede that integration.

`scripts/test-recording-claim-upgrade.py` prepares the actual original claim1
writer/current claim2 compatibility rehearsal using unchanged frozen85ac7c6
runtime and test sources plus a declared synthetic filesystem helper. It requires
`--run`, a clean candidate, fresh task output and storage admission. Synthetic
PID/namespaces are not live authority. Actual old-writer results remain pending.

The [actual claim1/current claim2 gate](diagnostics/2026-11-18-recording-claim2-source-upgrade.json)
now passes for candidate252aa548. Original job/claim/readiness/lock bytes survive
repeated lookup; old omitted identity facts remain unknown and the original claim
cannot be reused. The genuine old reader refuses claim2 unchanged. A compact
five-file fixture, unchanged source manifest and declared helper run in ordinary
tests. This closes only the source-format gate, not independent live-worker,
database/installed upgrade, power-loss or native recording qualification.

## Prepared launch journal guard

Schema64 adds private immutable recording intents and a separate bounded count/
digest head. Each binds the original running Start actor/request, target/revision,
exact runtime state and created container to the job/executable/admission digests.
Admission and head advancement commit together. Exact replay returns historical
evidence; it cannot authorize another worker submission. Opening validates ancestry
and detects missing tails without claiming, submitting or replaying a job.

Focused source/race SQLite gates pass for stale authority/state, concurrent jobs,
head-update rollback, process exit after commit and corruption. Original59–62 and
synthetic11–63 migration preservation pass. Actual frozen63 writer/reader refusal,
installed/native upgrade and authenticated normal-launch integration remain
required. Normal launch still creates/submits no recording jobs.

`scripts/test-recording-store-upgrade.py` prepares the actual frozen63 writer /
populated64 reader-refusal rehearsal. It preserves earlier Stop/app uncertainty
and linked recovery ancestry, then tests one synthetic pre-Start recording intent.
Fresh output, clean source, explicit `--run` and storage admission are required.
Its actual result is pending; no job service or native runtime is submitted.

The [actual63→64 source gate](diagnostics/2026-11-18-recording-schema64-source-upgrade.json)
now passes for frozen6fd35d6d. Original rows/DDL, earlier uncertainty and linked
audits remain exact; one new intent/head replays unchanged, and the genuine63
reader refuses populated64 without changing it. The compact actual63 SQL fixture
also passes ordinary race checks for preservation and rollback on missing original
ancestry. This closes the source database gate only. It does not qualify normal
worker submission, active-worker/native continuity or installed upgrade.

## Prepared executable retention

Source preparation now copies only the running process's kernel-selected image
to an existing private `recording-images` cache below the selected owner's state
home. Exact hash-addressed complete images become0500 after sync; verified reuse
does not replace original bytes. Partial, changed, exposed or linked originals
refuse without repair. Eight images/128MiB each, exclusive admission and a64MiB
space reserve bound this cache; no automatic image deletion is implemented.

Focused race/vet checks use inert bytes and launch nothing. This preparation is
not invoked by normal managed launch and does not qualify active-worker upgrade
continuity. The independent service handoff and native capture gates remain.

## Prepared detached service command

The source command uses only the exact retained image/job digest and explicitly
forwards the verified DEV selection. It requests notify/main with no automatic
restart, terminal or caller-owned scope, null standard streams and fixed worker
resource limits. Literal arguments use `--expand-environment=no`; the
[systemd launcher documentation](https://www.freedesktop.org/software/systemd/man/latest/systemd-run.html)
requires254 or later for that option. Unsupported launchers refuse. A lost
submission acknowledgement is uncertain and must lead to exact job/unit lookup,
never automatic resubmission.

Argument/namespace race checks and vet pass without executing a launcher. Normal
launch remains unwired. The coordinator must compose durable job/journal checks,
post-sync readiness and exact live/native fences before Start; actual service and
native/installed upgrade gates remain pending.

## Prepared readiness coordinator

Source composition now requires an exact durable unclaimed job and fresh trusted
launch fence before one submission. After notify acknowledgement it binds two
matching live-worker observations, immutable saved readiness/claim, an open
empty recording with exact metadata, and fresh native created-container admission.
Changed files, saved results, partial/output frames, closed writers, stale unit/
native identity or final fence refusal cannot authorize Start. The coordinator
sends no container lifecycle command; lost acknowledgement remains uncertain and
is never retried. Historical receipt replay cannot invoke this coordinator.

Focused race tests pass with inert submit/supervisor/native seams and real private
files. No launcher is executed and normal managed launch remains unwired. Actual
service/native and installed/active-worker upgrade gates remain required.

The store now supplies the coordinator's fresh read-only fence: it checks the
exact saved receipt/head, current original operation and exact runtime/configuration
generation. It refuses extra live containers and uncertain creation history.
Historical replay remains available after interruption and cannot grant this
authority. Focused SQLite/fixture race gates pass without changing original rows;
normal launch still does not invoke the coordinator.

## Prepared ordinary lifecycle hooks

An opt-in driver variant now requires both recording preparation and terminal
removal callbacks. Preparation follows durable native creation/stage save; a
fresh bounded guard runs immediately before Start. Terminal state/log evidence
is saved before a bounded pre-removal barrier. Refusal keeps the stopped container
and pier, and a subsequent Stop can finish removal without repeating its signal.
Callbacks receive independent snapshots and do not replace native ownership or
containment checks.

Admitted source race checks and vet pass with inert native effects, including
failed saves/handoffs/guards, snapshot isolation and the one-signal continuation.
The normal server does not select these hooks. Actual recorder preparation,
terminal drain/outcome handling and native service/upgrade integration remain;
projects, utilities and existing-running attachments need separate factories.

The explicit root provisioner now binds setup to the verified selected manager
directory and creates only the three private recording children. It preserves
existing content and refuses unsafe objects without repair/replacement. Source
race/vet checks pass with fresh private filesystem fixtures. Normal server
construction does not call this provisioner yet.

## Session allocation before attachment

The worker now durably links the original job/claim to a fresh session ID before
opening capture storage. This preserves lookup through a failed opening or
attachment without pretending a header or capture exists. Allocation has its own
observed time; it is not a process-start or capture epoch. Current live readiness
requires the unchanged link. Existing readiness/results validate it when present;
older jobs without a sidecar remain unknown and unchanged.

Source race checks pass for allocation-only lookup, storage failure, duplicate/
malformed/rebound/linked files, changed claims, late changes and unchanged actual
claim1 bytes. This is source/filesystem qualification only. Read-only status/drain,
normal manager integration and native/installed worker gates remain pending.

## Saved status and terminal-drain observation

Source status now distinguishes saved job/claim/allocation/readiness/result facts,
header availability and the writer lock. It returns no raw frames or exceptions.
The observed prefix does not certify full raw integrity or current process health.
Missing/moved headers remain observed absence with the original receipt retained;
legacy missing allocations remain unknown. Storage failures remain inspectable.

Bounded drain polling observes a closed saved finished capture/exit receipt with
an available header. Incomplete, missing and timed-out evidence remain distinct.
This grants no native removal permission and sends no signal or lifecycle action;
the caller still verifies its exact terminal/native and journal bindings. Source
race/vet checks pass with inert worker/native seams. Normal manager, native,
retention, public API/UI and installed-worker upgrade gates remain pending.

## Prepared manager adapter

The source manager adapter now derives capture inputs from its actual running
operation and exact saved snapshot, commits the private job/intent before one
submission, and returns a fresh journal/readiness/live-worker guard. Cleanup finds
the exact historical capture even across a management-mode promotion, rechecks
current state after drain, and keeps the original failure/gap for a durably closed
incomplete capture. Pending/unclosed/remapped evidence cannot permit cleanup;
older unrecorded containers are not adopted into new capture history.

Source race/vet checks use SQLite/private files and inert adapters, launching no
binary or runtime. The adapter is not selected by normal server construction.
Its bounded pilot has sixteen64MiB sessions plus metadata, eight retained images
and a2GiB free-space floor, with no automatic pruning. Configurable retention,
pressure recovery, actual service/native capture and interface/upgrade gates
remain required before comprehensive recording readiness.

M694 adds the explicit `serve-development --record-runtime` pilot selection. It
configures before background workers, uses the actual running operation and
advertises only `runtime-recording-pilot-v1`. Default startup remains unrecorded.
The pilot bounds sixteen64MiB sessions and leaves2GiB free, with no automatic
pruning; running-container adoption, projects and maintenance are excluded.
Strict original-job/native created-state proof permits never-started cleanup
without fabricating an exit. Otherwise the existing drain barrier applies.
Focused synthetic manager/server/CLI race gates and vet pass. Actual native
service/capture, retention, interfaces and installed/active-worker upgrades are
still required. This is not qualified comprehensive recording support.

The [M697 installed native pilot](diagnostics/2026-11-18-recording-pilot-native.json)
now passes pre-start ordinary capture, exact independent notify-service lifetime
through manager stop/reopen, terminal drain before ephemeral removal, and two
same-pier launches with unchanged first-session bytes. Each retains one observed
exit and no gap. Frozen2d028197 artifacts remain unchanged and both test units end.
This supersedes only the ordinary service/lifecycle deferral above. Real burst,
pressure/recovery, host restart, active-worker binary upgrade, other factories,
retention and public interfaces remain separate requirements. No preserved DEV
or production deployment was changed.

## Operator-private saved recording reads

The development shared operations `runtime.recordings.v1` and
`runtime.recording-page.v1` expose saved history and bounded windows through the
same `omabit call` and stdio MCP handlers. They require operator role, an allowed
selected runner advertising `runtime-recording-read-v1`, and explicit
`include_private_data=true`. Catalog entries are immutable launch intents; an
entry does not prove that capture started or that its bytes remain available.
History remains readable after the original operation is interrupted or its
instance/configuration changes. No read claims/submits a worker, opens an engine
stream, signals a runtime, repairs data or changes retention.

```sh
omabit --config-dir REGISTRY call runtime.recordings.v1 --input-json \
  '{"name":"DEV","runner_id":"RUNNER_ID","instance_id":"INSTANCE","include_private_data":true}'
```

Use the catalog's exact `job_id`/`job_sha256` with
`runtime.recording-page.v1`. Explicitly select `representation="redacted"` or
`"original"`; neither is inferred. A private JSON input file can retain the
selection and the exact returned cursor. Catalog windows contain at most32
entries. Frame windows contain at most64frames/128KiB original payload; encoded
replies must fit the existing1MiB local/SSH bound. A view that cannot fit refuses
without silently truncating a frame or changing representation.

Derived redacted windows preserve original timing/digests and account for omitted
ranges. `escaped_text` is an ASCII escaped literal for plain-text rendering; do
not interpret it as HTML, terminal commands or executable escape sequences.
Leading context after a cursor/gap and incomplete trailing lines may be withheld.
Finite credential-field rules preserve diagnostic atoms such as `0w...`; they
cannot identify every private application value. Owner review is still required
before sharing. This transformation never edits stored originals.

Original windows return unmodified octets only as explicit base64 fields. They
remain private operator data. Every response marks `complete_export=false`:
this interface does not package a complete session export. Keep original receipts
and partial/gap evidence. `snapshot-eof` describes only this read's prefix; it is
not a process exit or completeness claim. Polling the exact cursor follows saved
bytes without attaching again. A foreign/missing cursor refuses rather than
rewinding. Saved result, writer-lock and header-availability facts are separate;
a missing header never establishes that its bytes never existed or are disposable.

Source gates pass byte-exact multi-window output/exit reads, derived redaction,
historical SQLite/HTTP access without row changes, changed/private/refused
selections and actual CLI/official-SDK stdio parity, including revisions above
2^53. These read APIs are newer than M697's frozen native pilot. Matching native
API/installed upgrade, indexed search, retention/pins, complete export packaging
and UI qualification remain pending.

M700 prepares an independently audited retention-control model and schema65
storage. Policy and incident-pin receipts use exact head/revision CAS and original
actor/request replay; they change no existing job/recording/native state. The
settings include bounded host/per-instance bytes/age and job/session/segment/
reserve limits. Opening writes no policy event and retires nothing. These controls
are not yet wired to manager enforcement, retirement or UI. Source controls and
original63 gates pass; actual64 writer/refusal, broader/native/installed and
active-worker gates remain pending. The source-only format support is not a
production upgrade or authorization to prune existing user evidence.


## Recording controls and admission commitments

M702 wires the saved schema65 controls into the explicit development pilot. The
operator-private shared operations below use the same CLI and MCP handlers:

- `runtime.recording-controls.v1`: read settings, exact control head, pins and
  conservative capacity commitments, with `include_private_data=true`.
- `runtime.recording-control-apply.v1`: configure bounded limits or pin/unpin an
  exact catalog job/digest, using an original request ID and prior revision/digest.
- `runtime.recording-control-lookup.v1`: read that authenticated actor's original
  request receipt after a lost response. Lookup never writes or resubmits.

MCP writes require `--enable-mutations`; enrolled target allowlists, selected
capability and operator role still apply. Recording limits do not grant application
permissions. Automatic retirement requests are refused. Age settings are retained
for later retirement review; changing them currently deletes nothing.

The default remains16jobs,64MiB per session,8MiB segments,1GiB host and256MiB per
instance, with a2GiB free-space floor. Explicit per-instance overrides are bounded
by the host maximum. Each retained job charges its full immutable session maximum,
including closed, unclaimed and uncertain jobs. This is a conservative commitment,
not measured disk usage. Metadata, executable caches and unrelated filesystem use
remain separate. Free bytes are the minimum observed on both recording roots;
this read is not an exclusive reservation against other filesystem users.

Before native Apply/create, the pilot checks saved controls and all job/session
attribution. It checks again before job creation, freezes the selected limits in
the new job, and binds intent2 to the exact control revision/digest through the
final Start guard. Existing intent1 bytes remain unchanged and historical replay
never gains fresh launch authority. The single management work gate serializes
launches; new control writes wait while management workers are active, while
lookup and exact saved replay remain usable. Existing capture workers keep their
original limits after settings change or manager reopen.

Unknown, partial or foreign jobs and unattributed session directories refuse new
admission. Saved controls still remain readable with `capacity_status="unknown"`.
A failed capacity check does not authorize deletion, adoption, restart or data
repair. Pins are retained as exact audited protection selections; all data remains
retained until the separate retirement implementation is qualified.

Source tests cover host/instance/job/free-space limits, unknown storage, no native
dispatch on preflight refusal, changed-policy final guards, original-history
reopen/replay, authenticated actor lookup, client reply binding and actual CLI/
SDK stdio parity/lost-write behavior. Matching native APIs, active-worker binary
upgrade, measured usage, retirement, pressure recovery and UI remain pending.


## Source-only recording view

M707 adds a monitor **Recordings** tab for saved history, bounded windows and
read-only retention settings. Opening it makes no recording request. Enable the
private-read choice for the current selection, then request history and select
an exact saved job. Redacted windows and base64 originals have separate explicit
buttons. Derived text stays literal plain text, including markup/control escape
spellings. Receive times and stream/frame identity remain visible; unknown exit,
writer/storage facts, omitted spans, gaps and snapshot EOF stay distinct. Every
window states that it is not a complete export.

The view clears private data/consent when profile, transport, target, role or
admission changes. Late replies cannot cross those bindings; a foreign reply
cannot replace accepted evidence. Revoking consent clears displayed data and
cancels only the UI reader. Reads use the same typed `omabit call` operations;
no shell/Dojo/SQL/Docker channel, live attachment or runtime signal is exposed.
Settings show immutable maximum commitments and explicit unknown capacity. This
initial view does not author policy/pins, retire data or produce a full export.

An inert private offscreen check passes seven explicit reads covering consent,
representation, plain-text markup/escape rendering, unknown capacity, foreign
replies, late profile responses and revocation; the QML registry check passes.
The [synthetic view](diagnostics/2026-11-18-recording-ui-source.png) is visually
inspected. An initial fixture accidentally changed its transport identity when
changing simulated server behavior; correct that fixture before the passing runs.
This is source UI qualification, with no installed plugin or preserved DEV change.
Matching native API/upgrade and final desktop qualification remain required.

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

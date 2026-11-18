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

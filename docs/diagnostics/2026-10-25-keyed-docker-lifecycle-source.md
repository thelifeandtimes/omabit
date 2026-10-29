# Keyed Docker lifecycle source boundary

The Docker private-chain fixture now has an explicit guest-only lifecycle helper.
It accepts only fresh fixture zod/nec identities, a full recorded container ID,
a pinned official image and one labeled network-none anchor. Boot mounts a fresh
key and pinned pill read-only; reopen omits creation flags and preserves the named
`private-<ship>/<ship>` pier. Exact inspected commands, mounts, isolation and
resource limits must match before lifecycle effects.

First keyed shutdown uses the qualified native Lens exit convention; reopened
ships receive SIGTERM only. A durable fixture receipt precedes dispatch. Success
requires both acknowledgment and observed exit0 without OOM. Unknown replies
remain uncertain and do not resend; completed retries only observe. The helper
never invokes Docker stop's forced-kill fallback, removes containers, or scans
other ships. The only permitted engine is the disposable guest's rootless socket.

14 inert tests pass in a private namespace, covering command/identity boundaries,
wrong mounts/network/resources, host refusal before effects, dispatch persistence,
lost replies and false clean-stop observations. The initial sandbox could not
create a network namespace; a later count assertion expected15 instead of the
actual14 tests and stopped before execution. Both are retained, followed by the
corrected14-case pass. No actual Docker command or ship ran during these tests.

M338 changes the immediate work order after M337: advance the independent private
Docker-chain gate while profile claimed/unpublished and pending/unbound loss remain
tracked. The lifecycle helper is qualification source, not runner integration or a
Docker runtime pass. Next stage a chain-backed gateway in a fresh no-NIC VM and
run two newly keyed containers through messages, same-pier replacement and saved
application/subscription recovery. Do not reuse a native test pier or key input.

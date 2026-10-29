# Independent-session real-runtime checkpoint — 2026-10-21

The independent-session mechanism passed on a **fresh fake `~zod`** inside a
private Bubblewrap namespace. It used no existing pier, runner, Docker engine,
keyring, production credentials or desktop service. The fake shut down cleanly.

## Provenance and containment

- Omabit baseline `b233128`, topic `codex/platform-overhaul-plan`, with uncommitted
  source recorded by hash in [the receipt](2026-10-21-isolated-auth.json).
- Copied native Vere reports `urbit 4.6`; SHA-256
  `47ad302e8934271dfc00dcdab1d553c6d6460b105e3099ff47e0e4e4231bb26d`.
  This is not an execution of Omabit's pinned Docker image.
- Repository-pinned pill: generation `1790171686269740`, 217421172 bytes, SHA-256
  `f8b08c53dd6d0aa9c4f7654f0a7bcd11e94089222888a3c99c0a10eedcbb377b`.
- Separate user/network/mount/PID/IPC/UTS namespaces. No `/home`, host `/run`,
  Docker socket, desktop socket, SSH agent or external route was mounted.
- Unique transient user scope, 4 GiB actual memory, no swap, CPU quota 100%,
  task limit 128; runtime low priority and bounded boot/test/shutdown deadlines.
- Runtime flags include `-F zod`, `-L`, `--no-conn`, `--no-dock`, `--loom 31`,
  local HTTP bind, and a new `/lab/zod`. No host port was published.

The harness refused direct `--inside` invocation outside the sandbox. The live
Go test separately verified a lab marker and network/PID namespaces different
from the host before sending any authentication request.

## Observed result

`TestIndependentSessionsIsolatedLive` passed in 0.38 seconds after boot:

1. Login to the fresh fake with its standard fake code.
2. Use the authenticated `/spider/base/json/code/ship` thread twice and log in
   with fresh clients, obtaining two distinct new sessions.
3. Both sessions perform direct `hood /kiln/pikes` scries and see `%base`.
4. Logout of the first session makes a separately restored copy of its cookie
   fail authentication at the server.
5. The original session and the second session remain authenticated.
6. Remaining owned sessions are logged out; Vere exits cleanly after SIGTERM.

The source's synthetic tests additionally cover restored clients after issuing
clients close, malformed/missing responses, redirects and reused-token rejection.
The live test deliberately used desk inventory, not actual Theme/DMs clients or
application mutations. Plugin reads/writes/SSE and hosted compatibility remain
separate gates.

## First-run finding and retained evidence

The first isolated attempt stopped during fresh boot because a 6 GiB virtual
address-space limit prevented LMDB's reservation. No auth test ran. Replacing that
limit with the dedicated actual-memory cgroup cap allowed the second fresh attempt
to pass. Both generated piers and logs were retained; no existing data was reset.

Execution directories on this host:

- `/tmp/omabit-auth-run-20261021-01`: failed fresh boot.
- `/tmp/omabit-auth-run-20261021-02`: successful checkpoint.

After this run the harness gained stricter output-path, copied-pill and source-drift
checks. The receipt identifies the exact executed harness bytes; it does not claim
those later guards were in the executed artifact. The test and auth primitive
were unchanged by those guard additions.

## Next gate

Prepare profile-bound private session receivers in isolated Theme and DMs source
worktrees. Exercise them with separately labeled plugin artifacts and private
HOME/XDG/keyring/desktop boundaries, then test their own direct application
transports and standalone login. Define durable handoff and cleanup ownership
before exposing approval in Omabit's UI.

The protected managed `~lismyl-lismyl`, all tmux ships, all `~/temp-ur` contents,
and current installed plugins/services remain excluded under
[the user's constraints](../experiment-isolation.md).

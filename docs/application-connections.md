# Application connections (unreleased)

Omabit helps an installed plugin authenticate, then the plugin uses its own Eyre
client directly. It can also sign in using its own URL/+code flow without Omabit.
Each approved plugin/profile gets a separate ordinary Eyre owner session. These
sessions are not restricted to a desk; the approval UI says this explicitly.

## Desktop flow

Open Setup → Application Connections, or choose **Connect via Omabit** in an
installed plugin. The latter opens the same pane with that plugin highlighted.
The ship selector combines observed running fleet ships with saved HTTP-only
ship enrollments. For a capable fleet ship, enable **Allowed** and choose
**Connect**. Permission applies to that plugin/profile/helper and exact ship
incarnation across its registered URLs. Omabit chooses and verifies a reachable
address from this desktop; a remote runner's localhost is never assumed local.
Manage addresses in **Setup → Ship URLs**.

The fleet path retrieves the selected ship's code privately from its runner and
creates one independent session. Neither code nor cookie passes through QML,
command arguments or public output. The plugin then talks directly to Eyre.
Older managed/hosted paths retain per-attempt **Approved** choices and can use an
existing parent sign-in to create their independent child session. A changed
helper requires reviewed permission; an update never widens an older approval.

Complete version2 helpers support automatic reconnection under standing permission.
While Omabit is active, the desktop checks one eligible connection per minute.
A replacement uses a fresh origin-bound session; the previous session stays linked
until its exact cleanup finishes. Healthy sessions remain unchanged. Known failed
children back off for five minutes, with at most three automatic failed attempts
per predecessor before review. Missing credentials, uncertain effects and identity
changes stay fenced. Explicit Disconnect cancels pending handoff and is not undone.
The collector/UI pathway passes private fixtures; real receiver advertisement and
assembled qualification remain pending in this unreleased branch.

Use the separate **Remotely Hosted Urbits** Setup pane to add a ship by name,
URL and expected identity, then sign in once with its code. Codes use private
stdin and are cleared from the field. No runner installation is required for
hosted ships. Plugins retain their direct URL/code flow when Omabit is absent.
Explicit HTTP origins can address a LAN, VPN or public host; HTTPS provides
transport encryption and should be used on untrusted paths.

**Disconnect** closes the selected child session, reconciles its receiver account,
and archives the completed attempt. A failed cleanup stays visible with an
explicit recovery action; refresh never retries a mutation. Newer standalone
logins, the parent Omabit login and other plugin sessions remain separate.
The archive retains the private replay receipt, so the approval request ID cannot
issue another session. Connected or uncertain attempts cannot be archived.

An expired parent login or other failure before the new login starts is recorded
as `closed / not-issued`. The user may sign in again and explicitly approve a new
attempt. Losing a response after starting login remains uncertain. No token means
Omabit cannot prove which remote session to revoke; it preserves that unresolved
attempt rather than issuing another one.

## Commands

All commands support global `--config-dir DIR` and `--format json|text`, placed
before the command. `app-managed-connect`, `app-fleet-connect` and `app-fleet-reconnect` require a
pinned `--runner NAME --expected-runner-id ID`. Maintenance derives its runner
from the exact durable attempt; ordinary lifecycle commands do not accept a
runner/socket override.

| Command | Effect |
| --- | --- |
| `app-plugins` | Read installed capability declarations, without executing helpers |
| `ships` | Read saved ship IDs, revisions, origins and stored-session status; not a health check |
| `app-connections` | Read active nonsecret attempts, current revision and `next_actions` |
| `app-connection REQUEST_ID` | Read one active or archived attempt without contacting a ship |
| `app-fleet-connect` | Choose a registered route and connect under a standing ship/helper grant |
| `app-fleet-reconnect` | Replace an exact predecessor when needed, using current permission and a complete helper |
| `app-fleet-health` | Check one connected fleet session without mutation |
| `app-fleet-maintain` | Perform one bounded health/reconnect or lineage recovery step |
| `app-managed-connect` | Verify a fleet selection, obtain/reuse its private login, and hand off one plugin session |
| `app-connect NAME` | Explicitly approve and attempt one independent handoff |
| `app-disconnect REQUEST_ID` | Revoke only this attempt's session |
| `app-recover REQUEST_ID` | Observe an unacknowledged replacement or resume cleanup; never retry issuance/delivery |
| `app-reconcile REQUEST_ID` | Reconcile the receiver after remote cleanup |
| `app-archive REQUEST_ID` | Archive a fully closed/reconciled attempt, retaining its replay receipt |

Use `COMMAND --help` for flags and authority. `app-connect` requires
`--expected-ship-id`, `--expected-ship-revision`, `--plugin-id`, `--profile`,
`--artifact-sha256`, a fresh 32-character lowercase hexadecimal `--request-id`,
and `--approve-owner-session`. IDs/digests come from the discovery commands.
Keep the request ID on uncertain results; list attempts before another action.
The three cleanup commands and `app-archive` require `--expected-revision`, checked under the
attempt lock. A stale action returns `selection_changed` without acting.

Mutation output contains `code`, a fixed public `message` on failure, and the
durable `connection` when available, including on nonzero exit. No cookie, login
code, credential-store reference or receiver stdout appears in public results.
The parent enrollment's chosen credential adapter remains in effect. Desktop
enrollment defaults to Secret Service; there is no automatic plaintext fallback.

## Installed helper declaration

Discovery is confined to `HOME/.config/omarchy/plugins/PLUGIN_ID`, matching
Omarchy's installation layout. It is not a catalog or an app store. Omart remains
the discovery destination. A directory without a declaration is simply omitted;
an invalid declaration is shown as unavailable. Symlink checkouts are not accepted
as immutable connection artifacts; stage a separately identified installation.

Each supported plugin supplies `application-connection.json`:

```json
{
  "version": 1,
  "runtime": "python3",
  "authority": "eyre-owner-session",
  "entry": "application_connection.py",
  "files": ["manifest.json", "application-connection.json", "application_connection.py"]
}
```

Declaration2 keeps the initial offer/cleanup protocol and adds complete bound
preparation, replacement and status observation. Discovery reports this as
`replacement:true`. The protocol must be qualified before declaring2; do not
change a declaration merely to bypass a capability check.

Add every Python/JSON dependency used by the helper to `files`; Theme and DMs
include their actual dependency inventories. The manifest and declaration must
be included. Optional `application-profile.json` is automatically included in the
digest and must agree with the manifest ID. No shell entry points, absolute paths,
traversal, duplicate/unknown declaration keys or unsupported runtime/authority are
accepted. Loading/discovery never runs helper code.

The approved generation is retained under the collector's
`application-receivers/DIGEST` before the approval is journaled. Cleanup uses that
generation even after an installed plugin update. The launcher checks its bytes
and runs a bounded private copy; this is code pinning, not a sandbox against
malicious code running as the same desktop user. See the
[private handoff contract](application-handoff-v1.md).

## Completed history

The issuer keeps at most 64 active attempts. Explicit archival publishes and syncs
`application-history/REQUEST_ID.json` before removing the active row. A crash may
leave both copies; retry preserves the receipt and finishes removal. The first
archival advances a legacy application journal to2. Fleet attempts require3;
explicit replacement requires4. New readers accept1–4, with archive1–3. Actual
previous writers must refuse newer markers without mutation. Replacement archives
preserve immutable ancestry, bounded to2048 reachable records; active attempts
remain bounded to64. Do not reset markers or discard history to clear a fence.
`app-connection` returns `archived: true` with no `next_actions` for receipts.

Each receiver also archives reconciled entries into private per-request receipts
before compacting its active list. Pending and accepted entries remain. It can
compact a full legacy completed journal on the next new approval. Its active
version-1 shape stays readable by older retained helpers that still need to finish
cleanup. New helpers reject archived request IDs; older helpers rely on the
upgraded issuer's approval fence and must not be used to bypass it.

Receipts contain no cookies or login codes. They are permanent replay evidence,
so their total disk usage grows with completed requests. Retained helper caches
are not pruned by archival. A later cache collector must coordinate reference
checks with approval publication and preserve every still-needed generation.

## Qualification and remaining work

The [CLI/offscreen fixture checkpoint](diagnostics/2026-10-21-approval-flow.md)
and the subsequent [real public-flow gate](diagnostics/2026-10-21-public-auth-flow.md)
pass. The latter runs the actual approval panel and collector with both declared
receivers and private Secret Service on a fresh fake, then verifies direct plugin
reads/SSE after the panel exits. An already running Theme service and DMs bridge
adopt their newly approved accounts without another sign-in.

Nothing is installed or enabled on the active desktop by these checkpoints. Full
native desktop/overlay behavior, reconnect/data writes, hosted compatibility,
retained-helper cache pruning, and stable/dev/edge collections remain. Completed
archival passed its [fixture and real-runtime checkpoint](diagnostics/2026-10-22-application-history.md). Do not
delete uncertain attempts, replay receipts or retained helpers to clear limits. No backend installation or marketplace operation is added.

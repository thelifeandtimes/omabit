# Application connections (unreleased)

Omabit helps an installed plugin authenticate, then the plugin uses its own Eyre
client directly. It can also sign in using its own URL/+code flow without Omabit.
Each approved plugin/profile gets a separate ordinary Eyre owner session. These
sessions are not restricted to a desk; the approval UI says this explicitly.

## Desktop flow

Open Setup → Application Connections, or choose **Connect via Omabit** in an
installed plugin. The latter opens the same pane with that plugin highlighted.
The ship selector combines observed running fleet ships with saved HTTP-only
ship enrollments. For a remote runner, configure its explicit browser host in
runner display settings; a remote loopback URL is not treated as this desktop.

Choose a ship, enable **Approved** on a plugin row, then **Connect**. The collector
verifies the runner, instance revision, container and HTTP identity. It reuses a
working parent sign-in or retrieves the managed ship's code privately from its
runner. Neither code nor cookie passes through QML, command arguments or public
output. The selected plugin receives its own session and talks directly to Eyre.
The UI displays the selected endpoint once. Changing selections or refreshing
clears unsubmitted approval. A changed helper or ship needs fresh approval.

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
before the command. Only `app-managed-connect` additionally requires a pinned
`--runner NAME --expected-runner-id ID`; the remaining application commands do
not accept runner/socket selection.

| Command | Effect |
| --- | --- |
| `app-plugins` | Read installed capability declarations, without executing helpers |
| `ships` | Read saved ship IDs, revisions, origins and stored-session status; not a health check |
| `app-connections` | Read active nonsecret attempts, current revision and `next_actions` |
| `app-connection REQUEST_ID` | Read one active or archived attempt without contacting a ship |
| `app-managed-connect` | Verify a fleet selection, obtain/reuse its private login, and hand off one plugin session |
| `app-connect NAME` | Explicitly approve and attempt one independent handoff |
| `app-disconnect REQUEST_ID` | Revoke only this attempt's session |
| `app-recover REQUEST_ID` | Resume cleanup; never retry issuance/delivery |
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
archival advances only `applications.json` to version 2. New readers accept
versions 1 and 2; older issuers must refuse version 2 rather than ignore archived
IDs. Do not reset or downgrade the journal. Other collector/runner formats do not
change. `app-connection` returns `archived: true` with no `next_actions` for receipts.

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

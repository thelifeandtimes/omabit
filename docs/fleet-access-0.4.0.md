# Fleet URLs and application approval for0.4.0

The user approves an application/profile for a ship. Omabit can then choose a
working registered URL without asking the user to select its network each time.
The externally published URL remains useful for people visiting the ship's apps.

## Source implementation checkpoint

`internal/shiproute` implements the route model, network eligibility, deterministic
selection and unauthenticated identity probes. A binding includes runner ID,
instance/project generation, ship identity and an opaque verified pier identity.
Two independent fakes named `~zod`, or a replacement pier using an old instance
name, cannot share a binding. Routes never hold credentials.

Candidates prefer the same host's loopback route, then the current tailnet, an
explicitly selected LAN profile, and HTTPS public routes. A remote runner's
localhost is ineligible on another collector host. Disabled routes are skipped.
Every selected candidate gets a bounded identity check; mismatch, failure or a
redirect cannot become success. A probe does not issue a session or retry a write.

Tailscale discovery parses the daemon's local JSON state and actual full DNS name.
It advertises a direct HTTP candidate only when the known listener binds a matching
Tailscale interface or wildcard address family. Loopback-only listeners need a
separately configured proxy/tunnel. Discovery does not change Tailscale, open ports
or retain peer/user/key data. The JSON format can change, so missing required
fields refuse discovery. Sources: [CLI reference](https://tailscale.com/docs/reference/tailscale-cli),
[MagicDNS names](https://tailscale.com/docs/features/magicdns),
[status types](https://github.com/tailscale/tailscale/blob/main/ipn/ipnstate/ipnstate.go).

The collector's separate `application-ship-approvals.json` format1 stores explicit
standing grants and their revision history. Grants bind the ship incarnation and
exact plugin/profile/helper digest, with no URL field. Existing per-attempt
approvals do not become standing grants. Changing the helper requires a new choice;
replacing that grant does not leave the older helper independently approved.
Revocation blocks future attempts without silently erasing active-session cleanup
receipts. Stale writes, unsupported state and corrupt history refuse unchanged.

M555 adds the public `ship-routes-v1` runner API and matching collector/client/CLI.
Schema58 stores manual/tunnel URLs and automatic-route disable choices separately
from discovery. Settings/history are atomic and revisioned. A port change cannot
silently re-enable a disabled automatic route. The live native identity, pier
continuity and host listener are verified under lifecycle admission before reads
or edits. Replacements retain separate histories and start without old choices.
Automatic localhost routes use the runner ID as their host scope; the collector
must only enable that scope for a verified local Unix enrollment, never SSH.

`omabit ship-routes` inspects the registry; `ship-routes-configure` submits an exact
reviewed selection, binding, expected revision and settings. Two desktop collectors
share runner-owned choices. Corrupt history, stale edits, changed selections and
untrusted responses refuse. The actual schema57 writer passes migration/reopen,
rollback and older-reader refusal while preserving original desk/receiver/frontend
receipts. See [operator instructions](../skills/omabit-operator/references/fleet-urls.md)
and [M555 evidence](diagnostics/2026-11-07-fleet-registry.json).

M556 connects read-only local Tailscale discovery. It runs only `status --json
--peers=false` and `serve status --json`, with byte/time bounds and no ambient
socket/proxy overrides. A second status read rejects a tailnet profile change.
The response exposes only the local DNS name, suffix and a small discovery state;
peer/user/key data and raw command errors are discarded.

Automatic Serve discovery requires a whole-origin HTTPS root proxy to the exact
managed loopback port. Foreground and background configurations are supported;
443 is preferred, then the lowest matching port. A Funnel-enabled origin is
reported as public. Unknown/conflicting configuration, path-mounted apps, shadow
handlers, raw TCP forwarding, service VIPs and different upstreams need an explicit
route. Missing/unavailable Tailscale leaves localhost and configured routes usable.
Discovery changes no daemon settings. Source references:
[Serve CLI](https://tailscale.com/docs/reference/tailscale-cli/serve),
[configuration types](https://github.com/tailscale/tailscale/blob/main/ipn/serve.go).

M557 adds Setup → Ship URLs inside the existing full panel/sidebar. It selects a
verified running ship, edits configured URLs, and retains automatic-route disable
choices. Save uses the current binding and revision; failures invalidate editing
until refresh. Offline/replaced selections cannot retain edit authority. Wide and
narrow offline UI fixtures exercise persistence, selection isolation and a
concurrent-edit refusal. Registration remains separate from network configuration.

The fleet feature is still in progress. Standing-approval UI and automatic
reconnection are not connected yet. M558 adds collector `app-ship-review`,
`app-ship-approvals` and `app-ship-approve`. Allow compares the live ship selection
and installed helper; revoke is local and remains available offline or after
helper removal. These commands change permission only, never issue a session. Installed state is unchanged;
these fixture checks do not qualify a live Tailscale deployment or installed upgrade.

M559 adds internal `ship-route-code-v1` credential retrieval for the upcoming
automatic connection path. The runner holds lifecycle admission, checks the exact
ship/pier/process, registered route revision and enabled choice, and rechecks
Serve/Funnel configuration when selected. The runtime verifies native identity
and pier continuity before and after the private credential read. Client errors
and runner audits discard raw secret responses. No public command exposes this
new credential path; it creates no session or grant. Native-engine fixtures,
runner API and client race checks pass. Actual ship/transport qualification is
not claimed.

M560 connects the durable fleet-attempt and issuer core. New attempts retain the
full binding and exact standing-grant revision, with fresh checks before issuance
and delivery. They obtain their own origin-bound session and never reuse a parent
login. Uncertain effects fence retries across URLs. Legacy attempts without a
verified incarnation must finish recovery before the same integration can move to
a fleet route. Cleanup remains available after revocation. Active journal format3
and fleet archive format2 pass actual previous-writer preservation and older-reader
refusal checks; runner schema58 is unchanged. The public automatic connection flow is implemented in M561 below. See [attempt evidence](diagnostics/2026-11-07-fleet-attempts.json).

M561 adds `app-fleet-connect` for a selected standing grant and fresh request ID.
It discovers the collector's current tailnet (never assuming the runner's network),
uses local Unix enrollment for localhost eligibility, and supports explicitly
selected LAN profiles. A15-second overall probe budget bounds URL fallback. It
rechecks selection/route/grant before issuance; old requests and uncertain effects
never cause another login. Public connections use Secret Service without a file
fallback. Synthetic end-to-end CLI checks exercise delivery and failure recovery.
See [connection evidence](diagnostics/2026-11-07-fleet-connect.json).

## Integration still required

1. Setup URL editor, runner-owned registry, localhost and Tailscale/Serve discovery,
   explicit route/disable storage and CLI are implemented.
2. Collector network inspection, shared configuration and fresh binding checks
   are implemented in the public automatic connection command.
3. Setup standing approval and issuance integration (CLI authoring is implemented),
   the journal/issuer core now checks permission under the same lock that publishes
   a new connection attempt. The public automatic-route command is implemented; connect its permission UI.
4. Compatible receiver handoff and automatic reconnection. Preserve origin-bound
   sessions, original uncertain attempts and cleanup records; never replay pending
   app writes or silently copy a cookie to a different origin.
5. Agent-assisted exposure plans for Tailscale/Cloudflare/custom DNS, with verified
   registration and explicit disable behavior. Urbit apps own content publication.

Release qualification remains deferred until these features are complete.

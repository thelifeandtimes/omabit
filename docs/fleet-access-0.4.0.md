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

The fleet feature is still in progress. Standing approval is connected in M562;
automatic session reconnection remains pending. M558 adds collector `app-ship-review`,
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

M562 connects standing permission and automatic connection to the full panel.
Capable remote ships appear without requiring a guessed local URL. The Allowed
switch persists for the exact ship/helper; Disconnect manages the existing session
separately. Offline/uninstalled grants remain revocable, and pending/legacy attempts
retain recovery controls. QML structural identity checks tolerate object field
reordering. Failed saves display confirmed state; refresh does not replay effects.
Wide/narrow full-panel and hosted-flow fixtures pass, as does actual service drain
with inert helpers. See [UI evidence](diagnostics/2026-11-07-fleet-permission-ui.json).

M563 adds selected-ship Remote access help to Setup → Ship URLs. The default agent
receives a bounded nonsecret binding/runtime/revision hint and a provider/action
choice. The bundled skill covers audience, exact upstream, verification, registration
and targeted disable for Serve/Funnel, Cloudflare and custom domains. Help does not
configure a provider. Full-panel and prompt validation fixtures pass; no live provider
deployment is claimed. See [Help evidence](diagnostics/2026-11-07-remote-access-help.json).

M564 adds bounded, non-mutating session health and `app-fleet-health`. Current
permission, exact helper, ship incarnation and network eligibility precede credential
reads; the old attempt remains unchanged. Expiry, rejection and unreachable origins
are distinguished from malformed state, redirects and identity changes. Synthetic
route/engine/HTTP and public CLI checks pass. This prepares the remaining replacement
flow; it does not reconnect. See [health evidence](diagnostics/2026-11-07-fleet-session-health.json).

## Integration still required

1. Setup URL editor, runner-owned registry, localhost and Tailscale/Serve discovery,
   explicit route/disable storage and CLI are implemented.
2. Collector network inspection, shared configuration and fresh binding checks
   are implemented in the public automatic connection command.
3. Standing approval, journal/issuer and full-panel connection UI are implemented.
   Permission is checked under the lock that publishes each new attempt.
4. Compatible receiver handoff and automatic reconnection. Preserve origin-bound
   sessions, original uncertain attempts and cleanup records; never replay pending
   app writes or silently copy a cookie to a different origin.
5. Agent-assisted exposure planning/context and skill pathways are implemented.
   Live provider setup remains specific to the user’s authorized environment; Urbit
   apps own content publication.

Release qualification remains deferred until these features are complete.

M565 completes the scoped native Urgit workflow test and operator guidance without
an adapter. It does not change the outstanding receiver/reconnection work above.

M566 adds bound local replacement preparation in an isolated Theme receiver branch.
It validates the old accepted attempt/account/fingerprint, fences local admission,
waits for owners and blocks unresolved writes without pretending old channels were
deleted. Existing cleanup, credentials and account metadata remain. This is not
yet a reservation, successor delivery or a capability consumed by Omabit. DMs drain,
activation/recovery and issuer lineage remain pending. See
[preparation evidence](diagnostics/2026-11-07-theme-handoff-preparation.json).

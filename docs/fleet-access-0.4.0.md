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

These are internal foundations with isolated race coverage, not a finished fleet
feature. There is no public route registry endpoint, automatic Tailscale invocation,
standing-approval UI/CLI or automatic reconnection yet. Installed state is unchanged.

## Integration still required

1. Runner-owned registry API and persistence, fresh runtime/pier binding, automatic
   local/Tailscale candidates, explicit manual/tunnel routes and disable controls.
2. Collector inspection from each client's network, shared runner configuration and
   fresh binding verification before credentials can be sent.
3. Setup and CLI standing approval, checked under the same lock that publishes a
   new connection attempt. An old snapshot must not bypass revocation.
4. Compatible receiver handoff and automatic reconnection. Preserve origin-bound
   sessions, original uncertain attempts and cleanup records; never replay pending
   app writes or silently copy a cookie to a different origin.
5. Agent-assisted exposure plans for Tailscale/Cloudflare/custom DNS, with verified
   registration and explicit disable behavior. Urbit apps own content publication.

Release qualification remains deferred until these features are complete.

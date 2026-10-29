# Backend setup: inspect state before choosing an action

Use this reference when implementing a plugin's missing-backend button. It is a
source-reviewed pattern, not a qualified universal installer. Keep the descriptor,
UI and direct Eyre client in the plugin; do not require an Omabit catalog or proxy.

## Separate the observations

Keep transport/authentication errors, installation state and client compatibility
as separate values. An HTTP exception, redirect, missing route or failed scry does
not by itself establish that a Clay desk is absent. A reachable page does not prove
that the authenticated API or its schema is ready.

| Evidence | Appropriate next action |
| --- | --- |
| Required API responds with a supported schema | Open the feature |
| Ship unreachable or status cannot be read | Show the error and Check again |
| Session rejected | Reauthenticate this plugin's account |
| Desk source already recorded or download pending | Observe progress; preserve its source |
| Existing desk suspended | Offer Resume for that desk, preserving its source |
| Desk present but agent/API unavailable | Diagnose startup/version; do not reinstall by inference |
| Authoritative status establishes absence and no pending install | Offer the reviewed install target/publisher |
| Backend present but incompatible | Explain the supported update choice |

Omart's `pals-install-state` checks `%hood` sources, Clay's desk state and whether
`%pals` is running. Its UI distinguishes missing/installing/waiting/starting/
suspended/ready; failed status offers Check again. Its `/omart/install-pals` route
is specific to Omart. Reuse the decisions without requiring that route in unrelated
plugins. A `%hood` `/kiln/pikes` inventory includes source, hash, desk state and
pending wefts for existing desks; an omitted desk alone does not rule out a source
or installation that has not arrived yet.

## Verified Gall wire shapes

The reviewed Urbit `mar/kiln/install.hoon` accepts JSON on the `kiln-install` mark:

```json
{"local":"LOCAL_DESK","ship":"~PUBLISHER","desk":"REMOTE_DESK"}
```

Send this as a normal authenticated poke to the user's own `hood` agent. These
uppercase strings are placeholders, not valid targets. Bind all three values to
reviewed plugin-local metadata and show the target ship/publisher before dispatch.
Local and remote desk names can differ. Do not derive them from a display label,
URL, marketplace text or an assumed universal `%groups` publisher.

The reviewed `kiln-revive` mark accepts a JSON string naming the existing local
desk. Resume is a different action from installing or changing a source. Preserve
the observed source even when it differs from the plugin's suggested new-install
publisher. These shapes are corroborated by Talon's `LatticeInstall.installPoke`
and Omart's source-preserving resume flow; support still needs a real runtime test
against the selected backend/version before advertising one-click completion.

Talon's `installAndWait` correctly separates a poke acknowledgement from later
readiness and reports a wait timeout as potentially still completing. Its boolean
`isInstalled` probes catch failures and return false; do not copy that boolean as
install authority. Preserve the diagnostic distinction above. A different desk
requires its own readiness probe, not Lattice's manifest endpoint.

Record the exact submitted target and uncertain outcome. After a timeout, process
restart or lost reply, resume observation before offering another install. Current
kiln payloads shown here have no client idempotency field; do not invent one or
claim an uncertain write is safe to replay. A bounded UI wait is not proof that
the ship refused the operation.

## Grubbery adds an application layer

Distinguish the framework desk, available nexus code, installed app instance,
loaded application tree, bound route/API, and required permits. Lattice's example
instance `/apps/lattice.lattice_app` needs nexus `/lattice/app`; a plain folder or
committed nexus source alone is insufficient. Discover routes from that app's
`bind-http` and versioned interface; do not guess `/grubbery/<app>`. Changing the
framework publisher can affect other apps and is a separately reviewed action.

Do not turn Lattice's historical deployment note about cookies dying on restart
into a general client rule. The pinned Omabit recovery fixtures preserve their
original Eyre sessions across tested same-pier restarts. Treat actual authentication
refusal separately from transport/channel recovery; do not re-login on every
restart merely because a deployment note says so.

Source snapshots reviewed: Urbit `50c7f523`, Talon `7cbaa152`, Omart `6ebac449`,
Lattice `5d90b165`. Exact file hashes and limitations are recorded in the Omabit
backend-setup diagnostic. Recheck the target release's source when implementing
an adapter; these snapshots are not a claim about every hosted ship.

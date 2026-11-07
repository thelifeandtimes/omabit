# Ship URLs across the fleet

The full panel has **Setup → Ship URLs** for selecting a running ship, reviewing
detected addresses, editing configured URLs, and saving shared route choices.
Refresh after a conflict or lost save reply before making another edit.

Use the selected installation's `omabit` CLI and collector registry. A runner
advertising `ship-routes-v1` supports:

```sh
omabit --config-dir REGISTRY ship-routes \
  --runner NAME --expected-runner-id RUNNER_ID --instance INSTANCE
```

This reads the verified running ship, pier incarnation, saved URL settings and
automatic routes. It does not obtain a login code or configure a tunnel. Local
routes belong to that runner host; a remote runner's `localhost` is not the
desktop's localhost. Independent fakes with the same `~ship` remain distinct.

For an authorized URL registration/edit, save the inspection as `ship-urls.json`
and construct the update from its actual selection and revision:

```sh
jq '{selection, update: {binding: .record.binding,
  expected_revision: .record.revision, settings: .record.settings}}' \
  ship-urls.json > reviewed-urls.json
```

Edit `update.settings`, preserving existing choices. A manual public route is:

```json
{"id":"website","origin":"https://example.com","network":"public","scope":"","source":"manual","enabled":true}
```

Replace the example with the user's actual authorized origin; registering it does
not prove its reachability. Each configured route has all seven fields above.
Use `source: "manual"` or `"tunnel"`; automatic IDs/provenance are reserved. Sort
routes by ID and disable IDs lexically. Origins have no path, query, fragment or
credentials. Networks are `local` (exact runner-host scope), `tailnet` (full
MagicDNS name and tailnet suffix), `lan` (explicit network profile), or `public`
(HTTPS and empty scope). Do not infer a tailnet hostname from an SSH alias.

```sh
omabit --config-dir REGISTRY ship-routes-configure \
  --runner NAME --expected-runner-id RUNNER_ID --request-file reviewed-urls.json
```

The API checks the current process, pier and settings revision. A stale edit or
lost reply requires another read; do not silently overwrite a newer edit. Changing
URLs does not grant application access. Disabling a registered URL stops it being
an eligible candidate; it does not shut down a tunnel, firewall rule or public DNS.

Current implementation evidence covers isolated core, runner API/store, runtime
adapter, collector/client and CLI fixtures, plus the actual schema57 writer's
migration to58. It does not qualify an installed upgrade. Automatic localhost
and read-only Tailscale/Serve discovery are implemented. `network` reports whether
the local daemon is available and whether an HTTPS root proxy matches this ship.
Funnel-enabled routes are public; disabling `tailscale-serve` preserves the choice
across proxy port changes. The reader never changes Serve/Funnel, and a failed
lookup does not remove manually configured routes. Path proxies, service VIPs,
raw TCP forwarding and bespoke reverse proxies need explicit URL registration.
No live Tailnet qualification is claimed by these fixture checks. The full-panel
Setup URL editor passes offline interaction/rendering checks. Standing-approval authoring and automatic URL selection/connection are implemented
below. Their UI integration and automatic reconnection remain open.
Do not claim those features from the presence of this registry capability alone.

Keep existing sessions bound to their original URL. Never copy a cookie to another
origin or replay an uncertain application write to make a connection appear healthy.


## Standing application permission

Use the matching `urbit-collector` from the selected installation/profile. A grant
is for one plugin/profile/helper generation and verified ship incarnation across
its registered URLs. It permits broad owner sessions; it is not limited to a desk.
Do not turn an earlier per-attempt approval into standing permission, or grant
an agent's own integration access without the user's authorization.

```sh
urbit-collector --config-dir REGISTRY --runner NAME --expected-runner-id ID \
  app-ship-review --instance INSTANCE > ship-access.json
urbit-collector --config-dir REGISTRY app-plugins > installed-plugins.json
```

Construct a choice using `routes.record.binding`, `routes.selection`,
`approvals.revision`, and the exact selected plugin's `target` from these replies:

```json
{"version":1,"binding":BINDING,"selection":SELECTION,"target":TARGET,"expected_revision":REVISION,"allowed":true}
```

The placeholders above describe the object shape; replace them with the actual
JSON values before saving. Submit an explicitly authorized choice:

```sh
urbit-collector --config-dir REGISTRY --runner NAME --expected-runner-id ID \
  app-ship-approve --request-file reviewed-access.json --allow-owner-session
```

The command verifies the installed helper and live ship selection again. It does
not retrieve a login code, create an owner session, execute the helper or connect.
A helper update requires another user choice. Conflicts require a fresh read;
never automatically rebase an allow over a revocation.

`app-ship-approvals` reads the current local choices without contacting runners.
To revoke, copy the exact saved `binding` and `target`, use the current list's
`revision` as `expected_revision`, set `allowed:false`, and omit `selection`.
Run `app-ship-approve --request-file reviewed-revocation.json` **without** runner
flags or `--allow-owner-session`. Revocation remains available offline and after
a plugin is removed. It blocks new attempts; use each existing connection's
Disconnect/recovery path separately to clean up its session.


## Connect using an existing grant

The runner must also advertise `ship-route-code-v1`. Construct a request from the
current allowed entry in `app-ship-approvals`, using that entry's `revision` as
`grant_revision` (not the whole list's revision):

```json
{"version":1,"binding":BINDING,"target":TARGET,"grant_revision":GRANT_REVISION}
```

```sh
urbit-collector --config-dir REGISTRY --runner NAME --expected-runner-id ID \
  app-fleet-connect --request-file connection-choice.json --request-id FRESH_32_HEX_ID
```

The command resolves the exact installed helper, checks standing permission, and
selects a verified registered URL from the collector's current network. Localhost
is eligible only through a pinned local Unix enrollment. Tailscale membership is
read locally. If the user explicitly selects a current LAN profile, add
`"lan_profiles":["PROFILE_ID"]`; do not infer it from a remote runner's route list.
URL fallback uses only unauthenticated probes. Public URLs require HTTPS.

A successful connection owns a new origin-bound session stored in Secret Service.
No parent ship login is created or reused, and there is no plaintext fallback.
After any lost/failed reply, inspect `app-connection REQUEST_ID` or `app-connections`
and follow the reported recovery action. Never submit a new request or change URLs
to bypass an unresolved login/handoff. An existing request ID is not replayed.
Revoking permission does not prevent cleanup of its existing session. These CLI
paths pass synthetic offline runner/HTTP/receiver/keyring-transport checks; live
Tailnet, actual keyring and installed upgrade qualification remain separate.

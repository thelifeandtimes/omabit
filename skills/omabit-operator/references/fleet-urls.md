# Ship URLs across the fleet

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
No live Tailnet qualification is claimed by these fixture checks. Setup URL editing,
standing-approval UI and automatic application connection/reconnection remain open.
Do not claim those features from the presence of this registry capability alone.

Keep existing sessions bound to their original URL. Never copy a cookie to another
origin or replay an uncertain application write to make a connection appear healthy.

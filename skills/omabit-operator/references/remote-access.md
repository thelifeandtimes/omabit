# Agent-assisted remote access

Use **Setup → Ship URLs → Remote access help** with a verified running ship.
Agent Help opens Omarchy's configured default agent with the selected runner,
ship/pier binding, runtime and saved URL revision. It carries no credentials.
Choose private Tailnet access, public Tailscale Funnel, Cloudflare Tunnel, or a
custom-domain reverse proxy. The optional URL is a user-supplied planning hint;
the disable pathway requires an exact URL. Opening Help does not change a network.

This workflow exposes the ship's web interface. It does not publish a desk or
configure content in Urbit applications. Follow [app development](app-development.md)
for desk releases and [fleet URLs](fleet-urls.md) for URL registration and application
permission. A registered URL, a public tunnel and an application grant are separate.

## Inspect and prepare

1. Use the exact bundled CLI/profile in the Help prompt. Run its read-only
   `ship-routes` command and compare the full binding, including project generation
   and pier continuity, with the supplied selection. A replaced ship needs a fresh
   selection. Refresh after a runtime restart or URL revision change.
2. Identify the selected runner host and its verified HTTP listener. A remote
   runner's `127.0.0.1` belongs to that runner, not the desktop or a separate proxy
   container. Do not infer the upstream from an SSH alias or an old port.
3. Inspect only the relevant existing proxy, domain and network configuration,
   using the user's authorized access. Keep tokens, cookies, login codes, tunnel
   credentials and provider account data out of the prompt, argv and reports.
   Never read an entire provider configuration merely to copy it into diagnostics.
4. Prepare a concrete change and rollback: exact ship, host, upstream, hostname,
   audience, existing routes affected, provider configuration and verification.
   Preserve unrelated sites and shared tunnels. Honor authorization already given
   in the session; if exposure or account changes are not authorized, get approval
   for this concrete plan before applying them. The Help button itself supplies
   no such authority.

Proxy only the ship's HTTP listener. Never publish Omabit runner RPC, Lens/control
ports, Docker, SSH management, the pier filesystem or a directory containing keys.
Prefer an existing loopback listener plus a proxy on that host over widening the
ship's listener. Avoid exposing raw HTTP publicly. Keep public-origin certificate
and hostname verification enabled. Do not disable TLS validation to make a test pass.

## Choose the access path

**Private Tailscale Serve.** Read the current local node status and Serve mapping.
Use the daemon's actual MagicDNS name and check which Tailnet identities can reach
it. A whole-origin HTTPS proxy to the selected loopback web listener is compatible
with Omabit's automatic discovery. Inspect port/path conflicts before proposing a
change; replacing an existing Serve listener can affect another service. A private
Serve URL is not a clearweb publishing URL. Follow the installed version's
[Serve reference](https://tailscale.com/docs/reference/tailscale-cli/serve).

**Public Tailscale Funnel.** Funnel uses the node's `ts.net` name and supports
public listening ports 443, 8443 and 10000. It requires MagicDNS, HTTPS and the
relevant Tailnet policy permission. A Serve/Funnel port changes audience according
to its configuration, so review any existing mapping first. Do not promise a
custom-domain TLS endpoint by adding a CNAME to a Funnel name. These constraints
come from [Funnel documentation](https://tailscale.com/docs/features/tailscale-funnel).

Record the exact flags and selected mapping when configuring Funnel. Its `off`
form removes a selected server, while `reset` clears the configuration; never use
a global reset to remove one ship from a shared host. Background mode persists
across restarts. Review the installed CLI's help before producing commands.
[Funnel CLI reference](https://tailscale.com/docs/reference/tailscale-cli/funnel).

**Cloudflare Tunnel.** Select the authorized account/domain and an existing or
explicitly approved tunnel. A published application route maps the hostname to
the ship's web listener reachable from `cloudflared`. Route the whole origin,
preserving paths. Check existing Access, caching and security rules against the
intended audience; an interactive Access login can prevent ordinary application
connectors from working. Do not silently remove that policy to make a connector
pass. Follow [published application routing](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/routing-to-tunnel/).

DNS and tunnel configuration have separate lifetimes. An account's hostname can
point to its tunnel subdomain, but the DNS record alone does not establish a live
route. A stopped tunnel can leave DNS behind. Preserve shared tunnel mappings and
remove or restore only this plan's DNS/route changes when disabling access.
[Cloudflare DNS reference](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/routing-to-tunnel/dns/).

**Custom domain and reverse proxy.** Establish domain ownership and the exact
proxy host before proposing DNS, certificate, NAT or firewall changes. Reuse the
user's chosen proxy when practical. Prepare a dedicated hostname/vhost with HTTPS
and a whole-origin mapping to this ship's web listener. Verify the proxy can reach
that upstream from its own network namespace. Certificate renewal, address changes
and restoration of previous DNS/vhost configuration belong in the plan. A DNS
record by itself does not provide HTTPS or make a private address publicly routable.

## Verify, then register

After authorized provider changes, inspect the resulting configuration and probe
the intended HTTPS origin without cookies, credentials or redirects. Request
`/~/host` with a short deadline and compare the returned ship identity. Wrong ship,
redirect, certificate failure or an access-policy page is not successful verification.
Test from the intended audience's network: a test inside the Tailnet alone does
not prove clearweb reachability. Do not replay app writes as a connection check.

Check the intended Urbit app's browser behavior and long-lived HTTP/event transport.
WebSocket support alone does not establish support for every app protocol. Cloudflare
supports WebSockets but may terminate connections during updates or idle periods;
application recovery must not repeat uncertain writes. See
[WebSocket behavior](https://developers.cloudflare.com/network/websockets/).

Read `ship-routes` again. Verified automatic Serve/Funnel routes should appear with
the correct audience. Register other verified URLs using the current binding and
revision as described in [fleet URLs](fleet-urls.md); public origins use HTTPS and
an empty network scope. Do not overwrite unrelated settings or enable application
permission as part of exposure setup. New connections still need their standing
ship/helper grant.

## Take a URL offline

Keep these two effects explicit:

- Turning off a route in Omabit stops future automatic selection of that URL. It
  does not stop a tunnel, remove DNS or terminate an existing plugin session.
- Removing public/private access requires a targeted provider/proxy change. Identify
  the exact hostname, mapping and shared-service dependencies, then apply only the
  authorized change. Verify externally that the removed audience can no longer reach
  the ship; preserve any remaining intended local/private access.

Refresh Omabit after provider changes and disable/remove the corresponding saved
route with its current revision. Disconnect existing sessions separately if wanted.
Keep enough nonsecret before/after configuration information for rollback. Do not
delete an entire shared tunnel or global proxy configuration to remove one origin.

## Tested boundary

Omabit's typed Help context, selected-ship UI, default-agent dispatch, route registry
and permission/connection flows have isolated fixture coverage. This is an assisted
workflow, not an automatic provider installer or a certification of live provider
deployment. The user's agent must check the actual host, network audience, provider
version, DNS, TLS, proxy rules and Urbit app behavior. Never claim an installed or
public deployment passed because a local fixture did. Keep production exclusions
and the app's release gates in force throughout.

# Agreed product direction

These decisions come from the user; implementation status is separate in STATUS.md.

## Execution and scope

The default execution host is an always-on workstation; the laptop controls it
through ordinary SSH on the user's Tailscale network. Both run Linux/Omarchy.
One Linux account owns each host's ships. A persistent per-user host manager is
acceptable and remains independent of the graphical shell. SSH-accessible VPSes
use this same model. The first pilot uses disposable ships only.

“Cloud” means an existing hosted ship reached over HTTP/HTTPS Eyre without SSH,
not cloud-account provisioning. That connector is deferred. All backups,
including GroundSeg application backups, are out of scope.

The runner owns local canonical metadata. Client caches are explicitly staleable.
No automatic migration/failover, live-pier synchronization or per-client runtime
ownership. Stable IDs identify instances; ship names are not globally unique
because independent development groups may each contain fake `~zod`.

## Runtime and lifecycle

Use the standard pill and stable runtime by default, with explicit overrides.
Adopt GroundSeg's public channel vocabulary (`latest`, `edge`, `canary`), not
Vere's distinct pace vocabulary (`live`, `soon`, `edge`). Changed runtime images
should produce persistent Omarchy notifications that require dismissal. The
pilot implements notice records/checks, not final desktop notification delivery
or application of updates.

Use a shared fleet CPU/memory budget with warnings, not hard resource limits.
Archive removes the container and moves the pier into a retained archive;
permanent purge is a separate operation. There is no “delete everything” button.
The pilot budget is per execution host; cross-host budget aggregation is future
client work. Zero configured warning thresholds mean warnings are disabled.
Docker CPU percentages can exceed 100% because multiple CPU cores are involved.

Agents have unrestricted development operations but restricted real-ship access.
Real Hoon requires explicit permission. The current role split denies real
mutations to agents and uses operator confirmation for real operations; future
approval/delegation should not be confused with a same-account sandbox.

## Bar widget (not implemented yet)

One compact fleet icon opens a host-grouped list. Most users start with one or
two hosts, so show the host once as a group heading rather than on every pier row.

Each row shows identity/label, development classification, and two colored icons:
container status and runtime status. A disconnected host makes its observations
stale, not stopped. Preserve last-seen times in details/tooltips.

No independent browser-status indicator. Use a single Open GUI button when
available; otherwise disable that button and label it “needs tunnel” or
“unavailable.” The future host/group controls can establish required tunnels;
this pilot's `ship open` / `ship tunnel` does it explicitly.

A separate copy-code control fetches `+code` on demand and copies it on the client.
Stopped ships have no cached login-code affordance. Open the ship's home interface
rather than an application chooser for the first release.

## Full pane (not implemented yet)

Basic information: identity, CPU/memory and endpoint health. Disk availability,
restart count, uptime, image identity and all other diagnostics belong in deeper
details rather than cluttering the primary view. Heavy/kernel metrics are only
requested when the user explicitly opens Details. Live metrics suffice; no time
series database or historical retention is required.

Use GroundSeg's tmux console approach for a button that opens the current ship's
Dojo in the configured terminal. Do not launch another Vere process. The CLI's
interactive attachment is implemented; terminal-app launching from QML is later.

## Development environments

Support groups of fake ships, independent Docker networks and multiple separate
fakezod environments. Ultimately a group's galaxies must communicate across the
user's Tailscale network, e.g. fake `~zod` on workstation and fake `~nec` on laptop.
The same-group networking design must not leak into the real network or another
fake environment. See DEVNET.md for the shared-loopback foundation and relay work.
Source editing, mounts, desk workflows and other development conveniences remain
questions to explore later, not blockers for this pilot.

## Public exposure

Later fork/extend Anchor for an invite-only pilot on the user's `rooftop` server,
initially only their machines, eventually covering all Anchor-supported services.
Native Planet/StarTram signup/payment should be a provider adapter and must not
control private pier ownership or operation. The user handles Native Planet
outreach in parallel; a solid demonstration comes before requesting their support.
No Anchor infrastructure is modified by this package.

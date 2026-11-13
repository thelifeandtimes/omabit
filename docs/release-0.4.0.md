# Omabit 0.4.0 — fleet access and app development

Release candidate; the automated upgrade and feature scenarios pass. Stable
publication awaits passing required PR checks and manual review.
The upgrade being qualified is **0.3.7 → 0.4.0** on Linux/AMD64 with an
owner-user runner. Production installations have not been upgraded by this work.

## Changes

- **Remote runner setup:** dependency handling and first-boot guidance for tested
  Ubuntu, Debian and Arch/Omarchy installations. The operator skill distinguishes
  tested configurations from cases needing agent assistance. See the
  [acceptance matrix](remote-runner-acceptance.md).
- **Agent Help:** launch the agent selected in Omarchy, with the current context
  and bundled operator/developer skills. Interactive sessions remain open. Launch
  is supported; more detailed task guidance can evolve in later releases.
- **Fleet access:** each ship has a Ship URLs section. Discover local and Tailnet
  routes where supported, configure additional routes, and grant an application
  standing approval for that ship. Connections verify the ship before sending
  credentials. Supporting receivers can recover or move to another verified URL
  under the same approval. See [fleet access](fleet-access-0.4.0.md).
- **App development and publishing:** Setup shows a zoomable permission tree from
  global settings through runners, ships and desks. Desk creation, commit and
  publication start blocked. Explicit scoped overrides and inherited permissions
  support reviewed agent workflows. The CLI stages source, reports Unix/Clay
  differences, and follows the configured release checks for source, activation,
  frontend assets and publication. See [development workflows](app-development-0.4.0.md)
  and [permissions](app-development-policy.md).
- **Urgit workflow guidance:** the bundled skill documents the tested native
  Git-to-Clay pathway and its own release configuration. Omabit does not replace
  Urgit's release gates or orchestrate companion plugin releases.
- **Container resources:** choose RAM from1–16GiB and CPU limits from1–16. Existing
  omitted settings retain4GiB/2CPU defaults. Use initial boot's advanced settings,
  or stop a ship, save its runtime limits and restart it. See
  [resource policy](resource-policy.md).

## Upgrade and support window

Update the desktop plugin, allow its verified bundle to load, then review and
apply the service update offered in Setup. The desktop update and service update
are separate operations. Do not use this draft as a request to upgrade a live
runner before release approval.

The runner advances from schema47 to59. The migration preserves existing fleet,
access, storage and operation history and grants no new development permissions
or standing application approvals. Use matching collector/runner artifacts and
capability negotiation. An older runner must refuse the newer database; lowering
its schema marker or resetting state is not a downgrade procedure.

Older stable installations should follow their documented upgrade path to0.3.7
first. Remote service upgrades remain explicit. The qualification scope does not
include production ships, provider credentials, public DNS/Tunnel changes, remote
service updates or power-loss testing. Live public-provider configuration remains
for the operator to validate. See the [qualification record](releases/0.4.0/qualification.json),
[upgrade evidence](releases/0.4.0/rehearsal.json) and
[preparation findings](releases/0.4.0/preparation.md) for the exact results and limits.

## Before stable publication

Complete the [manual upgrade rehearsal](releases/0.4.0/manual-review.md), required
release PR checks and the user's live provider validation. Coordinate compatible
Theme/DMs receiver releases before advertising automatic reconnection to their
public users: the qualification records the exact tested private helper commits,
not a claim that the public plugin distributions already include them. Legacy
receivers retain their existing connection behavior. The draft release remains
unpublished and master remains unchanged until review and approval.

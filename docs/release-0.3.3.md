# Omabit 0.3.3 — reliable imports and Ubuntu package verification

This Linux/AMD64 maintenance release fixes the following existing flows:

- Ubuntu APT verification combines components of the same signed repository with
  identical trust authority and accepts bounded long unselected metadata fields.
  Missing package evidence names the affected packages. Signature and payload
  checks remain required; missing packages still use the setup terminal approval.
- Local stopped-pier import supports logical sizes up to8 TiB, with size-aware
  transfer deadlines. Same-filesystem moves retain the original object; cross-
  filesystem copies retain verification and free-space checks. Cross-host handoff
  remains capped at100 GB and uses its existing wire format.
- Import progress uses the active theme. A successful, identity-matched import
  clears the form and opens its monitoring page, with the ship still stopped.
- Application connection titles, approval toggles and buttons share one row.

## Apply the update

Run `omarchy plugin update thelifeandtimes.omabit`, wait for0.3.3 to activate,
then choose **Update Dependencies** in the menubar widget and approve the OS
prompt. Alternatively use **Setup → Local services & updates → Check services →
Update services**. Updating the plugin alone does not replace an installed runner.
The local import fix requires the matching0.3.3 runner.

Remote setup attempts retain their original helper for safe retry. An unfinished
0.3.2 attempt will not pick up this fix through **Resume setup**. Start a new
connection attempt with a different Advanced connection name, the same SSH host
and owner; preserve the old attempt's records. Existing installations are detected
and paired without resetting ships. This release does not automatically upgrade
remote runners, copied CLI tools, Docker packages or Vere.

## Support and qualification

Runner schema47, desktop state1 and bootstrap API1 remain unchanged. The supported
prior stable path is0.3.2 → 0.3.3. Earlier published installations can use the
[existing upgrade paths](releases-0.3-upgrade.md) to0.3.2 first. Matching
Omabit/collector tools activate as one verified bundle; the explicit service button
updates the owner-user runner while preserving running ships and stopped intent.

The [qualification record](releases/0.3.3/qualification.json) links the exact
artifacts, passed [upgrade rehearsal](releases/0.3.3/rehearsal.json) and limits.
Source tests cover large sparse fixtures, same-filesystem512 GiB moves, real small
cross-filesystem copies, interruption/recovery and stale completion rejection.
No full >100 GB physical transfer or new Ubuntu runtime installation is claimed.
Rooftop's authenticated installed-package verification was tested read-only; no
production ship was imported or changed. The disposable upgrade guest uses real
QML buttons and guest-only authorization rules; it does not test human password
entry, remote upgrades or power cuts.

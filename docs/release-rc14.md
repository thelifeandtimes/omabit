# Omabit 0.1.0-rc.14 — retained desktop releases and test profiles

RC.14 is a Linux/AMD64 **prerelease** introducing the new desktop bootstrap and
development/testing workflow on stock Omarchy. Runner state remains **schema21**;
the separate desktop activation/profile store is version1.

## How it works

1. **Bootstrap:** Omarchy loads a small retained service and replaceable bar adapter.
   The service survives supported plugin rescans and coordinates desktop activation.
2. **Release bundle:** the UI, JavaScript, icons and matching three tools are verified
   together and loaded from an immutable, release-specific cache directory. New
   versions have new paths, avoiding reuse of stale nested QML/JavaScript.
3. **Profile:** the selected bundle uses either existing default state or a named
   profile's private configuration, credentials, preferences and caches. A new test
   profile is a fresh desktop, not a reset of an installed runner or its piers.
4. **Runner:** the independent host service continues running ships. Desktop updates
   and profile switches do not perform privileged host upgrades or database downgrades.

The default profile follows the installed plugin's published pins unless explicitly
pinned to a candidate. Preparation and successful activation are distinct: busy work
defers switching, the loaded version/API and widget are checked, and failed
candidates retain the previous working desktop. Diagnostics show requested versus
loaded release/profile. The existing fleet is re-observed after activation.

## Changes

- New versioned native entry points and stock `keepLoaded` service behavior.
- Complete size/SHA-256-pinned UI/tool bundles, safe bounded extraction, retained
  archive verification, and paired tool paths for each running UI.
- Single-owner activation lease, restart-safe intent/acknowledgement state,
  failed-candidate recovery and unknown-format refusal.
- Development snapshot builder, release import/selection, named isolated profiles,
  paired CLI execution and guarded stock-install/update VM scenarios.
- Setup diagnostics for profile, source, loaded/requested version and collector state.
- Bounded inventory reconnect backoff, retained stale observations and explicitly
  flushed page/ship preferences across desktop restarts.
- One application version value for all visible version labels and live status.

See [desktop development](desktop-development.md) for commands and state boundaries.

## Fresh workstation installation

Run as the workstation's ordinary desktop user:

```sh
omarchy plugin install https://github.com/thelifeandtimes/omabit.git --enable
omarchy shell thelifeandtimes.omabit.bootstrap status
```

The initial download/verification may take a little time. Expect `phase:"ready"`,
`loaded_version:"0.1.0-rc.14"`, `profile:"default"`, and an empty error field.
An empty fleet is expected on a fresh workstation until a runner is enrolled.

Use **Setup → Manage runners → Connect another runner** for an existing remote
runner. That requires the workstation's authorized management key, a working SSH
route and verification of the runner/SSH host identities. Remote-only desktop use
does not require local Docker or running **Set up this computer**. Browser access
to remote loopback HTTP endpoints needs its own route and Browser host setting.

For local hosting, use **Set up this computer** and the normal OS-authorized setup.
See [native installation](native-plugin.md) and [binary installation](release-install.md).

## Updates and selected development versions

For the native plugin:

```sh
omarchy plugin update thelifeandtimes.omabit
omarchy shell thelifeandtimes.omabit.bootstrap status
```

An explicitly pinned development profile stays pinned. To resume following published
pins, run from the installed plugin checkout or a matching source checkout:

```sh
python3 -B scripts/omabit-dev.py select default --release follow
```

The downloaded archive's snapshot installer remains a separate installation model
and uses its existing cold-replacement procedure. The retained-bootstrap/profile
workflow is provided by the native GitHub installation.

Existing runner upgrades continue to use the [schema21 procedure](host-compatibility-upgrades.md).
Older runners remain observable; unavailable newer capabilities are shown as upgrade
guidance. A desktop rollback does not make a schema20 runner compatible with schema21.

## Verification and planned acceptance

The source implementation passed unit/model/offscreen checks, real Quickshell
activation/profile/recovery tests and a live stock-command sourceport migration.
The live check preserved two healthy ships, engine/runner processes, enrollment and
preferences, and required no shell restart. Those development results are recorded
separately from RC.14's frozen-artifact checks in the attached `qualification.json`.

The next acceptance sequence is the user's fresh workstation install, connection
to sourceport's existing runner, sourceport's Omarchy upgrade, then the other
workstation's update. The [step-by-step workstation handoff](rc14-workstation-test.md)
includes sourceport's separate runner/guard upgrade before its OS update. Those
results and the full two-version clean-host VM matrix
are pending; this prerelease does not claim universal distro/runtime qualification.
The Docker27.2–29.x/API1.47 policy, pinned Vere4.6 image and Recommended Apps pill
content remain as in RC.13. Stable0.1.0 remains gated on the upgrade policy.

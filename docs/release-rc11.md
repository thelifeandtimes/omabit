# Omabit 0.1.0-rc.11 — simpler ship boot defaults

## Changes

- Comet, fake and owned-key boot forms put loom/storage choices under collapsed
  **Advanced settings**. Defaults remain loom31 (2GiB) and `/srv/urbit/piers`.
- New loom selections are restricted to30–34 (1–16GiB) in the desktop, collector
  CLI and runner's registration/configuration endpoints.
- Native `conn.sock` control defaults off and is configurable only through the
  CLI. Desktop creation uses that default; desktop edits omit the option and
  preserve any existing CLI choice. Native diagnostics remain read-only.
- Owned-key boot includes a **Get an Urbit ID** link to
  <https://urbit.org/overview/running-urbit/get-urbit-id>.
- The loaded-QML status version is checked against the release manifest.

## In-place compatibility

Schema19 and the runtime/engine/pill pins are retained. Existing small-loom
configurations and launch records remain readable and manageable; historical
creation-request replays and signed transfers keep their original contracts.
The update does not clamp saved values or replace running containers. New
configuration choices must use30–34.

This release is intended for an in-place management/desktop update, preserving
ships, identities, settings, storage and operation history. Fresh-install resets
are development exercises, not an upgrade procedure.

## Verification

Full Go tests/vet and command builds, targeted stateful race tests, JavaScript
models, and mapped QML through both desktop entry points passed. Tests verify
collapsed defaults, all five loom choices, the guidance link, CLI-only native
controls, absence of native-control mutations from desktop commands, historical
small-loom lifecycle/replay and current request-bound refusals.

Native control retains the pinned Vere build's documented pending-reply disconnect
bug and CLI warning. The new forms default it off. No new runtime-image qualification
is claimed by these UI/policy changes; RC.10's runtime evidence remains separately
recorded.

# Omabit 0.1.0-rc.10 — configurable loom and native control

RC.10 integrates [PR #4](https://github.com/thelifeandtimes/omabit/pull/4) by
**wundrellama (~dinnyt-divsud)**, with compatibility and release-tooling fixes.

## Changes

- Loom exponents20–34, with labeled address-space sizes; default31 (2GiB).
  The separate 4GiB container memory cap and existing admission rules still apply.
- Explicit default-off native `conn.sock` access for every ship class, with a
  fixed private-permissions launcher and full-authority/development-use warning.
- Requested settings remain distinct from recorded container launch settings.
  Saving configuration leaves the current launch intact; explicit Stop/Start
  applies the selected settings.
- HTTP-only edits from older collectors preserve native opt-in. New collectors
  preserve omitted `--native-control`; explicit `=false` disables it.
- Cross-host moves preserve loom/native settings in signed transfer commitments.
  Nondefault settings require `runtime-settings-handoff` on both runners before
  source fencing. Default/historical transfers retain their original contract.
- Schema19 transactionally upgrades schemas11–18 while preserving records, stable
  identity and replay. Earlier binaries refuse the new schema. Packaging reads
  the actual schema from the built runner, and release tests verify consistency.
- Runtime-settings JavaScript regressions now run in CI.

## Qualification and upgrade scope

Go tests/vet, stateful race checks, JavaScript models, mapped QML through both entry
points, and installer/bootstrap checks exercise the integrated source. Regression
fixtures cover legacy-client edits, explicit disable, settings-preserving moves,
capability refusal before mutation, signed-setting tampering, repeated migrations,
failed-migration rollback and retained import-recovery fences.

The versioned release's `qualification.json` records final-binary runtime and
management-update observations. The contributed earlier fake-only observations
remain separately documented in [runtime validation](runtime-settings-validation.md).
Stable0.1.0 remains gated by the broader [upgrade policy](upgrade-policy.md).

Pinned Vere `4.6-5a7614f` still has the known crash when a native client disconnects
with replies pending. Native control is intended primarily for development; keep
it off for routine management and consume complete replies before disconnecting.
The runtime image, engine and supplied-pill pins remain the qualified RC.9 values.

## Fresh installation

```bash
omarchy plugin install https://github.com/thelifeandtimes/omabit.git --enable
```

Then open **~+ → Setup → Set up this computer**. Matching runner, collector and QML
are required for the new controls. Existing installations upgrade in place;
fresh-install retirement is a development exercise, not an upgrade procedure.

# Omabit 0.6.0-rc.4 manual review

The rc.4 candidate addresses the feedback from the persistent review040 DEV
environment. Product source: `27aac5001c43a7a1148df47dc9904bdeb1a4a536`;
runner schema67. All linked qualification gates pass and the DEV update is staged. This is a prerelease. Stable/latest remains0.5.0.

The immutable [rc.4 prerelease](https://github.com/thelifeandtimes/omabit/releases/tag/v0.6.0-rc.4)
is published and all nine public downloads match the qualified bytes. The distribution
checkout is `cadeb8930373ba41db5dc8eff9a8a01e6d71ba23`. The source tag predates these
bootstrap pins; do not install that tag as the native plugin. Publication does not
update an installed service. [Publication evidence](diagnostics/2026-11-25-runtime-feedback-publication.json).

## Update this DEV environment

Use the separate [owner update guide](review040-update.md). The old rc.3 update
is complete and must not be repeated. The rc.4 command is:

```bash
cd /home/thelifeandtimes/repos/omabit
python3 -B scripts/qualification/runtime-profiles/update_review040_rc4.py --check
python3 -B scripts/qualification/runtime-profiles/update_review040_rc4.py --apply
```

Run as the existing owner, without sudo. Require `"status": "complete"`, then:

```bash
omarchy shell thelifeandtimes.omabit.review040.review status
```

Expected: version0.6.0-rc.4, profile review-040, loaded true, local_setup false,
with helper paths in the versioned review040-v060rc4 plugin. Existing enrollment,
piers, runtime pins, stopped/running intent and history remain. The assistant does
not apply this command or perform live recovery on the owner's behalf.

## Focused review

1. Select **review-zod**, then **Stop** to reconcile its retained network endpoint.
   Require a terminal successful Stop, stopped state and cleared recovery flag;
   let another refresh complete to ensure it stays stopped. This operation does
   not recreate its pier. If it remains unknown, retain the operation's diagnostic;
   do not reset/reimport it or manually remove network resources.
2. On an ordinary disposable ship, press Stop and Start once each. The button
   should immediately show **Stopping…** or **Starting…**, stay disabled through
   submission and polling, and return only when its matching operation finishes.
   Pack/Meld/Roll use the same pending behavior.
3. Run Pack for the **zod** instance. Observe **release050-app-zod** separately.
   Its status should no longer turn unknown because an unrelated Pack container
   appears or disappears. These instances have distinct runtime identities despite
   sharing the fake Urbit identity ~zod.
4. Open the permission map. Each fake's card should display its **instance ID**,
   with its friendly label if different. A stopped expanded project member should
   retain its permission scope. Ordinary blocked permissions are still blocked.
5. Open **Set up this computer** in DEV. It should explain the separate development
   environment and offer **Add a ship**, without an indefinitely checking installer.
   The production host installer remains disabled in this review profile.
6. On the intended disposable runner, review all three runtime workflows: choose a
   runtime when booting a fresh fake; choose one before previewing/importing a stopped
   disposable pier; and Stop an enrolled ordinary ship, choose its new runtime,
   confirm the change, then Start explicitly. Confirm the requested and observed
   runtime details match. Selection alone must leave the ship stopped.

Project members/private-chain factories retain their separate runtime contract;
the new enrolled-ship selector applies to ordinary managed ships.

## Local builds and channels

[Runtime policy](runtime-policy.md) documents packaging a local Linux x86-64 Vere
executable, installing its manifest/archive as the runner owner and selecting its
immutable ID. The CLI supports an owner-declared loom limit up to40; the ordinary
UI offers30..34 and preserves a larger existing CLI value. Actual64-bit execution
and latest upstream channel downloads are not qualified by this candidate.

Live/edge/dev refer to the latest **installed** channel artifact on that runner.
Installing a newer channel build does not change a ship's saved pin. The upstream
latest endpoints returned HTTP403 during this work, so the UI does not claim an
unverified download is available. Local patched executables remain usable through
the owner packaging/install path. Compatibility declarations are owner assertions;
an incompatible downgrade from a required JSON correction is refused.

## Evidence

- [Frozen package and provenance](diagnostics/2026-11-25-runtime-feedback-build.json).
- [Source, all race shards, QML and reproducibility](diagnostics/2026-11-25-runtime-feedback-ci.json).
- [Bundled runtime:14 checks and16 container launches](diagnostics/2026-11-25-runtime-feedback-bundled-native.json).
- [Local-build fresh/switch/import:5 checks and8 launches](diagnostics/2026-11-25-runtime-feedback-local-native.json).
- [Direct0.5.0→rc.4 upgrade](diagnostics/2026-11-25-runtime-feedback-stable-upgrade.json) and
  [rc.3→rc.4 upgrade](diagnostics/2026-11-25-runtime-feedback-rc3-upgrade.json): real native
  authorization denial/retry, schema migration, older-writer refusal, retained
  running containers and original ship/application/access history.
- [Full isolated Omarchy shell](diagnostics/2026-11-25-runtime-feedback-full-shell.json).
- [Exact DEV wrapper and prepared owner update](diagnostics/2026-11-25-review040-rc4-prepared.json).


The original rc.3 artifacts and completed update receipts are retained. No protected
ship, tmux ship, ~/temp-ur content or production service was used for qualification.

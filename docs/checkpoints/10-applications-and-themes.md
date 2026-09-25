# Checkpoint 5B: application sessions, themes, and notifications

Application sign-in, status, logout, and forgetting are now available in Omabit.
UI selections carry immutable enrollment IDs; login codes are handed to the
collector only through stdin and cleared from the form/process handoff. Codes are
never queued or stored in plugin settings. Runtime management remains separate.

The Omabit-specific `omabit` executable owns palette resolution and Talon theme
policy. Generic `urbit-collector` still owns enrollment/credentials and generic
Eyre transport; `urbit-runner` has no desktop behavior.

## Automated evidence

- Real QML components exercised runtime dispatch and secret stdin handoff using
  isolated helper fixtures. Stale enrollment selection is rejected before reading
  a login code.
- Both credential adapters passed CLI process-level validation on the running
  `panel-check` fake at port **18085**.
- Real Eyre publication on that disposable fake updated Talon's theme/accent
  entries, received Gall acknowledgements, verified readback, and deleted owned
  channels. The test's HTTP session was logged out afterward. The fake retains the
  published Omarchy appearance; this was not a test against a production ship.
- `scripts/validate-theme.py --acknowledge-development-fake --port 18085` also
  passed the actual CLI enable/sync, unchanged read-only check, explicit resend,
  pause, and logout/credential cleanup using temporary collector/preferences state.
- Fixtures verified preservation of unrelated themes/fields, large numeric fields,
  rejection of duplicate/malformed settings, and refusal to treat matching old
  state as proof of an unacknowledged resend.
- Migration fixtures verified pause-before-transfer, origin/identity-bound cookie
  transfer without logging out the shared session, preserved sync preferences and
  fingerprints, and resumable cleanup failure.
- Notifications suppress initial historical state and repeated identical failures.
  Normal-urgency `notify-send` lets the desktop notification service enforce DND.

## Application UI

After updating the installed snapshot:

```bash
python3 -B scripts/install-panel.py --replace --enable
```

Open `~+`, choose **Add ship**, and enroll an application connection. For the
current development target use a fresh name, `http://127.0.0.1:18085`, and `~zod`.
Then select **Sign in** on that row. The development fake's public tutorial code
is `lidlut-tabwed-pillex-ridrup`; enter any real ship's code privately in the masked
field, never in chat. **Check** performs a live authentication check.

**Sync theme** explicitly enables publication of the current Omarchy palette to
all Talon clients for that ship, including disabling Talon's separate accent
override. **Pause theme** preserves the ship's last appearance and stops future
automatic publication. **Publish now** resends both entries with acknowledgements.
An already accepted publication can finish after Pause. No automatic loom change,
ship lifecycle operation, or app installation is part of theme synchronization.

Automatic sync checks every 30 seconds. Failed publications have a three-attempt
budget per palette/explicit enable intent; another ship's failure does not block
others. Changed palettes reset the budget. Unchanged passive checks are read-only.
Authentication rejection pauses that row. Reauthenticate, then enable sync again.

Readable commands from the checkout:

```bash
./build/urbit-collector --format text ships
./build/omabit --format text theme-preview
./build/omabit --format text theme-status
```

## Explicit legacy-theme handoff — decision required

The existing Urbit Theme plugin currently owns a real `~sarlev` connection. Its
live credentials and publication preferences have **not** been migrated by the
automated development tests. Preview the actual current state:

```bash
./build/omabit --format text theme-migrate
```

The preview performs no keyring lookup or ship request. It reports origins,
automatic/pending preferences, conflicts, and a source digest. To authorize the
handoff, copy that digest into this command:

```bash
./build/omabit --format text theme-migrate \
  --apply --expected-digest DIGEST_FROM_THE_PREVIEW
```

Apply disables the old plugin, locks and pauses its publisher, preserves original
nonsecret state in the Omabit migration journal, and imports each session into the
collector's Secret Service namespace. It checks origin and ship identity before
using the imported session. It clears the old keyring item **without logout**,
since logging out the shared token would invalidate the transferred session.
Original sync/pause, pending, and fingerprint state are retained. Incomplete
migrations suppress publication on imported rows and can resume using the same
original digest; `theme-status` displays that digest.

If a saved legacy session has expired, sign in to the newly created collector row,
then resume the same migration. The legacy schema has no expiry field, so imported
sessions receive a conservative local one-day deadline after live verification.
There is no silent origin rewrite; incompatible legacy URLs or collisions require
review. Version-1 legacy state must first be upgraded with its existing helper.

The handoff is forward-only tooling. The original nonsecret state is retained for
review, but an automatic rollback of keyring ownership is not claimed. To return
to the old plugin, pause Omabit publication and deliberately reauthenticate there.

## Remaining feedback

Application form usability and theme rendering in your actual Talon clients are
useful user checks. Backend storage acknowledgement does not prove that every
client rendered an update. The next runner milestone is bounded offline pack;
its root-owned binary upgrade requires administrative authorization.

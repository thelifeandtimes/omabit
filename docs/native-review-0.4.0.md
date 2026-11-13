# Live 0.4.0 manual review

Open **Omabit 0.4.0 DEV** in the application launcher, or click **DEV** in the
menubar. The full panel is also available with:

```sh
omarchy-shell thelifeandtimes.omabit.review040.monitor open
```

Select `review-local → ~zod` (`review-zod`). This is a fresh fake ship on a real,
separate rootless Docker engine. Its initial 4 GiB/2 CPU boot, stop, change to
16 GiB/4 CPUs and healthy restart passed. The pier was retained, and Docker's
actual memory/quota settings were verified. This checks configuration, not a
16 GiB stress workload. It is left running for manual review.

- On Overview, open the runtime configuration editor. Stop before changing RAM
  or CPU, reload settings, Save, then Start. There is no header resource shortcut.
- New standalone fake, comet and owned-key ship forms expose RAM/CPU under
  Advanced settings. Project members retain their separate fixed contract.
- Each ship has a **Ship URLs** tab. Setup no longer has a fleet-wide URL editor.
  Bulk edits use the agent CLI with a separate verified binding/revision per ship.
- **Agent Help** opens the configured Omarchy agent; this machine reports Opencode.
  Bundled instructions and private review profile are passed to it. No agent was
  launched automatically. Its launcher lifetime was fixed in M576: the agent
  session outlives the helper and panel. Skill installation remains available for manual review.
- Setup → App development and publishing uses the real runner's policy. New
  permissions start denied; previous simulated policy choices are preserved in
  the old preview's data, not applied to real ships.

This is unqualified development, not a published release. The inherited product
version badge still says0.3.7; REVIEW-040, the DEV launcher and development channel
identify this0.4.0 feature snapshot.

## Isolation and persistence

| Resource | Review location |
| --- | --- |
| Plugin | `~/.config/omarchy/plugins/thelifeandtimes.omabit.review040` |
| Runner ID | `runner-a9a208a18445179f05df5644ed6d245c` |
| Dedicated engine | `omabit-dev-040-docker.service` (user service) |
| Dedicated runner | `omabit-dev-040-runner.service` (user service) |
| Runner state, piers, tools | `/srv/dev/urbit-040/{state,piers,bin}` |
| Collector, preferences, receipts | `/srv/dev/urbit-040/desktop` |
| Engine socket | `/run/user/1000/omabit-dev-040/urbit-docker.sock` |
| Management socket | `/run/user/1000/omabit-dev-040/urbit-runner/runner.sock` |

Both new units are enabled for the user's session. Local host installation is
blocked in this review plugin; it already has its own local runner. Remote setup
remains available for manually authorized pairing. No rooftop enrollment was
copied into the new profile and no remote runner/key was changed or revoked.

The local TRIAL plugin and all its profile records were retired, preserving
permissions and credentials, under
`~/.local/share/omabit-review-040-retired-20261112`. The failed first DEV plugin copy
is also archived there. The former simulated preview is closed; its files and
saved choices remain under `~/.local/share/omabit-policy-preview-040`. Its desktop
launcher now opens live DEV. Production plugins/runners and earlier `/srv/dev/urbit`
review data were preserved.

Do not treat this human review state as a disposable automated fixture. For any
future update, stage complete plugin contents outside the watched plugin tree and
atomically install a fresh versioned directory. Quickshell can retain a failed
external module's type/file-name cache; rescanning its unchanged path was
insufficient. Do not restart the production shell to work around that cache.

Receipts: `desktop/installation.json`, `runtime-validation.json`,
`ship-urls-validation.json`, `review.png`. Installation records the immutable build
snapshot plus exact QML-registry and bundled-reference overlays. Runtime Go inputs
match the compiled build. Sources/tests are integrated on the0.4.0 feature branch;
master and release publication remain untouched.

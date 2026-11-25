# Omabit 0.6.0

0.6.0 adds per-ship runtime selection and the lifecycle and display improvements
reviewed in RC4, with Vere selection grouped inside Runtime Settings.

Choose the Vere runtime when booting a fresh ordinary ship, importing a stopped
pier, or changing an enrolled ship while it is stopped. Runtime selection on the
ship page now lives inside **Runtime Settings**.

- Runtime pins are immutable. Installing newer live/edge/dev channel builds does
  not move an existing ship to another version. The UI offers installed builds;
  the CLI also packages and installs arbitrary owner-provided Linux x86-64 builds.
- Runtime changes retain the pier and require explicit confirmation. Start remains
  a separate action. Local manifests bind executable/image/archive hashes and
  declare compatibility and loom limits; declarations are not proof of compatibility.
- Stop can recover a precisely identified orphaned endpoint left after removal of
  its recorded container. Recovery preserves instance identity and operation history.
- Lifecycle buttons show pending actions immediately. Unrelated Pack activity no
  longer changes another project's displayed status. Permission cards include
  instance names, and stopped project members retain their saved permission scope.

## Upgrading

The qualified stable paths are **0.4.0/schema59 → 0.6.0/schema67** and
**0.5.0/schema65 → 0.6.0/schema67** for Linux/AMD64 local owner-user installations. The [qualification record](qualification.json)
binds the exact release artifacts and upgrade rehearsals from [0.4.0](rehearsal-0.4.0.json)
and [0.5.0](rehearsal.json). Existing identities, piers, enrollments,
permissions, configuration and history are preserved; runtime pins do not change
automatically. Earlier stable installations follow their documented path to 0.4.0.

For an unmodified stable Omarchy Git installation:

```sh
omarchy plugin update thelifeandtimes.omabit
```

Then open **Setup → Local services & updates → Check services → Update services**
and authorize the OS prompt. The desktop and service updates are separate steps.
Finish pending operations first. Keep pier backups and preserve the runner's
runtime-profiles directory together with its management state. Older runners
refuse schema67; do not lower schema markers or restore old binaries alone.

## Limits

Project factories retain their separate official-runtime contract. Latest upstream
channel downloads and actual 64-bit Vere execution are not qualified by this release.
An owner-supplied build must be validated for its intended pier;
runtime changes can affect replay and are not necessarily reversible. Remote
service upgrades, active recording-worker upgrades and power-loss durability are
outside the qualified local upgrade support window. Existing 0.5.0 feature limits
continue to apply. Publication does not perform a production update.

## Verification

The exact frozen source is `fcf269ca8aa20aa9e43a70d7e282a91f12659cf9`.
Qualification includes package checks/provenance, full source/race/QML CI, native
desktop cold/hot activation, direct 0.4.0 and 0.5.0 service upgrades with preserved original
application history, 14 bundled-runtime checks and 5 local-build checks covering
24 actual container launches. Local-build execution uses the known reviewed
executable packaged as a separate content-addressed profile. The source tag binds
artifact provenance; install the stable native plugin through the commands above,
which select the distribution commit and its final artifact pins.

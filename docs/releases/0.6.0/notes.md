# Omabit 0.6.0

Release preparation is in progress. Publication follows exact-artifact qualification,
review and a protected merge; this document does not mark those gates complete.

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

The intended qualified stable path is **0.5.0/schema65 → 0.6.0/schema67** for
Linux/AMD64 local owner-user installations. Qualification must bind the final
0.6.0 artifacts before publication. Existing identities, piers, enrollments,
permissions, configuration and history are preserved; runtime pins do not change
automatically. Earlier stable installations follow their documented path to0.5.0.

For an unmodified stable Omarchy Git installation, after publication:

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
channel downloads and actual64-bit Vere execution were not established by the RC
qualification. An owner-supplied build must be validated for its intended pier;
runtime changes can affect replay and are not necessarily reversible. Remote
service upgrades, active recording-worker upgrades and power-loss durability are
outside the proposed local upgrade support window. Existing 0.5.0 feature limits
continue to apply. Publication does not perform a production update.

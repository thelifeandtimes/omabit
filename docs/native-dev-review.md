# Native DEV review

Use visibly labelled development surfaces with independent state. A DEV label
alone does not isolate credentials, runner hosts or ship data.

The supported repository workflow builds an immutable UI/tool bundle, stages it
in a named desktop profile and leaves activation to the operator. Follow
[daily development](desktop-development.md#daily-development). Verify the actual
loaded version, development label, digest and profile after selection; the Git
checkout or selected version alone does not identify the running application.

## Review boundaries

- Desktop profiles isolate configuration, state, cache and enrollment. They do
  not isolate host runner services, Docker engines or piers.
- Use a separately provisioned disposable runner for automated runtime tests.
  Preserve human-review state across iterations; it is not an automated fixture.
- Updating desktop/management code and upgrading the runtime are distinct actions.
  Follow the [management upgrade contract](upgrade-policy.md) and verify that an
  intended manager-only upgrade preserves running container processes.
- Theme and DMs use their own repositories and independent sessions. Review their
  actual loaded versions and connection identities separately. Do not assume an
  Omabit candidate automatically updates either plugin.
- Selecting a known-good desktop is not a runner-database downgrade. Retain failed
  candidate evidence and follow supported recovery paths.

## Existing static DEV installations

Some experimental installations predate the retained bootstrap and use copied QML
plus fixed development wrappers. They do not follow `omabit-dev.py select`.
Preserve those namespaces, sessions and wrappers until an explicit migration
verifies their configuration/state/cache/IPC/keyring isolation and runner boundary.
Do not point them at production tool wrappers to make a candidate appear loaded.

A common installer for separately namespaced Omabit, Theme and DMs DEV plugins is
not yet implemented. Profile staging and isolated QML tests are available now;
refreshing those legacy installations still needs a reviewed migration. Personal
inventories, exact paths and historical manual-review receipts belong in private
operator notes rather than this public guide.

# Installed declarations and explicit approval flow

2026-10-21, unreleased and uncommitted in the isolated development worktrees. See
the [source and evidence hashes](2026-10-21-approval-flow.json) and
[user/CLI/declaration contract](../application-connections.md).

## Demonstrated

- Installed Theme/DMs capability declarations identify a complete Python helper
  inventory. Discovery executes no helper; invalid declarations remain unavailable.
  Profile metadata, manifest and declaration are part of the approved digest.
- Approval CLI binds ship ID/revision, plugin/profile/digest, request ID and
  explicit ordinary owner-session authority. Duplicate approval issues once;
  stale selections are refused. Cleanup checks revision under the attempt lock.
- CLI fixture qualification uses an HTTP fixture and real Python child launch,
  including an installed-source update before retained-generation reconciliation.
  The parent login survives the child disconnect. Output contains no credential
  or credential-store reference, including failure output.
- Failure before child login begins is now `not-issued`; an expired parent does
  not strand an unissued session. Login-response uncertainty remains unresolved.
- The actual QML pane and Quickshell Process run offscreen with private HOME/XDG,
  no inherited display/DBus/Hyprland settings, inert style values and a fixture
  backend. Explicit sign-in → consent → approval → failed disconnect → manual
  recovery → receiver reconciliation passes. Only those five mutations appear in
  the fixture audit, and the login code is absent from argv. Changing selections
  clears consent; hidden panes clear the code. An existing active connection
  disables another approval for that plugin/ship.
- Real Theme/DMs declared inventories load under separate `.auth-lab` identities
  with invalid offers, without any HTTP/keyring work or account-state writes.

Final UI evidence is `/tmp/omabit-app-ui-20261021-04`. Its QML log has no binding or
type warnings and its approval screenshot was inspected. Earlier runs exposed a
fixture module-registration omission, an invalid screenshot target, and an
unqualified `Color` name resolving ambiguously under Quickshell. The source now
qualifies `Commons.Color` explicitly, preserving its palette values.

Race tests passed for applicationauth, collection, eyre and the collector CLI;
static checks cover the affected packages. Theme recovery/installer tests passed
26 tests, including compatibility with the previous installed file inventory.
Both actual declared inventories passed their isolated import check.

## Boundary and next gate

This is not a complete new-path real-ship/UI claim. The earlier real Theme/DMs
[receiver recovery checkpoint](2026-10-21-receiver-recovery.md) predates the public
CLI/declaration path. Next run that exact new path with both real receivers on a
fresh fake, then qualify loaded plugin UI/wakeup with an isolated desktop.
Completed-record/cache retention, reconnect/data writes and full profile switching
remain. Issuer and receiver journals still cap attempts at 64; uncertain entries
and their retained helpers must not be discarded.

No installed plugin, production keyring, active desktop, Docker daemon, runner,
managed `~lismyl-lismyl`, tmux ship or `~/temp-ur` ship was inspected or changed.
This checkpoint started no Urbit runtime. Marketplace/backend installation was not
added; all ongoing plugin traffic remains direct and standalone login remains.

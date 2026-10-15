# RC.3 owner-user implementation qualification

The selected model runs the runner and separate rootless Docker engine as the
existing pier-owning Unix user. Local development qualification used sourceport,
UID/GID 1000, default `/srv/urbit/piers/`, schema16 and runner
`runner-17db7b66a45d3b21badf17811840ec9d`. Sourcebox's fresh native installation is
reserved for the user. Prior dedicated-account retirement copies remain preserved.

## Live development gates

- Authorized fresh owner setup completed, including restricted management SSH and
  identity-pinned collector enrollment. An initial legacy engine-guard substitution
  error was corrected and covered by a real-template regression test; the partial
  installation was resumed without replacing its owner or rotating identities.
- Default-root fake `rc3-default` (`~zod`) and bare-bootstrap custom-root comet
  `rc3-custom` both initialized and became HTTP healthy.
- The custom root was selected through a shortcut to a mode0775 user workspace,
  including spaces and missing child directories. Its canonical directory was
  pinned. No privileged filesystem preparation was needed there.
- Both ships passed the same hood generators used by `|mount %base`, ordinary
  nonroot `cp -R`, directory creation and atomic editor replacement, `|commit`,
  unmount/remount and committed-content readback. Container-private Lens invoked
  those generators; conn.sock remained disabled. No ACL or ownership repair occurred.
- Both then passed clean Stop → Pack → Start and direct Clay content readback.
  Real UID maps confirmed container UID0 → host UID1000; dropped capabilities,
  no privilege flag and 4GiB memory limit remained. Starts used initialized-pier
  commands (`http-restart` / `comet-resume`), preserving ship identities/revisions.
- The installed root-owned absolute `owner-ssh` forced-command endpoint passed
  two-operator actor attribution, actor-scoped replay, observer read/write refusal,
  arbitrary-command/forwarding refusal and revocation of an existing session.
  The independent operator still worked. Temporary uninitialized registrations
  were deleted and the exact original grants restored.

Private receipts are in `build/rc3-qualification/`, notably
`default-desk-confirmed.json`, `custom-desk-ready.json`,
`{default,custom}-lifecycle-confirmed.json` and `owner-ssh-live-confirmed.json`.
They are not committed because nearby setup state contains private credentials.

## Harness findings and limits

- First start was correctly refused below 5 GiB available RAM. The user freed memory;
  a separately requested start passed. Resource thresholds were not reduced.
- The first fake test incorrectly sent a greeting poke instead of dispatching a
  hood generator. That harness was corrected. Clay's `txt` mark normalized the
  final newline, confirmed by direct readback and fresh export.
- The comet's first mount request timed out while first-boot app/Azimuth work was
  active. Later the ship was responsive and the mount had completed. That export
  was inspected/unmounted before a fresh full test, which passed. Logs included
  upstream HTTP504/retry errors; readiness does not establish full app/OTA health.
- Preliminary lifecycle assertions used the wrong container-kind names; corrected
  assertions verified the actual initialized commands and persisted content.
- The ad-hoc SSH harness initially omitted the protocol header, then needed a
  separate revocation synchronization signal. Failed receipts were retained; the
  final test passed through the installed service with original grants restored.

## Source regression gates

Full Go tests/vet and race checks for managed/runtime/transport/access/collection,
copy, host installer and pier storage passed with Go1.25.0. The opt-in real OpenSSH
fixture passed two operators, observer restrictions, identity/host-key mismatch,
replay/reconnect and revocation. Desktop model, snapshot installer, pinned native
bootstrap and mapped native-wrapper QML tests passed.

Owner-specific fixtures cover preserved account/groups/grants, missing subordinate
range allocation, conflicting legacy/owner/default-root/user-unit refusal, the real
engine guard, shortcut canonicalization, ordinary group-writable desk content,
private state checks and installed-identity boundaries.

This record describes development-build gates. The release's `qualification.json`
records exact frozen-binary update/artifact checks separately. No owner-user host
reboot or logout test is implied by prior dedicated-account reboot acceptance.

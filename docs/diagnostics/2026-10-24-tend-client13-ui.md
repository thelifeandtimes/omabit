# Standalone Tend backend transport and setup card

A separate Tend worktree at `/home/thelifeandtimes/repos/tend-backend-install`
(branch `codex/tend-backend-install`, base bd99c33) now preserves backend-free
plugin authentication, inventories Clay desks and all pending hood sources,
shows the canonical publisher, and sends an explicit direct Eyre install.
A per-connection locked/fsynced intent precedes dispatch; uncertain operations
are not replayed. Existing publishers and incompatible backends are preserved.
CLI login keeps its existing strict compatibility contract.

Thirteen new regression tests pass, followed by all138 Python tests, JS tests,
plugin validation and QML lint. The later extracted card/service pass a separate
real offscreen process test against synthetic Eyre: login, visible install action,
one exact canonical install, acknowledgement distinct from readiness, and Check
again to ready. Screenshot inspected. First UI run failed: QtQuick.Color shadowed
the Commons singleton and window screenshot capture lacked an engine. Explicit
Commons.Color and capturing the actual card resolve those failures. No installed
shell, theme or plugin was modified; fixture label/state are separate.

Native13's actual frozen transport correctly observes missing on fresh ~nec,
installs from a test-only ~zod override, records/deletes its receipt channel,
refuses duplicate dispatch, and returns ready with the same session after restart.
Seven transport cases and three clean shutdowns pass. The outer harness fails
because it still expects two direct-fixture install records, while the receiver
now records its receipt through the actual client. This is preserved as an overall
failure. Fresh14 will integrate the actual-client receipt into the overall check
and trace its commands, without weakening acknowledgement/readiness requirements.

M250's underlying protocol pass remains valid. This advances the standalone UI/
transport gate but does not claim release packaging, a complete native desktop,
public publisher availability, general source migration, Grubbery, or Omabit auth
integration for Tend. Independent copies/hashes of stopped data are retained;
input secrets, raw keyed logs and local cookie files are excluded.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-tend-client13-ui`.

# DMs explicit transport drain

DMs now has an opt-in QML/helper shutdown-transport path. It pauses new commands,
lets existing executor work finish, requires no live receipt waiters, deletes the
exact bound channel, invalidates that generation, and joins listener threads.
A failed deletion preserves the binding and reports blocked; repeated drain can
retry cleanup. Missing listener exit or unreadable outbound history cannot report
closure. Sessions and durable send records remain. The receipt always states
collectionReady=false because invitations/read markers and other effects are not
yet part of a complete transition contract.

All92 Python tests pass, including six new cleanup/admission/failure checks. The
real private QML/helpers/keyring fixture drains DMs before each stable→dev→stable
replacement. Three exact DMs channels are observed deleted and their helpers/
streams gone before replacements. The unknown send remains in stable, is absent
in dev, returns with the same journal, and is never resent. Final shell exits0;
source hashes and exact inactive scopes independently verify. Three Theme channels
remain undeleted, so M246 is superseded only for the opt-in DMs transport path.

This extends M253 without claiming automatic drain on arbitrary destruction,
real Gall cleanup, complete application quiescence or a collection coordinator.
The next corresponding prerequisite is Theme subscription/worker cleanup. No
installed plugin, host desktop/keyring or production ship changed.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-dms-transport-drain`.

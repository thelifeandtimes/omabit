# Native collection admission

The new explicit format1 permit store binds one activation to exact shell PID,
start time, boot and PID namespace, plus the three supported artifact/profile
members. Admission is sealed and each guard retains a one-use durable role claim.
Missing, unknown, closed, damaged, dead-owner or already-consumed state never
bootstraps/reopens itself. Exact lost-response retry is allowed only by the same
live claim owner and instance. Closing admission does not certify application,
channel or descendant closure. A deliberate later activation needs a fresh root
after coordinator recovery of prior effects; cross-root ownership is not yet wired.

An inert QML host starts no application QML until its direct-child guard admits
the exact shell/member. It freezes the selected metadata, checks the loaded URL
and identity, and fences/drains on guard loss without auto-replacement. The private
complete native registry and copied stock enable/service/panel loaders pass five
cases: enabled without permit creates zero apps; valid issuance creates three;
guard SIGKILL followed by native disable/re-enable refuses replacement; a new shell
with stale enabled configuration cannot consume an unused old-shell permit; explicit
new-shell activation creates three more. All six recorded guards exit. These are
synthetic app bodies and fixture configuration, not full native startup persistence.

All78 private tests pass, including process deaths, lost fsync responses, failed
writes, unknown/missing/linked state, wrong parent and repeated claims. Trial01
failed an invalid absolute QML import; its stock panel error path also logged an
undefined errorString. Trial02 completed admission scenarios but failed log checks
because synthetic services lacked settings. Correcting the fixture property gives
a clean final pass. All four scopes are inactive; source hashes independently match.

This supersedes M286/M287's missing durable admission prerequisite only. The test
host still lives outside artifact format1; immutable host/helper packaging, real
supported app/widgets, coordinator recovery, all in-flight constructors/drains and
external effects remain. Closing new claims alone is not a constructor/transport
barrier. Same-user locks/hash checks are not wallet key isolation. No production
plugins, services, ships, tmux sessions or temp-ur data were accessed or changed.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-native-admission`.

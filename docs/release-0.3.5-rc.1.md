# Omabit 0.3.5-rc.1 — idle-performance review candidate

This is a local review candidate, not a published stable release. UI, omabit,
urbit-collector and urbit-runner must come from the same frozen archive.

The candidate reduces hidden polling to 30 seconds, keeps two-second updates
while a panel/monitor or pending action is active, and requests fresh data on
opening. It retains bounded authenticated transports, removes bootstrap process
churn and no-op history notifications, and batches runner snapshot history reads.
See [measured results](diagnostics/2026-11-01-idle-performance.md). No layout or
feature expansion is included. Battery-life improvement has not been measured.

The intended review upgrade is 0.3.4 to 0.3.5-rc.1. Schema 47, desktop state 1
and bootstrap API 1 remain unchanged. Older releases first follow their published
upgrade paths to 0.3.4. Remote runner upgrades remain explicit; desktop selection
does not automatically replace installed services. No downgrade or reset is
required, and no imported ship is started automatically.

Before offering installation, record exact artifact checks, the isolated actual
QML-button service upgrade, and native desktop activation/recovery results.
Production and existing human DEV state remain outside the rehearsal.

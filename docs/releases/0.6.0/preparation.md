# 0.6.0 release preparation

The owner authorized a stable release on2026-11-25 and will update production
personally. The release includes qualified rc.4 functionality plus the reviewed
Runtime Settings layout. The owner completed the separate local UI1 update;
its receipt preserves runner/engine identity and confirms the new widget loaded.

Pending gates: freeze stable product source; build exact artifacts; run source,
race, QML and package checks; qualify the direct0.5.0 upgrade and runtime workflows
with disposable ships; verify native desktop activation; verify draft downloads;
record immutable distribution pins and qualification; pass final PR checks and
review; merge through protected master; publish and verify public downloads.

Earlier rc.4 evidence is supporting history, not qualification of the new version.
The reboot removed temporary assets, so required inputs must be restored from
verified public assets or retained immutable sources. Keep the48GiB cap, one
runtime VM and4GiB scratch admission. Production and human-review DEV state remain
untouched. Preserve all failure/recovery receipts from the completed owner update.

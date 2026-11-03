# Omabit 0.3.5 — lower idle CPU and SSH overhead

Omabit keeps a bounded pool of three authenticated SSH transports per runner,
observes every two seconds while a panel/monitor or operation is active, and
uses a thirty-second interval when hidden. Opening a view requests an immediate
observation without replacing the collector. Identity checks, authorization,
revocation and mutation non-replay remain enforced.

The native bootstrap reuses its lease process for selection checks. Hidden
clocks and unchanged chart samples no longer cause unnecessary updates. Runner
snapshots batch epoch-history reads without changing persistent state or history
selection. No visual controls, layouts, dependencies or schema versions change.

## Upgrade

Run `omarchy plugin update thelifeandtimes.omabit`, then open Omabit's menubar
and choose **Update Dependencies**. Review the offered local service update and
approve its OS authorization prompt. If needed, use **Setup → Local services &
updates → Check services → Update services**. Remote runners need their own
explicit update. No ship import or fleet reset is part of this upgrade.

After the service update reports **Local services match this desktop bundle**,
run `omarchy restart shell`. Stock plugin rescans retain Omabit's `keepLoaded`
service, so this reloads the new native bootstrap too. The bar briefly disappears
and returns; ship containers are not restarted by this shell command.

The supported patch path is 0.3.4 → 0.3.5. Earlier versions should follow the
documented intermediate upgrade paths to 0.3.4 first. Runner schema 47, desktop
state format 1 and bootstrap API 1 are unchanged. The desktop carries its matching
collector; updating the runner enables the snapshot improvements as well.

## Evidence and review

The performance pass measured 47 → 3 SSH sessions in a thirty-second isolated
OpenSSH fixture. Actual offscreen QML hidden main-thread CPU fell from 1.00338 to
0.06070 seconds over 32 seconds with 32 stopped fixture ships. Visible polling
stayed at two seconds. These are controlled CPU measurements, not battery-life
or full desktop compositor measurements. See the [performance report](diagnostics/2026-11-01-idle-performance.md).

The exact stable artifacts passed the isolated 0.3.4 → 0.3.5 QML-button upgrade,
native hot/cold activation and recovery, and package/provenance/CLI checks.
Protected source/race/QML/release-policy CI gates publication. Final evidence is
recorded in `docs/releases/0.3.5/qualification.json` and its linked rehearsal.
The service rehearsal uses fresh fake ships in an offline disposable guest;
production ships, installed services/plugins and human DEV state are excluded.

After updating, check the 0.3.5 version, open and close the menubar and monitor,
confirm immediate fresh observations when reopened, and compare idle CPU over
several minutes. Review your fleet read-only before choosing any ship operation.

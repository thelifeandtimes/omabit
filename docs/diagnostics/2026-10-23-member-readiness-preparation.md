# Serial cold boot and readiness preparation — 2026-10-23

The member fixture now waits for each exact public ship identity before booting
its peer, matching the earlier passing networking fixture. It uses no cookie,
proxy or redirects, caps the response body, rejects wrong identities, and records
attempt counts, last HTTP status and elapsed time. Five loopback HTTP cases cover
success, wrong identity, oversized response, redirect and unavailable endpoint.
They and readiness race regressions pass; vet and ordinary VM guards pass.

Each cold boot has a 900-second limit. The complete guarded operation has 2400
seconds; Go/guest/service/host limits of 3150/3210/3600/3900 leave bounded cleanup
time. The separate runtime observation/preflight freshness stays 30 seconds.
These bounds follow the earlier measured 405/585-second serial boots and keep
one CPU, no swap, no external NIC and the same memory/ownership isolation.

After both warmups, the real schema28 queue collects and persists each complete
readiness proof and checks replay/database reopen. Appliance15's sources,
artifacts and embedded test binary are independently verified. This is preparation
only; the next fresh trial must establish runtime readiness and clean shutdown.

# Private retirement queue and project admission — 2026-10-23

The private queue reconstructs the exact original launch/readiness/Stop/exit chain,
requires the adapter to return the accepted retirement unchanged, and runs behind
the common work/update/maintenance gates. Accepted active project work blocks a new
retirement. An unresolved retirement blocks new lifecycle/observation requests in
that same project, with a second check under the worker gate before any adapter
opens. Actor/request replay remains available, and unrelated projects are unaffected.
Cancellation and setup failure retain the accepted intent without replaying effects.

Seven queue/fence cases pass, along with all affected project lifecycle race suites,
historical migration fixtures and vet. Two added adapter cases verify an already
stopped peer and refusal of changed peer exit metadata, bringing that suite to21.
The previous model/persistence/runtime records retain their original scope. This
adds no format beyond schema31, public route, startup action or actual removal.
Real retirement, replacement/resume and later generation reconciliation remain.

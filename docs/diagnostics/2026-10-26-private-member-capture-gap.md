# Preserve capture before later running-state checks

M441, 2026-10-26. Source review identifies an unreleased recovery limitation,
independent of the unresolved M439 runtime failure.

`privateMemberLaunchDriver.ObserveRunning` captures the original pier/.urb objects
after acknowledged Start, then retains that evidence only in the driver's
`birthEvidence` field while it performs the full gateway and final process checks.
`ExecuteMemberLaunch` persists the `running` result only when this entire method
succeeds. On an intervening error or management loss, the durable launch remains
`started` with no birth identity. The in-memory capture does not survive driver
closure/restart. Source inspection cannot establish whether capture succeeded in
the failed VM; M440's diagnostic run is intended to locate that boundary.

M432 correctly persists birth identity for completed version2 running records, but
does not cover interruption between capture and later observation. Do not describe
that narrower guarantee as durable capture across every post-Start failure. This
clarifies M432 rather than invalidating its completed-record migration tests.

The next persistent lifecycle design should commit the independently validated
first-boot capture, attributed to its original launch/container/process, before
subsequent full gateway or HTTP checks. A separately attributed read-only recovery
may then consume that retained identity while preserving the interrupted intent.
It must never repeat Start, reopen the initial empty-mount capture after boot,
discover/adopt a current pier, or infer clean shutdown. A lost capture-commit reply
requires durable lookup; an absent capture remains unresolved.

This is a design requirement, not an implemented format or runtime fix. Keep the
current diagnostic binary/checks unchanged. Any successor format must preserve
schema45 history, use the frozen original45 writer, and demonstrate migration and
old-reader refusal. Test interruption at capture, commit/reply loss, later gateway
failure and management reopen before enabling the recovery path. Readiness and
second-member admission still require their own current observations.

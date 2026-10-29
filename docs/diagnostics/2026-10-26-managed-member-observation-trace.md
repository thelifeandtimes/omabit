# Logging-only post-start observation run

M440, 2026-10-26. A temporary overlay adds fixed return-location diagnostics to
six member/gateway observation files. Removing the injected logging calls recovers
the exact original source bytes. Each emitted record contains only a fixed source
location, package, sequence, elapsed milliseconds and context-expired boolean.
Emission requires the guest fixture environment, is bounded per package, and
includes no error text, path selections, key, application or pier payload.
The ordinary source tree, acceptance rules, freshness windows and runtime deadlines
are unchanged. The candidate runner is ordinary source; only the VM fixture uses
the logging overlay. Schema45 and the frozen original45 baseline are unchanged.

579 affected race checks pass without skips, plus vet and diff checks. The fully
armed test binary refuses host execution before tracing or effects. The separate
read-only auditor passes20 checks, rejects unknown fields/locations, omits raw
lines and retains at most the final128 valid records from each fixture log. Its
host guard also refuses. The diagnostic audit will never report normal runtime
qualification, even if the fixture itself completes successfully.

The fresh appliance is privately retained with byte verification and a compatibility
symlink. Four finished audit directories and three completed diagnostic build
directories were likewise retained; no runtime data was deleted and no storage
floor lowered. VM admission retains8GiB, with2GiB running reserve. The fresh
no-NIC VM has its own rootless Docker and no existing piers, using the same12GiB
host memory cap,7GiB guest memory,2CPU and no swap. Its one-shot supervisor is
bounded separately. Runtime and independent audit outcomes are pending.

Build SHA256: `96184d8f1cd2c43abe2abf3d3a953682120c62d2503b767f9937631b63368109`.
Fixture SHA256: `389577c68b4748a2929abb719125913e028c52f42f0cfca024020f6eb33b79fd`.
Auditor SHA256: `3e65462b80df0122257d64025ae10b53fa879a6f0f0f5ce0d43efc2ca65797f0`.

This advances M439's diagnostic preparation only. It does not resolve the first
post-start failure or qualify member readiness, Stop, recovery or application
state. M437/M438 remain source-tested and are not exercised by this fixture.
Never replay the previous interrupted launch. The next action is to finish this
fresh run, audit its stopped disk, and correct only the failure actually observed.

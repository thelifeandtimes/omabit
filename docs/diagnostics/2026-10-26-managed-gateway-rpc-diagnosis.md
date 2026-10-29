# Diagnostic VM identifies the live RPC probe as the failing stage

M409, 2026-10-26. The logging-only observer VM failed managed initialization
after 1,416.079 seconds and shut down cleanly. Its separate read-only auditor
preserved disk/base hashes; all owned units are terminal. The disk is retained
privately with its verified compatibility path. No recovery phase ran.

The initializer published GATEWAY_READY; its container lived 592.601 seconds,
then exited0 without OOM. The manager retains the original started gateway,
three operations and no member instances or separate recovery observations.

The fixed-field diagnostic has 300 valid events and retains the final128. These
show eight RPC-stage failures in under2s each, none with an expired observation
context. The preceding retained input preparation, engine/image/container,
namespace, filesystem and mount checks succeed. This locates the next defect
investigation in the RPC execution/probe/parser path. It does not yet distinguish
which internal check fails. Increasing the freshness deadline is not supported
by this evidence. No timeout, ownership or acceptance check was changed.

The independent auditor's input scans both succeed:77.038s cold and11.858s warm
with one CPU/2GiB. Those resource conditions differ from the runtime. They are
not proof of the RPC-stage cause. No key contents, chain payloads or raw logs
were exported. Only fixed stage/timing/boolean projections and hashes are public.

M409 supersedes M407's pending diagnostic-runtime statement. M405 source checks
and M408 synthetic preservation checks remain valid. The diagnostic overlay has
not been installed into the production observer, and current managed readiness
remains unqualified. Next: inspect bounded request-stage metadata or reproduce
the probe boundary, correct the evidenced defect, and run fresh ordinary binaries.

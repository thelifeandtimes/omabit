# Batched exact-container observations — 2026-10-23

Full inspection now batches the exact recorded anchor/member container IDs. It
rejects invalid selections and missing, extra, duplicate or foreign response IDs;
response order cannot select ownership. Each full contract, complete inventory,
bridge and repeated namespace check remains required. The second complete read is
also batched. No resource is discovered or adopted by name. This removes three
full-container CLI invocations when two members are present without relaxing the
30-second observation or preflight bounds.

Dispatch errors now include bounded preflight/inspection ages and the expected
phase to distinguish expiration from phase mismatches. Inert malformed-response,
ordering and freshness cases, focused race regressions and vet pass. Appliance14
has 102 independently verified source files and a verified embedded test binary.
No real runtime pass is claimed here. Trial12 and trial13 retain their original
failed evidence; early-boot shutdown remains unisolated. The read-only readiness
adapter is included in this binary but is not exercised by its launch fixture.

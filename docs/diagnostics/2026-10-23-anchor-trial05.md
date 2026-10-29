# Actual-ship anchor trial05 — failed, 2026-10-23

Fresh VM23-05 completed ordinary published messaging, same-pier receiver replacement
and subscription recovery, then wrote a fifth update before stopping its anchor.
Both stopped-anchor endpoints refused with curl7. Restart created a new namespace;
both old members remained live in the old one. The first post-restart endpoint
probe failed its assertion before either member was replaced. The probe did not
retain its curl outcome, so this trial cannot establish the precise cause.

All created ships subsequently stopped cleanly, the anchor stopped without OOM,
and the guest powered off in1321.99seconds. Subscription deletion and session
revocation could not complete through the unavailable endpoints. This is a failed
recovery gate, with incomplete session cleanup retained explicitly. Its fresh
fake data and sessions remain only in the offline isolated disks; no production
ship or plugin was involved. All18 extracted files and original disk/artifact
hashes were independently verified before preservation.

The separately retained endpoint20 fixture proves that the same rootless forwarding
pattern can return curl52/status000 after namespace replacement. That is evidence
for correcting the probe, not proof that trial05 had exactly that outcome. The
successor records bounded exit/status/body-length/error-length facts before
assertion. It admits only recognized transport failures with status000 and an
empty body; successful HTTP, HTTP errors, partial bodies, unknown errors and
malformed output are rejected. No response or diagnostic text is retained.
All43 offline network tests pass. A fresh runtime retry remains required.

# Application-state setup diagnostics — 2026-10-22

Controls12–15 failed before the persistent-state/subscription assertions. All were
new fake zod/nec pairs under an 8 GiB, no-swap, one-core private Bubblewrap scope;
all completed the existing transport controls and stopped both ships cleanly.
No production ships, services, plugins, Docker or excluded directories were used.

- Control12: Lens `|mount %base` timed out. No usable setup receipt.
- Control13: acknowledged Gall mount/commit and read-only file readiness passed;
  the app-start thread returned HTTP500.
- Control14: JSON setup receipts passed mount/commit; start still returned a
  non-JSON HTTP500. Changing response format did not resolve startup.
- Control15: read-only Lens checks confirmed Clay domes exist and the fresh base
  desk's extra-app override map is empty. The scry-containing start thread returned
  literal `crud!`; no state write or subscription had begun.

The next source revision uses a minimal fixed start poke only after checking that
map is empty, and moves the fixture snapshot helper outside Gall's ten-arm agent
core. Those are candidate corrections, not a proven root cause for HTTP500.
The fixture is not a general backend installer and must refuse preexisting override
entries rather than replacing them.

Original piers, frozen source, logs, result files and hashes were copied to
`/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-22-state-recovery/`.
Each relocation verified all file sizes, modes and SHA256 hashes and retained a
compatibility directory link at its original /tmp location. The accompanying JSON
contains inputs, outcomes and retained paths. No failed outcome is overwritten.

The passing earlier Docker transport checkpoint remains valid at its original
scope. These diagnostics do not establish application-state or subscription recovery.

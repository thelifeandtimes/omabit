# Stopped-pier copy qualification accepted — September 28 UTC

The user supplied the completed **6O** report for run
`185ff2ed58b9abdd6861dfe64d6650f5`, profile `stopped-pier-copy`.

- Checkpoint passed; cleanup complete; live limits and pier metadata observed.
- Copied **7 files / 1,016,840,204 logical bytes**.
- Content/tree digest:
  `7d897a1765a4b0d0b511b2a3d5ec5d721aad07f413d0cdd219aca71a98908ff8`.
- Original reserved, copy verified, both initialized-pier resumes verified, and
  original content unchanged. Transfer uncertainty was false.
- Probe, source boot, import resume and replacement containers were all removed
  with resolved create outcomes. Public fake identity checks passed on the source
  and both resumed copies.
- The original and copied private piers remain under the qualification run.

Report:
`/var/lib/urbit-runner/.local/share/urbit-runner/qualification/185ff2ed58b9abdd6861dfe64d6650f5/report.json`.

An independent identity-pinned collector snapshot confirmed management was running
again: alpha running/healthy at 2/2, panel-check running/healthy at 1/1, ssh-check
uninitialized at 1/0, and no active work or recovery requirements.

This accepts the bounded transfer primitive on a fresh stopped fake. Managed
registration, external-pier validation/upload, key/comet bootstrap and production
hosting remain separate work.

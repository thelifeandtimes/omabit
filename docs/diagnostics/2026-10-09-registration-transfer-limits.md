# Registration and transfer limits — 0.22.1 deployment

The fixed registration-count cap is removed from runner registration/enrollment,
incoming handoffs, collector preflight and UI checks. Pier contents may total
100 GB (100,000,000,000 bytes), plus the existing 32 MiB archive-format allowance.

Full Go tests, vet and UI model checks passed. Regression coverage exercises all
registration paths beyond the former cap, populated handoff destinations, and
the size boundary using sparse-file metadata. Native/static AMD64 and ARM64 stub
builds passed. This does not claim a full-size live 100 GB transfer.

## Installed artifacts

- Desktop IPC reports **0.22.1**, installed through the cold replacement flow.
- Local installed and plugin-bundled runner SHA256:
  `2b222900a8d15c5dec0a2502d57651f584f7a6c37250bea32ff26d991b4d7754`.
- Sourcebox installed runner SHA256:
  `0d1869db478a15d90a855e2e1d80fbfe6636a89a6d422f7165aff2297c95c04c`.
  The administrator SSH installer returned ready=true and the remote installed
  binary passed an explicit checksum comparison.

The first administrator SSH attempt ended with a broken pipe before file copy or
installation. The explicit retry unlocked the key locally in a temporary agent
before connecting; that agent was removed when deployment finished.

## Post-update verification

The before/after comparison preserved every ship identity, desired/observed state,
configuration/port/revision, maintenance policy, handoff descriptor, snapshot
interval and recorded container ID. Both runner identities remained pinned and
reachable, with no active operations, recovery requirements or incidents.

- Workstation: four running ships; `import-check` stopped; `panel-check` moved and
  fenced; `rus` and `ssh-check` uninitialized.
- Sourcebox: `remote-smoke` and `remote-comet` running; transferred `panel-check`
  stopped. Comet tailnet HTTP 18082 still returned its expected identity.

Private deployment receipt: `/tmp/opencode/deploy-limits-0221.json`, complete=true.
Remote installer/checksum receipt: `/tmp/opencode/sourcebox-limits-0221-retry.log`.
Release-pill work and the planned restart/reboot/interruption qualification remain
the next work; no host reboot or full-size transfer was performed in this update.

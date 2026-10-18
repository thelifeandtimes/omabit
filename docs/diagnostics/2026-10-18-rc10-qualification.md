# RC.10 frozen-binary qualification — 2026-10-18

Source: `f6981e6ce4b9bb1bddc1334ff23a3f1ae9e9794a`.
Release: [v0.1.0-rc.10](https://github.com/thelifeandtimes/omabit/releases/tag/v0.1.0-rc.10).
Evidence directory: `/tmp/opencode/rc10-qualification/` on sourceport.
Public sanitized evidence: the release's `qualification.json`.

## Source and package checks

- Full Go tests/vet, manager/runtime/client/collector race tests, both JS suites,
  mapped direct/native QML and installer/native-bootstrap fixtures passed.
- [Source CI39091555438](https://github.com/thelifeandtimes/omabit/actions/runs/39091555438) passed.
- Frozen package hashes, source revision, executable smoke checks, schema19 metadata,
  isolated install/replacement and customization-preservation checks passed.

## In-place management upgrade

A new owner installation used the published RC.9 runner and two disposable fakes.
`rc10-upgrade-stopped` (`~bus`, supplied pill) was initialized then stopped;
`rc10-upgrade-running` (`~zod`, bare bootstrap) remained running.

Two invocations of the final RC.10 `install-host` preserved:

- Runner `runner-9f1b1e23a7b8076433a6543f2959a216`.
- Running container `3d9dc7c85f6239527c395b0b2070bb739fd15f1981c0af79d1efaf49e4f75a91`,
  king PID956500, start time and arguments.
- The stopped instance's intent, both configurations/revisions and operation history.
- Engine process, SSH host identity, operator grants and owner profile.

The database migrated16→19. RC.9's original state-opening code refused an isolated
SQLite backup of the real schema19 database and retained its version marker.
The test did not attempt to downgrade the installed runner.

## Final runtime/native checks

- The existing default launch retained loom31, direct Vere and `--no-conn`.
- Saving loom34/native-on did not relabel it. Explicit Stop/Start produced the
  fixed `umask 077` wrapper; king and worker arguments both showed loom34.
- The native socket was owner-private. A file-based khan-eval thread returned the
  exact noun `[0 %avow 0 %noun 'omabit-rc10' 0 42]`.
- A frame-aware client consumed the entire single Newt reply before disconnecting;
  it used no killed-client timeout or packet-count heuristic. Reply bytes were
  compared with the pinned codec's encoding of the expected noun.
- The actual RC.9 collector preserved native opt-in during an HTTP-only edit.
- Explicit disable and replacement produced loom32, direct Vere, `--no-conn` and
  no native socket. The 4GiB/2CPU/256PID limits were verified independently.
- Newly generated comet `~modhut-wiswyd-samdes-todrec--darnet-falwyc-daptuc-litzod`
  booted with loom32/native-on and returned the same verified file-thread result.
- Owned Stop receipts succeeded for all three fixtures; containers were removed
  and stopped piers retained.

## Boundaries and resolved setup refusals

An older rootless preparation bundle was refused before host changes; regenerating
it with the frozen RC.9 runner resolved that mismatch. Initial starts were refused
below the existing 5GiB available-RAM threshold; the user freed memory before retry.

The first native assertion expected a term rendering; Vere printed its atom as a
quoted cord. The recorded, fully drained reply was verified by exact noun encoding
before continuing. No runtime code changed after the release build.

Cross-host handoff and other ship classes have synthetic regression coverage.
These checks do not qualify every stable upgrade boundary, large native replies,
the supplied pill's application inventory, or a fix for Vere's known pending-reply
disconnect bug. No protected pier was used.

# RC.12 preparation: mapped desktop qualification

## Result and scope

**PASS: eight mapped development-source cases on sourceport**, 2026-10-18,
15:24:40–15:27:03 UTC. Hyprland **0.56.2**, commit
`efb50993780079460b0cbed1363e2166a2de1d9f`, Lua configuration provider,
DRM backend, Wayland session; Quickshell **0.3.1**.

This qualifies the recorded, uncommitted UI bytes while the manifests still name
**0.1.0-rc.11**. It is preparation for RC.12, not a frozen RC.12 release result.
The report records individual SHA-256 hashes for the plugin, native files and
qualification scripts, the source commit/dirty status, commands, exit statuses
and input-unchanged check.

Evidence on sourceport:

```text
/tmp/opencode/rc12-desktop-source-run1/report.json
/tmp/opencode/rc12-desktop-source-run1/<case>/result.log
/tmp/opencode/rc12-desktop-source-run1/<case>/quickshell.log
/tmp/opencode/rc12-desktop-source-run1/<case>/mapped-windows.json
/tmp/opencode/rc12-desktop-source-run1/<case>/calls.jsonl
/tmp/opencode/rc12-desktop-source-run1/log-watch/events.jsonl
```

| Case | Result / observed boundary |
| --- | --- |
| `panel` | PASS; actual mapped views, runtime logs, setup, dialogs, maintenance, retained reviews, routine revisions, imports, typed deletion confirmation; **21** distinct fixture mutation request IDs |
| `panel-native` | PASS; same panel scenarios through canonical `native/Service.qml` and `native/Panel.qml`, with native tool-directory assertion |
| `recovery` | PASS; inventory refresh preserves review, stale/evidence-changed targets refused, one rejected request plus one explicit retry, stopped receipt, no Start |
| `recovery-native` | PASS; same recovery scenario through canonical service wrapper |
| `log-watch` | PASS; fragmented/multiple records, bounded persistent stream, pause/refresh, selection/revision/enrollment cancellation, hidden/navigation gates, retry backoff, denied/mismatched/oversized output, no overlapping or surviving collectors |
| `onboarding` | PASS; empty welcome, explicit setup approval, laptop preference persisted only in fixture state, health gating, port lookup, Advanced storage selection, uncertain registration never boots, uncertain Start never retries |
| `focus` | PASS; real menu button opens/focuses one window, cross-workspace reuse, repeated activation, special-workspace reveal, close/reopen; no ship mutation calls |
| `focus-native` | PASS; all focus scenarios through canonical wrapper with the stock loader's parentless post-start service creation pattern |

The minimized-state check in log-watch injects Qt's visibility state because
Hyprland does not implement native minimization. Other mapped lifecycle/focus
checks use the real compositor.

## Isolation and harness corrections

`scripts/desktop_fixture.py` gives every fixture private `HOME`, config, data,
cache, state and runtime roots. Only the Wayland socket and a short Hyprland IPC
alias connect it to the desktop. All panel management commands target synthetic
collector/helper executables. Tests do not read/enroll live collectors, operate
piers, install plugins, change installed preferences or restart the user's shell.
Only fixture windows move. Existing windows are checked by address, PID and
workspace; original focus is restored using the current Lua expression contract,
`hyprctl dispatch 'hl.dsp.focus({ window = "address:…" })'`.
Quickshell and its fixture children have a private process group that is reaped
on completion or failure. Compositor observations prove mapped fixture-owned
toplevels, beyond QML's `visible` property. Reports retain raw QML logs and CLI
audits, including failures.

The first onboarding run failed at **“Custom selection did not reveal pier
path”**. The old test selected a hidden storage control without expanding the
current Advanced section. It now asserts the initial collapsed state and clicks
the actual Advanced toggle before selecting storage. The corrected mapped test
passes; no production change was needed. Initial failure evidence is retained
at `/tmp/opencode/rc12-desktop-dev/onboarding/quickshell.log`.

The first private-cache smoke exposed a harness-created parent directory with
mode 0755. The bootstrap correctly refused it. The harness now creates both cache
root and manifest directory through the runtime's 0700 directory validator.

## Native bootstrap evidence

`python3 -B scripts/test-native-plugin.py`: **8 tests passed**, covering concurrent
download/cache locking, verified offline reuse, tamper/partial download refusal,
HTTPS/hash/size enforcement, platform/path checks, launchers and frozen exports.
`python3 -B scripts/test-desktop-qualification.py`: **4 tests passed**, covering
archive root selection and traversal/link/duplicate refusal, package content
tampering, complete writable-XDG isolation, and old-version/source pin rejection.
The recovery static prerequisite/refusal/retry/negative-control test also passed.
The targeted `TestCodeClipboardHandoffKeepsSecretOutOfArgumentsAndOutput` Go test
passed with its fake provider, checking secret-free argv/output and cancellation.

An additional run used the real already-published RC.11 binaries, with a newly
created private cache, without downloads:

```text
/tmp/opencode/rc12-desktop-final-runner-smoke/report.json
```

All three binaries matched `native/bootstrap/release.json`, embedded clean Git
provenance matched `abfff0529a19adc02cf9f4b93eeef78c645dfa8f`, all three native
launchers passed `--help`, and the runner returned **schema19**. The recovery-native
mapped case also passed in this run. These are the published old tools, not the
current schema20 implementation. Their existing pins are:

| Binary | Bytes | SHA-256 |
| --- | ---: | --- |
| omabit | 10055839 | `abcd883df060a3a791976fbb62676c5bb4fc3490b96fe2dbef652be47f3a32d8` |
| urbit-collector | 10186438 | `c8825ef9d5041a0b1cf4ac1f86fed84a135ebcc496a9e207c96302c486be5c78` |
| urbit-runner | 18810613 | `e53a798eb8bbe63d2e02fd5a284222690d06d5488bfa072364669d6baf9c6791` |

## Repeatable runner and frozen RC.12 gate

Run from an unlocked Lua Hyprland session, with Python, Quickshell, `hyprctl` and
Omarchy available. The binary-provenance check additionally uses `go version -m`.
The evidence output directory must be new. Desktop cases run sequentially.

Development source (also accepts another source root):

```bash
python3 -B scripts/qualify-desktop.py \
  --input /path/to/source \
  --output /path/to/new-evidence
```

After the parent freezes RC.12 and prepares matching native pins, run against the
**actual release archive** (an extracted package root is also accepted):

```bash
python3 -B scripts/qualify-desktop.py \
  --input /path/to/omabit-0.1.0-rc.12-linux-amd64.tar.gz \
  --native-root /path/to/frozen-distribution-source/native \
  --expect-version 0.1.0-rc.12 --require-frozen \
  --output /path/to/new-rc12-frozen-desktop-evidence
```

The runner checks archive members, complete package checksums, canonical/component
manifest equality and the loaded-QML status version, then checks
that the canonical wrappers import plugin bytes identical to the package. It
uses the package's `build/` executables, verifies their native size/hash pins,
embedded clean source revision, trimpath/static build flags and schema against
`BUILD.json`, and invokes only help/schema commands from the isolated native cache.
Source-root qualification may instead pass `--binary-dir` with a frozen `build/`
or release-assets directory. Frozen source must be clean; a later pin-only
distribution commit is accepted only if relevant source/build/UI/bootstrap inputs
still match the pinned binary-source commit. Inputs and binaries are rechecked
after execution. Missing wrappers or RC.11 pins cannot silently certify RC.12.

The mapped QML cases intentionally continue to use fixture management commands,
including when the exact frozen binaries are checked separately. No fixture UI
action can become a real host-management request through the native launcher.
`--cases` selects a subset for diagnosis; retain the default eight cases for the
complete mapped gate. A subset report identifies exactly what ran.

**Still required:** the full frozen RC.12 run above after version/pin preparation,
live desktop sensitive-clipboard expiry, and authorized stock Omarchy cold-update
integration. The existing clipboard unit test substitutes `wl-copy`; the panel's
copy-code fixture only emits a success acknowledgement. Neither establishes actual
30-second desktop clipboard expiry or clipboard-history behavior. This session
left the desktop clipboard untouched. This evidence also does not replace the
separate real-runtime, network, or nondestructive upgrade qualification gates.

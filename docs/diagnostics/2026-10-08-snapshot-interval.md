# Sixty-minute snapshot default — 0.21

The user requested a 60-minute Vere snapshot interval and deferred the remote host
reboot test. Official Vere 4.6 `pkg/vere/main.c` documents `--snap-time` in minutes;
its parser multiplies the argument by 60, while the omitted-flag default is 120
seconds. Omabit now supplies `--snap-time 60` to new long-running runtimes, including
fake/comet/owned boot, initialized resume and imported identity-verification runs.

Offline utilities receive no periodic-runtime flag. Each container intent persists
its interval before Docker creation, and containment validates the exact resulting
arguments. Existing omitted-flag records remain accurately recognized as two-minute
runtimes until replacement. Status exposes both actual recorded and default intervals.

## Verification

Full Go tests/vet, targeted runtime/managed race tests, model tests, mapped QML tests,
installer tests, native builds and the unsupported ARM64 stub build passed.
Regressions cover runtime/utility command separation, intent persistence, refusal
of omitted/changed/duplicate snapshot arguments, owned canonical/custom-pill boot,
manager observation of an existing two-minute container and replacement at 60 minutes.

The user installed both updated managers and authorized one-at-a-time replacements
of the five running ships. Both managers first continued observing the old containers
as healthy at two minutes while advertising the new 60-minute default.

Remote rollout succeeded first. Local alpha stopped cleanly, but its first restart
was refused by resource admission: available RAM had fallen to 3.6 GiB, below the
required 5 GiB. The rollout paused. After the user freed memory to 8.24 GiB, reviewed
request `snapshot60-20261008-start-alpha-memory-ready` succeeded; subsequent local
restarts checked headroom before stopping each ship. No host was rebooted.

Final healthy 60-minute runtimes:

| Runner / instance | Container |
| --- | --- |
| remote-test / remote-smoke | `d4e7ed7c9f7cc0b107d4daee249616d466c9c5b083713083a9b164b3eeca0c54` |
| workstation / alpha | `583d291aa9d4140e62e3222b5c144b01bb7a7959546624a318793c48ba7b95f3` |
| workstation / bus-lab | `e0518af95e218944a561c0e62a218b6ef4e8190e9689057110ab8c25b26fd7ee` |
| workstation / comet-check | `8836324a16877898d7173ab1a6ba0485c8c496e0393351e60711214d0804c66d` |
| workstation / moon | `9164c2d7960161aff3922a174b3335903aa4359b8a9ffdcf1fc0e91496ef08fc` |

Ship/pier identities, paths, configurations, revisions and policies are preserved.
Import-check and panel-check remain stopped; rus and ssh-check remain uninitialized.
Their next starts use 60 minutes. Both runners report no active operations,
recovery flags or incidents. Tailnet HTTP still returns `~zod` for remote-smoke.

Panel 0.21.0 is installed/loaded. Built, local installed and plugin-bundled runner
SHA256: `5ef30c93e12f56b83ce9126a9559639801c530b0c8ee5501c3a5328ce3197948`.
The static remote artifact SHA256 is
`9daf5227ef74e645ccf0dbc72a2e95f15d243b56ff5f2d88d4a7e78308ea456f`.
Receipt: `/tmp/opencode/snapshot60-rollout.json` (`complete: true`). Previous plugin:
`/tmp/opencode/omabit-panel-before-snapshot60.tar`.

Verification establishes the launch configuration and healthy lifecycle; it does
not claim a timed hour-long disk-write measurement. Event logging continues normally.
Remote reboot acceptance remains deferred by the user.

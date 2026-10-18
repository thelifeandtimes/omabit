# Sourceport RC.10 fresh-install handoff — 2026-10-18

The user requested PR #4 integration, its review fixes, a coordinated RC.10
publication and a fresh local installation to review manually.

[Final-binary qualification](2026-10-18-rc10-qualification.md) completed before
retirement. All three disposable fixtures were stopped through their identity-
and revision-bound Omabit operations; no runtime containers remained.

## Preserved data

Manifest:
`/var/lib/omabit-retired/sourceport-rc10-reset-20261018T041832Z/receipt.json`

Retired pier root: `/srv/urbit/.piers.retired-20261018T041832Z`

| Relative pier | Identity | Files | Bytes | Unchanged metadata-tree fingerprint |
| --- | --- | ---: | ---: | --- |
| `fakeships/rc10-upgrade-stopped/bus` | `~bus` (fake) | 7 | 447168524 | `2fe24d3d65cdbd3926c889817321d162a113079b4d68d30fb7d6d39a48166bf4` |
| `fakeships/rc10-upgrade-running/zod` | `~zod` (fake) | 7 | 1044492300 | `c5127ce7d5eb7a96cee1fa0e359b09c23e924005eca7967bb48bdcfcd239968d` |
| `livenet/comets/rc10-native-comet/pier` | `~modhut-wiswyd-samdes-todrec--darnet-falwyc-daptuc-litzod` | 7 | 1017311244 | `7cf45899ba04b6bb6681cff15a24967304fe160bb45e12d76c357ceaf57c3198` |

Read-only metadata established each identity and fake/live classification. The
reviewed root contained exactly these three piers. Services, owner profile, binaries,
management/engine data and remaining runtime directories were retired with
same-parent renames. Before/after checks preserved the earlier pier archives,
accounts/groups/subIDs, preexisting lingering and user manager. Shared Docker
remained inactive with its socket active; its socket was not contacted.
No protected pier was a qualification or retirement target.

## Verified handoff state

At `2026-10-18T04:18:53.743130+00:00`:

- App installation/service paths and runner/engine sockets were absent.
- The Omabit plugin was absent from the installed plugin catalogue.
- Unrelated plugin enablement and shell configuration matched before/after reload.
- Cold shell instance `4jrleb53nt`, PID1015456, launched at04:18:50UTC.
- The frozen RC.10 setup probe reported `ready:false`, `host_present:false`,
  `can_setup:true`; an isolated fresh collector reported no runners.
- All retired-pier fingerprints matched again after the shell reload.

Machine-readable evidence:
`/tmp/opencode/rc10-qualification/{reset-result,verification,setup-status}.json`.

## Manual installation

```bash
omarchy plugin install https://github.com/thelifeandtimes/omabit.git --enable
```

Open **~+ → Setup → Set up this computer**. Published RC.10 native pins select
matching runner, collector and helper binaries. Default loom is31 and native
control is off. This reset is a development walkthrough; the separately tested
in-place upgrade preserves existing user installations.

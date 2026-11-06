# 0.3.7 preparation evidence — 2026-11-05

**Historical preparation record.** The service rehearsal subsequently passed.
See the [final qualification](qualification.json) and [service evidence](rehearsal.json).
The following records the state at handoff. Draft [PR #14](https://github.com/thelifeandtimes/omabit/pull/14)
contains the requested release source. The published baseline remains0.3.6.
Source/race CI is pending; the exact-artifact service-upgrade VM rehearsal is
blocked because sourcebox is unavailable. Final qualification and distribution
pins remain pending.

## Frozen artifacts

- Source: `076f97280960961fc21c88d45fef529185b4fbe2`
- Tree: `552380beec78212a587e4511c309e88f3814cd54`
- Backend source SHA256: `7a827fdcc469a5fde9b0b8591a253f420d2f9d378592175fe8bbc85b67318fb7`
- Build: isolated committed inputs, `go1.27.3-X:nodwarf5`, Linux/AMD64.
- Previous archive: published `omabit-0.3.6-linux-amd64.tar.gz`, SHA256
  `5946740af13a5e1c4e71fe037edf6eaa4d89006f83e8b6ad88b9481cd5e2279d`.

| Candidate asset | SHA256 |
| --- | --- |
| `omabit-0.3.7-linux-amd64.tar.gz` | `c9ef29e0b547a78817acce7c6144895f85ae9ea8d9c8a78db7d2b34497596a70` |
| `omabit-linux-amd64` | `8dd21915946b9d03f54c761584126e3498ce9e492b6e4dd8a19f44ac0934069d` |
| `urbit-collector-linux-amd64` | `57f949a91528bc382d7f33b79ef42e98c697dd576cf691851a26eae9102ba6ef` |
| `urbit-runner-linux-amd64` | `c2623c3071dd40fd4e77ee0e05f6637f714404e026548767f79a4184e05feb49` |
| `omabit-skills-0.3.7.tar.gz` | `4c921a47d76e1c685ad85a00741b784bc92e32c71a8c9c7e6f6cf245a88077a1` |
| `omabit-recommended-apps-0.3.7.pill` | `85635c0b17a138a6473580ca362475faadf281e09cc9fa3746cfc698cc054f62` |

Local assets, checksummed previous downloads and bounded evidence are retained at
`~/.local/share/omabit-review/release-0.3.7/`. `preparation.json` records artifact,
log and harness hashes. These are local candidates, not published assets.

## Passed local checks

- Asset/archive checksums, binary provenance, schema47, packaged UI versions,
  installer dependencies, fresh isolated destination, replacement and refusal to
  overwrite operator customization (`scripts/test-release.py`).
- All133 frozen-binary help entries match the CLI references in a networkless
  Bubblewrap sandbox; help creates no state (`generate-cli-reference.py --check`).
- Panel/runtime models, native plugin, bootstrap and bootstrap observation,
  release-policy fixtures, skill packaging and observation scheduling.
- Offscreen panel, host compatibility and service-upgrade QML, compact picker
  interactions, and isolated first-run onboarding.
- Real picker Browse, Escape, pointer selection of a Unicode/space-containing
  filename, reopening/cancellation and restored background input. This uses a
  private networkless headless Sway, with no host desktop socket.
- Exact published0.3.6 → frozen0.3.7 desktop hot activation, cold restart,
  synthetic failed-candidate recovery and preserved page/ship preferences.
  `docs/releases/0.3.2/desktop-upgrade-harness.py` ran inside a separate networkless
  Bubblewrap/headless-Sway environment with private HOME/XDG roots and read-only
  artifacts/source. The retained wrapper is `evidence/desktop-driver.py`.

Desktop stack: Quickshell `0.3.2.r7.g2b4501a-1`, Qt base `6.12.0-3`, Qt declarative
`6.12.0-2`. These desktop fixtures use empty isolated profiles, not live runners.
They do not establish service-upgrade continuity or actual OS authorization UI.

Selected evidence SHA256:

| Evidence | SHA256 |
| --- | --- |
| `logs/archive.log` | `ae8996ddb7f01d2b95c414f87c4918d1d43019921521f21ca964beb47a5849fb` |
| `logs/cli.log` | `f7c0e262ccbb4c99b4bd8d5268f8e2ef62dda31d78dd1c7963deb6433f21ab60` |
| `logs/wayland-picker-corrected.log` | `c35a47276a99ac24f2b82177ee5ce9eaa42f483f86df083ced7cd01c4f2c22bc` |
| `logs/desktop-upgrade-corrected.log` | `e60c186184ebbbc1aa91cc02faedef755bb4ac495276e91b2584d021ee306b54` |
| `evidence/desktop-driver.py` | `b2dc70cfaf3480e1f125b9003677c926eefb56c5ee69b8216a7e0959572418b0` |

The first picker invocation passed the tool parent instead of its extracted `root`
and could not launch Sway. The first desktop wrapper lacked the `/etc/fonts`
mountpoint and Bubblewrap refused startup. Both failures preceded application
execution. Corrected invocations passed; original failure logs remain retained.
No product-source change or artifact rebuild was needed.

## Outstanding gates

- Full frozen-source source/race/QML [CI run42153140057](https://github.com/thelifeandtimes/omabit/actions/runs/42153140057)
  was still running when this record was written. Do not infer success from local
  focused checks. The draft PR carries subsequent CI results.
- [Release-policy run42153139688](https://github.com/thelifeandtimes/omabit/actions/runs/42153139688)
  correctly refused the incomplete proposal: final pins still name0.3.6.
- Sourcebox's existing isolated service-upgrade lab must rehearse exact published
  0.3.6 → these frozen0.3.7 assets, including authorization denial/retry, manager
  restart, running-process continuity, stopped intent, identity, access, storage
  and operation-history preservation. Sourceport has no retained lab/base.
- Then attach reviewed qualification, add exact distribution pins, pass all
  required checks and merge with a merge commit before stable publication.

The user-confirmed fresh-install trial and live same-pier repair remain useful
development evidence; they are not substitutes for the missing upgrade rehearsal.

# RC.13 publication and download verification — 2026-10-19

Release: [v0.1.0-rc.13](https://github.com/thelifeandtimes/omabit/releases/tag/v0.1.0-rc.13).
Published at **17:49:54 UTC**, as a Linux/AMD64 prerelease with schema21.
Frozen source: `13d6f82206312d7c7d9b9dec5dd9eb8cf821b3c5`.
Source tree: `1629a511e7a58eafc8dc568c0a05b670c506b9eb`.

## Source and frozen artifacts

- [Source CI39296287631](https://github.com/thelifeandtimes/omabit/actions/runs/39296287631)
  passed tests/vet, transport/runtime race checks, builds, model/installer/native
  bootstrap fixtures, disposable-harness guard tests, release source isolation and
  reproducibility, and both actual offscreen Qt suites.
- Two independent clean source exports built with Go `go1.27.0-X:nodwarf5` produced
  byte-identical release files. Static binaries retain clean VCS/source-tree/epoch
  provenance. Build metadata records the native engine policy rather than an exact
  Docker29.7.2 requirement; the rootless launcher keeps its separate pinned identity.
- Frozen asset/archive inventories, checksums, binary/schema/provenance, executable
  help, isolated desktop installation/replacement and customization preservation passed.
- Govulncheck v1.9.0 in binary mode found no vulnerabilities in any of the three tools.
- The root/plugin manifests, monitor header, setup header, bar tooltip and live
  monitor status all identify **0.1.0-rc.13**. The visible labels previously left at
  RC.9 were corrected. Source and archive checks now enforce the current UI version.

| Asset | Bytes | SHA-256 |
| --- | ---: | --- |
| `omabit-0.1.0-rc.13-linux-amd64.tar.gz` | 29832489 | `c092ab3a13cb38d29506f79a0ee191420406c5a2b7d3e43202f3603952bc8550` |
| `omabit-linux-amd64` | 17672681 | `2a13b2ed3b1f150f2fe03c79f126f09168be7369c193d8780109a77c18a514ab` |
| `urbit-collector-linux-amd64` | 11873020 | `edf274777446dfbcaa6720708f4fd322c03d66ee1b713da4f47448691d581321` |
| `urbit-runner-linux-amd64` | 22362579 | `2f17cb080e19c4c420adff19366d675133aecbdf09c8d354c1f3737e8662f4e4` |
| Supplied Recommended Apps pill | 59351751 | `85635c0b17a138a6473580ca362475faadf281e09cc9fa3746cfc698cc054f62` |
| `qualification.json` | 4506 | `49dd97b61fa013abbc77a96f661ed60c51b5e9610373737d0c6d0df5752e8713` |

The tag identifies the frozen source. Native binary pins advance in a subsequent
distribution commit, referring to the published source and exact executable hashes.

## Publication and native cold-cache checks

All eight draft assets were downloaded independently before publication and passed
the complete archive/package checks. Ordinary asset downloads do not preserve the
standalone files' executable bits; content hashes and the archive's executable
modes were checked independently.

After publication, all eight public assets were downloaded again. Their exact
inventory, sizes and SHA-256 matched the local frozen assets and GitHub's reported
digests. The qualification record's source and artifact list matched release metadata.
The published archive again passed provenance/schema and isolated installer checks.

An empty private native cache then downloaded the three tools through the real
canonical launchers and public HTTPS URLs. All three `--help` commands passed;
the runner's `state-schema` returned21. Cached files matched the new pins, and the
packaged QML/manifest matched the native source files byte-for-byte.

Native pin-file SHA-256:
`5b6c1542a1f3a2549078ab62018749b35e5a8b32a88621a050e414b86297cdd9`.
Local evidence: `build/rc13-release/published-verification.json`; final assets under
`dist/rc13-final/0.1.0-rc.13/`, reproduction under `dist/rc13-reproduction/0.1.0-rc.13/`.

## Scope and handoff

The attached `qualification.json` distinguishes these final-artifact checks from
the [development native-package and migration evidence](host-compatibility-development.md).
The native Ubuntu/Arch transactions used signed inert synthetic packages. They do
not establish live Docker/Vere qualification for every accepted engine/distro stack.

The user will next perform fresh installations on **two Omarchy versions**. Those
results, live Docker29.9.1/Vere qualification, broader distribution profiles and
stable0.1.0 acceptance remain pending. No installed packages, desktop configuration,
services, enrollment state or live piers were changed to publish this release.

Fresh native installation uses:

```sh
omarchy plugin install https://github.com/thelifeandtimes/omabit.git --enable
```

Then open **~+ → Setup → Set up this computer** and confirm the UI shows RC.13.
Existing hosts use the [schema21 upgrade procedure](../host-compatibility-upgrades.md).

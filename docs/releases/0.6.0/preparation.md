# 0.6.0 release qualification and distribution

The owner authorized stable release publication on2026-11-25 and will update
production personally (M806). The reviewed RC4/UI1 source was frozen at
`fcf269ca8aa20aa9e43a70d7e282a91f12659cf9`, then built as0.6.0/schema67.

The [qualification record](qualification.json) binds the exact artifacts,
0.4.0 and0.5.0 upgrades, bundled/local runtime, native desktop and source/race/QML results.
The post-freeze distribution commit changes only documentation and artifact pins;
no runtime, UI, installer, packaging or skill inputs are rebuilt or changed.
All ten [draft downloads](../../diagnostics/2026-11-25-release060-draft-downloads.json)
match the qualified bytes. [Storage and evidence preservation](../../diagnostics/2026-11-25-release060-storage-and-evidence.json) records the bounded runs and read-only recovery of damaged serial reports.

[PR25](https://github.com/thelifeandtimes/omabit/pull/25) must pass final required
checks and review, then merge with a protected merge commit preserving the frozen
source. The verified draft is published as stable/latest afterward, followed by
public download verification and native cold startup. No installed production or
DEV update is performed as part of publication. Earlier RC artifacts remain immutable.

All native runtime experiments use fresh disposable offline VMs, one at a time,
within the48GiB aggregate cap and4GiB declared scratch. Reports/logs and immutable
inputs are retained; each ended VM's generated disks are retired with exact manifests.

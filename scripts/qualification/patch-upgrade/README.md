# Guided patch and review-candidate upgrades

Explicit `--portable`, `--previous`, `--candidate`, `--rootless`, `--output` and
`--run` are required. The previous and candidate paths contain exact published
and frozen release assets, respectively. Their verified manifests supply the
versions, including immutable `-rc.N` review candidates. `--rootless` supplies the verified
launcher bundle used by the previous installer. The new output must be a direct
`patch-upgrade-NNN-NAME` child of the existing private labs root.

This extends the retained0.3.1 QML-button rehearsal to the schemas declared by
the exact previous and candidate manifests. A schema increase also checks that
the actual previous writer refuses the new database without changing it. It uses
actual sealed bundles, a pristine immutable base, offline fresh fakes, actual
service installation, authorization denial/retry, and a manager restart. It checks
unchanged running container/engine processes, stopped intent, fleet history,
identities and access. The guest alone gets an exact-program polkit fixture.

The launcher enforces one4 GiB scratch allowance under the combined40 GiB storage
budget. No host Docker, runner, keyring, desktop, SSH or network is exposed.
It never reuses an overlay. Keep only compact results after a successful powered-
off run; classify and record generated disks before retirement. This fixture does
not test human password input, the stock plugin updater, remote upgrades or power
cuts. Desktop preferences and cold/hot QML activation have a separate native test.

The rehearsal also makes a separate copy of its own stopped fake, adds
synthetic macOS metadata, checks the previous validator (0.3.3 refuses; 0.3.4 accepts), then verifies
guided identity discovery and a complete stopped import with the new runner.
All companion bytes survive and the unrelated running fake retains its process.
No production pier or prior manual review state is used.

If owner-only accounting cannot descend into rootless overlay work directories,
the launcher records that limitation and reserves an additional 4 GiB. Any other
measurement failure refuses the run. Reported charged usage includes this reserve;
it is not a claim that inaccessible directories were fully measured.

For targets starting at 0.3.6, the guest adds a sparse file of 100 GB + 1 byte
to that disposable stopped source. It demonstrates the 0.3.5 collector rejection,
acceptance by the target collector with the previous and target runners, and no
source or enrollment mutation from preview. Only that generated sparse file is
removed before the normal small import. This qualifies large logical previews,
not a full 100 GB copy or its duration/disk requirements.

Storage accounting uses the shared capsule inventory, including both DEV roots,
installed review artifacts and retained `/tmp/omabit-*` outputs. The default stays
40 GiB; `--storage-budget-gib 48` requires separate temporary operator approval.
The guest itself remains capped at 4 GiB of additional scratch.

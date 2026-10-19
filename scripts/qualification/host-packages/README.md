# Disposable native package-authentication qualification

This directory retains the **guest driver and fixture generator**, independent of
the ignored `build/` VM launcher. It exercises real native APT/Pacman commands,
signatures, installed bytes, root receipts, owner reads and package-transaction
hooks using **inert synthetic payload packages**. It does not qualify Docker,
containers, Vere, piers, host installation or management activation.

No arguments, `--help`, and `plan` are harmless. `validate` is read-only and does
not execute the candidate. Only `run --acknowledge-disposable-guest`, after all
admission checks and the exact candidate schema check, can mutate a guest.

```sh
python3 -B scripts/qualification/host-packages/qualify.py --help
python3 -B scripts/qualification/host-packages/qualify.py plan
python3 -B scripts/qualification/host-packages/qualify.py plan --format text
```

## CI: safe checks only

From the repository root, without privileges or native package tools:

```sh
python3 -B -m unittest discover \
  -s scripts/qualification/host-packages -p 'test_*.py' -v
```

The tests inject **guard observations**, not native verification success. They
exercise rejection before mutation dispatch for root/platform, marker, VM,
hostname/UUID, owner/state/process, path, candidate/version, hook hash and tool
prerequisite failures. They also run actual harmless CLI help/plan/validate output
checks from an empty temporary working directory. Native test phases are never
started by CI. A bare `validate` correctly returns exit2 with JSON refusal.

## Admission contract

The reusable entry point requires all of the following before creating output,
installing fixtures, changing sources, mounting anything, or importing test keys:

* Linux/amd64, effective root **inside a positively detected QEMU/KVM VM**, no
  container, and a hostname beginning `hostcompat-qual-` matching the CLI pin.
* A separately provisioned, root-owned, non-writable regular marker at
  `/etc/omabit-host-packages-qualification.json`. Its hostname, hypervisor DMI UUID,
  workspace, provider, candidate SHA-256/version/schema and hook hashes must match
  the invocation. The actual DMI UUID must match too. The harness never creates
  this marker or enrolls the machine itself.
* No existing `/etc/urbit-runner`, management installation, Omabit receipts,
  `/var/lib/urbit-runner`, `/srv/urbit`, prior fixture hooks or runtime executables.
  Live Urbit/Vere, Docker/containerd and runner processes are refused, never killed.
  An unreadable process inventory fails closed. Native APT credentials are refused.
* Root-owned canonical inputs and parents without writable components, symlinks,
  hardlinked input files or special files. A fresh `workspace/run` is required.
  Candidate and hook hashes must match the provisioned pins.
* An explicit development version label and state-schema number. The runner has
  no general `version` command: the label names the reviewed build, the digest
  identifies its bytes, and `state-schema` is checked for exact output **after
  admission and before the first write**. `validate` checks the pins without
  running the binary. Neither an arbitrary label nor a schema number substitutes
  for the digest.

Paths are parameters, with no developer home or checkout path embedded. Standard
native OS paths inside the guest are intentionally fixed: those are the paths the
real verifier and generated transaction hooks consume. A nonempty `APT_CONFIG`,
loader override or Python optimization setting is refused. Do not use `python -O`.

This is a fresh-guest, one-shot driver. **It refuses the already-used historical
qualification VM** because that VM contains a management executable and receipts.
Do not remove an unidentified existing installation to satisfy a guard. Prepare a
new disposable VM/root instead. No auto-reset, purge or live-host bypass exists.

## Independently prepare the guest

Use an officially signed/digest-verified Ubuntu24.04 or Debian cloud image, or an
equivalent clean dedicated test VM. The captured Ubuntu run used APT2.8.3. Give
QEMU an explicit fresh `-uuid`, a dedicated hostname, **2 vCPUs / 1536 MiB RAM**,
a disposable disk, and no host filesystem mounts or forwarded engine sockets.
The retained `build/hostcompat-native/prepare.py` can be consulted locally, but is
not required or imported by this harness. No image/binary/key/log is distributed
in this source directory.

Inside that new VM, before provisioning the marker, install native prerequisites
from its verified distribution repositories:

```sh
apt-get update
apt-get install -y --no-install-recommends \
  python3 gpg gpg-agent gpgv apt-utils strace libarchive-tools util-linux curl
systemctl mask docker.service docker.socket containerd.service
```

Do not install Docker. The fixture packages supply its required path names with
scripts that simply exit97; the harness never executes those scripts. Python3.10+
is required. The native validators must already exist; the driver installs no
prerequisite package and imports no external trust key.

For **Pacman**, independently verify an official Arch bootstrap signature and
archive digest, then extract it **as guest root** into an otherwise unused directory
under the dedicated workspace, for example:

```text
/opt/hostcompat-qualification/arch/root.x86_64
```

Keep it unmounted before validation. It needs genuine `pacman`, `pacman-conf`,
`repo-add` (with `--include-sigs`), GnuPG, bsdtar, curl and prlimit. This is an Arch
test root **inside the admitted VM**, not a host-root bind or an Arch boot test.
The suite mounts only the guest's `/dev` and `/proc`, and uses native `chroot`.
Bootstrap `shadow` is removed through native Pacman **only in that fresh subroot**
to avoid duplicate ownership of newuidmap/newgidmap. `--provider pacman` runs only
those cases; `--provider both` additionally exercises the VM's native APT root.

## Freeze and stage the candidate

On an unprivileged build machine, use a frozen source snapshot and retain its
per-file hashes, including the generated hook assets:

```sh
CGO_ENABLED=0 GOMAXPROCS=2 GOMEMLIMIT=768MiB \
  go build -p 2 -trimpath -buildvcs=false \
  -o build/urbit-runner-auth-development ./backend/cmd/urbit-runner
sha256sum build/urbit-runner-auth-development
build/urbit-runner-auth-development state-schema
```

Choose a reviewed version label such as `development-<source-manifest-hash>`.
Copy the static candidate, this source directory, and these **same-snapshot** files
into root-owned guest staging directories (normal mode0755 parents):

```text
/opt/hostcompat-inputs/urbit-runner-development
/opt/hostcompat-inputs/package-hooks/refresh-host-packages
/opt/hostcompat-inputs/package-hooks/00-omabit-host-packages
/opt/hostcompat-inputs/package-hooks/00-omabit-host-packages.hook
/opt/hostcompat-harness/{qualify,guards,fixtures,suite}.py
```

Use the three assets from `internal/rootless/assets/package-hooks/`. Do not use
symlinked inputs. Create `/opt/hostcompat-qualification` root-owned mode0755.
Preserve the build manifest separately; the harness records candidate/hook pins,
not a claim that a caller-supplied version label is a released build.

## Provision the explicit markers

The VM provisioner, **not the harness**, writes this JSON to the fixed marker path
with root ownership and mode0600. Replace every placeholder with the reviewed
values. Hash each staged hook file with SHA-256. The UUID must be the one assigned
to this new VM by the hypervisor, not a value copied from a live host.

```json
{
  "schema": 1,
  "purpose": "omabit-disposable-native-package-auth",
  "hostname": "hostcompat-qual-example",
  "vm_uuid": "<hypervisor-assigned-uuid>",
  "workspace": "/opt/hostcompat-qualification",
  "candidate_sha256": "<reviewed-candidate-sha256>",
  "candidate_version": "development-<source-manifest-hash>",
  "candidate_schema": "<exact-state-schema-output>",
  "provider": "both",
  "pacman_root": "/opt/hostcompat-qualification/arch/root.x86_64",
  "hook_sha256": {
    "refresh-host-packages": "<sha256>",
    "00-omabit-host-packages": "<sha256>",
    "00-omabit-host-packages.hook": "<sha256>"
  }
}
```

For APT-only, set `provider` to `apt`, `pacman_root` to `null`, and omit the CLI
`--pacman-root` argument. For Pacman-only, use `provider: pacman` and keep the
subroot declaration. All three hook hashes are required for every invocation.

For a Pacman subroot, also write a root-owned mode0600
`<pacman-root>/.hostcompat-clean-root.json`:

```json
{
  "schema": 1,
  "purpose": "omabit-disposable-pacman-root",
  "vm_uuid": "<same-hypervisor-assigned-uuid>",
  "root": "/opt/hostcompat-qualification/arch/root.x86_64"
}
```

This declaration is accepted only with the VM guard, the matching main marker,
clean-state checks and no mounts beneath the subroot. It is not an alternate-root
escape hatch for the live machine.

## Validate, then run

Inside the provisioned guest, set the variables to the exact reviewed values from
the marker; do not derive an authorization marker automatically on the target:

```sh
COMMON=(
  --provider both
  --workspace /opt/hostcompat-qualification
  --hostname hostcompat-qual-example
  --vm-uuid "$EXPECTED_VM_UUID"
  --candidate /opt/hostcompat-inputs/urbit-runner-development
  --candidate-sha256 "$EXPECTED_SHA256"
  --candidate-version "$EXPECTED_VERSION"
  --candidate-schema "$EXPECTED_SCHEMA"
  --hooks /opt/hostcompat-inputs/package-hooks
  --pacman-root /opt/hostcompat-qualification/arch/root.x86_64
)
python3 -B /opt/hostcompat-harness/qualify.py validate "${COMMON[@]}"
python3 -B /opt/hostcompat-harness/qualify.py run "${COMMON[@]}" \
  --acknowledge-disposable-guest
```

Exit0 from `validate` means only the read-only guard passed. `run` exits nonzero on
failed assertions or native tool failures. A package-hook wrapper's exit0 alone
is never treated as authentication: receipts, modes, native versions and UID1000
reader results are independently checked. There are no mock validators or
download success overrides.

### Cases retained

* Exact-version native APT update/download, per-source Signed-By, native gpgv /
  apt-helper / dpkg-deb, `_apt` credentials and private sealed evidence.
* Offline retained-proof refresh after cache deletion; installed-byte tampering;
  wrong source key and armor; native unreadable-key, missing-account and HTTP403
  failures, with recovery.
* First-use/replacement native package transactions under umask077, late archive
  cleanup, root0644 receipts / private0700 cache, immediate UID1000 reads and old
  role reuse. Known legacy restrictive-layout repair and unrelated dpkg updates.
* Hidden-source and config/environment authority withdrawal; real multi-signature
  exit2 acceptance, invalid-only and selected revoked signer denial; colon-named
  redirect rejection and benign colon config acceptance.
* Native Pacman standard cache/database PGPSIG fallback, exact-build fetch via
  pacman-conf/curl, retained offline proof, tamper/revocation denial and both hooks.

Fixture setup uses umask022 for public staging parents. Each verifier/transaction
permission case explicitly switches to umask077. `_apt` UID/GID are read from the
fresh guest rather than assuming Ubuntu's historical UID42. Versions and fixture
revisions are recorded; no upstream Docker version is actually launched.

## Outputs, failures and teardown

Everything generated is outside the repository, under `workspace/run` or the
declared guest native paths. `run/evidence/` contains command/exit ledgers, native
tool versions, credential samples, traces, reports, receipts and results.
`run/inputs/` retains public signed inputs including multisignature negative cases.
The rest of `run/` contains generated private signing keys and retired fixture
proofs. **Do not commit or blindly publish it.** Export reviewed evidence/public
inputs only; exclude `key-*`, `multi-keys`, all secret-key files and any raw logs
that have not been reviewed. No generated artifacts are included here.

A failure preserves evidence and makes the one-shot guard refuse another run.
Do not turn failures into success by changing core or bypassing the guard. Fix the
candidate separately and use a fresh disposable VM/root with a new source/hash pin.
The driver attempts to stop its own GPG agents and unmount its Arch test mounts;
it never force-unmounts unrelated paths. **Power off the guest after collecting
evidence**, even when a busy mount or failure prevented local cleanup. There is
no automatic host-service restart, owner creation, pier action or VM teardown API.

## Captured native pass versus this reusable source

The source cases were promoted from these real native runs:

| Capture | Core source manifest | Candidate SHA-256 |
|---|---|---|
| Ubuntu/Pacman full pass, 2026-10-19 02:33Z | `8c610d20cf023aae9ef33af692098e80e9452bafd8e357612cc820a28b4f3a20` | `ae059034de9ce163b928cbb07f78818490499accee5d4256c5881c418fd3648e` |
| Focused APT colon pass, 2026-10-19 03:38Z | `396150ec2aca6b512b377d3b606c7f88fd8ee2263ab4fa329d7bb7d3092b5e28` | `a599b589a0e3e4b03d10c54484914957aec8858ea39d1c6253e4105ec8a9d8a9` |

The earlier full run had 40 verifier calls, 31 UID1000 reads and four native hook
transactions on Ubuntu24.04/APT2.8.3 plus an Arch2026.10.01 bootstrap/Pacman7.1.0.
The colon recheck had four root refreshes and five owner reads. Original evidence
is locally retained under `build/hostcompat-native/run-fixed-20261019T023329Z/` and
`run-colon-20261019T033848Z/`; those ignored artifacts are not a repository dependency.

This promotion adds stronger VM/marker/hash/owner guards, fresh-fixture admission,
parameterized paths and `_apt` identity, provider selection and safe CI tests.
It retains the native cases, but **its new entry point and setup were not given
another full native qualification run during promotion**. Passing guard tests is
not native qualification of this script revision or any future candidate.
Record the current script hashes alongside a future run:

```sh
sha256sum scripts/qualification/host-packages/*.py
```

The local captured scripts are identified in `PROVENANCE.md`; those hashes are
distinct from the reusable scripts' hashes and from the runner/core manifests.

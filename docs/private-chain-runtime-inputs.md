# Inspect a private-chain runtime input generation

`private-chain-inputs` is a local, read-only development command. It verifies an
already prepared input tree; it does not download dependencies, install software,
contact a runner, open a chain with a database library, or execute a file. Current
gateway programs are guest-only qualification candidates. A successful result
always has `runtime_authorized: false`.

```sh
urbit-collector --format text private-chain-inputs \
  --directory /absolute/sealed-inputs \
  --manifest-sha256 SEPARATELY_REVIEWED_MANIFEST_HASH \
  --foundation-digest REVIEWED_PLAN_FOUNDATION_DIGEST
```

Omit runner/socket selection. The collector registry is not opened. Obtain the
manifest hash during separate artifact review and the foundation digest from
the reviewed private-chain plan. Reading a hash from the same untrusted download
is not publisher authentication. This source step does not publish a supported
runtime bundle or claim that arbitrary matching dependencies work together.

The version1 `private-chain-runtime-inputs-v1` manifest is canonical JSON with
an explicit linux/amd64 platform, pinned Vere image reference, public foundation,
and complete sorted file inventory. Each file records its relative path, byte
count, SHA-256 and executable bit. The reader checks pinned Vere, bootstrap pill,
Node, package-lock, Bridge metadata and the glibc native addon, plus the required
gateway, loader, compiler, dependency-closure and foundation files. It hashes all
listed bytes and refuses unlisted entries. The image reference is review metadata;
no engine/image availability or image contents are observed by this command.

The tree is owned by the calling user: directories0500; files0400, or0500 when
the manifest marks them executable. Its root must be an absolute canonical path.
Links, unsupported types, mount traversal, changed files, missing inputs and
unknown/ambiguous manifest versions refuse. Linux openat2 rejects symlink and
mount traversal, including bind mounts on the same filesystem; unsupported kernels
refuse instead of falling back. Metadata and total file bytes have fixed limits.
Verification does not load executable files or interpret the chain database.

Treat failures as input-review failures. Preserve the original bundle and any
partial preparation. Do not recursively chmod a project, installed dependency,
chain or pier to make it pass. Prepare a new inert generation from reviewed inputs.
The selected UID and its parent directories are trusted; sealed modes do not
isolate this bundle from other programs running as the same user.

The source command `scripts/qualification/private-chains/prepare_runtime_inputs.py`
can now prepare this inert generation from a separately hash-reviewed, already
built private-chain qualification appliance and the pinned pill:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 -B scripts/qualification/private-chains/prepare_runtime_inputs.py \
  --appliance /absolute/private-appliance \
  --build-sha256 SEPARATELY_REVIEWED_BUILD_HASH \
  --pill /absolute/verified/boot.pill \
  --output /absolute/private-parent/new-input-generation
```

The output parent must be owned, canonical and mode0700; the output must not
exist. Preparation verifies the reviewed build, inventory, embedded payload and
pill hashes, copies only dependency/foundation inputs, then fsyncs and seals the
new tree. It never boots an appliance or executes its dependencies. Keep the
bundle private because it contains the original unkeyed foundation database.
The result requires the separate Go inspection command above; preparation is not
publisher authentication, proof of an unkeyed chain or startup authorization.
A failed/partial output remains preserved and must not be reused or repaired in
place. Current gateway sources remain guest-only qualification candidates.

Inspection is a snapshot, not a startup receipt. An eventual runner operation must
freshly bind the exact input generation to its runner/project/generation and
operation, then verify the observed chain and runtime objects. A claimed foundation
in a manifest does not prove that its database is unkeyed or that an endpoint has
the claimed genesis/contracts. Existing schema41 reservations remain unchanged.
Initialization, retained startup, publisher verification and supported runtime
delivery remain separate acceptance gates.

M388 adds a held-root verifier for internal callers. Opening checks only the
canonical root and review identifiers; each Verify must freshly validate contents.
It rechecks the original pathname before and after hashing and refuses replacement,
even by a same-byte tree. Close releases the handle and prevents later verification.
The offline inspector shares these checks. This remains a snapshot under a trusted
UID, with runtime_authorized=false; no execution or repair API is exposed. See
[held input evidence](diagnostics/2026-10-25-private-input-verifier.md).

M391 corrects initialization from sealed database inputs: copying0400 files must
produce a fresh private writable database, not chmod the reviewed source or reuse
a retained chain. The qualified helper creates an exclusive0700/0600 copy and syncs
it; partial output is preserved. The changed initializer requires a new sealed
generation and runtime qualification. Generation06 remains historical and unchanged;
its successful inspection alone does not qualify booting it. See
[fresh copy evidence](diagnostics/2026-10-25-private-gateway-fresh-copy.md).

M392 adds initial startup close/sync/reopen verification and an explicit read-only
startup status. It checks live code and keys at head 0x50 before publishing
readiness. This is source-qualified only; a fresh sealed generation and no-NIC VM
run must establish actual behavior. Runner observation must independently bind
the live endpoint to its selected engine/container and check current chain and
filesystem evidence. Saved ready metadata alone is insufficient. See
[startup checkpoint evidence](diagnostics/2026-10-25-private-gateway-startup-checkpoint.md).


M399 supplies the real internal runner adapter. It derives the input path from
the opened managed state root and the reviewed manifest hash:
`managed/private-runtime-inputs/<manifest-sha256>`. It does not download, populate
or repair that location. Before queue admission it verifies the complete bundle,
the fixed initializer and the owned empty project workspace without creating
client metadata. Accepted startup exclusively creates a separate operation-bound
Docker client directory. Recovery opens only the matching retained client with
read-only runtime authority. No public lifecycle route or supported installer is
added by this source checkpoint. See
[managed session evidence](diagnostics/2026-10-25-private-gateway-managed-session.md).

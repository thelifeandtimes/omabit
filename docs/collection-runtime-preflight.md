# Prepared collection runtime compatibility

Check the exact immutable Omabit, Theme and DMs generations before requesting a
forward profile switch or initial/replacement admission. From the source checkout:

```sh
python3 -B scripts/check-collection-runtime.py \
  --specification /absolute/path/collection.json \
  --generations /absolute/path/generations.json
```

The specification is the version1 collection definition from
[the collection journal](collection-journal.md), including all three pinned member
hashes. The generations file maps `omabit`, `theme` and `dms` to their absolute
immutable generation directories. Paths and input files must be explicit; this
command does not discover or select an installed profile.

Output includes the collection hash, required protocol versions and a result per
member with its path/hash, artifact format, declared protocol, readiness, reason
and next action. Exit0 means every member matches; exit2 means a valid check found
an incompatible/invalid artifact; exit1 means invalid command input. The default
requires shared admission2, authority1, reservation3 and collection journal3.
`--authority-format`, `--reservation-format` and `--journal-format` let an explicit caller describe a
different required version. Unsupported future requirements remain blocked.

Artifact format4 pins `runtime-protocol.json` with the complete admission
support inventory. Preparation and inspection compare that declaration with the
literal version constants consumed by the bundled readers. Inspection parses
source without importing or running bundled Python/application code. Mixing an
older reader with a new declaration refuses before generation publication. The
hash binds the declaration, readers, application payload and generated wrappers.
This validates a versioned bundle contract; it is not a semantic audit or sandbox
for arbitrary publisher code.

Formats1–3 remain byte-identical and readable for cleanup/history but have no
explicit runtime protocol declaration. Do not infer new admission capability from
readability. Prepare a distinct complete format4 generation from verified source;
never relabel, mutate or delete the historical generation to pass the check. The
original format3 writer/support/generation fixture is retained and reproduces.
Old readers refuse4. Runtime authority, reservation, journal and admission state
formats are unchanged by this packaging addition.

A ready result proves only exact bundle/protocol compatibility at inspection time.
It does not prove application/backend availability, credentials, process closure,
desktop isolation or permission to activate. Recheck pinned generations at effect
boundaries; never reuse readiness after bytes or selection change. The private
native adapter checks the complete forward/rollback set before forward draining,
and the target before reservation or restart. Candidate cleanup remains available
independently; compatibility checks must not obstruct cleanup already required.

The supported public workflow here is read-only inspection. Public collection
activation, migration from an already loaded legacy collection, interrupted
provisioning and cross-session bootstrap remain separate acceptance gates. See
[qualification evidence](diagnostics/2026-10-25-runtime-protocol-preflight.md).

## Linked successor bundles

Use `--authority-format 2` to inspect generations intended for an explicitly linked
replacement desktop. Artifact5 adds successor-protocol2 with authority readers
`[1,2]`, the successor guard and its complete local import set. Preparation/verification
checks literal declarations and statically resolves ordinary imports against bundled
modules or the standard library. It does not execute bundled code. The old protocol1
file remains accurate for the old modules; it is not reinterpreted as authority2.

The new internal successor preflight defaults to2. The public command retains its
authority1 default for existing callers. Mixed4/5 passes an explicit1 requirement but
refuses2. Formats1–4 remain readable with their original capabilities; no immutable
generation is rewritten. The frozen original4 writer reproduces and refuses5.
This qualifies packaging/inspection only. Native successor activation is still a
separate gate; see [M320 evidence](diagnostics/2026-10-25-successor-runtime-bundles.md).

M321 subsequently passes private native artifact5 activation under linked authority2,
including controller-loss retries and direct Theme/DMs subscription restoration.
It uses the same generations before/after desktop loss. This narrows the integration
gap above without qualifying installed upgrades or public activation; see
[native evidence](diagnostics/2026-10-25-native-successor-activation.md).

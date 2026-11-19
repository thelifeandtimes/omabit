# Recording upgrade retries: retained lessons

The later retries separate a product contract bug from a fixture lifetime bug.
Both whole-case outcomes remain failures; passing earlier sections does not
qualify unreached checks.

| Case | Finding | Correction and evidence |
| --- | --- | --- |
| 05 | A real nonempty original recording passed runner/client verification but failed the shared facade's output schema. The prior CLI/MCP fixture returned zero frames. | Map Go byte slices to their actual JSON base64 representation, preserving fixed byte arrays. Nonempty original/redacted CLI and official-SDK parity now checks exact octets, digests, frame times and revisions above2^53. [Pinned failed case](2026-11-19-recording-native-output-schema-failure.json). |
| 06 | The corrected native API, policy replay/lookup, pin and new-launch limits passed after a process-preserving64→65 upgrade. The fixture then found SQLite WAL/SHM at its closed-database precondition. | The fixture's Python connection context ends transactions without closing handles. A real SQLite regression reproduces the leaked reader; explicit closure on success/error passes. [Pinned failed case](2026-11-19-recording-native-sqlite-fixture-failure.json). |

Keep inferred schemas aligned with the actual JSON encoder, and exercise payloads
that reach every representation-bearing field. Do not weaken target/private-data
checks or transform original recording bytes to satisfy an incorrect schema.
Empty-window discovery tests alone cannot qualify original recording delivery.

Qualification helpers must release their own resources before assessing product
shutdown. Close read-only database handles deterministically; garbage collection
and transaction context exit are insufficient. Preserve original SQLite and all
sidecars together before analysis. Query separate copies, and reverify original
hashes afterward. Do not delete sidecars, issue a repair/checkpoint against the
failed original, or relax the unchanged-file old-reader requirement to make a
fixture pass.

Case06 retains both complete native recordings, original SQLite/WAL/SHM, receipts,
logs and full image/input disks. Its independent analysis copy is schema65 with
`quick_check=ok`, two recording intents and two control events. Genuine old-reader
refusal and final manager reopen remain unreached. The corrected fresh retry uses
the same frozen703de9f2 runtime artifacts with only the tested fixture closure
change; it still requires storage admission before launch.

The user's prior disposal authorization applied to03/04 only. Automatic approval
review separately refused removal of case05's whole-image/input files; those files
remained unchanged pending explicit authorization. Its classified retirement plan
retains exact compact SQLite, receipts, original recording frames, logs, source
and binaries while keeping the newer full case06 and the separate acknowledged
app fixture. No production or preserved DEV deployment is involved.

M712 records the later separate user authorization and completed two-file
case05 disposal, with all65 retained evidence/artifact files unchanged. No archive
was created; the old whole image is unavailable. The corrected fresh case07 now
passes the full native policy/API/upgrade scenario, including genuine old-reader
refusal and final reopen. See its [pinned result](2026-11-19-recording-policy-native-upgrade.json).

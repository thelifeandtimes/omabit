# Canonical Hoon seed formatting

A fresh observer trial stops before any chain or pier is created because the
evaluator returns a syntax error. The random32-byte seed can begin with zero
nibbles, but Hoon's hexadecimal literal requires canonical leading digits.
The applied key-validation fixture now strips only redundant leading zeroes
before four-digit grouping; the numeric seed and all256 random input bits are
preserved. Zero itself formats as0. Invalid non-hex input is refused.

The actual pinned evaluator rejects the deterministic original `0x0001` and
accepts the corrected `0x1`; grouped and all-zero cases also pass. A fresh private
chain-backed key-validation run then passes both keys, wrong-key/revision
refusals and point/galaxy/DNS parsing. Exact source hashes and inactive scopes
are independently checked. The failed pre-boot trial is retained separately.
This fixes the fixture, not deployed authentication or wallet key storage.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-private-chain-hex`.

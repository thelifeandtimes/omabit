# Persistent application state and subscriptions survive official keyed restart

Fresh native trial05 installs the pinned disposable Gall state fixture on the
new receiver. Four acknowledged remote writes produce exactly four unique stored
entries. Two writes precede restart, including one queued while the client is
disconnected; two follow it. The original session and channel resume past the
saved SSE cursor, deliver the queued update, then deliver the next write. A fresh
channel reconciles to revision3 and observes revision4. Both channels are deleted.
The original sessions, two-way messaging and all three clean stops also pass.

Independent audit verifies three installation receipts, four write records, seven
exact scry snapshots and six subscription snapshots across two channels, monotonic
per-channel cursors, the installed Hoon source hash and four final LMDB hashes.
The copied-chain audit confirms original public keys, genesis, contract code,
three receipts/two key events and unchanged head0x50 without transactions. The
exact scopes are inactive; retained pier/chain trees and source are hash/mode
checked. Secret input files and raw keyed bootstrap logs are excluded.

The unchanged official binary uses acknowledged Lens exit for initial keyed
processes and SIGTERM after reopening. This does not fix the first-boot signal
defect. These are private native keyed application/reconnect results, not Docker,
runner lifecycle, native desktop, wallet or production-network qualification.
A repository `native-peers` qualification stage is being prepared from this
passing fixture with explicit inputs and the same isolation/resource admission.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-keyed-application-recovery`.

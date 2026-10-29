# Private chain foundation

Pinned Bridge504e47335e0b65e569458a4c1f3732c3ac847d0a and
Azimuthbcf1d7bcf64cd73a3688434feb786be39a116819 source now produce a repeatable
private qualification fixture. Two compiler runs produce identical output with
Solidity0.4.24, optimizer200/Byzantium. All six expected local addresses match;
deployment nonces are0,4,7,8,9,10 for Azimuth, Ecliptic, DelegatedSending,
ConditionalStarRelease, LinearStarRelease and Naive respectively.

The final run deploys nine contracts and creates two unkeyed development galaxies,
with13 successful receipts. Bytecode, ownership, chain/genesis identity and all
receipts survive a database close/reopen. Distinct chain-ID and genesis nodes,
and missing code, refuse the fixture admission check before writes. The first
prototype merely distinguished chain IDs; the retained final fixture adds explicit
admission/refusal. It does not create usable Urbit identities or boot ships.

Actual scope admission confirms4GiB/no-swap/one-core; network/user/PID/mount
namespaces differ from the host, with no host HOME/run or HTTP listener. Node26.12.0
falls back from Ganache's unavailable native uWS binary to JS; Solc reports an
asm.js optimization warning. Both are retained; neither is suppressed or treated
as an unqualified server performance result. Dependencies were installed only in
a fresh temporary tools directory with scripts disabled. Source/dependency bytes,
all trials and independent receipt/compiler reconciliation are retained.

The applied fixture lives at `scripts/qualification/private-chains/`. This advances
M026's research to a chain-only foundation. Actual HTTP transport, boot gateway,
fresh network keys, pinned pill/key observation, two-ship communication/restart
and runner classification remain. No production ships/plugins/services changed.

Further source inspection finds that the pinned Vere dawn path calls Urbit
`getPoint`/`getDns` gateway methods, not raw Ethereum `eth_*` methods. Therefore a
Ganache RPC URL alone is insufficient for normal keyed boot. This is a source
inference pending actual binary request capture; it reinforces rather than removes
M026's warning. Implement/qualify a private gateway backed by the admitted local
chain, or a deliberately pinned equivalent boot path, without forwarding to a
public roller. Current chain deployment evidence does not satisfy that gate.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-private-chain-foundation`.

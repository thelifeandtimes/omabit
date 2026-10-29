# Retained gateway source and admission checks

The qualified Docker gateway is intentionally a one-time initializer: startup
generates two keys and copies the original unkeyed chain. Reusing it as a restart
service would violate retained-state ownership. Its M346/M352 source is unchanged.

A separate guest-only resume candidate now accepts a specific prior project,
operation, gateway identity and hashes of the original clean result, foundation
and compiler inputs. It never copies a chain, reads/generates network-key files,
submits transactions or mines blocks. It requires an existing LevelDB tree and
verifies chain1337, genesis, exact current head, nine contracts and the original
public keys/life/suite. RPC is read-only; clean shutdown requires an unchanged
before/after head hash and key observation. It records only public scalar metadata.

The current candidate deliberately covers the qualified first head0x50/key-life1
profile. It is not general rotation/recovery support. Selection metadata does not
prove Docker or filesystem ownership: the future controller must freshly verify
the exact stopped prior container, selected root and input bundle before mounting
anything. `wallet.totalAccounts=0` on actual Ganache reopen remains unqualified.

Five top-level tests/36 including subcases pass with mock RPC only. They cover
binding/byte changes, incomplete or inconsistent predecessor proof, live chain/
genesis/head/contract/key/life/suite mismatch, read-only method selection and unchanged
shutdown observation. Import is inert and host execution refuses before dependency
or chain access. Initial sandbox child setup returned EPERM; the direct pure tests
and separate host refusal exposed that boundary, and a scoped escalation passed
the complete inert suite. Syntax checks also passed.

No chain, container, VM, ship or installed plugin was opened or started. Actual
retained-chain startup, gateway replacement, peer reconnection, publication and
recovery still require a fresh controlled no-NIC VM experiment. The common original
appliance preparation continues to package the unchanged initializer.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-25-private-gateway-resume-source`.

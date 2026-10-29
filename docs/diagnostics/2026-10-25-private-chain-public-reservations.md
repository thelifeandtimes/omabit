# Public private-chain reservations and inventory

Development source now exposes separate `private-chain-register` and
`private-chain-projects` commands. Registration requires a pinned operator runner,
reviewed JSON plan/digest, request ID and explicit reservation acknowledgement.
The runner reconstructs the public plan and atomically saves schema41 ownership;
it creates no keys, chain, gateway, piers, containers or running ships. Observer
inventory reads public plans/bindings/registration references without probing a
runtime or exporting private member tokens. Capabilities remain separate from
future private-chain lifecycle support.

The client refuses wrong project/runner/generation/foundation responses and never
automatically repeats a mutation. Lookup-only retry retains the original inputs.
The first synthetic lost-reply test caught an omitted replay header on the new
transport path; its failure log is preserved. The narrow fix passes the replay
and existing creation-replay tests. JSON ambiguity, keys, unknown fields, oversized
documents, symlinks, FIFOs and directories refuse before opening a CLI registry.

The full managed store regression passes330 top-level tests/1,367 including
subcases in183.165s, closing M353's pending full-store gate. The five affected
client/CLI/protocol/access/planner packages pass157 top-level tests/765 including
subcases. Managed API race checks pass14 top-level/67 including subcases. Vet,
both skill validators and a built offline help/planner walkthrough pass; the
walkthrough opens no registry. The uninstalled collector SHA256 is
`e1b749a781a9841f799ccb94548645b759c3dabc465e6008d0f8ac01988d1400`.

The internal schema41 contract is unchanged. M353's original40 snapshot/reproduction
and successor-refusal evidence remain applicable. This is not installed-runner,
private-runtime publication or simultaneous-project VM qualification. Production
ships, installed plugins/services, tmux and excluded directories were untouched.

Next: verified immutable private-runtime inputs and one coherent durable chain/
gateway/member lifecycle, followed by isolated VM publication/recovery/isolation.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-25-private-chain-public-reservations`.

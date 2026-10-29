# Retained gateway controller and fresh VM preparation

The explicit `--gateway-resume` qualification variant runs the complete original
private-chain peer/recovery case, requires its clean stop, then opens that exact
chain with the separate resume candidate. Before creating or starting a replacement,
the controller checks the original container ID, image, project label, stopped
state, resource contract, environment, exact mounts, root/chain device and inode,
and original proof hash. It mounts only the selected chain plus read-only public
inputs; no key or pier directory is exposed to the replacement.

Forty-nine inert Python tests pass, including controller ownership tampering and
proof that the new guest changes only the appended resume hook. Payload comparison
finds exactly three added helpers and one changed guest entry; every other input
matches the qualified VM09 appliance. The new payload has12,360 files. Existing
initializer source and prior disks remain unchanged.

Fresh VM10 was admitted with no NIC, its own rootless Docker engine,12GiB host
scope/no swap/two CPUs,7GiB guest and a one-hour deadline. Runtime is in progress,
not a pass. The test will check unchanged chain/head/public keys, refusal of RPC
transactions and mining, and clean replacement shutdown. Independent stopped-disk
resume auditing is the next source task. Public runner lifecycle and peer
reconnection through a replacement gateway remain separate gates.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-25-private-gateway-resume-preparation`.

# Docker channel trial 03

The isolated run completed in 1510.64 seconds and **failed the peer-message gate**.
Both new fake ships booted, identity/session checks passed, and each same-ship
channel poke returned the expected Gall application acknowledgement. The first
remote channel submission was refused before delivery. No peer receipt or restart
pass is claimed. Both ships stopped cleanly, the VM powered off, and its exact
resource scope is inactive. All original disk and extracted evidence hashes were
verified. [Evidence](2026-10-22-fake-network-channel-trial.json).

A separate fresh native control reproduced HTTP 500 with body `crud!` for the
remote request, following successful local positive/negative controls. Its channel
was deleted and the fake stopped cleanly. Gall source rejects foreign `%poke-as`
operations; Eyre emits that operation for these requests. This is evidence against
using the direct remote channel probe, not evidence of broken Docker transport.

Before this trial, the restart harness was corrected: creation arguments cannot
resume an existing pier. It now preserves the stopped boot container and creates
a resume container without `-F`, `-B` or `-c`, checking the same pier device/inode,
anchor namespace and original sessions. This run never reached that step.

Next: qualify the built-in local Khan thread's ordinary Gall poke with exact
positive and deliberate negative receipts, then repeat the private Docker case.
No protected ship, installed plugin or production Docker engine was accessed.

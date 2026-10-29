# Docker peer messaging and session recovery

2026-10-22. **Passed in a fresh private VM:** bidirectional application messages
and same-pier process/container recovery with the original Eyre sessions.
[Machine-readable verified evidence](2026-10-22-docker-peer-recovery.json).
The run took 1389.05 seconds (23 minutes 9 seconds), including both fresh boots.

## Observed outcomes

| Check | Result |
|---|---|
| Same-ship controls | One accepted nonce-bearing poke on each fake |
| Peer messages before restart | `~zod` → `~nec` and `~nec` → `~zod` acknowledged |
| Deliberate remote rejection | Actual `poke-hi-fail` returned through Gall |
| Restart | `~nec` stopped cleanly; a new runtime container resumed its existing pier |
| Durable identity | Pier device/inode unchanged; HTTP/admin identity rechecked |
| Network continuity | Both members retained the original anchor namespace |
| Authentication continuity | Original session jars reused; no new login after restart |
| Peer messages after restart | Both directions acknowledged with fresh receipts |
| Session separation | Each ship refused the other ship's cookie |
| Shutdown | Three clean ship stops including the planned restart; anchor stopped; VM powered off; exact scope inactive |

All seven application outcomes were independently checked against their bindings:
two local controls, four peer messages and one intentional rejection. All 16
extracted result files (51,661 bytes), original disks and backing artifacts were
hash-verified. Original evidence is retained, including the stopped boot container
and the resumed container. The converted extraction image is derived and may be
released after verification; compatibility paths preserve access to relocated
originals.

## Isolation and protocol

The no-NIC QEMU guest used its own rootless Docker 29.9.1 engine and the pinned
Vere image `4.6-5a7614f`. Its outer scope was admitted at 12 GiB, no swap, one CPU
core and 128 tasks. The guest had 7 GiB RAM and two emulated CPUs. No host Docker,
home, SSH agent, desktop socket or existing pier was mounted. The anchor used
`--network none`; only loopback was present, with no IPv4 route or published ports.
Each ship had its own pier and HTTP port (8081/8082), with discovered and verified
administrative ports (12321/12322).

A built-in local `khan-eval` thread sends an ordinary Gall `%helm-hi` poke, waits
for its acknowledgement, then returns the unique expected receipt. The typed
`ted-eval` input and constrained generated program are covered by the passing
[native peer control](2026-10-22-gall-peer-control.md) and six offline tests.
Generic HTTP success cannot establish completion; a deliberate rejection must
contain the precise Gall failure labels. No uncertain request was retried.

The previous [remote Eyre channel trial](2026-10-22-fake-network-channel-trial.md)
remains failed: Eyre's converted poke maps to `%poke-as`, which Gall refuses for
foreign requests. The corrected local thread uses an ordinary Gall poke instead.
Earlier generic-noun Spider timeouts and the native pair's 4 GiB OOM are retained
as failures, not relabeled as successes.

Resume uses the existing pier without `-F`, `-B` or `-c`. Replaying a container's
original creation command would fail on that pier. The stopped creation container
is preserved; this result qualifies deliberate process/container replacement,
not arbitrary replay of bootstrap arguments.

## Remaining acceptance

This completes the first transport rung and proves a subset of restart recovery.
Persistent application data, subscriptions/watches and client reconnect remain
untested. The next checkpoint should exercise those, then repeat duplicate fake
identities in a second project to prove separation. Anchor loss/recreation,
endpoint publication, runner integration, power loss, two-runner projects and
private Ethereum chains remain separate gates. Shared loopback trusts members
of the same development project; it is not a hostile-application sandbox.

No operations were performed on `~lismyl-lismyl`, tmux ships, `~/temp-ur`, installed
plugins or production Docker. No production feature or release was installed.

Original disks, run metadata, extracted receipts and verification scripts are
retained under `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-22-network04-passed/`, with hash manifests.

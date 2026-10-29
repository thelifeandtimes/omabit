# VM09 stops at fresh-host activation before key proof

M422, 2026-10-26. The frozen M420 appliance passed source/build/host-refusal and
storage admission, then ran once in a fresh no-NIC QEMU guest with its own
rootless engine. Fresh host installation reached management activation and
failed; gateway initialization and member-key proof were never invoked. The
guest powered off cleanly in310.294s with no timeout or storage failure. Its
supervisor is terminal (failed exit-code); its VM scope is inactive.

Two independent read-only inspections leave both disk and pristine base hashes
unchanged and power off cleanly. The first projects fixed source diagnostics;
the second additionally retains the bounded426-byte installer message privately
because the projection omitted its underlying system command error. That private
console/message is not copied into public evidence. Neither reader accesses
ship keys, piers or chain payload. The exact message reports Go's ErrWaitDelay
from systemctl --user --machine=lab@.host enable --now urbit-runner.service.
The systemctl stderr records creation of the expected service-enable symlink.

The installed Go1.27.3 os/exec source defines ErrWaitDelay as successful process
exit with output pipes still open after WaitDelay. Source SHA256:
3ec96eeef97a375aa5e39663961f8adf3177db9449dacd3c407fd258bc9a2498.
Omabit currently treats that output-drain failure as a failed activation. It
preserves the partial installation and retained generation, correctly avoiding
an uncertain mutation retry. This does not prove the service was healthy; no
new activation acknowledgement or live management check was completed.

The next fix should distinguish confirmed successful process exit with incomplete
output from unsuccessful/cancelled/oversized commands. Only an explicit service
effect whose output is not consumed may use that distinction, followed by the
existing service/peer/executable/schema/token verification. JSON-producing and
read-only probes must continue refusing incomplete output. Do not relax the
output bound, command deadline, activation verification or retry the failed VM.

The failed disk is durably retained byte-for-byte at private managed-gateway-
disk-20261026-06, with its compatibility path preserved. SHA256:
88675da4cbd64002c38d3065e1b2815048eb2440e94da95de222a4079fa85216.
All owned units are terminal. The8GiB admission and2GiB running reserve remain.
Audits08/09 may be retained before the next fresh appliance to restore temporary
headroom; do not lower admission. No protected ship, tmux, ~/temp-ur, installed
plugin or host Docker was accessed.

M422 supersedes M420's pending VM09 status and M421's unknown installation cause.
The key-proof runtime gate remains unexercised. M417's earlier ordinary gateway
recovery evidence and M421's separate source-qualified filesystem helper remain
valid. This failure is progress on an isolated installer defect, not a blocked
goal or a private member/runtime pass.

# Initializer readiness and observer timing

Fresh supervised run03 used corrected appliance25 in a no-NIC VM with its own
rootless engine. The controller remained active through a recorded failure and
clean shutdown (1317.18 seconds). The fixture failed at its 12-minute operation
deadline; recovery and second-manager verification never ran.

Audits05/06 preserve the same disk and base hashes. The initializer generated
fresh keys, configured the private chain, completed its public checks and wrote
gateway-ready.json/GATEWAY_READY. It later stopped with exit0 and no evaluator,
failure or OOM evidence. This is progress beyond M401, but is not a successful
managed readiness receipt: source phase is started, with three operations and
zero recovery observations. No member ships were booted.

A guarded UID1000 child of the read-only audit PID1 ran the actual input verifier
twice on the sealed public input bundle: 12,352 files / 834,044,028 bytes. The cold
pass took40.977 seconds and the warm pass10.953 seconds. Neither input programs
nor initialized private chain/key payloads were read or executed. This timing VM
has one CPU/2GiB, while the runtime VM has two vCPUs/7GiB; these measurements do
not establish the exact live observer timing or prove the only failure cause.
They demonstrate why bulk preparation must not start a25-second mutable-snapshot
clock. The observer previously included both full passes inside that window.

The stopped disk is privately retained unchanged, with compatibility path and
SHA256245f01112dd39495f3e4598065005f6b282726acdf72a5b3b02ec50fe4225cef.
The transient supervisor is terminal with exit1/no retry; all VM/audit scopes are
inactive. M404 supersedes M402's unattempted corrected runtime and M403's next-run
pending statement only. Managed readiness/recovery remain unqualified. Protected
ships, tmux ships, ~/temp-ur, installed plugins and production services stayed excluded.

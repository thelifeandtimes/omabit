# Public member retry40: bounded two-core emulation

This new overlay uses exactly appliance39's887 frozen guest source/artifact inputs.
The only runtime-configuration change is the explicit two-host-core cap and
multi-thread TCG, still two guest CPUs. Memory remains12GiB, guest RAM7GiB, no swap,
no NIC,4500-second outer deadline, private disks and private rootless Docker.
Actual scope admission confirms200% CPU and all other resource bounds.

The host driver now accepts `boot --cpu-cores 2` only for explicit ship runtime
cases. Its default remains one core/single-thread TCG; arbitrary CPU caps and
implicit increases for non-runtime images are refused. Three CPU admission and
three public-entry tests pass. The draft initially lacked the normal directory
depth for its repository-root import; the corrected layout and both attempts are
retained. This changes no production engine API timeout or ownership/freshness
rule. The running retry tests a resource-sensitivity hypothesis after M214, not
a proven fix. The separate source change M210 is already in appliance39.

The independent member/application verifier is prepared, syntax checked, unrun.
No whole public lifecycle pass is claimed until stopped-disk verification.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-public-members40-preparation`.
Driver source/tests: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-vm-cpu-admission`.

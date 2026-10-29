# Public recovery retry38: launch observation failure

The immutable appliance36 retry shut down cleanly after713.68seconds, QEMU exit0,
outer exit1. It failed before application checks or public recovery: the first
member was created and started, but its final ownership observation was refused.
The original receipt remains interrupted at phase `started` with its exact
workspace/container identity. The generic message does not identify which
ownership/freshness check failed. A slow observation is a hypothesis only.

Independent stopped-disk audit confirms schema40, one project, two members, zero
ordinary instances, five successful setup operations and one interrupted launch.
Journal and operation runtime mirrors match exactly. Three earlier observations
were complete. Bridge/anchor/member creation and initial Starts happened once.
The anchor stopped cleanly; the member did not. Docker logged forced termination
after its300-second Stop timeout; exit137 with OOMKilled false is preserved. Clean
VM shutdown must not be confused with a clean ship stop.

The original overlay, extracted databases/results/journal and source appliance
remain retained. Hashes were checked before removing only the regenerable12GiB
raw conversion. Extraction first refused insufficient temporary quota; verified
duplicate copies of two completed probe artifact sets were replaced with links
to their retained originals. The unchanged storage threshold then passed.
The first independent audit used the planned rather than observed preparation
field for the anchor ID; both attempts remain, with the corrected audit passing.

This supersedes M206's running status, not M192's separate private recovery pass
or M205's initial public lifecycle pass. M198's read-cost fix was not reached.
Next: distinguish observation refusal reasons without weakening freshness or
ownership checks, then rerun in a new isolated VM.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-public-recovery38`.

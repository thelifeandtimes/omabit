# Interrupted recovery appliance33 preparation

Fresh schema40 appliance33 is frozen with 833 source hashes, including all Go
inputs for the actual collector/status path. Artifact hashes and embedded runner,
collector, test binary and guest fixture bytes match the retained image. The test
binary refuses execution outside its disposable guest admission boundary.

The no-NIC QEMU run is admitted under a dedicated user scope with a 12 GiB memory
ceiling, no swap and one CPU core. It sees only its new lab disks and read-only
tools/artifacts. Existing ships, installed plugins and host Docker are excluded.

This fixture loses three replies after actual successful actions (Stop, Create
with returned ID, Start), closes/reopens the store at every interruption, then
uses two explicit observations and three explicit continuation attempts. It also
checks the actual public collector status output after the final application
cycle. The independent stopped-disk verifier is prepared; it is not yet run.

The source full40 regression and focused public status gate already pass (M178,
M180). Runtime completion, independent disk checks and public mutation routes
remain pending. Store reopening is not an OS runner-service restart.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-anchor-continuation-preparation`.
Run: `/tmp/omabit-import-guest-run-20261024-33`.

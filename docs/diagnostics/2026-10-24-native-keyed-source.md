# Reproducible native keyed recovery qualification stage

The passing M237 fixture is integrated as `--stage native-peers` in the private
chain qualification runner. It accepts explicit compile/deployment/binary/pill
inputs, records exact helper hashes and creates fresh private keys/piers. Existing
foundation stages keep their4GiB/180-second limits. Native peers require8GiB
available before admitting6GiB/no swap/one core and a900-second outer deadline.

Two narrow source tests pass: host refusal before process/HTTP construction and
actual named-pier boot/reopen commands with creation-only flags absent on reopen.
Python/Node syntax and changed-file whitespace checks pass. The applied README
documents the official runtime's explicit Lens first shutdown and the distinction
between private qualification and a production/Docker devnet. No production
runner, plugin or storage schema changes are made.

Fresh run06 is admitted from this exact applied source. Independent input checking
matches its five helper files, frozen Node driver and outer script to source.
Its full runtime result remains pending. Source before/after files, tests and
admission metadata are retained.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-native-keyed-source`.

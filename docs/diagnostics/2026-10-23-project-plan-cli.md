# Offline fake-project plan in the collector CLI

`project-plan` exposes the existing pure planner without opening collector state,
contacting a runner, reserving ports or dispatching a runtime. It accepts the full
fresh-galaxy member set, display label and explicit shared-administration flag.
JSON/text output includes canonical members/internal ports and a deterministic
review digest. The output explicitly says host reservations were not checked and
runtime authority was not granted. Runner/socket selectors are refused.

`help projects` progressively discloses this boundary and points to command help.
Input checks cover invalid/duplicate identities and ports, missing acknowledgement,
extra arguments, forbidden runner selection and canonical ordering. The full
collector race suite passes in7.847s and vet is clean. A built-CLI walkthrough in
new user/network/mount/PID/IPC/UTS namespaces passes JSON/text planning, topic help,
duplicate-port refusal and runner-selection refusal. No host home, /run, Docker,
SSH or keyring is mounted into that walkthrough.

This exposes the existing pure planner for review only; it is not public runner
project registration or lifecycle control. Prepared VM23 predates this CLI addition
and its application fixture does not depend on it. No installed collector changed.

Example using the development binary:

```sh
urbit-collector project-plan --label demo \
  --member demo-zod:~zod:18381 --member demo-nec:~nec:18382 \
  --shared-administration
```

The operator skill now routes offline planning before runner enrollment and links a
focused project reference with the tested command and authority limits. Four
release-package tests pass, references resolve, and the two plain-scalar
frontmatter fields were manually reviewed. The skill-creator validator was
attempted with system and bundled Python but both lack PyYAML; neither attempt is
reported as a pass. Logs and source hashes are retained in
`operator-skill-update.json` beside this checkpoint's evidence. No global skill was
installed.

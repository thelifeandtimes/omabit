# Guided fake-project creation (unreleased)

Setup → Project networks → Create a connected fake project creates the chosen
fresh galaxies together. The fake-ship form links here. Choose a short project
name, distinct identities (default `~zod, ~nec`), and a free consecutive range of
host HTTP ports. Review the members and consent, then create. Each project uses
the existing private bridge and shared network namespace needed by fake Ames.
Members share administrative loopback access and must trust one another.

The CLI exposes the same workflow:

```sh
urbit-collector --runner local --expected-runner-id RUNNER_ID project-create \
  --label demo --member demo-zod:~zod:18381 --member demo-nec:~nec:18382 \
  --request-id demo-initial-1 --shared-administration --confirm-start
```

Use the actual runner ID from enrollment. The command emits JSON progress lines
(or text with global `--format text`). It reserves the complete membership,
downloads and verifies the release-pinned Vere image and core bootstrap pill if
needed, prepares the bridge/anchor and member storage, boots every member, then records
identity/readiness. Initial readiness waits up to ten minutes using read-only
checks; it does not repeat Start. The full guided command has a twenty-five-minute
deadline. Success describes the completed check, not perpetual live health.

Keep the exact arguments and request ID. After a lost reply, explicitly rerun with
`--resume`: reservation lookup is replay-only and subsequent steps use their
original deterministic IDs. Failed or interrupted operations stop progress. A
new ID is not a repair. An unrelated operation or truncated history also stops
this coordinator; inspect `project-status PROJECT` and use explicit lifecycle
or recovery commands. No automatic destructive cleanup is performed.

Current scope is 2–16 fresh galaxies with the pinned project bootstrap. Existing
ordinary fleet piers are not moved into a project by selecting a folder. Import,
dynamic membership, normal fleet projection and simplified member lifecycle
controls remain separate work. The lower-level `project-plan`, `project-register`,
`project-operate` and recovery commands remain available.

Dependency preparation is an explicit consented cache step before the first network
effect. It never starts a ship and only accepts the release pins. Existing verified
inputs are reused; a mismatched cached artifact is refused unchanged. A download
failure preserves the reservation and can be continued explicitly with the same
creation intent. Resume after network effects does not repeat cache preparation.

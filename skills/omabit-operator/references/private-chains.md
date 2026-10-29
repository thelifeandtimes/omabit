# Review and reserve a private local chain

Use the intended collector generation's `private-chain-plan --help`. This source
command is offline; an older installed build may not have it. Keep that limitation
visible instead of changing the selected installation or substituting Docker calls.

```sh
urbit-collector --format text private-chain-plan --label local-apps \
  --foundation /absolute/public-foundation.json \
  --member dev-zod:~zod:18381 --member dev-nec:~nec:18382 \
  --shared-administration
```

The foundation file contains public version1 metadata for the qualified local
chain1337 profile: genesis plus nine contract addresses/code hashes. In the source
checkout, `internal/privatechain/testdata/qualified-foundation.json` is a public
example; `docs/private-chain-project-contract.md` defines its limits. A manifest
is review data, not a fresh observation of an endpoint. Never pass a key, chain
database or pier as this file. The initial profile describes fresh `~zod`/`~nec` only.

Omit runner selection. The command reads the named metadata file and creates no
registry, connection, reservation, key or runtime. JSON output preserves the full
plan and review digest; text is for reading. Requested HTTP ports have not been
checked or published. Members acknowledge shared administrative loopback access;
RPC/Ames publication is disabled in this plan.

`private-local-keyed-v1` is distinct from fake and livenet classifications.
`project-register` refuses it. Use the separate reservation commands below when
the selected runner advertises them. Private-chain runtime lifecycle is not yet
implemented; reservation support does not imply Start/Stop support.
For authorized source qualification, use the documented preparation, no-NIC VM
launch and stopped-disk audit in `scripts/qualification/private-chains/README.md`.
These commands create disposable test resources and are not fleet health checks.

Two cloned development chains can share genesis and ship names. Runtime admission
also needs the exact runner, project and chain generation; a label, URL or `~zod`
identity alone cannot establish that binding. Initial keyed shutdown with the
qualified official runtime needs an acknowledged hood Lens exit and observed
exit0. A timeout or sent exit request is not success, and is not permission to
force-stop a different process or regenerate the chain, key or pier.

## Reserve and inspect the reviewed plan

Save the full JSON output from `private-chain-plan` and keep its review digest.
Text output is for reading, not registration. Use the same intended tool generation
and existing task authorization; acknowledgement flags do not require asking again.

```sh
urbit-collector --config-dir REGISTRY --runner NAME --expected-runner-id RUNNER_ID private-chain-register \
  --plan /absolute/reviewed-private-plan.json --review-digest REVIEW_DIGEST \
  --request-id ORIGINAL_REQUEST --confirm-reservation
urbit-collector --config-dir REGISTRY --runner NAME --expected-runner-id RUNNER_ID private-chain-projects
```

Registration requires operator access and `private-chain-registration-v1`. It
atomically reserves the complete member names/HTTP ports against private projects,
fake projects and standalone ships. It checks host port availability, but creates
no keys, chain, gateway, piers or containers. Preserve the returned project ID,
generation, foundation digest and operation receipt. Existing ships cannot join
through this initial fresh-project contract.

After a lost or malformed reply, preserve the exact runner, actor, plan, digest
and request ID, and add only `--replay-only`. This requires the runner's existing
creation-replay capability and never creates a missing reservation. A missing
receipt or interrupted connection does not authorize a new request ID.

`private-chain-projects` requires `private-chain-reservations-v1`; observer access
is sufficient. It reads consistent saved claims and public review metadata, reports
runtime observation as `not-performed`, and exports no private runtime tokens.
An unavailable capability does not mean no project exists. Do not route private
IDs through fake-project actions or substitute Docker commands. Schema41 and its
original40 refusal are source-qualified; installed rollout and runtime delivery
remain separate gates.

## Inspect prepared runtime inputs

Use `private-chain-inputs --help` from the intended development generation before
inspecting an explicitly selected local sealed bundle. Omit runner/socket selection:
this command hashes files only and opens no collector registry.

```sh
urbit-collector --format text private-chain-inputs \
  --directory /absolute/sealed-inputs \
  --manifest-sha256 SEPARATELY_REVIEWED_HASH \
  --foundation-digest REVIEWED_PLAN_FOUNDATION_DIGEST
```

Keep the review hash separate from an untrusted bundle's own claim. A pass reports
`runtime_authorized: false`; it neither authenticates a publisher nor verifies a
live chain or engine. Current gateway inputs are guest-only qualification sources.
Preserve failed/partial bundles and prepare a new generation; never chmod existing
projects, piers or installed files to satisfy input checks. See the source checkout's
`docs/private-chain-runtime-inputs.md` for the version1 format and delivery limits.

# Checkpoint 6Q: fresh comet bootstrap qualification

The qualification workflow is implemented and passes local tests, vet and relevant
race checks. **Live qualification passed**, run `cb31b1d255ffe45103cf90ed7458a593`;
see the [acceptance record](../diagnostics/2026-09-29-fresh-comet.md). This is the first newly generated
non-fake identity workflow; managed comet creation remains gated on its result.

## Scope

`qualify-comet` creates one **new disposable comet** using the pinned Vere 4.6
binary and verified upstream pill. Vere generates the identity internally. No
existing pier, identity name or key input is accepted.

- Holds management and qualification locks through cleanup.
- Uses the existing dedicated rootless engine and fixed 4 GiB/no-extra-swap,
  two-CPU, 256-PID, loom-31 limits.
- Uses an owned IPv4 NAT bridge with normal outbound access, including Ames.
  Only ephemeral **loopback HTTP** is published. No host UDP/Ames port, clearweb
  binding, remote exposure or claimed peer-connectivity result is added.
- Creation uses `-c` with the verified pill, without fake/local-only flags.
  Replacement resume omits creation/pill/key flags and uses the same pier.
- Reads only credential-free public `/~/host` over the inspected loopback binding.
  The response must have the pinned comet name shape and remain exactly equal
  across runtime replacement. Proxies, redirects and cookies are excluded.
- Cleanly stops both runtimes and removes all three containers and their bridge.
  The private identity-bearing pier and report are retained.
- Journals provisioning uncertainty before dispatch. Partial creation, a failed
  resume or changed identity does not trigger a retry; cleanup does not erase
  the unresolved provisioning outcome.

Bootstrap and resume each have a **20-minute readiness bound**. Both require fresh
admission of at least **5 GiB available RAM / 8 GiB free disk**. The last workstation
check had approximately 9.5 GiB available RAM, but the runner checks again.

Reviewed runtime usage:
<https://github.com/urbit/vere/blob/5a7614f/pkg/vere/main.c> documents `-c <my-comet>`
for comet creation, `--ames-port`, and initialized-pier resume. Public-name shape
checking is not a general @p decoder, proof of Ames peering, or production support.

## Checks passed

Fixtures cover fixed command construction, no supplied identity/key, creation-only
pill mounting, exact identity retention, replacement endpoint discovery, unsafe
HTTP/UDP publication refusal, managed-context exclusion, resource refusal, readiness
timeout, redacted invalid responses, cleanup, and unresolved/missing outcome
history. Existing fake lifecycle/HTTP/import workflows remain regression-tested.

## Live gate

Review from any directory:

```bash
/home/thelifeandtimes/repos/omabit/build/urbit-runner comet-plan --format text
```

Press **Ctrl+C in the management `serve-fake` terminal**, then install the rebuilt
qualifier. Existing managed runtime containers continue independently.

```bash
sudo /usr/bin/install -D -o root -g root -m 0755 \
  /home/thelifeandtimes/repos/omabit/build/urbit-runner \
  /usr/local/lib/urbit-runner/checkpoints/6q/urbit-runner
```

Run the new-comet qualification as the dedicated account:

```bash
sudo runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/6q/urbit-runner \
  qualify-comet --acknowledge-disposable-run --format text
```

Expected: `Profile: fresh-comet-bootstrap`, `Checkpoint: PASSED`, a public comet
name, initial boot and same-identity resume completed, uncertainty false, three
containers removed, and network cleanup complete. Share this qualification output.
If it fails, let cleanup finish and retain the report rather than trying a new
creation automatically.

After qualification exits, restore management and leave it running:

```bash
sudo runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/6q/urbit-runner \
  serve-fake --acknowledge-development
```

The management endpoint still serves the accepted fake/imported-fake workflows.
Managed comet provisioning, external import, key bootstrap, host Ames networking
and routine maintenance remain subsequent increments.

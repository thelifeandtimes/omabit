# Checkpoint 4C: origin-bound Eyre ship sessions

Programmatic acceptance passed on the running development fake with the SSH
listener stopped. Both the explicit file adapter and desktop Secret Service
passed login, replacement, persistence across CLI processes, authenticated status,
desk scry, SSE subscription, logout, and cleanup. Further manual repetition is
optional. The protocol/storage boundaries are in [ship sessions](../ship-sessions.md).

## Reproduce the automated gate

From the checkout, with the known fake alpha still running at port 18081:

```bash
go build -o ./build/ ./cmd/...
python3 scripts/validate-eyre.py --acknowledge-development-fake
```

This is explicitly restricted to the known localhost fake `~zod` and its public
tutorial code. It uses temporary collector profiles, creates only owned HTTP
sessions/subscriptions, and cleans those up. Output contains readable PASS summaries
and desk names, never session credentials. It does not use SSH, Docker, Dojo, or
the Vere control socket.

## Use the new commands yourself

The existing root-owned checkpoint 4B binaries remain appropriate for runner/SSH
management. The new ship commands can run as your ordinary desktop user directly
from the checkout; the collector needs no administrative installation to do this.

In each new terminal, from the checkout, define:

```bash
shipcollect() {
  "$PWD/build/urbit-collector" --format text "$@"
}

shipcollect ship-enroll alpha-app \
  --url http://127.0.0.1:18081 --ship '~zod'
shipcollect ships
```

Enrollment is independent of the `workstation` runner entry and does not log in.
It defaults to your desktop Secret Service. For an actual headless environment,
explicitly select `--credentials file` at enrollment; see its plaintext storage
semantics in the protocol document.

Login reads the code from stdin. For this development fake the public tutorial
code is `lidlut-tabwed-pillex-ridrup`. For any real ship, enter its code privately;
do not paste it into chat, a command argument, or an environment variable.

```bash
read -r -s -p 'Ship web login code: ' SHIP_CODE
printf '\n'
printf '%s' "$SHIP_CODE" | shipcollect ship-login alpha-app --code-stdin
unset SHIP_CODE

shipcollect ship-status alpha-app
shipcollect ship-scry alpha-app --app hood --path /kiln/pikes
shipcollect ship-watch alpha-app --app hood --path /kiln/updates --events 1 --duration 15s
```

`SHIP_CODE` is a non-exported shell variable and is cleared after the pipe. The
collector stores only the resulting session. Expected: `authenticated`; desk data
including base/groups/webterm; and an initial subscription diff followed by normal
channel cleanup. A channel event may legitimately have ID zero.

For an ongoing subscription, omit `--events` and `--duration`. Ctrl+C cancels it and
attempts channel deletion. A `reconcile` error requires a fresh application
snapshot before a new watch; the generic collector cannot invent that app's rules.

## Logout and remove enrollment

```bash
shipcollect ship-logout alpha-app
shipcollect ship-forget alpha-app --url http://127.0.0.1:18081
```

Logout clears this enrollment's credentials and attempts to revoke only its own
server sessions. It reports if server-side logout was not confirmed. Cleanup
failures preserve retry intent; repeat `ship-logout` after resolving the storage
problem before forgetting the enrollment.

No runtime stop or bootstrap is part of this checkpoint. The next plan milestone
is the Omabit management interface built on these collector contracts.

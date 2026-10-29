# Theme and DMs private handoff checkpoint — 2026-10-21

The real Python receivers and direct client helpers passed against a fresh fake
ship, with an isolated Secret Service. This is a headless integration checkpoint;
the approval UI and complete application-connection lifecycle are not delivered.

## Executed environment

Run: `/tmp/omabit-auth-run-20261021-04`. The
[machine-readable receipt](2026-10-21-plugin-handoff.json) identifies the exact
source files and copied artifacts by SHA-256. Sources were uncommitted development
worktrees based on Omabit `b233128`, Theme `1719e9f`, and DMs `cf236fd`.

The run retained the [previous namespace/resource boundary](2026-10-21-isolated-auth.md):
fresh fake `~zod`, no existing piers or host network, no production home/service
sockets, no Docker, 4 GiB memory, no swap, one CPU core and bounded deadlines.
The same copied native Vere 4.6 executable and pinned pill were used; this is not
qualification of the pinned Docker image or a hosted provider.

The harness added synthetic account records, a private D-Bus session, and a new
GNOME Keyring Secret Service unlocked with a random test-only password supplied
on stdin. The host desktop bus/keyring was not mounted or contacted. No plaintext
credential fallback was used. The keyring and all test state belong to the lab.

Copied, headless plugin artifacts had distinct identities and labels:

- `thelifeandtimes.urbit-theme.auth-lab` — Network Theme (Auth Lab).
- `urbit.dms.auth-lab` — Urbit DMs (Auth Lab).

Their state and keyring application names also carry `.auth-lab`. No QML service,
desktop theme application, hook, launcher or current installed plugin was enabled
by the test. The actual DMs bridge used its existing injectable store interface
to suppress launcher registration while exercising real storage and transport.

## Passing gates

`TestIndependentSessionsIsolatedLive`: **PASS**, 0.31 seconds.

`TestApplicationPluginHandoffIsolatedLive`: **PASS**, 16.25 seconds:

1. Create separate Theme/DMs sessions from one authenticated owner session.
2. Send private offers to the actual Python receivers through stdin pipes.
3. Verify the receivers durably save their own keyring sessions and account
   metadata before returning correctly bound, nonsecret acknowledgements.
4. Reject a second Theme offer without replacing its existing account; the
   issuer revokes the rejected new session.
5. Close every issuing client. Start new plugin processes from stored credentials.
6. Theme reads `%settings` and receives its subscription acknowledgement. DMs
   reads `%chat` and receives a direct subscription acknowledgement. Neither
   needs a running Omabit service or an application-data proxy.
7. DMs' actual logout revokes only its cookie; replay of that cookie fails while
   Theme and the original owner session remain valid. Theme reads again, then
   its actual disconnect revokes its session independently.
8. Both actual helper implementations perform standalone URL + `+code` login,
   direct reads/subscriptions and logout, without invoking any Omabit binary.
9. Remaining issuer-owned sessions are cleaned up; the fake shuts down cleanly.

The receipt reports `test_exit: 0`, `clean_shutdown: true`, and no log truncation.

## Source changes and regression evidence

Both plugin branches now contain a private receiver, strict offer validation and
manifest-bound instance metadata. Default standalone behavior and default storage
names remain available. Assisted sessions carry an optional expiry honored on
lookup and in running transports; DMs also carries it into its browser handoff.

DMs account writes/removal now use a private cross-process lock. Its HTTP client
does not follow redirects or use proxy environment settings. Logout sends no
`all`/`sid`, retains credentials on uncertain remote outcomes and reports observed
revocation separately from rejection. Theme's existing installer inventory now
includes the receiver files and documentation while accepting older receipts.

Automated checks passed for malformed/oversized/duplicate offers, wrong profile,
public identity mismatch before cookie delivery, redirects, keyring failure,
account replacement refusal, expiry, private response output, separate credential
namespaces, direct logout and storage contention. DMs' 47 affected tests passed.
Theme's affected authentication/helper suites passed; corrected historical
installer fixtures and the final 17 installer/package tests passed. Eyre's race
suite and static checks are recorded alongside the source work.

The first combined run (`...-03`) stopped at the Theme subscription probe: the
probe supplied callbacks where Theme's API requires `threading.Event` objects.
Correcting the probe required no Theme watch change. That run also shut its fresh
fake down cleanly; its artifacts were retained.

## Remaining gates

- Durable issuer attempt/credential ownership, acknowledgement loss, cancellation,
  restart recovery, confirmed/pending revocation and deliberate reauthentication.
- User-visible request/approval management and actual independently labeled QML
  instances. Existing desktop plugins and their credentials remain excluded.
- Real plugin writes, complete stream/reconnect behavior, stale completions across
  profile switches, hosted compatibility and collection-wide stable/dev/edge UX.
- Plugin-owned backend installation and compatibility handling.

No shared client package has been extracted, and no release or production deployment
has occurred. See [the handoff contract](../application-handoff-v1.md) and the
[full active goal](../goal-progress.md).

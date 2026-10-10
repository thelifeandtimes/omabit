# First remote runner: sourcebox

## Installation and transport

The user installed the headless runner on sourcebox after adding RootlessKit and
slirp4netns through Omarchy. Platform/systemd/cgroup/user-namespace prerequisites
and Docker 29.7.2 passed preflight. Host installation returned ready, with dedicated
UID 954 and runner `runner-b4043fa2d0aee501aa2cf78322aed911`; engine, management and
dedicated SSH services were active. This was a fresh Omabit installation on a host
with an existing system Docker installation, which was preserved.

The existing tailnet provides connectivity. Tailscale Serve forwards tailnet TCP
22222 to the dedicated loopback SSH service; no administrator SSH tunnel is needed.
The desktop verified the supplied SSH host key and operator fingerprint
`SHA256:YPCW+HUbqQiqgClBJj4HXREyV5Immu5PiyloqxreS1w`, then enrolled `remote-test`.
Local and remote runner observations are available concurrently.

## Port-conflict discovery

`remote-smoke` registered as fake `~zod` on HTTP 18080, but initial Start
`op-a3781291416668cbd14aed54d07e459c` failed before Vere ran. Retained Docker state
reported created / exit 128 / OOMKilled=false, with a RootlessKit bind failure for
`127.0.0.1:18080`. Administrator inspection found an existing `docker-proxy` listener.
The new test Serve forward on 18080 was removed; the pre-existing listener was untouched.

Cleanup operation `op-798edc5379bcc806a46a934a61323299` removed the never-started
container. The registration returned to uninitialized with no recovery flag.
HTTP was changed to 18081 (configuration revision 2). Start
`op-ac2120d6be8801846b62a5b497c42cd9` succeeded at 20:17:10 UTC, about 52 seconds
after acceptance. Its initial container was
`844bfc433946b8c89a851eda2bf616762c9ba08641ca2dffc3c055bff9af0f74`.

The user added Serve TCP 18081 → `127.0.0.1:18081` and configured Browser host
`http://100.72.66.56`. Open ship displayed the `~zod` login page. A direct tailnet
GET to `/~/host` independently returned `~zod`. Runtime logs and resource readings
were available through the restricted SSH route.

## Explicit lifecycle check

- Stop `op-15ac8d173439ea8e31b27b881f468df2` succeeded cleanly at 20:23:05 UTC.
- The stopped container's bounded log tail was retained.
- Resume `op-c28725569c22d042e044de4afc8f0dbd` succeeded at 20:23:26 UTC.
- Same identity, path `fakeships/remote-smoke/pier`, configuration/applied revision
  2/2 and pier fingerprint
  `adbc2fc875a253495ed74237a909a7eee9aee7157a63ad644c847c657a5f4ab6`.
- Initialized replacement kind `http-restart`; container
  `e417c39907d6dafb6426a0bcd4e3a2be0ff2c197c20dee9ca55b15d2fdb3dd77`.
- Running/healthy, no recovery flag; tailnet HTTP identity still matches.

## Remaining checks and findings

The user confirmed successful browser login using Copy +code. The corresponding
secret-free login-code audit is `op-4fc443d0bb346ebb10514fbc5d08ce8b` at 20:30:27 UTC.
Runtime logs earlier contained HTTP 504 / docket asset-download retries and
`%retry-too-many`; individual application/desk readiness is not inferred from login.
Remote comet boot, host reboot, disconnect/reconnect, access revocation and
backup/restore remain later gates.

The pre-management-restart baseline is running/healthy at revision 2/2 in container
`e417c39907d6dafb6426a0bcd4e3a2be0ff2c197c20dee9ca55b15d2fdb3dd77`.

The user restarted only `urbit-runner.service`. Post-restart verification at
20:36:40 UTC confirmed the same runner/operator identities, exact same running
container, unchanged pier fingerprint/path and revision 2/2, no recovery/incidents,
and no `startup-resume` jobs. Tailnet HTTP still returned `~zod`. The management
restart gate passed without cycling the ship.

Release follow-ups exposed by this trial:

1. Fake/comet registration and configuration should check actual host listener
   availability in addition to runner reservations, and recheck before dispatch.
2. Report a bounded/sanitized Docker launch reason when a container never starts;
   an empty Vere log plus generic Docker exit requires unnecessary admin inspection.

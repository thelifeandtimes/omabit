# Checkpoint 2C: host-loopback HTTP

Prerequisite: the rootless setup and isolated lifecycle checkpoints have passed.
Run `35aac1083d71188d9782ec64b49ba771` supplied the accepted 2B evidence on this
host. This new checkpoint is implemented and fixture-tested, but its real Docker
networking path awaits your manual validation.

## What changes in this checkpoint

The new `qualify-http` command uses the same pinned image, pill, dedicated
account, and resource limits as 2B. It creates a **new fake pier**, not a copy or
continuation of the accepted run's pier.

Its workflow is:

1. Verify the private rootless engine and check unresolved prior work.
2. Verify the image and download/check the pinned upstream pill.
3. Run the accepted network-none cgroup/version probe.
4. Create a uniquely named, run-owned IPv4 NAT bridge.
5. Boot a fake `~zod`, publishing **only** `127.0.0.1:<ephemeral-port>` to its
   ordinary HTTP port `8080/tcp`.
6. From the host process, request `/~/host` and require HTTP 200 with identity
   `~zod`. Do not authenticate, store cookies, use a proxy, or follow redirects.
7. Stop/remove that container, start a replacement against the new pier, discover
   its assigned port again, and repeat the host check.
8. Stop/remove owned containers, remove the now-unused owned network, and retain
   the private report and fake-pier data.

**This profile allows normal outbound network access from the fake container.**
It is not the network-none test. Inbound publication remains loopback-only; no
LAN/tailnet address, wildcard host address, IPv6 publication, Ames UDP port, or
Vere control socket is exposed. Other local processes can access a loopback
publication while it exists; this is not application authentication.

### Why a normal bridge, rather than `--internal`

The initial design considered an internal bridge. The pinned Docker 29.7.2 source
only adds NAT port mappings for the endpoint selected as a gateway. Internal
bridges do not provide that gateway, so this would not exercise the required
host-port path. The implementation explicitly requests and verifies a normal
dedicated NAT bridge. It does not silently retry with broader networking after
a failed internal-network attempt.

Vere binds `0.0.0.0` **inside the container** for this test; Docker publishes only
on host `127.0.0.1`. Those are different network namespaces. `-F zod`, `-L`,
`--no-conn`, and `--no-dock` remain in effect. No real identity, login code, key,
or existing pier is used. Docker manages the temporary bridge inside its rootless
environment; the tool does not change system Docker, Tailscale, or global firewall
configuration.

The original `qualify-runtime` profile still uses network-none and runs pack. This
HTTP profile does not repeat pack or run meld. Both profiles share the account lock
and refuse unresolved prior resources, including networks.

## 1. Build and review

From the repository root as your ordinary user:

```bash
go test ./...
go vet ./...
go build -o ./build/ ./cmd/...
./build/urbit-runner http-plan
```

The plan is read-only. Expect:

- `profile: host-http` and `mode: disposable-fake-host-http`.
- The same candidate image, pill generation, hash, and byte count as 2B.
- The dedicated NAT bridge and its outbound access explicitly described.
- Only an ephemeral IPv4 loopback host port.
- At least **5 GiB available RAM** and **8 GiB free disk** at admission.
- Per-container limits: **4 GiB RAM, no additional swap, two CPUs, 256 PIDs**.
- `runtime_qualified: false`.

The image cache can be reused by Docker. This version downloads a fresh copy of
the approximately 207 MiB pinned pill into each new run directory. It does not
trust or adopt an old pier or silently follow a changed default pill URL.

## 2. Copy the binary

`install` below is the **file-copy command**; it also sets ownership and executable
permissions. It is not a rerun of rootless Docker installation.

```bash
sudo install -D -o root -g root -m 0755 \
  ./build/urbit-runner \
  /usr/local/lib/urbit-runner/checkpoints/2c/urbit-runner
```

No output normally means success. Rebuilding the repository binary alone does
not update this copied executable. Copy again after subsequent code fixes, with
no qualification run active.

## 3. Run as the dedicated account

```bash
sudo runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/2c/urbit-runner \
  qualify-http --acknowledge-disposable-run
```

The qualifier runs as `urbit-runner`, not root. The outer sudo authorizes switching
accounts. It discovers the UID and socket, rather than assuming UID 959 on every
host.

Expected progress:

```json
{"phase":"engine"}
{"phase":"image"}
{"phase":"pill"}
{"phase":"probe"}
{"phase":"network"}
{"phase":"host-http-boot"}
{"phase":"host-http-restart"}
{"phase":"cleanup"}
```

Allow several minutes for provisioning and initial boot. Readiness is bounded to
ten minutes per ship start; individual host HTTP requests have a five-second
limit. No response bodies, cookies, login codes, or raw runtime logs are printed.

## 4. Check the report

Expected successful top-level fields:

```json
{
  "profile": "host-http",
  "checkpoint_passed": true,
  "runtime_qualified": false,
  "cleanup_complete": true,
  "limits_observed": true,
  "pier_metadata_seen": true
}
```

There should be three container records: `probe`, `http-boot`, and `http-restart`.
All should be removed with resolved create outcomes. The two HTTP records should
have `host_http_verified: true` and a `host_endpoint` such as
`http://127.0.0.1:38101/~/host`.

The assigned port **may change** during replacement. The tool must rediscover it;
it does not assume the old endpoint still works. These reported URLs are historical
evidence, not a server left running for browsing after the command finishes.

`private_http_reachable` refers to the old in-container curl probe. It remains
false for these HTTP containers because the new test uses the host client instead.

The `network` record should have:

- A full ID and the run-specific name.
- `create_dispatched: true`.
- `create_uncertain: false`.
- `removed: true`.

The private `report_path` records these observations. The run's pill and fake
pier remain under its new qualification directory. The accepted 2B report and
pier remain untouched.

This demonstrates local HTTP publication/reconnect, not application login,
Tailscale/VPS routing, clearweb exposure, Ames delivery, desk completeness, meld,
reboot behavior, or production support. `runtime_qualified` therefore remains
false.

## Failure, cancellation, and retry

Ctrl+C requests bounded cleanup. Allow it to finish: stopping an active ship can
take up to the five-minute grace period, and uncertain Docker outcomes have a
separate recovery budget of up to eleven minutes per resource.

If cleanup is incomplete or a create is uncertain, stop and share the result.
Do not remove networks by broad label/name patterns, disconnect unrelated
containers, prune Docker, or edit report flags to bypass the prior-run gate.
An empty inventory does not prove an in-flight create has finished.

The new binary accepts older, completed 2B reports that lack the new profile and
network fields. To rerun 2B after a 2C attempt, use this updated binary with
`qualify-runtime`; an older copied 2B binary does not understand 2C reports.

Send the final report and any `qualification_error` output. In particular, confirm
both host HTTP checks and complete removal of the three containers and the network.

## Development evidence

Tests use fake Docker responses and temporary files for resource workflows, plus
real temporary loopback HTTP servers for the credential-free client. They cover
unsafe publication/identity responses, redirects, request cancellation, changed
bindings, normal and uncertain cleanup, and compatibility with accepted 2B report
shapes. They do not create real Docker networks or boot ships.

Relevant pinned networking implementation:

- [Bridge gateway and port mapping](https://github.com/moby/moby/blob/6a43e3d5afddf4111da0f864bbc7cae5d7e95001/daemon/libnetwork/drivers/bridge/bridge_linux.go)
- [NAT binding selection](https://github.com/moby/moby/blob/6a43e3d5afddf4111da0f864bbc7cae5d7e95001/daemon/libnetwork/drivers/bridge/port_mapping_linux.go)
- [Loopback publication documentation](https://docs.docker.com/engine/network/port-publishing/)

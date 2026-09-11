#!/usr/bin/env python3
"""Exercise Tend's owner-authoritative protocol on three running fake ships."""

from __future__ import annotations

import argparse
from concurrent.futures import ThreadPoolExecutor
from dataclasses import dataclass
import importlib.util
import json
from pathlib import Path
import re
import subprocess
import sys
import time
import uuid


ROOT = Path(__file__).resolve().parents[1]
TRANSPORT_PATH = ROOT / "omarchy-plugin" / "transport" / "eyre_client.py"
SHIP_PATTERN = re.compile(r"[a-z-]{1,255}")
URBIT_DATE_PATTERN = re.compile(
    r"^~([0-9]+)\.([0-9]+)\.([0-9]+)\.\.([0-9]+)\.([0-9]+)\.([0-9]+)(.*)$"
)


class HarnessFailure(RuntimeError):
    """A release invariant failed."""


def load_transport():
    spec = importlib.util.spec_from_file_location("tend_multiship_transport", TRANSPORT_PATH)
    if spec is None or spec.loader is None:
        raise HarnessFailure(f"cannot load Tend transport from {TRANSPORT_PATH}")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


transport = load_transport()


@dataclass(frozen=True)
class Endpoint:
    role: str
    ship: str
    config: Path
    cookie: Path
    pier: Path


def operation_id(label: str) -> str:
    return f"fault-{label}-{uuid.uuid4().hex}"


def snapshot(endpoint: Endpoint) -> dict[str, object]:
    value = transport.scry_state(endpoint.config, endpoint.cookie)
    body = value.get("snapshot") if isinstance(value, dict) else None
    if not isinstance(body, dict) or not isinstance(body.get("lists"), list):
        raise HarnessFailure(f"{endpoint.role} returned an invalid Tend snapshot")
    return body


def accesses(endpoint: Endpoint) -> list[dict[str, object]]:
    values = transport.scry_accesses(endpoint.config, endpoint.cookie)
    if not all(isinstance(value, dict) for value in values):
        raise HarnessFailure(f"{endpoint.role} returned invalid Tend access metadata")
    return values


def task_list(endpoint: Endpoint, list_id: int) -> dict[str, object] | None:
    for value in snapshot(endpoint)["lists"]:
        if isinstance(value, dict) and int(value.get("id", -1)) == list_id:
            return value
    return None


def list_with_title(endpoint: Endpoint, title: str) -> dict[str, object] | None:
    matches = [
        value
        for value in snapshot(endpoint)["lists"]
        if isinstance(value, dict) and value.get("title") == title
    ]
    if len(matches) > 1:
        raise HarnessFailure(f"{endpoint.role} has duplicate fixture list titles")
    return matches[0] if matches else None


def access_for_ref(
    endpoint: Endpoint,
    host: str,
    host_list_id: int,
) -> dict[str, object] | None:
    normalized_host = "~" + host.removeprefix("~")
    matches = [
        value
        for value in accesses(endpoint)
        if value.get("host") == normalized_host
        and int(value.get("host-list-id", -1)) == host_list_id
    ]
    if len(matches) > 1:
        raise HarnessFailure(f"{endpoint.role} returned duplicate access records")
    return matches[0] if matches else None


def canonical_json(value: object) -> object:
    if isinstance(value, dict):
        return {key: canonical_json(item) for key, item in value.items()}
    if isinstance(value, list):
        return [canonical_json(item) for item in value]
    if isinstance(value, str) and (matched := URBIT_DATE_PATTERN.fullmatch(value)):
        parts = [str(int(part)) for part in matched.groups()[:6]]
        return f"~{'.'.join(parts[:3])}..{'.'.join(parts[3:])}{matched.group(7)}"
    return value


def canonical_list(value: dict[str, object]) -> str:
    comparable = dict(canonical_json(value))
    comparable.pop("id", None)
    return json.dumps(comparable, sort_keys=True, separators=(",", ":"))


def reminder_titles(value: dict[str, object]) -> list[str]:
    reminders = value.get("reminders")
    if not isinstance(reminders, list):
        raise HarnessFailure("Tend list has invalid reminder data")
    return [str(reminder.get("title", "")) for reminder in reminders if isinstance(reminder, dict)]


def poke(endpoint: Endpoint, action: dict[str, object]) -> None:
    result = transport.poke(endpoint.config, endpoint.cookie, action)
    if result != {"status": "ok"}:
        raise HarnessFailure(f"{endpoint.role} did not acknowledge a Tend action")


def wait_for(label: str, predicate, timeout: float, interval: float = 0.5):
    deadline = time.monotonic() + timeout
    last_error: Exception | None = None
    while time.monotonic() < deadline:
        try:
            value = predicate()
            if value:
                return value
        except Exception as error:  # Transient peer/Eyre state is expected while recovering.
            last_error = error
        time.sleep(interval)
    detail = f": {last_error}" if last_error else ""
    raise HarnessFailure(f"timed out waiting for {label}{detail}")


def wait_receipt(endpoint: Endpoint, op_id: str, timeout: float) -> dict[str, object]:
    return wait_for(
        f"{endpoint.role} receipt {op_id}",
        lambda: transport.scry_receipt(endpoint.config, endpoint.cookie, op_id),
        timeout,
    )


def rejection_reason(receipt: dict[str, object]) -> str | None:
    rejected = receipt.get("rejected")
    return str(rejected.get("reason")) if isinstance(rejected, dict) else None


def run_conn(endpoint: Endpoint, source: str, result: str, timeout: float) -> None:
    socket = endpoint.pier / ".urb" / "conn.sock"
    if not socket.is_socket():
        raise HarnessFailure(f"{endpoint.role} has no running conn.sock at {socket}")
    if "'" in source or not result.isidentifier():
        raise HarnessFailure("unsafe conn test source")
    request = f"[0 %fyrd %base %khan-eval %noun %ted-eval '{source}']"
    expected = f"[0 %avow 0 %noun %{result}]"
    try:
        jammed = subprocess.run(
            ["urbit", "eval", "-jn"],
            input=(request + "\n").encode(),
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=True,
            timeout=timeout,
        ).stdout
        response = subprocess.run(
            ["nc", "-U", "-W", str(max(1, int(timeout))), str(socket)],
            input=jammed,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=True,
            timeout=timeout,
        ).stdout
        rendered = subprocess.run(
            ["urbit", "eval", "-cn"],
            input=response,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=True,
            timeout=timeout,
        ).stdout.decode().strip()
    except (FileNotFoundError, subprocess.CalledProcessError, subprocess.TimeoutExpired) as error:
        raise HarnessFailure(f"{endpoint.role} conn control failed") from error
    if rendered != expected:
        raise HarnessFailure(f"{endpoint.role} conn control returned an unexpected result")


def hood_poke(endpoint: Endpoint, mark: str, noun: str, result: str, timeout: float) -> None:
    source = (
        "=/  m  (strand ,vase)  "
        f";<  ~  bind:m  (poke [~{endpoint.ship} %hood] %{mark} !>({noun}))  "
        f"(pure:m !>(%{result}))"
    )
    run_conn(endpoint, source, result, timeout)


def suspend(endpoint: Endpoint, timeout: float) -> None:
    hood_poke(endpoint, "kiln-suspend-many", "[%tend ~]", "suspended", timeout)


def revive(endpoint: Endpoint, timeout: float) -> None:
    hood_poke(endpoint, "kiln-revive", "%tend", "revived", timeout)
    wait_for(f"{endpoint.role} Tend revival", lambda: snapshot(endpoint), timeout, 1)


def snub(endpoint: Endpoint, peer: Endpoint, deny: bool, timeout: float) -> None:
    ships = f"[~{peer.ship} ~]" if deny else "~"
    result = "denied" if deny else "cleared"
    hood_poke(
        endpoint,
        "helm-ames-snub",
        f"[%deny {ships}]",
        result,
        timeout,
    )


def wait_access_status(
    endpoint: Endpoint,
    host: Endpoint,
    host_list_id: int,
    status: str,
    timeout: float,
) -> dict[str, object]:
    return wait_for(
        f"{endpoint.role} to report {host.ship} {status}",
        lambda: (
            found
            if (found := access_for_ref(endpoint, host.ship, host_list_id))
            and found.get("status") == status
            else None
        ),
        timeout,
    )


def wait_converged(
    owner: Endpoint,
    members: list[Endpoint],
    owner_list_id: int,
    timeout: float,
) -> dict[str, object]:
    deadline = time.monotonic() + timeout
    last_issue = "no state sampled"
    while time.monotonic() < deadline:
        try:
            authoritative = task_list(owner, owner_list_id)
            if authoritative is None:
                last_issue = "owner list is absent"
                time.sleep(0.5)
                continue
            wanted = canonical_list(authoritative)
            issues: list[str] = []
            for member in members:
                access = access_for_ref(member, owner.ship, owner_list_id)
                if access is None:
                    issues.append(f"{member.role} access is absent")
                    continue
                if access.get("status") != "online":
                    issues.append(f"{member.role} status is {access.get('status')}")
                    continue
                replica = task_list(member, int(access["alias"]))
                if replica is None:
                    issues.append(f"{member.role} replica is absent")
                elif canonical_list(replica) != wanted:
                    issues.append(
                        f"{member.role} replica revision {replica.get('revision')} "
                        f"differs from owner revision {authoritative.get('revision')}"
                    )
            if not issues:
                return authoritative
            last_issue = "; ".join(issues)
        except Exception as error:  # Transient peer/Eyre state is expected while recovering.
            last_issue = str(error)
        time.sleep(0.5)
    raise HarnessFailure(f"timed out waiting for all replicas to converge: {last_issue}")


def wait_invitation(
    member: Endpoint,
    owner: Endpoint,
    owner_list_id: int,
    timeout: float,
) -> dict[str, object]:
    def found():
        matches = [
            value
            for value in transport.scry_invitations(member.config, member.cookie)
            if isinstance(value, dict)
            and value.get("host") == f"~{owner.ship}"
            and int(value.get("host-list-id", -1)) == owner_list_id
        ]
        return matches[0] if len(matches) == 1 else None

    return wait_for(f"invitation at {member.role}", found, timeout)


def create_fixture(owner: Endpoint, title: str, timeout: float) -> int:
    poke(owner, {"create-list": {"operation-id": operation_id("create"), "title": title}})
    created = wait_for("fixture list creation", lambda: list_with_title(owner, title), timeout)
    list_id = int(created["id"])
    add_op = operation_id("seed")
    poke(owner, {"add-reminder": {
        "operation-id": add_op,
        "list-id": list_id,
        "title": "Owner seed",
        "tags": ["fault-matrix", "shared"],
        "base-revision": int(created["revision"]),
    }})
    wait_receipt(owner, add_op, timeout)
    return list_id


def invite_and_accept(
    owner: Endpoint,
    member: Endpoint,
    owner_list_id: int,
    can_invite: bool,
    timeout: float,
) -> None:
    invite_op = operation_id(f"invite-{member.ship}")
    poke(owner, {"invite-member": {
        "operation-id": invite_op,
        "list-id": owner_list_id,
        "ship": f"~{member.ship}",
        "can-invite": can_invite,
    }})
    invitation = wait_invitation(member, owner, owner_list_id, timeout)
    token = str(invitation["token"])
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        poke(member, {"accept-invitation": {
            "operation-id": operation_id(f"accept-{member.ship}"),
            "host": f"~{owner.ship}",
            "token": token,
        }})
        for _ in range(10):
            access = access_for_ref(member, owner.ship, owner_list_id)
            if access is not None:
                return
            time.sleep(0.5)
        owner_access = access_for_ref(owner, owner.ship, owner_list_id)
        owner_members = {
            entry.get("ship")
            for entry in (owner_access or {}).get("members", [])
            if isinstance(entry, dict)
        }
        owner_pending = set((owner_access or {}).get("pending", []))
        if f"~{member.ship}" not in owner_members | owner_pending:
            raise HarnessFailure(f"{member.role} disappeared from the owner ACL during acceptance")
    owner_access = access_for_ref(owner, owner.ship, owner_list_id)
    owner_members = {
        entry.get("ship")
        for entry in (owner_access or {}).get("members", [])
        if isinstance(entry, dict)
    }
    owner_pending = set((owner_access or {}).get("pending", []))
    retained = any(
        isinstance(value, dict)
        and value.get("host") == f"~{owner.ship}"
        and int(value.get("host-list-id", -1)) == owner_list_id
        for value in transport.scry_invitations(member.config, member.cookie)
    )
    raise HarnessFailure(
        f"timed out accepting the invitation at {member.role} "
        f"(owner-member={f'~{member.ship}' in owner_members}, "
        f"owner-pending={f'~{member.ship}' in owner_pending}, "
        f"invitation-retained={retained})"
    )


def remote_add(
    member: Endpoint,
    alias: int,
    revision: int,
    title: str,
    op_id: str,
) -> None:
    poke(member, {"add-reminder": {
        "operation-id": op_id,
        "list-id": alias,
        "title": title,
        "tags": ["fault-matrix"],
        "base-revision": revision,
    }})


def remove_member(owner: Endpoint, list_id: int, member: Endpoint) -> None:
    poke(owner, {"remove-member": {
        "operation-id": operation_id(f"remove-{member.ship}"),
        "list-id": list_id,
        "ship": f"~{member.ship}",
    }})


def delete_fixture(owner: Endpoint, title: str, timeout: float) -> None:
    found = list_with_title(owner, title)
    if found is None:
        return
    op_id = operation_id("cleanup")
    poke(owner, {"delete-list": {
        "operation-id": op_id,
        "list-id": int(found["id"]),
        "base-revision": int(found["revision"]),
    }})
    wait_for("fixture cleanup", lambda: list_with_title(owner, title) is None, timeout)


def endpoint_from_args(role: str, config: Path, cookie: Path, pier: Path) -> Endpoint:
    connection = transport.read_connection(config)
    ship = connection["ship"].removeprefix("~")
    if SHIP_PATTERN.fullmatch(ship) is None:
        raise HarnessFailure(f"{role} connection contains an invalid ship identity")
    if not cookie.is_file():
        raise HarnessFailure(f"{role} cookie jar does not exist")
    return Endpoint(role, ship, config.resolve(), cookie.resolve(), pier.resolve())


def run_matrix(args: argparse.Namespace) -> None:
    timeout = float(args.timeout)
    owner = endpoint_from_args("owner", args.owner_config, args.owner_cookie, args.owner_pier)
    member_a = endpoint_from_args("member-a", args.member_a_config, args.member_a_cookie, args.member_a_pier)
    member_b = endpoint_from_args("member-b", args.member_b_config, args.member_b_cookie, args.member_b_pier)
    endpoints = [owner, member_a, member_b]
    if len({endpoint.ship for endpoint in endpoints}) != 3:
        raise HarnessFailure("owner and participant configurations must name three distinct ships")
    for endpoint in endpoints:
        snapshot(endpoint)

    fixture_title = f"Tend fault matrix {uuid.uuid4().hex[:12]}"
    owner_list_id: int | None = None
    suspended: set[Endpoint] = set()
    snubbed: set[tuple[Endpoint, Endpoint]] = set()

    try:
        owner_list_id = create_fixture(owner, fixture_title, timeout)
        invite_and_accept(owner, member_a, owner_list_id, True, timeout)
        invite_and_accept(owner, member_b, owner_list_id, False, timeout)
        authoritative = wait_converged(owner, [member_a, member_b], owner_list_id, timeout)
        print("PASS invitation, ACL, complete-list replication, and initial convergence", flush=True)

        access_a = access_for_ref(member_a, owner.ship, owner_list_id)
        access_b = access_for_ref(member_b, owner.ship, owner_list_id)
        if access_a is None or access_b is None:
            raise HarnessFailure("accepted participants have no replica aliases")
        alias_a = int(access_a["alias"])
        alias_b = int(access_b["alias"])

        online_title = f"online-{uuid.uuid4().hex[:8]}"
        online_op = operation_id("online")
        remote_add(member_a, alias_a, int(authoritative["revision"]), online_title, online_op)
        wait_receipt(owner, online_op, timeout)
        authoritative = wait_converged(owner, [member_a, member_b], owner_list_id, timeout)
        if reminder_titles(authoritative).count(online_title) != 1:
            raise HarnessFailure("online participant edit did not converge exactly once")
        print("PASS online participant edit and three-ship convergence", flush=True)

        duplicate_title = f"duplicate-{uuid.uuid4().hex[:8]}"
        duplicate_op = operation_id("duplicate")
        before_revision = int(authoritative["revision"])
        remote_add(member_a, alias_a, before_revision, duplicate_title, duplicate_op)
        remote_add(member_a, alias_a, before_revision, duplicate_title, duplicate_op)
        wait_receipt(owner, duplicate_op, timeout)
        authoritative = wait_converged(owner, [member_a, member_b], owner_list_id, timeout)
        if int(authoritative["revision"]) != before_revision + 1:
            raise HarnessFailure("duplicate operation changed the canonical revision more than once")
        if reminder_titles(authoritative).count(duplicate_title) != 1:
            raise HarnessFailure("duplicate operation created duplicate reminders")
        print("PASS duplicate delivery idempotence", flush=True)

        stale_title = f"stale-{uuid.uuid4().hex[:8]}"
        stale_op = operation_id("stale")
        stable_revision = int(authoritative["revision"])
        remote_add(member_b, alias_b, max(0, stable_revision - 1), stale_title, stale_op)
        stale_receipt = wait_receipt(owner, stale_op, timeout)
        if rejection_reason(stale_receipt) != "stale-list":
            raise HarnessFailure("stale remote operation was not rejected as stale-list")
        authoritative = task_list(owner, owner_list_id)
        if authoritative is None or int(authoritative["revision"]) != stable_revision:
            raise HarnessFailure("stale operation changed canonical state")
        if stale_title in reminder_titles(authoritative):
            raise HarnessFailure("stale operation inserted content")
        print("PASS stale edit rejection without canonical mutation", flush=True)

        snub(owner, member_a, True, timeout)
        snubbed.add((owner, member_a))
        pressure_revision = stable_revision
        pressure_ops = [operation_id(f"pressure-{index}") for index in range(args.backpressure_count)]
        pressure_titles = [f"pressure-{uuid.uuid4().hex[:8]}-{index}" for index in range(args.backpressure_count)]
        with ThreadPoolExecutor(max_workers=min(args.backpressure_count, 32)) as executor:
            futures = [
                executor.submit(remote_add, member_a, alias_a, pressure_revision, title, op_id)
                for title, op_id in zip(pressure_titles, pressure_ops, strict=True)
            ]
            for future in futures:
                future.result()
        blocked_state = task_list(owner, owner_list_id)
        if blocked_state is None or int(blocked_state["revision"]) != pressure_revision:
            raise HarnessFailure("Ames-denied operations reached the owner before release")
        snub(owner, member_a, False, timeout)
        snubbed.discard((owner, member_a))
        pressure_receipts = [wait_receipt(owner, op_id, timeout) for op_id in pressure_ops]
        accepted = [receipt for receipt in pressure_receipts if rejection_reason(receipt) is None]
        rejected = [rejection_reason(receipt) for receipt in pressure_receipts if rejection_reason(receipt) is not None]
        if len(accepted) != 1 or any(reason != "stale-list" for reason in rejected):
            raise HarnessFailure("released pressure burst did not settle as one commit plus stale conflicts")
        authoritative = wait_converged(owner, [member_a, member_b], owner_list_id, timeout)
        pressure_count = sum(reminder_titles(authoritative).count(title) for title in pressure_titles)
        if int(authoritative["revision"]) != pressure_revision + 1 or pressure_count != 1:
            raise HarnessFailure("pressure burst violated owner sequencing")
        print(f"PASS Ames pressure/release with {args.backpressure_count} in-flight submissions", flush=True)

        print("INFO suspending the owner and waiting for both liveness deadlines", flush=True)
        suspend(owner, timeout)
        suspended.add(owner)
        wait_access_status(member_a, owner, owner_list_id, "offline", timeout)
        wait_access_status(member_b, owner, owner_list_id, "offline", timeout)
        cached_a = task_list(member_a, alias_a)
        cached_b = task_list(member_b, alias_b)
        if cached_a is None or cached_b is None:
            raise HarnessFailure("owner-offline transition removed a readable replica")
        if canonical_list(cached_a) != canonical_list(authoritative) or canonical_list(cached_b) != canonical_list(authoritative):
            raise HarnessFailure("owner-offline replicas differ from the last confirmed state")
        offline_op = operation_id("offline")
        remote_add(member_a, alias_a, int(cached_a["revision"]), "must-not-queue-offline", offline_op)
        offline_receipt = wait_receipt(member_a, offline_op, timeout)
        if rejection_reason(offline_receipt) != "host-offline":
            raise HarnessFailure("Gall did not reject a direct shared write while the owner was offline")
        if "must-not-queue-offline" in reminder_titles(task_list(member_a, alias_a) or {}):
            raise HarnessFailure("an offline shared write changed the readable replica")
        print("INFO reviving the owner and waiting for authoritative catch-up", flush=True)
        revive(owner, timeout)
        suspended.discard(owner)
        authoritative = wait_converged(owner, [member_a, member_b], owner_list_id, timeout)
        if "must-not-queue-offline" in reminder_titles(authoritative):
            raise HarnessFailure("an owner-offline write was queued and applied after recovery")
        print("PASS owner-offline readable replicas, Gall write block, and session catch-up", flush=True)

        snub(owner, member_b, True, timeout)
        snubbed.add((owner, member_b))
        outstanding_title = f"removed-in-flight-{uuid.uuid4().hex[:8]}"
        outstanding_op = operation_id("removed-in-flight")
        removal_revision = int(authoritative["revision"])
        remote_add(member_b, alias_b, removal_revision, outstanding_title, outstanding_op)
        if transport.scry_receipt(owner.config, owner.cookie, outstanding_op) is not None:
            raise HarnessFailure("the removal fixture operation was not held in flight")
        held_state = task_list(owner, owner_list_id)
        if held_state is None or outstanding_title in reminder_titles(held_state):
            raise HarnessFailure("the held operation reached canonical state before revocation")
        remove_member(owner, owner_list_id, member_b)
        wait_for(
            "owner ACL revocation",
            lambda: (
                found
                if (found := access_for_ref(owner, owner.ship, owner_list_id))
                and f"~{member_b.ship}" not in {entry.get("ship") for entry in found.get("members", []) if isinstance(entry, dict)}
                else None
            ),
            timeout,
        )
        snub(owner, member_b, False, timeout)
        snubbed.discard((owner, member_b))
        wait_for(
            "revoked participant replica cleanup",
            lambda: access_for_ref(member_b, owner.ship, owner_list_id) is None and task_list(member_b, alias_b) is None,
            timeout,
        )
        time.sleep(min(5.0, timeout / 4))
        stable = task_list(owner, owner_list_id)
        if stable is None or int(stable["revision"]) != removal_revision:
            raise HarnessFailure("an edit held across revocation changed canonical revision")
        if outstanding_title in reminder_titles(stable):
            raise HarnessFailure("a revoked participant's held edit reached canonical state")
        authoritative = stable
        print("PASS participant removal with an already-submitted edit in flight", flush=True)

        for cycle in range(args.restart_cycles):
            suspend(member_a, timeout)
            suspended.add(member_a)
            revive(member_a, timeout)
            suspended.discard(member_a)
            authoritative = wait_converged(owner, [member_a], owner_list_id, timeout)
            suspend(owner, timeout)
            suspended.add(owner)
            wait_access_status(member_a, owner, owner_list_id, "offline", timeout)
            revive(owner, timeout)
            suspended.discard(owner)
            authoritative = wait_converged(owner, [member_a], owner_list_id, timeout)
            print(f"PASS restart/resubscription soak cycle {cycle + 1}/{args.restart_cycles}", flush=True)

        poke(member_a, {"leave-shared-list": {
            "operation-id": operation_id(f"leave-{member_a.ship}"),
            "list-id": alias_a,
        }})
        wait_for(
            "leaving participant replica cleanup",
            lambda: access_for_ref(member_a, owner.ship, owner_list_id) is None and task_list(member_a, alias_a) is None,
            timeout,
        )
        wait_for(
            "owner ACL update after participant leave",
            lambda: (
                found
                if (found := access_for_ref(owner, owner.ship, owner_list_id))
                and f"~{member_a.ship}" not in {entry.get("ship") for entry in found.get("members", []) if isinstance(entry, dict)}
                else None
            ),
            timeout,
        )
        print("PASS participant leave and local replica cleanup", flush=True)

        delete_fixture(owner, fixture_title, timeout)
        owner_list_id = None
        print("PASS owner fixture cleanup", flush=True)
        print("Tend three-ship fault matrix passed", flush=True)
    finally:
        for blocker, peer in list(snubbed):
            try:
                snub(blocker, peer, False, timeout)
            except Exception as error:
                print(f"WARNING could not clear {blocker.role} Ames fault: {error}", file=sys.stderr)
        for endpoint in list(suspended):
            try:
                revive(endpoint, timeout)
            except Exception as error:
                print(f"WARNING could not revive {endpoint.role}: {error}", file=sys.stderr)
        if owner_list_id is not None:
            try:
                delete_fixture(owner, fixture_title, timeout)
            except Exception as error:
                print(f"WARNING could not remove fixture list {owner_list_id}: {error}", file=sys.stderr)


def parser() -> argparse.ArgumentParser:
    root = argparse.ArgumentParser(description=__doc__)
    for role in ("owner", "member-a", "member-b"):
        option = role.replace("-", "_")
        root.add_argument(f"--{role}-config", dest=f"{option}_config", type=Path, required=True)
        root.add_argument(f"--{role}-cookie", dest=f"{option}_cookie", type=Path, required=True)
        root.add_argument(f"--{role}-pier", dest=f"{option}_pier", type=Path, required=True)
    root.add_argument("--backpressure-count", type=int, default=12)
    root.add_argument("--restart-cycles", type=int, default=2)
    root.add_argument("--timeout", type=float, default=90)
    return root


def main(argv: list[str] | None = None) -> int:
    args = parser().parse_args(argv)
    if args.backpressure_count < 2 or args.backpressure_count > 1_000:
        parser().error("--backpressure-count must be between 2 and 1000")
    if args.restart_cycles < 0 or args.restart_cycles > 100:
        parser().error("--restart-cycles must be between 0 and 100")
    if args.timeout < 15 or args.timeout > 600:
        parser().error("--timeout must be between 15 and 600 seconds")
    try:
        run_matrix(args)
        return 0
    except (HarnessFailure, transport.TendTransportError, OSError) as error:
        print(f"FAIL {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())

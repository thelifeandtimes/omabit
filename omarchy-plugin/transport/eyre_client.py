#!/usr/bin/env python3
"""Small, dependency-free Eyre transport used by the Tend shell service."""

from __future__ import annotations

import argparse
import http.cookiejar
import ipaddress
import json
import os
from pathlib import Path
import sys
import tempfile
import urllib.error
import urllib.parse
import urllib.request
import uuid


APP = "tend"
ACTION_MARK = "tend-action-1"
MAX_JSON_BYTES = 32 * 1024 * 1024
MAX_SSE_LINE_BYTES = 32 * 1024 * 1024
MAX_SSE_EVENT_BYTES = 32 * 1024 * 1024
MAX_ACTION_BYTES = 1024 * 1024


class TendTransportError(RuntimeError):
    def __init__(self, message: str, code: str = "transport-error") -> None:
        super().__init__(message)
        self.code = code


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, request, file_pointer, code, message, headers, new_url):
        return None


def normalize_base_url(raw: str) -> str:
    value = raw.strip().rstrip("/")
    parsed = urllib.parse.urlsplit(value)
    if parsed.scheme not in {"http", "https"} or not parsed.hostname:
        raise TendTransportError("Enter a complete http:// or https:// ship URL")
    if parsed.username or parsed.password:
        raise TendTransportError("The ship URL must not contain credentials")
    if parsed.query or parsed.fragment:
        raise TendTransportError("The ship URL must not contain a query or fragment")
    if parsed.path not in {"", "/"}:
        raise TendTransportError("The ship URL must not contain a path")
    if parsed.scheme == "http" and not is_loopback_host(parsed.hostname):
        raise TendTransportError("Non-loopback ship URLs must use HTTPS")
    return value


def is_loopback_host(host: str) -> bool:
    if host.lower() == "localhost":
        return True
    try:
        return ipaddress.ip_address(host).is_loopback
    except ValueError:
        return False


def ensure_private_parent(path: Path) -> None:
    path.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
    os.chmod(path.parent, 0o700)


def refuse_symlink(path: Path) -> None:
    if path.is_symlink():
        raise TendTransportError(f"Refusing symbolic-link credential path: {path}", "unsafe-path")


def atomic_private_text(path: Path, value: str) -> None:
    ensure_private_parent(path)
    refuse_symlink(path)
    descriptor, temporary_name = tempfile.mkstemp(prefix=path.name + ".", dir=path.parent, text=True)
    temporary = Path(temporary_name)
    try:
        os.fchmod(descriptor, 0o600)
        with os.fdopen(descriptor, "w", encoding="utf-8") as stream:
            stream.write(value)
            stream.flush()
            os.fsync(stream.fileno())
        os.replace(temporary, path)
    finally:
        if temporary.exists():
            temporary.unlink()


def load_cookie_jar(path: Path) -> http.cookiejar.MozillaCookieJar:
    refuse_symlink(path)
    jar = http.cookiejar.MozillaCookieJar(str(path))
    if path.exists():
        jar.load(ignore_discard=True, ignore_expires=True)
    return jar


def opener_for(cookie_path: Path) -> tuple[urllib.request.OpenerDirector, http.cookiejar.MozillaCookieJar]:
    jar = load_cookie_jar(cookie_path)
    opener = urllib.request.build_opener(NoRedirect(), urllib.request.HTTPCookieProcessor(jar))
    opener.addheaders = [("User-Agent", "Omabit-Tend/0.1")]
    return opener, jar


def save_cookie_jar(jar: http.cookiejar.MozillaCookieJar, path: Path) -> None:
    ensure_private_parent(path)
    refuse_symlink(path)
    descriptor, temporary_name = tempfile.mkstemp(prefix=path.name + ".", dir=path.parent)
    os.close(descriptor)
    temporary = Path(temporary_name)
    try:
        jar.save(str(temporary), ignore_discard=True, ignore_expires=True)
        os.chmod(temporary, 0o600)
        with temporary.open("rb") as stream:
            os.fsync(stream.fileno())
        os.replace(temporary, path)
    finally:
        if temporary.exists():
            temporary.unlink()


def json_request(
    opener: urllib.request.OpenerDirector,
    url: str,
    *,
    method: str = "GET",
    payload: object | None = None,
    timeout: float = 20,
) -> object:
    data = None
    headers = {"Accept": "application/json"}
    if payload is not None:
        data = json.dumps(payload, separators=(",", ":")).encode("utf-8")
        headers["Content-Type"] = "application/json"
    request = urllib.request.Request(url, data=data, headers=headers, method=method)
    with opener.open(request, timeout=timeout) as response:
        content_length = response.headers.get("Content-Length")
        if content_length:
            try:
                if int(content_length) > MAX_JSON_BYTES:
                    raise TendTransportError("Eyre JSON response exceeds the 32 MiB limit", "response-too-large")
            except ValueError as error:
                raise TendTransportError("Eyre returned an invalid Content-Length header") from error
        body = response.read(MAX_JSON_BYTES + 1)
    if len(body) > MAX_JSON_BYTES:
        raise TendTransportError("Eyre JSON response exceeds the 32 MiB limit", "response-too-large")
    if not body:
        return None
    try:
        return json.loads(body.decode("utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError) as error:
        raise TendTransportError("Eyre returned invalid JSON") from error


def scry_whoami(opener: urllib.request.OpenerDirector, base_url: str) -> str:
    result = json_request(opener, base_url + "/~/scry/tend/whoami.json")
    if not isinstance(result, str) or not result.strip():
        raise TendTransportError("The authenticated ship did not return a valid identity")
    return result.strip().removeprefix("~")


def scry_state(config_path: Path, cookie_path: Path) -> object:
    connection = read_connection(config_path)
    opener, _ = opener_for(cookie_path)
    return json_request(opener, connection["baseUrl"] + "/~/scry/tend/state.json")


def write_connection(path: Path, base_url: str, ship: str) -> None:
    atomic_private_text(
        path,
        json.dumps({"baseUrl": base_url, "ship": ship}, separators=(",", ":")) + "\n",
    )


def read_connection(path: Path) -> dict[str, str]:
    refuse_symlink(path)
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as error:
        raise TendTransportError("Tend has not been connected to a ship") from error
    base_url = normalize_base_url(str(value.get("baseUrl", "")))
    ship = str(value.get("ship", "")).strip().removeprefix("~")
    if not ship:
        raise TendTransportError("The saved ship identity is invalid")
    return {"baseUrl": base_url, "ship": ship}


def login(base_url: str, cookie_path: Path, config_path: Path, code: str) -> dict[str, object]:
    base_url = normalize_base_url(base_url)
    if not code.strip():
        raise TendTransportError("Enter the current +code from your ship")
    ensure_private_parent(cookie_path)
    refuse_symlink(cookie_path)
    jar = http.cookiejar.MozillaCookieJar(str(cookie_path))
    opener = urllib.request.build_opener(NoRedirect(), urllib.request.HTTPCookieProcessor(jar))
    opener.addheaders = [("User-Agent", "Omabit-Tend/0.1")]
    body = urllib.parse.urlencode({"password": code.strip()}).encode("utf-8")
    request = urllib.request.Request(
        base_url + "/~/login",
        data=body,
        headers={"Content-Type": "application/x-www-form-urlencoded"},
        method="POST",
    )
    with opener.open(request, timeout=20) as response:
        if len(response.read(MAX_JSON_BYTES + 1)) > MAX_JSON_BYTES:
            raise TendTransportError("Eyre login response exceeds the 32 MiB limit", "response-too-large")
    if not list(jar):
        raise TendTransportError("Eyre accepted the request without issuing a session cookie")
    ship = scry_whoami(opener, base_url)
    save_cookie_jar(jar, cookie_path)
    write_connection(config_path, base_url, ship)
    return {"status": "ok", "baseUrl": base_url, "ship": ship}


def disconnect(cookie_path: Path, config_path: Path) -> dict[str, object]:
    removed: list[str] = []
    for path, label in ((cookie_path, "session"), (config_path, "connection")):
        if path.exists() or path.is_symlink():
            path.unlink()
            removed.append(label)
    return {"status": "ok", "removed": removed}


def connection_status(config_path: Path, cookie_path: Path) -> dict[str, object]:
    connection = read_connection(config_path)
    return {
        "status": "configured",
        **connection,
        "authenticated": cookie_path.exists() and cookie_path.stat().st_size > 0,
    }


def channel_url(base_url: str, channel_id: str) -> str:
    return f"{base_url}/~/channel/{channel_id}"


def send_channel_commands(
    opener: urllib.request.OpenerDirector,
    url: str,
    commands: list[dict[str, object]],
) -> None:
    json_request(opener, url, method="PUT", payload=commands)


def iter_sse(response: object):
    event_id = ""
    data_lines: list[str] = []
    event_bytes = 0
    for raw_line in response:
        if len(raw_line) > MAX_SSE_LINE_BYTES:
            raise TendTransportError("Eyre SSE line exceeds the 32 MiB limit", "response-too-large")
        event_bytes += len(raw_line)
        if event_bytes > MAX_SSE_EVENT_BYTES:
            raise TendTransportError("Eyre SSE event exceeds the 32 MiB limit", "response-too-large")
        try:
            line = raw_line.decode("utf-8").rstrip("\r\n")
        except UnicodeDecodeError as error:
            raise TendTransportError("Eyre returned invalid UTF-8 in the event stream") from error
        if line == "":
            if data_lines:
                yield event_id, "\n".join(data_lines)
            event_id = ""
            data_lines = []
            event_bytes = 0
        elif line.startswith("id:"):
            event_id = line[3:].strip()
        elif line.startswith("data:"):
            data_lines.append(line[5:].lstrip())
    if data_lines:
        yield event_id, "\n".join(data_lines)


def event_id_number(value: str) -> int:
    if not value.isdecimal() or len(value) > 20:
        raise TendTransportError("Eyre returned an invalid SSE event ID")
    return int(value)


def event_json(value: str) -> object:
    try:
        return json.loads(value)
    except json.JSONDecodeError as error:
        raise TendTransportError("Eyre returned invalid JSON in the event stream") from error


def open_event_stream(opener: urllib.request.OpenerDirector, url: str):
    request = urllib.request.Request(url, headers={"Accept": "text/event-stream"})
    return opener.open(request, timeout=300)


def stream(config_path: Path, cookie_path: Path, output) -> None:
    connection = read_connection(config_path)
    opener, _ = opener_for(cookie_path)
    channel_id = uuid.uuid4().hex
    url = channel_url(connection["baseUrl"], channel_id)
    send_channel_commands(
        opener,
        url,
        [{
            "id": 1,
            "action": "subscribe",
            "ship": connection["ship"],
            "app": APP,
            "path": "/all",
        }],
    )
    try:
        with open_event_stream(opener, url) as response:
            for event_id, raw_data in iter_sse(response):
                message = event_json(raw_data)
                output.write(json.dumps({"eventId": event_id, "message": message}) + "\n")
                output.flush()
                if event_id:
                    send_channel_commands(
                        opener,
                        url,
                        [{"action": "ack", "event-id": event_id_number(event_id)}],
                    )
    finally:
        try:
            send_channel_commands(opener, url, [{"id": 2, "action": "delete"}])
        except Exception:
            pass


def poke(config_path: Path, cookie_path: Path, action: object) -> dict[str, object]:
    if not isinstance(action, dict) or len(action) != 1:
        raise TendTransportError("A Tend action must be a one-key JSON object")
    if len(json.dumps(action, separators=(",", ":")).encode("utf-8")) > MAX_ACTION_BYTES:
        raise TendTransportError("A Tend action must not exceed 1 MiB", "request-too-large")
    connection = read_connection(config_path)
    opener, _ = opener_for(cookie_path)
    url = channel_url(connection["baseUrl"], uuid.uuid4().hex)
    send_channel_commands(
        opener,
        url,
        [{
            "id": 1,
            "action": "poke",
            "ship": connection["ship"],
            "app": APP,
            "mark": ACTION_MARK,
            "json": action,
        }],
    )
    try:
        with open_event_stream(opener, url) as response:
            for event_id, raw_data in iter_sse(response):
                message = event_json(raw_data)
                if not isinstance(message, dict):
                    raise TendTransportError("Eyre returned an invalid poke response")
                if event_id:
                    send_channel_commands(
                        opener,
                        url,
                        [{"action": "ack", "event-id": event_id_number(event_id)}],
                    )
                if message.get("id") != 1 or message.get("response") != "poke":
                    continue
                if "err" in message:
                    raise TendTransportError(str(message["err"]))
                return {"status": "ok"}
    finally:
        try:
            send_channel_commands(opener, url, [{"id": 2, "action": "delete"}])
        except Exception:
            pass
    raise TendTransportError("The poke channel closed before Eyre acknowledged the action")


def parser() -> argparse.ArgumentParser:
    root = argparse.ArgumentParser(description=__doc__)
    commands = root.add_subparsers(dest="command", required=True)

    login_parser = commands.add_parser("login")
    login_parser.add_argument("--url", required=True)
    login_parser.add_argument("--cookie", type=Path, required=True)
    login_parser.add_argument("--config", type=Path, required=True)

    status_parser = commands.add_parser("status")
    status_parser.add_argument("--cookie", type=Path, required=True)
    status_parser.add_argument("--config", type=Path, required=True)

    stream_parser = commands.add_parser("stream")
    stream_parser.add_argument("--cookie", type=Path, required=True)
    stream_parser.add_argument("--config", type=Path, required=True)

    poke_parser = commands.add_parser("poke")
    poke_parser.add_argument("--cookie", type=Path, required=True)
    poke_parser.add_argument("--config", type=Path, required=True)

    state_parser = commands.add_parser("state")
    state_parser.add_argument("--cookie", type=Path, required=True)
    state_parser.add_argument("--config", type=Path, required=True)

    disconnect_parser = commands.add_parser("disconnect")
    disconnect_parser.add_argument("--cookie", type=Path, required=True)
    disconnect_parser.add_argument("--config", type=Path, required=True)
    return root


def main(argv: list[str] | None = None) -> int:
    args = parser().parse_args(argv)
    try:
        if args.command == "login":
            result = login(args.url, args.cookie, args.config, sys.stdin.readline())
        elif args.command == "status":
            result = connection_status(args.config, args.cookie)
        elif args.command == "stream":
            stream(args.config, args.cookie, sys.stdout)
            return 0
        elif args.command == "poke":
            result = poke(args.config, args.cookie, json.load(sys.stdin))
        elif args.command == "state":
            result = scry_state(args.config, args.cookie)
        elif args.command == "disconnect":
            result = disconnect(args.cookie, args.config)
        else:
            raise AssertionError(args.command)
        print(json.dumps(result, separators=(",", ":")))
        return 0
    except (TendTransportError, urllib.error.URLError, OSError, json.JSONDecodeError) as error:
        code = error.code if isinstance(error, TendTransportError) else "transport-error"
        if isinstance(error, urllib.error.HTTPError) and error.code in {401, 403}:
            code = "authentication-required"
        print(json.dumps({"status": "error", "code": code, "message": str(error)}), file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())

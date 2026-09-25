#!/usr/bin/env python3
"""Small, dependency-free Eyre transport used by the Tend shell service."""

from __future__ import annotations

import argparse
import copy
from calendar import monthrange
from datetime import datetime, timedelta, timezone
import http.cookiejar
import ipaddress
import json
import os
from pathlib import Path
import re
import sys
import tempfile
import urllib.error
import urllib.parse
import urllib.request
import uuid
from zoneinfo import ZoneInfo, ZoneInfoNotFoundError


APP = "tend"
ACTION_MARK = "tend-action-1"
PROTOCOL_VERSION = 2
MAX_JSON_BYTES = 32 * 1024 * 1024
MAX_SSE_LINE_BYTES = 32 * 1024 * 1024
MAX_SSE_EVENT_BYTES = 32 * 1024 * 1024
MAX_ACTION_BYTES = 1024 * 1024
URBIT_DATE_PATTERN = re.compile(
    r"^~(\d+)\.(\d+)\.(\d+)\.\.(\d+)\.(\d+)\.(\d+)(?:\.\.[0-9a-fA-F]+)?$"
)
LOCAL_DATE_PATTERN = re.compile(r"^(\d{4})-(\d{2})-(\d{2})(?:T(\d{2}):(\d{2})(?::(\d{2}))?)?$")


class TendTransportError(RuntimeError):
    def __init__(self, message: str, code: str = "transport-error") -> None:
        super().__init__(message)
        self.code = code


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, request, file_pointer, code, message, headers, new_url):
        return None


def validate_snapshot_protocol(value: object) -> object:
    snapshot = value.get("snapshot") if isinstance(value, dict) else None
    version = snapshot.get("protocol-version") if isinstance(snapshot, dict) else None
    if version != PROTOCOL_VERSION:
        raise TendTransportError(
            f"Tend desk protocol is incompatible; expected version {PROTOCOL_VERSION}",
            "incompatible-protocol",
        )
    return value


def urbit_datetime(value: str) -> datetime:
    match = URBIT_DATE_PATTERN.match(str(value or ""))
    if not match:
        raise TendTransportError("Schedule contains an invalid Urbit date", "invalid-schedule-time")
    try:
        return datetime(*(int(part) for part in match.groups()), tzinfo=timezone.utc)
    except ValueError as error:
        raise TendTransportError("Schedule contains an invalid Urbit date", "invalid-schedule-time") from error


def urbit_date(value: datetime) -> str:
    instant = value.astimezone(timezone.utc).replace(microsecond=0)
    return f"~{instant.year}.{instant.month}.{instant.day}..{instant.hour:02d}.{instant.minute:02d}.{instant.second:02d}"


def zone(value: str) -> ZoneInfo:
    try:
        return ZoneInfo(value)
    except (ZoneInfoNotFoundError, ValueError) as error:
        raise TendTransportError(f"Unknown IANA time zone: {value}", "invalid-schedule-time") from error


def _schedule_field(value: dict[str, object], kebab: str, camel: str, default: object = None) -> object:
    return value[kebab] if kebab in value else value.get(camel, default)


def _valid_local_instants(wall: datetime, timezone_value: ZoneInfo) -> list[datetime]:
    candidates: list[datetime] = []
    for fold in (0, 1):
        candidate = wall.replace(tzinfo=timezone_value, fold=fold).astimezone(timezone.utc)
        round_trip = candidate.astimezone(timezone_value).replace(tzinfo=None)
        if round_trip == wall and candidate not in candidates:
            candidates.append(candidate)
    return sorted(candidates)


def _resolve_recurrence_wall_time(wall: datetime, timezone_value: ZoneInfo) -> datetime:
    """Resolve a generated local recurrence, shifting DST gaps forward."""
    candidates = _valid_local_instants(wall, timezone_value)
    if candidates:
        return candidates[0]
    shifted: list[tuple[datetime, datetime]] = []
    for fold in (0, 1):
        candidate = wall.replace(tzinfo=timezone_value, fold=fold).astimezone(timezone.utc)
        round_trip = candidate.astimezone(timezone_value).replace(tzinfo=None)
        if round_trip > wall:
            shifted.append((round_trip, candidate))
    if shifted:
        return min(shifted, key=lambda value: (value[0], value[1]))[1]
    raise TendTransportError(
        f"Could not resolve recurring wall time {wall.isoformat()} in {timezone_value.key}",
        "invalid-schedule-time",
    )


def _weekday_sunday_zero(value: datetime) -> int:
    return (value.weekday() + 1) % 7


def _shift_month(value: datetime, months: int) -> datetime:
    total = value.year * 12 + value.month - 1 + months
    year, month_index = divmod(total, 12)
    return value.replace(year=year, month=month_index + 1, day=1)


def _ordinal_month_day(year: int, month: int, index: int, weekday: int) -> int:
    maximum = monthrange(year, month)[1]
    if index == 5:
        last = datetime(year, month, maximum)
        return maximum - ((_weekday_sunday_zero(last) - weekday) % 7)
    first = datetime(year, month, 1)
    day = 1 + ((weekday - _weekday_sunday_zero(first)) % 7) + (index - 1) * 7
    if day > maximum:
        day -= 7
    return day


def _next_recurrence_wall(current: datetime, recurrence: dict[str, object]) -> datetime:
    frequency = str(recurrence.get("frequency") or "daily")
    interval = max(1, int(recurrence.get("interval") or 1))
    if frequency == "daily":
        return current + timedelta(days=interval)
    if frequency == "weekly":
        weekdays = sorted({int(day) for day in recurrence.get("weekdays") or [] if 0 <= int(day) <= 6})
        if not weekdays:
            return current + timedelta(days=interval * 7)
        current_weekday = _weekday_sunday_zero(current)
        later = [day for day in weekdays if day > current_weekday]
        days = min(later) - current_weekday if later else interval * 7 + min(weekdays) - current_weekday
        return current + timedelta(days=days)
    if frequency == "monthly":
        month_days = sorted({int(day) for day in _schedule_field(recurrence, "month-days", "monthDays", []) or [] if 1 <= int(day) <= 31})
        month_week = _schedule_field(recurrence, "month-week", "monthWeek")
        maximum = monthrange(current.year, current.month)[1]
        later = [day for day in month_days if current.day < day <= maximum]
        if later:
            return current.replace(day=min(later))
        if isinstance(month_week, dict):
            index = max(1, min(5, int(month_week.get("index") or 1)))
            weekday = max(0, min(6, int(month_week.get("weekday") or 0)))
            candidate = _ordinal_month_day(current.year, current.month, index, weekday)
            if candidate > current.day:
                return current.replace(day=candidate)
            target = _shift_month(current, interval)
            return target.replace(day=_ordinal_month_day(target.year, target.month, index, weekday))
        target = _shift_month(current, interval)
        desired = min(month_days) if month_days else current.day
        return target.replace(day=min(desired, monthrange(target.year, target.month)[1]))
    if frequency == "yearly":
        month_days = sorted({int(day) for day in _schedule_field(recurrence, "month-days", "monthDays", []) or [] if 1 <= int(day) <= 31})
        target_year = current.year + interval
        desired = min(month_days) if month_days else current.day
        return current.replace(year=target_year, day=min(desired, monthrange(target_year, current.month)[1]))
    raise TendTransportError(f"Unknown recurrence frequency: {frequency}", "invalid-schedule-time")


def recurrence_advance(schedule: object, *, now: datetime | None = None) -> dict[str, object] | None:
    """Return the first future occurrence hint for one recurring schedule."""
    if not isinstance(schedule, dict) or not isinstance(schedule.get("recurrence"), dict):
        return None
    recurrence = schedule["recurrence"]
    assert isinstance(recurrence, dict)
    raw_due = _schedule_field(schedule, "due-at", "dueAt")
    if not raw_due:
        return None
    due = urbit_datetime(str(raw_due)) if str(raw_due).startswith("~") else datetime.fromisoformat(str(raw_due).replace("Z", "+00:00")).astimezone(timezone.utc)
    timezone_value = zone(str(schedule.get("timezone") or "UTC"))
    occurrence = max(0, int(schedule.get("occurrence") or 0))
    maximum_value = _schedule_field(recurrence, "max-occurrences", "maxOccurrences")
    maximum = None if maximum_value is None else max(1, int(maximum_value))
    end_value = _schedule_field(recurrence, "end-at", "endAt")
    end_at = None if not end_value else (urbit_datetime(str(end_value)) if str(end_value).startswith("~") else normalize_wall_time(end_value, str(schedule.get("timezone") or "UTC")))
    if isinstance(end_at, str):
        end_at = urbit_datetime(end_at)
    current = due
    current_now = (now or datetime.now(timezone.utc)).astimezone(timezone.utc)
    frequency = str(recurrence.get("frequency") or "daily")
    if frequency == "hourly":
        interval = max(1, int(recurrence.get("interval") or 1))
        duration = timedelta(hours=interval)
        steps = max(1, int((current_now - current) // duration) + 1) if current <= current_now else 1
        occurrence += steps
        next_due = current + duration * steps
        if (maximum is not None and occurrence >= maximum) or (end_at is not None and next_due > end_at):
            return {"due-at": None, "occurrence": occurrence}
        return {"due-at": urbit_date(next_due), "occurrence": occurrence}
    for _ in range(100_000):
        occurrence += 1
        if maximum is not None and occurrence >= maximum:
            return {"due-at": None, "occurrence": occurrence}
        local = current.astimezone(timezone_value).replace(tzinfo=None)
        next_wall = _next_recurrence_wall(local, recurrence)
        current = _resolve_recurrence_wall_time(next_wall, timezone_value)
        if end_at is not None and current > end_at:
            return {"due-at": None, "occurrence": occurrence}
        if current > current_now:
            return {"due-at": urbit_date(current), "occurrence": occurrence}
    raise TendTransportError("Recurrence is too far overdue to advance safely", "invalid-schedule-time")


def local_timezone_name() -> str:
    candidates: list[str] = []
    if os.environ.get("TZ"):
        candidates.append(os.environ["TZ"])
    try:
        localtime = Path("/etc/localtime").resolve()
        zone_root = Path("/usr/share/zoneinfo").resolve()
        candidates.append(str(localtime.relative_to(zone_root)))
    except (OSError, ValueError):
        pass
    try:
        candidates.append(Path("/etc/timezone").read_text(encoding="utf-8").strip())
    except OSError:
        pass
    for candidate in candidates:
        try:
            ZoneInfo(candidate)
            return candidate
        except (ZoneInfoNotFoundError, ValueError):
            continue
    return "UTC"


def normalize_wall_time(value: object, timezone_name: str, *, all_day: bool = False, all_day_minute: int = 540) -> str:
    raw = str(value or "").strip()
    timezone_value = zone(timezone_name)
    if raw.startswith("~"):
        return urbit_date(urbit_datetime(raw))
    if re.search(r"(?:[zZ]|[+-]\d{2}:\d{2})$", raw):
        try:
            explicit = datetime.fromisoformat(raw.replace("Z", "+00:00").replace("z", "+00:00"))
        except ValueError as error:
            raise TendTransportError("Schedule contains an invalid ISO-8601 time", "invalid-schedule-time") from error
        if explicit.tzinfo is None:
            raise TendTransportError("Schedule time is missing an offset", "invalid-schedule-time")
        return urbit_date(explicit)

    match = LOCAL_DATE_PATTERN.match(raw)
    if not match:
        raise TendTransportError("Schedule time must use YYYY-MM-DDTHH:MM", "invalid-schedule-time")
    year, month, day, hour, minute, second = (int(part) if part is not None else None for part in match.groups())
    if all_day:
        if not 0 <= all_day_minute < 1440:
            raise TendTransportError("All-day alert minute is outside the day", "invalid-schedule-time")
        hour, minute, second = all_day_minute // 60, all_day_minute % 60, 0
    elif hour is None or minute is None:
        raise TendTransportError("Timed reminder requires an hour and minute", "invalid-schedule-time")
    else:
        second = second or 0
    try:
        wall = datetime(year, month, day, hour, minute, second)
    except ValueError as error:
        raise TendTransportError("Schedule contains an invalid calendar time", "invalid-schedule-time") from error
    candidates: list[datetime] = []
    for fold in (0, 1):
        candidate = wall.replace(tzinfo=timezone_value, fold=fold).astimezone(timezone.utc)
        round_trip = candidate.astimezone(timezone_value).replace(tzinfo=None)
        if round_trip == wall and candidate not in candidates:
            candidates.append(candidate)
    if not candidates:
        raise TendTransportError(
            f"{raw} does not exist in {timezone_name} because of a clock change",
            "invalid-schedule-time",
        )
    return urbit_date(min(candidates))


def normalize_tend_action(action: object) -> object:
    normalized = copy.deepcopy(action)
    if not isinstance(normalized, dict) or len(normalized) != 1:
        return normalized
    if "set-completed" in normalized:
        body = normalized["set-completed"]
        if isinstance(body, dict):
            schedule = body.pop("_schedule", None)
            body["advance"] = recurrence_advance(schedule) if body.get("completed") is True else None
        return normalized
    if "batch-set-completed" in normalized:
        body = normalized["batch-set-completed"]
        if isinstance(body, dict):
            schedules = body.pop("_schedules", [])
            body["advances"] = []
            if body.get("completed") is True and isinstance(schedules, list):
                for entry in schedules:
                    if not isinstance(entry, dict):
                        continue
                    advance = recurrence_advance(entry.get("schedule"))
                    if advance is not None:
                        body["advances"].append({"reminder-id": int(entry.get("reminder-id", 0)), **advance})
        return normalized
    if "set-schedule" not in normalized:
        return normalized
    body = normalized["set-schedule"]
    if not isinstance(body, dict) or body.get("schedule") is None:
        return normalized
    schedule = body["schedule"]
    if not isinstance(schedule, dict):
        raise TendTransportError("Schedule must be an object", "invalid-schedule-time")
    timezone_name = str(schedule.get("timezone") or "")
    all_day_minute = schedule.pop("_all-day-alert-minute", 540)
    if isinstance(all_day_minute, bool) or not isinstance(all_day_minute, int):
        raise TendTransportError("All-day alert minute must be an integer", "invalid-schedule-time")
    schedule["due-at"] = normalize_wall_time(
        schedule.get("due-at"),
        timezone_name,
        all_day=schedule.get("all-day") is True,
        all_day_minute=all_day_minute,
    )
    recurrence = schedule.get("recurrence")
    if isinstance(recurrence, dict) and recurrence.get("end-at") is not None:
        recurrence["end-at"] = normalize_wall_time(recurrence["end-at"], timezone_name)
    return normalized


def display_wall_time(value: object, timezone_name: str) -> str | None:
    try:
        return urbit_datetime(str(value)).astimezone(zone(timezone_name)).strftime("%Y-%m-%dT%H:%M")
    except TendTransportError:
        return None


def enrich_tend_json(value: object) -> object:
    if isinstance(value, list):
        for item in value:
            enrich_tend_json(item)
        return value
    if not isinstance(value, dict):
        return value
    if "due-at" in value and "timezone" in value:
        local_due = display_wall_time(value.get("due-at"), str(value.get("timezone") or ""))
        if local_due:
            value["local-due"] = local_due
        recurrence = value.get("recurrence")
        if isinstance(recurrence, dict) and recurrence.get("end-at") is not None:
            local_end = display_wall_time(recurrence["end-at"], str(value.get("timezone") or ""))
            if local_end:
                recurrence["local-end"] = local_end
    for child in value.values():
        enrich_tend_json(child)
    return value


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
    result = json_request(opener, connection["baseUrl"] + "/~/scry/tend/state.json")
    return validate_snapshot_protocol(result)


def scry_accesses(config_path: Path, cookie_path: Path) -> list[object]:
    connection = read_connection(config_path)
    opener, _ = opener_for(cookie_path)
    result = json_request(opener, connection["baseUrl"] + "/~/scry/tend/accesses.json")
    if not isinstance(result, dict) or not isinstance(result.get("accesses"), list):
        raise TendTransportError("The Tend agent returned invalid ownership metadata")
    return result["accesses"]


def scry_invitations(config_path: Path, cookie_path: Path) -> list[object]:
    connection = read_connection(config_path)
    opener, _ = opener_for(cookie_path)
    result = json_request(opener, connection["baseUrl"] + "/~/scry/tend/invitations.json")
    if not isinstance(result, dict) or not isinstance(result.get("invitations-updated"), list):
        raise TendTransportError("The Tend agent returned an invalid invitation inbox")
    return result["invitations-updated"]


def scry_activities(config_path: Path, cookie_path: Path, list_id: int) -> list[object]:
    if list_id < 0:
        raise TendTransportError("The Tend list ID is invalid")
    connection = read_connection(config_path)
    opener, _ = opener_for(cookie_path)
    result = json_request(opener, connection["baseUrl"] + f"/~/scry/tend/activities/{list_id}.json")
    body = result.get("activities-updated") if isinstance(result, dict) else None
    if not isinstance(body, dict) or int(body.get("list-id", -1)) != list_id or not isinstance(body.get("activities"), list):
        raise TendTransportError("The Tend agent returned an invalid activity log")
    return body["activities"]


def scry_settings(config_path: Path, cookie_path: Path) -> dict[str, object]:
    connection = read_connection(config_path)
    opener, _ = opener_for(cookie_path)
    result = json_request(opener, connection["baseUrl"] + "/~/scry/tend/settings.json")
    body = result.get("local-settings-updated") if isinstance(result, dict) else None
    if not isinstance(body, dict):
        raise TendTransportError("Tend returned an invalid local-settings response")
    return body


def scry_receipt(config_path: Path, cookie_path: Path, operation_id: str) -> dict[str, object] | None:
    if not operation_id or len(operation_id.encode("utf-8")) > 256:
        raise TendTransportError("The Tend operation ID is invalid")
    connection = read_connection(config_path)
    opener, _ = opener_for(cookie_path)
    encoded_id = urllib.parse.quote(operation_id, safe="")
    try:
        result = json_request(opener, connection["baseUrl"] + f"/~/scry/tend/receipt/{encoded_id}.json")
    except urllib.error.HTTPError as error:
        if error.code == 404:
            error.close()
            return None
        raise
    if not isinstance(result, dict) or len(result) != 1:
        raise TendTransportError("The Tend agent returned an invalid operation receipt")
    return result


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
    state = json_request(opener, base_url + "/~/scry/tend/state.json")
    validate_snapshot_protocol(state)
    save_cookie_jar(jar, cookie_path)
    write_connection(config_path, base_url, ship)
    return {
        "status": "ok",
        "baseUrl": base_url,
        "ship": ship,
        "protocolVersion": PROTOCOL_VERSION,
        "localTimezone": local_timezone_name(),
    }


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
        "localTimezone": local_timezone_name(),
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
                message = enrich_tend_json(message)
                if isinstance(message, dict) and message.get("response") == "diff":
                    body = message.get("json")
                    if isinstance(body, dict) and "snapshot" in body:
                        validate_snapshot_protocol(body)
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
    action = normalize_tend_action(action)
    action_limit = MAX_JSON_BYTES if "restore-empty" in action else MAX_ACTION_BYTES
    if len(json.dumps(action, separators=(",", ":")).encode("utf-8")) > action_limit:
        label = "32 MiB" if action_limit == MAX_JSON_BYTES else "1 MiB"
        raise TendTransportError(f"A Tend action must not exceed {label}", "request-too-large")
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

    settings_parser = commands.add_parser("settings")
    settings_parser.add_argument("--cookie", type=Path, required=True)
    settings_parser.add_argument("--config", type=Path, required=True)

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
            result = poke(args.config, args.cookie, json.loads(sys.stdin.readline()))
        elif args.command == "state":
            result = scry_state(args.config, args.cookie)
        elif args.command == "settings":
            result = scry_settings(args.config, args.cookie)
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

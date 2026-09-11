from __future__ import annotations

import importlib.util
import io
import json
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
import stat
import tempfile
import threading
import unittest
import urllib.parse
import urllib.error
from unittest import mock


MODULE_PATH = Path(__file__).parents[1] / "omarchy-plugin" / "transport" / "eyre_client.py"
SPEC = importlib.util.spec_from_file_location("eyre_client", MODULE_PATH)
assert SPEC and SPEC.loader
eyre_client = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(eyre_client)


class EyreHandler(BaseHTTPRequestHandler):
    server_version = "FakeEyre/0.1"

    def log_message(self, format, *args):
        return

    def send_bytes(self, status: int, body: bytes, content_type: str = "application/json", headers=None):
        self.send_response(status)
        self.send_header("Content-Type", content_type)
        self.send_header("Content-Length", str(len(body)))
        self.send_header("Connection", "close")
        for key, value in headers or []:
            self.send_header(key, value)
        self.end_headers()
        self.wfile.write(body)

    def do_POST(self):
        length = int(self.headers.get("Content-Length", "0"))
        body = self.rfile.read(length).decode("utf-8")
        fields = urllib.parse.parse_qs(body)
        if self.path != "/~/login" or fields.get("password") != ["lidlut-test"]:
            self.send_bytes(403, b"forbidden", "text/plain")
            return
        if self.server.redirect_login:
            self.send_bytes(307, b"", headers=[("Location", self.server.redirect_target)])
            return
        self.send_bytes(204, b"", headers=[("Set-Cookie", f"{self.server.cookie_name}=session; Path=/; HttpOnly")])

    def do_PUT(self):
        length = int(self.headers.get("Content-Length", "0"))
        commands = json.loads(self.rfile.read(length).decode("utf-8"))
        self.server.commands.setdefault(self.path, []).extend(commands)
        self.send_bytes(204, b"")

    def do_GET(self):
        if self.path == "/~/scry/tend/whoami.json":
            if f"{self.server.cookie_name}=session" not in self.headers.get("Cookie", ""):
                self.send_bytes(403, b"forbidden", "text/plain")
                return
            self.send_bytes(200, json.dumps(self.server.ship).encode("utf-8"))
            return

        if self.path == "/~/scry/tend/state.json":
            if f"{self.server.cookie_name}=session" not in self.headers.get("Cookie", ""):
                self.send_bytes(403, b"forbidden", "text/plain")
                return
            body = self.server.state_body or json.dumps({"snapshot": {"lists": []}}).encode("utf-8")
            self.send_bytes(200, body)
            return

        if self.path == "/~/scry/tend/accesses.json":
            if f"{self.server.cookie_name}=session" not in self.headers.get("Cookie", ""):
                self.send_bytes(403, b"forbidden", "text/plain")
                return
            body = self.server.accesses_body or json.dumps({"accesses": []}).encode("utf-8")
            self.send_bytes(200, body)
            return

        if self.path == "/~/scry/tend/invitations.json":
            if f"{self.server.cookie_name}=session" not in self.headers.get("Cookie", ""):
                self.send_bytes(403, b"forbidden", "text/plain")
                return
            body = self.server.invitations_body or json.dumps({"invitations-updated": []}).encode("utf-8")
            self.send_bytes(200, body)
            return

        if self.path.startswith("/~/scry/tend/activities/"):
            if f"{self.server.cookie_name}=session" not in self.headers.get("Cookie", ""):
                self.send_bytes(403, b"forbidden", "text/plain")
                return
            body = self.server.activities_body or json.dumps({"activities-updated": {"list-id": 1, "activities": []}}).encode("utf-8")
            self.send_bytes(200, body)
            return

        if self.path == "/~/scry/tend/settings.json":
            if f"{self.server.cookie_name}=session" not in self.headers.get("Cookie", ""):
                self.send_bytes(403, b"forbidden", "text/plain")
                return
            body = self.server.settings_body or json.dumps({"local-settings-updated": {"list-order": [], "presentations": [], "collaboration-policies": []}}).encode("utf-8")
            self.send_bytes(200, body)
            return

        if self.path.startswith("/~/scry/tend/receipt/"):
            if f"{self.server.cookie_name}=session" not in self.headers.get("Cookie", ""):
                self.send_bytes(403, b"forbidden", "text/plain")
                return
            if not self.server.receipt_body:
                self.send_bytes(404, b"missing", "text/plain")
                return
            self.send_bytes(200, self.server.receipt_body)
            return

        commands = self.server.commands.get(self.path, [])
        first = next((command for command in commands if command.get("action") in {"subscribe", "poke"}), None)
        if not first:
            self.send_bytes(404, b"missing", "text/plain")
            return
        if first["action"] == "subscribe":
            events = [
                (1, {"id": 1, "response": "subscribe", "ok": None}),
                (2, {"id": 1, "response": "diff", "json": {"snapshot": {"lists": []}}}),
            ]
        else:
            events = [(1, {"id": 1, "response": "poke", "ok": None})]
        body = "".join(
            f"id: {event_id}\ndata: {json.dumps(payload, separators=(',', ':'))}\n\n"
            for event_id, payload in events
        ).encode("utf-8")
        self.send_bytes(200, body, "text/event-stream")


class FakeEyre:
    def __init__(self, cookie_name="urbauth-test", ship="zod"):
        self.cookie_name = cookie_name
        self.ship = ship

    def __enter__(self):
        self.server = ThreadingHTTPServer(("127.0.0.1", 0), EyreHandler)
        self.server.commands = {}
        self.server.cookie_name = self.cookie_name
        self.server.ship = self.ship
        self.server.redirect_login = False
        self.server.redirect_target = ""
        self.server.state_body = b""
        self.server.accesses_body = b""
        self.server.invitations_body = b""
        self.server.activities_body = b""
        self.server.settings_body = b""
        self.server.receipt_body = b""
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.thread.start()
        host, port = self.server.server_address
        self.base_url = f"http://{host}:{port}"
        return self

    def __exit__(self, exc_type, exc, traceback):
        self.server.shutdown()
        self.server.server_close()
        self.thread.join(timeout=2)

    @property
    def all_commands(self):
        return [command for commands in self.server.commands.values() for command in commands]


class UrlValidationTests(unittest.TestCase):
    def test_accepts_https_and_loopback_http(self):
        self.assertEqual(
            eyre_client.normalize_base_url("https://sampel-palnet.arvo.network/"),
            "https://sampel-palnet.arvo.network",
        )
        self.assertEqual(eyre_client.normalize_base_url("http://127.0.0.1:8080"), "http://127.0.0.1:8080")

    def test_rejects_insecure_remote_and_embedded_credentials(self):
        with self.assertRaisesRegex(eyre_client.TendTransportError, "HTTPS"):
            eyre_client.normalize_base_url("http://example.com")
        with self.assertRaisesRegex(eyre_client.TendTransportError, "credentials"):
            eyre_client.normalize_base_url("https://user:secret@example.com")
        with self.assertRaisesRegex(eyre_client.TendTransportError, "path"):
            eyre_client.normalize_base_url("https://example.com/urbit")


class TimeZoneConversionTests(unittest.TestCase):
    def test_normalizes_timed_all_day_and_explicit_offset_values(self):
        self.assertEqual(
            eyre_client.normalize_wall_time("2026-09-10T17:30", "America/Los_Angeles"),
            "~2026.9.11..00.30.00",
        )
        self.assertEqual(
            eyre_client.normalize_wall_time(
                "2026-09-10",
                "America/Los_Angeles",
                all_day=True,
                all_day_minute=540,
            ),
            "~2026.9.10..16.00.00",
        )
        self.assertEqual(
            eyre_client.normalize_wall_time("2026-09-10T17:30:00-07:00", "UTC"),
            "~2026.9.11..00.30.00",
        )
        self.assertEqual(
            eyre_client.normalize_wall_time("~2026.9.11..00.30.00..abcd", "UTC"),
            "~2026.9.11..00.30.00",
        )

    def test_rejects_dst_gap_and_chooses_earlier_fold(self):
        with self.assertRaisesRegex(eyre_client.TendTransportError, "does not exist") as raised:
            eyre_client.normalize_wall_time("2026-03-08T02:30", "America/Los_Angeles")
        self.assertEqual(raised.exception.code, "invalid-schedule-time")
        self.assertEqual(
            eyre_client.normalize_wall_time("2026-11-01T01:30", "America/Los_Angeles"),
            "~2026.11.1..08.30.00",
        )
        with self.assertRaisesRegex(eyre_client.TendTransportError, "Unknown IANA"):
            eyre_client.normalize_wall_time("2026-09-10T12:00", "Not/A_Zone")
        with self.assertRaisesRegex(eyre_client.TendTransportError, "Unknown IANA"):
            eyre_client.normalize_wall_time("~2026.9.10..12.00.00", "Not/A_Zone")
        with self.assertRaisesRegex(eyre_client.TendTransportError, "invalid Urbit date"):
            eyre_client.normalize_wall_time("~2026.9.10..12.00.00junk", "UTC")

    def test_normalizes_schedule_action_without_mutating_input(self):
        action = {
            "set-schedule": {
                "operation-id": "zone-op",
                "list-id": 1,
                "reminder-id": 2,
                "base-revision": 3,
                "schedule": {
                    "due-at": "2026-09-10T17:30",
                    "all-day": False,
                    "timezone": "America/Los_Angeles",
                    "_all-day-alert-minute": 540,
                    "early-seconds": [],
                    "recurrence": {
                        "frequency": "weekly",
                        "interval": 1,
                        "weekdays": [4],
                        "month-days": [],
                        "month-week": None,
                        "end-at": "2026-10-01T17:30",
                        "max-occurrences": None,
                    },
                },
            }
        }
        normalized = eyre_client.normalize_tend_action(action)
        schedule = normalized["set-schedule"]["schedule"]
        self.assertEqual(schedule["due-at"], "~2026.9.11..00.30.00")
        self.assertEqual(schedule["recurrence"]["end-at"], "~2026.10.2..00.30.00")
        self.assertNotIn("_all-day-alert-minute", schedule)
        self.assertEqual(action["set-schedule"]["schedule"]["due-at"], "2026-09-10T17:30")

    def test_enriches_canonical_schedule_for_wall_clock_display(self):
        update = {
            "snapshot": {
                "lists": [{
                    "reminders": [{
                        "schedule": {
                            "due-at": "~2026.9.11..00.30.00",
                            "timezone": "America/Los_Angeles",
                            "recurrence": {"end-at": "~2026.10.2..00.30.00"},
                        }
                    }]
                }]
            }
        }
        eyre_client.enrich_tend_json(update)
        schedule = update["snapshot"]["lists"][0]["reminders"][0]["schedule"]
        self.assertEqual(schedule["local-due"], "2026-09-10T17:30")
        self.assertEqual(schedule["recurrence"]["local-end"], "2026-10-01T17:30")


class EyreFlowTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.root = Path(self.temporary.name)
        self.cookie = self.root / "runtime" / "cookies.txt"
        self.config = self.root / "config" / "connection.json"

    def tearDown(self):
        self.temporary.cleanup()

    def login(self, fake: FakeEyre):
        result = eyre_client.login(fake.base_url, self.cookie, self.config, "lidlut-test")
        self.assertEqual(result["ship"], fake.ship)
        self.assertEqual(result["baseUrl"], fake.base_url)

    def test_login_saves_only_cookie_and_connection_metadata(self):
        with FakeEyre() as fake:
            self.login(fake)
            self.assertTrue(self.cookie.exists())
            self.assertEqual(stat.S_IMODE(self.cookie.stat().st_mode), 0o600)
            self.assertNotIn("lidlut-test", self.cookie.read_text(encoding="utf-8"))
            self.assertEqual(
                json.loads(self.config.read_text(encoding="utf-8")),
                {"baseUrl": fake.base_url, "ship": "zod"},
            )
            self.assertEqual(stat.S_IMODE(self.config.stat().st_mode), 0o600)

    def test_reconnecting_replaces_prior_ship_cookie_jar(self):
        with FakeEyre("urbauth-first") as first:
            self.login(first)
        self.assertIn("urbauth-first", self.cookie.read_text(encoding="utf-8"))
        with FakeEyre("urbauth-second") as second:
            self.login(second)
        contents = self.cookie.read_text(encoding="utf-8")
        self.assertIn("urbauth-second", contents)
        self.assertNotIn("urbauth-first", contents)

    def test_login_accepts_planet_moon_and_comet_identities(self):
        ships = (
            "sampel-palnet",
            "doznec-dozzod-dozzod",
            "doznec--dozzod-dozzod-dozzod-dozzod",
        )
        for index, ship in enumerate(ships):
            with self.subTest(ship=ship), FakeEyre(f"urbauth-rank-{index}", ship) as fake:
                self.login(fake)
                self.assertEqual(eyre_client.read_connection(self.config)["ship"], ship)

    def test_login_refuses_redirects_without_saving_credentials(self):
        with FakeEyre() as fake:
            fake.server.redirect_login = True
            fake.server.redirect_target = fake.base_url + "/capture"
            with self.assertRaises(urllib.error.HTTPError) as raised:
                eyre_client.login(fake.base_url, self.cookie, self.config, "lidlut-test")
        self.assertEqual(raised.exception.code, 307)
        raised.exception.close()
        self.assertFalse(self.cookie.exists())
        self.assertFalse(self.config.exists())

    def test_refuses_symbolic_link_credential_paths(self):
        target = self.root / "target"
        target.write_text("keep", encoding="utf-8")
        self.cookie.parent.mkdir(parents=True)
        self.cookie.symlink_to(target)
        with FakeEyre() as fake:
            with self.assertRaisesRegex(eyre_client.TendTransportError, "symbolic-link"):
                eyre_client.login(fake.base_url, self.cookie, self.config, "lidlut-test")
        self.assertEqual(target.read_text(encoding="utf-8"), "keep")

    def test_stream_subscribes_emits_events_and_acknowledges(self):
        with FakeEyre() as fake:
            self.login(fake)
            output = io.StringIO()
            eyre_client.stream(self.config, self.cookie, output)
            envelopes = [json.loads(line) for line in output.getvalue().splitlines()]
            self.assertEqual(envelopes[0]["message"]["response"], "subscribe")
            self.assertEqual(envelopes[1]["message"]["json"], {"snapshot": {"lists": []}})
            self.assertEqual(
                [command["event-id"] for command in fake.all_commands if command.get("action") == "ack"],
                [1, 2],
            )

    def test_poke_uses_tend_agent_mark_and_authenticated_ship(self):
        with FakeEyre() as fake:
            self.login(fake)
            action = {"create-list": {"operation-id": "test-op", "title": "Inbox"}}
            self.assertEqual(eyre_client.poke(self.config, self.cookie, action), {"status": "ok"})
            poke = next(command for command in fake.all_commands if command.get("action") == "poke")
            self.assertEqual(poke["ship"], "zod")
            self.assertEqual(poke["app"], "tend")
            self.assertEqual(poke["mark"], "tend-action-1")
            self.assertEqual(poke["json"], action)

    def test_poke_normalizes_wall_clock_schedule_before_eyre(self):
        with FakeEyre() as fake:
            self.login(fake)
            action = {
                "set-schedule": {
                    "operation-id": "zone-poke",
                    "list-id": 1,
                    "reminder-id": 2,
                    "base-revision": 3,
                    "schedule": {
                        "due-at": "2026-09-10T17:30",
                        "all-day": False,
                        "timezone": "America/Los_Angeles",
                        "_all-day-alert-minute": 540,
                        "early-seconds": [],
                        "recurrence": None,
                    },
                }
            }
            self.assertEqual(eyre_client.poke(self.config, self.cookie, action), {"status": "ok"})
            poke = next(command for command in fake.all_commands if command.get("action") == "poke")
            schedule = poke["json"]["set-schedule"]["schedule"]
            self.assertEqual(schedule["due-at"], "~2026.9.11..00.30.00")
            self.assertNotIn("_all-day-alert-minute", schedule)

    def test_state_scry_and_disconnect(self):
        with FakeEyre() as fake:
            self.login(fake)
            self.assertEqual(
                eyre_client.scry_state(self.config, self.cookie),
                {"snapshot": {"lists": []}},
            )
            fake.server.accesses_body = json.dumps({"accesses": [{"alias": 1, "owner": True}]}).encode("utf-8")
            self.assertEqual(
                eyre_client.scry_accesses(self.config, self.cookie),
                [{"alias": 1, "owner": True}],
            )
            fake.server.invitations_body = json.dumps({"invitations-updated": [{"token": "invite-1"}]}).encode("utf-8")
            self.assertEqual(
                eyre_client.scry_invitations(self.config, self.cookie),
                [{"token": "invite-1"}],
            )
            fake.server.activities_body = json.dumps({"activities-updated": {"list-id": 1, "activities": [{"id": "event-1"}]}}).encode("utf-8")
            self.assertEqual(
                eyre_client.scry_activities(self.config, self.cookie, 1),
                [{"id": "event-1"}],
            )
            settings = {"list-order": [2, 1], "presentations": [], "collaboration-policies": []}
            fake.server.settings_body = json.dumps({"local-settings-updated": settings}).encode("utf-8")
            self.assertEqual(eyre_client.scry_settings(self.config, self.cookie), settings)
            self.assertIsNone(eyre_client.scry_receipt(self.config, self.cookie, "missing"))
            fake.server.receipt_body = json.dumps({"snapshot": {"lists": []}}).encode("utf-8")
            self.assertEqual(
                eyre_client.scry_receipt(self.config, self.cookie, "operation-1"),
                {"snapshot": {"lists": []}},
            )
            result = eyre_client.disconnect(self.cookie, self.config)
            self.assertEqual(result, {"status": "ok", "removed": ["session", "connection"]})
            self.assertFalse(self.cookie.exists())
            self.assertFalse(self.config.exists())

    def test_state_scry_rejects_oversized_json(self):
        with FakeEyre() as fake:
            self.login(fake)
            fake.server.state_body = b'{"padding":"' + b"x" * 100 + b'"}'
            with mock.patch.object(eyre_client, "MAX_JSON_BYTES", 32):
                with self.assertRaisesRegex(eyre_client.TendTransportError, "exceeds") as raised:
                    eyre_client.scry_state(self.config, self.cookie)
        self.assertEqual(raised.exception.code, "response-too-large")


class EventValidationTests(unittest.TestCase):
    def test_event_parser_rejects_oversized_and_invalid_utf8_input(self):
        with mock.patch.object(eyre_client, "MAX_SSE_LINE_BYTES", 8):
            with self.assertRaisesRegex(eyre_client.TendTransportError, "line exceeds"):
                list(eyre_client.iter_sse([b"data: 1234\n"]))
        with self.assertRaisesRegex(eyre_client.TendTransportError, "UTF-8"):
            list(eyre_client.iter_sse([b"data: \xff\n"]))

    def test_event_ids_and_json_are_validated(self):
        self.assertEqual(eyre_client.event_id_number("42"), 42)
        for value in ("-1", "one", "1" * 21):
            with self.assertRaisesRegex(eyre_client.TendTransportError, "event ID"):
                eyre_client.event_id_number(value)
        with self.assertRaisesRegex(eyre_client.TendTransportError, "invalid JSON"):
            eyre_client.event_json("{")

    def test_oversized_action_is_rejected_before_network_access(self):
        with mock.patch.object(eyre_client, "MAX_ACTION_BYTES", 32):
            with self.assertRaisesRegex(eyre_client.TendTransportError, "1 MiB") as raised:
                eyre_client.poke(Path("missing-config"), Path("missing-cookie"), {"create-list": {"title": "x" * 100}})
        self.assertEqual(raised.exception.code, "request-too-large")


if __name__ == "__main__":
    unittest.main()

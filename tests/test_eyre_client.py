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
        self.send_bytes(204, b"", headers=[("Set-Cookie", "urbauth-test=session; Path=/; HttpOnly")])

    def do_PUT(self):
        length = int(self.headers.get("Content-Length", "0"))
        commands = json.loads(self.rfile.read(length).decode("utf-8"))
        self.server.commands.setdefault(self.path, []).extend(commands)
        self.send_bytes(204, b"")

    def do_GET(self):
        if self.path == "/~/scry/tend/whoami.json":
            if "urbauth-test=session" not in self.headers.get("Cookie", ""):
                self.send_bytes(403, b"forbidden", "text/plain")
                return
            self.send_bytes(200, json.dumps("zod").encode("utf-8"))
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
    def __enter__(self):
        self.server = ThreadingHTTPServer(("127.0.0.1", 0), EyreHandler)
        self.server.commands = {}
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
        self.assertEqual(result["ship"], "zod")
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


if __name__ == "__main__":
    unittest.main()


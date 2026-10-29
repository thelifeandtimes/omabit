"""Direct HTTP client used only after private guest publication admission."""
from dataclasses import dataclass
import http.client
import re
import socket
import time

from docker_gateway_peers import GatewayShip, private_read
from docker_peer import Transport
from docker_runtime import require_guest
from docker_publication import PublishedShip
from state_case import parse_sse, matching_state


@dataclass(frozen=True)
class PublishedGatewayShip(GatewayShip, PublishedShip):
    """Retained generation identity with the same published-ship engine contract."""


def cookie_header(raw, ship):
    """Read a single fixture curl cookie; reject extra identity/domain/path data."""
    if ship not in ('zod', 'nec') or not isinstance(raw, bytes) or not 0 < len(raw) <= 8192:
        raise ValueError('bounded private session input required')
    found = []
    for line in raw.decode('ascii').splitlines():
        if line.startswith('#HttpOnly_'):
            line = line[len('#HttpOnly_'):]
        elif not line or line.startswith('#'):
            continue
        fields = line.split('\t')
        if len(fields) != 7 or fields[:4] != ['127.0.0.1', 'FALSE', '/', 'FALSE']:
            raise ValueError('private session origin differs')
        expiry, name, value = fields[4:]
        if not re.fullmatch('[0-9]{1,12}', expiry) or name != 'urbauth-~' + ship or not re.fullmatch('[A-Za-z0-9._~+/=-]{1,4096}', value):
            raise ValueError('private session identity differs')
        found.append(name + '=' + value)
    if len(found) != 1:
        raise ValueError('one retained private session required')
    return found[0]


class PublishedTransport(Transport):
    def __init__(self, engine, specs, publication, root):
        super().__init__(engine, specs)
        self.publication = publication
        self.root = root
        self.requests = []

    def request(self, ship, path, data=None, content=None, accept=None, method=None,
                auth=None, timeout=90, cursor=None, stream=False):
        require_guest()
        if ship not in ('zod', 'nec') or auth not in (None, 'zod', 'nec'):
            raise ValueError('bound private ship required')
        if (not isinstance(path, str) or not path.startswith('/') or path.startswith('//')
            or len(path) > 512 or any(ord(c) < 32 or ord(c) == 127 for c in path)):
            raise ValueError('bound direct HTTP path required')
        if method not in (None, 'PUT', 'DELETE') or type(timeout) not in (int, float) or not 0 < timeout <= 90:
            raise ValueError('bounded direct HTTP method required')
        self.engine.check_anchor(self.specs[ship])
        headers = {'Cookie': cookie_header(private_read(self.root / ((auth or ship) + '.cookie'), 8192), auth or ship)}
        for key, value in [('Content-Type', content), ('Accept', accept)]:
            if value is not None:
                if not isinstance(value, str) or len(value) > 128 or '\r' in value or '\n' in value:
                    raise ValueError('bounded direct HTTP header required')
                headers[key] = value
        if cursor is not None:
            if type(cursor) is not int or cursor < 0:
                raise ValueError('nonnegative retained cursor required')
            headers['Last-Event-ID'] = str(cursor)
        if data is not None and (not isinstance(data, bytes) or len(data) > 65536):
            raise ValueError('bounded direct HTTP body required')
        verb = method or ('POST' if data is not None else 'GET')
        port = self.publication.ports[18081 + self.specs[ship].number]
        connection = http.client.HTTPConnection('127.0.0.1', port, timeout=timeout)
        try:
            # A failed mutation is uncertain and is never automatically resent.
            connection.request(verb, path, data, headers)
            response = connection.getresponse()
            if stream:
                chunks = [];size = 0;deadline = time.monotonic() + timeout
                while time.monotonic() < deadline:
                    try:
                        chunk = response.read1(min(4096, 65537 - size))
                    except (TimeoutError, socket.timeout):
                        break
                    if not chunk:
                        break
                    chunks.append(chunk);size += len(chunk)
                    if size > 65536:
                        raise RuntimeError('direct stream exceeds bound')
                body = b''.join(chunks)
            else:
                body = response.read(65537)
                if len(body) > 65536:
                    raise RuntimeError('direct response exceeds bound')
            self.requests.append({'ship': ship, 'port': port, 'method': verb, 'path': path,
                                  'status': response.status, 'response_bytes': len(body),
                                  'stream': stream, 'crossed_session': auth is not None and auth != ship})
            if len(self.requests) > 200:
                raise RuntimeError('direct request receipt bound')
            return response.status, body
        except (OSError, http.client.HTTPException):
            raise RuntimeError('direct private HTTP failed; request not replayed') from None
        finally:
            connection.close()

    def call(self, ship, path, data=None, content=None, accept=None, admin=False,
             method=None, auth=None, login=False, timeout=90):
        require_guest()
        if login:
            raise RuntimeError('retained publication must not log in again')
        if admin:
            return super().call(ship, path, data, content, accept, admin=True,
                                method=method, auth=auth, timeout=timeout)
        return self.request(ship, path, data, content, accept, method, auth, timeout)

    def events(self, channel, cursor, expected, initial):
        require_guest()
        if not re.fullmatch('/~/channel/omabit-state-[0-9a-f]{32}', channel):
            raise ValueError('bound retained subscription required')
        rows = [];ack = not initial;fact = False;deadline = time.monotonic() + 60
        while time.monotonic() < deadline:
            status, body = self.request('nec', channel, accept='text/event-stream', timeout=3,
                                        cursor=cursor, stream=True)
            if status != 200:
                raise RuntimeError('direct subscription reconnect refused')
            for event_id, event in parse_sse(body.splitlines(keepends=True)):
                rows.append((event_id, event));cursor = event_id
                fact = matching_state(event, expected) or fact
                ack = ack or (event.get('response') == 'subscribe' and event.get('id') == 1 and event.get('ok') == 'ok')
            if len(rows) > 100:
                raise RuntimeError('direct SSE event bound')
            if fact and ack:
                return rows
        raise RuntimeError('direct SSE observation deadline')

    def check_unpublished(self):
        require_guest()
        self.engine.check_anchor(self.specs['zod'])
        attempts = [('127.0.0.1', 25432, 'rpc')]
        attempts += [('127.0.0.1', self.admin[s], 'admin-' + s) for s in ('zod', 'nec')]
        attempts += [('::1', self.publication.ports[18081 + self.specs[s].number], 'ipv6-' + s) for s in ('zod', 'nec')]
        rows = []
        for host, port, kind in attempts:
            try:
                connection = socket.create_connection((host, port), timeout=2)
            except OSError:
                rows.append({'kind': kind, 'host': host, 'port': port, 'refused': True})
            else:
                connection.close()
                raise RuntimeError('unpublished private endpoint accepted a connection')
        return rows

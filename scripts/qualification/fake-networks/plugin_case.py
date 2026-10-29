"""Actual Theme/DMs transports against fresh peers; desktop and keyring stay inert."""
import contextlib
import copy
import hashlib
import json
from pathlib import Path
import re
import sys
import threading
import time
import uuid


def matched_message(events, event, peer, text):
    return [row for row in events if row.get('event') == event and row.get('ship') == '~' + peer
            and isinstance(row.get('message'), dict) and row['message'].get('text') == text]


def assert_history(page, expected):
    messages = page['messages']
    for text, author in expected:
        matches = [m for m in messages if m.get('text') == text and m.get('author') == '~' + author]
        if len(matches) != 1:
            raise RuntimeError('missing or duplicate DM history write')
    return [{'text': text, 'author': '~' + author,
             'id': next(m['id'] for m in messages if m.get('text') == text and m.get('author') == '~' + author)}
            for text, author in expected]


class MemoryStore:
    """Deliberate test adapter; the real Secret Service has its own runtime gate."""
    def __init__(self): self.session = None
    def account(self):
        return {k: self.session[k] for k in ('url', 'ship')} if self.session else None
    def lookup(self, url, ship, check_expiry=True):
        if not self.session or (url, ship) != (self.session['url'], self.session['ship']):
            raise ValueError('fixture account mismatch')
        return self.session
    def save(self, session): self.session = copy.deepcopy(session)
    def forget(self): self.session = None


class StreamTap:
    """Observe bounded frames while the plugin's real parser consumes unchanged bytes."""
    def __init__(self, response, callback):
        self.response, self.callback = response, callback
        self.buffer = b''
        self.data = []
        self.identity = None
    def __getattr__(self, name): return getattr(self.response, name)
    def read1(self, size):
        chunk = self.response.read1(size)
        self.buffer += chunk
        if len(self.buffer) + sum(map(len, self.data)) > 262144:
            raise RuntimeError('fixture SSE observation bound')
        while b'\n' in self.buffer:
            line, self.buffer = self.buffer.split(b'\n', 1)
            line = line.rstrip(b'\r')
            if line.startswith(b'id:'): self.identity = line[3:].strip().decode()
            elif line.startswith(b'data:'): self.data.append(line[5:].lstrip())
            elif not line:
                if self.data: self.callback(self.identity, json.loads(b'\n'.join(self.data)))
                self.identity, self.data = None, []
        return chunk


class PluginRecovery:
    def __init__(self, http, ships, save, emit):
        if not Path('/run/omabit-thread-probe.json').is_file() or Path('/home').exists():
            raise RuntimeError('private native lab required')
        sys.path[:0] = ['/artifacts/plugins/theme', '/artifacts/plugins/dms']
        import bridge
        import client.eyre as eyre
        import client.sync as sync
        from client.instance import INSTANCE as theme_instance
        from instance import INSTANCE as dms_instance
        if (theme_instance.profile, dms_instance.profile) != ('auth-lab', 'auth-lab'):
            raise RuntimeError('experimental plugin identities required')
        self.bridge_module, self.sync = bridge, sync
        self.http, self.ships, self.save, self.emit = http, ships, save, emit
        self.lock = threading.RLock()
        self.report = {'ok': False, 'desktop_adapter': 'disabled', 'credential_store': 'in-memory fixture',
                       'plugin_ids': [theme_instance.plugin_id, dms_instance.plugin_id],
                       'theme': {'actions': [], 'frames': [], 'writes': []},
                       'dms': {}, 'history': [], 'cleanup': []}
        self.clients, self.expected = {}, []
        self.watch = None
        self.original_eyre = sync.Eyre
        outer = self
        class ObservedEyre(eyre.Eyre):
            @contextlib.contextmanager
            def request(self, method, path, body=None, content_type='application/json', stream=False):
                with super().request(method, path, body, content_type, stream) as response:
                    if stream:
                        yield StreamTap(response, lambda identity, event: outer.record_theme_frame(path, identity, event))
                    else: yield response
            def put(self, path, messages):
                result = super().put(path, messages)
                with outer.lock:
                    for message in messages:
                        outer.bounded(outer.report['theme']['actions'], {'channel': path,
                            **{k: message[k] for k in ('id', 'action', 'app', 'path') if k in message}})
                return result
        self.eyre_class = ObservedEyre
        sync.Eyre = ObservedEyre

    @staticmethod
    def bounded(rows, value):
        if len(rows) >= 1024: raise RuntimeError('plugin observation bound')
        rows.append(value)

    def record_theme_frame(self, path, identity, event):
        with self.lock:
            self.bounded(self.report['theme']['frames'], {'channel': path, 'event_id': identity, 'event': event})

    def wait(self, predicate, label, timeout=65):
        deadline = time.monotonic() + timeout
        while time.monotonic() < deadline:
            with self.lock:
                if predicate(): self.save(); return
            time.sleep(.1)
        self.save()
        raise RuntimeError('plugin timeout: ' + label)

    def code(self, ship):
        status, body = self.http(ship, '/', json.dumps({'source': {'dojo': '+code'}, 'sink': {'stdout': None}}).encode(), content='application/json', admin=True)
        code = json.loads(body).strip().removeprefix('~') if status == 200 else ''
        if not re.fullmatch('[a-z]{6}(?:-[a-z]{6}){3}', code): raise RuntimeError('fresh fake code unavailable')
        return code

    def dm_client(self, ship):
        client = self.bridge_module.Bridge(session_store=MemoryStore())
        row = {'actions': [], 'frames': [], 'events': [], 'pokes': []}
        self.report['dms'][ship] = row
        action, fact, request = client.channel_action, client.on_fact, client.request
        def observed_request(path, data=None, method=None):
            if method == 'PUT' and path.startswith('/~/channel/') and data:
                for body in json.loads(data):
                    if body.get('action') == 'poke':
                        with self.lock:
                            self.bounded(row['pokes'], {'channel': path.removeprefix('/~/channel/'),
                                'id': body['id'], 'mark': body['mark'], 'payload': body['json']})
            return request(path, data, method)
        def observed_action(fields, *, binding=None):
            # The reentrant lock also pins the observation to the actual target.
            with client.channel_lock:
                channel = client.channel
                ident = action(fields, binding=binding)
            with self.lock:
                self.bounded(row['actions'], {'channel': channel, 'id': ident,
                    **{k: fields[k] for k in ('action', 'app', 'path') if k in fields}})
            return ident
        def observed_fact(outer, event_id):
            channel = client.channel
            fact(outer, event_id)
            with self.lock:
                self.bounded(row['frames'], {'channel': channel, 'event_id': event_id,
                    'id': outer.get('id'), 'response': outer.get('response'), 'ok': outer.get('ok'),
                    'error': outer.get('err') is not None})
        def emitted(event, **fields):
            if event not in ('message', 'sent', 'accepted', 'error'): return
            with self.lock: self.bounded(row['events'], {'event': event, **fields})
        client.channel_action, client.on_fact, client.emit, client.request = observed_action, observed_fact, emitted, observed_request
        self.clients[ship] = client
        if client.login('http://127.0.0.1:' + str(self.ships[ship]['port']), self.code(ship)) != '~' + ship:
            raise RuntimeError('plugin identity mismatch')
        return client

    def dm_ready(self, ship, exclude=None):
        client, row = self.clients[ship], self.report['dms'][ship]
        channel = client.channel
        if channel == exclude: return False
        subscriptions = [a['id'] for a in row['actions'] if a['channel'] == channel
                         and a['action'] == 'subscribe' and a.get('app') == 'chat' and a.get('path') == '/v4']
        return any(f['channel'] == channel and f['id'] in subscriptions and f['response'] == 'subscribe'
                   and f['ok'] == 'ok' and not f['error'] for f in row['frames'])

    def theme_ready(self, exclude=None):
        if not self.watch.ready.is_set(): return False
        actions, frames = self.report['theme']['actions'], self.report['theme']['frames']
        channels = {a['channel'] for a in actions if a['action'] == 'subscribe' and a.get('app') == 'settings'}
        return any(f['channel'] in channels and f['channel'] != exclude
                   and f['event'].get('response') == 'subscribe' and f['event'].get('ok') == 'ok'
                   for f in frames)

    def theme_write(self, phase):
        profile = {'schemaVersion': 1, 'updateId': uuid.uuid4().hex, 'deviceId': 'a' * 32,
                   'theme': 'omabit-transport-' + phase, 'font': 'Fixture Mono',
                   'colors': {'mode': 'dark', 'background': '#112233', 'foreground': '#EEEEEE', 'blue': '#123ABC'},
                   'shell': {}, 'windows': {}, 'animations': {'curves': [], 'rules': []}, 'files': {}}
        with self.lock: start = len(self.report['theme']['frames'])
        row = {'phase': phase, 'profile': profile, 'state': 'pending'}
        self.report['theme']['writes'].append(row); self.save()
        self.remote.write(profile)  # One mutation only; ACK timeout is never replayed.
        row['state'] = 'acknowledged'; self.save()
        self.wait(lambda: any(f['event'].get('response') == 'diff' for f in self.report['theme']['frames'][start:]), 'Theme live invalidation')
        if self.remote.read() != profile: raise RuntimeError('Theme exact readback mismatch')
        row.update(state='verified', live_invalidation=True); self.profile = profile; self.save()

    def send(self, sender, receiver, phase, invite=False):
        text = 'omabit-plugin-' + phase + '-' + uuid.uuid4().hex
        request = uuid.uuid4().hex
        row = {'sender': sender, 'receiver': receiver, 'phase': phase, 'text': text, 'request': request, 'state': 'pending'}
        self.report.setdefault('writes', []).append(row); self.save()
        self.clients[sender].send('~' + receiver, text, request)
        sent = matched_message(self.report['dms'][sender]['events'], 'sent', receiver, text)
        if len(sent) != 1 or sent[0].get('request') != request: raise RuntimeError('matching DMs ACK absent')
        row['state'] = 'acknowledged'; row['message_id'] = sent[0]['message']['id']; self.save()
        if invite:
            self.wait(lambda: '~' + sender in self.clients[receiver].scry('chat', '/dm/invited'), 'DM invitation')
            self.clients[receiver].accept('~' + sender)
            row['invitation_accepted'] = True
        self.wait(lambda: len(matched_message(self.report['dms'][receiver]['events'], 'message', sender, text)) >= 1, 'DM receiver live event')
        received = matched_message(self.report['dms'][receiver]['events'], 'message', sender, text)
        if any(m['message'].get('id') != row['message_id'] for m in received): raise RuntimeError('DM event identity mismatch')
        row.update(state='delivered', receiver_events=len(received))
        self.expected.append((text, sender)); self.histories(phase); self.save()

    def histories(self, phase):
        for ship, peer in [('zod', 'nec'), ('nec', 'zod')]:
            page = self.bridge_module.parse_page(self.clients[ship].scry('chat', '/v4/dm/~' + peer + '/writs/newest/20/light'))
            checked = assert_history(page, self.expected)
            self.report['history'].append({'phase': phase, 'ship': ship, 'messages': checked})

    def before_restart(self):
        self.emit('plugin-login')
        for ship in ('zod', 'nec'): self.dm_client(ship)
        url = 'http://127.0.0.1:' + str(self.ships['nec']['port'])
        self.theme = self.eyre_class(url, timeout=15, ack_timeout=12)
        session = self.theme.login(self.code('nec'))
        self.session = copy.deepcopy(session)
        dm_session = self.clients['nec'].session_store.session
        if session['cookieValue'] == dm_session['cookieValue']: raise RuntimeError('plugins share session')
        self.report['independent_sessions'] = True
        self.session_hashes = {ship: hashlib.sha256(c.session_store.session['cookieValue'].encode()).hexdigest() for ship, c in self.clients.items()}
        self.remote = self.sync.Remote(self.theme)
        self.watch = self.sync.Watch(url, session)
        self.wait(lambda: self.dm_ready('zod') and self.dm_ready('nec') and self.theme_ready(), 'initial subscriptions')
        self.theme_write('before')
        self.send('zod', 'nec', 'first', invite=True)
        self.send('nec', 'zod', 'reply')
        self.report['old_dm_channel'] = self.clients['nec'].channel
        self.report['old_theme_channel'] = next(f['channel'] for f in reversed(self.report['theme']['frames'])
            if f['event'].get('response') == 'subscribe' and f['event'].get('ok') == 'ok')
        self.save(); self.emit('plugin-before-restart-passed')

    def after_restart(self):
        self.wait(lambda: self.dm_ready('nec', self.report['old_dm_channel'])
                  and self.theme_ready(self.report['old_theme_channel']), 'fresh plugin subscriptions after restart', timeout=100)
        if self.remote.read() != self.profile: raise RuntimeError('Theme profile did not persist')
        self.histories('after-restart-before-writes')
        if self.theme.session != self.session or self.session_hashes != {
            ship: hashlib.sha256(c.session_store.session['cookieValue'].encode()).hexdigest() for ship, c in self.clients.items()}:
            raise RuntimeError('plugin reauthenticated on restart')
        self.report['original_sessions_retained'] = True
        self.report['new_dm_channel'] = self.clients['nec'].channel
        self.report['new_theme_channel'] = next(f['channel'] for f in reversed(self.report['theme']['frames'])
            if f['event'].get('response') == 'subscribe' and f['event'].get('ok') == 'ok')
        self.theme_write('after')
        self.send('zod', 'nec', 'after')
        self.send('nec', 'zod', 'after-reply')
        self.report['ok'] = True; self.save(); self.emit('plugin-recovery-passed')

    def close(self):
        if self.watch:
            self.watch.stop.set(); self.watch.thread.join(timeout=55)
            if self.watch.thread.is_alive(): raise RuntimeError('Theme watch did not stop')
            self.report['cleanup'].append('theme-watch-stopped')
        for ship, client in reversed(list(self.clients.items())):
            if client.session_store.session:
                outcome = client.logout()
                self.report['cleanup'].append({'dms_logout': ship, 'outcome': outcome})
            else: client.reset_runtime()
            client.pool.shutdown(wait=True, cancel_futures=True)
            if ship == 'nec' and getattr(self, 'theme', None) and self.report['ok']:
                if self.remote.read() != self.profile: raise RuntimeError('DMs logout invalidated Theme session')
                self.report['independent_logout'] = True
        if getattr(self, 'theme', None) and self.theme.session:
            self.theme.logout(); self.report['cleanup'].append('theme-session-logout')
        self.sync.Eyre = self.original_eyre
        self.save()

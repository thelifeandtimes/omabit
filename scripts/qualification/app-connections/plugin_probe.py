"""Run the real plugin helpers inside the private authentication lab."""
import json
from pathlib import Path
import sys
import threading


def theme(mode, root, profile):
    sys.path.insert(0, root)
    from client.eyre import Eyre
    from client.instance import INSTANCE
    from client.main import Helper
    from client.support import Keyring, StateStore
    assert INSTANCE.profile == profile
    helper = Helper()
    if mode == 'standalone':
        value = json.loads(sys.stdin.buffer.read(1024))
        result = helper.run('login', {'url': 'http://127.0.0.1:18081', 'code': value['code']})
        if not result['ok']:
            raise ValueError('standalone Theme login failed')
    record = StateStore().load()
    if len(record['ships']) != 1:
        raise ValueError('Theme account was not durably stored')
    row = record['ships'][0]
    if mode == 'logout':
        result = helper.run('disconnect', {'expectedAccount': {key: row[key] for key in ('id', 'url', 'ship')}})
        if not result['ok'] or result['warning']:
            raise ValueError('Theme logout was not confirmed')
        return
    session = Keyring().lookup(row['url'], row['ship'])
    client = Eyre(row['url'], session)
    if not isinstance(client.scry(), dict):
        raise ValueError('Theme did not read settings directly')
    stop, ready = threading.Event(), threading.Event()
    failures = []
    def watch():
        try:
            client.watch('talon', stop, threading.Event(), ready)
        except Exception:
            if not stop.is_set():
                failures.append(True)
    watcher = threading.Thread(target=watch, daemon=True)
    watcher.start()
    acknowledged = ready.wait(7)
    stop.set()
    watcher.join(timeout=7)
    if not acknowledged or failures or watcher.is_alive():
        raise ValueError('Theme direct subscription was not confirmed/closed')


def dms(mode, root, profile):
    sys.path.insert(0, root)
    from bridge import Bridge
    from instance import INSTANCE
    from session_store import SessionStore
    assert INSTANCE.profile == profile
    bridge = Bridge(session_store=SessionStore())
    bridge.emit = lambda *_args, **_kwargs: None
    ready = threading.Event()
    original = bridge.on_fact
    def on_fact(outer, event_id):
        if isinstance(outer, dict) and outer.get('response') == 'subscribe' and not outer.get('err'):
            ready.set()
        original(outer, event_id)
    bridge.on_fact = on_fact
    try:
        if mode == 'logout':
            if bridge.logout() != 'revoked':
                raise ValueError('DM logout was not confirmed')
            return
        if mode == 'standalone':
            value = json.loads(sys.stdin.buffer.read(1024))
            ship = bridge.login('http://127.0.0.1:18081', value['code'])
        else:
            ship = bridge.restore()
        if ship != '~zod' or not isinstance(bridge.scry('chat', '/dm'), list):
            raise ValueError('DMs did not restore and read directly')
        if not ready.wait(7):
            raise ValueError('DM direct subscription was not confirmed')
    finally:
        bridge.reset_runtime()
        bridge.pool.shutdown(wait=True, cancel_futures=True)


def main():
    from qualify import check_namespace
    check_namespace()
    plugin, mode = sys.argv[1:3]
    root, profile = '/artifacts/plugins/' + plugin, 'auth-lab'
    if len(sys.argv) != 3:
        if len(sys.argv) != 5 or sys.argv[4] not in ('cli-lab', 'ui-lab'):
            raise ValueError('unsupported private probe profile')
        root, profile = sys.argv[3:]
        canonical = {'theme': 'thelifeandtimes.urbit-theme', 'dms': 'urbit.dms'}[plugin]
        if root != '/lab/' + profile + '/home/.config/omarchy/plugins/' + canonical + '.' + profile:
            raise ValueError('probe root must be its disposable installed copy')
    if mode not in ('read', 'logout', 'standalone'):
        raise ValueError('unsupported lab operation')
    {'theme': theme, 'dms': dms}[plugin](mode, root, profile)
    print(json.dumps({'ok': True, 'plugin': plugin, 'operation': mode}))


if __name__ == '__main__':
    try:
        main()
    except Exception:
        # Keep arbitrary plugin/HTTP exception payloads out of test logs.
        print(json.dumps({'ok': False, 'error': 'plugin-probe-failed'}))
        sys.exit(1)

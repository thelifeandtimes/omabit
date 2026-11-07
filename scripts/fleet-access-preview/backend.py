#!/usr/bin/python3
"""Inert full-panel fleet-access fixture: only /data/state JSON is mutable."""
import json
from pathlib import Path
import subprocess
import sys

mode, scenario, *args = sys.argv[1:]
path = Path('/data/state/fleet-access.json')
state = json.loads(path.read_text()) if path.exists() else {'version': 1, 'revision': 0, 'entries': [], 'attempts': []}
with Path('/data/state/access-calls.jsonl').open('a') as f:
    f.write(json.dumps(args) + '\n')
plugin = {'name': 'Messenger', 'status': 'available', 'target': {'plugin_id': 'fixture.messenger', 'profile': 'default', 'artifact_sha256': 'a' * 64}}
second = {'name': 'Desktop theme', 'status': 'available', 'target': {'plugin_id': 'fixture.theme', 'profile': 'default', 'artifact_sha256': 'b' * 64}}

def option(name):
    return args[args.index(name) + 1]

def approvals():
    return {k: state[k] for k in ('version', 'revision', 'entries')}

def routes(instance):
    return json.loads(subprocess.check_output(['/usr/bin/python3', '-B', '/preview/routes-backend.py', mode, 'ship-routes', '--runner', option('--runner'), '--expected-runner-id', option('--expected-runner-id'), '--instance', instance], text=True))

exitcode = 0
try:
    command = next(x for x in args if x.startswith('app-') or x == 'ships')
    if command == 'app-plugins':
        result = [plugin, second]
    elif command == 'ships':
        result = []
    elif command == 'app-connections':
        result = state['attempts']
    elif command == 'app-ship-approvals':
        result = approvals()
    elif command == 'app-ship-review':
        result = {'version': 1, 'routes': routes(option('--instance')), 'approvals': approvals()}
    elif command == 'app-ship-approve':
        q = json.loads(option('--request-json'))
        assert q['expected_revision'] == state['revision']
        if q['allowed']:
            assert '--allow-owner-session' in args
            r = routes(q['binding']['target']['instance_id'])
            assert r['record']['binding'] == q['binding'] and r['selection'] == q['selection']
            if scenario == 'conflict':
                raise ValueError('Permission changed; refresh before allowing this plugin')
        else:
            assert '--runner' not in args and '--allow-owner-session' not in args
            assert any(e['binding'] == q['binding'] and e['target'] == q['target'] for e in state['entries'])
        state['revision'] += 1
        state['entries'] = [e for e in state['entries'] if not (e['binding'] == q['binding'] and e['target']['plugin_id'] == q['target']['plugin_id'] and e['target']['profile'] == q['target']['profile'])]
        state['entries'].append({'binding': q['binding'], 'target': q['target'], 'allowed': q['allowed'], 'revision': state['revision'], 'updated_at': '2026-11-07T00:00:00Z'})
        result = approvals()
    elif command == 'app-fleet-connect':
        q = json.loads(option('--request-json'))
        assert '--url' not in args and '--approve-owner-session' not in args
        assert any(e['allowed'] and e['binding'] == q['binding'] and e['target'] == q['target'] and e['revision'] == q['grant_revision'] for e in state['entries'])
        r = routes(q['binding']['target']['instance_id'])
        request = option('--request-id')
        assert not any(a['request_id'] == request for a in state['attempts'])
        row = {'request_id': request, 'connection_id': 'c' * 32, 'connection_revision': q['grant_revision'], 'origin': 'https://runner.preview.ts.net', 'ship': q['binding']['ship'], 'target': q['target'], 'authority': 'eyre-owner-session', 'phase': 'connected', 'revision': 5, 'receiver_reconciliation': True, 'next_actions': ['app-disconnect'], 'fleet': {'version': 1, 'runner_name': option('--runner'), 'grant_revision': q['grant_revision'], 'request': {'selection': r['selection'], 'binding': q['binding'], 'registry_revision': r['registry']['revision'], 'route': r['registry']['routes'][1]}}}
        if scenario == 'lost-connect':
            row.update(phase='unresolved', receiver_reconciliation=False, next_actions=['app-recover'])
            exitcode = 1
        state['attempts'].append(row)
        result = {'code': 'recovery_required' if exitcode else 'ok', 'connection': row, 'message': 'The connection reply was lost. Follow its recovery action.' if exitcode else ''}
    elif command in ('app-disconnect', 'app-recover', 'app-reconcile', 'app-archive'):
        row = next(a for a in state['attempts'] if a['request_id'] == args[args.index(command) + 1])
        assert row['revision'] == int(option('--expected-revision'))
        row['revision'] += 1
        if command == 'app-disconnect':
            row.update(phase='closed', next_actions=['app-reconcile'])
        elif command in ('app-reconcile', 'app-recover'):
            row.update(phase='closed', receiver_reconciliation=False, next_actions=['app-archive'])
        else:
            assert row['phase'] == 'closed' and not row['receiver_reconciliation']
            state['attempts'].remove(row)
        result = {'code': 'ok', 'connection': row}
    else:
        raise ValueError('Unexpected fixture command')
    path.write_text(json.dumps(state, indent=2) + '\n')
    print(json.dumps(result))
except (ValueError, AssertionError, KeyError, StopIteration, subprocess.CalledProcessError):
    print(json.dumps({'error': {'message': 'Permission or ship changed; refresh before continuing.'}}), file=sys.stderr)
    exitcode = 1
sys.exit(exitcode)

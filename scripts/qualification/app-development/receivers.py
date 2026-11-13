"""Optional two-fake receiver phase inside the already admitted private capsule."""
import json
import os
from pathlib import Path
import signal
import subprocess
import time
import urllib.request


def interrupted_boot_source(previous):
    """Admit only the recorded OOM before either receiver install was attempted.

    This does not turn an unclean result into a clean one or authorize replay.
    The next native test must first re-observe the original publisher hash and
    both receivers' unused target names before its first receiver effects.
    """
    if previous.is_symlink() or (previous / 'lab').is_symlink():
        raise ValueError('physical private capsule required')
    guard = json.loads((previous / 'guard.json').read_text())
    result = json.loads((previous / 'lab/result.json').read_text())
    termination = json.loads((previous / 'scope-termination.json').read_text())
    unit = guard['unit'] + '.scope'
    if not guard.get('receivers') or result.get('clean_shutdown') or termination['unit'] != unit or termination['observed']['Result'] != 'oom-kill':
        raise ValueError('only recorded pre-install receiver OOM is eligible')
    state = subprocess.run(['systemctl', '--user', 'show', unit,
                            '--property=ActiveState,Result,ControlGroup'],
                           capture_output=True, text=True, check=True)
    values = dict(line.split('=', 1) for line in state.stdout.splitlines() if '=' in line)
    if values != {'ActiveState': 'failed', 'Result': 'oom-kill', 'ControlGroup': ''}:
        raise ValueError('original OOM scope is not proved inactive')
    if (previous / 'lab/receiver-proof.json').exists() or any((previous / 'lab').glob('receiver-*effects.jsonl')):
        raise ValueError('receiver effects require separate reconciliation, never replay')
    proof = json.loads((previous / 'lab/workflow-proof.json').read_text())
    if proof.get('ok') is not True or result.get('workflow_exit') not in (None, 0):
        raise ValueError('publisher workflow did not complete')
    ledger = {}
    for line in (previous / 'lab/workflow-effects.jsonl').read_text().splitlines():
        row = json.loads(line)
        ledger[row['id']] = row['state']
    if not ledger or any(value != 'acknowledged' for value in ledger.values()):
        raise ValueError('publisher has an uncertain native effect')
    for name in ('zod', 'nec', 'bud'):
        p = previous / 'lab' / name
        if not p.is_dir() or p.is_symlink() or p.stat().st_uid != os.getuid():
            raise ValueError('owned private peer unavailable')
        for relative in ('.urb', '.urb/log', '.urb/log/data.mdb'):
            if (p / relative).is_symlink() or not (p / relative).exists():
                raise ValueError('private peer log unavailable')
    return proof['glob-upload-committed-docket']


def run_receivers(lab, guard, desk):
    # The parent checks the full namespace contract; repeat the decisive boundary
    # before spawning any new child. No host runtime or existing pier is selected.
    for kind in ('net', 'pid', 'mnt', 'user'):
        if os.readlink('/proc/self/ns/' + kind) == guard['namespaces'][kind]:
            raise RuntimeError('host namespace refused')
    if any(Path(p).exists() for p in ('/home', '/run/user', '/var/run/docker.sock')):
        raise RuntimeError('host access refused')
    class NoRedirect(urllib.request.HTTPRedirectHandler):
        def redirect_request(self, *args, **kwargs):
            raise RuntimeError('redirect refused')
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())
    children, metadata = [], []
    result = {'ok': False, 'clean_shutdown': False, 'children': []}
    def save():
        (lab / 'receiver-result.json').write_text(json.dumps(result, indent=2) + '\n')
    save()
    try:
        for name, port, ames in [('nec', 18082, 31338), ('bud', 18083, 31339)]:
            pier = lab / name
            resume = bool(guard.get('recover_receiver_boot'))
            if pier.exists() != resume or pier.is_symlink():
                raise RuntimeError('receiver pier selection changed')
            log = (lab / (name + '.log')).open('xb')
            args = ['/artifacts/vere', '-t', '-L', '--no-conn', '--no-dock',
                    '--loom', '31', '--snap-time', '60', '-b', '127.0.0.1',
                    '--http-port', str(port), '-p', str(ames), '-F', name,
                    '-B', '/artifacts/boot.pill', '-c', str(pier)]
            if resume:
                args = args[:args.index('-F')] + [str(pier)]
            try:
                child = subprocess.Popen(args, stdin=subprocess.DEVNULL, stdout=log,
                                         stderr=subprocess.STDOUT, start_new_session=True)
            except BaseException:
                log.close()
                raise
            children.append((name, child, log))
            end = time.monotonic() + 240
            while child.poll() is None and time.monotonic() < end:
                try:
                    with opener.open(f'http://127.0.0.1:{port}/~/host', timeout=1) as response:
                        if response.status == 200 and response.read(128).strip() == ('~' + name).encode():
                            break
                except OSError:
                    pass
                time.sleep(1)
            else:
                raise RuntimeError('receiver readiness deadline: ' + name)
            rows = [row.split() for row in (pier / '.http.ports').read_text().splitlines()]
            ports = [int(row[0]) for row in rows if row[1:] == ['insecure', 'loopback']]
            if len(ports) != 1 or not 1024 <= ports[0] <= 65535:
                raise RuntimeError('receiver Lens inventory invalid')
            stat = Path(f'/proc/{child.pid}/stat').read_text()
            metadata.append({'pid': child.pid, 'start': stat[stat.rfind(')') + 2:].split()[19],
                             'desk': desk, 'ship_name': '~' + name, 'lens_port': ports[0],
                             'http_port': port,
                             'vere_sha256': 'b5c54439ae5b9f0ff94d75319e67e04f04369776fa5e2180a5ab1a5ebd4c2ff0'})
            print('RECEIVER ready ' + name, flush=True)
        (lab / 'receivers-runtime.json').write_text(json.dumps(metadata, indent=2) + '\n')
        with (lab / 'receiver-workflow.log').open('xb') as log:
            run = subprocess.run(['/artifacts/workflow.test', '-test.run=^TestNativeDeskReceivers$',
                                  '-test.v', '-test.timeout=5m'], stdout=log, stderr=subprocess.STDOUT,
                                 timeout=310)
        result['workflow_exit'] = run.returncode
        result['ok'] = run.returncode == 0 and json.loads((lab / 'receiver-proof.json').read_text()).get('ok') is True
    except Exception as error:
        result['error'] = type(error).__name__ + ': ' + str(error)
    finally:
        for name, child, log in reversed(children):
            forced = False
            if child.poll() is None:
                os.killpg(child.pid, signal.SIGTERM)
            try:
                child.wait(timeout=60)
            except subprocess.TimeoutExpired:
                forced = True
                os.killpg(child.pid, signal.SIGKILL)
                child.wait(timeout=5)
            log.close()
            result['children'].append({'ship': name, 'exit': child.returncode, 'forced': forced})
        result['clean_shutdown'] = len(children) == 2 and all(row['exit'] == 0 and not row['forced'] for row in result['children'])
        result['ok'] = result['ok'] and result['clean_shutdown']
        save()
    return result

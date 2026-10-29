#!/usr/bin/env python3
"""Exercise the built collector with the built in-memory simulator in isolation."""
import argparse
import hashlib
import http.client
import http.server
import json
import os
from pathlib import Path
import re
import shutil
import socket as socketlib
import socketserver
import threading
import subprocess
import time


def sha(path):
    with path.open('rb') as file:
        return hashlib.file_digest(file, 'sha256').hexdigest()


class ReplyLossProxy:
    """Private simulator HTTP/Unix proxy; drop one already-accepted reply."""
    def __init__(self, path, upstream):
        self.receipts=[]
        owner=self
        class Connection(http.client.HTTPConnection):
            def connect(self):
                self.sock=socketlib.socket(socketlib.AF_UNIX,socketlib.SOCK_STREAM)
                self.sock.settimeout(5)
                self.sock.connect(upstream)
        class Handler(http.server.BaseHTTPRequestHandler):
            protocol_version='HTTP/1.1'
            def log_message(self,*args):pass
            def forward(self):
                length=int(self.headers.get('Content-Length','0'))
                if not 0<=length<=1024*1024:raise RuntimeError('simulator request bound')
                data=self.rfile.read(length)
                connection=Connection('private-simulator',timeout=5)
                try:
                    headers={k:v for k,v in self.headers.items() if k.lower() not in ('host','connection')}
                    connection.request(self.command,self.path,body=data if length else None,headers=headers)
                    response=connection.getresponse();body=response.read(1024*1024+1)
                    if len(body)>1024*1024:raise RuntimeError('simulator response bound')
                    request=json.loads(data) if data and self.path.endswith('/operations') else {}
                    if request.get('request_id')=='lost-stop-reply':
                        owner.receipts.append({'status':response.status,'body':json.loads(body)})
                        if len(owner.receipts)==1:
                            self.close_connection=True
                            self.connection.shutdown(socketlib.SHUT_RDWR)
                            return
                    self.send_response(response.status)
                    # Preserve the runner protocol/identity headers. The proxy
                    # changes framing only; otherwise it must be transparent.
                    for key,value in response.getheaders():
                        if key.lower() not in ('content-length','transfer-encoding','connection','date','server'):
                            self.send_header(key,value)
                    self.send_header('Content-Length',str(len(body)))
                    self.end_headers();self.wfile.write(body)
                except OSError:
                    self.close_connection=True
                    try:self.connection.shutdown(socketlib.SHUT_RDWR)
                    except OSError:pass
                finally:
                    connection.close()
            do_GET=forward
            do_POST=forward
            do_PUT=forward
        class Server(socketserver.ThreadingMixIn,socketserver.UnixStreamServer):
            daemon_threads=True
        self.server=Server(path,Handler)
        self.thread=threading.Thread(target=self.server.serve_forever,kwargs={'poll_interval':0.1},daemon=True)
        self.thread.start()
    def close(self):
        self.server.shutdown();self.server.server_close();self.thread.join(timeout=3)


def inside():
    lab = Path('/lab')
    if (lab / 'marker').read_text() != 'omabit-operator-simulation-v1\n':
        raise RuntimeError('private simulation marker required')
    records = []
    process = None
    log = (lab / 'simulator.log').open('wb')
    socket = '/lab/run/runner.sock'
    proxy = ReplyLossProxy('/lab/run/collector.sock',socket)

    def start():
        child = subprocess.Popen(['/lab/runner', 'demo', '--socket', socket,
                                  '--operation-delay', '2s'], stdout=log, stderr=log)
        deadline = time.monotonic() + 5
        while not Path(socket).exists():
            if child.poll() is not None or time.monotonic() > deadline:
                child.terminate()
                child.wait(timeout=5)
                raise RuntimeError('private simulator did not start')
            time.sleep(0.02)
        return child

    def stop(child):
        child.terminate()
        if child.wait(timeout=5) != 0:
            raise RuntimeError('private simulator did not stop cleanly')

    def call(args, *, format='json', selected=False, expected=0):
        command = ['/lab/collector', '--config-dir', '/lab/registry', '--format', format]
        if selected:
            command += ['--runner', 'lab', '--expected-runner-id', runner_id]
        command += args
        result = subprocess.run(command, capture_output=True, text=True, timeout=10)
        records.append({'args': command[1:], 'exit': result.returncode,
                        'stdout': result.stdout, 'stderr': result.stderr})
        if result.returncode != expected or (expected == 0 and result.stderr):
            raise RuntimeError('unexpected CLI result: ' + json.dumps(records[-1]))
        stream = result.stderr if expected and result.stderr else result.stdout
        return json.loads(stream) if format == 'json' else stream

    try:
        process = start()
        call(['help'])
        call(['help', 'ships'], format='text')
        enrolled = call(['enroll', 'lab', '--socket', '/lab/run/collector.sock'])
        runner_id = enrolled['runner_id']
        runners = call(['runners'])
        if len(runners) != 1 or runners[0]['runner_id'] != runner_id:
            raise RuntimeError('runner selection did not preserve verified identity')
        if runner_id not in call(['runners'], format='text'):
            raise RuntimeError('readable selection lacks runner identity')
        hello = call(['hello'], selected=True)
        if hello['runner_id'] != runner_id or hello['mode'] != 'simulation':
            raise RuntimeError('wrong simulator identity or mode')
        call(['hello'], selected=True, format='text')
        before = call(['snapshot'], selected=True)
        instance = next(v for v in before['instances'] if v['id'] == 'alpha')
        revision = instance['config_revision']
        config = instance['config']

        # Simulate another operator staging a change after our initial read.
        changed = call(['configure', 'alpha', '--revision', str(revision),
                        '--http-port', str(config['http_port']), '--ames-port', str(config['ames_port']),
                        '--loom', '32'], selected=True)
        if changed['config_revision'] != revision + 1:
            raise RuntimeError('simulated concurrent edit did not advance revision')
        for format in ('json', 'text'):
            error = call(['operate', 'alpha', 'start', '--revision', str(revision),
                          '--request-id', 'rejected-stale-' + format], selected=True,
                         format=format, expected=1)
            if format == 'json':
                if error['error']['code'] != 'revision_conflict' or error['error']['current_revision'] != revision + 1:
                    raise RuntimeError('structured revision conflict lacks current revision')
                if error.get('guidance',{}).get('kind')!='review_current_revision':
                    raise RuntimeError('revision conflict lacks structured review guidance')
            elif 'revision_conflict' not in error:
                raise RuntimeError('readable revision conflict is ambiguous')
        fresh = call(['snapshot'], selected=True)
        if fresh['operations']:
            raise RuntimeError('stale operation was dispatched')
        current = next(v for v in fresh['instances'] if v['id'] == 'alpha')
        revision = current['config_revision']

        # After reviewing the fresh state, submit the still-intended operation.
        request = ['operate', 'alpha', 'start', '--revision', str(revision),
                   '--request-id', 'reviewed-start']
        accepted = call(request, selected=True)
        if accepted['status'] not in ('pending', 'running'):
            raise RuntimeError('submission did not expose asynchronous status')
        visible = call(['snapshot'], selected=True, format='text')
        if accepted['id'] not in visible or 'reviewed-start' not in visible:
            raise RuntimeError('readable snapshot lacks the exact receipt')
        deadline = time.monotonic() + 10
        while True:
            snapshot = call(['snapshot'], selected=True)
            operation = next(v for v in snapshot['operations'] if v['id'] == accepted['id'])
            if operation['status'] == 'succeeded':
                break
            if operation['status'] not in ('pending', 'running') or time.monotonic() > deadline:
                raise RuntimeError('simulated operation did not finish')
            time.sleep(0.2)
        state = next(v for v in snapshot['instances'] if v['id'] == 'alpha')
        if state['observed_state'] != 'running' or state.get('active_operation_id'):
            raise RuntimeError('terminal receipt disagrees with resulting state')
        call(['snapshot'], selected=True, format='text')
        replay = call(request, selected=True)
        if replay['id'] != accepted['id']:
            raise RuntimeError('terminal request replay submitted duplicate work')

        # The runner accepts this mutation, but its reply never reaches the CLI.
        lost_request=['operate','alpha','stop','--revision',str(state['config_revision']),
                      '--request-id','lost-stop-reply']
        missing_reply=call(lost_request,selected=True,expected=1)
        if not missing_reply.get('error') or len(proxy.receipts)!=1 or proxy.receipts[0]['status'] not in (200,202):
            raise RuntimeError('lost-ack scenario was not established')
        if missing_reply.get('guidance',{}).get('kind')!='reconcile_uncertain_request':
            raise RuntimeError('uncertain mutation lacks structured receipt guidance')
        lost_receipt=proxy.receipts[0]['body']
        # Observe exact receipt before considering replay. No new request ID.
        deadline=time.monotonic()+10
        while True:
            snapshot=call(['snapshot'],selected=True)
            matches=[v for v in snapshot['operations'] if v['request_id']=='lost-stop-reply']
            if len(matches)!=1 or matches[0]['id']!=lost_receipt['id']:
                raise RuntimeError('lost response lost/duplicated the durable receipt')
            if matches[0]['status']=='succeeded':break
            if time.monotonic()>deadline:raise RuntimeError('lost-reply operation did not finish')
            time.sleep(.2)
        replay=call(lost_request,selected=True)
        if replay['id']!=lost_receipt['id'] or len(proxy.receipts)!=2:
            raise RuntimeError('lost-reply reconciliation submitted duplicate work')
        after=call(['snapshot'],selected=True)
        if len([v for v in after['operations'] if v['request_id']=='lost-stop-reply'])!=1:
            raise RuntimeError('request replay created another operation')
        current=next(v for v in after['instances'] if v['id']=='alpha')
        resumed=call(['operate','alpha','start','--revision',str(current['config_revision']),
                      '--request-id','resume-after-transport-test'],selected=True)
        deadline=time.monotonic()+10
        while True:
            snapshot=call(['snapshot'],selected=True)
            row=next(v for v in snapshot['operations'] if v['id']==resumed['id'])
            if row['status']=='succeeded':break
            if time.monotonic()>deadline:raise RuntimeError('simulator resume failed')
            time.sleep(.2)
        if next(v for v in snapshot['instances'] if v['id']=='alpha')['observed_state']!='running':
            raise RuntimeError('simulator was not running before availability check')

        # An unavailable endpoint must not turn last-known running into stopped.
        stop(process)
        process = None
        unavailable = call(['inventory'], expected=1)
        observation = unavailable['connections'][0]
        if observation['status'] != 'unavailable' or observation.get('snapshot') is not None:
            raise RuntimeError('unavailable runner was given a current ship snapshot')
        text = call(['inventory'], format='text', expected=1)
        if 'unavailable' not in text or 'alpha' in text or 'stopped' in text:
            raise RuntimeError('readable unavailable result invents a ship state')

        # A new simulator at the same socket is a different runner, not a repair.
        process = start()
        mismatch = call(['hello'], selected=True, expected=1)
        if mismatch['error']['code'] != 'runner_identity_mismatch':
            raise RuntimeError('replacement endpoint was silently trusted')
        if mismatch.get('guidance',{}).get('kind')!='resolve_runner_identity':
            raise RuntimeError('identity mismatch lacks structured selection guidance')
        if call(['runners'])[0]['runner_id'] != runner_id:
            raise RuntimeError('identity mismatch changed enrollment')
        result = {'passed': True, 'scope': 'in-memory simulator only; no real runtime, import, engine, desktop or ship',
                  'tasks': ['select intended runner', 'review stale revision before action',
                            'follow exact asynchronous receipt', 'reconcile a lost operation reply without duplicate work',
                            'distinguish unavailable from stopped',
                            'refuse replacement runner identity'],
                  'runner_id': runner_id, 'operation_id': accepted['id'],
                  'lost_reply':{'operation_id':lost_receipt['id'],'request_id':'lost-stop-reply','forwarded_requests':len(proxy.receipts),'one_terminal_operation':True}, 'records': records}
        (lab / 'result.json').write_text(json.dumps(result, indent=2) + '\n')
        print(json.dumps({'passed': True, 'tasks': len(result['tasks']), 'commands': len(records)}), flush=True)
    finally:
        if process is not None and process.poll() is None:
            stop(process)
        proxy.close()
        log.close()
        (lab / 'transcript.json').write_text(json.dumps(records, indent=2) + '\n')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--collector', type=Path)
    parser.add_argument('--runner', type=Path)
    parser.add_argument('--output', type=Path)
    parser.add_argument('--inside', action='store_true', help=argparse.SUPPRESS)
    args = parser.parse_args()
    if args.inside:
        inside()
        return
    if not all((args.collector, args.runner, args.output)):
        parser.error('--collector, --runner and --output are required')
    work = args.output.absolute()
    if work.parent != Path('/tmp') or not re.fullmatch('omabit-operator-walkthrough-[A-Za-z0-9-]+', work.name) or work.exists():
        parser.error('use a fresh /tmp/omabit-operator-walkthrough-NAME')
    os.umask(0o077)
    work.mkdir(mode=0o700)
    for directory in ('home', 'run'):
        (work / directory).mkdir(mode=0o700)
    for source, name in ((args.collector, 'collector'), (args.runner, 'runner'), (Path(__file__), 'driver.py')):
        shutil.copyfile(source, work / name)
        (work / name).chmod(0o700)
    (work / 'marker').write_text('omabit-operator-simulation-v1\n')
    inputs = {'source_sha256': sha(Path(__file__)), 'collector_sha256': sha(work / 'collector'), 'runner_sha256': sha(work / 'runner')}
    (work / 'inputs.json').write_text(json.dumps(inputs, indent=2) + '\n')
    namespace = ['bwrap', '--unshare-all', '--die-with-parent', '--clearenv', '--ro-bind', '/usr', '/usr',
                 '--symlink', 'usr/bin', '/bin', '--symlink', 'usr/lib', '/lib', '--symlink', 'usr/lib', '/lib64',
                 '--proc', '/proc', '--dev', '/dev', '--tmpfs', '/tmp', '--bind', str(work), '/lab', '--chdir', '/lab',
                 '--setenv', 'HOME', '/lab/home', '--setenv', 'XDG_RUNTIME_DIR', '/lab/run',
                 '--setenv', 'PATH', '/usr/bin:/bin', '--', '/usr/bin/python3', '-I', '-B', '/lab/driver.py', '--inside']
    result = subprocess.run(namespace, capture_output=True, text=True, timeout=60)
    (work / 'driver.log').write_text(result.stdout + result.stderr)
    if result.returncode:
        raise RuntimeError('walkthrough failed; retained evidence: ' + str(work))
    print(result.stdout.strip())
    print(work)


if __name__ == '__main__':
    main()

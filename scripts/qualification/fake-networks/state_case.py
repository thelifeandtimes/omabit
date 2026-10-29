"""Bounded persistent Gall state and Eyre reconnect qualification on fresh fakes."""
import hashlib
import json
from pathlib import Path
import re
import secrets
import time
from network_case import jam, thread_receipt


def parse_sse(lines):
    """Parse complete bounded SSE events without treating partial data as a fact."""
    size = 0
    identity = None
    data = []
    for raw in lines:
        size += len(raw)
        if not raw or len(raw) > 8192 or size > 65536:
            raise ValueError('SSE bound/EOF')
        line = raw.decode().rstrip('\r\n')
        if line.startswith('id:'):
            identity = line[3:].strip()
        elif line.startswith('data:'):
            data.append(line[5:].strip())
        elif not line:
            if data:
                if identity is None or not re.fullmatch('0|[1-9][0-9]{0,15}', identity):
                    raise ValueError('invalid SSE identity')
                event = json.loads('\n'.join(data))
                if not isinstance(event, dict):
                    raise ValueError('invalid SSE event')
                yield int(identity), event
            identity = None
            data = []
    if data:
        raise ValueError('incomplete SSE event')


def matching_state(event, expected, subscription=1):
    if type(event.get('id')) is not int or event['id'] != subscription:
        return False
    if event.get('response') == 'quit' or 'err' in event:
        raise ValueError('subscription failed')
    if event.get('response') != 'diff':
        return False
    value = event.get('json')
    if not isinstance(value, dict) or type(value.get('revision')) is not int:
        raise ValueError('invalid application snapshot')
    if value != expected:
        raise ValueError('unexpected state: ' + json.dumps(value, sort_keys=True))
    return True


def state_poke(receiver, entry, receipt):
    if receiver not in ('zod', 'nec') or not re.fullmatch('omabit-state-[0-9a-f]{24}', entry) or not re.fullmatch('omabit-receipt-[0-9a-f]{32}', receipt):
        raise ValueError('invalid state poke binding')
    code = "=/  m  (strand ,vase)  ;<  ~  bind:m  (poke [~"+receiver+" %omabit-state] %noun !>('"+entry+"'))  (pure:m !>('"+receipt+"'))"
    return jam(int.from_bytes(code.encode(), 'little'))


class Recovery:
    def __init__(self, http, events, save, emit, target="nec", sender="zod"):
        if target not in ("zod", "nec") or sender not in ("zod", "nec"):
            raise ValueError("fixture identities must be disposable zod/nec")
        self.target, self.sender = target, sender
        self.http = http
        self.events = events
        self.save = save
        self.emit = emit
        self.channel = None
        self.cursor = None
        self.entries = []
        self.report = {'ok': False, 'writes': [], 'observations': [], 'channels_deleted': []}

    def stage(self, stage):
        self.report['stage'] = stage
        self.save()
        self.emit('application-state', phase=stage)

    def install(self, pier, source):
        self.stage('mount-test-desk')
        def setup(action):
            receipt='omabit-receipt-'+secrets.token_hex(16)
            expressions={
                'mount':";<  b=beak  bind:m  get-beak  ;<  ~  bind:m  (poke [~nec %hood] %kiln-mount !>([(en-beam [b ~]) %base]))",
                'commit':";<  ~  bind:m  (poke [~nec %hood] %kiln-commit !>([%base %.n]))",
                'start':";<  ~  bind:m  (poke [~nec %hood] %kiln-rein !>([%base (~(put by *(map @tas ?)) %omabit-state &)]))",
            }
            code="=/  m  (strand ,vase)  "+expressions[action].replace("~nec", "~"+self.target)+"  (pure:m !>(s+'"+receipt+"'))"
            attempt={'action':action,'state':'pending'}
            self.report.setdefault('installation',[]).append(attempt);self.save()
            status,body=self.http(self.target,'/spider/base/ted-eval/khan-eval/json',jam(int.from_bytes(code.encode(),'little')),content='application/x-urb-jam',accept='application/json')
            attempt.update(status=status,bytes=len(body));self.save()
            try:attempt['response']=json.loads(body)
            except ValueError:attempt['response_text']=body.decode(errors='replace');self.save();raise RuntimeError('fixture '+action+' returned non-JSON HTTP '+str(status))
            self.save()
            if status!=200 or attempt['response']!=receipt:raise RuntimeError('fixture '+action+' failed; bounded trace retained')
            attempt.update(state='acknowledged',receipt=receipt);self.save()
        setup('mount')
        path = Path(pier)/'base/app'
        deadline = time.monotonic()+60
        while not path.is_dir() and time.monotonic() < deadline:
            time.sleep(.2)
        if not path.is_dir():
            raise RuntimeError('fresh base mount unavailable')
        target = path/'omabit-state.hoon'
        with target.open('xb') as out:
            out.write(Path(source).read_bytes())
        # Docker rootless maps container root to lab. Preserve mount ownership.
        import os
        os.chown(target, path.stat().st_uid, path.stat().st_gid)
        self.report['fixture_sha256'] = hashlib.sha256(target.read_bytes()).hexdigest()
        setup('commit')
        # Commit acknowledgement precedes the asynchronous filesystem import.
        # Poll a read-only expression; never repeat an uncertain mutation.
        deadline=time.monotonic()+60
        while time.monotonic()<deadline:
            expression='=/  src  .^(@t %cx %/app/omabit-state/hoon)  %.y'
            status,body=self.http(self.target,'/',json.dumps({'source':{'dojo':expression},'sink':{'stdout':None}}).encode(),content='application/json',admin=True)
            if status==200 and json.loads(body).strip()=='%.y':break
            time.sleep(.5)
        else:raise RuntimeError('committed fixture unavailable')
        self.report['start_diagnostics']=[]
        diagnostics=[
            ('lens-domes', '=/  c  .^(cone:clay %cx /~nec//(scot %da now)/domes)  =(~ c)'),
            ('lens-rein', '=/  c  .^(cone:clay %cx /~nec//(scot %da now)/domes)  =/  d  (~(gut by c) [~nec %base] *dome:clay)  ~(tap by ren.d)'),
        ]
        for label,expression in diagnostics:
            expression=expression.replace("~nec", "~"+self.target)
            status,body=self.http(self.target,'/',json.dumps({'source':{'dojo':expression},'sink':{'stdout':None}}).encode(),content='application/json',admin=True)
            self.report['start_diagnostics'].append({'label':label,'status':status,'body':body.decode(errors='replace')});self.save()
            if label=='lens-rein' and (status!=200 or json.loads(body).strip()!='~'):raise RuntimeError('fresh fixture requires an empty extra-app map; preserve any existing entries')
        setup('start')
        self.stage('waiting-for-test-app')
        deadline = time.monotonic()+120
        while time.monotonic() < deadline:
            status, body = self.http(self.target, '/~/scry/omabit-state/state.json')
            if status == 200:
                self.verify_state(status, body)
                return
            time.sleep(1)
        raise RuntimeError('fresh fixture did not become ready')

    def expected(self):
        return {'revision': len(self.entries), 'entries': self.entries.copy()}

    def verify_state(self, status, body):
        if status != 200 or json.loads(body) != self.expected():
            raise RuntimeError('durable state mismatch')
        self.report['observations'].append({'via': 'scry', 'state': self.expected()})
        self.save()

    def send(self, actions):
        status, _ = self.http(self.target, self.channel, json.dumps(actions).encode(), content='application/json', method='PUT')
        if status not in (200, 204):
            raise RuntimeError('channel operation refused')

    def subscribe(self):
        self.channel = '/~/channel/omabit-state-'+secrets.token_hex(16)
        self.cursor = None
        self.report.setdefault('channels', []).append(self.channel)
        self.send([{'id': 1, 'action': 'subscribe', 'ship': self.target, 'app': 'omabit-state', 'path': '/state'}])
        self.receive(initial=True)

    def receive(self, initial=False):
        expected = self.expected()
        rows = self.events(self.channel, self.cursor, expected, initial)
        facts = [(identity, event) for identity, event in rows if matching_state(event, expected)]
        if len(facts) != 1 or (initial and not any(event.get('response') == 'subscribe' and type(event.get('id')) is int and event['id'] == 1 and event.get('ok') == 'ok' for _, event in rows)):
            raise RuntimeError('matched subscription/snapshot absent')
        ids = [identity for identity, _ in rows]
        if ids != sorted(set(ids)) or (self.cursor is not None and any(identity <= self.cursor for identity in ids)):
            raise RuntimeError('SSE replay/out-of-order event')
        self.cursor = max(ids)
        self.send([{'action': 'ack', 'event-id': self.cursor}])
        self.report['observations'].append({'via': 'subscription', 'channel': self.channel, 'events': rows, 'cursor': self.cursor, 'state': expected})
        self.save()

    def write(self, phase):
        entry = 'omabit-state-'+secrets.token_hex(12)
        receipt = 'omabit-receipt-'+secrets.token_hex(16)
        attempt = {'phase': phase, 'entry': entry, 'receipt': receipt, 'state': 'pending'}
        self.report['writes'].append(attempt)
        self.save()
        status, body = self.http(self.sender, '/spider/base/ted-eval/khan-eval/noun', state_poke(self.target, entry, receipt), content='application/x-urb-jam', accept='application/x-urb-jam')
        attempt.update(http_status=status,response_bytes=len(body),response_sha256=hashlib.sha256(body).hexdigest())
        if status!=200:attempt['failure_body_hex']=body.hex()
        self.save()
        thread_receipt(status, body, receipt)
        attempt['state'] = 'acknowledged'
        self.entries.append(entry)
        self.verify_state(*self.http(self.target, '/~/scry/omabit-state/state.json'))

    def before_restart(self):
        self.stage('subscribe-and-write')
        self.subscribe()
        self.write('connected')
        self.receive()
        self.stage('queue-while-disconnected')
        self.write('disconnected-before-restart')
        self.report['resume_cursor'] = self.cursor
        self.report['resume_channel'] = self.channel
        self.save()

    def after_restart(self):
        self.stage('recover-existing-subscription')
        self.verify_state(*self.http(self.target, '/~/scry/omabit-state/state.json'))
        self.receive()
        self.write('existing-subscription-after-restart')
        self.receive()
        self.close()
        self.stage('fresh-client-reconciliation')
        self.subscribe()
        self.write('fresh-subscription-after-restart')
        self.receive()
        self.verify_state(*self.http(self.target, '/~/scry/omabit-state/state.json'))
        self.close()
        self.report.update(ok=True, state_survived=True, queued_update_survived=True, existing_subscription_survived=True, fresh_subscription_reconciled=True, writes_replayed=False)
        self.stage('complete')

    def close(self):
        if self.channel:
            self.send([{'action': 'delete'}])
            self.report['channels_deleted'].append(self.channel)
            self.channel = None
            self.save()

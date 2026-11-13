"""Private remote desktop side of the offline release VM fleet test."""
import hashlib
import json
import os
from pathlib import Path
import secrets
import socket
import socketserver
import subprocess
import sys
import threading
import time

root=Path(sys.argv[1]);assert root==Path('/home/review/fleet-qualification') and Path.home()==root/'home'
assert 'omabit.disposable-guest=release-upgrade-v1' in Path('/proc/cmdline').read_text().split()
assert sorted(name for _,name in socket.if_nameindex())==['lo','ofleet-client']
meta=json.loads((root/'input.json').read_text());collector=meta['runner_tool'];omabit=meta['omabit_tool'];runner=meta['runner_id']
registry=root/'config/urbit-collector';report={'passed':False,'connections':[]};calls=[]

def call(args,expected=0):
    r=subprocess.run([str(a) for a in args],capture_output=True,text=True,timeout=190)
    # These APIs return only nonsecret receipts. Retain outputs privately, never
    # export the Secret Service contents or raw SSH/runtime diagnostics.
    calls.append({'args':[str(a) for a in args],'exit':r.returncode,'out':r.stdout,'error':r.stderr})
    (root/'evidence/commands.json').write_text(json.dumps(calls,indent=2)+'\n')
    if r.returncode!=expected:raise RuntimeError('fleet command failed: '+str(args[1])+' '+r.stderr[-500:])
    return json.loads(r.stdout) if r.stdout.strip() else None

def cli(*args,pin=False,expected=0):
    return call([collector,'--config-dir',registry,*(['--runner','remote','--expected-runner-id',runner] if pin else []),*args],expected)

def inspect():
    return call([omabit,'--config-dir',registry,'ship-routes','--runner','remote','--expected-runner-id',runner,'--instance','upgrade-zod'])

def save(): (root/'evidence/result.json').write_text(json.dumps(report,indent=2)+'\n')

class Decoy(socketserver.BaseRequestHandler):
    def handle(self):
        report['loopback_requests']+=1
        self.request.recv(16384)
        self.request.sendall(b'HTTP/1.1 200 OK\r\nContent-Length: 4\r\nConnection: close\r\n\r\n~bud')

keyring=None;decoy=None
try:
    report['loopback_requests']=0
    decoy=socketserver.ThreadingTCPServer(('127.0.0.1',19180),Decoy)
    threading.Thread(target=decoy.serve_forever,daemon=True).start()
    keyring=subprocess.Popen(['gnome-keyring-daemon','--foreground','--unlock','--components=secrets'],stdin=subprocess.PIPE,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
    keyring.stdin.write(secrets.token_hex(32).encode());keyring.stdin.close()
    for _ in range(100):
        r=subprocess.run(['busctl','--user','call','org.freedesktop.DBus','/org/freedesktop/DBus','org.freedesktop.DBus','GetConnectionUnixProcessID','s','org.freedesktop.secrets'],capture_output=True,text=True)
        if r.stdout.strip()=='u '+str(keyring.pid):break
        assert keyring.poll() is None;time.sleep(.1)
    else:raise RuntimeError('private Secret Service unavailable')
    entry=cli('enroll','remote','--ssh','qualification-runner','--ssh-config',root/'ssh_config')
    routes=inspect();assert routes['runner_id']==runner
    assert routes['record']['revision']==0
    settings={'disabled_automatic':[],'routes':[{'id':name,'origin':'http://10.77.40.1:'+str(port),'network':'lan','scope':'qualification','source':'manual','enabled':True} for name,port in [('a-wrong-ship',20080),('b-primary',20081),('c-alternate',20082)]]}
    request={'selection':routes['selection'],'update':{'binding':routes['record']['binding'],'expected_revision':0,'settings':settings}}
    saved=call([omabit,'--config-dir',registry,'ship-routes-configure','--runner','remote','--expected-runner-id',runner,'--request-json',json.dumps(request)])
    assert saved['record']['revision']==1
    assert inspect()['record']==saved['record']
    discovered=cli('app-plugins');assert len(discovered)==2 and all(p['status']=='available' for p in discovered)
    previous=[]
    for plugin in discovered:
        target=plugin['target']
        review=cli('app-ship-review','--instance','upgrade-zod',pin=True)
        grant={'version':1,'expected_revision':review['approvals']['revision'],'binding':review['routes']['record']['binding'],'target':target,'allowed':True,'selection':review['routes']['selection']}
        approved=cli('app-ship-approve','--request-json',json.dumps(grant),'--allow-owner-session',pin=True)
        # Read back the durable grant; use its actual revision, never infer one.
        choices=cli('app-ship-approvals');chosen=next(g for g in choices['entries'] if g['target']==target)
        connect={'version':1,'binding':grant['binding'],'target':target,'grant_revision':chosen['revision'],'lan_profiles':['qualification']}
        row=cli('app-fleet-connect','--request-json',json.dumps(connect),'--request-id',secrets.token_hex(16),pin=True)['connection']
        assert row['phase']=='connected' and row['origin']=='http://10.77.40.1:20081'
        health=cli('app-fleet-health',row['request_id'],'--expected-revision',str(row['revision']),'--lan-profile','qualification')
        assert health['health']=='healthy',health
        previous.append(row);report['connections'].append({'plugin':target['plugin_id'],'initial':row['origin']});save()
    (root/'route-one-offline').touch()
    for i,row in enumerate(previous):
        result=cli('app-fleet-maintain',row['request_id'],'--expected-revision',str(row['revision']),'--lan-profile','qualification')
        successor=result['connection']
        assert successor['phase']=='connected' and successor['origin']=='http://10.77.40.1:20082',result
        assert successor['request_id']!=row['request_id']
        health=cli('app-fleet-health',successor['request_id'],'--expected-revision',str(successor['revision']),'--lan-profile','qualification')
        assert health['health']=='healthy',health
        report['connections'][i]['replacement']=successor['origin']
        report['connections'][i]['standing_approval_reused']=True
        save()
    assert report['loopback_requests']==0,'remote runner localhost was incorrectly treated as desktop-local'
    # Restore only the test proxy and close/reconcile exact tracked sessions.
    (root/'route-one-offline').unlink()
    for _ in range(8):
        pending=cli('app-connections')
        if not pending:break
        for row in pending:
            if row['phase']=='connected':
                row=cli('app-disconnect',row['request_id'],'--expected-revision',str(row['revision']))['connection']
            if row['phase']=='closed' and row['receiver_reconciliation']:
                row=cli('app-reconcile',row['request_id'],'--expected-revision',str(row['revision']))['connection']
            if row['phase']=='closed' and not row['receiver_reconciliation']:
                cli('app-archive',row['request_id'],'--expected-revision',str(row['revision']))
            elif row['phase']!='connected':
                cli('app-fleet-maintain',row['request_id'],'--expected-revision',str(row['revision']),'--lan-profile','qualification')
    assert cli('app-connections')==[]
    report.update(passed=True,session_cleanup=True,remote_localhost_skipped=True,route_registry_shared=True)
except BaseException as error:
    report['error']=type(error).__name__+': '+str(error)
    raise
finally:
    save()
    if decoy:decoy.shutdown();decoy.server_close()
    if keyring:keyring.terminate();keyring.wait(timeout=5)

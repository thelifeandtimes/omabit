"""Private namespace-only exercise of unchanged plugin credential stores."""
import hashlib,json,os,secrets,subprocess,sys,time
from pathlib import Path

ROOT=Path('/lab')

def guard():
    assert os.environ.get('OMABIT_KEYRING_LAB')=='private-v1'
    assert os.environ.get('HOME')=='/lab/home' and not Path('/home').exists()
    assert [line.split(':',1)[0].strip() for line in Path('/proc/net/dev').read_text().splitlines()[2:]]==['lo']
    selected=json.loads((ROOT/'inputs.json').read_text())
    for name,value in selected['namespaces'].items():assert os.readlink('/proc/self/ns/'+name)!=value
    assert os.environ['DBUS_SESSION_BUS_ADDRESS'].startswith('unix:')

def probe():
    guard()
    value=json.load(sys.stdin);role=value['role'];profile=value['profile'];action=value['action'];session=value['session']
    package=ROOT/'artifacts'/profile/role
    sys.path.insert(0,str(package/('client' if role=='theme' else '')))
    if role=='theme':
        from support import Keyring,Failure
        store=Keyring();save=lambda s:store.store(s['url'],s);lookup=lambda:store.lookup(session['url'],session['ship']);error=Failure
    else:
        from session_store import SessionStore,StoreError
        store=SessionStore();save=store.save;lookup=lambda:store.lookup(session['url'],session['ship']);error=StoreError
    if action=='save':save(session);assert lookup()==session
    elif action=='lookup':assert lookup()==session
    elif action in ('locked','replace-blocked'):
        try:
            if action=='locked':lookup()
            else:save(dict(session,cookieValue='replacement-must-not-be-saved'))
        except error as e:
            if role=='theme':assert e.code=='keyring'
            else:assert 'Secret Service' in str(e)
        else:raise AssertionError('locked keyring accepted operation')
    else:raise AssertionError('unknown probe')
    print(json.dumps({'passed':True,'role':role,'profile':profile,'action':action}))

def main():
    guard();os.umask(0o077)
    password=secrets.token_hex(32).encode();daemon=None;report={'ok':False,'checks':[],'private_data_exported':False}
    def bus(*args):
        q=subprocess.run(['/usr/bin/busctl','--user',*args],capture_output=True,text=True,timeout=5)
        assert q.returncode==0,'private bus request failed'
        return q.stdout.strip()
    def run_probe(role,profile,action,session):
        q=subprocess.run(['/usr/bin/python3','-B','/lab/fixture.py','probe'],input=json.dumps(dict(role=role,profile=profile,action=action,session=session)).encode(),capture_output=True,timeout=15)
        assert q.returncode==0,'private credential probe failed: '+role+':'+profile+':'+action
        row=json.loads(q.stdout);assert row==dict(passed=True,role=role,profile=profile,action=action);report['checks'].append(row)
        (ROOT/'progress.json').write_text(json.dumps({'completed_checks':len(report['checks']),'last':row})+'\n')
    def account_hashes():
        return {str(p.relative_to(ROOT/'state')):hashlib.sha256(p.read_bytes()).hexdigest() for p in (ROOT/'state').rglob('account.json')}
    try:
        daemon=subprocess.Popen(['/usr/bin/gnome-keyring-daemon','--foreground','--unlock','--components=secrets','--control-directory=/lab/runtime/keyring'],stdin=subprocess.PIPE,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
        daemon.stdin.write(password);daemon.stdin.close()
        deadline=time.monotonic()+8
        while bus('call','org.freedesktop.DBus','/org/freedesktop/DBus','org.freedesktop.DBus','NameHasOwner','s','org.freedesktop.secrets')!='b true':
            assert daemon.poll() is None and time.monotonic()<deadline;time.sleep(.1)
        owner=bus('call','org.freedesktop.DBus','/org/freedesktop/DBus','org.freedesktop.DBus','GetConnectionUnixProcessID','s','org.freedesktop.secrets')
        assert owner=='u '+str(daemon.pid)
        sessions={}
        for profile in ('stable','dev'):
            for role in ('theme','dms'):
                session=dict(url='http://127.0.0.1:18081',ship='~zod',cookieName='urbauth-~zod',cookieValue=secrets.token_hex(24))
                sessions[profile,role]=session;run_probe(role,profile,'save',session)
        before=account_hashes();assert len(before)==2;report['phase']='sessions-saved'
        collection=bus('call','org.freedesktop.secrets','/org/freedesktop/secrets','org.freedesktop.Secret.Service','ReadAlias','s','default')
        assert collection=='o "/org/freedesktop/secrets/collection/login"'
        path='/org/freedesktop/secrets/collection/login'
        bus('call','org.freedesktop.secrets','/org/freedesktop/secrets','org.freedesktop.Secret.Service','Lock','ao','1',path)
        assert bus('get-property','org.freedesktop.secrets',path,'org.freedesktop.Secret.Collection','Locked')=='b true'
        for (profile,role),session in sessions.items():
            run_probe(role,profile,'locked',session);run_probe(role,profile,'replace-blocked',session)
        report['phase']='checking-locked-state'
        report['locked_account_metadata_unchanged']=account_hashes()==before
        assert report['locked_account_metadata_unchanged']
        assert bus('get-property','org.freedesktop.secrets',path,'org.freedesktop.Secret.Collection','Locked')=='b true'
        report['phase']='private-daemon-restart'
        daemon.terminate();daemon.wait(timeout=8);report['original_daemon_exit']=daemon.returncode
        assert daemon.returncode==0
        daemon=subprocess.Popen(['/usr/bin/gnome-keyring-daemon','--foreground','--unlock','--components=secrets','--control-directory=/lab/runtime/keyring'],stdin=subprocess.PIPE,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
        daemon.stdin.write(password);daemon.stdin.close()
        deadline=time.monotonic()+8
        while bus('call','org.freedesktop.DBus','/org/freedesktop/DBus','org.freedesktop.DBus','NameHasOwner','s','org.freedesktop.secrets')!='b true':
            assert daemon.poll() is None and time.monotonic()<deadline;time.sleep(.1)
        assert bus('call','org.freedesktop.DBus','/org/freedesktop/DBus','org.freedesktop.DBus','GetConnectionUnixProcessID','s','org.freedesktop.secrets')=='u '+str(daemon.pid)
        report['replacement_daemon_owner_verified']=True
        assert bus('get-property','org.freedesktop.secrets',path,'org.freedesktop.Secret.Collection','Locked')=='b false'
        report['phase']='checking-recovered-sessions'
        for (profile,role),session in sessions.items():run_probe(role,profile,'lookup',session)
        assert account_hashes()==before
        for subdir in ('state','config','cache'):
            for file in (ROOT/subdir).rglob('*'):
                if file.is_file():
                    content=file.read_bytes()
                    assert all(v['cookieValue'].encode() not in content for v in sessions.values()),'plaintext session fallback found'
        report.update(ok=True,recovery_mode='private-daemon-restart-with-original-password',session_count=4,locked_lookup_refusals=4,locked_save_refusals=4,recovered_original_sessions=4,account_metadata_unchanged=True,private_daemon_owner_verified=True,plaintext_fallback=False,network_requests=0,helper_processes=16)
    except Exception as e:
        report['error_type']=type(e).__name__;report['error_sha256']=hashlib.sha256(str(e).encode()).hexdigest()
    finally:
        if daemon:
            if daemon.poll() is None:
                daemon.terminate()
                try:daemon.wait(timeout=8)
                except subprocess.TimeoutExpired:daemon.kill();daemon.wait(timeout=3)
            report['daemon_exit']=daemon.returncode
        (ROOT/'result.json').write_text(json.dumps(report,indent=2)+'\n')
    print(json.dumps({'ok':report['ok'],'checks':len(report['checks'])}),flush=True)
    raise SystemExit(0 if report['ok'] else 1)

if __name__=='__main__':
    if sys.argv[1:] == ['probe']:probe()
    else:main()
